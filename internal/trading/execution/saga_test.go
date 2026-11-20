package execution

import (
	"context"
	"encoding/json"
	"testing"
)

// saga 账锚（工单#25807）：常态成=全步 done、leg2 败=leg1 undone 在账、
// 断点恢复=陈 active 转 compensating 反序调 undo、补偿败=failed 留尸。

func stepOf(t *testing.T, ctx context.Context, r *Runner, sagaID int64, name string) sagaStep {
	t.Helper()
	var b []byte
	if err := r.pool.QueryRow(ctx, `SELECT steps FROM fin_saga WHERE id=$1`, sagaID).Scan(&b); err != nil {
		t.Fatal(err)
	}
	var steps []sagaStep
	if err := json.Unmarshal(b, &steps); err != nil {
		t.Fatal(err)
	}
	for _, s := range steps {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("步缺: %s", name)
	return sagaStep{}
}

func sagaRow(t *testing.T, ctx context.Context, r *Runner, execID int64) (int64, string) {
	t.Helper()
	var id int64
	var st string
	if err := r.pool.QueryRow(ctx, `SELECT id,state FROM fin_saga WHERE ref_id=$1`, execID).Scan(&id, &st); err != nil {
		t.Fatalf("saga 行缺 exec#%d: %v", execID, err)
	}
	return id, st
}

func TestSagaNormalFlowDone(t *testing.T) {
	ctx, r := setup(t, &fakeVenue{})
	did := mkToken(t, ctx, r, false, false, "allow")
	eid, err := r.Execute(ctx, mkIntent(did))
	if err != nil {
		t.Fatal(err)
	}
	sid, st := sagaRow(t, ctx, r, eid)
	if st != "done" {
		t.Fatalf("saga state=%s（应 done）", st)
	}
	if s := stepOf(t, ctx, r, sid, "leg1"); s.Status != "done" || s.UndoRef != "rollback_leg" {
		t.Fatalf("leg1 步=%+v", s)
	}
	if s := stepOf(t, ctx, r, sid, "earn_redeem"); s.Status != "skipped" {
		t.Fatalf("earn 未挂应 skipped: %+v", s)
	}
	if s := stepOf(t, ctx, r, sid, "position_row"); s.Status != "done" {
		t.Fatalf("position_row=%+v", s)
	}
}

func TestSagaLeggedUndoTrace(t *testing.T) {
	fv := &fakeVenue{leg2Fail: true}
	ctx, r := setup(t, fv)
	did := mkToken(t, ctx, r, false, false, "allow")
	eid, err := r.Execute(ctx, mkIntent(did))
	if err == nil {
		t.Fatal("leg2 败应 LEGGED")
	}
	sid, st := sagaRow(t, ctx, r, eid)
	if st != "done" {
		t.Fatalf("saga=%s（已处置败局=done——'failed' 域=补偿败专属）", st)
	}
	if s := stepOf(t, ctx, r, sid, "leg2"); s.Status != "failed" {
		t.Fatalf("leg2=%+v", s)
	}
	if s := stepOf(t, ctx, r, sid, "leg1"); s.Status != "undone" {
		t.Fatalf("leg1=%+v（inline 回滚应 undone）", s)
	}
}

// 断点恢复锚：手工造陈 active saga（崩于 leg1 成后/leg2 前）→ResumeStaleSagas
// 转 compensating→leg1 反序收回→saga done+exec rolled_back。
func TestSagaResumeCompensates(t *testing.T) {
	fv := &fakeVenue{}
	ctx, r := setup(t, fv)
	// 宿主行+成腿在账（模拟崩中尸——saga active 陈+leg1 filled）。
	var eid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state)
		VALUES(1,'BTC-USDT-SWAP',100,'open') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	r.legPx(ctx, eid, 1, Leg{Venue: "okx", InstID: "BTC-USDT", Side: "buy", Sz: "0.01"},
		"ph1x1x1", "filled", "", "", 50000, 0.01, "market", 0)
	sid := r.sagaBegin(ctx, "open", eid, []sagaStep{
		{Name: "exec_row", Status: "done", UndoRef: "fail_exec"},
		{Name: "earn_redeem", Status: "skipped"},
		{Name: "leg1", Status: "done", UndoRef: "rollback_leg"},
		{Name: "leg2", Status: "pending", UndoRef: "rollback_leg"},
		{Name: "position_row", Status: "pending", UndoRef: "delete_position"},
	})
	// 陈化（10min 阈外——模拟重启前尸）。
	if _, err := r.pool.Exec(ctx, `UPDATE fin_saga SET created_at=now()-interval '1 hour' WHERE id=$1`, sid); err != nil {
		t.Fatal(err)
	}
	if n := r.ResumeStaleSagas(ctx); n != 1 {
		t.Fatalf("resume 处置数=%d（应 1）", n)
	}
	var st string
	if err := r.pool.QueryRow(ctx, `SELECT state FROM fin_saga WHERE id=$1`, sid).Scan(&st); err != nil || st != "done" {
		t.Fatalf("saga=%s（补偿走完应 done）", st)
	}
	if s := stepOf(t, ctx, r, sid, "leg1"); s.Status != "undone" {
		t.Fatalf("leg1 补偿后=%+v", s)
	}
	var es string
	if err := r.pool.QueryRow(ctx, `SELECT state FROM fin_execution WHERE id=$1`, eid).Scan(&es); err != nil {
		t.Fatal(err)
	}
	if es != "rolled_back" {
		t.Fatalf("exec=%s（腿回滚收口应 rolled_back——fail_exec 禁覆写在案）", es)
	}
	// 鲜 saga 不触（10min 阈内——拍内死误判护栏）。
	sid2 := r.sagaBegin(ctx, "open", eid, []sagaStep{{Name: "leg1", Status: "pending"}})
	if n := r.ResumeStaleSagas(ctx); n != 0 {
		t.Fatalf("鲜 saga 被误收: %d", n)
	}
	r.pool.Exec(ctx, `DELETE FROM fin_saga WHERE id=$1`, sid2)
}

// 补偿败锚：rollback 径全瘫→undo_failed 步钉+saga failed 留尸（票面③）。
func TestSagaCompensateFailLeavesCorpse(t *testing.T) {
	fv := &fakeVenue{rbFail: true}
	ctx, r := setup(t, fv)
	var eid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state)
		VALUES(1,'BTC-USDT-SWAP',100,'open') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	r.legPx(ctx, eid, 1, Leg{Venue: "okx", InstID: "BTC-USDT", Side: "buy", Sz: "0.01"},
		"ph1x1x2", "filled", "", "", 50000, 0.01, "market", 0)
	sid := r.sagaBegin(ctx, "open", eid, []sagaStep{
		{Name: "exec_row", Status: "done", UndoRef: "fail_exec"},
		{Name: "leg1", Status: "done", UndoRef: "rollback_leg"},
	})
	r.sagaCompensate(ctx, sid, eid)
	var st string
	if err := r.pool.QueryRow(ctx, `SELECT state FROM fin_saga WHERE id=$1`, sid).Scan(&st); err != nil || st != "failed" {
		t.Fatalf("saga=%s（补偿败应 failed 留尸）", st)
	}
	if s := stepOf(t, ctx, r, sid, "leg1"); s.Status != "undo_failed" {
		t.Fatalf("leg1=%+v", s)
	}
}
