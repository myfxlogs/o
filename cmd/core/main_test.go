package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/gen/dashboardv1"
	"o/internal/guard"
	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
	"o/internal/trading/okxexec"
)

// T-0043 锚①：Warden 平仓径双腿必置 Close=true（漏置=按开仓腿装配→
// short 仓上 buy 反向开 long 非平仓——阻断级雷）。
func TestCloseLegOfSetsClose(t *testing.T) {
	spec := guard.CloseSpec{NotionalUSD: 100, Legs: [2]guard.CloseLeg{
		{Venue: "okx", InstID: "BTC-USDT-SWAP", Side: "buy", Sz: 1},
		{Venue: "okx", InstID: "BTC-USDT", Side: "sell", Sz: 0.0012},
	}}
	for i, l := range spec.Legs {
		got := closeLegOf(spec, l)
		if !got.Close {
			t.Fatalf("leg%d Close 漏置（反向开仓雷）: %+v", i+1, got)
		}
		if got.Venue != l.Venue || got.InstID != l.InstID || got.Side != l.Side || got.Notional != 50 {
			t.Fatalf("leg%d 装配串味: %+v", i+1, got)
		}
	}
}

// T-0043 欠件锚（T-0046 补）：closeIntent Kind 必为 "close"（live_switch 翻 false
// 时平仓被拦=安全律反向——Kind 错=平仓被开仓闸吞）。
func TestCloseIntentKindIsClose(t *testing.T) {
	in := closeIntent(guard.CloseSpec{DecisionID: 1, StrategyRef: 1, InstID: "BTC-USDT-SWAP", NotionalUSD: 100,
		Legs: [2]guard.CloseLeg{{Venue: "okx", InstID: "BTC-USDT-SWAP", Side: "buy", Sz: 1},
			{Venue: "okx", InstID: "BTC-USDT", Side: "sell", Sz: 0.001}}})
	if in.Kind != "close" {
		t.Fatalf("close Intent Kind=%q（应 close——免闸免赎语义位）", in.Kind)
	}
}

// openEthLeg 开仓 ETH 腿 fixture（fund 集={okx:USDT,ETH}——在持腿基币入口径）。
func openEthLeg(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	var xid, pid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES('ETH-USDT',100,'open') RETURNING id`).Scan(&xid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd)
		VALUES($1,'ETH-USDT',100) RETURNING id`, xid).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state)
		VALUES($1,1,'cl-ovw','okx','buy','ETH-USDT',1,'filled')`, xid); err != nil {
		t.Fatal(err)
	}
}

// T-0095 v2+T-0096 锚：NAV=基金的钱口径合计（USDT∪open 腿基币+MT5 全计）
// ——账外尘（AVAX/COMP/DOGE）出 NAV 不出卡不返行（零遗留概念）；腿平=基币
// 跌出口径；MT5 户行=净值现读随 NAV。
func TestOverviewFundScopeVenueCard(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM md_account_events WHERE channel IN ('account','earn_parking')"); err != nil {
		t.Fatal(err)
	}
	openEthLeg(t, ctx, pool)
	// 交易户帧：USDT 2100+ETH 腿 23.6 入口径；AVAX 尘 3=托管尘出局。
	if _, err := pool.Exec(ctx, `INSERT INTO md_account_events(ts,channel,ccy,data)
		VALUES(now(),'account','USDT',$1)`,
		[]byte(`{"details":[{"ccy":"USDT","eqUsd":"2100"},{"ccy":"ETH","eqUsd":"23.6"},{"ccy":"AVAX","eqUsd":"3"}]}`)); err != nil {
		t.Fatal(err)
	}
	// T-0100 ②：DOGE 残币停泊凭据（residual_park decision 行→Earn 域回集）；
	// T-0100 ①：Earn ETH 0.0013 利息尘=腿基币渗漏托管不显（Earn 域只有
	// USDT+停泊溯源币）。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen)
		VALUES('{"kind":"transfer","op":"residual_park","venue":"okx","ccy":"DOGE","amt":295}','allow','transfer',1)`); err != nil {
		t.Fatal(err)
	}
	svc := coreService{pool: pool,
		fundingLive: func(context.Context) (map[string]float64, error) {
			return map[string]float64{"USDT": 0.2, "COMP": 0.05}, nil
		},
		earnLive: func(context.Context) (map[string]float64, error) {
			return map[string]float64{"USDT": 2057.37, "DOGE": 295.0, "ETH": 0.0013}, nil
		},
		mt5Live: func(context.Context) ([]mt5Acct, error) {
			return []mt5Acct{{Login: "3001157", Equity: 3247.41, Balance: 3200}}, nil
		}}
	o, err := svc.Overview(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	// NAV=2100+23.6+0.2+2057.37+3247.41（托管尘零计入）
	if o.NavTotal < 7428.57 || o.NavTotal > 7428.59 {
		t.Fatalf("NAV 基金口径合计: %v", o.NavTotal)
	}
	sec := map[string][]*dashboardv1.MoneyRow{}
	for _, m := range o.Money {
		if m.Legacy {
			t.Fatalf("零遗留概念——legacy 位恒 false: %+v", m)
		}
		sec[m.Venue+"/"+m.Section] = append(sec[m.Venue+"/"+m.Section], m)
	}
	if len(sec["okx/交易户"]) != 2 || len(sec["okx/资金户"]) != 1 || len(sec["okx/Earn"]) != 2 {
		t.Fatalf("三区行数（Earn=USDT+停泊 DOGE 回集，ETH 渗漏尘不显）: %+v", o.Money)
	}
	var earnDog bool
	for _, m := range sec["okx/Earn"] {
		if m.Ccy == "DOGE" {
			earnDog = true
		}
		if m.Ccy == "ETH" {
			t.Fatalf("Earn 域腿基币渗漏尘不应返行: %+v", m)
		}
	}
	if !earnDog {
		t.Fatalf("停泊溯源币未回集: %+v", sec["okx/Earn"])
	}
	if len(sec["mt5/户"]) != 1 || sec["mt5/户"][0].Label != "户 3001157" ||
		sec["mt5/户"][0].Usd != 3247.41 {
		t.Fatalf("MT5 户行: %+v", sec["mt5/户"])
	}
	for _, m := range o.Money {
		if m.Ccy == "AVAX" || m.Ccy == "COMP" {
			t.Fatalf("账外尘不应返行: %+v", m)
		}
	}
	// 腿平=基币跌出口径（ETH 行消失，NAV 同步 -23.6）。
	if _, err := pool.Exec(ctx, `UPDATE fin_position SET closed_at=now()`); err != nil {
		t.Fatal(err)
	}
	o2, err := svc.Overview(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if o2.NavTotal < 7404.97 || o2.NavTotal > 7404.99 {
		t.Fatalf("腿平后 NAV（ETH 跌出）: %v", o2.NavTotal)
	}
	for _, m := range o2.Money {
		if m.Ccy == "ETH" {
			t.Fatalf("腿平残币=托管尘不应返行: %+v", m)
		}
	}
}

// T-0086 ①+T-0096 锚：缺户=0 不误报；MT5 读件缺/读败=venue 区显「读件缺」
// 行不藏（禁假零禁假绿）。
func TestOverviewMissingAccountsZero(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM md_account_events WHERE channel IN ('account','earn_parking')"); err != nil {
		t.Fatal(err)
	}
	svc := coreService{pool: pool}
	o, err := svc.Overview(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if o.NavTotal != 0 {
		t.Fatalf("缺户 NAV 应 0: %v", o.NavTotal)
	}
	if len(o.Money) != 2 || o.Money[0].Venue != "mt5" || o.Money[0].Note != "读件缺" ||
		o.Money[1].Section != "结构性持仓" || o.Money[1].Note != "读件缺" {
		t.Fatalf("MT5/dcd 读件缺行不藏: %+v", o.Money)
	}
	// 读败同径（err 上抛→读件缺行）。
	svc2 := coreService{pool: pool,
		mt5Live: func(context.Context) ([]mt5Acct, error) {
			return nil, fmt.Errorf("mt5 connect refused")
		}}
	o2, err := svc2.Overview(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(o2.Money) != 2 || o2.Money[0].Note != "读件缺" {
		t.Fatalf("MT5 读败应显读件缺: %+v", o2.Money)
	}
}

// dcdFixtureOrder 业主实爆单形（T-0105 病灶锚——BTC-USDT-261225-95000-C
// #11448262 名义 0.0207 BTC）。
func dcdFixtureOrder() okxexec.DcdOrder {
	return okxexec.DcdOrder{OrdID: "11448262", ProductID: "BTC-USDT-261225-95000-C",
		State: "live", NotionalCcy: "BTC", NotionalSz: "0.0207", Strike: "95000",
		AnnualizedYield: "0.0857",
		SettleTime:      fmt.Sprint(time.Now().Add(60 * 24 * time.Hour).UnixMilli()),
		CTime:           fmt.Sprint(time.Now().Add(-24 * time.Hour).UnixMilli())}
}

func doingTexts(o *dashboardv1.OverviewResponse) string {
	var b strings.Builder
	for _, d := range o.Doing {
		b.WriteString(d.Text + "\n")
	}
	return b.String()
}

// T-0105 锚：在干列表全集化——①无仓有 DCD→DCD 行无静持（展示引用不重复
// 入 NAV）②双有→两行 ③全空→静持 ④DCD 读件缺→显形非静默非假静持。
func TestOverviewDoingFullCoverage(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM md_account_events WHERE channel IN ('account','earn_parking')"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,inst_id,last) VALUES(now(),'BTC-USDT',80000)`); err != nil {
		t.Fatal(err)
	}
	ord := dcdFixtureOrder()
	newSvc := func(dcds []okxexec.DcdOrder, err error) coreService {
		return coreService{pool: pool, dcdLive: func(context.Context) ([]okxexec.DcdOrder, error) {
			return dcds, err
		}}
	}
	// ①无仓有 DCD。
	o1, err := newSvc([]okxexec.DcdOrder{ord}, nil).Overview(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	tx1 := doingTexts(o1)
	if !strings.Contains(tx1, "DCD BTC-USDT-261225-95000-C") ||
		!strings.Contains(tx1, "#11448262") || !strings.Contains(tx1, "持仓中") {
		t.Fatalf("DCD 行应入列: %q", tx1)
	}
	if strings.Contains(tx1, "静持") {
		t.Fatalf("有 DCD 不得显静持: %q", tx1)
	}
	// NAV 单计锚：钱卡 dcdRows 计 0.0207×80000=1656——在干展示引用不再加。
	if o1.NavTotal < 1655.99 || o1.NavTotal > 1656.01 {
		t.Fatalf("NAV 应单计 1656: %v", o1.NavTotal)
	}
	// ③全空→静持（恰一行——前端空态契约不变）。
	o3, err := newSvc(nil, nil).Overview(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(o3.Doing) != 1 || !strings.HasPrefix(o3.Doing[0].Text, "静持") {
		t.Fatalf("真空应恰静持一行: %+v", o3.Doing)
	}
	// ④DCD 读件缺→显形行、无静持（不假零不误报）。
	o4, err := newSvc(nil, fmt.Errorf("dcd 429")).Overview(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	tx4 := doingTexts(o4)
	if !strings.Contains(tx4, "读件缺") || strings.Contains(tx4, "静持") {
		t.Fatalf("读件缺应显形且不静持: %q", tx4)
	}
	// ②双有→对冲仓+DCD 两行。
	openEthLeg(t, ctx, pool)
	o2, err := newSvc([]okxexec.DcdOrder{ord}, nil).Overview(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	tx2 := doingTexts(o2)
	if !strings.Contains(tx2, "ETH 对冲仓") || !strings.Contains(tx2, "DCD BTC-USDT") {
		t.Fatalf("双有应两源入列: %q", tx2)
	}
	if strings.Contains(tx2, "静持") {
		t.Fatalf("双有不得显静持: %q", tx2)
	}
}

// T-0107 锚：再开冷却窗——guard 平仓后同策略同标的窗内拒再开；窗外放行；
// 键缺 fail-closed（宁可冻结不再开也不裸奔连绞）。
func TestReopenCooldownBlocked(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var stratID, execID, posID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue) VALUES('BTC-USDT-SWAP','guard-test','live',25,'okx') RETURNING id`).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state) VALUES($1,'BTC-USDT-SWAP',25,'open') RETURNING id`, stratID).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd)
		VALUES($1,$2,'BTC-USDT-SWAP',25) RETURNING id`, execID, stratID).Scan(&posID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO fin_guard_close(position_id,state,trigger_kind,updated_at)
		VALUES($1,'closed','delta_drift',now())`, posID); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	// 窗内（updated_at=now < 300s）→blocked。
	blocked, err := reopenCooldownBlocked(ctx, cfg, pool, stratID, "BTC-USDT-SWAP")
	if err != nil || !blocked {
		t.Fatalf("冷却窗内应拦: blocked=%v err=%v", blocked, err)
	}
	// 窗外（updated_at=301s 前）→放行。
	if _, err := pool.Exec(ctx,
		`UPDATE fin_guard_close SET updated_at=now()-interval '301 seconds' WHERE position_id=$1`, posID); err != nil {
		t.Fatal(err)
	}
	if blocked, err = reopenCooldownBlocked(ctx, cfg, pool, stratID, "BTC-USDT-SWAP"); err != nil || blocked {
		t.Fatalf("窗外应放行: blocked=%v err=%v", blocked, err)
	}
	// 键缺=fail-closed 拦截。
	if _, err := pool.Exec(ctx, `DELETE FROM app_config WHERE key='exec.reopen_cooldown_sec'`); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = reopenCooldownBlocked(ctx, cfg, pool, stratID, "BTC-USDT-SWAP"); err == nil {
		t.Fatal("键缺应 fail-closed 上抛")
	}
}
