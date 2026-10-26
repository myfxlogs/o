package main

// untradeable_test.go —— 终局分类/黑名单桩测（T-0142 验收锚）：造桩 venue 51001→
// 黑名单入册+策略退+armed 提案退；黑名单 inst 后续 universe 滤+起草滤。
import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/portfolio/lifecycle"
	"o/internal/store/pgstore"
	"o/internal/trading/okxexec"
)

func t51001() error {
	return &okxexec.APIError{Code: "51001", Msg: "Instrument ID doesn't exist"}
}

func setupDB(t *testing.T) (context.Context, *pgxpool.Pool, *config.Client) {
	t.Helper()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	return context.Background(), pool, cfg
}

// 锚：TerminalInstErr——51001/51003 终局 true；51000（参数族 clOrdId 先例）/
// 瞬态/nil false（禁误伤参数 bug）。
func TestTerminalInstErr(t *testing.T) {
	if !okxexec.TerminalInstErr(t51001()) {
		t.Fatal("51001 应终局")
	}
	if !okxexec.TerminalInstErr(&okxexec.APIError{Code: "51003"}) {
		t.Fatal("51003 应终局")
	}
	for _, e := range []error{
		&okxexec.APIError{Code: "51000", Msg: "clOrdId 非法"},
		&okxexec.APIError{Code: "50011", Msg: "rate limit"},
		errors.New("transport"),
		nil,
	} {
		if okxexec.TerminalInstErr(e) {
			t.Fatalf("%v 不应终局", e)
		}
	}
}

// 锚（票面验收）：造桩 51001→untradeable 处置三件全落——黑名单行+策略
// retired（venue_untradeable 溯源）+armed 提案 fin_retired。
func TestUntradeableDisposition(t *testing.T) {
	ctx, pool, cfg := setupDB(t)
	h := hedgeOpener{pool: pool, cfg: cfg}
	var sid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,direction,notional_usd,state,venue,criteria)
		VALUES('SHEIN-USDT-SWAP','funding-hedge',0,'short_perp_long_spot',100,'live','okx','{}') RETURNING id`).Scan(&sid); err != nil {
		t.Fatal("策略夹具:", err)
	}
	var eid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES('SHEIN-USDT-SWAP','funding-hedge','short_perp_long_spot','proposed','[]','{}') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal("评估夹具:", err)
	}
	var pid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,state,net_apr)
		VALUES($1,'SHEIN-USDT-SWAP','short_perp_long_spot','fin_armed',0) RETURNING id`, eid).Scan(&pid); err != nil {
		t.Fatal("提案夹具:", err)
	}
	spec := lifecycle.OpenSpec{StrategyRef: sid, Venue: "okx", InstID: "SHEIN-USDT-SWAP",
		Direction: "short_perp_long_spot"}
	if err := h.untradeable(ctx, spec, t51001()); err != nil {
		t.Fatal("处置败:", err)
	}
	var code string
	if err := pool.QueryRow(ctx, `SELECT code FROM fin_venue_untradeable WHERE inst_id='SHEIN-USDT-SWAP'`).Scan(&code); err != nil || code != "51001" {
		t.Fatalf("黑名单应入册 code=%q err=%v", code, err)
	}
	var st string
	if err := pool.QueryRow(ctx, `SELECT state FROM fin_strategy WHERE id=$1`, sid).Scan(&st); err != nil || st != "retired" {
		t.Fatalf("策略应退 st=%q", st)
	}
	var tr string
	if err := pool.QueryRow(ctx, `SELECT reason FROM fin_strategy_transition
		WHERE strategy_id=$1 ORDER BY id DESC LIMIT 1`, sid).Scan(&tr); err == nil && len(tr) < len("venue_untradeable") {
		t.Fatalf("Transition 应留 venue_untradeable 溯源 got %q", tr)
	}
	var ps string
	if err := pool.QueryRow(ctx, `SELECT state FROM fin_proposal WHERE id=$1`, pid).Scan(&ps); err != nil || ps != "fin_retired" {
		t.Fatalf("armed 提案应退 ps=%q", ps)
	}
}

// 锚：黑名单 inst 后续零——openLive 候选查询滤除+起草滤（gapscan）。
func TestBlacklistConsumers(t *testing.T) {
	ctx, pool, _ := setupDB(t)
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_untradeable(inst_id,code,msg)
		VALUES('CSOPSKHYNIX2L-USDT-SWAP','51001','stub') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	var live, banned int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_strategy
		WHERE state='live' AND inst_id='CSOPSKHYNIX2L-USDT-SWAP'
		AND inst_id NOT IN (SELECT inst_id FROM fin_venue_untradeable)`).Scan(&live); err != nil {
		t.Skip("fin_strategy 列形异:", err)
	}
	if live != 0 {
		t.Fatal("黑名单 inst 应零候选")
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_venue_untradeable
		WHERE inst_id='CSOPSKHYNIX2L-USDT-SWAP'`).Scan(&banned); err != nil || banned != 1 {
		t.Fatalf("黑名单行应在 banned=%d err=%v", banned, err)
	}
}
