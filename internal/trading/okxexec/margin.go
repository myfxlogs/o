// margin.go —— OKX 保证金借币面 RPC（O-39460 能力面）。
// 只读族=interest-rate/max-loan/borrow-repay-history/spot-borrow-repay-history；
// 写族=borrow-repay(quick margin)/spot-manual-borrow-repay/set-auto-repay。
// 幂等律：借贷写径 venue 侧无 clOrdId 概念——幂等由调用面持仓态对账担
// （借=加贷、还=减贷，重复借会超借——调用面禁盲重试，见 BorrowRepay 注）。
package okxexec

import (
	"context"
	"encoding/json"
	"fmt"
)

// MarginRate 借币利率行（GET /account/interest-rate——interestRate=每小时
// 小数利率；实测 BIGTIME 0.00036/h≈315%/yr、BTC 0.00000058/h≈0.5%/yr）。
type MarginRate struct {
	Ccy          string  `json:"ccy"`
	InterestRate float64 `json:"-"`
	NextEstRate  string  `json:"nextEstRate"`
}

// InterestRates GET /api/v5/account/interest-rate?ccy=A,B（逗号分隔多币）。
func (c *Client) InterestRates(ctx context.Context, ccys []string) ([]MarginRate, error) {
	if len(ccys) == 0 {
		return nil, fmt.Errorf("interest-rate: ccy 空")
	}
	raw, err := c.Get(ctx, "/api/v5/account/interest-rate?ccy="+joinCcy(ccys))
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Ccy          string `json:"ccy"`
		InterestRate string `json:"interestRate"`
		NextEstRate  string `json:"nextEstRate"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("interest-rate parse: %w", err)
	}
	out := make([]MarginRate, 0, len(rows))
	for _, r := range rows {
		var rate float64
		fmt.Sscanf(r.InterestRate, "%g", &rate)
		out = append(out, MarginRate{Ccy: r.Ccy, InterestRate: rate, NextEstRate: r.NextEstRate})
	}
	return out, nil
}

// MaxLoanRow GET /api/v5/account/max-loan 四必参（ccy=借币币, instId=现货/
// MARGIN 对, mgnMode=cross|isolated, mgnCcy=保证金币）。maxLoan 按 side 分行
// ——side=sell=该币可借出量（做空腿所需），buy=可借计价币。实测本账户额度
// ≈0（acctLv=2 非保证金模式+enableSpotBorrow=false——能力闸在账户面）。
type MaxLoanRow struct {
	Ccy     string  `json:"ccy"`
	InstID  string  `json:"instId"`
	MgnCcy  string  `json:"mgnCcy"`
	MgnMode string  `json:"mgnMode"`
	Side    string  `json:"side"`
	MaxLoan float64 `json:"-"`
}

func (c *Client) MaxLoan(ctx context.Context, ccy, instID, mgnMode, mgnCcy string) ([]MaxLoanRow, error) {
	raw, err := c.Get(ctx, fmt.Sprintf("/api/v5/account/max-loan?ccy=%s&instId=%s&mgnMode=%s&mgnCcy=%s",
		ccy, instID, mgnMode, mgnCcy))
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Ccy     string `json:"ccy"`
		InstID  string `json:"instId"`
		MgnCcy  string `json:"mgnCcy"`
		MgnMode string `json:"mgnMode"`
		Side    string `json:"side"`
		MaxLoan string `json:"maxLoan"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("max-loan parse: %w", err)
	}
	out := make([]MaxLoanRow, 0, len(rows))
	for _, r := range rows {
		var ml float64
		fmt.Sscanf(r.MaxLoan, "%g", &ml)
		out = append(out, MaxLoanRow{Ccy: r.Ccy, InstID: r.InstID, MgnCcy: r.MgnCcy,
			MgnMode: r.MgnMode, Side: r.Side, MaxLoan: ml})
	}
	return out, nil
}

// MaxLoanCcy ccy-only 形（O-40707 采集面——instId 与 ccy 互斥 venue 50024，
// 全量四参形 MaxLoan 对采集宇宙不可用；ccy-only 回单行 maxLoan=该币可借量）。
func (c *Client) MaxLoanCcy(ctx context.Context, ccy, mgnMode, mgnCcy string) (float64, error) {
	raw, err := c.Get(ctx, fmt.Sprintf("/api/v5/account/max-loan?ccy=%s&mgnMode=%s&mgnCcy=%s",
		ccy, mgnMode, mgnCcy))
	if err != nil {
		return 0, err
	}
	var rows []struct {
		MaxLoan string `json:"maxLoan"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return 0, fmt.Errorf("max-loan(ccy) parse: %w", err)
	}
	if len(rows) == 0 {
		return 0, nil
	}
	var ml float64
	fmt.Sscanf(rows[0].MaxLoan, "%g", &ml)
	return ml, nil
}

// BorrowRepay quick margin 借/还（POST /account/borrow-repay——{ccy,amt,side,
// instId?}；side=borrow|repay）。**无幂等键**——venue 侧重复调用=重复借贷，
// 调用面必须先持仓态对账再决定是否补发（禁盲重试同 okx-exec 歧态势律）。
func (c *Client) BorrowRepay(ctx context.Context, ccy string, amt float64, side, instID string) (json.RawMessage, error) {
	body := map[string]any{"ccy": ccy, "amt": fmt.Sprintf("%g", amt), "side": side}
	if instID != "" {
		body["instId"] = instID
	}
	b, _ := json.Marshal(body)
	return c.Post(ctx, "/api/v5/account/borrow-repay", b)
}

// SpotManualBorrowRepay 现货模式借/还（POST /account/spot-manual-borrow-repay
// ——{ccy,amt,side}；enableSpotBorrow=false 时 venue 拒=账户模式闸）。
func (c *Client) SpotManualBorrowRepay(ctx context.Context, ccy string, amt float64, side string) (json.RawMessage, error) {
	body := map[string]any{"ccy": ccy, "amt": fmt.Sprintf("%g", amt), "side": side}
	b, _ := json.Marshal(body)
	return c.Post(ctx, "/api/v5/account/spot-manual-borrow-repay", b)
}

// SetAutoRepay 自动还币开关（POST /account/set-auto-repay——{autoRepay}；
// 实测本账户 51010 current account mode=现货模式未开借币故拒）。
func (c *Client) SetAutoRepay(ctx context.Context, autoRepay bool) (json.RawMessage, error) {
	b, _ := json.Marshal(map[string]any{"autoRepay": autoRepay})
	return c.Post(ctx, "/api/v5/account/set-auto-repay", b)
}

// BorrowRepayHistRow 借还历史行（GET borrow-repay-history=quick margin /
// spot-borrow-repay-history=现货模式——账簿对账面）。
type BorrowRepayHistRow struct {
	Ccy  string `json:"ccy"`
	Amt  string `json:"amt"`
	Side string `json:"side"`
	TS   string `json:"ts"`
}

func (c *Client) BorrowRepayHistory(ctx context.Context, instID, ccy, side string, limit int) ([]BorrowRepayHistRow, error) {
	q := fmt.Sprintf("/api/v5/account/borrow-repay-history?instId=%s&limit=%d", instID, limit)
	if ccy != "" {
		q += "&ccy=" + ccy
	}
	if side != "" {
		q += "&side=" + side
	}
	return c.borrowHist(ctx, q)
}

func (c *Client) SpotBorrowRepayHistory(ctx context.Context, ccy, typ string, limit int) ([]BorrowRepayHistRow, error) {
	q := fmt.Sprintf("/api/v5/account/spot-borrow-repay-history?limit=%d", limit)
	if ccy != "" {
		q += "&ccy=" + ccy
	}
	if typ != "" {
		q += "&type=" + typ
	}
	return c.borrowHist(ctx, q)
}

func (c *Client) borrowHist(ctx context.Context, path string) ([]BorrowRepayHistRow, error) {
	raw, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Ccy  string `json:"ccy"`
		Amt  string `json:"amt"`
		Side string `json:"side"`
		TS   string `json:"ts"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("borrow-hist parse: %w", err)
	}
	out := make([]BorrowRepayHistRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, BorrowRepayHistRow{Ccy: r.Ccy, Amt: r.Amt, Side: r.Side, TS: r.TS})
	}
	return out, nil
}

func joinCcy(ccys []string) string {
	s := ""
	for i, c := range ccys {
		if i > 0 {
			s += ","
		}
		s += c
	}
	return s
}

// AccountAdjEquity /account/balance 顶层 adjEq（O-40707 margin 健康闸抵押物
// 口径——实证 adjEq=Earn赎回后 trading 自由保证金口径，策略冻结资金不计）。
func (c *Client) AccountAdjEquity(ctx context.Context) (float64, error) {
	raw, err := c.Get(ctx, "/api/v5/account/balance")
	if err != nil {
		return 0, err
	}
	var rows []struct {
		AdjEq string `json:"adjEq"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil || len(rows) == 0 {
		return 0, fmt.Errorf("balance adjEq parse: %w rows=%d", err, len(rows))
	}
	return parseF(rows[0].AdjEq), nil
}
