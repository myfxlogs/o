package config

import (
	"context"
	"encoding/json"
	"testing"
)

// 值域闸锚（工单#25801/破口 B4）：高危键错形/越界写径拒——读器双形
// （原生∨串编码）镜像收；未列键开放域透传。

func TestCheckValueDomain(t *testing.T) {
	cases := []struct {
		name string
		key  string
		val  string
		ok   bool
	}{
		// 布尔域——停机/闸类双形收、异形拒
		{"kill-bool", "gate.kill.dirmom", `true`, true},
		{"kill-str", "gate.kill.dirmom", `"false"`, true}, // gatefx 存形=jsonb 串
		{"kill-badstr", "gate.kill.dirmom", `"yes"`, false},
		{"kill-num", "gate.kill.dirmom", `1`, false},
		{"enabled-str", "notify.feishu.events.enabled", `"false"`, true}, // 在册实证形
		{"enabled-bool", "earn.auto_purchase_enabled", `true`, true},     // earn_face 写形
		{"live-switch", "invest.lifecycle.live_switch", `"false"`, true}, // 在册串形实证
		{"halted", "earn.residual_clear_halted", `false`, true},
		{"xvenue", "exec.xvenue_open", `true`, true},
		{"ws-disabled", "cmc.ws_disabled", `true`, true},
		// 分数域——名 pct 实 frac 在案
		{"frac-ok", "risk.dir.funding_pct", `0.2`, true},
		{"frac-over", "risk.dir.funding_pct", `20`, false},
		{"frac-str", "lifecycle.dir.stop_frac", `"0.10"`, true},
		{"frac-neg", "lifecycle.dir.take_frac", `-0.1`, false},
		// 正数/非负/百分域
		{"notional", "risk.max_notional_usd", `5000`, true},
		{"notional-zero", "risk.max_notional_usd", `0`, false},
		{"notional-str", "risk.max_notional_usd", `"5000"`, true},
		{"age-zero", "guard.funding_max_age_min", `0`, true}, // 0=回缺省约定在案
		{"age-neg", "guard.funding_max_age_min", `-5`, false},
		{"floor-pct", "trx.floor_pct", `80`, true},
		{"floor-pct-over", "trx.floor_pct", `120`, false},
		{"death-factor", "lifecycle.death_factor", `0.85`, true},
		{"death-factor-neg", "lifecycle.death_factor", `-0.5`, false},
		{"timeout", "funding.scan_timeout_s", `30`, true},
		{"venue-pct", "risk.max_venue_pct", `0.35`, true},
		{"venue-pct-over", "risk.max_venue_pct", `35`, false},
		{"trx-vote-pct", "trx.vote_yield_drop_pct", `1.0`, true},
		{"probe-notional", "exec.probe_notional_usd", `"50"`, true},
		{"paper-min-usd", "invest.lifecycle.paper_min_net_usd", `"0.01"`, true},
		{"hedge-bps", "exec.hedge_band_bps", `20`, true},
		{"bps-neg", "guard.flip_bps", `"-5"`, false},
		{"cooldown-str", "exec.reopen_cooldown_sec", `300`, true},
		{"max-hold-str", "guard.max_hold_hours", `"72"`, true},
		// 开放域——未列键透传（validScalar 已兜标量守）
		{"open-num", "exec.hedge_band_bps", `20`, true},
		{"open-str", "lifecycle.friction_at.okx", `1730000000`, true},
		{"open-any", "research.funding.min_net_apr", `0.07`, true},
		{"open-gate", "gate.sizing.0.5x", `"0.5"`, true},
		{"open-gate-pct", "gate.lesson.cost_diverge_pct", `20`, true}, // 在册 20——pct 非全分数实锤
	}
	for _, tc := range cases {
		err := CheckValueDomain(tc.key, json.RawMessage(tc.val))
		if (err == nil) != tc.ok {
			t.Errorf("%s: key=%s val=%s ok=%v err=%v", tc.name, tc.key, tc.val, tc.ok, err)
		}
	}
}

// DMSet 值域拒写锚——闸拒=零写零审计行（fail-closed 同域闸语义）。
func TestDMSetValueDomainRejects(t *testing.T) {
	c, pool := dmSetFixture(t)
	ctx := context.Background()
	if _, err := c.DMSet(ctx, "gate.kill.dirmom",
		json.RawMessage(`"yes"`), "ev", "T-x"); err == nil {
		t.Fatal("布尔域外值未拒")
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM app_config WHERE key='gate.kill.dirmom'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("拒写后竟落行 n=%d err=%v", n, err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE event='参数调整' AND note LIKE '%gate.kill.dirmom%'`).
		Scan(&n); err != nil || n != 0 {
		t.Fatalf("拒写后竟落审计行 n=%d err=%v", n, err)
	}
}
