// collect6.go —— T-0313 六候选 adapter（行数闸拆分；签名单册在 collect.go）。
package scout

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// collectBitget v2 mix tickers USDT-FUTURES（fundingRate+markPrice+indexPrice
// 同帧——interval 端点无档如实 0 走 config 回退；种子端点实测 2026-11-03）。
func collectBitget(ctx context.Context, f Fetcher) ([]row, error) {
	b, err := f(ctx, "https://api.bitget.com/api/v2/mix/market/tickers?productType=USDT-FUTURES", "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data []struct {
			Symbol  string `json:"symbol"`
			Mark    string `json:"markPrice"`
			Index   string `json:"indexPrice"`
			Funding string `json:"fundingRate"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, err
	}
	out := make([]row, 0, len(resp.Data))
	for _, x := range resp.Data {
		out = append(out, row{inst: x.Symbol, rate: f2p(x.Funding),
			mark: f2p(x.Mark), index: f2p(x.Index)})
	}
	return out, nil
}

// collectGate v4 futures/usdt tickers（funding_rate+mark_price+index_price+contract
// 方言——interval 端点无档如实 0；种子端点实测 2026-11-03）。
func collectGate(ctx context.Context, f Fetcher) ([]row, error) {
	b, err := f(ctx, "https://api.gateio.ws/api/v4/futures/usdt/tickers", "")
	if err != nil {
		return nil, err
	}
	var xs []struct {
		Contract string `json:"contract"`
		Funding  string `json:"funding_rate"`
		Mark     string `json:"mark_price"`
		Index    string `json:"index_price"`
	}
	if err := json.Unmarshal(b, &xs); err != nil {
		return nil, err
	}
	out := make([]row, 0, len(xs))
	for _, x := range xs {
		out = append(out, row{inst: x.Contract, rate: f2p(x.Funding),
			mark: f2p(x.Mark), index: f2p(x.Index)})
	}
	return out, nil
}

// collectKucoin contracts/active（fundingFeeRate+currentFundingRateGranularity
// ms 小时换算 per-inst 实取混合档——1h/4h/8h 并存如实不虚构；markPrice/indexPrice
// 同帧；种子端点单 symbol 弃换 active 全量并记 evidence——票面钉注①）。
func collectKucoin(ctx context.Context, f Fetcher) ([]row, error) {
	b, err := f(ctx, "https://api-futures.kucoin.com/api/v1/contracts/active", "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data []struct {
			Symbol  string   `json:"symbol"`
			Mark    float64  `json:"markPrice"`
			Index   float64  `json:"indexPrice"`
			Funding *float64 `json:"fundingFeeRate"`
			GranMs  float64  `json:"currentFundingRateGranularity"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, err
	}
	out := make([]row, 0, len(resp.Data))
	for _, x := range resp.Data {
		out = append(out, row{inst: x.Symbol, rate: x.Funding,
			interval: x.GranMs / 3600000, // ms→h per-inst 实取
			mark:     f2f(x.Mark), index: f2f(x.Index)})
	}
	return out, nil
}

// collectMexc contract/ticker（fundingRate+indexPrice+fairPrice→mark——
// fairPrice=标记价实测在帧；interval 无档如实 0；种子端点实测 2026-11-03）。
func collectMexc(ctx context.Context, f Fetcher) ([]row, error) {
	b, err := f(ctx, "https://contract.mexc.com/api/v1/contract/ticker", "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data []struct {
			Symbol  string  `json:"symbol"`
			Funding float64 `json:"fundingRate"`
			Fair    float64 `json:"fairPrice"`
			Index   float64 `json:"indexPrice"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, err
	}
	out := make([]row, 0, len(resp.Data))
	for _, x := range resp.Data {
		out = append(out, row{inst: x.Symbol, rate: f2f(x.Funding),
			mark: f2f(x.Fair), index: f2f(x.Index)})
	}
	return out, nil
}

// collectDydx indexer v4 perpetualMarkets（nextFundingRate=1h 档率——dYdX
// 逐时结；oraclePrice=index 正名，mark 无独立帧如实 nil；种子端点实测）。
func collectDydx(ctx context.Context, f Fetcher) ([]row, error) {
	b, err := f(ctx, "https://indexer.dydx.trade/v4/perpetualMarkets", "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Markets map[string]struct {
			Ticker string `json:"ticker"`
			Rate   string `json:"nextFundingRate"`
			Oracle string `json:"oraclePrice"`
		} `json:"markets"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, err
	}
	out := make([]row, 0, len(resp.Markets))
	for _, x := range resp.Markets {
		out = append(out, row{inst: x.Ticker, rate: f2p(x.Rate),
			interval: 1, index: f2p(x.Oracle)}) // 1h 档钉（dYdX 逐时结）
	}
	return out, nil
}

// collectDeribit -PERPETUAL 族 ticker（get_instruments kind=future expired=false
// 筛 PERPETUAL→逐件 ticker 拉 funding_8h/mark/index；连续结算 interval 如实 0
// 走 config 回退——票面钉注③不虚构小时档；种子端点 history 型换 instruments
// +ticker 径记 evidence）。
var deribitCCYs = []string{"BTC", "ETH", "SOL", "XRP", "ADA", "DOGE"}

func collectDeribit(ctx context.Context, f Fetcher) ([]row, error) {
	var out []row
	for _, ccy := range deribitCCYs {
		b, err := f(ctx, "https://www.deribit.com/api/v2/public/get_instruments?currency="+ccy+"&kind=future&expired=false", "")
		if err != nil {
			continue // 单币种挂不拖全局
		}
		var resp struct {
			Result []struct {
				Name string `json:"instrument_name"`
			} `json:"result"`
		}
		if err := json.Unmarshal(b, &resp); err != nil {
			continue
		}
		for _, ins := range resp.Result {
			if !strings.HasSuffix(ins.Name, "-PERPETUAL") {
				continue
			}
			tb, err := f(ctx, "https://www.deribit.com/api/v2/public/ticker?instrument_name="+ins.Name, "")
			if err != nil {
				continue
			}
			var t struct {
				Result struct {
					F8H   float64 `json:"funding_8h"`
					Mark  float64 `json:"mark_price"`
					Index float64 `json:"index_price"`
				} `json:"result"`
			}
			if err := json.Unmarshal(tb, &t); err != nil {
				continue
			}
			out = append(out, row{inst: ins.Name, rate: f2f(t.Result.F8H / 8 * 1), // 8h 累积→等效 1h 率如实换算
				mark: f2f(t.Result.Mark), index: f2f(t.Result.Index)})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("deribit 全币种空")
	}
	return out, nil
}

func f2f(v float64) *float64 { return &v }
