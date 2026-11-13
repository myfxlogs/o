// klines_test.go —— 工单#13065 日K数据面桩：数组载荷解析/收线帧节流豁免/
// md_klines 同键 upsert（形成中覆写→收线终态）。DB 桩 O_TEST_PG_DSN 缺=skip。
package okxws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func TestParseCandle(t *testing.T) {
	raw := json.RawMessage(`["1761000000000","67000","67500","66800","67200","123.4","8300000","0","0"]`)
	k := parseCandle("BTC-USDT-SWAP", "1D", raw)
	if k == nil || k.ts.UnixMilli() != 1761000000000 || k.o != 67000 || k.c != 67200 || k.confirm {
		t.Fatalf("形成中帧: %+v", k)
	}
	raw = json.RawMessage(`["1760913600000","66500","67100","66000","67000","200","13300000","0","1"]`)
	k = parseCandle("BTC-USDT-SWAP", "1D", raw)
	if k == nil || !k.confirm || k.h != 67100 || k.volCcy != 13300000 {
		t.Fatalf("收线帧: %+v", k)
	}
	for _, bad := range []string{`{"ts":"1"}`, `["x","1","1","1","1","1","1","0","0"]`, `["0","1","1","1","1","1","1","0","0"]`} {
		if parseCandle("BTC-USDT-SWAP", "1D", json.RawMessage(bad)) != nil {
			t.Fatalf("畸形帧应收 nil: %s", bad)
		}
	}
}

// 收线帧节流豁免——confirm 帧在 3s 闸窗内也必落（收线终态不可丢）；
// 形成中帧同键 3s 节流（写洪防）。
func TestKlineThrottleConfirmBypass(t *testing.T) {
	c := &Collector{passAt: map[string]time.Time{}}
	form := rowItem{kline: &klineRow{instID: "BTC-USDT-SWAP", bar: "1D"}}
	done := rowItem{kline: &klineRow{instID: "BTC-USDT-SWAP", bar: "1D", confirm: true}}
	if !c.passThrottle(form) {
		t.Fatal("首帧应过")
	}
	if c.passThrottle(form) {
		t.Fatal("形成中同 inst 3s 内应跳帧")
	}
	if !c.passThrottle(done) {
		t.Fatal("收线帧节流窗内仍必落")
	}
}

// md_klines upsert 桩：形成中帧覆写同键→收线帧终态落——行数恒一。
func TestKlineStoreUpsert(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	// fin 域桩——SG collector 写库=fin 链（md_klines=bridge 双链同形；
	// md 域单链会致生产 write_err，本桩钉 SG 写侧 schema 实存）。
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM md_klines WHERE inst_id='XKL-USDT-SWAP'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	c := &Collector{pool: pool}
	ts := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)
	rows := func(c4, confirm float64) []rowItem {
		return []rowItem{{kline: &klineRow{instID: "XKL-USDT-SWAP", bar: "1D",
			ts: ts, o: 1, h: 2, l: 0.5, c: c4, vol: 10, volCcy: 100, confirm: confirm == 1}}}
	}
	if err := c.insertBatch(ctx, rows(1.5, 0)); err != nil {
		t.Fatalf("forming: %v", err)
	}
	if err := c.insertBatch(ctx, rows(1.9, 1)); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	var n int
	var close float64
	var confirm bool
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM md_klines WHERE inst_id='XKL-USDT-SWAP'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("行数=%d err=%v（同键应覆写非追加）", n, err)
	}
	if err := pool.QueryRow(ctx, `SELECT c,confirm FROM md_klines WHERE inst_id='XKL-USDT-SWAP' AND bar='1D'`).Scan(&close, &confirm); err != nil || close != 1.9 || !confirm {
		t.Fatalf("终态 c=%v confirm=%v err=%v", close, confirm, err)
	}
}

// business 臂端对端：klineLoop→klineDial→handle→writer→md_klines 行落
// （fake server 接帧即推——candle 帧走 RawMessage 数组载荷径）。
func TestKlineDialFeedE2E(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM md_klines WHERE inst_id='XKL-USDT-SWAP'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	frame := `{"arg":{"channel":"candle1D","instId":"XKL-USDT-SWAP"},"data":[["1761000000000","67000","67500","66800","67200","123.4","8300000","0","0"]]}`
	up := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.WriteMessage(websocket.TextMessage, []byte(frame))
		time.Sleep(3 * time.Second) // sched-exempt-ok（桩挂连接断言窗）
	}))
	defer srv.Close()

	c := NewCollector(pool, nil)
	c.bizURL = "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()
	go c.writer(ctx2) // 生产径=Run() 起写协程——桩内同构自起（队列消费件）
	go c.klineLoop(ctx2)

	deadline := time.Now().Add(8 * time.Second)
	for {
		var n int
		_ = pool.QueryRow(ctx, "SELECT count(*) FROM md_klines WHERE inst_id='XKL-USDT-SWAP'").Scan(&n)
		if n > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("candle 帧未落 md_klines（business 臂链路断）")
		}
		time.Sleep(150 * time.Millisecond) // sched-exempt-ok（桩断言轮询期界）
	}
}

// 切片③桩：backfillSweep——REST 页滤 confirm=1 收线行+剔库内已有 ts+
// 形成中帧弃（归 WS 臂）；fake restGet 注入免 venue。
func TestKlineBackfillSweep(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM md_klines`); err != nil {
		t.Fatal(err)
	}
	// 预置已收线行 ts=1794412800000（库内已有→应跳过不覆写）。
	if _, err := pool.Exec(ctx,
		`INSERT INTO md_klines(venue,inst_id,bar,ts,o,h,l,c,vol,vol_ccy,confirm,recv_ts)
		 VALUES('okx','XBK-USDT-SWAP','1D',to_timestamp(1794412800000.0/1000),9,9,9,9,9,9,true,now())`); err != nil {
		t.Fatal(err)
	}
	page := `[` +
		`["1794499200000","1","2","1","1.5","10","10","100","0"],` + // 形成中帧→弃
		`["1794412800000","1","2","1","1.5","10","10","100","1"],` + // 已收线在库→跳
		`["1794326400000","1","2","1","1.5","10","10","100","1"],` + // 缺齿→补
		`["1794240000000","1","2","1","1.5","10","10","100","1"]]` // 缺齿→补
	c := NewCollector(pool, nil)
	c.AttachRestGet(func(ctx context.Context, pq string) (json.RawMessage, error) {
		if !strings.Contains(pq, "history-candles") || !strings.Contains(pq, "instId=XBK-USDT-SWAP") {
			t.Fatalf("回填请求形异: %s", pq)
		}
		return json.RawMessage(page), nil
	})
	// 活跃集真源=legset.ActiveSetUnion——桩造 fin_strategy live 行供枚举。
	var sid int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue)
		 VALUES('XBK-USDT-SWAP','backfill-test','live',1,'okx') RETURNING id`).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()
	go c.writer(ctx2)
	c.backfillSweep(ctx2)

	deadline := time.Now().Add(8 * time.Second)
	for {
		var n int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM md_klines WHERE inst_id='XBK-USDT-SWAP'`).Scan(&n)
		if n >= 3 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("回填缺齿未落: n=%d", n)
		}
		time.Sleep(100 * time.Millisecond) // sched-exempt-ok（桩断言轮询期界）
	}
	// 断言①：只补 2 行（预置行不覆写 recv_ts 应仍≈now 原值——count=3=1预置+2补）。
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM md_klines WHERE inst_id='XBK-USDT-SWAP' AND ts=to_timestamp(1794499200000.0/1000)`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("形成中帧被回填覆写——WS 臂主权被侵: n=%d err=%v", n, err)
	}
	// 断言②：两行皆 confirm=true 收线终态。
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM md_klines WHERE inst_id='XBK-USDT-SWAP' AND confirm=false`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("回填落了非收线行: n=%d err=%v", n, err)
	}
}
