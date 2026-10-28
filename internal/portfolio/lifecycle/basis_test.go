// basis_test.go —— T-0216 返修①锚：basisPaperAccounting 记账面补装——
// 双价齐鲜→基差溢价线性衰减摊提 fill 真生成（detail 与价源行同源对拍）；
// 现货腿缺/陈=fail-closed 让位不造零（T-0171 律同 dcd）；perp_quarter_spread
// 加计 perp 腿 funding 负号记账（long_perp 付正租）且与溢价摊提按 inst_id
// 分键互不污染双方 last-ts 基准。basisSpotLeg 归一链纯件锚。
package lifecycle

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func basisSetup(t *testing.T) (context.Context, *Runner, *pgxpool.Pool) {
	ctx, pool, cfg := setup(t)
	for _, q := range []string{
		"DELETE FROM fin_paper_fill", "DELETE FROM fin_strategy_transition",
		"DELETE FROM fin_strategy", "DELETE FROM fin_proposal WHERE inst_id LIKE 'BTC-USD-26%'",
		"DELETE FROM md_mark_index WHERE inst_id='BTC-USD-261225'",
		"DELETE FROM md_ticks WHERE inst_id='BTC-USDT'",
		"DELETE FROM md_funding WHERE inst_id='BTC-USD-SWAP'"} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	return ctx, NewRunner(pool, cfg), pool
}

// basisStrategy 造 paper 策略行（entered_at 回 2 天触发日摊节拍）。
func basisStrategy(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	tmpl, inst, direction string, notional float64) int64 {
	var eid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES($1,$2,$3,'proposed','{}','{}') RETURNING id`, inst, tmpl, direction).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	var pid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
		VALUES($1,$2,$3,0.1) RETURNING id`, eid, inst, direction).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	var sid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,state,notional_usd,direction,entered_at)
		VALUES($1,$2,$3,'paper',$4,$5,now()-interval '2 days') RETURNING id`,
		inst, tmpl, pid, notional, direction).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	return sid
}

// basisMarks 期货 mark-price+现货 tick 双价行（contango 形：fut 溢价 10%）。
func basisMarks(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	if _, err := pool.Exec(ctx, `INSERT INTO md_mark_index(ts,venue,inst_id,channel,px,recv_ts)
		VALUES(now(),'okx','BTC-USD-261225','mark-price',110000,now())`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,inst_id,last,bid,ask)
		VALUES(now(),'BTC-USDT',100000,99990,100010)`); err != nil {
		t.Fatal(err)
	}
}

// 锚①：双价齐鲜→溢价摊提 fill 真生成；detail 与价源行同源（spot=100000/fut=110000）；
// contango p=0.1>0 → amt 正收入且不超全剩余溢价（notional×p=1000×0.1=100 上界）。
func TestBasisPaperFillConvergence(t *testing.T) {
	ctx, r, pool := basisSetup(t)
	basisStrategy(t, ctx, pool, "quarter_basis_long", "BTC-USD-261225", "basis_quarter", 1000)
	basisMarks(t, ctx, pool)
	if err := r.basisPaperAccounting(ctx); err != nil {
		t.Fatal(err)
	}
	var amt float64
	var detail map[string]any
	if err := pool.QueryRow(ctx, `SELECT amount_usd, detail FROM fin_paper_fill
		WHERE inst_id='BTC-USD-261225' AND kind='funding' ORDER BY id DESC LIMIT 1`).Scan(&amt, &detail); err != nil {
		t.Fatal("双价齐鲜应真生成溢价摊提 fill:", err)
	}
	if amt <= 0 || amt >= 100 {
		t.Fatalf("contango 摊提应正且夹在 (0,notional×p) 界内: %v", amt)
	}
	if detail["spot"] != 100000.0 || detail["fut"] != 110000.0 {
		t.Fatalf("fill detail 应与价源行同源: %v", detail)
	}
	// 现货腿价陈（>120s 闸）→后续拍让位不新增
	if _, err := pool.Exec(ctx, `UPDATE md_ticks SET ts=now()-interval '10 minutes' WHERE inst_id='BTC-USDT'`); err != nil {
		t.Fatal(err)
	}
	if err := r.basisPaperAccounting(ctx); err != nil {
		t.Fatal(err)
	}
	var cnt int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_paper_fill WHERE inst_id='BTC-USD-261225'`).Scan(&cnt); err != nil {
		t.Fatal(err)
	}
	if cnt != 1 {
		t.Fatalf("现货腿陈应 fail-closed 让位（fill 恒 1 行）: cnt=%d", cnt)
	}
}

// 锚②：现货腿缺（md_ticks 无行）=fail-closed 让位不造零 fill（T-0171 律）。
func TestBasisPaperFillFailClosedSpotMissing(t *testing.T) {
	ctx, r, pool := basisSetup(t)
	basisStrategy(t, ctx, pool, "quarter_basis_long", "BTC-USD-261225", "basis_quarter", 1000)
	if _, err := pool.Exec(ctx, `INSERT INTO md_mark_index(ts,venue,inst_id,channel,px,recv_ts)
		VALUES(now(),'okx','BTC-USD-261225','mark-price',110000,now())`); err != nil {
		t.Fatal(err)
	}
	if err := r.basisPaperAccounting(ctx); err != nil {
		t.Fatal(err)
	}
	var cnt int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_paper_fill`).Scan(&cnt); err != nil {
		t.Fatal(err)
	}
	if cnt != 0 {
		t.Fatalf("现货腿缺应让位不造 fill（cnt=%d）", cnt)
	}
}

// 锚③：perp_quarter_spread 加计 perp 腿 funding——long_perp 付正租=负号记账
// （inst_id=perp 键与溢价摊提分键）；溢价摊提 fill 仍独立落行。
func TestBasisPerpFundingNegativeSign(t *testing.T) {
	ctx, r, pool := basisSetup(t)
	basisStrategy(t, ctx, pool, "perp_quarter_spread", "BTC-USD-261225", "perp_quarter_spread", 1000)
	basisMarks(t, ctx, pool)
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate,recv_ts)
		VALUES(now()-interval '8 hours','BTC-USD-SWAP',0.0001,now()-interval '8 hours')`); err != nil {
		t.Fatal(err)
	}
	if err := r.basisPaperAccounting(ctx); err != nil {
		t.Fatal(err)
	}
	var amt float64
	var detail map[string]any
	if err := pool.QueryRow(ctx, `SELECT amount_usd, detail FROM fin_paper_fill
		WHERE inst_id='BTC-USD-SWAP' AND kind='funding' ORDER BY id DESC LIMIT 1`).Scan(&amt, &detail); err != nil {
		t.Fatal("perp 腿 funding fill 应落行:", err)
	}
	if amt != -0.1 { // −rate×notional = −0.0001×1000
		t.Fatalf("long_perp 付正租应负号记账 −0.1: %v", amt)
	}
	if detail["leg"] != "perp_long" {
		t.Fatalf("detail leg 应=perp_long: %v", detail)
	}
	var conv int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_paper_fill
		WHERE inst_id='BTC-USD-261225' AND kind='funding'`).Scan(&conv); err != nil {
		t.Fatal(err)
	}
	if conv != 1 {
		t.Fatalf("溢价摊提 fill 应独立落行 1 笔: %d", conv)
	}
}

// 锚④：basisSpotLeg 归一链纯件——币本位/USDT 本位/指数键三形+非期货键空让位。
func TestBasisSpotLeg(t *testing.T) {
	for _, c := range [][2]string{
		{"BTC-USD-261225", "BTC-USDT"},
		{"BTC-USDT-261225", "BTC-USDT"},
		{"BTC-USDG-261225", "BTC-USDT"},
		{"ETH-USDT-270326", "ETH-USDT"},
		{"BTC-USDT-SWAP", ""}, // 非到期日期货键=空让位
	} {
		if got := basisSpotLeg(c[0]); got != c[1] {
			t.Fatalf("basisSpotLeg(%s)=%s 望 %s", c[0], got, c[1])
		}
	}
}
