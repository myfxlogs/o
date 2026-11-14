// doingRows 在办检查点面:开在办→在干行出;同unit 收口/果败(note unit=)
// 或处置引 在办#id→闭行不出;2h 无同unit事件→悬账;多unit并行各出一行。
package main

import (
	"context"
	"fmt"
	"testing"

	"o/internal/gen/dashboardv1"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func TestDoingRows(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event`); err != nil {
		t.Fatal(err)
	}
	svc := coreService{pool: pool}
	ev := func(seat, event, ticket, note string, ageMin int) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,ticket,note,ts)
			VALUES($1,$2,$3,$4,now()-$5*interval '1 minute') RETURNING id`,
			seat, event, ticket, note, ageMin).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	rows := func() []*dashboardv1.CogDoingRow {
		r := &dashboardv1.GetCockpitResponse{}
		svc.doingRows(ctx, r)
		if len(r.Missing) > 0 {
			t.Fatalf("doing missing: %v", r.Missing)
		}
		return r.Doing
	}
	find := func(unit string) *dashboardv1.CogDoingRow {
		for _, d := range rows() {
			if d.Unit == unit {
				return d
			}
		}
		return nil
	}

	// ① 空面→零行
	if n := len(rows()); n != 0 {
		t.Fatalf("empty want 0 got %d", n)
	}
	// ② u1 开在办(10min)→在干
	ev("唤知", "在办", "o-cognition-1", "升件#9001:取证中", 10)
	d := find("o-cognition-1")
	if d == nil || d.State != "在干" {
		t.Fatalf("u1 want 在干 got %+v", d)
	}
	// ③ u2 并行开在办→两行各在
	ev("唤知", "在办", "o-cognition-2", "工单#42:施工中", 5)
	if find("o-cognition-2") == nil || find("o-cognition-1") == nil {
		t.Fatal("u2 parallel missing")
	}
	// ④ u1 相位变更(新在办行)→显最新一行
	ev("唤知", "在办", "o-cognition-1", "升件#9001:落钉收口", 2)
	if d = find("o-cognition-1"); d == nil || d.Doing != "升件#9001:落钉收口" {
		t.Fatalf("u1 phase want latest got %+v", d)
	}
	// ⑤ u2 会话收口(note unit= 配)→闭行出局
	ev("唤知", "会话收口", "", "unit=o-cognition-2 第1轮:毕", 1)
	if find("o-cognition-2") != nil {
		t.Fatal("u2 closed still shown")
	}
	// ⑥ u3 开在办→唤起败闭
	ev("唤知", "在办", "o-cognition-3", "矿件:在挖", 3)
	ev("唤知机", "唤起败", "", "unit=o-cognition-3 rc=1", 1)
	if find("o-cognition-3") != nil {
		t.Fatal("u3 dead-closed still shown")
	}
	// ⑦ u4 开在办→处置引 在办#id 弃置销
	id4 := ev("唤知", "在办", "o-cognition-4", "陈残:弃置", 1)
	ev("处置席", "处置", "", fmt.Sprintf("在办#%d 销——陈残弃置", id4), 0)
	if find("o-cognition-4") != nil {
		t.Fatal("u4 disposed still shown")
	}
	// ⑧ u5 开在办但 3h 无同unit事件→悬账
	ev("唤知", "在办", "o-cognition-5", "猝死残:断气前在干X", 200)
	if d = find("o-cognition-5"); d == nil || d.State != "悬账" {
		t.Fatalf("u5 want 悬账 got %+v", d)
	}
}
