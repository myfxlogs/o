package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"o/internal/gen/dashboardv1"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// TestPanelV2ThreeFaces 三径读面冒烟——漏斗四级序/哑弹同判据/面格栅行数/
// 决策链 open↔引证处置配对；夹具前缀 T-PV2 全清。
func TestPanelV2ThreeFaces(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// 夹具清面：T-PV2 前缀 eval/strategy + 探测席测试 scope + 条件请求件。
	if _, err := pool.Exec(ctx, `DELETE FROM fin_opp_eval WHERE inst_id LIKE 'T-PV2%';
		DELETE FROM fin_strategy WHERE inst_id LIKE 'T-PV2%';
		DELETE FROM ops_event WHERE esc_scope LIKE 't-pv2%';
		DELETE FROM ops_event WHERE event='条件请求' AND note LIKE 'T-PV2%'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	// eval 层夹具：3 rejected + 1 proposed（窗内）。
	for i, v := range []string{"rejected", "rejected", "rejected", "proposed"} {
		// proposed 行 CHECK 强制 net_apr 非空（提案须携净年化判据）。
		if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence,net_apr)
			VALUES($1,'t_pv2','dcd_put',$2,'[]','{}'::jsonb,
				CASE WHEN $2='proposed' THEN 0.1 ELSE NULL END)`,
			fmt.Sprintf("T-PV2-%d", i), v); err != nil {
			t.Fatalf("eval%d: %v", i, err)
		}
	}
	// 哑弹夹具：live+entered>7d+零仓位——该上榜。
	var dudID int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,direction,state,notional_usd,entered_at)
		VALUES('T-PV2-dud','t_pv2','dcd_put','live',500,now()-interval '10 days') RETURNING id`).
		Scan(&dudID); err != nil {
		t.Fatalf("dud: %v", err)
	}
	// 决策链夹具：升件→引证处置（已销）+孤立升件（未决）。
	var esc1, esc2 int64
	if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,esc_class,esc_scope,note)
		VALUES('探测席','升件','测试','t-pv2-done','T-PV2 已销件') RETURNING id`).Scan(&esc1); err != nil {
		t.Fatalf("esc1: %v", err)
	}
	var disp1 int64
	if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1) RETURNING id`,
		fmt.Sprintf("升件#%d 销——T-PV2 判词", esc1)).Scan(&disp1); err != nil {
		t.Fatalf("disp1: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,esc_class,esc_scope,note)
		VALUES('探测席','升件','测试','t-pv2-open','T-PV2 未决件') RETURNING id`).Scan(&esc2); err != nil {
		t.Fatalf("esc2: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','条件请求','T-PV2 请求文')`); err != nil {
		t.Fatalf("condreq: %v", err)
	}
	defer func() {
		pool.Exec(ctx, `DELETE FROM ops_event WHERE id IN ($1,$2,$3)`, esc1, esc2, disp1)
		pool.Exec(ctx, `DELETE FROM ops_event WHERE event='条件请求' AND note LIKE 'T-PV2%'`)
		pool.Exec(ctx, `DELETE FROM fin_opp_eval WHERE inst_id LIKE 'T-PV2%'`)
		pool.Exec(ctx, `DELETE FROM fin_strategy WHERE id=$1`, dudID)
	}()
	svc := coreService{pool: pool}

	// —— GetFunnel：四级序齐+哑弹同判据上榜 ——
	f, err := svc.GetFunnel(ctx, &dashboardv1.GetFunnelRequest{WindowDays: 30})
	if err != nil {
		t.Fatalf("GetFunnel: %v", err)
	}
	if len(f.Levels) != 4 || f.Levels[0].Key != "eval" || f.Levels[3].Key != "position" {
		t.Fatalf("漏斗四级序错: %+v", f.Levels)
	}
	// 全层 n≥0——fail-soft -1 显缺闸（实库 string_agg/ts 双虫教训：层读件缺
	// 静默过闸=面板上「读件缺」无警，层断言锁读径）。
	for _, l := range f.Levels {
		if l.N < 0 {
			t.Fatalf("%s 层读件缺（%s）——查询径断", l.Key, l.Note)
		}
	}
	if f.Levels[0].N < 4 {
		t.Fatalf("eval 层漏夹具: n=%d", f.Levels[0].N)
	}
	var sawDud bool
	for _, d := range f.Duds {
		if d.Id == dudID {
			sawDud = true
		}
	}
	if !sawDud {
		t.Fatalf("哑弹榜漏夹具 dudID=%d", dudID)
	}

	// —— GetFaceGrid：面行非零+会话/部署面在册（容缺不炸） ——
	g, err := svc.GetFaceGrid(ctx, &dashboardv1.GetFaceGridRequest{})
	if err != nil {
		t.Fatalf("GetFaceGrid: %v", err)
	}
	if len(g.Faces) == 0 {
		t.Fatalf("faces 零行——healthComps 注册表缺席")
	}
	for _, fc := range g.Faces {
		if fc.State < 0 || fc.State > 3 {
			t.Fatalf("face %s 非法 state=%d", fc.Name, fc.State)
		}
	}
	if g.Session == nil {
		t.Fatalf("session 面缺席")
	}

	// —— GetDecisionTrail：open↔引证配对+条件请求入榜 ——
	tr, err := svc.GetDecisionTrail(ctx, &dashboardv1.GetDecisionTrailRequest{Limit: 50})
	if err != nil {
		t.Fatalf("GetDecisionTrail: %v", err)
	}
	var done, open bool
	for _, r := range tr.Trail {
		if r.EscId == esc1 {
			done = true
			if r.Open || r.DispId != disp1 || r.Verdict == "" {
				t.Fatalf("已销件配对错: %+v", r)
			}
		}
		if r.EscId == esc2 {
			open = true
			if !r.Open {
				t.Fatalf("未决件判态错: %+v", r)
			}
		}
	}
	if !done || !open {
		t.Fatalf("链夹具缺席: done=%v open=%v", done, open)
	}
	var sawCond bool
	for _, r := range tr.Condreqs {
		if r.Note == "T-PV2 请求文" {
			sawCond = true
			if !r.Open {
				t.Fatalf("条件请求 open 判态错: %+v", r)
			}
		}
	}
	if !sawCond {
		t.Fatalf("条件请求榜漏夹具")
	}
	if tr.Mine == nil {
		t.Fatalf("矿面缺席")
	}
}

// TestFaceEvalBoundaries 判态边界单测——zeroState/zeroNote/dead/warn 四支。
func TestFaceEvalBoundaries(t *testing.T) {
	now := time.Now()
	b := compBeat{name: "t", warn: time.Minute, dead: 10 * time.Minute,
		zeroState: 3, zeroNote: "中性零行", fix: "修法注"}
	if st, note, _ := faceEval(b, nil, fmt.Errorf("q")); st != 2 || note == "" {
		t.Fatalf("查询败支: st=%d note=%q", st, note)
	}
	if st, note, _ := faceEval(b, nil, nil); st != 3 || note != "中性零行" {
		t.Fatalf("zeroNote 支: st=%d note=%q", st, note)
	}
	fresh := now.Add(-10 * time.Second)
	if st, _, lag := faceEval(b, &fresh, nil); st != 0 || lag <= 0 {
		t.Fatalf("鲜支: st=%d lag=%d", st, lag)
	}
	stale := now.Add(-5 * time.Minute)
	if st, note, _ := faceEval(b, &stale, nil); st != 1 || note != "修法注" {
		t.Fatalf("warn 支: st=%d note=%q", st, note)
	}
	dead := now.Add(-30 * time.Minute)
	if st, note, _ := faceEval(b, &dead, nil); st != 2 || note != "修法注" {
		t.Fatalf("dead 支: st=%d note=%q", st, note)
	}
}
