package okxexec

import (
	"context"
	"encoding/json"
	"fmt"
)

// AvailBal 交易户 ccy 可用量（/account/balance details[].availBal，缺则 availEq）
// ——现货平仓量决策时点现读面（T-0046 ②：taker 费基准币内扣致实持<委托记录）。
func (c *Client) AvailBal(ctx context.Context, ccy string) (float64, error) {
	raw, err := c.Get(ctx, "/api/v5/account/balance")
	if err != nil {
		return 0, err
	}
	var rows []struct {
		Details []struct {
			Ccy      string `json:"ccy"`
			AvailBal string `json:"availBal"`
			AvailEq  string `json:"availEq"`
		} `json:"details"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil || len(rows) == 0 {
		return 0, fmt.Errorf("balance parse: %w rows=%d", err, len(rows))
	}
	for _, d := range rows[0].Details {
		if d.Ccy != ccy {
			continue
		}
		if v := parseF(d.AvailBal); v > 0 {
			return v, nil
		}
		return parseF(d.AvailEq), nil
	}
	return 0, fmt.Errorf("ccy %s 不在余额明细——现读败 fail-closed", ccy)
}

// ContractPosSz 合约净持仓张数 abs（/account/positions?instId=——合约平仓量
// 现读面。无行=0（无仓可平——由调用方按对齐零 fail-closed））。
func (c *Client) ContractPosSz(ctx context.Context, instID string) (float64, error) {
	raw, err := c.Get(ctx, "/api/v5/account/positions?instId="+instID)
	if err != nil {
		return 0, err
	}
	var rows []struct {
		Pos string `json:"pos"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return 0, err
	}
	sum := 0.0
	for _, r := range rows {
		sum += parseF(r.Pos)
	}
	if sum < 0 {
		return -sum, nil
	}
	return sum, nil
}
