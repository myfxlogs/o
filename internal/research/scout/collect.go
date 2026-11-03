package scout

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// Fetcher HTTP 取件缝（测试桩替——生产 httpGet；禁任何需凭据端点）。
type Fetcher func(ctx context.Context, url string, body string) ([]byte, error)

// httpGet 公开端点取件（GET body=空；POST 供 hyperliquid 类——零凭据零头）。
func httpGet(ctx context.Context, url, body string) ([]byte, error) {
	var req *http.Request
	var err error
	if body == "" {
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	} else {
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

// row 采集行（venue 原生符号——归一映射归度量期）。
type row struct {
	inst     string
	rate     *float64
	next     *float64
	interval float64
	mark     *float64
	index    *float64
}

// adapters venue→采集函数册（measuring 态 venue 逐位跑；协议面取证纪律——
// 字段方言以实测响应为准，下「无此面」结论前消息体全文核证）。
var adapters = map[string]func(ctx context.Context, f Fetcher) ([]row, error){
	"binance-usdm": collectBinance,
	"bybit":        collectBybit,
	"hyperliquid":  collectHyperliquid,
}

// collectAll 对 state='measuring' 候选行逐个采集落 md_venue_funding。
func (r *Runner) collectAll(ctx context.Context) error {
	venues, err := r.measuringVenues(ctx)
	if err != nil {
		return err
	}
	for _, v := range venues {
		ad, ok := adapters[v]
		if !ok {
			continue // 无 adapter=骨架期未采（candidate 态如实）
		}
		rows, err := ad(ctx, r.fetch)
		if err != nil {
			r.rej.Inc("venue_down")
			continue // 单 venue 挂不拖全局——下一拍重试
		}
		if err := r.writeRows(ctx, v, rows); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) measuringVenues(ctx context.Context) ([]string, error) {
	rs, err := r.pool.Query(ctx,
		"SELECT venue FROM fin_venue_candidate WHERE state IN ('measuring','recommended','registered')")
	if err != nil {
		return nil, err
	}
	defer rs.Close()
	var out []string
	for rs.Next() {
		var v string
		if err := rs.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rs.Err()
}

func (r *Runner) writeRows(ctx context.Context, venue string, rows []row) error {
	for _, x := range rows {
		if _, err := r.pool.Exec(ctx, `INSERT INTO md_venue_funding(ts,venue,inst,funding_rate,next_funding,interval_h,mark,index)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, r.now(), venue, x.inst, x.rate, x.next, x.interval, x.mark, x.index); err != nil {
			r.rej.Inc("write_err")
			return err
		}
	}
	return nil
}

// f2p 字符串→*float64（空串/缺=nil——不虚构数值）。
func f2p(s string) *float64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

// collectBinance fapi premiumIndex（lastFundingRate+markPrice+indexPrice 全符号一拍）。
func collectBinance(ctx context.Context, f Fetcher) ([]row, error) {
	b, err := f(ctx, "https://fapi.binance.com/fapi/v1/premiumIndex", "")
	if err != nil {
		return nil, err
	}
	var xs []struct {
		Symbol      string `json:"symbol"`
		MarkPrice   string `json:"markPrice"`
		IndexPrice  string `json:"indexPrice"`
		FundingRate string `json:"lastFundingRate"`
		NextTime    int64  `json:"nextFundingTime"`
		FundingIntH int    `json:"fundingIntervalHours"`
	}
	if err := json.Unmarshal(b, &xs); err != nil {
		return nil, err
	}
	out := make([]row, 0, len(xs))
	for _, x := range xs {
		if !strings.HasSuffix(x.Symbol, "USDT") {
			continue
		}
		out = append(out, row{inst: x.Symbol, rate: f2p(x.FundingRate),
			interval: float64(x.FundingIntH), mark: f2p(x.MarkPrice), index: f2p(x.IndexPrice)})
	}
	return out, nil
}

// collectBybit v5 linear tickers（fundingRate+nextFundingTime+markPrice+indexPrice）。
func collectBybit(ctx context.Context, f Fetcher) ([]row, error) {
	b, err := f(ctx, "https://api.bybit.com/v5/market/tickers?category=linear&settleCoin=USDT", "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Result struct {
			List []struct {
				Symbol      string `json:"symbol"`
				FundingRate string `json:"fundingRate"`
				MarkPrice   string `json:"markPrice"`
				IndexPrice  string `json:"indexPrice"`
				NextTime    string `json:"nextFundingTime"`
				FundingIntH string `json:"fundingIntervalHour"`
			} `json:"list"`
		} `json:"result"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, err
	}
	out := make([]row, 0, len(resp.Result.List))
	for _, x := range resp.Result.List {
		out = append(out, row{inst: x.Symbol, rate: f2p(x.FundingRate),
			interval: parseHours(x.FundingIntH), mark: f2p(x.MarkPrice), index: f2p(x.IndexPrice)})
	}
	return out, nil
}

// collectHyperliquid /info metaAndAssetCtxs（funding+markPx+oraclePx 全资产一拍）。
func collectHyperliquid(ctx context.Context, f Fetcher) ([]row, error) {
	b, err := f(ctx, "https://api.hyperliquid.xyz/info", `{"type":"metaAndAssetCtxs"}`)
	if err != nil {
		return nil, err
	}
	var resp []json.RawMessage
	if err := json.Unmarshal(b, &resp); err != nil || len(resp) != 2 {
		return nil, fmt.Errorf("metaAndAssetCtxs 形状异: %v", err)
	}
	var meta struct {
		Universe []struct {
			Name string `json:"name"`
		} `json:"universe"`
	}
	var ctxs []struct {
		Funding string `json:"funding"`
		MarkPx  string `json:"markPx"`
		OraPx   string `json:"oraclePx"`
	}
	if err := json.Unmarshal(resp[0], &meta); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(resp[1], &ctxs); err != nil {
		return nil, err
	}
	out := make([]row, 0, len(meta.Universe))
	for i, u := range meta.Universe {
		if i >= len(ctxs) {
			break
		}
		out = append(out, row{inst: u.Name + "USDT", rate: f2p(ctxs[i].Funding),
			interval: 1, mark: f2p(ctxs[i].MarkPx), index: f2p(ctxs[i].OraPx)})
	}
	return out, nil
}

func parseHours(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
