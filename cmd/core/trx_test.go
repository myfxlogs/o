package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func trxTestSetup(t *testing.T) (context.Context, *trxWiring, *http.ServeMux) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	w := &trxWiring{pool: pool, cfg: cfg, st: pgstore.NewStore(pool)}
	mux := http.NewServeMux()
	mountTrxAPI(mux, w)
	return ctx, w, mux
}

func trxGet(t *testing.T, mux *http.ServeMux, path string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec.Code, rec.Body.String()
}

// T-0110 桩测锚：全读面 200+合法 JSON；空表=读件缺显形不造零；
// cex_scan 读件缺 face 核销形（empty 数组+扫描器未迁 note）。
func TestTrxFacesParity(t *testing.T) {
	_, _, mux := trxTestSetup(t)
	for _, p := range []string{"account", "txs", "yield_radar", "yield_curve",
		"hedge_structure", "hedge_flow", "cmc_history", "cex_scan"} {
		code, body := trxGet(t, mux, "/api/trx/"+p)
		if code != 200 {
			t.Fatalf("%s 端点态: %d", p, code)
		}
		var v any
		if err := json.Unmarshal([]byte(body), &v); err != nil {
			t.Fatalf("%s 出数非法 JSON: %v", p, err)
		}
	}
	if _, body := trxGet(t, mux, "/api/trx/account"); !strings.Contains(body, "读件缺") {
		t.Fatalf("空表 account 应读件缺显形: %s", body)
	}
	if _, body := trxGet(t, mux, "/api/trx/cex_scan"); !strings.Contains(body, "扫描器未迁") ||
		!strings.Contains(body, `"spreads":[]`) {
		t.Fatalf("cex_scan 核销形: %s", body)
	}
	// 刷新踢：nil 采集件静默不炸+读件缺显形（非钱径读侧拉新）。
	if code, _ := trxGet(t, mux, "/api/trx/account_refresh"); code != 200 {
		t.Fatal("刷新踢应 200（静默败不阻断渲染）")
	}
}

// T-0110 锚：hedge_flow 双源 UNION 归一（trx TRX 折值/cmc kind 映射+cheat/vote/
// 买卖成交剔除）+计数；bandwidthStatus 阈值锚（trxvote 常量同源）。
func TestHedgeFlowUnionAndBandwidth(t *testing.T) {
	ctx, w, mux := trxTestSetup(t)
	var execID int64
	if err := w.pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES('ETH-USDT',100,'open') RETURNING id`).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd)
		VALUES($1,'ETH-USDT',100)`, execID); err != nil {
		t.Fatal(err)
	}
	for _, tx := range []struct {
		txid, kind string
		amount     int64
		cheat      bool
	}{
		{"tx-a", "rental_income", 5_000_000, false},
		{"tx-b", "vote", 1_000_000, false}, // vote 行剔除
		{"tx-c", "deposit", 9_000_000, false},
		{"tx-d", "rental_income", 7_000_000, true}, // cheat 剔除
	} {
		if _, err := w.pool.Exec(ctx, `INSERT INTO fin_trx_txs(tx_id,ts,kind,amount_sun,cheat_status)
			VALUES($1,now(),$2,$3,$4)`, tx.txid, tx.kind, tx.amount, tx.cheat); err != nil {
			t.Fatal(err)
		}
	}
	for _, hh := range []struct {
		txid, kind string
		amount     float64
	}{
		{"h-1", "OVERNIGHT_SETTLEMENT", 0.5},
		{"h-2", "SELL_TRADE", 99}, // 买卖成交剔除
	} {
		if _, err := w.pool.Exec(ctx, `INSERT INTO fin_cmc_history(account_id,transaction_id,transaction_type,amount,transaction_time)
			VALUES(1,$1,$2,$3,now())`, hh.txid, hh.kind, hh.amount); err != nil {
			t.Fatal(err)
		}
	}
	_, body := trxGet(t, mux, "/api/trx/hedge_flow")
	var resp struct {
		Total int `json:"total"`
		Txs   []struct {
			Source      string  `json:"source"`
			Kind        string  `json:"kind"`
			AmountValue float64 `json:"amount_value"`
			AmountUnit  string  `json:"amount_unit"`
		} `json:"txs"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Total != 3 || len(resp.Txs) != 3 {
		t.Fatalf("UNION 归一应 3 行（vote/cheat/SELL_TRADE 剔除）: total=%d n=%d", resp.Total, len(resp.Txs))
	}
	if resp.Txs[0].Source != "cmc" || resp.Txs[0].Kind != "overnight" || resp.Txs[0].AmountUnit != "USD" {
		t.Fatalf("首行应 cmc overnight USD: %+v", resp.Txs[0])
	}
	found := false
	for _, x := range resp.Txs {
		if x.Source == "trx" && x.Kind == "rental_income" && x.AmountValue == 5 && x.AmountUnit == "TRX" {
			found = true
		}
	}
	if !found {
		t.Fatalf("trx rental 5 TRX 折值行缺: %+v", resp.Txs)
	}
	if bandwidthStatus(600, 0, 0, 0) != "normal" || bandwidthStatus(400, 0, 0, 0) != "yellow" ||
		bandwidthStatus(200, 0, 0, 0) != "red" {
		t.Fatal("带宽状态阈值错（500/250 线）")
	}
}

// T-0110 锚：日终 rollup 幂等（ON CONFLICT(day)）+能量/投票成分口径。
func TestTrxRollupIdempotent(t *testing.T) {
	ctx, w, _ := trxTestSetup(t)
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_trx_txs(tx_id,ts,kind,amount_sun,cheat_status)
		VALUES('r-1',now(),'rental_income',3000000,false)`); err != nil {
		t.Fatal(err)
	}
	// 账户快照两行（reward 正增量=投票成分 2 TRX）。
	for _, rw := range []struct {
		reward int64
		ago    string
	}{{10000000, "8 hours"}, {12000000, "2 hours"}} {
		if _, err := w.pool.Exec(ctx, `INSERT INTO fin_trx_account(address,votes_total,balance_sun,
			allowance_sun,reward_num_sun,energy_delegated,net_limit,net_used,free_net_limit,
			free_net_used,energy_limit,energy_used,bandwidth_staked,tron_power,fetched_at)
			VALUES('T-test',0,0,0,$1,0,0,0,0,0,0,0,0,0, now() - $2::interval)`,
			rw.reward, rw.ago); err != nil {
			t.Fatal(err)
		}
	}
	trxRollup(ctx, w)
	trxRollup(ctx, w) // 幂等——不重复落行
	var n int
	var energy, voting float64
	if err := w.pool.QueryRow(ctx, `SELECT count(*), COALESCE(max(energy_trx),0),
		COALESCE(max(voting_trx),0) FROM position_daily_income WHERE source='auto'`).Scan(&n, &energy, &voting); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("rollup 应幂等 1 行: %d", n)
	}
	if energy != 3 || voting != 2 {
		t.Fatalf("成分口径: energy=%v（应 3）voting=%v（应 2=Δreward 正增量）", energy, voting)
	}
}

// T-0110 锚：引擎旗歇——trx.vote_enabled 缺省/false=构造即让位（零私钥读零链连）；
// 眠置态零触。
func TestTrxvoteFlagOffDormant(t *testing.T) {
	ctx, w, _ := trxTestSetup(t)
	errc := make(chan error, 4)
	done := make(chan struct{})
	go func() { trxvoteArm(ctx, errc, w); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second): // sched-exempt-ok 桩测期界等待（非生产轮询）
		t.Fatal("旗歇态引擎应即刻让位返回")
	}
	// 停驶态零越权：fin_trx_vote_rebalance 零写入（引擎未启无改投行）。
	var n int
	if err := w.pool.QueryRow(ctx,
		`SELECT count(*) FROM fin_trx_vote_rebalance`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("停驶态不得产生引擎写入: %d %v", n, err)
	}
}
