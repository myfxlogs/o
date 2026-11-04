// remnant_test.go —— T-0327 残仓类级断言（锚②）：venue 在持但无活策略归属
// 的仓工件面/在持面必现；有归属仓不双行；读件缺=显缺行不静默。
package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"o/internal/trading/okxexec"

	"github.com/jackc/pgx/v5/pgxpool"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// remSvc 残仓桩装（posLive/dcdLive/mt5Live 三面注入——nil=该面缺席跳臂）。
func remSvc(pool *pgxpool.Pool,
	pos []okxexec.Position, posErr error,
	dcds []okxexec.DcdOrder, dcdErr error,
	accts []mt5Acct) coreService {
	return coreService{pool: pool,
		posLive: func(context.Context) ([]okxexec.Position, error) { return pos, posErr },
		dcdLive: func(context.Context) ([]okxexec.DcdOrder, error) { return dcds, dcdErr },
		mt5Live: func(context.Context) ([]mt5Acct, error) { return accts, nil }}
}

// T-0327 锚②类级断言：venue 在持无活策略归属→工件面必现「残仓」；
// 归属仓（未平仓 inst/活策略 inst/腿 inst）不标残仓不双行。
func TestVenueRemnantWorkItems(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	// 归属面：未平仓 ETH-USDT + 活策略 SOL-USDT-SWAP + 纸面策略 DOGE-USDT-SWAP
	// （paper 不出资不归属——同 inst venue 仓仍残）。
	openEthLeg(t, ctx, pool)
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue)
		VALUES('SOL-USDT-SWAP','guard-test','live',25,'okx'),
		      ('DOGE-USDT-SWAP','guard-test','paper',25,'okx')`); err != nil {
		t.Fatal(err)
	}
	svc := remSvc(pool,
		[]okxexec.Position{
			{InstID: "ETH-USDT"},       // 归属=未平仓 inst
			{InstID: "SOL-USDT-SWAP"},  // 归属=活策略 inst
			{InstID: "DOGE-USDT-SWAP"}, // 残仓=纸面策略不归属
			{InstID: "XRP-USDT-SWAP"}}, // 残仓=零归属
		nil,
		[]okxexec.DcdOrder{dcdFixtureOrder()}, nil, // 残仓=无策略归属 dcd 产品
		[]mt5Acct{{Login: "1", Equity: 1, Orders: []mt5Ord{
			{Symbol: "XAUUSD", Ticket: 99, At: time.Now()}}}}) // 残仓=MT5 仓
	r, err := svc.GetFlowPulse(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	type pk struct{ ref, stage string }
	got := map[pk]bool{}
	for _, w := range r.WorkItems {
		got[pk{w.Ref, w.Stage}] = true
	}
	want := []pk{
		{"ETH-USDT", "持仓中"},                   // 归属仓正行非残
		{"OKX DOGE-USDT-SWAP", "残仓"},          // 纸面策略=非活不归属
		{"OKX XRP-USDT-SWAP", "残仓"},           // 零归属 venue 仓
		{"DCD BTC-USDT-261225-95000-C", "残仓"}, // 本案锚——dcd 残仓工件面必现
		{"MT5 XAUUSD #99", "残仓"},              // MT5 无归属仓
	}
	for _, w := range want {
		if !got[w] {
			t.Fatalf("工件缺 %+v（got=%v）", w, got)
		}
	}
	for _, nw := range []pk{{"ETH-USDT", "残仓"}, {"OKX ETH-USDT", "残仓"},
		{"OKX SOL-USDT-SWAP", "残仓"}, {"SOL-USDT-SWAP", "残仓"}} {
		if got[nw] {
			t.Fatalf("归属仓误标残仓 %+v", nw)
		}
	}
}

// T-0327 锚②读件缺面：venue 源读败=显缺行不静默（静默=幽灵态复辟）。
func TestVenueRemnantMissing(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	svc := remSvc(pool, nil, fmt.Errorf("positions 429"),
		nil, fmt.Errorf("dcd 429"), nil)
	r, err := svc.GetFlowPulse(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var miss []string
	for _, w := range r.WorkItems {
		if w.Stage == "读件缺" {
			miss = append(miss, w.Ref)
		}
	}
	if len(miss) != 2 {
		t.Fatalf("两源缺应两显缺行: %v", miss)
	}
}

// T-0327 在持面残仓臂：doing 列表 venue 残仓入列+读件缺显形+真空不冒静持；
// dcd 残仓由上臂行承载不双行。
func TestOverviewDoingRemnant(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,inst_id,last) VALUES(now(),'BTC-USDT',80000)`); err != nil {
		t.Fatal(err)
	}
	ord := dcdFixtureOrder()
	svc := remSvc(pool,
		[]okxexec.Position{{InstID: "XRP-USDT-SWAP", CTime: "1792585306000"}}, nil,
		[]okxexec.DcdOrder{ord}, nil,
		[]mt5Acct{{Login: "1", Equity: 1, Orders: []mt5Ord{{Symbol: "XAUUSD", Ticket: 7}}}})
	o, err := svc.Overview(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	tx := doingTexts(o)
	for _, want := range []string{
		"DCD BTC-USDT-261225-95000-C", // dcd 臂正行
		"OKX XRP-USDT-SWAP", "残仓在持",   // OKX 残仓臂行
		"MT5 XAUUSD #7", "无活策略归属"} { // MT5 残仓臂行
		if !strings.Contains(tx, want) {
			t.Fatalf("在持面缺 %q: %q", want, tx)
		}
	}
	if strings.Contains(tx, "静持") {
		t.Fatalf("有残仓不得显静持: %q", tx)
	}
	// dcd 残仓不双行（上臂已载——同品只出一行）。
	n := 0
	for _, d := range o.Doing {
		if strings.Contains(d.Text, "BTC-USDT-261225-95000-C") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("dcd 残仓应恰一行: %d", n)
	}
}

// T-0327 静持边界：venue 面全空+归属集空=静持依旧（残仓臂零噪声）。
func TestOverviewDoingStillIdle(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	svc := remSvc(pool, []okxexec.Position{}, nil, nil, nil, nil)
	o, err := svc.Overview(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Doing) != 1 || !strings.Contains(o.Doing[0].Text, "静持") {
		t.Fatalf("真空应恰静持一行: %+v", o.Doing)
	}
}
