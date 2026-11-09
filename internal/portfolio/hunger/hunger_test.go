package hunger

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// T-0376 锚③：饥饿态触发桩测——构造 zero-proposal 窗→响应序列跑通→
// 每步事件行落账；锚④运行半面：venue 推进止步于条件请求盒（凭据边界=
// 业主面，无任何凭据写径）。

func hungerFixture(t *testing.T) (*Sensor, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return New(pool, cfg), pool
}

func seedEval(t *testing.T, pool *pgxpool.Pool, template string) int64 {
	t.Helper()
	var evalID int64
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,net_apr,legs,evidence)
		VALUES('okx:perp:BTC-USDT-SWAP',$1,'long_perp_short_spot','proposed',0.1,'[]','{}') RETURNING id`,
		template).Scan(&evalID); err != nil {
		t.Fatal(err)
	}
	return evalID
}

// seedProposal 鲜提案（age=行龄——构造窗口用）。
func seedProposal(t *testing.T, pool *pgxpool.Pool, evalID int64, age time.Duration) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr,created_at)
		VALUES($1,'okx:perp:BTC-USDT-SWAP','long_perp_short_spot',0.1,now()-make_interval(secs=>$2))`,
		evalID, age.Seconds()); err != nil {
		t.Fatal(err)
	}
}

func TestDetectTriggersOnZeroProposalAndPaperEmpty(t *testing.T) {
	s, _ := hungerFixture(t)
	rep, err := s.Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if !rep.Triggered {
		t.Fatal("净库（零提案+零 paper）应触发饥饿")
	}
	if len(rep.Reasons) != 2 || rep.WindowDays != 3 {
		t.Fatalf("机读定义不符: %+v", rep)
	}
	joined := strings.Join(rep.Reasons, "|")
	if !strings.Contains(joined, "zero_proposal") || !strings.Contains(joined, "paper_empty") {
		t.Fatalf("触发形词缺: %v", rep.Reasons)
	}
}

func TestDetectQuietWithFreshProposalAndPaperPool(t *testing.T) {
	s, pool := hungerFixture(t)
	ctx := context.Background()
	seedProposal(t, pool, seedEval(t, pool, "funding"), time.Hour)
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd)
		VALUES('okx:perp:BTC-USDT-SWAP','funding','paper',100)`); err != nil {
		t.Fatal(err)
	}
	rep, err := s.Detect(ctx)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if rep.Triggered {
		t.Fatalf("鲜提案+paper 池在不应触发: %+v", rep)
	}
}

func TestRespondSequenceRunsAllStepsWithRows(t *testing.T) {
	s, pool := hungerFixture(t)
	ctx := context.Background()
	// 桩景：休眠族有 proposed 评估证据（复活径）；recommended venue（请求盒补位径）。
	seedEval(t, pool, "sleep_family")
	if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_template(template,venue,enabled,state)
		VALUES('sleep_family','okx',false,'retired')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_candidate(venue,kind,state,kyc_ok)
		VALUES('bybit','cex-perp','recommended',true)
		ON CONFLICT (venue) DO UPDATE SET state='recommended',kyc_ok=true`); err != nil {
		t.Fatal(err)
	}
	rep := &Report{Triggered: true, WindowDays: 3, Reasons: []string{"zero_proposal:桩景"}}
	steps, err := s.Respond(ctx, rep)
	if err != nil {
		t.Fatalf("respond: %v", err)
	}
	if len(steps) != 4 {
		t.Fatalf("四步序列: %d", len(steps))
	}
	for i, want := range []string{"放行线检校", "休眠模板族再评估", "低产live重组候选", "venue候选推进"} {
		if steps[i].Step != want {
			t.Fatalf("步序 %d=%s 非 %s", i, steps[i].Step, want)
		}
	}
	// 复活径实迹：族翻位+参数调整行三列齐。
	var en bool
	var st string
	if err := pool.QueryRow(ctx,
		`SELECT enabled,state FROM fin_opp_template WHERE template='sleep_family'`,
	).Scan(&en, &st); err != nil || !en || st != "enabled" {
		t.Fatalf("休眠族未带证据直启: en=%v st=%s err=%v", en, st, err)
	}
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='参数调整' AND ticket='' ORDER BY id DESC LIMIT 1`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	for _, frag := range []string{"target=fin_opp_template", "旧=enabled=false/state=retired",
		"新=enabled=true/state=enabled", "证据=proposed_eval×1"} {
		if !strings.Contains(note, frag) {
			t.Fatalf("参数调整行缺列 %q: %s", frag, note)
		}
	}
	// venue 推进止步请求盒：credential 条件请求落位。
	var gapn int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM fin_condition_gap WHERE kind='credential' AND venue='bybit' AND state='pending'`,
	).Scan(&gapn); err != nil || gapn != 1 {
		t.Fatalf("凭据边界条件请求缺: n=%d err=%v", gapn, err)
	}
	// 每步事件行落账（四步齐）。
	var evn int
	if err := pool.QueryRow(ctx,
		`SELECT count(DISTINCT note) FROM ops_event WHERE event='饥饿响应' AND ticket=''`,
	).Scan(&evn); err != nil || evn < 4 {
		t.Fatalf("步事件行不齐: %d", evn)
	}
}

func TestRespondQuietWindowNoFalseAction(t *testing.T) {
	s, pool := hungerFixture(t)
	ctx := context.Background()
	// 无休眠族无 recommended——序列跑通零动作（检校照常落行）。
	rep := &Report{Triggered: true, WindowDays: 3, Reasons: []string{"zero_proposal:桩景"}}
	steps, err := s.Respond(ctx, rep)
	if err != nil {
		t.Fatalf("respond: %v", err)
	}
	for _, st := range steps {
		if st.Acted {
			t.Fatalf("净库不应有动作: %+v", st)
		}
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE event='参数调整'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("零证据不得落参数调整行: %d", n)
	}
}

func TestDetectFailsClosedOnBadWindowKey(t *testing.T) {
	s, pool := hungerFixture(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`UPDATE app_config SET value='0' WHERE key='dm.hunger.zero_proposal_days'`); err != nil {
		t.Fatal(err)
	}
	if err := s.cfg.Load(ctx); err != nil { // 热读快照重载——写后取值为准
		t.Fatal(err)
	}
	if _, err := s.Detect(ctx); err == nil {
		t.Fatal("窗键非正应 fail-closed 拒（检测器不带默认阈滑行）")
	}
}
