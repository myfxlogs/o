// Package dcd —— 探员双币赢枚举臂（T-0102，research 第二模板）：
// 枚举面从能力面注册表 fin_capability_surface 读（v2 钦定——零硬编码端点清单），
// 扫 GET <endpoint_family>/currency-pair → products?baseCcy&quoteCcy&optType 全对在架产品（三参方言——uly 伪参 50014 实证），
// 评估落 fin_opp_eval（template=dcd_call/dcd_put）→阈闸进 fin_proposal 同链。
// 模板语义：C=锁币卖 call（向上让渡换权利金）、P=锁 USDT 卖 put（向下接盘价
// 换权利金）——dca/卖权 carry 族，非腿对冲。
// 禁线：quote/trade/redeem 写面零调用（钱径另票批）；dcd 持仓非仓位（T-0101）。
package dcd

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Surface 能力面注册表行（枚举面唯一真源——注册+模板适用声明=探员即枚举）。
type Surface struct {
	Venue          string
	Surface        string
	EndpointFamily string
	ProductFamily  string
	Enabled        bool
}

// Product 在架产品行（feed 件产出——生产=REST products?uly=，测试=夹具）。
type Product struct {
	ProductID       string
	Uly             string
	QuoteCcy        string // 计价/锁仓币（USDT/USDG/USDC/RLUSD 族；P 产品锁此币）
	Direction       string // C=call / P=put
	Strike          float64
	Expiry          time.Time
	AnnualizedYield float64
	MinSize         float64
	MaxSize         float64
	Raw             json.RawMessage
}

// Feed 枚举读件（注入缝——research 域零 venue 依赖，红线 research↛trading；
// 生产闭包=cmd 装配 REST 签名读，测试=夹具）。读败=err 上抛 fail-closed。
type Feed func(ctx context.Context, s Surface) ([]Product, error)

// Pair currency-pair 行（baseCcy+quoteCcy+optType 三元组=products 调用参数形）。
type Pair struct {
	BaseCcy  string
	QuoteCcy string
	OptType  string // C/P
}

// ParseCurrencyPairs currency-pair 响应解析（宽松——三元组缺件行弃）。
func ParseCurrencyPairs(raw []byte) ([]Pair, error) {
	var rows []struct {
		BaseCcy  string `json:"baseCcy"`
		QuoteCcy string `json:"quoteCcy"`
		OptType  string `json:"optType"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("currency-pair parse: %w", err)
	}
	var out []Pair
	seen := map[Pair]bool{}
	for _, r := range rows {
		p := Pair{r.BaseCcy, r.QuoteCcy, r.OptType}
		if p.BaseCcy == "" || p.QuoteCcy == "" || (p.OptType != "C" && p.OptType != "P") || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out, nil
}

// ParseProducts products 响应解析（宽松——strike/expiry 缺=零值，评估面
// unevaluable 兜底；expiryTime=毫秒数字符串 venue 方言）。
func ParseProducts(raw []byte) ([]Product, error) {
	var env struct {
		Products []struct {
			ProductID       string          `json:"productId"`
			Uly             string          `json:"uly"`
			QuoteCcy        string          `json:"quoteCcy"`
			Direction       string          `json:"optType"`
			Strike          string          `json:"strike"`
			ExpiryTime      string          `json:"expTime"`
			AnnualizedYield string          `json:"annualizedYield"`
			MinSize         string          `json:"minSize"`
			MaxSize         string          `json:"maxSize"`
			Raw             json.RawMessage `json:"-"`
		} `json:"products"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("products parse: %w", err)
	}
	rows := env.Products // data={"products":[...]} 对象包络（venue 方言）
	var docs []json.RawMessage
	var env2 struct {
		Products []json.RawMessage `json:"products"`
	}
	_ = json.Unmarshal(raw, &env2)
	docs = env2.Products
	out := make([]Product, 0, len(rows))
	for i, r := range rows {
		if r.ProductID == "" || (r.Direction != "C" && r.Direction != "P") {
			continue
		}
		p := Product{ProductID: r.ProductID, Uly: r.Uly, QuoteCcy: r.QuoteCcy, Direction: r.Direction,
			Strike: parseF(r.Strike), AnnualizedYield: parseF(r.AnnualizedYield),
			MinSize: parseF(r.MinSize), MaxSize: parseF(r.MaxSize)}
		if ms, err := strconv.ParseInt(r.ExpiryTime, 10, 64); err == nil && ms > 0 {
			p.Expiry = time.UnixMilli(ms).UTC()
		}
		if i < len(docs) {
			p.Raw = docs[i]
		}
		out = append(out, p)
	}
	return out, nil
}

func parseF(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
