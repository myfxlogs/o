package risk

import (
	"testing"
)

// close 判径闸内契约（T-0015）：kill_switch/各上限不拦 close（敞口只减不增，
// 拦平仓=存风险）、open 照拦；决策行 kind 列留痕区分判径。
func TestCloseIgnoresKillSwitch(t *testing.T) {
	ctx, g := setup(t)
	if _, err := g.pool.Exec(ctx, `UPDATE app_config SET value='"true"' WHERE key='risk.kill_switch'`); err != nil {
		t.Fatal(err)
	}
	if err := g.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if d := g.Check(ctx, intent); d.Allow || d.Reason != "kill_switch" {
		t.Fatalf("open 应被 kill_switch 拦，got %+v", d)
	}
	closeIn := intent
	closeIn.Kind = "close"
	d := g.Check(ctx, closeIn)
	if !d.Allow {
		t.Fatalf("close 应放行（kill_switch 不拦平仓径），got %+v", d)
	}
	var kind string
	if err := g.pool.QueryRow(ctx,
		"SELECT kind FROM fin_risk_decision WHERE id=$1", d.DecisionID).Scan(&kind); err != nil {
		t.Fatal(err)
	}
	if kind != "close" {
		t.Fatalf("决策行 kind=%q，期望 close（留痕区分判径）", kind)
	}
}

func TestCloseIgnoresCaps(t *testing.T) {
	ctx, g := setup(t)
	big := intent
	big.NotionalUSD = 999999
	if d := g.Check(ctx, big); d.Allow || d.Reason != "notional_cap" {
		t.Fatalf("open 超名义上限应拦，got %+v", d)
	}
	big.Kind = "close"
	if d := g.Check(ctx, big); !d.Allow {
		t.Fatalf("close 不受名义上限拦（敞口只减不增），got %+v", d)
	}
}

// close 径 fail-closed 底线仍在：ttl 键缺=deny（免拦的只有 kill_switch/上限，不是免检）。
func TestCloseMissingTTLFailClosed(t *testing.T) {
	ctx, g := setup(t)
	if _, err := g.pool.Exec(ctx, `DELETE FROM app_config WHERE key='risk.token_ttl_seconds'`); err != nil {
		t.Fatal(err)
	}
	if err := g.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	closeIn := intent
	closeIn.Kind = "close"
	if d := g.Check(ctx, closeIn); d.Allow || d.Reason != "missing_param" {
		t.Fatalf("close 缺 ttl 键应 fail-closed deny，got %+v", d)
	}
}
