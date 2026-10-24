package okxexec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// APIError venue 业务层拒（envelope code≠0）——类型化供调用方按码定态
// （如 51603=Order does not exist→absent，禁字符串匹配）。SCode/SMsg 透内层
// data[0] 真实拒因（envelope code=1「All operations failed」只是壳——真码在里层，
// 实测 2026-10-24：余额不足报 code=1 壳+内层 51008）。
type APIError struct{ Code, Msg, SCode, SMsg string }

func (e *APIError) Error() string {
	if e.SCode != "" {
		return "okx code=" + e.Code + "/" + e.SCode + " msg=" + e.Msg + "/" + e.SMsg
	}
	return "okx code=" + e.Code + " msg=" + e.Msg
}

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
	TgtCcy     string `json:"tgtCcy,omitempty"`     // spot 市价单 sz 口径：base_ccy|quote_ccy（市价买默认 quote_ccy——T-0042 根修；SWAP/FUTURES 不带）
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

// CancelOrder POST /trade/cancel-order——撤挂事务单发（ordId/clOrdId 二选一）。
func (c *Client) CancelOrder(ctx context.Context, instID, ordID, clOrdID string) (json.RawMessage, error) {
	body, _ := json.Marshal(map[string]string{"instId": instID, "ordId": ordID, "clOrdId": clOrdID})
	return c.do(ctx, "POST", "/api/v5/trade/cancel-order", body)
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
// 例外：51603 Order does not exist=确定的单未建（absent 语义——同 clOrdId 可重试
// 幂等键；真单 exec_id=2 实勘），非查询败。
func (c *Client) ResolveOrder(ctx context.Context, instID, clOrdID string) (OrderState, error) {
	st, _, err := c.ResolveOrderDetail(ctx, instID, clOrdID)
	return st, err
}

// ResolveOrderDetail 定真态+成交均价（T-0077：avgPx 落库锚——px 缺席=mtm/已
// 实现盈亏双断；签名增值不破坏旧口径）。
func (c *Client) ResolveOrderDetail(ctx context.Context, instID, clOrdID string) (OrderState, float64, error) {
	raw, err := c.Get(ctx, fmt.Sprintf("/api/v5/trade/order?instId=%s&clOrdId=%s", instID, clOrdID))
	if err != nil {
		var ae *APIError
		if errors.As(err, &ae) && ae.Code == "51603" {
			return StateAbsent, 0, nil
		}
		return StateUnknown, 0, fmt.Errorf("resolve 查询败（歧态势禁补单）: %w", err)
	}
	var rows []struct {
		State string `json:"state"`
		AvgPx string `json:"avgPx"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return StateUnknown, 0, err
	}
	if len(rows) == 0 {
		return StateAbsent, 0, nil
	}
	px := ParseF(rows[0].AvgPx)
	switch rows[0].State {
	case "filled":
		return StateFilled, px, nil
	case "live":
		return StateLive, px, nil
	case "partially_filled":
		return StatePartial, px, nil
	default: // canceled/mmp_canceled 等→按 absent 消费语义由调用方判
		return OrderState(rows[0].State), px, nil
	}
}

// AccountPosMode GET /account/config posMode（long_short_mode|net_mode）——
// 双向户开仓腿 posSide 装配依据（T-0040）。空值/读败上抛=调用方 fail-closed。
func (c *Client) AccountPosMode(ctx context.Context) (string, error) {
	raw, err := c.Get(ctx, "/api/v5/account/config")
	if err != nil {
		return "", err
	}
	var rows []struct {
		PosMode string `json:"posMode"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil || len(rows) == 0 || rows[0].PosMode == "" {
		return "", fmt.Errorf("posMode 读不回（fail-closed）: err=%v rows=%d", err, len(rows))
	}
	return rows[0].PosMode, nil
}

func parseF(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// ParseF venue 数字串解析壳（台账装配层消费）。
func ParseF(s string) float64 { return parseF(s) }
