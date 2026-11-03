// flow_pulse_test.go —— 活水段桩（T-0307：七段+心跳行+fail-soft 显缺不炸）。
package main

import (
	"context"
	"testing"

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
