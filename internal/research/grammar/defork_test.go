// defork_test.go —— T-0216 返修③锚：提案去重语叉根修——同 inst 双 basis 族
// （quarter_basis_long/perp_quarter_spread）direction 去分叉后双族提案并存：
// 原 (inst_id,direction='basis_quarter') 共键下 md_1054 闸 RETURN NULL 静默吞
// 第二族（+run.go 预查 inst 单键同病）——修后两族各自 direction 各自提案。
package grammar

import (
	"testing"
)

// 锚：同 inst 双族评估→双提案并存（direction 分叉消解，闸键不撞）。
func TestBasisDeforkDualFamilyProposals(t *testing.T) {
	ctx, r := setup(t)
	for _, q := range []string{
		"DELETE FROM fin_proposal WHERE inst_id='BTC-USD-261225'",
		"DELETE FROM fin_opp_eval WHERE inst_id='BTC-USD-261225'",
		"DELETE FROM md_mark_index WHERE inst_id='BTC-USD-261225'",
		"DELETE FROM md_ticks WHERE inst_id='BTC-USD'"} {
		if _, err := r.pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	// contango 形：fut 溢价 10%，年化 |netApr|=0.1/57×365≈0.64>0.05 → proposed
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_mark_index(ts,venue,inst_id,channel,px,recv_ts)
		VALUES(now(),'okx','BTC-USD-261225','mark-price',110000,now())`); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_ticks(ts,inst_id,last,bid,ask)
		VALUES(now(),'BTC-USD',100000,99990,100010)`); err != nil {
		t.Fatal(err)
	}
	if err := r.evalBasis(ctx, Family{Template: "quarter_basis_long",
		Direction: "basis_quarter", Kind: "basis"}); err != nil {
		t.Fatal(err)
	}
	if err := r.evalBasis(ctx, Family{Template: "perp_quarter_spread",
		Direction: "perp_quarter_spread", Kind: "basis"}); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal p
		JOIN fin_opp_eval e ON e.id=p.eval_id
		WHERE p.inst_id='BTC-USD-261225' AND e.verdict='proposed'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("同 inst 双族应双提案并存（去分叉前第二族被闸吞 n=1）: n=%d", n)
	}
	var dirs int
	if err := r.pool.QueryRow(ctx, `SELECT count(DISTINCT p.direction) FROM fin_proposal p
		WHERE p.inst_id='BTC-USD-261225'`).Scan(&dirs); err != nil {
		t.Fatal(err)
	}
	if dirs != 2 {
		t.Fatalf("双族 direction 应分叉各异: %d", dirs)
	}
	// 每族提案 direction 与其 eval 模板配对（quarter_basis_long→basis_quarter；
	// perp_quarter_spread→perp_quarter_spread——词表锚）。
	for _, c := range [][2]string{{"quarter_basis_long", "basis_quarter"},
		{"perp_quarter_spread", "perp_quarter_spread"}} {
		var d string
		if err := r.pool.QueryRow(ctx, `SELECT p.direction FROM fin_proposal p
			JOIN fin_opp_eval e ON e.id=p.eval_id
			WHERE p.inst_id='BTC-USD-261225' AND e.template=$1`, c[0]).Scan(&d); err != nil {
			t.Fatalf("族 %s 提案缺席: %v", c[0], err)
		}
		if d != c[1] {
			t.Fatalf("族 %s direction 望 %s got %s", c[0], c[1], d)
		}
	}
}
