package seatcard

// T-0458 班态判定表（CTO 裁定①五标——unit 态∧事件龄双证+行在案覆盖）
// 与装配面（别名归并/在办五事件/队列深穿透/零活性闸）桩测。

import (
	"context"
	"testing"
	"time"
)

func TestDeriveStatus(t *testing.T) {
	now := base
	row := func(event string, age time.Duration) *EventRow {
		return &EventRow{Seat: "X", Ts: now.Add(-age), Event: event, Ticket: "T-1"}
	}
	cases := []struct {
		name   string
		latest *EventRow
		active bool
		want   string
	}{
		{"挂起行在案", row("挂起", time.Minute), true, StatusBlocked},
		{"挂起行在案-unit停", row("挂起", time.Minute), false, StatusBlocked},
		{"异常行在案", row("异常", time.Minute), true, StatusError},
		{"退班行-unit停", row("退班", time.Hour), false, StatusOff},
		{"退班行优先于龄兜底", row("退班", 3*time.Hour), false, StatusOff},
		{"在岗-新事件", row("领单", 5*time.Minute), true, StatusOnDuty},
		{"作业中-active事件老", row("领单", 30*time.Minute), true, StatusWorking},
		{"作业中-班未退拍间态", row("领单", 30*time.Minute), false, StatusWorking},
		{"异常-2h无动未退班", row("领单", 3*time.Hour), false, StatusError},
		{"异常-2h无动unit活", row("领单", 3*time.Hour), true, StatusWorking},
		{"零行", nil, false, StatusOff},
	}
	for _, tc := range cases {
		if got := DeriveStatus(tc.latest, tc.active, now); got != tc.want {
			t.Errorf("%s: got=%q want=%q", tc.name, got, tc.want)
		}
	}
}

func TestAssembleAliasesAndWorking(t *testing.T) {
	st := &memStore{rows: map[string][]EventRow{
		"CTO": {
			{Seat: "CTO", Ts: base.Add(-time.Hour), Event: "裁决", Ticket: "T-0455", Note: "通过"},
			{Seat: "CTO", Ts: base.Add(-3 * time.Hour), Event: "移交", Ticket: "T-0455"},
		},
		"决策席": { // 旧席名归并——末动/在办不断史
			{Seat: "决策席", Ts: base.Add(-30 * time.Minute), Event: "领单", Ticket: "T-0458"},
		},
	}, revApp: [2]int{3, 5}}
	states, err := Assemble(context.Background(), st, testProbe(true), Phase1Seats, base)
	if err != nil {
		t.Fatalf("装配败: %v", err)
	}
	if len(states) != len(Phase1Seats) {
		t.Fatalf("席数=%d 期 %d", len(states), len(Phase1Seats))
	}
	var cto *CardState
	for i := range states {
		if states[i].Seat == "CTO席" {
			cto = &states[i]
		}
	}
	if cto == nil {
		t.Fatal("CTO席 卡缺")
	}
	if cto.Latest == nil || cto.Latest.Ticket != "T-0458" {
		t.Fatalf("别名归并败——末动应取最新跨别名行: %+v", cto.Latest)
	}
	if cto.Working == nil || cto.Working.Event != "领单" {
		t.Fatalf("在办段应取最近五事件行: %+v", cto.Working)
	}
	if cto.QueueReview != 3 || cto.QueueApproved != 5 {
		t.Fatalf("队列深穿透断: %d/%d", cto.QueueReview, cto.QueueApproved)
	}
	if len(cto.Workers) != 0 {
		t.Fatalf("CTO席 不应有并行活件行: %+v", cto.Workers)
	}
	// 批量班不糊律：并行活件行归施工席卡（o-worker-* 探针面）。
	var workerRow *Worker
	for i := range states {
		if states[i].Seat == "施工席" && len(states[i].Workers) > 0 {
			workerRow = &states[i].Workers[0]
		}
	}
	if workerRow == nil || workerRow.TID != "T-0001" {
		t.Fatal("施工席并行活件行缺")
	}
}

func TestActiveSeatGate(t *testing.T) {
	live := CardState{Latest: &EventRow{Ts: base.Add(-time.Hour)}}
	dead := CardState{Latest: &EventRow{Ts: base.Add(-49 * time.Hour)}}
	none := CardState{}
	if !ActiveSeat(live, base) {
		t.Fatal("活性席误判死")
	}
	if ActiveSeat(dead, base) || ActiveSeat(none, base) {
		t.Fatal("零活性席误判活——懒建闸破")
	}
}

// workerTID unit 名→票号。
func TestWorkerTID(t *testing.T) {
	if got := workerTID("o-worker-T-0451.service"); got != "T-0451" {
		t.Fatalf("workerTID=%q", got)
	}
}
