package main

import (
	"errors"
	"testing"
	"time"
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

// TestTruncRunes 摘要截断——多字节不截半字。
func TestTruncRunes(t *testing.T) {
	if got := truncRunes("工单#1 认领:开修——修码", 6); got != "工单#1 认" {
		t.Fatalf("截断 %q", got)
	}
	if got := truncRunes("短", 9); got != "短" {
		t.Fatalf("短串 %q", got)
	}
}
