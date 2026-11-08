package notify

import (
	"context"
	"strings"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// TestEventPushOffNoSend 开关缺/false→零发（fail-closed 静默锚）。
func TestEventPushOffNoSend(t *testing.T) {
	f := &fakeSender{}
	e := NewEventPush(nil, f, func() string { return "oc_x" }, func() bool { return false })
	_ = e.Scan(context.Background())
	if len(f.sent) != 0 {
		t.Fatalf("off 应零发 got=%v", f.sent)
	}
}

// TestEventPushDedup 同 key 一键一推——重注入不重复。
func TestEventPushDedup(t *testing.T) {
	f := &fakeSender{}
	e := NewEventPush(nil, f, func() string { return "oc_x" }, func() bool { return true })
	ctx := context.Background()
	e.push(ctx, "opened:1", "开仓 BTC-USDT-SWAP")
	e.push(ctx, "opened:1", "开仓 BTC-USDT-SWAP")
	if len(f.sent) != 1 || !strings.Contains(f.sent[0], "[经营事件]") {
		t.Fatalf("去重推=%v", f.sent)
	}
}

// T-0489 锚：经营事件推送带 sim/live 域标记——业主肉眼可分（径A配套钦定：
// 「[经营事件] 推送必带 sim/live 标记」；sim 仓行落 sim_* 域表，推送面双域
// 选读各带域标）。修前红：推送只读 fin 正本且不标域——模拟仓按真仓播报
// （业主 OKX 对账扑空实发）；修后绿。
func TestEventPushDomainTag(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	f := &fakeSender{}
	e := NewEventPush(pool, f, func() string { return "oc_x" }, func() bool { return true })
	var execID, posID int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES('CORE-USDT-SWAP',25,'open') RETURNING id`).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd,opened_at)
		VALUES($1,'CORE-USDT-SWAP',25,now()) RETURNING id`, execID).Scan(&posID); err != nil {
		t.Fatal(err)
	}
	var simExecID, simPosID int64
	if err := pool.QueryRow(ctx, `INSERT INTO sim_execution(inst_id,notional_usd,state)
		VALUES('MUU-USDT-SWAP',25,'open') RETURNING id`).Scan(&simExecID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO sim_position(execution_id,inst_id,notional_usd,opened_at)
		VALUES($1,'MUU-USDT-SWAP',25,now()) RETURNING id`, simExecID).Scan(&simPosID); err != nil {
		t.Fatal(err)
	}
	if err := e.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	var liveMsg, simMsg string
	for _, s := range f.sent {
		if strings.Contains(s, "CORE") {
			liveMsg = s
		}
		if strings.Contains(s, "MUU") {
			simMsg = s
		}
	}
	if liveMsg == "" {
		t.Fatalf("live 仓开仓推送缺: sent=%v", f.sent)
	}
	if simMsg == "" {
		t.Fatalf("sim 仓开仓推送缺（模拟仓按真仓播报病灶对侧=sim 域漏报亦不可）: sent=%v", f.sent)
	}
	if !strings.Contains(liveMsg, "[live]") {
		t.Fatalf("live 推送缺域标记: %q", liveMsg)
	}
	if !strings.Contains(simMsg, "[sim]") {
		t.Fatalf("sim 推送缺域标记（业主肉眼不可分=对账扑空病灶）: %q", simMsg)
	}
}
