// TronScan HTTP fetch（独立于 internal/collect/trx，v1.1 A1/A2 修复）。
// TRON API JSON shape 是公开事实不是代码逻辑，自建 fetch+struct ~20 行。
package trxvote

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/account"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
)

// addrHexToB58 TRON hex 地址（41 前缀）→base58（gotron address 件）。
func addrHexToB58(hexAddr string) (string, error) {
	a, err := address.HexToAddress(hexAddr)
	if err != nil {
		return "", err
	}
	return a.String(), nil
}

// tronscanWitnessJSON 对应 TronScan /api/witness 单行（与 internal/collect/trx/types.go:162 同 shape，独立定义）。
type tronscanWitnessJSON struct {
	Address        string `json:"address"`
	Name           string `json:"name"`
	AnnualizedRate string `json:"annualizedRate"` // 年化 %（字符串，实测 "3.111945"）
	Brokerage      int    `json:"brokerage"`
	RealTimeVotes  int64  `json:"realTimeVotes"`
	Votes          int64  `json:"votes"`
	Producer       bool   `json:"producer"`
	ProducedTotal  int64  `json:"producedTotal"`
}

// httpClient 可被测试替换（httptest.NewServer + SetHTTPClient）。
var httpClient = &http.Client{Timeout: 30 * time.Second}

// SetHTTPClient 替换默认 HTTP 客户端（测试用 httptest.Server）。
func SetHTTPClient(c *http.Client) { httpClient = c }

// tronscanGet 自建 HTTP GET（~10 行，与 internal/collect/trx/apr.go:246 同逻辑但独立）。
func tronscanGet(ctx context.Context, url string) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// T-0192②：votes 读路 429 单次退避重试（boot 风暴实证——多路并发
	// fetch 撞限流；稳态低频不撞，一重试兜底即过）。
	if resp.StatusCode == 429 {
		resp.Body.Close()
		time.Sleep(1500 * time.Millisecond) // sched-exempt-ok：429 单次退避
		resp2, err := httpClient.Do(req.Clone(ctx))
		if err != nil {
			return nil, err
		}
		defer resp2.Body.Close()
		if resp2.StatusCode != 200 {
			return nil, fmt.Errorf("tronscan GET %s: status %d (429 retry)", url, resp2.StatusCode)
		}
		return io.ReadAll(resp2.Body)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("tronscan GET %s: status %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// tronscanVoteJSON 对应 /api/vote?voter= 响应 data[] 行（D-117：账户投票分布读路）。
// shape 与 internal/collect/trx/apr.go 的 tronscanVoteResponseJSON 同协议源，独立定义。
type tronscanVoteJSON struct {
	Data []struct {
		CandidateAddress    string `json:"candidateAddress"`
		Votes               int64  `json:"votes"`
		VoterAvailableVotes int64  `json:"voterAvailableVotes"`
	} `json:"data"`
	TotalVotes int64 `json:"totalVotes"`
}

// TronscanAccountFetcher tronscan /api/vote?voter= 读账户投票分布（D-117 主读路——
// trongrid gRPC GetAccount 对 SG IP 持续 429 实证 2026-10-07~08；tronscan HTTP 同链数据）。
type TronscanAccountFetcher struct{ base string }

// NewTronscanAccountFetcher 构造（base 空=调用方已三级回落过，这里不兜底）。
func NewTronscanAccountFetcher(base string) *TronscanAccountFetcher {
	return &TronscanAccountFetcher{base: base}
}

// GetAccountDetail 拉投票分布。TronPower=voterAvailableVotes（可用票数=冻结 TRX 数）；
// 无可用票字段时回退 Σvotes。免认证端点，限流宽松于 trongrid gRPC。
func (f *TronscanAccountFetcher) GetAccountDetail(ctx context.Context, addr string) (*AccountDetail, error) {
	body, err := tronscanGet(ctx, f.base+"/api/vote?voter="+addr)
	if err != nil {
		return nil, fmt.Errorf("fetch votes: %w", err)
	}
	var resp tronscanVoteJSON
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse votes: %w", err)
	}
	votes := make(map[string]int64, len(resp.Data))
	var avail, sum int64
	for _, v := range resp.Data {
		if v.CandidateAddress == "" {
			continue
		}
		votes[v.CandidateAddress] += v.Votes
		sum += v.Votes
		if v.VoterAvailableVotes > avail {
			avail = v.VoterAvailableVotes
		}
	}
	tp := avail
	if tp <= 0 {
		tp = sum // voterAvailableVotes 缺省时以实际已投为下界
	}
	return &AccountDetail{Votes: votes, TronPower: tp}, nil
}

// grpcAccountFetcher gotron gRPC GetAccountDetailedCtx 适配器（D-117 兜底路）。
type grpcAccountFetcher struct {
	conn interface {
		GetAccountDetailedCtx(context.Context, string) (*account.Account, error)
	}
}

// NewGRPCAccountFetcher 构造 gRPC 兜底读路。
func NewGRPCAccountFetcher(conn interface {
	GetAccountDetailedCtx(context.Context, string) (*account.Account, error)
}) AccountFetcher {
	return &grpcAccountFetcher{conn: conn}
}

func (g *grpcAccountFetcher) GetAccountDetail(ctx context.Context, addr string) (*AccountDetail, error) {
	acc, err := g.conn.GetAccountDetailedCtx(ctx, addr)
	if err != nil {
		return nil, err
	}
	return &AccountDetail{Votes: acc.Votes, TronPower: acc.TronPower}, nil
}

// fallbackAccountFetcher 主读失败时兜底次读（D-117：tronscan 主 → trongrid gRPC 备）。
type fallbackAccountFetcher struct{ primary, backup AccountFetcher }

// NewFallbackAccountFetcher 主备链（backup nil=纯主读）。
func NewFallbackAccountFetcher(primary, backup AccountFetcher) AccountFetcher {
	if backup == nil {
		return primary
	}
	return &fallbackAccountFetcher{primary: primary, backup: backup}
}

func (f *fallbackAccountFetcher) GetAccountDetail(ctx context.Context, addr string) (*AccountDetail, error) {
	d, err := f.primary.GetAccountDetail(ctx, addr)
	if err == nil {
		return d, nil
	}
	log.Printf("[trxvote] primary account fetch failed, trying backup: %v", err)
	return f.backup.GetAccountDetail(ctx, addr)
}

// fetchSRList 拉 TronScan /api/witness → 解析 → 转 []SRInfo（D-269 §2.3 v1.1）。
func fetchSRList(ctx context.Context, base string) ([]SRInfo, error) {
	body, err := tronscanGet(ctx, base+"/api/witness")
	if err != nil {
		return nil, fmt.Errorf("fetch witness: %w", err)
	}
	var rows []tronscanWitnessJSON
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("parse witness: %w", err)
	}
	out := make([]SRInfo, 0, len(rows))
	for i, r := range rows {
		rate, perr := f64z(r.AnnualizedRate)
		if perr != nil {
			return nil, fmt.Errorf("parse %s=%q: %w", "r.AnnualizedRate", r.AnnualizedRate, perr)
		} // 字符串→float，解析失败=0
		out = append(out, SRInfo{
			Address:        r.Address,
			Name:           r.Name,
			AnnualizedRate: rate,
			Brokerage:      r.Brokerage,
			RealTimeVotes:  r.RealTimeVotes,
			Producer:       r.Producer,
			Rank:           i + 1, // TronScan 返回按得票降序，rank=index+1
		})
	}
	seedLastGoodRate(out) // T-0192③：主源成功喂 last-good 年化缓存
	return out, nil
}

// —— T-0192③ witness 备读源：trongrid /wallet/listwitnesses（带 TRON-PRO-API-KEY；
// shape 差异抹平=hex→base58 地址转+年化 cache-last-good——trongrid 无
// annualizedRate 字段，首拍前无缓存=该组退避不评不臆造）。

// trongridWitnessJSON /wallet/listwitnesses 行形（address=41 前缀 hex）。
type trongridWitnessJSON struct {
	Address       string `json:"address"`
	Url           string `json:"url"`
	Producer      bool   `json:"isJobs"`
	RealTimeVotes int64  `json:"realTimeVotes"`
}

// lastGoodRate 年化 last-good 缓存（per-address——备源无年化字段沿缓存值）。
var lastGoodRate = map[string]float64{}

// FetchSRListTrongridFallback exported 备读径（cmd 装配面注入——
// SetWitFallback 回调签名）。
func FetchSRListTrongridFallback(ctx context.Context, trongridBase, apiKey string) ([]SRInfo, error) {
	return fetchSRListFallbackTrongrid(ctx, trongridBase, apiKey)
}

// fetchSRListFallbackTrongrid trongrid 备读（POST /wallet/listwitnesses——
// apilist 主源断供时顶；API key=trx.trongrid_api_key 径注入）。
func fetchSRListFallbackTrongrid(ctx context.Context, trongridBase, apiKey string) ([]SRInfo, error) {
	req, _ := http.NewRequestWithContext(ctx, "POST", trongridBase+"/wallet/listwitnesses", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", apiKey)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("trongrid listwitnesses: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("trongrid listwitnesses: status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var wrap struct {
		Witnesses []trongridWitnessJSON `json:"witnesses"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return nil, fmt.Errorf("trongrid parse: %w", err)
	}
	rows := wrap.Witnesses
	out := make([]SRInfo, 0, len(rows))
	for i, r := range rows {
		b58, err := addrHexToB58(r.Address)
		if err != nil {
			return nil, fmt.Errorf("trongrid addr hex→b58: %w", err)
		}
		rate, ok := lastGoodRate[b58]
		if !ok {
			continue // 首拍前无缓存=该组退避不评（钦定：不臆造年化）
		}
		out = append(out, SRInfo{
			Address:        b58,
			Name:           r.Url, // trongrid 无 name 字段——url 代显名（备源降级语义）
			AnnualizedRate: rate,  // cache-last-good 年化
			RealTimeVotes:  r.RealTimeVotes,
			Producer:       r.Producer,
			Rank:           i + 1,
		})
	}
	return out, nil
}

// seedLastGoodRate 主源成功时喂年化缓存（备源沿缓存值——a-first-prime 期前
// 无缓存=备源零行如实退避）。
func seedLastGoodRate(list []SRInfo) {
	for _, sr := range list {
		lastGoodRate[sr.Address] = sr.AnnualizedRate
	}
}
