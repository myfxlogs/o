package main

import (
	"testing"
	"time"
)

func TestFrameText(t *testing.T) {
	now := time.Now()
	for _, c := range []struct {
		name    string
		seats   []frameSeat
		esc     int64
		tickets int64
		pick    int64
		want    string
	}{
		{"空闲", nil, 0, 8, 2, "在办: 全员空闲 | 信号0 · 工单8(可拣2)"},
		{"施工领票", []frameSeat{{seat: "施工席", ticket: 11481,
			subject: "stall探针误报簇", claimTs: now.Add(-37 * time.Minute)}},
			2, 7, 4, "在办: 施工在修工单#11481「stall探针误报簇」(37分) | 信号2 · 工单7(可拣4)"},
		{"在跑未领", []frameSeat{{seat: "唤知席"}}, 1, 9, 9,
			"在办: 值班在跑(未领件) | 信号1 · 工单9(可拣9)"},
		{"双席", []frameSeat{
			{seat: "施工席", ticket: 11481, claimTs: now.Add(-2 * time.Hour)},
			{seat: "审计席"}},
			0, 5, 3, "在办: 施工在修工单#11481(2h0分) | 审计在跑(未领件) | 信号0 · 工单5(可拣3)"},
		{"刚领", []frameSeat{{seat: "施工席", ticket: 7, claimTs: now}},
			0, 1, 1, "在办: 施工在修工单#7(刚领) | 信号0 · 工单1(可拣1)"},
	} {
		if got := frameText(c.seats, c.esc, c.tickets, c.pick); got != c.want {
			t.Fatalf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

func TestFrameAge(t *testing.T) {
	for _, c := range []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "刚领"},
		{59 * time.Minute, "59分"},
		{90 * time.Minute, "1h30分"},
	} {
		if got := frameAge(c.d); got != c.want {
			t.Fatalf("age %v: got %q want %q", c.d, got, c.want)
		}
	}
}
