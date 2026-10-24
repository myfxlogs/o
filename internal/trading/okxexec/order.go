package okxexec

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// OrderParams 下单入参（方言：tdMode=cash[spot]/cross[perp]；posSide=net 单向户）。
type OrderParams struct {
	InstID     string `json:"instId"`
	TdMode     string `json:"tdMode"`
	Side       string `json:"side"`    // buy|sell
	OrdType    string `json:"ordType"` // market|limit
	Sz         string `json:"sz"`
	Px         string `json:"px,omitempty"`
	ClOrdID    string `json:"clOrdId"`
	PosSide    string `json:"posSide,omitempty"`    // 双向户平腿：long|short（net 单向户空省）
	ReduceOnly bool   `json:"reduceOnly,omitempty"` // 平仓腿必置——防「反向单=新开仓」自欺
}

// PlaceOrder POST /trade/order——钱径事务单发；传输败上抛 httpStatusError
// （调用方接歧态势走 ResolveOrder 定真态，禁盲重试）。
func (c *Client) PlaceOrder(ctx context.Context, p OrderParams) (json.RawMessage, error) {
	body, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, "POST", "/api/v5/trade/order", body)
}

// OrderState clOrdId 定态三分支（okx-exec 契约：filled→成交录；live/partially→
// 确认轨；absent→同 clOrdId 可重试幂等键不新生成；查询亦败=unknown 禁补单）。
type OrderState string

const (
	StateFilled  OrderState = "filled"
	StateLive    OrderState = "live"
	StatePartial OrderState = "partially_filled"
	StateAbsent  OrderState = "absent"
	StateUnknown OrderState = "unknown" // 歧态势——禁补单，critical 面消费
)

// ResolveOrder GET /trade/order?instId&clOrdId 定真态；查询败→StateUnknown。
func (c *Client) ResolveOrder(ctx context.Context, instID, clOrdID string) (OrderState, error) {
	raw, err := c.Get(ctx, fmt.Sprintf("/api/v5/trade/order?instId=%s&clOrdId=%s", instID, clOrdID))
	if err != nil {
		return StateUnknown, fmt.Errorf("resolve 查询败（歧态势禁补单）: %w", err)
	}
	var rows []struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return StateUnknown, err
	}
	if len(rows) == 0 {
		return StateAbsent, nil
	}
	switch rows[0].State {
	case "filled":
		return StateFilled, nil
	case "live":
		return StateLive, nil
	case "partially_filled":
		return StatePartial, nil
	default: // canceled/mmp_canceled 等→按 absent 消费语义由调用方判
		return OrderState(rows[0].State), nil
	}
}

func parseF(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// ParseF venue 数字串解析壳（台账装配层消费）。
func ParseF(s string) float64 { return parseF(s) }
