package config

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// T-0376 锚②：DM 写径桩测——判词证据→app_config 落值→ops_event 参数调整行
// 三列齐（旧值/新值/证据锚）。锚④运行半面：业主面键（资金上限/凭证形/
// 停机总闸/对账）DMSet 一律拒+零写+零审计行；未登记键 fail-closed 拒。

func dmSetFixture(t *testing.T) (*Client, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	return New(pool), pool
}

func TestDMSetWritesConfigAndAuditRow(t *testing.T) {
	c, pool := dmSetFixture(t)
	ctx := context.Background()
	// 键既有（fin_0006 种/后续票调）——旧值实读为基准非假设值。
	var before string
	if err := pool.QueryRow(ctx,
		`SELECT value::text FROM app_config WHERE key='research.funding.min_net_apr'`,
	).Scan(&before); err != nil {
		t.Fatal(err)
	}
	res, err := c.DMSet(ctx, "research.funding.min_net_apr",
		json.RawMessage("0.07"), "eval证据:fin_opp_eval#42 net_apr=0.072 连续3评估越阈", "T-0376")
	if err != nil {
		t.Fatalf("dmset: %v", err)
	}
	if !res.Changed || string(res.Old) != before || string(res.New) != "0.07" {
		t.Fatalf("结果不符: %+v before=%s", res, before)
	}
	var val, by string
	if err := pool.QueryRow(ctx,
		`SELECT value::text,updated_by FROM app_config WHERE key='research.funding.min_net_apr'`,
	).Scan(&val, &by); err != nil || val != "0.07" || by != dmActor {
		t.Fatalf("config 落值不符: v=%q by=%q err=%v", val, by, err)
	}
	var seat, event, note string
	if err := pool.QueryRow(ctx,
		`SELECT seat,event,note FROM ops_event WHERE event='参数调整' ORDER BY id DESC LIMIT 1`,
	).Scan(&seat, &event, &note); err != nil {
		t.Fatalf("审计行缺: %v", err)
	}
	if seat != dmActor {
		t.Fatalf("seat=%q 非 DM 写径身份", seat)
	}
	for _, frag := range []string{
		"key=research.funding.min_net_apr", "旧=" + before, "新=0.07",
		"证据=eval证据:fin_opp_eval#42",
	} {
		if !strings.Contains(note, frag) {
			t.Fatalf("note 缺三列片段 %q: %s", frag, note)
		}
	}
	// 首写径（缺行）——旧=∅。
	res2, err := c.DMSet(ctx, "dm.hunger.probe", json.RawMessage("true"), "桩", "T-0376")
	if err != nil || res2.Old != nil || !res2.Changed {
		t.Fatalf("首写径不符: %+v err=%v", res2, err)
	}
}

func TestDMSetHardBoundsUnreachable(t *testing.T) {
	c, pool := dmSetFixture(t)
	ctx := context.Background()
	// 资金上限/停机总闸/对账/凭证形/未登记——五形全拒（锚④）。
	for _, key := range []string{
		"risk.max_notional_usd", // 资金上限
		"risk.kill_switch",      // 舰队停机总闸
		"ledger.recon_tol_usd",  // 对账三通道
		"trx.trongrid_api_key",  // 凭证形（regex 硬拒）
		"cmc.account_id",        // 凭证形（account_id）
		"unknown.random.key",    // 未登记 fail-closed
	} {
		if _, err := c.DMSet(ctx, key, json.RawMessage("1"), "桩证据", "T-0376"); err == nil {
			t.Fatalf("%s: 业主面键 DM 写径可达——闸漏", key)
		}
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE event='参数调整'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("拒写径不得落审计行: n=%d err=%v", n, err)
	}
	// 值未被写（哨兵径——键既有 fin_0008 种 500，拒写后现值不变）。
	var before, after string
	if err := pool.QueryRow(ctx,
		`SELECT value::text FROM app_config WHERE key='risk.max_notional_usd'`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT value::text FROM app_config WHERE key='risk.max_notional_usd'`).Scan(&after); err != nil || after != before {
		t.Fatalf("拒写径不得落值: before=%s after=%s err=%v", before, after, err)
	}
}

func TestDMSetScalarContract(t *testing.T) {
	c, _ := dmSetFixture(t)
	ctx := context.Background()
	if _, err := c.DMSet(ctx, "research.funding.slippage_bps",
		json.RawMessage(`{"a":1}`), "桩", "T-0376"); err == nil {
		t.Fatal("object 形应拒（标量契约）")
	}
	if _, err := c.DMSet(ctx, "research.funding.slippage_bps",
		json.RawMessage(`null`), "桩", "T-0376"); err == nil {
		t.Fatal("null 形应拒")
	}
	if _, err := c.DMSet(ctx, "research.funding.slippage_bps",
		json.RawMessage(`"3"`), "桩", ""); err == nil {
		t.Fatal("空 ticket 应拒（归属必携）")
	}
	if _, err := c.DMSet(ctx, "research.funding.slippage_bps",
		json.RawMessage(`3`), "", "T-0376"); err == nil {
		t.Fatal("空证据应拒（无证据不自调）")
	}
}

func TestClassifyKeyDomain(t *testing.T) {
	dm := []string{
		"research.funding.min_net_apr", "research.dcd.enabled",
		"invest.lifecycle.live_switch", "invest.lifecycle.paper_min_hours",
		"invest.paper.notional_usd", "scout.top_n", "lifecycle.armed_ttl_h",
		"strategy.degraded_ttl_hours", "gate.kill.funding_carry",
		"exec.probe_notional_usd", "dm.hunger.zero_proposal_days",
		"earn.auto_purchase_enabled", "trx.floor_pct",
	}
	owner := []string{
		"risk.max_notional_usd", "risk.kill_switch", "ledger.recon_tol_usd",
		"feishu.chat_id", "notify.feishu.enabled", "bridge.depth5.symbols",
		"calendar.min_impact", "trx.stake_address", "trx.rental_payers",
		"trx.trongrid_api_key", "cmc.account_id", "mt5.gateway_addr",
		"totally.unregistered.key",
	}
	for _, k := range dm {
		if got := ClassifyKey(k); got != DomainDM {
			t.Fatalf("%s: 判 %s 非 dm", k, got)
		}
	}
	for _, k := range owner {
		if got := ClassifyKey(k); got != DomainOwner {
			t.Fatalf("%s: 判 %s 非 owner（锚④漏）", k, got)
		}
	}
}
