package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"o/internal/trading/okxexec"
)

// OkxExecutor okxexec 适配器（venue 调用只走 T-0008 已验 FreshClient/clOrdId 径——
// 禁另起 venue 径）。posMode 首单实读 /account/config 一次缓存（禁硬编假设——
// 读不回=fail-closed 拒单）。
type OkxExecutor struct {
	C *okxexec.Client

	confMu sync.Mutex
	conf   *acctConf // nil=未读（读败不留缓存——瞬断下次自愈）
}

type acctConf struct{ posMode, acctLv string }

// confOf 账户 config 缓存读（静态面首读即缓存；refresh=true 强制重取——
// acctLv 是业主侧外部可变面，闸拒后强制重取一次让切档自愈不赖重启）。
func (e *OkxExecutor) confOf(ctx context.Context, refresh bool) (acctConf, error) {
	e.confMu.Lock()
	defer e.confMu.Unlock()
	if e.conf != nil && !refresh {
		return *e.conf, nil
	}
	pm, lv, err := e.C.AccountConfig(ctx)
	if err != nil {
		if e.conf != nil {
			return *e.conf, nil // 刷新败回落旧缓存（静态面降级优于拒）
		}
		return acctConf{}, err
	}
	e.conf = &acctConf{pm, lv}
	return *e.conf, nil
}

// acctLvGate 账户模式预检（2026-11-27 实案：acctLv=1 现货档下 SWAP 单全谱
// 51010「current account mode」拒——venue 级终局非参数病，盲续试=噪声风暴）。
// cross 腿须 acctLv≥2（保证金档）；现金腿（cash/spot）现货档可过。
func acctLvGate(tdMode, acctLv string) error {
	if tdMode != "cross" {
		return nil
	}
	if lv, err := strconv.Atoi(acctLv); err != nil || lv < 2 {
		return fmt.Errorf("okx acctLv=%s 现货档不支持衍生品单（cross）——业主侧账户模式改回保证金档自解", acctLv)
	}
	return nil
}

func (e *OkxExecutor) Place(ctx context.Context, leg Leg, clOrdID string) (OrderResult, error) {
	conf, err := e.confOf(ctx, false)
	if err != nil {
		return OrderResult{ClOrdID: clOrdID, State: "unknown"}, fmt.Errorf("account config fail-closed: %w", err)
	}
	if err := acctLvGate(tdMode(leg), conf.acctLv); err != nil {
		conf, err = e.confOf(ctx, true)
		if err != nil {
			return OrderResult{ClOrdID: clOrdID, State: "unknown"}, fmt.Errorf("account config fail-closed: %w", err)
		}
		if err := acctLvGate(tdMode(leg), conf.acctLv); err != nil {
			return OrderResult{ClOrdID: clOrdID, State: "unknown"}, err
		}
	}
	raw, err := e.C.PlaceOrder(ctx, buildOrderParams(leg, clOrdID, conf.posMode))
	if err != nil {
		return OrderResult{ClOrdID: clOrdID, State: "unknown"}, err
	}
	var rows []struct {
		OrdID string `json:"ordId"`
		SCode string `json:"sCode"`
		SMsg  string `json:"sMsg"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil || len(rows) == 0 {
		return OrderResult{ClOrdID: clOrdID, State: "unknown"}, err
	}
	if rows[0].SCode != "0" {
		// 业务拒=单未建（absent）+拒码留痕（真拒因进 fin_order_leg——T-0040）。
		return OrderResult{ClOrdID: clOrdID, State: "absent", VenueCode: rows[0].SCode, VenueMsg: rows[0].SMsg}, nil
	}
	// market 单下单即成交通常——为严谨走 Resolve 定真态。
	return e.Resolve(ctx, leg, clOrdID)
}

func (e *OkxExecutor) Resolve(ctx context.Context, leg Leg, clOrdID string) (OrderResult, error) {
	st, px, fillSz, fee, feeCcy, err := e.C.ResolveOrderDetail(ctx, leg.InstID, clOrdID)
	return OrderResult{ClOrdID: clOrdID, State: string(st), FillPx: px, FillSz: fillSz, Fee: fee, FeeCcy: feeCcy}, err
}

// buildOrderParams 腿→下单参装配（posSide 注入纯函数——T-0040 测试锚）：
// posMode=long_short_mode 且 SWAP（tdMode=cross）必带 posSide——开仓 sell→short/
// buy→long；平仓 buy→short、sell→long（平对向仓，T-0036）。net_mode 或 spot
// （tdMode=cash）免带（双向户 spot 无持仓向语义）。
func buildOrderParams(leg Leg, clOrdID, posMode string) okxexec.OrderParams {
	p := okxexec.OrderParams{
		InstID: leg.InstID, TdMode: tdMode(leg), Side: leg.Side,
		OrdType: "market", Sz: leg.Sz, ClOrdID: clOrdID}
	if leg.Maker { // T-0238 maker 径：post_only+挂价——成=低费/返佣面，超时径外取消降级
		p.OrdType, p.Px = "post_only", strconv.FormatFloat(leg.LimitPx, 'f', -1, 64)
	}
	// reduceOnly 域界（T-0043）：官方口径仅适用杠杆+合约——合约腿（cross）平仓
	// 必置防反向新开；spot 腿（cash）带 reduceOnly=51000 参数拒，禁带。
	if leg.Close && p.TdMode == "cross" {
		p.ReduceOnly = true
	}
	// T-0042 方言根修：spot 市价买 sz 默认按计价币（quote_ccy）解释——base 量意图
	// 必带 base_ccy 钉死口径（exec_id=3 实弹实证）；T-0064 追加：TgtQuote 位=显式
	// quote_ccy 委托（sz=USDT 额 venue 内换算——配比径落法，粒度残差→舍入级）。
	if p.TdMode == "cash" && p.Side == "buy" && p.OrdType == "market" {
		if leg.TgtQuote {
			p.TgtCcy = "quote_ccy"
		} else {
			p.TgtCcy = "base_ccy"
		}
	}
	if posMode == "long_short_mode" && p.TdMode == "cross" {
		if leg.Close {
			if leg.Side == "buy" {
				p.PosSide = "short"
			} else {
				p.PosSide = "long"
			}
		} else if leg.Side == "sell" {
			p.PosSide = "short"
		} else {
			p.PosSide = "long"
		}
	}
	return p
}

func tdMode(leg Leg) string {
	// okx 方言：spot=cash、perp(SWAP)=cross（okx-exec skill 钉）。
	if len(leg.InstID) > 5 && leg.InstID[len(leg.InstID)-4:] == "SWAP" {
		return "cross"
	}
	return "cash"
}

// Cancel okx 撤挂（T-0238 maker 时限件——clOrdId 键）。
func (e *OkxExecutor) Cancel(ctx context.Context, leg Leg, clOrdID string) error {
	_, err := e.C.CancelOrder(ctx, leg.InstID, "", clOrdID)
	return err
}
