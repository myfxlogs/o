package lifecycle

import (
	"testing"
)

// fin_0175 锚：admit_block 规约行=机器闸——命中提案不开户留 fin_proposed;
// 未命中/inactive/未知键规约不拦;funding-hedge 在 execfam 册(对照组
// 走常径开户证闸只拦规约命中)。
func TestEconRuleAdmitBlock(t *testing.T) {
	ctx, pool, cfg := setup(t)
	r := NewRunner(pool, cfg)

	mkprop := func(inst string) int64 {
		t.Helper()
		var eid, pid int64
		if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
			VALUES($1,'funding-hedge','short_perp_long_spot',0.08,'proposed','{}','{}') RETURNING id`, inst).Scan(&eid); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
			VALUES($1,$2,'short_perp_long_spot',0.11) RETURNING id`, eid, inst).Scan(&pid); err != nil {
			t.Fatal(err)
		}
		return pid
	}
	blockedPID := mkprop("RULE-BLOCKED-USDT-SWAP")
	freePID := mkprop("RULE-FREE-USDT-SWAP")

	// 规约:funding-hedge×RULE-BLOCKED 拦;外加未知键规约(fail-closed 不拦)
	if _, err := pool.Exec(ctx, `INSERT INTO fin_econ_rule(kind,match,source)
		VALUES('admit_block','{"template_ref":"funding-hedge","inst_id":"RULE-BLOCKED-USDT-SWAP"}','canon:test')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_econ_rule(kind,match,source)
		VALUES('admit_block','{"venue":"mars","inst_id":"RULE-FREE-USDT-SWAP"}','canon:badkey')`); err != nil {
		t.Fatal(err)
	}

	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_strategy
		WHERE inst_id='RULE-BLOCKED-USDT-SWAP'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("规约命中不应建策略行 n=%d err=%v", n, err)
	}
	var st string
	if err := pool.QueryRow(ctx, `SELECT state FROM fin_proposal WHERE id=$1`, blockedPID).
		Scan(&st); err != nil || st != "fin_proposed" {
		t.Fatalf("命中提案应留 fin_proposed: %s %v", st, err)
	}
	if err := pool.QueryRow(ctx, `SELECT state FROM fin_proposal WHERE id=$1`, freePID).
		Scan(&st); err != nil || st != "fin_armed" {
		t.Fatalf("未知键规约不许拦对照组: %s %v", st, err)
	}

	// 撤规→复进
	if _, err := pool.Exec(ctx, `UPDATE fin_econ_rule SET active=false
		WHERE match->>'inst_id'='RULE-BLOCKED-USDT-SWAP'`); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT state FROM fin_proposal WHERE id=$1`, blockedPID).
		Scan(&st); err != nil || st != "fin_armed" {
		t.Fatalf("撤规后应开户 armed: %s %v", st, err)
	}
}
