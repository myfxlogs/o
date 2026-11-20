package guard

// fsm_test.go —— fin_position FSM 收口测试（工单#25806）：
//   · 对账孤儿径：exec 终态∧pos 活→loopProbe 标 orphaned（验收钉死路径）；
//   · 物化随动：closed_at 置位→state 恒随 closed（触发器单点）；
//   · 转移合法集：legal 边过/illegal 边（closed→* 等）落账拒。

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestFSMOrphanReconcile execution 断腿（终态）∧position 活→对账标 orphaned；
// exec 仍 open（正常在途）→不动。
func TestFSMOrphanReconcile(t *testing.T) {
	ctx, w, _ := setup(t)
	posID := mkPosition(t, ctx, w, "live", 1, time.Now())
	var execID int64
	if err := w.pool.QueryRow(ctx,
		`SELECT execution_id FROM fin_position WHERE id=$1`, posID).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	// 正常在途（exec open）→loopProbe 不动 state。
	w.loopProbe(ctx)
	var st string
	if err := w.pool.QueryRow(ctx,
		`SELECT state FROM fin_position WHERE id=$1`, posID).Scan(&st); err != nil {
		t.Fatal(err)
	}
	if st != "open" {
		t.Fatalf("exec 在途仓不应标孤儿，got state=%q", st)
	}
	// 断腿：exec 迁终态 failed→loopProbe 对账标 orphaned。
	if _, err := w.pool.Exec(ctx,
		`UPDATE fin_execution SET state='failed' WHERE id=$1`, execID); err != nil {
		t.Fatal(err)
	}
	w.loopProbe(ctx)
	if err := w.pool.QueryRow(ctx,
		`SELECT state FROM fin_position WHERE id=$1`, posID).Scan(&st); err != nil {
		t.Fatal(err)
	}
	if st != "orphaned" {
		t.Fatalf("exec 终态仓应对账标 orphaned，got state=%q", st)
	}
	// 幂等：再扫不炸不迁（orphaned→orphaned 同态过）。
	w.loopProbe(ctx)
}

// TestFSMClosedAutoDerive closed_at 置位→state 恒随 closed（物化防漏写——
// 存量/测试径直写 closed_at 自愈归位）。
func TestFSMClosedAutoDerive(t *testing.T) {
	ctx, w, _ := setup(t)
	posID := mkPosition(t, ctx, w, "live", 1, time.Now())
	if _, err := w.pool.Exec(ctx,
		`UPDATE fin_position SET closed_at=now() WHERE id=$1`, posID); err != nil {
		t.Fatal(err)
	}
	var st string
	if err := w.pool.QueryRow(ctx,
		`SELECT state FROM fin_position WHERE id=$1`, posID).Scan(&st); err != nil {
		t.Fatal(err)
	}
	if st != "closed" {
		t.Fatalf("closed_at 置位应随动 state=closed，got %q", st)
	}
}

// TestFSMLegalTransitions 合法边全谱过：open→closing→closed、
// open→orphaned→closed（orphan 收口径）。
func TestFSMLegalTransitions(t *testing.T) {
	ctx, w, _ := setup(t)
	posID := mkPosition(t, ctx, w, "live", 1, time.Now())
	// open→closing（markClosing 业务写点同径）。
	if err := w.markClosing(ctx, posID, "funding_flip"); err != nil {
		t.Fatal(err)
	}
	var st string
	if err := w.pool.QueryRow(ctx,
		`SELECT state FROM fin_position WHERE id=$1`, posID).Scan(&st); err != nil {
		t.Fatal(err)
	}
	if st != "closing" {
		t.Fatalf("markClosing 应迁 closing，got %q", st)
	}
	// closing→closed（平仓确认——closed_at 置位随动）。
	if _, err := w.pool.Exec(ctx,
		`UPDATE fin_position SET closed_at=now() WHERE id=$1`, posID); err != nil {
		t.Fatal(err)
	}
	if err := w.pool.QueryRow(ctx,
		`SELECT state FROM fin_position WHERE id=$1`, posID).Scan(&st); err != nil {
		t.Fatal(err)
	}
	if st != "closed" {
		t.Fatalf("closing 平仓确认应 closed，got %q", st)
	}
	// open→orphaned→closed（孤儿收口径——closed_at 同置）。
	// active_dedupe（venue,inst,direction,template WHERE state<>'retired'）
	// ——首策略退役让位再造第二 fixture。
	if _, err := w.pool.Exec(ctx,
		`UPDATE fin_strategy SET state='retired' WHERE template_ref='guard-test'`); err != nil {
		t.Fatal(err)
	}
	posID2 := mkPosition(t, ctx, w, "live", 1, time.Now())
	for _, edge := range []struct {
		q string
	}{{`UPDATE fin_position SET state='orphaned' WHERE id=` + itoa(posID2)},
		{`UPDATE fin_position SET closed_at=now() WHERE id=` + itoa(posID2)}} {
		if _, err := w.pool.Exec(ctx, edge.q); err != nil {
			t.Fatalf("合法边应过: %s: %v", edge.q, err)
		}
	}
	if err := w.pool.QueryRow(ctx,
		`SELECT state FROM fin_position WHERE id=$1`, posID2).Scan(&st); err != nil {
		t.Fatal(err)
	}
	if st != "closed" {
		t.Fatalf("orphaned 收口应 closed，got %q", st)
	}
}

// TestFSMIllegalTransition 非法迁移落账拒：closed→open、closed→orphaned、
// orphaned→open、state=closed 而 closed_at NULL。
func TestFSMIllegalTransition(t *testing.T) {
	ctx, w, _ := setup(t)
	posID := mkPosition(t, ctx, w, "live", 1, time.Now())
	if _, err := w.pool.Exec(ctx,
		`UPDATE fin_position SET closed_at=now() WHERE id=$1`, posID); err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{
		`UPDATE fin_position SET state='open' WHERE id=` + itoa(posID),
		`UPDATE fin_position SET state='orphaned' WHERE id=` + itoa(posID),
		`UPDATE fin_position SET state='closing' WHERE id=` + itoa(posID),
	} {
		if _, err := w.pool.Exec(ctx, edge); err == nil || !strings.Contains(err.Error(), "转移非法") {
			t.Fatalf("非法迁移应拒: %s: err=%v", edge, err)
		}
	}
	// state=closed 而 closed_at NULL=矛盾行禁立。
	// active_dedupe（venue,inst,direction,template WHERE state<>'retired'）
	// ——退役首策略让位再造第二 fixture（同 TestFSMLegalTransitions 例）。
	if _, err := w.pool.Exec(ctx,
		`UPDATE fin_strategy SET state='retired' WHERE template_ref='guard-test'`); err != nil {
		t.Fatal(err)
	}
	posID2 := mkPosition(t, ctx, w, "live", 1, time.Now())
	if _, err := w.pool.Exec(ctx,
		`UPDATE fin_position SET state='closed' WHERE id=`+itoa(posID2)); err == nil ||
		!strings.Contains(err.Error(), "闭合必带时标") {
		t.Fatalf("closed 无时标应拒: err=%v", err)
	}
}

// itoa int64→string（测试 SQL 拼接辅助——参数化同效，直拼免双占位）。
func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}
