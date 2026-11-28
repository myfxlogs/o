package lifecycle

import (
	"context"
	"fmt"
	"testing"

	"o/internal/platform/testutil"
)

// 工单#45468：drillFeedCandidates 候选集判据——state=probe∧窗内无凭据
// ∧非 simdrill%∧Drillable；fail 凭据在案=已钻不重投（病灶修复归闸语义），
// 窗外陈凭据不算数（execDrillPass entered_at-2d 同窗同真源）。
func TestDrillFeedCandidates(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	seq := 0
	mk := func(tmpl, state string) int64 {
		seq++
		var id int64
		if err := pool.QueryRow(ctx,
			`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue)
			VALUES($1,$2,$3,100,'okx') RETURNING id`,
			fmt.Sprintf("DF-%d-USDT-SWAP", seq), tmpl, state).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	// s1 probe+drillable 无凭据=候选
	s1 := mk("funding-hedge", "probe")
	// s2 probe+窗内 fail 凭据=已钻不重投
	s2 := mk("dir_perp_mom", "probe")
	if _, err := pool.Exec(ctx, `INSERT INTO fin_exec_drill(strategy_ref,inst_id,verdict,rules_version,detail)
		VALUES($1,'DF-TEST-USDT-SWAP','fail','t','test')`, s2); err != nil {
		t.Fatal(err)
	}
	// s3 simdrill% ephemeral=非真策略排除
	s3 := mk("simdrill-cell-test", "probe")
	// s4 paper 态=非 probe 排除
	s4 := mk("funding-hedge", "paper")
	// s5 xvenue-carry=Drillable false（ShapeXvenue Run 拒载族）排除
	s5 := mk("xvenue-carry", "probe")
	// s6 窗外陈凭据（entered_at-2d 前）=不算数仍候选
	s6 := mk("funding-hedge", "probe")
	if _, err := pool.Exec(ctx, `INSERT INTO fin_exec_drill(strategy_ref,inst_id,verdict,rules_version,detail,ts)
		VALUES($1,'DF-TEST-USDT-SWAP','pass','t','test',now()-interval '3 days')`, s6); err != nil {
		t.Fatal(err)
	}

	cands, err := drillFeedCandidates(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	got := map[int64]bool{}
	for _, j := range cands {
		got[j.ref] = true
	}
	for _, id := range []int64{s1, s6} {
		if !got[id] {
			t.Fatalf("应入候选缺席 id=%d", id)
		}
	}
	for _, id := range []int64{s2, s3, s4, s5} {
		if got[id] {
			t.Fatalf("不应入候选却在列 id=%d", id)
		}
	}
}
