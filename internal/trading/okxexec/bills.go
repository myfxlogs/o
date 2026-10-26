package okxexec

import (
	"context"
	"encoding/json"
	"fmt"
)

// Bill /account/bills 行（venue 账单——funding 实收/手续费记账源，T-0044）。
// subType 口径（官方 docs-v5 核证）：173 资金费支出/174 资金费收入；
// fee 正=返佣、负=扣除；balChg 正=余额增。sz 合约=张。
type Bill struct {
	BillID   string
	Type     string
	SubType  string
	InstID   string
	InstType string
	Ccy      string
	TsMs     int64
	BalChg   float64
	Fee      float64
	Pnl      float64
	Sz       float64
	Px       float64
}

// Bills GET /account/bills（instType 过滤，新→旧序，limit=100/页）。
// afterBillID 非空=取该账单之前（更旧）一页——分页游标。
func (c *Client) Bills(ctx context.Context, instType, afterBillID string) ([]Bill, error) {
	q := "/api/v5/account/bills?limit=100"
	if instType != "" {
		q += "&instType=" + instType // T-0160 ③空串=无过滤全谱（ASSET 非法参 51000 实证——dcd 160-163 唯捕获径）
	}
	if afterBillID != "" {
		q += "&after=" + afterBillID
	}
	raw, err := c.Get(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("bills 拉取: %w", err)
	}
	var rows []struct {
		BillID   string `json:"billId"`
		Type     string `json:"type"`
		SubType  string `json:"subType"`
		InstID   string `json:"instId"`
		InstType string `json:"instType"`
		Ccy      string `json:"ccy"`
		Ts       string `json:"ts"`
		BalChg   string `json:"balChg"`
		Fee      string `json:"fee"`
		Pnl      string `json:"pnl"`
		Sz       string `json:"sz"`
		Px       string `json:"px"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("bills 解析: %w", err)
	}
	out := make([]Bill, 0, len(rows))
	for _, r := range rows {
		b := Bill{BillID: r.BillID, Type: r.Type, SubType: r.SubType, InstID: r.InstID,
			InstType: r.InstType, Ccy: r.Ccy, TsMs: int64(parseF(r.Ts)),
			BalChg: parseF(r.BalChg), Fee: parseF(r.Fee), Pnl: parseF(r.Pnl),
			Sz: parseF(r.Sz), Px: parseF(r.Px)}
		if b.BillID == "" {
			return nil, fmt.Errorf("bills 行缺 billId——禁无名账单入账")
		}
		out = append(out, b)
	}
	return out, nil
}
