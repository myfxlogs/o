package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// TestPulseState 常驻器官三态判——读败=?/静阈内=动/僵阈内=静/越僵阈=僵。
func TestPulseState(t *testing.T) {
	now := time.Now()
	at := func(d time.Duration) *time.Time { t := now.Add(-d); return &t }
	cases := []struct {
		ts   *time.Time
		err  error
		want string
	}{
		{at(10 * time.Second), nil, "动"},
		{at(2 * time.Minute), nil, "静"},
		{at(10 * time.Minute), nil, "僵"},
		{nil, nil, "?"},
		{nil, errors.New("x"), "?"},
	}
	for i, c := range cases {
		if got := pulseState(c.ts, c.err, time.Minute, 5*time.Minute); got != c.want {
			t.Fatalf("case %d: got %s want %s", i, got, c.want)
		}
	}
}

// TestSpawnlessOrphanGrace 唤起无果在飞宽限（87帧常开实证根修）：
// 新唤<15min未配对=在飞不计；>15min未配对=孤儿照计；配对收口=不计。
func TestSpawnlessOrphanGrace(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE event IN ('唤起','唤起果','唤起败','会话收口') AND ts>now()-interval '2 hours'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	spawn := func(unit string, age time.Duration) {
		// seat=审计机（唤知机×唤起=迁移外直插产线行，测库矩阵只载种子）；
		// 判据不滤 seat——同闸内组合即合法。
		if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note,ts)
			VALUES('审计机','唤起',$1,now()-$2*interval '1 second')`,
			"unit="+unit+" trigger=t", age.Seconds()); err != nil {
			t.Fatalf("spawn %s: %v", unit, err)
		}
	}
	count := func() int {
		n, err := svc.spawnlessOrphans(ctx)
		if err != nil {
			t.Fatalf("spawnlessOrphans: %v", err)
		}
		return n
	}
	spawn("t-fresh", 5*time.Second) // 在飞——宽限内不计
	if got := count(); got != 0 {
		t.Fatalf("在飞唤误计 got %d want 0", got)
	}
	spawn("t-orphan", 20*time.Minute) // 越宽限未配对=孤儿——计
	spawn("t-paired", 20*time.Minute) // 越宽限有配对——不计
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('唤知','会话收口','unit=t-paired 收口')`); err != nil {
		t.Fatalf("close: %v", err)
	}
	if got := count(); got != 1 {
		t.Fatalf("孤儿唤漏计 got %d want 1", got)
	}
}

// TestMoneyDist 资金分布三段面（工单O-30472）——部署=开仓仓名义+dcd
// 策略域凭据tick估值、息=earn%持仓USD估值、闲=NAV残差：三段加总
// ≈NAV（残差式恒等）+部署率=部署/NAV；非USDT币种按 md_ticks 估值、
// 已平仓仓剔出。PoolIsolated——DDL 破坏（DROP）须逐测面独占。
func TestMoneyDist(t *testing.T) {
	ctx := context.Background()
	pool := testutil.PoolIsolated(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	svc := coreService{pool: pool}
	var xid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,decision_id,token_id,inst_id,notional_usd,state)
		VALUES(0,0,gen_random_uuid(),'STUB',100,'open') RETURNING id`).Scan(&xid); err != nil {
		t.Fatalf("exec桩: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd)
		VALUES($1,0,'STUB',100)`, xid); err != nil {
		t.Fatalf("开仓桩: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,state,closed_at)
		VALUES($1,0,'STUB',999,'closed',now())`, xid); err != nil {
		t.Fatalf("平仓桩: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_holding(venue,kind,ccy,product,amount)
		VALUES('okx','earn_flexible','USDT','savings',9000),
		      ('okx','earn_flexible','BTC','savings',0.1)`); err != nil {
		t.Fatalf("息腿桩: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last)
		VALUES(now(),'okx','BTC-USDT',50000)`); err != nil {
		t.Fatalf("tick桩: %v", err)
	}
	// 部署=100(开仓)；息=9000+0.1×50000=14000；闲=20000-100-14000=5900；
	// 部署率=100/20000=0.5%。三段加总=20000=NAV 恒等（残差式）。
	if got, want := svc.moneyDist(ctx, 20000), "部署100(0.5%) 息14000 闲5900"; got != want {
		t.Fatalf("moneyDist=%q want %q", got, want)
	}
}

// TestMoneyDistDegrade 缺段降级——每腿独立读败标 ? 不炸帧：
// 息腿表缺席→息?闲?；部署面视图缺席→部署?；两腿俱败→全 ?。
func TestMoneyDistDegrade(t *testing.T) {
	ctx := context.Background()
	pool := testutil.PoolIsolated(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	svc := coreService{pool: pool}
	var xid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,decision_id,token_id,inst_id,notional_usd,state)
		VALUES(0,0,gen_random_uuid(),'STUB',100,'open') RETURNING id`).Scan(&xid); err != nil {
		t.Fatalf("exec桩: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd)
		VALUES($1,0,'STUB',100)`, xid); err != nil {
		t.Fatalf("仓桩: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_holding(venue,kind,ccy,product,amount)
		VALUES('okx','earn_flexible','USDT','savings',9000)`); err != nil {
		t.Fatalf("息腿桩: %v", err)
	}
	if got, want := svc.moneyDist(ctx, 20000), "部署100(0.5%) 息9000 闲10900"; got != want {
		t.Fatalf("基线 moneyDist=%q want %q", got, want)
	}
	// 案①息腿读败：列改名（public.fin_holding 存在——DROP 会穿透到
	// public 空表产息0假象，列改名让 h.amount 必败=真读败形）。
	if _, err := pool.Exec(ctx, `ALTER TABLE fin_holding RENAME COLUMN amount TO amount_x`); err != nil {
		t.Fatalf("rename amount: %v", err)
	}
	if got, want := svc.moneyDist(ctx, 20000), "部署100 息? 闲?"; got != want {
		t.Fatalf("息腿缺席 moneyDist=%q want %q", got, want)
	}
	// 案②部署面视图缺席：列名还回→DROP fin_position_view（无 public
	// 同名视图=真缺席）→部署腿整句读败→部署?；息腿照实读。
	if _, err := pool.Exec(ctx, `ALTER TABLE fin_holding RENAME COLUMN amount_x TO amount`); err != nil {
		t.Fatalf("restore amount: %v", err)
	}
	if _, err := pool.Exec(ctx, `DROP VIEW fin_position_view`); err != nil {
		t.Fatalf("drop view: %v", err)
	}
	if got, want := svc.moneyDist(ctx, 20000), "部署? 息9000 闲?"; got != want {
		t.Fatalf("部署腿缺席 moneyDist=%q want %q", got, want)
	}
	// 案③两腿俱败：息腿列再改名→全 ?。
	if _, err := pool.Exec(ctx, `ALTER TABLE fin_holding RENAME COLUMN amount TO amount_x`); err != nil {
		t.Fatalf("rename amount2: %v", err)
	}
	if got, want := svc.moneyDist(ctx, 20000), "部署? 息? 闲?"; got != want {
		t.Fatalf("双腿缺席 moneyDist=%q want %q", got, want)
	}
}

// TestSelfAwareFrameDist 帧面锚（工单O-30472）——分布面真进
// self_aware 量表帧：赚段尾挂 |钱在:三段，既有判词形不动。
func TestSelfAwareFrameDist(t *testing.T) {
	ctx := context.Background()
	pool := testutil.PoolIsolated(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	svc := coreService{pool: pool}
	if _, err := pool.Exec(ctx, `INSERT INTO md_equity_curve(ts,venue,ccy,equity,src)
		VALUES(now(),'ledger','NAV',20000,'t-dist')`); err != nil {
		t.Fatalf("nav桩: %v", err)
	}
	var xid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,decision_id,token_id,inst_id,notional_usd,state)
		VALUES(0,0,gen_random_uuid(),'STUB',100,'open') RETURNING id`).Scan(&xid); err != nil {
		t.Fatalf("exec桩: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd)
		VALUES($1,0,'STUB',100)`, xid); err != nil {
		t.Fatalf("仓桩: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_holding(venue,kind,ccy,product,amount)
		VALUES('okx','earn_flexible','USDT','savings',9000)`); err != nil {
		t.Fatalf("息腿桩: %v", err)
	}
	svc.selfAwareBeat(ctx)
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='量表' AND ticket='self_aware' ORDER BY id DESC LIMIT 1`).Scan(&note); err != nil {
		t.Fatalf("帧未落: %v", err)
	}
	for _, want := range []string{"钱在:部署100(0.5%)", "息9000", "闲10900"} {
		if !strings.Contains(note, want) {
			t.Fatalf("帧缺分布段 %q——note=%.160q", want, note)
		}
	}
}

// TestTruncRunes 摘要截断——多字节不截半字。
func TestTruncRunes(t *testing.T) {
	if got := truncRunes("工单#1 认领:开修——修码", 6); got != "工单#1 认" {
		t.Fatalf("截断 %q", got)
	}
	if got := truncRunes("短", 9); got != "短" {
		t.Fatalf("短串 %q", got)
	}
}
