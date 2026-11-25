package lifecycle

// fasttrack_test.go —— O-41014 probe 直通径桩测五案（spec §3 验收锚）：
// ①极值 eval→mint state=probe+notional=阈件；②亚阈→paper 旧径零回归；
// ③并发顶=第 N+1 件照走 paper；④闸 off=全走 paper；
// ⑤入场条件未到=probe armed 不开仓（armed≠open 执行闸在位证）。

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
)

// ftCfg 直通阈件键种+cfg 重载（缺省=enabled 关——各案按需覆写）。
func ftCfg(t *testing.T, ctx context.Context, pool *pgxpool.Pool, cfg *config.Client, on bool) {
	t.Helper()
	en := "false"
	if on {
		en = "true"
	}
	for k, v := range map[string]string{
		"probe.fasttrack_enabled": en, "probe.fasttrack_score": "8.0",
		"probe.fasttrack_notional_usd": "50", "probe.fasttrack_max_open": "3"} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
}

// mkFTProp eval+proposal 对插（dir 族形——eval verdict=proposed 供
// dirEntryGate 信号续存判词；proposal.net_apr=直通强度判词源）。
func mkFTProp(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	inst, tmpl, dir string, evalAPR, propAPR float64) int64 {
	t.Helper()
	var eid, pid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES($1,$2,$3,$4,'proposed','{}','{}') RETURNING id`, inst, tmpl, dir, evalAPR).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		"INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr) VALUES($1,$2,$3,$4) RETURNING id",
		eid, inst, dir, propAPR).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	return pid
}

func ftClean(t *testing.T, ctx context.Context, pool *pgxpool.Pool, inst string) {
	t.Helper()
	for _, q := range []string{
		"DELETE FROM fin_paper_fill WHERE strategy_id IN (SELECT id FROM fin_strategy WHERE inst_id=$1)",
		"DELETE FROM fin_strategy_transition WHERE strategy_id IN (SELECT id FROM fin_strategy WHERE inst_id=$1)",
		"DELETE FROM fin_position WHERE strategy_ref IN (SELECT id FROM fin_strategy WHERE inst_id=$1)",
		"DELETE FROM fin_strategy WHERE inst_id=$1",
		"DELETE FROM fin_proposal WHERE inst_id=$1",
		"DELETE FROM fin_opp_eval WHERE inst_id=$1"} {
		if _, err := pool.Exec(ctx, q, inst); err != nil {
			t.Fatal(err)
		}
	}
}

// 案①：极值 eval（net_apr=10≥8）→mint state=probe+notional=50 阈件+
// criteria.fasttrack=true+transition proposed→probe+proposal fin_armed。
func TestFasttrackProbeMint(t *testing.T) {
	ctx, pool, cfg := setup(t)
	ftClean(t, ctx, pool, "FT1-USDT-SWAP")
	ftCfg(t, ctx, pool, cfg, true)
	r := NewRunner(pool, cfg)
	mkFTProp(t, ctx, pool, "FT1-USDT-SWAP", "dir_perp_mom", "long_perp", 10, 10)
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	var notional float64
	var crit string
	if err := pool.QueryRow(ctx, `SELECT state,notional_usd,criteria::text FROM fin_strategy
		WHERE inst_id='FT1-USDT-SWAP'`).Scan(&state, &notional, &crit); err != nil {
		t.Fatal(err)
	}
	if state != "probe" || notional != 50 {
		t.Fatalf("直通应 mint probe+阈件 notional=50: state=%s notional=%v", state, notional)
	}
	if !strings.Contains(crit, `"fasttrack":true`) && !strings.Contains(crit, `"fasttrack": true`) {
		t.Fatalf("criteria 缺 fasttrack 旗标: %s", crit)
	}
	var toState, pstate string
	if err := pool.QueryRow(ctx, `SELECT to_state FROM fin_strategy_transition st
		JOIN fin_strategy s ON s.id=st.strategy_id WHERE s.inst_id='FT1-USDT-SWAP'
		ORDER BY st.id DESC LIMIT 1`).Scan(&toState); err != nil || toState != "probe" {
		t.Fatalf("transition 应 proposed→probe: to=%s err=%v", toState, err)
	}
	if err := pool.QueryRow(ctx, "SELECT state FROM fin_proposal WHERE inst_id='FT1-USDT-SWAP'").Scan(&pstate); err != nil || pstate != "fin_armed" {
		t.Fatalf("proposal 应 fin_armed: %s err=%v", pstate, err)
	}
}

// 案②：亚阈 eval（net_apr=5<8）→mint paper+paper notional=100+无旗标
// （旧径零回归——慢信号照走慢证明）。
func TestFasttrackSubThresholdPaper(t *testing.T) {
	ctx, pool, cfg := setup(t)
	ftClean(t, ctx, pool, "FT2-USDT-SWAP")
	ftCfg(t, ctx, pool, cfg, true)
	r := NewRunner(pool, cfg)
	mkFTProp(t, ctx, pool, "FT2-USDT-SWAP", "dir_perp_mom", "long_perp", 5, 5)
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	var notional float64
	var crit string
	if err := pool.QueryRow(ctx, `SELECT state,notional_usd,criteria::text FROM fin_strategy
		WHERE inst_id='FT2-USDT-SWAP'`).Scan(&state, &notional, &crit); err != nil {
		t.Fatal(err)
	}
	if state != "paper" || notional != 100 {
		t.Fatalf("亚阈应走 paper+paper notional=100: state=%s notional=%v", state, notional)
	}
	if strings.Contains(crit, "fasttrack") {
		t.Fatalf("paper 径不应沾 fasttrack 旗标: %s", crit)
	}
}

// 案③：并发顶——在册 fasttrack probe=1=max_open→新极值件照走 paper。
func TestFasttrackMaxOpen(t *testing.T) {
	ctx, pool, cfg := setup(t)
	ftClean(t, ctx, pool, "FT3-USDT-SWAP")
	ftClean(t, ctx, pool, "FT3B-USDT-SWAP")
	ftCfg(t, ctx, pool, cfg, true)
	if _, err := pool.Exec(ctx, `UPDATE app_config SET value='"1"' WHERE key='probe.fasttrack_max_open'`); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Load(ctx); err != nil { // cfg 快照须重载——max_open=1 生效
		t.Fatal(err)
	}
	// 在册直通 probe 一行（criteria.fasttrack=true——并发闸基线真源）。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,criteria,direction,venue,entered_at)
		VALUES('FT3B-USDT-SWAP','dir_perp_mom','probe',50,'{"fasttrack":true,"min_hours":0,"min_net":0,"max_dd":5}'::jsonb,'long_perp','okx',now())`); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, cfg)
	mkFTProp(t, ctx, pool, "FT3-USDT-SWAP", "dir_perp_mom", "long_perp", 10, 10)
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := pool.QueryRow(ctx, "SELECT state FROM fin_strategy WHERE inst_id='FT3-USDT-SWAP'").Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "paper" {
		t.Fatalf("并发顶满新极值件应照走 paper: state=%s", state)
	}
}

// 案④：闸 off（enabled=false）→极值 eval 全走 paper（点亮权归部署）。
func TestFasttrackDisabledAllPaper(t *testing.T) {
	ctx, pool, cfg := setup(t)
	ftClean(t, ctx, pool, "FT4-USDT-SWAP")
	ftCfg(t, ctx, pool, cfg, false)
	r := NewRunner(pool, cfg)
	mkFTProp(t, ctx, pool, "FT4-USDT-SWAP", "dir_perp_mom", "long_perp", 10, 10)
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := pool.QueryRow(ctx, "SELECT state FROM fin_strategy WHERE inst_id='FT4-USDT-SWAP'").Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "paper" {
		t.Fatalf("闸 off 极值件应走 paper: state=%s", state)
	}
}

// 案⑤：入场条件未到=probe armed 不开仓——eval 信号陈（>72h）后
// openProbe 延评，r.op.Open 零调用、fin_position 零行、策略留 probe。
func TestFasttrackProbeArmedNoOpen(t *testing.T) {
	ctx, pool, cfg := setup(t)
	ftClean(t, ctx, pool, "FT5-USDT-SWAP")
	ftCfg(t, ctx, pool, cfg, true)
	fo := &fakeOpener{}
	r := NewRunner(pool, cfg).AttachOpener(fo)
	mkFTProp(t, ctx, pool, "FT5-USDT-SWAP", "dir_perp_mom", "long_perp", 10, 10)
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var sid int64
	var state string
	if err := pool.QueryRow(ctx, "SELECT id,state FROM fin_strategy WHERE inst_id='FT5-USDT-SWAP'").Scan(&sid, &state); err != nil {
		t.Fatal(err)
	}
	if state != "probe" {
		t.Fatalf("案⑤前置=直通 mint probe: state=%s", state)
	}
	// 信号陈——dirEntryGate「评估行陈>72h」延评（armed≠open 脊柱证）。
	if _, err := pool.Exec(ctx, `UPDATE fin_opp_eval SET ts=now()-interval '73 hours'
		WHERE inst_id='FT5-USDT-SWAP'`); err != nil {
		t.Fatal(err)
	}
	deferN := int(r.rej.Snapshot()["lifecycle/entry_cond_defer"])
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fo.calls) != 0 {
		t.Fatalf("入场条件未到 op.Open 应零调用: calls=%d", len(fo.calls))
	}
	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM fin_position WHERE strategy_ref=$1", sid).Scan(&n); err != nil || n != 0 {
		t.Fatalf("armed 不开仓 fin_position 应零行: n=%d err=%v", n, err)
	}
	if err := pool.QueryRow(ctx, "SELECT state FROM fin_strategy WHERE id=$1", sid).Scan(&state); err != nil || state != "probe" {
		t.Fatalf("延评不死——策略应留 probe: state=%s err=%v", state, err)
	}
	if got := int(r.rej.Snapshot()["lifecycle/entry_cond_defer"]) - deferN; got < 1 {
		t.Fatalf("entry_cond_defer 应计延评: delta=%d", got)
	}
}
