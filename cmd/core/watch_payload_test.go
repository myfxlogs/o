// watch_payload_test.go —— T-0220a 载荷路由对拍：带载荷面 kind=查表出
// structpb（反读 AsMap 与源读面同值）；无载荷面 kind=nil 纯信号；链上流水
// 面经 trx wiring 同源。
package main

import (
	"context"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func payloadSetup(t *testing.T) coreService {
	t.Helper()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatal(err)
	}
	return coreService{pool: pool, trx: &trxWiring{pool: pool, st: pgstore.NewStore(pool)}}
}

// 锚②：ev_position→ListClosedPositions 最新 5 行载荷——AsMap 反读 total/行数
// 与源读面同值（protojson 往返一致）。
func TestWatchPayloadPosition(t *testing.T) {
	c := payloadSetup(t)
	ctx := context.Background()
	var execID int64
	if err := c.pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES ('BTC-USDT-SWAP',100,'closed') RETURNING id`).Scan(&execID); err != nil {
		t.Skipf("fin_execution 形不符跳过: %v", err)
	}
	_, err := c.pool.Exec(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd,opened_at,closed_at,settled_amount,settled_ccy)
		VALUES ($1,'BTC-USDT-SWAP',100,now()-interval '9 days',now()-interval '2 days',120,'USDT'),
		       ($1,'ETH-USDT-SWAP',100,now()-interval '8 days',now()-interval '4 hours',7,'USD'),
		       ($1,'OKB-USDT',100,now()-interval '7 days',now()-interval '3 hours',1,'USDT'),
		       ($1,'XRP-USDT',100,now()-interval '6 days',now()-interval '2 hours',2,'USDT'),
		       ($1,'DOT-USDT',100,now()-interval '5 days',now()-interval '1 hours',3,'USDT'),
		       ($1,'LTC-USDT',100,now()-interval '4 days',now(),4,'USDT')`, execID)
	if err != nil {
		t.Skipf("fin_position 形不符跳过: %v", err)
	}
	st, err := c.watchPayload(ctx, "ev_position")
	if err != nil {
		t.Fatal(err)
	}
	if st == nil {
		t.Fatal("ev_position 应带载荷")
	}
	m := st.AsMap()
	total, ok := m["total"].(float64)
	if !ok || total != 6 {
		t.Fatalf("total 反读应 6: %v", m["total"])
	}
	pos, ok := m["positions"].([]interface{})
	if !ok || len(pos) != 5 {
		t.Fatalf("载荷应最新 5 行: %v", m["positions"])
	}
	first := pos[0].(map[string]interface{})
	if first["instId"] != "LTC-USDT" {
		t.Fatalf("应闭仓倒序最新一行: %v", first["instId"])
	}
}

// 锚③：ev_hostmetrics 全量载荷（零行=空集载荷非 nil）；watch_open/ev_account
// =纯信号帧 nil。
func TestWatchPayloadSignalKinds(t *testing.T) {
	c := payloadSetup(t)
	ctx := context.Background()
	st, err := c.watchPayload(ctx, "ev_hostmetrics")
	if err != nil || st == nil {
		t.Fatalf("ev_hostmetrics 应带载荷: %v %v", st, err)
	}
	if hosts, ok := st.AsMap()["hosts"].([]interface{}); !ok || len(hosts) != 0 {
		t.Fatalf("零行应显式空集载荷: %v", st.AsMap())
	}
	for _, kind := range []string{"watch_open", "ev_account", "ev_strategy"} {
		st, err := c.watchPayload(ctx, kind)
		if err != nil || st != nil {
			t.Fatalf("%s 应纯信号帧: %v %v", kind, st, err)
		}
	}
}

// 锚①：ev_cashflow→ListHedgeFlow 载荷（trx/cmc 链上面同源——空链=空集载荷）。
func TestWatchPayloadHedgeFlow(t *testing.T) {
	c := payloadSetup(t)
	st, err := c.watchPayload(context.Background(), "ev_cashflow")
	if err != nil {
		t.Fatal(err)
	}
	if st == nil {
		t.Fatal("ev_cashflow 应带载荷")
	}
	if _, ok := st.AsMap()["total"].(float64); !ok {
		t.Fatalf("载荷应 ListHedgeFlowResponse 形: %v", st.AsMap())
	}
}
