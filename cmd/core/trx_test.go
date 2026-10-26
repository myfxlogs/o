package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	connect "connectrpc.com/connect/v2"
	connecthttp "connectrpc.com/connect/v2/connecthttp"

	"o/internal/gen/dashboardv1"
	"o/internal/gen/dashboardv1/dashboardv1connect"
	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func trxTestSetup(t *testing.T) (context.Context, *trxWiring, coreService) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	w := &trxWiring{pool: pool, cfg: cfg, st: pgstore.NewStore(pool)}
	return ctx, w, coreService{pool: pool, trx: w}
}

// T-0156 锚：REST 手拼面全撤 404 化——mountUI+mountCore 同生产装配，
// /api/overview、/api/watch、/api/trx/* 一律 404（数据面唯一入口=Connect）。
func TestRestFacesRetired(t *testing.T) {
	_, w, _ := trxTestSetup(t)
	mux := http.NewServeMux()
	mountUI(mux)
	mountCore(mux, w.pool, nil, nil, nil, nil, w)
	for _, p := range []string{"/api/overview", "/api/watch", "/api/trx/account",
		"/api/trx/yield_radar", "/api/trx/hedge_flow", "/api/trx/account_refresh"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", p, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s 应 404 化（REST 面已撤）: %d", p, rec.Code)
		}
	}
}

// T-0156 桩测锚：全 proto 读面空表可答——读件缺显形不造零（found=false），
// 刷新踢 nil 采集件静默不炸；收益面冷启动零样本透传。
func TestTrxFacesParity(t *testing.T) {
	ctx, _, svc := trxTestSetup(t)
	if r, err := svc.GetTrxAccount(ctx, &dashboardv1.GetTrxAccountRequest{}); err != nil || r.Found {
		t.Fatalf("空表 account 应 found=false 读件缺显形: %+v %v", r, err)
	}
	if r, err := svc.TrxAccountRefresh(ctx, &dashboardv1.TrxAccountRefreshRequest{}); err != nil || r.Found {
		t.Fatalf("刷新踢 nil 采集件应静默+found=false: %+v %v", r, err)
	}
	if r, err := svc.ListTrxTxs(ctx, &dashboardv1.ListTrxTxsRequest{}); err != nil || len(r.Txs) != 0 {
		t.Fatalf("空表 txs 应空集: %+v %v", r, err)
	}
	if r, err := svc.GetTrxYieldRadar(ctx, &dashboardv1.GetTrxYieldRadarRequest{}); err != nil ||
		r.SampleDays_7D != 0 || r.SampleDays_30D != 0 {
		t.Fatalf("冷启动雷达应零样本透传: %+v %v", r, err)
	}
	if r, err := svc.GetTrxYieldCurve(ctx, &dashboardv1.GetTrxYieldCurveRequest{}); err != nil ||
		len(r.Points) != 0 {
		t.Fatalf("冷启动曲线应空集不猜: %+v %v", r, err)
	}
	if r, err := svc.GetHedgeStructure(ctx, &dashboardv1.GetHedgeStructureRequest{}); err != nil ||
		r.Structure == nil || r.Structure.LegSpot == nil || r.Structure.LegShort == nil {
		t.Fatalf("冷启动结构应双腿空壳不炸: %+v %v", r, err)
	}
}

// T-0110/T-0156 锚：hedge_flow 双源 UNION 归一（trx TRX 折值/cmc kind 映射+
// cheat/vote/买卖成交剔除）+计数；bandwidthStatus 阈值锚（trxvote 常量同源）。
func TestHedgeFlowUnionAndBandwidth(t *testing.T) {
	ctx, w, svc := trxTestSetup(t)
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
	resp, err := svc.ListHedgeFlow(ctx, &dashboardv1.ListHedgeFlowRequest{})
	if err != nil {
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

// T-0156 锚：折值与状态字段真值透传（账户快照→proto 折值后端算定）。
func TestTrxAccountRowMapping(t *testing.T) {
	ctx, w, svc := trxTestSetup(t)
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_trx_account(address,votes_total,balance_sun,
		allowance_sun,reward_num_sun,energy_delegated,net_limit,net_used,free_net_limit,
		free_net_used,energy_limit,energy_used,bandwidth_staked,tron_power,sr_address,fetched_at)
		VALUES('T-test',3,5000000,0,7000000,2000000,600,0,400,0,0,0,1000000,3000000,'SR-x',now())`); err != nil {
		t.Fatal(err)
	}
	r, err := svc.GetTrxAccount(ctx, &dashboardv1.GetTrxAccountRequest{})
	if err != nil || !r.Found {
		t.Fatalf("有快照应 found=true: %+v %v", r, err)
	}
	if r.TrxBalance != 5 || r.TronPower != 3 || r.TrxAllowance != 7 ||
		r.EnergyDelegated != 2 || r.Votes != 3 || r.SrAddress != "SR-x" {
		t.Fatalf("折值口径错（sun→TRX=1e6）: %+v", r)
	}
	if r.BandwidthStatus != "normal" { // 剩余 600+400=1000 ≥ 500 线
		t.Fatalf("带宽状态应 normal: %s", r.BandwidthStatus)
	}
}

// T-0110 锚：日终 rollup 幂等（ON CONFLICT(day)）+能量/投票成分口径。
// D-0026：桩时戳钉当日 UTC 日内（day+2h/+8h，与 trxRollup/ListTRXAccountByDay
// 谓词同域）——固定位 now()-N h 在 UTC 00:00–08:00 漂过日界被当日谓词滤掉。
func TestTrxRollupIdempotent(t *testing.T) {
	ctx, w, _ := trxTestSetup(t)
	day := time.Now().UTC().Truncate(24 * time.Hour)
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_trx_txs(tx_id,ts,kind,amount_sun,cheat_status)
		VALUES('r-1',$1,'rental_income',3000000,false)`, day.Add(8*time.Hour)); err != nil {
		t.Fatal(err)
	}
	// 账户快照两行（reward 正增量=投票成分 2 TRX）。
	for _, rw := range []struct {
		reward int64
		at     time.Time
	}{{10000000, day.Add(2 * time.Hour)}, {12000000, day.Add(8 * time.Hour)}} {
		if _, err := w.pool.Exec(ctx, `INSERT INTO fin_trx_account(address,votes_total,balance_sun,
			allowance_sun,reward_num_sun,energy_delegated,net_limit,net_used,free_net_limit,
			free_net_used,energy_limit,energy_used,bandwidth_staked,tron_power,fetched_at)
			VALUES('T-test',0,0,0,$1,0,0,0,0,0,0,0,0,0,$2)`,
			rw.reward, rw.at); err != nil {
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

// T-0156 验收桩锚：Connect 全链路——mountCore 同生产装配+connect-go 客户端
// 过线打读面（REST 撤后唯一入口），+Watch 帧 NOTIFY→帧达 <1s（事件驱动律：
// fin_cashflow 行落→ev_cashflow 触发器→LISTEN 流帧，所见即所得刷新源）。
func TestConnectEndToEndAndWatchLatency(t *testing.T) {
	ctx, w, _ := trxTestSetup(t)
	mux := http.NewServeMux()
	mountUI(mux)
	mountCore(mux, w.pool, nil, nil, nil, nil, w)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cl := dashboardv1connect.NewCoreServiceClient(connect.NewClient(
		connecthttp.NewTransport(srv.Client(), srv.URL)))
	if r, err := cl.GetTrxAccount(ctx, &dashboardv1.GetTrxAccountRequest{}); err != nil || r.Found {
		t.Fatalf("过线 GetTrxAccount 应空表 found=false: %+v %v", r, err)
	}
	if r, err := cl.GetTrxYieldRadar(ctx, &dashboardv1.GetTrxYieldRadarRequest{}); err != nil {
		t.Fatalf("过线 GetTrxYieldRadar: %v", err)
	} else if r.SampleDays_7D != 0 {
		t.Fatalf("冷启动雷达应零样本: %+v", r)
	}
	// Watch 帧 <1s 桩锚：watch_open 首帧=就绪信号（LISTEN 注册在先——无
	// 就绪竞速），随后拍行 ev_cashflow 帧达测距 <1s（NOTIFY 同事务提交即醒，
	// 实测 ms 级）。
	wctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	stream, err := cl.Watch(wctx, &dashboardv1.WatchRequest{})
	if err != nil {
		t.Fatal(err)
	}
	open, err := stream.Receive()
	if err != nil {
		t.Fatalf("就绪首帧未达: %v", err)
	}
	if open.Kind != "watch_open" {
		t.Fatalf("首帧应 watch_open: %s", open.Kind)
	}
	t0 := time.Now()
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_cashflow
		(kind,venue,ccy,amount,source_ref,source_table,occurred_at)
		VALUES('fee','okx','USDT',-0.1,1,'fin_execution',now())`); err != nil {
		t.Fatal(err)
	}
	ev, err := stream.Receive()
	if err != nil {
		t.Fatalf("流帧未达: %v", err)
	}
	if el := time.Since(t0); el > time.Second {
		t.Fatalf("NOTIFY→帧达应 <1s（事件驱动律）: %v", el)
	}
	if k := ev.Kind; k != "ev_cashflow" {
		t.Fatalf("帧通道应 ev_cashflow: %s", k)
	}
	cancel()
}
