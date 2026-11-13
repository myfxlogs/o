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
