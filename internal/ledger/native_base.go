// native_base.go —— O-34287 原生币基线武装件（自 run.go 拆出守 300 行
// 棘轮）。存量非 USDT/USD 资产首拍 venue 真值驱动幂等建 opening 凭据。
package ledger

import (
	"context"
	"log"
)

// armNativeBaselines 对拍前置武装：ven 中未白名单且非零的 (venue,ccy)
// 键→residual=venue−已知账本净（现金流+停泊+dcd基线+原生基线+mtm 同
// reconcile 口径）→INSERT native_baseline 凭据（NOT EXISTS 幂等）。
// resid=0=账本已全覆盖→只入对拍域不建行；建账后 wl[k]=true 本拍即比对。
func (r *Runner) armNativeBaselines(ctx context.Context, ven map[[2]string]float64,
	wl map[[2]string]bool, parked, dcdBase, nativeBase map[[2]string]float64) error {
	// 台账有量 venue 无行=补零强制比对（漏报根修——T-0100 原径语义随迁）。
	for _, m := range []map[[2]string]float64{parked, dcdBase, nativeBase} {
		for k := range m {
			if _, ok := ven[k]; !ok {
				ven[k] = 0
			}
		}
	}
	for k, eq := range ven {
		if wl[k] || eq == 0 {
			continue
		}
		var cf float64
		if err := r.pool.QueryRow(ctx,
			"SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE venue=$1 AND ccy=$2 AND kind NOT IN ('earn_in','earn_out')",
			k[0], k[1]).Scan(&cf); err != nil {
			r.rej.Inc("recon_probe_failed")
			log.Printf("WARN [ledger] 基线武装喂数读败 %s/%s——本键让位: %v", k[0], k[1], err)
			continue
		}
		net := cf + parked[k] + dcdBase[k] + nativeBase[k]
		if r.mtm != nil {
			if mv, merr := r.mtm(ctx); merr == nil {
				net += mv[k]
			}
		}
		resid := eq - net
		if resid == 0 {
			wl[k] = true // 账本已全覆盖——并入对拍域，漂移不静默
			continue     // 零基线不建行
		}
		tag, err := r.pool.Exec(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen,kind)
			SELECT jsonb_build_object('op','native_baseline','venue',$1::text,'ccy',$2::text,'amt',$3::float8),
				'allow','allow',1,'open'
			WHERE NOT EXISTS (SELECT 1 FROM fin_risk_decision
				WHERE intent->>'op'='native_baseline' AND intent->>'venue'=$1 AND intent->>'ccy'=$2)`,
			k[0], k[1], resid)
		if err != nil {
			r.rej.Inc("write_err")
			log.Printf("WARN [ledger] 基线凭据写败 %s/%s: %v", k[0], k[1], err)
			continue
		}
		if tag.RowsAffected() > 0 {
			log.Printf("[ledger] 原生币基线建账 %s/%s resid=%.8f（venue 真值首拍归因）", k[0], k[1], resid)
			nativeBase[k] += resid
			wl[k] = true // 建账成功本拍即并入对拍域；写败/竞态下拍让位重试
		}
	}
	return nil
}
