// residual.go —— 残币处置面（T-0439 钱径风控面改写，原 T-0095 v3 活期归口
// 病灶下线：杂币扫进 Earn=系统自造未授权方向持仓，CORE 残袋实证）。交易户
// 逐币余量三岔处置——
//
//	① 扫袋白名单：earn.sweep_whitelist（计价币集，首版=["USDT"]——USDT 走
//	   sweepIdle 不经本面）内币种才准入 Earn 停泊；白名单外扫袋一律禁行；
//	② 杂币实存即清：白名单外余量估值≥earn.residual_clear_min_usd（阈下=
//	   粉尘挂账待归并，fin_residual_custody 如实标）→执行径市价清成 USDT
//	   （复用 execution.Executor Place/Resolve 定真态三分支——venue 写
//	   不新造下单码，见 residual_clear.go）；清仓败/歧态势=升件不补单
//	   （ops_event 同键节流）；
//	③ 豁免册激活：fin_known_residual 死家具转用——白名单外且暂不可清/
//	   不可卖（venue 终拒 51001/51003=未上市/停售）入册，扫袋与清仓双径
//	   读册跳过；册行=审计锚（为何留）；
//	④ 升件面：杂币敞口首检→ops_event「杂币敞口」行（检出即案）；已豁免/
//	   已升件同键节流不连鸣。
//
// 判据源=coins 逐币可动余量现读件（nil=处置面让位不判）；触发=腿平事件主
// 触发（ev_position——close 执行后扫）+ev_account 巡检拍兜底（复用不新造环）。
// 在持腿币不误动（FundCcy 口径内天然排除）；清仓不经 auto_purchase 闸——
// 处置=风险收回非申购，同 close 免 live_switch 律。
package earn

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/ledger"
	"o/internal/trading/okxexec"
)

// minPurchaseUsd Earn 灵活申购起投底额（USD 等值）——白名单币停泊径归口经济
// 性底（低于此额=托管如实标）。杂币粉尘阈另有 earn.residual_clear_min_usd。
const minPurchaseUsd = 1.0

// ClearLeg 清仓腿规格（earn 自有型——不引 execution 域型，D-514 方向闸零边；
// 装配层 cmd 组合根翻译）。Close=true 语义位随产线同构。
type ClearLeg struct {
	Venue, InstID, Side string
	Close               bool
	Sz                  string // venue 方言币量（base 口径）
}

// ClearResult 清仓定态回报（execution.OrderResult 同构投影——装配层翻译）。
type ClearResult struct {
	ClOrdID   string
	State     string // filled|live|partially_filled|absent|unknown
	FillPx    float64
	FillSz    float64
	VenueCode string
	VenueMsg  string
}

// Clearer 清仓执行缝（注入式——产线=execution.OkxExecutor 装配适配同一件
// T-0008 已验 FreshClient/clOrdId 径，earn 只消费 Place/Resolve 定真态三分支
// 不新造下单码）。
type Clearer interface {
	Place(ctx context.Context, leg ClearLeg, clOrdID string) (ClearResult, error)
	Resolve(ctx context.Context, leg ClearLeg, clOrdID string) (ClearResult, error)
}

// AttachClearer 挂清仓执行缝（cmd 装配层接线——未挂=清仓径让位，检出升件照发）。
func (w *Worker) AttachClearer(c Clearer) { w.clr = c }

// fundCcy 在持腿基币集（T-0100 ①：ledger.FundCcy 唯一真源 trading 域过滤
// ——coins 源=交易户余量，腿基币豁免只在 trading 域生效）：集合内=基金的
// 钱不清不扫；集合外=残币候选。
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

// sweepWhitelist 扫袋白名单集（earn.sweep_whitelist JSON 数组键——缺/坏=
// 只余本位 w.ccy，fail-closed 白名单外一律禁扫）。w.ccy 恒在集内。
func (w *Worker) sweepWhitelist() map[string]bool {
	out := map[string]bool{w.ccy: true}
	for _, c := range w.cfg.GetStringList("earn.sweep_whitelist") {
		out[c] = true
	}
	return out
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

// residualExempted 豁免册判定（T-0439 ③：fin_known_residual 未过期行=已定性
// 不可清残存——扫袋与清仓双径同读跳过）。查败=不豁免照判（豁免是减噪面
// 非钱险闸，宁可多判一拍不盲免）。
func (w *Worker) residualExempted(ctx context.Context, ccy string) bool {
	var ok bool
	if err := w.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM fin_known_residual
		WHERE venue=$1 AND ccy=$2 AND expires_at>now())`, w.venue, ccy).Scan(&ok); err != nil {
		w.rej.Inc("write_err")
		return false
	}
	return ok
}

// clearEscalated 「杂币清仓」升件键判定——已升件=清仓径让位同键节流
// （升件不补单律：败/歧态后同币不再自动重发，等人工处置）。
func (w *Worker) clearEscalated(ctx context.Context, ccy string) bool {
	var ok bool
	if err := w.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ops_event
		WHERE event='杂币清仓' AND note LIKE $1)`, w.venue+" "+ccy+" %").Scan(&ok); err != nil {
		w.rej.Inc("write_err")
		return false // 查败=不压拍照判（宁可多试一拍不盲压钱径动作）
	}
	return ok
}

// valCcyOf 代币数→USD 估值（md_ticks 最新 last——无 tick=0；与 cmd/core
// valCcy 同源口径——粉尘阈判据估值面）。
func valCcyOf(ctx context.Context, pool *pgxpool.Pool, ccy string, units float64) float64 {
	var last *float64
	_ = pool.QueryRow(ctx, `SELECT v FROM (SELECT last v,ts FROM md_ticks WHERE inst_id=$1
		UNION ALL SELECT px,ts FROM md_mark_index WHERE inst_id=$1) q
		ORDER BY ts DESC LIMIT 1`, ccy+"-USDT").Scan(&last)
	if last == nil {
		return 0
	}
	return units * *last
}

// sweepResidual 残币处置一拍：逐非基金币种量判——
//   - 豁免册在册 → 跳过（扫袋/清仓双径同读）；
//   - 白名单内 → Earn 停泊旧径（摆渡→申购；拒=回摆+托管标）；
//   - 白名单外+粉尘阈下 → 托管如实标（挂账待归并）；
//   - 白名单外+阈上/无价（usd=0 不可证小→保守侧敞口论） → 首检升件+
//     执行径市价清 USDT；终拒（不可卖）→豁免册；败/歧态 → 升件不补单
//     同键节流。
func (w *Worker) sweepResidual(ctx context.Context) error {
	if w.coins == nil {
		return nil // 处置面让位（判据源缺席——巡检拍下拍再判）
	}
	dustUSD, err := w.getF("earn.residual_clear_min_usd")
	if err != nil {
		w.rej.Inc("missing_param")
		return fmt.Errorf("earn.residual_clear_min_usd: %w", err)
	}
	fund, err := w.fundCcy(ctx)
	if err != nil {
		w.rej.Inc("missing_param")
		return fmt.Errorf("residual fund set: %w", err)
	}
	coins, err := w.coins(ctx)
	if err != nil {
		w.rej.Inc("idle_missing")
		log.Printf("ALERT [earn] 残币判据源读败——本拍让位: %v", err)
		return nil
	}
	wl := w.sweepWhitelist()
	for ccy, amt := range coins {
		if ccy == w.ccy || fund[ccy] || amt <= 0 {
			continue // USDT 本位走 sweepIdle；在持腿基币不误动
		}
		if w.residualExempted(ctx, ccy) {
			continue // 豁免册在册——扫袋/清仓双径同跳过
		}
		usd := valCcyOf(ctx, w.pool, ccy, amt)
		if wl[ccy] {
			w.parkWhitelisted(ctx, ccy, amt, usd)
			continue
		}
		if usd > 0 && usd < dustUSD {
			w.persistCustody(ctx, ccy, amt) // 粉尘级挂账待归并（不升件不清仓）
			continue
		}
		if w.clearEscalated(ctx, ccy) {
			continue // 已升件同键节流——升件不补单律，等人工处置
		}
		w.exposureEscalated(ctx, ccy, amt, usd)
		w.clearResidual(ctx, ccy, amt, usd)
	}
	return nil
}

// parkWhitelisted 白名单币停泊径（T-0095 v3 旧归口语义，白名单内专属）：
// 够起投=交易户→资金户→savings purchase 零换手停泊；业务拒=回摆+托管标。
// 单飞锁 TryLock 让位（USDT 回存优先，败拍下拍再来）。
func (w *Worker) parkWhitelisted(ctx context.Context, ccy string, amt, usd float64) {
	if w.api == nil {
		w.rej.Inc("venue_absent")
		return // venue 停泊面未挂——停泊径让位（清仓面不受影响）
	}
	if usd < minPurchaseUsd {
		w.persistCustody(ctx, ccy, amt)
		return
	}
	if w.custodyMarked(ctx, ccy) {
		return // T-0100 ④：30 天托管窗内零动零警（拒后空转斩——窗满重试）
	}
	if !w.autoPurchaseOn() {
		return // T-0342：停用中实际申购前让位——零摆渡零申购（钱留交易户）
	}
	if !w.mu.TryLock() {
		w.rej.Inc("sweep_busy")
		return
	}
	defer w.mu.Unlock()
	out, terr := w.transfer(ctx, ccy, amt, okxexec.AcctTrading, okxexec.AcctFunding)
	if out != txMoved {
		w.rej.Inc("deposit_failed")
		log.Printf("ALERT [earn] 白名单残币 %.8f %s 摆渡败（%v）——留交易户下拍再判", amt, ccy, terr)
		return
	}
	if perr := w.api.SavingsPurchase(ctx, ccy, amt); perr != nil {
		log.Printf("ALERT [earn] 白名单残币 %.8f %s 申购拒（%v）——回摆补偿", amt, ccy, perr)
		if cout, _ := w.transfer(ctx, ccy, amt, okxexec.AcctFunding, okxexec.AcctTrading); cout != txMoved {
			w.rej.Inc("compensation_failed")
			log.Printf("ALERT [earn] 白名单残币回摆补偿亦败——%.8f %s 滞资金户（人工升级）", amt, ccy)
			return
		}
		w.rej.Inc("deposit_failed")
		w.persistCustody(ctx, ccy, amt)
		return
	}
	w.afterMove(ctx, "residual_park", ccy, amt, "", 0,
		fmt.Sprintf("白名单残币 %.8f %s 活期归口停泊 Earn（申购径零换手）", amt, ccy))
}

// persistCustody 托管如实标（幂等——已有未过期行不重复写）：业主令禁兑换径
// （摩擦>本金）——粉尘阈下残币挂账待归并；要处理业主自行换。
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
	log.Printf("[earn] 残币托管标 %.8f %s（粉尘挂账待归并——业主令禁兑换径）", amt, ccy)
}
