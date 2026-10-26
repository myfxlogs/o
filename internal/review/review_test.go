package review

import (
	"context"
	"os"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func TestVerdictBands(t *testing.T) {
	for _, tc := range []struct {
		pred, act float64
		want      string
	}{
		{1.0, 0.95, "strong"}, {1.0, 0.8, "mild"}, {1.0, 0.5, "marginal"},
		{1.0, -0.1, "falsified"}, {1.0, 0.3, "falsified"}, // dev>60%
	} {
		if got := Verdict(tc.pred, tc.act); got != tc.want {
			t.Fatalf("Verdict(%v,%v)=%s want %s", tc.pred, tc.act, got, tc.want)
		}
	}
}

func setup(t *testing.T) (context.Context, *Runner) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	// T-0167 分链：本包测试面跨域（研究域表+判据键）——两链并集态（md 先 fin 后
	// =历史重放方向：md 链瞬态超表层先落，fin 链状态化/守卫后到）。
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	return ctx, NewRunner(pool)
}

func TestReviewAndForward(t *testing.T) {
	ctx, r := setup(t)
	defer os.RemoveAll(stagingFile) // 测试用待归纳件（仓内生成——测后即清不入档）
	stagingFile = "/tmp/staging_t017.md"
	defer os.Remove(stagingFile)
	// 造 settled 仓+eval 预测行
	var eid, pid, sid, execID int64
	_ = r.pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES('RV-USDT-SWAP','funding-hedge','short_perp_long_spot',1.0,'proposed','{}','{}') RETURNING id`).Scan(&eid)
	var prid int64
	_ = r.pool.QueryRow(ctx,
		"INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr) VALUES($1,'RV-USDT-SWAP','x',1.0) RETURNING id", eid).Scan(&prid)
	_ = r.pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,state,notional_usd)
		VALUES('RV-USDT-SWAP','funding-hedge',$1,'probe',100) RETURNING id`, prid).Scan(&sid)
	_ = r.pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state)
		VALUES($1,'RV-USDT-SWAP',100,'open') RETURNING id`, sid).Scan(&execID)
	_ = r.pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,closed_at,settled_amount,settled_ccy,settled_at)
		VALUES($1,$2,'RV-USDT-SWAP',100,now(),-0.5,'USDT',now()) RETURNING id`, execID, sid).Scan(&pid)
	if err := r.reviewSettled(ctx); err != nil {
		t.Fatal(err)
	}
	var verdict string
	if err := r.pool.QueryRow(ctx,
		"SELECT verdict FROM fin_review WHERE position_ref=$1", pid).Scan(&verdict); err != nil {
		t.Fatal(err)
	}
	if verdict != "falsified" {
		t.Fatalf("actual<0 应 falsified got=%s", verdict)
	}
	// 远期窗 3 行
	var n int
	if err := r.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_forward_check WHERE obj_kind='position' AND obj_ref=$1", pid).Scan(&n); err != nil || n != 3 {
		t.Fatalf("远期窗行=%d", n)
	}
	// falsified→待归纳行+提案态行
	b, _ := os.ReadFile(stagingFile)
	if len(b) == 0 {
		t.Fatal("falsified 未落待归纳行")
	}
	if err := r.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_template_proposal WHERE state='proposed'").Scan(&n); err != nil || n == 0 {
		t.Fatal("复盘写回提案行缺位")
	}
}

func TestRejectedForwardCheck(t *testing.T) {
	ctx, r := setup(t)
	// 否决提案排期锚
	var eid int64
	_ = r.pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence,ts)
		VALUES('RJ-USDT-SWAP','funding-hedge','short_perp_long_spot',0.8,'rejected','{}','{}',now()-interval '25 hours') RETURNING id`).Scan(&eid)
	if err := r.scheduleRejected(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := r.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_forward_check WHERE obj_kind='rejected_proposal' AND obj_ref=$1", eid).Scan(&n); err != nil || n != 1 {
		t.Fatalf("否决提案排期=%d", n)
	}
	// 到期判对错锚：md_funding 正值→anchor>0 actual>0→judged_ok=true
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate)
		VALUES(now()-interval '2 hours','RJ-USDT-SWAP',0.01)`); err != nil {
		t.Fatal(err)
	}
	if err := r.judgeDue(ctx); err != nil {
		t.Fatal(err)
	}
	var ok bool
	if err := r.pool.QueryRow(ctx,
		"SELECT judged_ok FROM fin_forward_check WHERE obj_ref=$1", eid).Scan(&ok); err != nil || !ok {
		t.Fatalf("判对错=%v err=%v", ok, err)
	}
	_ = time.Now
}
