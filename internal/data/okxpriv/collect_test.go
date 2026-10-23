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
		if !loginOK {
			_ = conn.WriteMessage(websocket.TextMessage,
				[]byte(`{"event":"login","code":"60011","msg":"bad key"}`))
			return
		}
		if _, _, err := conn.ReadMessage(); err != nil { // subscribe
			return
		}
		for _, f := range frames {
			_ = conn.WriteMessage(websocket.TextMessage, []byte(f))
		}
		time.Sleep(2 * time.Second)
	}))
}

func TestPrivE2E(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	evt := `{"arg":{"channel":"balance_and_position"},"data":[{"ccy":"USDT","eqUsd":"21.6","availEq":"20"}]}`
	srv := fakePrivWS(t, true, []string{evt})
	defer srv.Close()

	c := &Collector{pool: pool, url: "ws" + strings.TrimPrefix(srv.URL, "http"),
		lastEq: map[string]float64{}}
	c.rej = newTestRej()
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()
	go c.dialLoop(ctx2)
	time.Sleep(1500 * time.Millisecond)

	var n int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM md_account_events WHERE channel='balance_and_position'").Scan(&n); err != nil || n == 0 {
		t.Fatalf("事件未落事实行 n=%d err=%v", n, err)
	}
	var eq float64
	if err := pool.QueryRow(ctx,
		"SELECT equity FROM fin_venue_account_snapshot WHERE venue='okx' AND ccy='USDT'").Scan(&eq); err != nil || eq != 21.6 {
		t.Fatalf("快照行未刷 eq=%v err=%v", eq, err)
	}
}
