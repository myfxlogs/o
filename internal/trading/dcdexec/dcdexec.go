// Package dcdexec —— DCD 产品开仓执行件（T-0461 自 cmd/core/open.go 抽立
// 单源：产线 hedgeOpener.dcdOpen 与 simdrill DCD 演习格同径消费——venue
// 三步缝注入式，venue IO 之外全产线同码，禁影子实现）。
//
// 序=产线钉死：notionalCcy 锁仓币回填（md_dcd_product.raw——缺件
// fail-closed 不臆造）→quote 重取（陈旧提案 productId 重 quote 不盲执）
// →风控闸→fin_execution intent 行→trade→order-status 对拍定真态→
// tx{fin_position+exec state}。拒因持久化同原位：quote 步上游拦=
// fin_eval_reject 拒件；trade 步=fin_execution.venue_err_* 同 UPDATE。
package dcdexec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/risk"
	"o/internal/trading/okxexec"
)

// Venue dcd 执行三步缝（产线=*okxexec.Client 同签名实现；仿真=dcdsim 桩）
// +钱径摆渡缝三件（sfp/dcd=finance 产品——扣款源=funding 户,缺额自
// trading 幂等摆渡,T-0059 三态仲裁律；仿真桩=恒满仓径过）。
type Venue interface {
	DcdQuote(ctx context.Context, productID, notionalSz, notionalCcy string) (string, error)
	DcdTrade(ctx context.Context, quoteID string) (json.RawMessage, error)
	DcdOrderStatus(ctx context.Context, ordID string) (json.RawMessage, error)
	FundingAvail(ctx context.Context, ccy string) (float64, error)
	AssetTransfer(ctx context.Context, ccy string, amt float64, from, to string) (clTID, transID string, err error)
	AssetTransferState(ctx context.Context, ccy, clTID string) (string, error)
}

// 户编码（venue 方言——OKX: 6=funding 资金户/18=trading 交易户；
// finance 产品申购扣款源恒=funding 户,交易户持币须摆渡）。
const (
	AcctFunding = okxexec.AcctFunding
	AcctTrading = okxexec.AcctTrading
)

// Spec dcd 开仓规格（自持型——不引 lifecycle.OpenSpec 防域反向依赖）。
type Spec struct {
	StrategyRef int64
	Venue       string
	InstID      string // productId
	Direction   string
	Template    string // dcd_call|dcd_put|vol_sell_single_call
	NotionalUSD float64
}

// Deps 开仓依赖面（pool=台账/gate=风控闸——产线装配同位注入）。
type Deps struct {
	Pool *pgxpool.Pool
	Gate *risk.Gate
}

// Open dcd 产品开仓全径（产线 dcdOpen 原码搬迁——行为零漂移）。
func Open(ctx context.Context, d Deps, v Venue, spec Spec) (int64, error) {
	// T-0207：notionalCcy=产品锁定币（md_dcd_product.raw 回填勿猜——
	// quote 真径三参；缺件 fail-closed 不臆造）。
	var notionalCcy string
	if err := d.Pool.QueryRow(ctx, `SELECT raw->>'notionalCcy' FROM md_dcd_product
		WHERE product_id=$1 ORDER BY ts DESC LIMIT 1`, spec.InstID).Scan(&notionalCcy); err != nil || notionalCcy == "" {
		return 0, fmt.Errorf("dcd notionalCcy 缺 fail-closed（产品 raw 无此字段）: %v", err)
	}
	// 陈旧提案闸：按 spec.InstID=productId 重取 quote——产品失效=弃权如实；
	// quoteId 报价绑单径（每开必新 quote=陈旧提案不盲执语义自然保）。
	quoteID, err := v.DcdQuote(ctx, spec.InstID, Sz(spec.NotionalUSD), notionalCcy)
	if err != nil {
		// T-0259：上游拦 venue 拒因落 fin_eval_reject——exec 行未建径的
		// 持久化位（容器重建毁 stdout 实证，拒码+拒文+hits 留证）。
		vc, vm := VenueErr(err)
		QuoteReject(ctx, d.Pool, spec, vc, vm)
		return 0, fmt.Errorf("dcd quote 失效弃权（陈旧提案不盲执）: %w", err)
	}
	// 风控过闸同 carry 仓（notional/exposure 谓词现行；Legs 记 dcd 单腿形——
	// T-0353：put=卖权收权利金腿标如实分记）。
	legLabel := "dcd_call_buy"
	if spec.Template == "dcd_put" {
		legLabel = "dcd_put_sell"
	}
	dec := d.Gate.Check(ctx, risk.Intent{StrategyRef: spec.StrategyRef, Venue: spec.Venue,
		InstID: spec.InstID, Legs: []string{legLabel},
		NotionalUSD: spec.NotionalUSD, Leverage: 1, Kind: "open"})
	if !dec.Allow {
		return 0, fmt.Errorf("闸拒: %s", dec.Reason)
	}
	var execID int64
	if err := d.Pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,decision_id,inst_id,notional_usd,state)
		VALUES($1,$2,$3,$4,'open') RETURNING id`,
		spec.StrategyRef, dec.DecisionID, spec.InstID, spec.NotionalUSD).Scan(&execID); err != nil {
		return 0, err
	}
	// 钱径摆渡（T-0462——sfp/dcd 扣款源=funding 户实证 51736:交易户持币
	// 不够数。缺额 trading→funding 幂等摆渡；exec 行先落='open'=earn
	// 在途让位闸锚——防 sweepIdle 抢渡款买 Earn 的 ping-pong 病灶。
	// 歧态势 fail-closed：下拍重读 funding 余按新缺额自愈,禁盲动。
	// 域界：摆渡只覆 USDT 锁仓——非 USDT 锁仓币直通交 venue 仲裁
	// （ccy 计价尺度换算另票,禁拿 NotionalUSD 当异币额充数）。
	if notionalCcy == "USDT" {
		avail, aerr := v.FundingAvail(ctx, "USDT")
		if aerr != nil {
			Mark(ctx, d.Pool, execID, "failed", aerr)
			return execID, fmt.Errorf("funding 户余读败 fail-closed: %w", aerr)
		}
		if deficit := spec.NotionalUSD - avail; deficit > 0 {
			clTID, transID, terr := v.AssetTransfer(ctx, "USDT", deficit, AcctTrading, AcctFunding)
			switch {
			case terr == nil:
				log.Printf("[dcdexec] 摆渡成 clTId=%s transId=%s 缺额=%.4f USDT", clTID, transID, deficit)
			case isVenueReject(terr): // venue 业务终拒→禁补,如实败（581xx/5xxxx 族）
				MarkVE(ctx, d.Pool, execID, "failed", "", terr.Error())
				return execID, fmt.Errorf("摆渡终拒（clTId=%s）: %w", clTID, terr)
			default: // 传输名败→transfer-state 仲裁定真态
				st, qerr := v.AssetTransferState(ctx, "USDT", clTID)
				switch {
				case qerr == nil && st == "success":
					log.Printf("[dcdexec] 摆渡传输败实成 clTId=%s", clTID)
				case qerr == nil && st == "failed":
					Mark(ctx, d.Pool, execID, "failed", terr)
					return execID, fmt.Errorf("摆渡 state=failed（clTId=%s）: %w", clTID, terr)
				default:
					Mark(ctx, d.Pool, execID, "ambiguous", terr)
					return execID, fmt.Errorf("摆渡歧态势禁盲动（clTId=%s）: %w / %v", clTID, terr, qerr)
				}
			}
		}
	}
	// T-0207：venue trade 无 clOrdId 幂等键——fin_execution intent 行承幂等
	// 径照旧；歧态势=order-status?ordId/order-history 反查真态禁盲补。
	raw, err := v.DcdTrade(ctx, quoteID)
	if err != nil {
		// T-0208②：429=未执行定态分流（venue 限速拒非歧态势——trade 未
		// 落即拒可退避重试；真歧态势=请求已入未定才 ambiguous）。
		if strings.Contains(err.Error(), "429") {
			Mark(ctx, d.Pool, execID, "failed", err)
			return execID, fmt.Errorf("dcd trade 429 限速退避（exec_id=%d）: %w", execID, err)
		}
		// 传输/超时败=歧态势：禁补单——exec 行待人工/读件定真态（okx-exec 纪律）。
		Mark(ctx, d.Pool, execID, "ambiguous", err)
		return execID, fmt.Errorf("dcd trade 歧态势禁补单（exec_id=%d）: %w", execID, err)
	}
	var rows []struct {
		OrdID string `json:"ordId"`
		SCode string `json:"sCode"`
		SMsg  string `json:"sMsg"`
	}
	if uerr := json.Unmarshal(raw, &rows); uerr != nil || len(rows) == 0 {
		if uerr == nil {
			uerr = fmt.Errorf("data 空集")
		}
		Mark(ctx, d.Pool, execID, "ambiguous", uerr)
		return execID, fmt.Errorf("dcd trade 响应解析败（歧态势禁补单）: %v", uerr)
	}
	if rows[0].SCode != "0" {
		MarkVE(ctx, d.Pool, execID, "failed", rows[0].SCode, rows[0].SMsg)
		return execID, fmt.Errorf("dcd 业务拒 %s: %s", rows[0].SCode, rows[0].SMsg)
	}
	// order-status 对拍定真态（T-0160 返修①：读败/空态→ambiguous 禁默认
	// LIVE 造仓——歧态势禁补单纪律同 clOrdId 律；exec 行待查证非吞）。
	oraw, serr := v.DcdOrderStatus(ctx, rows[0].OrdID)
	var state string
	if serr == nil {
		var orows []struct {
			State string `json:"state"`
		}
		if json.Unmarshal(oraw, &orows) == nil && len(orows) > 0 {
			state = orows[0].State
		}
	}
	if serr != nil || state == "" {
		if serr == nil {
			serr = fmt.Errorf("order-status 空态无行")
		}
		Mark(ctx, d.Pool, execID, "ambiguous", serr)
		return execID, fmt.Errorf("order-status 读败/空态（歧态势禁补单——exec_id=%d 待查证非默认 LIVE 造仓）: %w", execID, serr)
	}
	// T-0160 返修⑤：exec 状态更新+position 落位同事务（position 写败→
	// exec ambiguous 标非半成——歧态势=exec 行在仓行缺=查证面不吞）。
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return execID, err
	}
	var pid int64
	if err := tx.QueryRow(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,opened_at)
		VALUES($1,$2,$3,$4,now()) RETURNING id`, execID, spec.StrategyRef, spec.InstID, spec.NotionalUSD).Scan(&pid); err != nil {
		_ = tx.Rollback(ctx)
		// T-0259 顺手根修：state='orphan' 越 0082 CHECK 集——写永静默败、行留
		// 'open' 幻影敞口；'ambiguous'=同语义合法态（exec 在仓行缺=查证面不吞）。
		Mark(ctx, d.Pool, execID, "ambiguous", err)
		return execID, fmt.Errorf("position 落位败（exec ambiguous 标——orphan 越 CHECK 静默败已修）: %w", err)
	}
	estate := ExecState(state)
	if estate == "" {
		_ = tx.Rollback(ctx)
		Mark(ctx, d.Pool, execID, "ambiguous", fmt.Errorf("order-status 态 %q 未载映射表", state))
		return execID, fmt.Errorf("order-status 态 %q 未载映射表（禁默认 open 造仓——歧态势）", state)
	}
	if _, err := tx.Exec(ctx, "UPDATE fin_execution SET state=$1 WHERE id=$2", estate, execID); err != nil {
		_ = tx.Rollback(ctx)
		return execID, err
	}
	if err := tx.Commit(ctx); err != nil {
		return execID, err
	}
	log.Printf("[dcdexec] DCD 开仓 exec=%d pos=%d ordId=%s state=%s", execID, pid, rows[0].OrdID, state)
	return execID, nil
}

// isVenueReject venue 业务终拒分型（*APIError=拒绝态非传输态——
// earn transfer 同律:终拒禁补单禁仲裁,如实败）。
func isVenueReject(err error) bool {
	var ae *okxexec.APIError
	return errors.As(err, &ae)
}

// VenueErr venue 拒因双层抽取（T-0259：内层 sCode 在且≠0=真实拒因优先——
// envelope code=1「All operations failed」壳码实证在案 order.go:13；非
// APIError=传输/解析败，原文留 msg、code=NULL 不臆造）。
func VenueErr(err error) (code, msg string) {
	var ae *okxexec.APIError
	if err == nil {
		return "", ""
	}
	if !errors.As(err, &ae) {
		return "", err.Error()
	}
	if ae.SCode != "" && ae.SCode != "0" {
		if ae.SMsg != "" {
			return ae.SCode, ae.SMsg
		}
		return ae.SCode, ae.Msg
	}
	return ae.Code, ae.Msg
}

// Mark exec 失败标定+拒因同 UPDATE（同语句=同原子写——乱序重建不丢证）；
// 写败显形非吞（拒因落库败=取证面再毁）。err 径供 APIError/传输败复用。
func Mark(ctx context.Context, pool *pgxpool.Pool, execID int64, state string, err error) {
	vc, vm := VenueErr(err)
	MarkVE(ctx, pool, execID, state, vc, vm)
}

// MarkVE 显式码文版（venue 响应行 sCode/sMsg 等非 error 形拒因直落）。
func MarkVE(ctx context.Context, pool *pgxpool.Pool, execID int64, state, vc, vm string) {
	if _, err := pool.Exec(ctx, `UPDATE fin_execution SET state=$2,
		venue_err_code=NULLIF($3,''), venue_err_msg=NULLIF($4,'') WHERE id=$1`,
		execID, state, vc, vm); err != nil {
		log.Printf("ALERT [dcdexec] exec %d 标定败（state=%s——拒因落库丢证风险）: %v", execID, state, err)
	}
}

// QuoteReject quote 步上游拦落件（T-0259：exec 行未建 venue 拒——
// fin_eval_reject 同构拒因表；reason 携码 per-(inst,template,direction,code)
// 幂等行，evidence.hits 累计尝试数=取证面拒率分母，全层码文入 evidence 不丢）。
func QuoteReject(ctx context.Context, pool *pgxpool.Pool, spec Spec, vc, vm string) {
	reason := "dcd_quote_reject"
	if vc != "" {
		reason += ":" + vc
	}
	ev, _ := json.Marshal(map[string]any{
		"step": "quote", "venue_code": vc, "venue_msg": vm,
		"notional_usd": spec.NotionalUSD, "strategy_ref": spec.StrategyRef, "hits": 1,
	})
	if _, err := pool.Exec(ctx, `INSERT INTO fin_eval_reject(inst_id,template,direction,reason,evidence)
		VALUES($1,$2,$3,$4,$5)
		ON CONFLICT (inst_id,template,direction,reason) DO UPDATE SET ts=now(),
		evidence=jsonb_set(EXCLUDED.evidence,'{hits}',
			to_jsonb(COALESCE((fin_eval_reject.evidence->>'hits')::int,0)+1))`,
		spec.InstID, spec.Template, spec.Direction, reason, ev); err != nil {
		log.Printf("ALERT [dcdexec] dcd quote 拒因落件败（%s %s——证毁风险）: %v", spec.InstID, reason, err)
	}
}

// ExecState order-status 态→fin_execution 态映射（T-0160 返修⑤桩锚件——
// 未载态=空串让位（禁默认 open）；歧态势由调用径置 ambiguous）。
func ExecState(vstate string) string {
	return map[string]string{"LIVE": "open", "SETTLED": "closed", "CANCELLED": "failed"}[vstate]
}

// Sz 数量格式化（venue sz 方言拒 sci——FormatFloat f,-1 去指数形；
// T-0160 返修②实证锚）。
func Sz(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
