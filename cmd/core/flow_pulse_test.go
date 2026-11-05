// flow_pulse_test.go —— 活水段桩（T-0307：七段+心跳行+fail-soft 显缺不炸）。
package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"o/internal/gen/dashboardv1"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// 锚：七段返回+ok 态+roster 字段在列（段查询败=该段 ok=false 不炸面）。
func TestFlowPulseStages(t *testing.T) {
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(context.Background(), pool, "md"); err != nil {
		t.Fatal(err)
	}
	c := coreService{pool: pool}
	r, err := c.GetFlowPulse(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Stages) != 7 {
		t.Fatalf("七段应齐: %d", len(r.Stages))
	}
	keys := map[string]bool{}
	for _, st := range r.Stages {
		keys[st.Key] = true
		if st.Label == "" || st.ExpectS <= 0 {
			t.Fatalf("段 %s 缺标/阈: %+v", st.Key, st)
		}
	}
	for _, k := range []string{"collect", "bridge", "eval", "proposal", "exec", "report", "scout"} {
		if !keys[k] {
			t.Fatalf("段 %s 缺", k)
		}
	}
	// roster 段字段在（行数可 0 不炸——fail-soft）
	if r.RosterCount < 0 || r.RosterWorstLagS < 0 {
		t.Fatalf("roster 字段异值: %d/%d", r.RosterCount, r.RosterWorstLagS)
	}
	// 灯阈派生：collect 阈 300s——lag=0 绿；lag>4×阈 红
	if StageLamp("collect", 0, true) != 0 || StageLamp("collect", 2000, true) != 2 {
		t.Fatal("StageLamp 阈派生错位")
	}
}

// 锚：work_items 三类 union——venue measuring 行+proposal 活态+position 未平；
// 排序 proposal→position→venue；fail-soft 表缺不炸面。
func TestFlowPulseWorkItems(t *testing.T) {
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(context.Background(), pool, "md"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_eval(id,ts,inst_id,template,direction,verdict,net_apr,legs,evidence)
		VALUES(9901,now(),'BTC-USDT-SWAP','funding','basis_quarter','proposed',0.1,'[]'::jsonb,'{}'::jsonb)`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_proposal(id,eval_id,inst_id,direction,net_apr,state,created_at)
		VALUES(9901,9901,'BTC-USDT-SWAP','basis_quarter',0.1,'fin_proposed',now())`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_execution(id,inst_id,notional_usd,state)
		VALUES(9901,'ETH-USDT-SWAP',100,'open')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd,opened_at)
		VALUES(9901,'ETH-USDT-SWAP',100,now())`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE fin_venue_candidate SET state='measuring'
		WHERE venue='binance-usdm'`); err != nil {
		t.Fatal(err)
	}
	c := coreService{pool: pool}
	r, err := c.GetFlowPulse(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, w := range r.WorkItems {
		kinds[w.Kind] = true
	}
	for _, k := range []string{"proposal", "position", "venue"} {
		if !kinds[k] {
			t.Fatalf("工件类 %s 缺", k)
		}
	}
	// proposal 首序（money 近者先）
	if r.WorkItems[0].Kind != "proposal" {
		t.Fatalf("序应 proposal 先: %s", r.WorkItems[0].Kind)
	}
}

// 锚：需求驱动段态标（T-0350——提案/渲态非龄，饿/饱两态+全拒带因+被动行
// 剔出+真执行回退龄渲；定时拍段零动归 StageLamp 既有锚）。
func TestDemandStageStatus(t *testing.T) {
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	stage := func(r *dashboardv1.GetFlowPulseResponse, key string) *dashboardv1.FlowStage {
		for _, st := range r.Stages {
			if st.Key == key {
				return st
			}
		}
		t.Fatalf("段 %s 缺", key)
		return nil
	}
	snap := func() *dashboardv1.GetFlowPulseResponse {
		r, err := (coreService{pool: pool}).GetFlowPulse(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	// ① 全空库：两需求段=闲·无 armed（评估零行不称「全拒」——不虚构因）
	if st := stage(snap(), "proposal").Status; st != "闲·无 armed" {
		t.Fatalf("空库提案段态标: %q", st)
	}
	if st := stage(snap(), "exec").Status; st != "闲·无 armed" {
		t.Fatalf("空库执行段态标: %q", st)
	}
	// ② 窗外旧拒不计数（24h 窗语义）
	if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_eval(id,ts,inst_id,template,direction,verdict,legs,evidence)
		VALUES(9902,now()-interval '48 hours','T0350A-USDT-SWAP','funding','basis_quarter','rejected','[]'::jsonb,
		'{"reason":"net_apr -0.9 < 阈 0.0600"}'::jsonb)`); err != nil {
		t.Fatal(err)
	}
	if st := stage(snap(), "proposal").Status; st != "闲·无 armed" {
		t.Fatalf("仅窗外旧拒应仍闲: %q", st)
	}
	// ③ 近窗全拒→带 dominant 短标（数字归一聚 2+1 碎片为同一拒因）
	for i, raw := range []string{"net_apr -0.3346 < 阈 0.0600", "net_apr -0.2798 < 阈 0.0600", "net_apr -0.2878 < 阈 0.0600"} {
		if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_eval(id,ts,inst_id,template,direction,verdict,legs,evidence)
			VALUES($1,now(),$2,'funding','basis_quarter','rejected','[]'::jsonb,jsonb_build_object('reason',$3::text))`,
			int64(9903+i), fmt.Sprintf("T0350R%d-USDT-SWAP", i), raw); err != nil {
			t.Fatal(err)
		}
	}
	if st := stage(snap(), "proposal").Status; st != "闲·评估全拒（末因 net_apr<阈）" {
		t.Fatalf("全拒态标: %q", st)
	}
	// ④ 混入 unevaluable 不称「全拒」
	if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_eval(id,ts,inst_id,template,direction,verdict,legs,evidence)
		VALUES(9906,now(),'T0350B-USDT-SWAP','funding','basis_quarter','unevaluable','[]'::jsonb,
		'{"reason":"apr_est_missing"}'::jsonb)`); err != nil {
		t.Fatal(err)
	}
	if st := stage(snap(), "proposal").Status; st != "闲·无 armed" {
		t.Fatalf("混行态标: %q", st)
	}
	// ⑤ armed 活态→活跃（饱态）
	if _, err := pool.Exec(ctx, `INSERT INTO fin_proposal(id,eval_id,inst_id,direction,net_apr,state,created_at)
		VALUES(9902,9903,'T0350C-USDT-SWAP','basis_quarter',0.1,'fin_proposed',now())`); err != nil {
		t.Fatal(err)
	}
	if st := stage(snap(), "proposal").Status; st != "活跃·1 armed" {
		t.Fatalf("活态态标: %q", st)
	}
	// ⑥ 执行段被动行剔出：interest 在窗仍闲（票面钦定非真执行类）
	if _, err := pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,source_ref,source_table,occurred_at)
		VALUES('interest','okx','USDT',0.01,9902,'fin_x',now())`); err != nil {
		t.Fatal(err)
	}
	if st := stage(snap(), "exec").Status; st != "闲·无 armed" {
		t.Fatalf("被动行态标: %q", st)
	}
	// ⑦ 真执行类在窗→空标回退龄渲（沿态）
	if _, err := pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,source_ref,source_table,occurred_at)
		VALUES('trade_pnl','okx','USDT',1.0,9902,'fin_x',now())`); err != nil {
		t.Fatal(err)
	}
	if st := stage(snap(), "exec").Status; st != "" {
		t.Fatalf("真执行应空标回退龄渲: %q", st)
	}
}

// 锚：拒因短标——数字归一+空白剔净+超长截断（数字=快照瞬值不入短标）。
func TestReasonShort(t *testing.T) {
	for raw, want := range map[string]string{
		"net_apr -0.3346 < 阈 0.0600":                          "net_apr<阈",
		"below_earn: net_apr -0.12 < Earn 底 0.03（静态底 0.05 过）": "below_earn:net_apr<Earn底（静态底过）",
		"basis_below_min": "basis_below_min",
	} {
		if got := reasonShort(raw); got != want {
			t.Fatalf("reasonShort(%q)=%q want %q", raw, got, want)
		}
	}
	long := strings.Repeat("长", 32)
	if got := reasonShort(long); got != strings.Repeat("长", 30)+"…" {
		t.Fatalf("截短: %q", got)
	}
}
