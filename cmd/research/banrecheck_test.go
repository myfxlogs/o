// banrecheck_test.go —— O-33393③ 核径桩锚：venue_capability 实探分型
// （在架对>0=证伪/业务拒·面缺席=仍立/非 REST 族=unknown）+apr_regime
// APR 带对拍（≥判据=证伪/<判据=仍立/非 dual-investment=unknown）。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/rejects"
	"o/internal/research/banrecheck"
	"o/internal/trading/okxexec"
)

// brGetter 路径编程 getter（banrecheck 核径桩——path→响应/错误分发行）。
type brGetter struct {
	resp map[string]json.RawMessage
	errs map[string]error
}

func (g brGetter) Get(ctx context.Context, path string) (json.RawMessage, error) {
	for k, e := range g.errs {
		if path == k {
			return nil, e
		}
	}
	if r, ok := g.resp[path]; ok {
		return r, nil
	}
	return nil, fmt.Errorf("unstubbed path %s", path)
}

func brRow(kind, family string) banrecheck.Row {
	return banrecheck.Row{Template: "t", Venue: "okx", Surface: "dcd",
		BanPremiseKind: kind, EndpointFamily: family, ProductFamily: "dual-investment"}
}

func TestProbeVenueCapFalsified(t *testing.T) {
	g := brGetter{resp: map[string]json.RawMessage{
		"/api/v5/finance/sfp/dcd/currency-pair": json.RawMessage(
			`[{"baseCcy":"BTC","quoteCcy":"USDT","optType":"C"},{"baseCcy":"ETH","quoteCcy":"USDT","optType":"P"}]`),
	}}
	res, err := probeVenueCap(context.Background(), g, brRow("venue_capability", "/api/v5/finance/sfp/dcd"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != "falsified" || !strings.Contains(res.Detail, "在架对=2") {
		t.Fatalf("在架对>0 应证伪: %+v", res)
	}
}

func TestProbeVenueCapUpheldBizReject(t *testing.T) {
	// 族错配 51001 型——DCD 族应答=前提仍立（核径查的是对的族）。
	g := brGetter{errs: map[string]error{
		"/api/v5/finance/sfp/dcd/currency-pair": &okxexec.APIError{Code: "51001", Msg: "instrument nonexistent"},
	}}
	res, err := probeVenueCap(context.Background(), g, brRow("venue_capability", "/api/v5/finance/sfp/dcd"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != "upheld" {
		t.Fatalf("业务拒应仍立: %+v", res)
	}
}

func TestProbeVenueCapUpheldAbsent(t *testing.T) {
	g := brGetter{errs: map[string]error{
		"/api/v5/finance/sfp/dcd/currency-pair": errors.New("http 404: not found"),
	}}
	res, err := probeVenueCap(context.Background(), g, brRow("venue_capability", "/api/v5/finance/sfp/dcd"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != "upheld" {
		t.Fatalf("面缺席应仍立: %+v", res)
	}
}

func TestProbeVenueCapTransientYields(t *testing.T) {
	// 瞬态错上抛（fail-closed 下拍重核——不误裁仍立/证伪）。
	g := brGetter{errs: map[string]error{
		"/api/v5/finance/sfp/dcd/currency-pair": errors.New("http 429: too many"),
	}}
	if _, err := probeVenueCap(context.Background(), g, brRow("venue_capability", "/api/v5/finance/sfp/dcd")); err == nil {
		t.Fatal("429 瞬态应上抛")
	}
}

func TestProbeVenueCapNonRESTUnknown(t *testing.T) {
	r := brRow("venue_capability", "mt5-symbol-catalog")
	r.ProductFamily = "symbols"
	res, err := probeVenueCap(context.Background(), brGetter{}, r)
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != "unknown" {
		t.Fatalf("非 REST 族应 unknown: %+v", res)
	}
}

func TestProbeAprRegimeFalsified(t *testing.T) {
	// currency-pair 一对 + products APR=0.15 ≥ 判据 0.04 → 证伪。
	g := brGetter{resp: map[string]json.RawMessage{
		"/api/v5/finance/sfp/dcd/currency-pair": json.RawMessage(
			`[{"baseCcy":"BTC","quoteCcy":"USDT","optType":"C"}]`),
		"/api/v5/finance/sfp/dcd/products?baseCcy=BTC&quoteCcy=USDT&optType=C": json.RawMessage(
			`{"products":[{"productId":"BTC-USDT-1-C","baseCcy":"BTC","quoteCcy":"USDT","optType":"C","strike":"95000","expiry":"1735000000000","annualizedYield":"0.15","minSz":"0.1","maxSz":"5","spot":"88000"}]}`),
	}}
	cfg := config.New(nil) // 无池——GetFloat 键缺席走默认 0.04
	res, err := probeAprRegime(context.Background(), g, cfg, rejects.New("research", "x"),
		brRow("apr_regime", "/api/v5/finance/sfp/dcd"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != "falsified" || !strings.Contains(res.Detail, "APR=0.1500") {
		t.Fatalf("APR≥判据应证伪: %+v", res)
	}
}

func TestProbeAprRegimeUpheld(t *testing.T) {
	// APR=0.01 < 判据 → 仍立。
	g := brGetter{resp: map[string]json.RawMessage{
		"/api/v5/finance/sfp/dcd/currency-pair": json.RawMessage(
			`[{"baseCcy":"BTC","quoteCcy":"USDT","optType":"C"}]`),
		"/api/v5/finance/sfp/dcd/products?baseCcy=BTC&quoteCcy=USDT&optType=C": json.RawMessage(
			`{"products":[{"productId":"BTC-USDT-1-C","strike":"95000","expiry":"1735000000000","annualizedYield":"0.01","minSz":"0.1","maxSz":"5"}]}`),
	}}
	res, err := probeAprRegime(context.Background(), g, config.New(nil),
		rejects.New("research", "x"), brRow("apr_regime", "/api/v5/finance/sfp/dcd"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != "upheld" {
		t.Fatalf("APR<判据应仍立: %+v", res)
	}
}

func TestProbeAprRegimeNonDualUnknown(t *testing.T) {
	r := brRow("apr_regime", "/api/v5/finance/sfp/dcd")
	r.ProductFamily = "symbols"
	res, err := probeAprRegime(context.Background(), brGetter{}, config.New(nil),
		rejects.New("research", "x"), r)
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != "unknown" {
		t.Fatalf("非 dual-investment 应 unknown: %+v", res)
	}
}
