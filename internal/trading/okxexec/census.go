package okxexec

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// CensusFacts 探索普查公共事实源（O-39480 oppCensus——/public/* 无签名
// 公开面钦定保留）：现货可买集+可借币率集。perp 存在=md_funding 符号集
// （采集面事实非 API 再取——spec 钉死单一事实源）。
type CensusFacts struct {
	SpotLive   map[string]bool    // instId → live 现货仪器（instType=SPOT state=live）
	BorrowRate map[string]float64 // ccy → 小时借币率（decimal——/public/interest-rate-loan-quota basic[].rate）
}

// CensusFacts 公共事实两拉（6h 拍距——instruments ~1.3MB+loan-quota
// 各一发；读败整体上抛由调用方标「不可判」不半真）。
func (c *Client) CensusFacts(ctx context.Context) (*CensusFacts, error) {
	out := &CensusFacts{SpotLive: map[string]bool{}, BorrowRate: map[string]float64{}}
	raw, err := c.PublicGet(ctx, "/api/v5/public/instruments?instType=SPOT")
	if err != nil {
		return nil, fmt.Errorf("spot instruments 拉败: %w", err)
	}
	var insts []struct {
		InstID string `json:"instId"`
		State  string `json:"state"`
	}
	if err := json.Unmarshal(raw, &insts); err != nil {
		return nil, fmt.Errorf("spot instruments 解败: %w", err)
	}
	for _, r := range insts {
		if r.State == "live" && r.InstID != "" {
			out.SpotLive[r.InstID] = true
		}
	}
	raw, err = c.PublicGet(ctx, "/api/v5/public/interest-rate-loan-quota")
	if err != nil {
		return nil, fmt.Errorf("loan-quota 拉败: %w", err)
	}
	var quota []struct {
		Basic []struct {
			Ccy  string `json:"ccy"`
			Rate string `json:"rate"`
		} `json:"basic"`
	}
	if err := json.Unmarshal(raw, &quota); err != nil || len(quota) == 0 {
		return nil, fmt.Errorf("loan-quota 解败 rows=%d err=%v", len(quota), err)
	}
	for _, row := range quota {
		for _, b := range row.Basic {
			if b.Ccy == "" {
				continue
			}
			if rate, perr := strconv.ParseFloat(b.Rate, 64); perr == nil {
				out.BorrowRate[b.Ccy] = rate // 小时借币率 decimal
			}
		}
	}
	return out, nil
}
