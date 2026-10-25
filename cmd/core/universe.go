// universe.go —— T-0122 宇宙放开件：okx.symbols 含 "*"=全 USDT-SWAP live+
// 季度期货（alias=quarter/USDT 尾——instruments 面实读；凭据缺席期公开面
// 无凭据 PublicGet 直读降级，凭据件只读客同通）。
package main

import (
	"context"
	"encoding/json"
	"strings"

	"o/internal/trading/okxexec"
)

func universeExpander(getCli func() *okxexec.Client) func(ctx context.Context) ([]string, error) {
	return func(ctx context.Context) ([]string, error) {
		cli := getCli()
		if cli == nil {
			cli = okxexec.NewClient(okxexec.Credentials{}, false)
		}
		raw, err := cli.PublicGet(ctx, "/api/v5/public/instruments?instType=SWAP")
		if err != nil {
			return nil, err
		}
		var swap []struct {
			InstID string `json:"instId"`
			State  string `json:"state"`
		}
		if err := json.Unmarshal(raw, &swap); err != nil {
			return nil, err
		}
		var out []string
		seenSpot := map[string]bool{}
		for _, i := range swap {
			if i.State == "live" && strings.HasSuffix(i.InstID, "-USDT-SWAP") {
				out = append(out, i.InstID)
				// T-0129 根修：宇宙放开须含现货腿面——SWAP 基币现货对
				// 同入订阅集（对冲腿现价源非指数价——index-tickers 不可代）。
				base := strings.TrimSuffix(i.InstID, "-USDT-SWAP") + "-USDT"
				if !seenSpot[base] {
					seenSpot[base] = true
					out = append(out, base)
				}
			}
		}
		raw, err = cli.PublicGet(ctx, "/api/v5/public/instruments?instType=FUTURES")
		if err != nil {
			return out, nil // SWAP 全集已得——期货面缺=降级不连带让位
		}
		var fut []struct {
			InstID string `json:"instId"`
			Alias  string `json:"alias"`
			State  string `json:"state"`
		}
		if err := json.Unmarshal(raw, &fut); err == nil {
			for _, i := range fut {
				if i.State == "live" && i.Alias == "quarter" &&
					strings.HasSuffix(i.InstID, "-USDT") {
					out = append(out, i.InstID)
				}
			}
		}
		return out, nil
	}
}

// spotFeedFor 现货现价 REST 读件（T-0134——index-tickers WS 撤下转轮询替；
// 凭据缺席期无凭据 PublicGet 公开面直读同 universeExpander 径）。
func spotFeedFor(getCli func() *okxexec.Client) func(ctx context.Context) ([]byte, error) {
	return func(ctx context.Context) ([]byte, error) {
		cli := getCli()
		if cli == nil {
			cli = okxexec.NewClient(okxexec.Credentials{}, false)
		}
		return cli.PublicGet(ctx, "/api/v5/market/tickers?instType=SPOT")
	}
}
