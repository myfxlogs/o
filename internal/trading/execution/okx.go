package execution

import (
	"context"
	"encoding/json"

	"phase/internal/trading/okxexec"
)

// OkxExecutor okxexec 适配器（venue 调用只走 T-0008 已验 FreshClient/clOrdId 径——
// 禁另起 venue 径）。
type OkxExecutor struct{ C *okxexec.Client }

func (e OkxExecutor) Place(ctx context.Context, leg Leg, clOrdID string) (OrderResult, error) {
	raw, err := e.C.PlaceOrder(ctx, okxexec.OrderParams{
		InstID: leg.InstID, TdMode: tdMode(leg), Side: leg.Side,
		OrdType: "market", Sz: leg.Sz, ClOrdID: clOrdID})
	if err != nil {
		return OrderResult{ClOrdID: clOrdID, State: "unknown"}, err
	}
	var rows []struct {
		OrdID string `json:"ordId"`
		SCode string `json:"sCode"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil || len(rows) == 0 {
		return OrderResult{ClOrdID: clOrdID, State: "unknown"}, err
	}
	if rows[0].SCode != "0" {
		return OrderResult{ClOrdID: clOrdID, State: "absent"}, nil
	}
	// market 单下单即成交通常——为严谨走 Resolve 定真态。
	return e.Resolve(ctx, leg, clOrdID)
}

func (e OkxExecutor) Resolve(ctx context.Context, leg Leg, clOrdID string) (OrderResult, error) {
	st, err := e.C.ResolveOrder(ctx, leg.InstID, clOrdID)
	return OrderResult{ClOrdID: clOrdID, State: string(st)}, err
}

func tdMode(leg Leg) string {
	// okx 方言：spot=cash、perp(SWAP)=cross（okx-exec skill 钉）。
	if len(leg.InstID) > 5 && leg.InstID[len(leg.InstID)-4:] == "SWAP" {
		return "cross"
	}
	return "cash"
}
