package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"phase/internal/trading/okxexec"
)

// OkxExecutor okxexec 适配器（venue 调用只走 T-0008 已验 FreshClient/clOrdId 径——
// 禁另起 venue 径）。posMode 首单实读 /account/config 一次缓存（禁硬编假设——
// 读不回=fail-closed 拒单）。
type OkxExecutor struct {
	C *okxexec.Client

	posOnce sync.Once
	posMode string
	posErr  error
}

// posModeOf 惰性读账户 posMode（一次缓存——account config 静态面）。
func (e *OkxExecutor) posModeOf(ctx context.Context) (string, error) {
	e.posOnce.Do(func() { e.posMode, e.posErr = e.C.AccountPosMode(ctx) })
	return e.posMode, e.posErr
}

func (e *OkxExecutor) Place(ctx context.Context, leg Leg, clOrdID string) (OrderResult, error) {
	posMode, err := e.posModeOf(ctx)
	if err != nil {
		return OrderResult{ClOrdID: clOrdID, State: "unknown"}, fmt.Errorf("posMode fail-closed: %w", err)
	}
	raw, err := e.C.PlaceOrder(ctx, buildOrderParams(leg, clOrdID, posMode))
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
	st, err := e.C.ResolveOrder(ctx, leg.InstID, clOrdID)
	return OrderResult{ClOrdID: clOrdID, State: string(st)}, err
}

// buildOrderParams 腿→下单参装配（posSide 注入纯函数——T-0040 测试锚）：
// posMode=long_short_mode 且 SWAP（tdMode=cross）必带 posSide——开仓 sell→short/
// buy→long；平仓 buy→short、sell→long（平对向仓，T-0036）。net_mode 或 spot
// （tdMode=cash）免带（双向户 spot 无持仓向语义）。
func buildOrderParams(leg Leg, clOrdID, posMode string) okxexec.OrderParams {
	p := okxexec.OrderParams{
		InstID: leg.InstID, TdMode: tdMode(leg), Side: leg.Side,
		OrdType: "market", Sz: leg.Sz, ClOrdID: clOrdID}
	if leg.Close {
		p.ReduceOnly = true
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
