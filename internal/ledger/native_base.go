// native_base.go —— O-34287 原生币基线武装件（自 run.go 拆出守 300 行
// 棘轮）。存量非 USDT/USD 资产首拍 venue 真值驱动幂等建 opening 凭据。
package ledger

import (
	"context"
	"fmt"
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

// compensateLateBaselineBooks 迟簿双计自愈（O-37868 机制闸——判词=要闸）：
// 已武装键上 occurred_at<arm_ts ∧ ts>arm_ts 的 custody 现金流=武装时窗外
// 迟到账（scanner 史前回填/迁移补账/币形闸放开首扫同型——NFT #1808 实证，
// SHIB/BTT 靠 withdraw 对冲巧合幸免非设计）——基线 resid 已含其 venue
// 效应，cashflow 再入 net=双计。逐行补 native_baseline 负腿 amt=-amount
// （lateSrc=cfid 幂等锚——重放/重启/迁移手腿同键不双补；earn_in/out 同
// arm/reconcile cf 口径排除）。只补已武装键——白名单键无基线可双计自愈；
// arm_ts=该键 native_baseline 行 MIN(ts)（补偿腿 ts 恒晚于武装行不扰锚）。
func (r *Runner) compensateLateBaselineBooks(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen,kind)
		SELECT jsonb_build_object('op','native_baseline','venue',f.venue,'ccy',f.ccy,
				'amt',-f.amount,'lateSrc',f.id::text),
			'allow','allow',1,'open'
		FROM fin_cashflow f
		JOIN (SELECT intent->>'venue' v, intent->>'ccy' c, min(ts) arm_ts
			FROM fin_risk_decision WHERE intent->>'op'='native_baseline'
			GROUP BY 1,2) b ON b.v=f.venue AND b.c=f.ccy
		WHERE f.kind NOT IN ('earn_in','earn_out')
		  AND f.occurred_at < b.arm_ts AND f.ts > b.arm_ts
		  AND NOT EXISTS (SELECT 1 FROM fin_risk_decision x
			WHERE x.intent->>'op'='native_baseline' AND x.intent->>'lateSrc'=f.id::text)`)
	if err != nil {
		return fmt.Errorf("迟簿补偿腿写败: %w", err)
	}
	return nil
}
