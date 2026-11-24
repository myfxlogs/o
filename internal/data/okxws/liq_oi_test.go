package okxws

// liq_oi_test.go —— O-35587② 桩锚：liquidation-orders 解析（docs-v5 实证
// 载荷形）+md_liquidation 天然键去重幂等（DB 桩 O_TEST_PG_DSN 缺=skip）。
import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// docs-v5 实证推送例（IOST-USDT-SWAP liquidation-orders）。
const liqFrameJSON = `{
  "arg": {"channel": "liquidation-orders", "instType": "SWAP"},
  "data": [{
    "details": [
      {"bkLoss": "0", "bkPx": "0.007831", "ccy": "", "posSide": "short",
       "side": "buy", "sz": "13", "ts": "1692266434010"},
      {"bkLoss": "0", "bkPx": "0.007800", "ccy": "USDT", "posSide": "long",
       "side": "sell", "sz": "5", "ts": "1692266435000"}
    ],
    "instFamily": "IOST-USDT", "instId": "IOST-USDT-SWAP",
    "instType": "SWAP", "uly": "IOST-USDT"
  }]
}`

func TestParseLiquidationDocsShape(t *testing.T) {
	var msg wsMsg
	if err := json.Unmarshal([]byte(liqFrameJSON), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Arg.Channel != "liquidation-orders" || msg.Arg.InstID != "" {
		t.Fatalf("频道级订阅 arg 无 instId: %+v", msg.Arg)
	}
	var d map[string]json.RawMessage
	if err := json.Unmarshal(msg.Data[0], &d); err != nil {
		t.Fatal(err)
	}
	liqs := parseLiquidation(d)
	if len(liqs) != 2 {
		t.Fatalf("details 两条应展开两行, 实得 %d", len(liqs))
	}
	r := liqs[0]
	if r.instID != "IOST-USDT-SWAP" || r.instFam != "IOST-USDT" || r.uly != "IOST-USDT" {
		t.Fatalf("帧级字段随行败: %+v", r)
	}
	if r.posSide != "short" || r.side != "buy" || r.sz != 13 || r.bkPx != 0.007831 {
		t.Fatalf("detail 字段解析败: %+v", r)
	}
	want := time.UnixMilli(1692266434010).UTC()
	if !r.ts.Equal(want) {
		t.Fatalf("ts 毫秒 epoch 解析败: %v want %v", r.ts, want)
	}
	// 空 details/缺 instId=拒入
	if got := parseLiquidation(map[string]json.RawMessage{}); got != nil {
		t.Fatal("空载荷应拒入")
	}
}

func TestInsertBatchLiquidationDedup(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatalf("migrate md: %v", err)
	}
	c := &Collector{pool: pool}
	ts := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	mk := func() rowItem {
		return rowItem{liqs: []liqRow{{instID: "TEST-USDT-SWAP", instFam: "TEST-USDT",
			uly: "TEST-USDT", ts: ts, posSide: "long", side: "sell", sz: 1.5,
			bkPx: 100.5, bkLoss: 0, ccy: "USDT"}}}
	}
	if err := c.insertBatch(ctx, []rowItem{mk()}); err != nil {
		t.Fatalf("首插败: %v", err)
	}
	// 同天然键重推（重连/重试形）——DO NOTHING 不重。
	if err := c.insertBatch(ctx, []rowItem{mk()}); err != nil {
		t.Fatalf("重推应幂等: %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM md_liquidation
		WHERE venue='okx' AND inst_id='TEST-USDT-SWAP' AND ts=$1`, ts).Scan(&n); err != nil {
		t.Fatalf("计数败: %v", err)
	}
	if n != 1 {
		t.Fatalf("天然键去重败——行数=%d want 1", n)
	}
	// 清场（append-only 账面测试行——测试库隔离域内删行合规）。
	if _, err := pool.Exec(ctx, `DELETE FROM md_liquidation WHERE inst_id='TEST-USDT-SWAP'`); err != nil {
		t.Fatalf("清场败: %v", err)
	}
}

func TestOIPollerUpsert(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatalf("migrate md: %v", err)
	}
	get := func(ctx context.Context, path string) (json.RawMessage, error) {
		if path != "/api/v5/public/open-interest?instType=SWAP" {
			t.Fatalf("OI 端点形错: %s", path)
		}
		// PublicGet 返文=data 数组本体（信封已剥——工单#36961：信封形桩
		// 曾掩盖解析败，桩须同真形）。
		return json.RawMessage(`[
			{"instId":"TEST-USDT-SWAP","instType":"SWAP","oi":"12345.6","oiCcy":"98765.4"},
			{"instId":"TEST2-USDT-SWAP","instType":"SWAP","oi":"1","oiCcy":"2"}
		]`), nil
	}
	p := NewOIPoller(pool, get)
	p.poll(ctx)
	var oi, oiccy float64
	if err := pool.QueryRow(ctx, `SELECT oi, oi_ccy FROM md_open_interest
		WHERE venue='okx' AND inst_id='TEST-USDT-SWAP'`).Scan(&oi, &oiccy); err != nil {
		t.Fatalf("OI 行缺: %v", err)
	}
	if oi != 12345.6 || oiccy != 98765.4 {
		t.Fatalf("OI 值解析败: oi=%v oi_ccy=%v", oi, oiccy)
	}
	// 二拍同键 upsert 最新态（state 表律）
	get2 := func(ctx context.Context, path string) (json.RawMessage, error) {
		return json.RawMessage(`[{"instId":"TEST-USDT-SWAP","oi":"999","oiCcy":"888"}]`), nil
	}
	p2 := NewOIPoller(pool, get2)
	p2.poll(ctx)
	if err := pool.QueryRow(ctx, `SELECT oi FROM md_open_interest
		WHERE venue='okx' AND inst_id='TEST-USDT-SWAP'`).Scan(&oi); err != nil {
		t.Fatal(err)
	}
	if oi != 999 {
		t.Fatalf("upsert 最新态败: oi=%v want 999", oi)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM md_open_interest WHERE inst_id LIKE 'TEST%'`); err != nil {
		t.Fatalf("清场败: %v", err)
	}
}
