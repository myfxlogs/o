// residual.go —— 腿平残币活期归口（T-0095 v3，原 T-0097 折入）：对在持腿基币
// **之外**的交易户非 USDT 余量两岔处置——
//
//	① Earn 同币申购够起投 → 交易户→资金户→savings purchase（零换手——配管
//	   停泊径同构扩币种，钱不睡觉扩面）；
//	② 不够起投/不支持 → 托管如实标（业主令：禁兑换径——摩擦>本金；要处理
//	   业主自行换）。
//
// 判据源=coins 逐币可动余量现读件（nil=归口让位不判）；触发=腿平事件主触发
// （ev_position——close 执行后扫）+ev_account 巡检拍兜底（复用不新造环）。
// 在持腿币不误搬（FundCcy 口径内天然排除）；申购业务拒=回摆补偿+托管标。
package earn

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/ledger"
	"o/internal/trading/okxexec"
)

// minPurchaseUsd Earn 灵活申购起投底额（USD 等值）——OKX 灵活申购无平台级
// 最低额，币种面以可分精度为底；此处=归口经济性底（低于此额=托管如实标）。
const minPurchaseUsd = 1.0

// fundCcy 在持腿基币集（T-0100 ①：ledger.FundCcy 唯一真源 trading 域过滤
// ——coins 源=交易户余量，腿基币豁免只在 trading 域生效）：集合内=基金的
// 钱不归口不托管；集合外=残币候选。
func (w *Worker) fundCcy(ctx context.Context) (map[string]bool, error) {
	set, err := ledger.FundCcy(ctx, w.pool)
	out := map[string]bool{}
	for k := range set {
		if k[0] == w.venue+":trading" {
			out[k[1]] = true
		}
	}
	return out, err
}

// custodyMarked 30 天托管窗判定（T-0100 ④：申购拒已标币窗内整币跳过——
// 摆渡→拒→回摆空转+告警循环斩；窗满自然重试一次合语义）。
func (w *Worker) custodyMarked(ctx context.Context, ccy string) bool {
	var n int
	if err := w.pool.QueryRow(ctx, `SELECT count(*) FROM fin_residual_custody
		WHERE venue=$1 AND ccy=$2 AND marked_at>now()-interval '30 days'`,
		w.venue, ccy).Scan(&n); err != nil {
		w.rej.Inc("write_err")
		return false // 查败=不豁免照判（误判一摆可回摆非钱险；写败计数留痕）
	}
	return n > 0
}

// valCcyOf 代币数→USD 估值（md_ticks 最新 last——无 tick=0；与 cmd/core
// valCcy 同源口径——起投判据估值面）。
func valCcyOf(ctx context.Context, pool *pgxpool.Pool, ccy string, units float64) float64 {
	var last *float64
	_ = pool.QueryRow(ctx, `SELECT last FROM md_ticks WHERE inst_id=$1
		ORDER BY ts DESC LIMIT 1`, ccy+"-USDT").Scan(&last)
	if last == nil {
		return 0
	}
	return units * *last
}

// sweepResidual 残币归口一拍：逐非基金币种量判——
//   - 估值 < minPurchaseUsd → 托管标（幂等静默）；
//   - ≥ 底额 → 摆渡 funding→申购；申购业务拒=确认未动回摆+托管标。
//
// 单飞锁让位 sweep（TryLock——USDT 回存优先，残币下拍再来；腿平触发拍重叠时
// 禁双归口同币）。
func (w *Worker) sweepResidual(ctx context.Context) error {
	if w.api == nil || w.coins == nil {
		return nil // 归口让位（凭据/判据源缺席——巡检拍下拍再判）
	}
	fund, err := w.fundCcy(ctx)
	if err != nil {
		w.rej.Inc("missing_param")
		return fmt.Errorf("residual fund set: %w", err)
	}
	coins, err := w.coins(ctx)
	if err != nil {
		w.rej.Inc("idle_missing")
		log.Printf("ALERT [earn] 残币归口判据源读败——本拍让位: %v", err)
		return nil
	}
	for ccy, amt := range coins {
		if ccy == w.ccy || fund[ccy] || amt <= 0 {
			continue // USDT 本位走 sweepIdle；在持腿基币不误搬
		}
		if valCcyOf(ctx, w.pool, ccy, amt) < minPurchaseUsd {
			w.persistCustody(ctx, ccy, amt)
			continue
		}
		if w.custodyMarked(ctx, ccy) {
			continue // T-0100 ④：30 天托管窗内零动零警（拒后空转斩——窗满重试）
		}
		if !w.mu.TryLock() {
			w.rej.Inc("sweep_busy")
			return nil
		}
		out, terr := w.transfer(ctx, ccy, amt, okxexec.AcctTrading, okxexec.AcctFunding)
		if out != txMoved {
			w.rej.Inc("deposit_failed")
			log.Printf("ALERT [earn] 残币 %.8f %s 摆渡败（%v）——留交易户下拍再判", amt, ccy, terr)
			w.mu.Unlock()
			continue
		}
		if perr := w.api.SavingsPurchase(ctx, ccy, amt); perr != nil {
			log.Printf("ALERT [earn] 残币 %.8f %s 申购拒（%v）——回摆补偿", amt, ccy, perr)
			if cout, _ := w.transfer(ctx, ccy, amt, okxexec.AcctFunding, okxexec.AcctTrading); cout != txMoved {
				w.rej.Inc("compensation_failed")
				log.Printf("ALERT [earn] 残币回摆补偿亦败——%.8f %s 滞资金户（人工升级）", amt, ccy)
				w.mu.Unlock()
				continue
			}
			w.rej.Inc("deposit_failed")
			w.persistCustody(ctx, ccy, amt)
			w.mu.Unlock()
			continue
		}
		w.afterMove(ctx, "residual_park", ccy, amt, "", 0,
			fmt.Sprintf("腿平残币 %.8f %s 活期归口停泊 Earn（申购径零换手）", amt, ccy))
		w.mu.Unlock()
	}
	return nil
}

// persistCustody 托管如实标（幂等——已有未过期行不重复写）：业主令禁兑换径
// （摩擦>本金）——够起投但申购拒的残币同样落标；要处理业主自行换。
func (w *Worker) persistCustody(ctx context.Context, ccy string, amt float64) {
	var n int
	if err := w.pool.QueryRow(ctx, `SELECT count(*) FROM fin_residual_custody
		WHERE venue=$1 AND ccy=$2 AND marked_at>now()-interval '30 days'`,
		w.venue, ccy).Scan(&n); err != nil {
		w.rej.Inc("write_err")
		return
	}
	if n > 0 {
		return
	}
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_residual_custody(venue,ccy,amt)
		VALUES($1,$2,$3)`, w.venue, ccy, amt); err != nil {
		w.rej.Inc("write_err")
		return
	}
	w.rej.Inc("custody_marked")
	log.Printf("[earn] 残币托管标 %.8f %s（业主令禁兑换径——要处理业主自行换）", amt, ccy)
}
