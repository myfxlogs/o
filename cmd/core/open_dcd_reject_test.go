// open_dcd_reject_test.go —— T-0259 桩锚：dcdOpen quote/trade 步 venue 拒因
// 持久化（非仅日志）——quote 上游拦→fin_eval_reject；trade 步→fin_execution
// .venue_err_* 随标定同写。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/portfolio/lifecycle"
	"o/internal/risk"
	"o/internal/store/pgstore"
	"o/internal/trading/okxexec"
)

// stubDcdVenue dcd 三步缝伪件（dcdVenue 接口注入——venue 拒因可控注入）。
type stubDcdVenue struct {
	qid  string
	qerr error
	traw json.RawMessage
	terr error
	sraw json.RawMessage
	serr error
}

func (s stubDcdVenue) DcdQuote(context.Context, string, string, string) (string, error) {
	return s.qid, s.qerr
}
func (s stubDcdVenue) DcdTrade(context.Context, string) (json.RawMessage, error) {
	return s.traw, s.terr
}
func (s stubDcdVenue) DcdOrderStatus(context.Context, string) (json.RawMessage, error) {
	return s.sraw, s.serr
}

// dcdRejectFixture 迁移+产品行+闸放行装配（risk.* 键=0008 迁移种值，equity
// 现读件注入大额放行——本桩面只验持久化非闸判据）。
func dcdRejectFixture(t *testing.T, productID string, stub stubDcdVenue) (context.Context, *pgxpool.Pool, hedgeOpener) {
	t.Helper()
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_dcd_product(ts,venue,surface,uly,product_id,direction,raw)
		VALUES(now(),'okx','dcd','BTC-USDT',$1,'C','{"notionalCcy":"BTC"}')`, productID); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	gate := risk.NewGate(pool, cfg).WithEquity(func(context.Context, string) (float64, error) {
		return 1e6, nil
	})
	return ctx, pool, hedgeOpener{pool: pool, cfg: cfg, gate: gate, dcd: stub}
}

func dcdSpec(productID string) lifecycle.OpenSpec {
	return lifecycle.OpenSpec{StrategyRef: 1, Venue: "okx", InstID: productID,
		Direction: "long_call", NotionalUSD: 100, Template: "dcd_call"}
}

// 锚①：quote 步 52927 拒（exec 行未建上游拦）→fin_eval_reject 可查 venue
// 码+msg；重试幂等行 hits 累计不洪行。
func TestDcdQuoteRejectPersist(t *testing.T) {
	pid := "BTC-USDT-261225-95000-C"
	stub := stubDcdVenue{qerr: &okxexec.APIError{
		Code: "1", Msg: "shell", SCode: "52927", SMsg: "no quote available"}}
	ctx, pool, h := dcdRejectFixture(t, pid, stub)
	for i := 0; i < 2; i++ {
		if _, err := h.dcdOpen(ctx, dcdSpec(pid)); err == nil {
			t.Fatal("quote 拒应上抛")
		}
	}
	var reason, vc, vm string
	var hits int
	if err := pool.QueryRow(ctx, `SELECT reason, evidence->>'venue_code',
		evidence->>'venue_msg', (evidence->>'hits')::int FROM fin_eval_reject
		WHERE inst_id=$1`, pid).Scan(&reason, &vc, &vm, &hits); err != nil {
		t.Fatalf("拒因行应可查（非仅日志）: %v", err)
	}
	if reason != "dcd_quote_reject:52927" || vc != "52927" || vm != "no quote available" || hits != 2 {
		t.Fatalf("拒因行错位: reason=%q vc=%q vm=%q hits=%d", reason, vc, vm, hits)
	}
	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM fin_execution").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("上游拦不应产 exec 行（幻影敞口）: %d", n)
	}
}

// 锚②③：trade 步 venue 拒→fin_execution.venue_err_* 随标定同写（同
// UPDATE=同原子——断言同行同取）。三态：envelope 拒=ambiguous+内层码；
// per-item sCode 拒=failed+原码；429 限速=failed+原文码 NULL 不臆造。
func TestDcdTradeRejectPersist(t *testing.T) {
	cases := []struct {
		name      string
		pid       string
		stub      stubDcdVenue
		wantState string
		wantCode  *string
		wantMsg   string
	}{
		{"envelope 壳码内层真因", "BTC-USDT-261225-95000-C", stubDcdVenue{qid: "q-1",
			terr: &okxexec.APIError{Code: "1", Msg: "shell", SCode: "51008", SMsg: "Insufficient balance"}},
			"ambiguous", strPtr("51008"), "Insufficient balance"},
		{"per-item sCode 业务拒", "BTC-USDT-261225-96000-C", stubDcdVenue{qid: "q-2",
			traw: json.RawMessage(`[{"ordId":"","sCode":"52927","sMsg":"no quote left"}]`)},
			"failed", strPtr("52927"), "no quote left"},
		{"429 限速拒", "BTC-USDT-261225-97000-C", stubDcdVenue{qid: "q-3",
			terr: fmt.Errorf("http 429: too many requests")},
			"failed", nil, "http 429: too many requests"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, pool, h := dcdRejectFixture(t, tc.pid, tc.stub)
			eid, err := h.dcdOpen(ctx, dcdSpec(tc.pid))
			if err == nil || eid == 0 {
				t.Fatalf("trade 拒应上抛且 exec 行在: eid=%d err=%v", eid, err)
			}
			var state string
			var vc, vm *string
			if err := pool.QueryRow(ctx, `SELECT state, venue_err_code, venue_err_msg
				FROM fin_execution WHERE id=$1`, eid).Scan(&state, &vc, &vm); err != nil {
				t.Fatal(err)
			}
			if state != tc.wantState {
				t.Fatalf("state=%q want %q", state, tc.wantState)
			}
			if (vc == nil) != (tc.wantCode == nil) || (vc != nil && *vc != *tc.wantCode) {
				t.Fatalf("venue_err_code=%v want %v", vc, tc.wantCode)
			}
			if vm == nil || *vm != tc.wantMsg {
				t.Fatalf("venue_err_msg=%v want %q", vm, tc.wantMsg)
			}
		})
	}
}

func strPtr(s string) *string { return &s }
