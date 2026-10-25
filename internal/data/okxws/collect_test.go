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

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// fakeOKX 抬升 ws→读订阅帧→推 tickers+funding 帧（fake server 形证链路，
// venue 帧格式照 OKX v5 实拍 schema）。
func fakeOKX(t *testing.T, frames []string) *httptest.Server {
	t.Helper()
	up := websocket.Upgrader{}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if _, _, err := conn.ReadMessage(); err != nil { // subscribe 帧先读
			return
		}
		for _, f := range frames {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(f)); err != nil {
				return
			}
		}
		time.Sleep(3 * time.Second) // 挂住连接供断言窗
	}))
}

func TestCollectE2E(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app_config(key,value)
		VALUES('okx.symbols','["TEST-USDT-SWAP"]') ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	tick := `{"arg":{"channel":"tickers","instId":"TEST-USDT-SWAP"},"data":[{"last":"1.5","bidPx":"1.4","askPx":"1.6","vol24h":"9","ts":"1761000000000"}]}`
	fund := `{"arg":{"channel":"funding-rate","instId":"TEST-USDT-SWAP"},"data":[{"fundingRate":"0.0001","fundingTime":"1761000000000","nextFundingTime":"1761003600000"}]}`
	srv := fakeOKX(t, []string{tick, fund})
	defer srv.Close()

	c := NewCollector(pool, cfg)
	c.url = "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()
	errc := make(chan error, 8)
	go c.Run(ctx2, errc)

	deadline := time.Now().Add(8 * time.Second)
	for {
		var n int
		_ = pool.QueryRow(ctx, "SELECT count(*) FROM md_ticks WHERE inst_id='TEST-USDT-SWAP'").Scan(&n)
		var fn int
		_ = pool.QueryRow(ctx, "SELECT count(*) FROM md_funding WHERE inst_id='TEST-USDT-SWAP'").Scan(&fn)
		if n > 0 && fn > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("帧未落库 ticks=%d funding=%d（队列消费断）", n, fn)
		}
		time.Sleep(150 * time.Millisecond)
	}
	// 行情巡在岗（fin_employee_registry 实拍）
	var dept string
	if err := pool.QueryRow(ctx,
		"SELECT dept FROM fin_employee_registry WHERE name='行情巡'").Scan(&dept); err != nil || dept != "data" {
		t.Fatalf("行情巡不在岗 dept=%q err=%v", dept, err)
	}
	// rejects 门键预注册在册（五键全注册，零值可见非缺席）
	snap := c.rej.Snapshot()
	for _, k := range []string{"non_usdt_instid", "seq_gap", "parse_fail", "write_err", "symbols_missing"} {
		if _, ok := snap["okxws/"+k]; !ok {
			t.Fatalf("rejects 门键 %s 未注册", k)
		}
	}
	cancel()
	select {
	case err := <-errc:
		t.Fatalf("collector err: %v", err)
	case <-time.After(4 * time.Second):
	}
	_ = json.RawMessage{}
}
