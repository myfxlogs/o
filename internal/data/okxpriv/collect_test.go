package okxpriv

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"phase/internal/platform/testutil"
	"phase/internal/store/pgstore"
)

// fakePrivWS 抬升→读 login+subscribe→推 balance_and_position 帧。
func fakePrivWS(t *testing.T, loginOK bool, frames []string) *httptest.Server {
	t.Helper()
	up := websocket.Upgrader{}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if _, _, err := conn.ReadMessage(); err != nil { // login
			return
		}
		// login ack 契约（生产=先 ack 后订阅，T-0008 竞态修入的 ack 先行语义——
		// 桩不回 ack=握手不对称挂死，即 CI 红真根）。
		if !loginOK {
			_ = conn.WriteMessage(websocket.TextMessage,
				[]byte(`{"event":"login","code":"60011","msg":"bad key"}`))
			return
		}
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"event":"login","code":"0","msg":""}`))
		if _, _, err := conn.ReadMessage(); err != nil { // subscribe
			return
		}
		for _, f := range frames {
			_ = conn.WriteMessage(websocket.TextMessage, []byte(f))
		}
		time.Sleep(5 * time.Second) // 挂住连接供断言窗（2s 边际过窄——竞态坑）
	}))
}

func TestPrivE2E(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	// 专属标记锚（共享测试实例——残留行假绿坑，断言只认本拍注入件）
	evt := `{"arg":{"channel":"balance_and_position"},"data":[{"ccy":"TSTCCY","instId":"TESTPRIV-USDT-SWAP","eqUsd":"21.6","availEq":"20"}]}`
	for _, q := range []string{
		"DELETE FROM md_account_events WHERE inst_id='TESTPRIV-USDT-SWAP'",
		"DELETE FROM md_account_events WHERE ccy='TSTCCY'"} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	srv := fakePrivWS(t, true, []string{evt})
	defer srv.Close()

	c := &Collector{pool: pool, url: "ws" + strings.TrimPrefix(srv.URL, "http"),
		lastEq: map[string]float64{}}
	c.rej = newTestRej()
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()
	go c.dialLoop(ctx2)

	// 事件驱动断言（禁固定睡眠——dial/login/ack/订阅/推帧/落库全异步链，
	// 固定 1.5s 在 CI 慢机竞态必炸：坑根=按时序假设而非就绪条件断言）。
	deadline := time.Now().Add(10 * time.Second)
	var n int
	var eq float64
	for {
		_ = pool.QueryRow(ctx,
			"SELECT count(*) FROM md_account_events WHERE inst_id='TESTPRIV-USDT-SWAP'").Scan(&n)
		_ = pool.QueryRow(ctx,
			"SELECT COALESCE((data->>'eqUsd')::float,(data->>'eq')::float) FROM md_account_events WHERE venue='okx' AND ccy='TSTCCY' ORDER BY ts DESC LIMIT 1").Scan(&eq)
		if n > 0 && eq == 21.6 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("链路未通 n=%d eq=%v（事件行/快照行双锚超时）", n, eq)
		}
		time.Sleep(150 * time.Millisecond)
	}
}

// TestAccountDetailsSnapshot account 频道 details[] 实拍形状（生产库 md_account_events
// 亲证 2026-10-23）→per-ccy equity+avail 双值落快照行（扫额闲余判据取值源）。
func TestAccountDetailsSnapshot(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		"DELETE FROM md_account_events WHERE ccy IN ('TSTCCY2','TSTCCY3')",
		"DELETE FROM md_account_events WHERE ccy IN ('TSTCCY2','TSTCCY3')"} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	defer pool.Exec(ctx, "DELETE FROM md_account_events WHERE ccy IN ('TSTCCY2','TSTCCY3')")
	defer pool.Exec(ctx, "DELETE FROM md_account_events WHERE ccy IN ('TSTCCY2','TSTCCY3')")
	frame := `{"arg":{"channel":"account"},"data":[{"uTime":"1792769662989","details":[
		{"ccy":"TSTCCY2","eq":"100.5","eqUsd":"100.5","availEq":"80.25","availBal":"80.25"},
		{"ccy":"TSTCCY3","eq":"7","eqUsd":"9","availEq":"5","availBal":"5"}]}]}`
	c := &Collector{pool: pool, lastEq: map[string]float64{}}
	c.rej = newTestRej()
	c.handle(ctx, []byte(frame))
	var eq, avail float64
	if err := pool.QueryRow(ctx, `SELECT (d->>'eq')::float,COALESCE((d->>'availEq')::float,(d->>'availBal')::float)
		FROM md_account_events e CROSS JOIN LATERAL jsonb_array_elements(e.data->'details') d
		WHERE e.channel='account' AND d->>'ccy'='TSTCCY2' ORDER BY e.ts DESC LIMIT 1`).Scan(&eq, &avail); err != nil || eq != 100.5 || avail != 80.25 {
		t.Fatalf("TSTCCY2 eq=%v avail=%v err=%v", eq, avail, err)
	}
	if err := pool.QueryRow(ctx, `SELECT (d->>'eq')::float,COALESCE((d->>'availEq')::float,(d->>'availBal')::float)
		FROM md_account_events e CROSS JOIN LATERAL jsonb_array_elements(e.data->'details') d
		WHERE e.channel='account' AND d->>'ccy'='TSTCCY3' ORDER BY e.ts DESC LIMIT 1`).Scan(&eq, &avail); err != nil || eq != 7 || avail != 5 {
		t.Fatalf("TSTCCY3 eq=%v（ccy 本位口径优先）avail=%v err=%v", eq, avail, err)
	}
}

// T-0044 锚：余额/持仓变动帧触发 bills 记账钩（事件驱动——REST 非轮询主径）。
func TestBillsHookOnAccountFrame(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	c := &Collector{pool: pool, rej: newTestRej(), lastEq: map[string]float64{}}
	hit := make(chan struct{}, 4)
	c.SetBillsHook(func(context.Context) { hit <- struct{}{} })
	frame := []byte(`{"arg":{"channel":"account"},"data":[{"ccy":"USDT","eqUsd":"100"}]}`)
	c.handle(ctx, frame)
	select {
	case <-hit:
	case <-time.After(2 * time.Second): // sched-exempt-ok 测试等待异步钩——非周期调度
		t.Fatal("account 帧未触发 bills 钩")
	}
	// 节流窗内第二帧不再触发（30s 窗）。
	c.handle(ctx, frame)
	select {
	case <-hit:
		t.Fatal("节流窗内不应重复触发")
	case <-time.After(200 * time.Millisecond): // sched-exempt-ok 测试等待节流窗——非周期调度
	}
	// 非余额面频道（未知频道）不触发。
	c2 := &Collector{pool: pool, rej: newTestRej(), lastEq: map[string]float64{}}
	c2.SetBillsHook(func(context.Context) { t.Fatal("无关频道不应触发") })
	c2.handle(ctx, []byte(`{"arg":{"channel":"orders"},"data":[{"instId":"X"}]}`))
}
