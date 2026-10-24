// savings.go —— OKX Simple Earn 灵活+资金划转 REST 面（D-0006 停泊态钱径）。
// 界桩出处=okx-rest-boundary 钦定保留面（金融产品面/funding 户无 WS op，事务单发）：
//
//	/finance/savings/purchase-redempt · /finance/savings/balance · lending-rate-summary（公开）
//	/asset/transfer · /asset/transfer-state
//
// 实证（2026-10-23 curl 亲拍）：lending-rate-summary?ccy=USDT → avgRate="0.035"（24h 平均
// 年化出借率）；purchase-redempt 无凭据=50103（私有面）。赎回侧 side 拼写=`redempt`。
// 歧态势处置律同 okx-exec：transfer 传输败→transfer-state 按 clTId 定真态，禁盲重发。
package okxexec

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// account 编码（OKX 方言：6=funding 资金户，18=trading 交易户）——Earn 申购只吃
// funding 户资产，赎回回 funding 户；与交易户之间靠 /asset/transfer 摆渡。
const (
	AcctFunding = "6"
	AcctTrading = "18"
)

// RateSummary lending-rate-summary 采样（年化 APR 小数——0.035=3.5%）。
type RateSummary struct {
	AvgRate float64 // 24h 平均年化出借率（Earn 灵活现率口径——比较底取值源）
	PreRate float64 // 上一期年化
	EstRate float64 // 下期预估年化
}

// LendingRateSummary 公开面利率采样（无凭据——现率采集不依赖 app_secrets 落位）。
// h1 钉死传输与 NewClient 同律（h2 感染病灶），但不签名（公开端点）。
func LendingRateSummary(ctx context.Context, ccy string) (RateSummary, error) {
	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{NextProtos: []string{"http/1.1"}},
		TLSNextProto:      map[string]func(string, *tls.Conn) http.RoundTripper{},
		ForceAttemptHTTP2: false,
		DisableKeepAlives: true,
		DialContext:       (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
	}
	hc := &http.Client{Transport: tr, Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		BaseURL+"/api/v5/finance/savings/lending-rate-summary?ccy="+ccy, nil)
	if err != nil {
		return RateSummary{}, err
	}
	resp, err := hc.Do(req)
	if err != nil {
		return RateSummary{}, err
	}
	defer resp.Body.Close()
	var env struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			AvgRate string `json:"avgRate"`
			PreRate string `json:"preRate"`
			EstRate string `json:"estRate"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return RateSummary{}, err
	}
	if env.Code != "0" {
		return RateSummary{}, fmt.Errorf("okx code=%s msg=%s", env.Code, env.Msg)
	}
	if len(env.Data) == 0 {
		return RateSummary{}, fmt.Errorf("lending-rate-summary %s 空数据", ccy)
	}
	d := env.Data[0]
	avg, err := strconv.ParseFloat(d.AvgRate, 64)
	if err != nil {
		return RateSummary{}, fmt.Errorf("avgRate parse %q: %w", d.AvgRate, err)
	}
	out := RateSummary{AvgRate: avg}
	out.PreRate, _ = strconv.ParseFloat(d.PreRate, 64)
	out.EstRate, _ = strconv.ParseFloat(d.EstRate, 64)
	return out, nil
}

// SavingsPurchase Earn 灵活申购（funding 户→Earn 停泊）。
func (c *Client) SavingsPurchase(ctx context.Context, ccy string, amt float64) error {
	return c.savingsMove(ctx, ccy, amt, "purchase")
}

// SavingsRedeem Earn 灵活赎回（Earn→funding 户，灵活 T+0）。amt<=0 或超持仓=业务拒上抛。
func (c *Client) SavingsRedeem(ctx context.Context, ccy string, amt float64) error {
	return c.savingsMove(ctx, ccy, amt, "redempt")
}

func (c *Client) savingsMove(ctx context.Context, ccy string, amt float64, side string) error {
	body := fmt.Sprintf(`{"ccy":%q,"amt":%q,"side":%q}`, ccy, fmtAmt(amt), side)
	_, err := c.do(ctx, http.MethodPost, "/api/v5/finance/savings/purchase-redempt", []byte(body))
	return err
}

// SavingsBalance /finance/savings/balance 持仓查询已落 client.go（raw 返回，T-holding
// 钦定保留面）；ccy 语义封装（过滤+pendingAmt 计入）在 internal/earn 消费侧。

// AssetTransfer 账户内划转（type=0，funding↔trading）——携幂等键（POST 体字段名=
// clientId；查询侧 GET transfer-state 参名同为 clientId——实测 2026-10-24：写
// clTId 被静默吞，查 clientId=自建值 返 51000 参数错）。
func (c *Client) AssetTransfer(ctx context.Context, ccy string, amt float64, from, to string) (clTID string, err error) {
	clTID = fmt.Sprintf("%d", time.Now().UnixNano()%1e15)
	body := fmt.Sprintf(`{"type":"0","ccy":%q,"amt":%q,"from":%q,"to":%q,"clientId":%q}`,
		ccy, fmtAmt(amt), from, to, clTID)
	_, err = c.do(ctx, http.MethodPost, "/api/v5/asset/transfer", []byte(body))
	return clTID, err
}

// AssetTransferState 按 clTId 定划转真态（success/failed/其他=在途或未知——调用方
// 禁对非终态盲动：在途禁重发，未知 fail-closed）。
func (c *Client) AssetTransferState(ctx context.Context, ccy, clTID string) (string, error) {
	raw, err := c.Get(ctx, "/api/v5/asset/transfer-state?type=0&ccy="+ccy+"&clientId="+clTID)
	if err != nil {
		return "", err
	}
	var rows []struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "unknown", nil
	}
	return rows[0].State, nil
}

// trunc8 向零截断 8 位小数（非四舍五入——转出额恒≤余额真值）。
func trunc8(amt float64) float64 {
	return math.Trunc(amt*1e8) / 1e8
}

// fmtAmt 金额串（8 位小数截断非四舍五入——T-0058 实证：75.927133877… 四舍
// 五入成 75.92713388>availBal 即 58350 拒；截断=不超余额真值）。
func fmtAmt(amt float64) string {
	s := strconv.FormatFloat(trunc8(amt), 'f', 8, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}
