// audit_1111_riskguard_test.go —— 审计/风控哨戒 RED 锚（audit/1111-riskguard）：
// 断言正确语义——修复落库后自然转绿，门转正为回归锚。
// 显跑：AUDIT_RED=1 go test ./internal/risk ./internal/guard
package risk

import (
	"os"
	"testing"
)

func auditRedGate(t *testing.T) {
	t.Helper()
	if os.Getenv("AUDIT_RED") != "1" {
		t.Skip("audit RED 锚——修复期显跑: AUDIT_RED=1 go test -run " + t.Name() + " ./internal/risk；修好后去门转正为回归锚")
	}
}

// ① 在持仓敞口双计：fin_execution state='open' 无在途/在持分态——在持仓的
// 开仓 exec 行持仓期恒 'open'（markClosed 才迁 closed，产线实证 state 分布
// open=2=在仓数），vExp/tExp 把同仓按 position+exec 计两次。T-0136 注释钦定
// 「在途 intent（已发令牌未成交执行行）」与实现相悖——venue 集中度/总敞口
// 实阈打半折=合法开仓被 venue_concentration/exposure_cap 误拒（armed 死锁面）。
// 正确语义=在途 intent 与在持仓不双计（exec 在持分态或 position 谓词互斥）。
func TestAuditExposureDoubleCountsEstablished(t *testing.T) {
	auditRedGate(t)
	ctx, g := setup(t)
	// equity=1000，max_venue_pct=0.35→venue 额 350；在持仓名义 100（exec 在持
	// 'open' 同额同 venue）→本意 vExp=100；意图 240：100+240=340≤350 当放。
	// 实计 vExp=100(position)+100(exec)=200→200+240=440>350 误拒。
	if _, err := g.pool.Exec(ctx,
		"UPDATE app_config SET value=to_jsonb('0.35'::text) WHERE key='risk.max_venue_pct'"); err != nil {
		t.Fatal(err)
	}
	if err := g.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	var stratID int64
	if err := g.pool.QueryRow(ctx,
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue)
		 VALUES('RK-USDT-SWAP','RK-audit','live',100,'okx') RETURNING id`).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	var execID int64
	if err := g.pool.QueryRow(ctx,
		`INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state)
		 VALUES($1,'RK-USDT-SWAP',100,'open') RETURNING id`, stratID).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	if _, err := g.pool.Exec(ctx,
		`INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,opened_at)
		 VALUES($1,$2,'RK-USDT-SWAP',100,now())`, execID, stratID); err != nil {
		t.Fatal(err)
	}
	in := Intent{StrategyRef: stratID, Venue: "okx", InstID: "RK-USDT-SWAP",
		Legs: []string{"spot_long", "perp_short"}, NotionalUSD: 240, Leverage: 1}
	if d := g.Check(ctx, in); !d.Allow {
		t.Fatalf("在持仓被 position+exec 双计——venue_concentration 误拒: reason=%s（本意 vExp=100→340≤350 当放；实计 200→440 拒）", d.Reason)
	}
}

// ② 令牌 intent 篡改拒因误标：ConsumeToken 对「令牌在册未过期未用但 intent
// 哈希不符」返 token_missing——与「令牌不存在」同词，窜改意图与缺件同判词，
// 分诊面无法区分（拒因语义黑洞）。正确语义=intent 不符有专属拒因词（注册表
// 扩键），与 token_missing 分流。
func TestAuditTokenHashMismatchReason(t *testing.T) {
	auditRedGate(t)
	ctx, g := setup(t)
	d := g.Check(ctx, intent)
	if !d.Allow {
		t.Fatal("allow 未产令牌")
	}
	mut := intent
	mut.NotionalUSD = 250 // 同令牌异意图=哈希不符径
	x := g.ConsumeToken(ctx, d.TokenID, mut)
	if x.Allow {
		t.Fatal("异意图令牌不应放行")
	}
	if x.Reason == "token_missing" {
		t.Fatalf("intent 哈希不符被判 token_missing——窜改与缺件同判词（拒因语义黑洞，分诊歧指）")
	}
}
