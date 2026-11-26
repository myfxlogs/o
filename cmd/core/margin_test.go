package main

// margin_test.go —— O-40707 桩测四面：
// ①采集 upsert 幂等 ②息腿簿记（borrow_interest 负腿+accrued 推进）
// ③marginRatio 三档（warn 升件/kill flatten→repay 序）④exec_enabled
// 缺省 false=执行闸 fail-closed + 开仓全序桩（赎→借→确认→卖→簿记）。

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/portfolio/lifecycle"
	"o/internal/risk"
	"o/internal/trading/okxexec"
)

type fakeResolveResp struct {
	st            okxexec.OrderState
	avgPx, fillSz float64
	err           error
}

type fakeMarginVenue struct {
	calls       []string
	rateCcyArg  []string // InterestRates 入参留证（O-41363 无参全量锚）
	rates       []okxexec.MarginRate
	maxLoan     float64
	maxLoanErr  error
	hist        []okxexec.BorrowRepayHistRow
	histErr     error
	adjEq       float64
	adjErr      error
	placeErr    error
	resolveSt   okxexec.OrderState
	avgPx       float64
	fillSz      float64
	resolveErr  error
	resolveSeq  []fakeResolveResp // FIFO 优先——空回落 resolveSt/avgPx/fillSz/resolveErr
	borrowErr   error
	liab        float64
	liabPresent bool
	liabErr     error
	borrowRepay int
}

func (f *fakeMarginVenue) InterestRates(_ context.Context, ccys []string) ([]okxexec.MarginRate, error) {
	f.calls = append(f.calls, "rates")
	f.rateCcyArg = append([]string(nil), ccys...)
	return f.rates, nil
}
func (f *fakeMarginVenue) MaxLoanCcy(_ context.Context, _, _, _ string) (float64, error) {
	f.calls = append(f.calls, "maxloan")
	return f.maxLoan, f.maxLoanErr
}
func (f *fakeMarginVenue) SpotManualBorrowRepay(_ context.Context, _ string, _ float64, side string) (json.RawMessage, error) {
	f.calls = append(f.calls, "borrowrepay:"+side)
	if side == "repay" {
		f.borrowRepay++
	}
	if f.borrowErr != nil {
		return nil, f.borrowErr
	}
	return json.RawMessage(`{}`), nil
}

// SpotBorrowLiab O-41350④repay 额 venue 负债面桩。
func (f *fakeMarginVenue) SpotBorrowLiab(_ context.Context, _ string) (float64, bool, error) {
	f.calls = append(f.calls, "liab")
	return f.liab, f.liabPresent, f.liabErr
}
func (f *fakeMarginVenue) SpotBorrowRepayHistory(_ context.Context, _, _ string, _ int) ([]okxexec.BorrowRepayHistRow, error) {
	f.calls = append(f.calls, "hist")
	return f.hist, f.histErr
}
func (f *fakeMarginVenue) PlaceOrder(_ context.Context, p okxexec.OrderParams) (json.RawMessage, error) {
	f.calls = append(f.calls, "place:"+p.Side)
	return json.RawMessage(`{}`), f.placeErr
}
func (f *fakeMarginVenue) ResolveOrderDetail(_ context.Context, _, _ string) (okxexec.OrderState, float64, float64, float64, string, error) {
	f.calls = append(f.calls, "resolve")
	if len(f.resolveSeq) > 0 {
		r := f.resolveSeq[0]
		f.resolveSeq = f.resolveSeq[1:]
		return r.st, r.avgPx, r.fillSz, 0, "", r.err
	}
	return f.resolveSt, f.avgPx, f.fillSz, 0, "", f.resolveErr
}
func (f *fakeMarginVenue) AccountAdjEquity(_ context.Context) (float64, error) {
	f.calls = append(f.calls, "adjeq")
	return f.adjEq, f.adjErr
}

func marginCfg(t *testing.T, pool *pgxpool.Pool) *config.Client {
	t.Helper()
	cfg := config.New(pool)
	if err := cfg.Load(context.Background()); err != nil {
		t.Fatal("cfg load:", err)
	}
	return cfg
}

// ①采集：负 funding 币+锚币→md_margin_borrow upsert 幂等（重跑不炸不重复键）。
func TestMarginCollectOnce(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	// md_funding 负率行（采集宇宙源）+锚币 BTC/ETH/USDT 自动并入。
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,venue,inst_id,funding_rate)
		VALUES(now(),'okx','TST-USDT-SWAP',-0.001)`); err != nil {
		t.Fatal("md_funding seed:", err)
	}
	fv := &fakeMarginVenue{
		rates: []okxexec.MarginRate{
			{Ccy: "TST", InterestRate: 0.0001}, {Ccy: "BTC", InterestRate: 0.0000005},
			{Ccy: "ETH", InterestRate: 0.000001}, {Ccy: "USDT", InterestRate: 0},
		},
		maxLoan: 100,
	}
	n, err := marginCollectOnce(ctx, pool, fv)
	if err != nil {
		t.Fatal(err)
	}
	if n < 4 {
		t.Fatalf("采集行数=%d want≥4（TST+三锚）", n)
	}
	// O-41363 锚①：利率步必须无参全量调用——venue 不吃逗号多币，
	// ≥2 币拼接出参=51000 毒批（本断言锁死回归）。
	if len(fv.rateCcyArg) != 0 {
		t.Fatalf("InterestRates 入参=%v——须无参全量（nil/空集）", fv.rateCcyArg)
	}
	var apr, ml float64
	if err := pool.QueryRow(ctx, `SELECT borrow_apr,max_loan FROM md_margin_borrow
		WHERE ccy='TST'`).Scan(&apr, &ml); err != nil {
		t.Fatal("TST 证据行缺:", err)
	}
	if apr != 0.876 || ml != 100 { // 0.0001/h×8760=0.876 年化
		t.Fatalf("apr=%.4f ml=%.1f want 0.876/100", apr, ml)
	}
	// 幂等重跑——upsert 同键不炸。
	if _, err := marginCollectOnce(ctx, pool, fv); err != nil {
		t.Fatal("重跑败:", err)
	}
}

// O-41363 锚②③：宇宙币缺响应=跳过不产证据行；产出行数=宇宙∩响应交集
// （venue 无借贷市的币不虚构证据——闸按 missing 拒的语义不变）。
func TestMarginCollectMissingCcySkips(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,venue,inst_id,funding_rate)
		VALUES(now(),'okx','MISS-USDT-SWAP',-0.001)`); err != nil {
		t.Fatal("md_funding seed:", err)
	}
	fv := &fakeMarginVenue{
		// 响应只含 BTC 锚——本测宇宙=MISS+BTC/ETH/USDT 锚，
		// 响应缺 MISS/ETH/USDT → 只 BTC 产行。
		rates:   []okxexec.MarginRate{{Ccy: "BTC", InterestRate: 0.0000005}},
		maxLoan: 50,
	}
	n, err := marginCollectOnce(ctx, pool, fv)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("交集行数=%d want=1（仅 BTC 在响应∩宇宙）", n)
	}
	// 缺响应币不产证据行（n==1 已证仅 BTC 写——MISS 为本测独占币锚直查）。
	var cnt int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM md_margin_borrow WHERE ccy='MISS'`).Scan(&cnt); err != nil || cnt != 0 {
		t.Fatalf("MISS 缺响应不应产证据行 cnt=%d err=%v", cnt, err)
	}
}

// ②息腿：open loan+新鲜 md_margin_borrow→borrow_interest 负腿入账+
// accrued_interest/last_accrual_at 推进；重拍 elapsed≈0 不重复记。
func TestMarginAccrueInterest(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO md_margin_borrow(ccy,inst_id,mgn_mode,borrow_apr,max_loan,ts)
		VALUES('ACCR','ACCR-USDT','cross',0.5,10,now())`); err != nil {
		t.Fatal(err)
	}
	var loanID int64
	err := pool.QueryRow(ctx, `INSERT INTO fin_margin_loan(venue,ccy,inst_id,amt,apr,status,last_accrual_at)
		VALUES('okx','ACCR','ACCR-USDT',2,0.4,'open',now()-interval '10 hours') RETURNING id`).Scan(&loanID)
	if err != nil {
		t.Fatal("loan seed:", err)
	}
	loans, err := marginOpenLoans(ctx, pool)
	if err != nil || len(loans) == 0 {
		t.Fatalf("open loans=%v err=%v", loans, err)
	}
	if err := marginAccrue(ctx, pool, loans); err != nil {
		t.Fatal(err)
	}
	// 息=2币×0.5/8760×10h≈0.00114 币——负腿入账。
	var amt float64
	var kind string
	if err := pool.QueryRow(ctx, `SELECT kind,amount FROM fin_cashflow
		WHERE source_table='fin_margin_loan' AND source_ref=$1`, loanID).Scan(&kind, &amt); err != nil {
		t.Fatal("borrow_interest 行缺:", err)
	}
	if kind != "borrow_interest" || amt >= 0 {
		t.Fatalf("kind=%s amount=%.6f want borrow_interest 负腿", kind, amt)
	}
	var accrued, freshN float64
	if err := pool.QueryRow(ctx, `SELECT accrued_interest,
		CASE WHEN last_accrual_at>now()-interval '1 minute' THEN 1 ELSE 0 END
		FROM fin_margin_loan WHERE id=$1`, loanID).
		Scan(&accrued, &freshN); err != nil || accrued <= 0 || freshN != 1 {
		t.Fatalf("accrued=%v fresh=%v err=%v", accrued, freshN, err)
	}
	// 重拍——elapsed≈0 零新增（幂等不双计）。
	before := 0
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM fin_cashflow WHERE source_ref=$1`, loanID).Scan(&before)
	loans2, _ := marginOpenLoans(ctx, pool)
	_ = marginAccrue(ctx, pool, loans2)
	var after int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM fin_cashflow WHERE source_ref=$1`, loanID).Scan(&after)
	if after != before {
		t.Fatalf("重拍重复记息 %d→%d", before, after)
	}
	// 清场——open loan 残留会污染 marginOpenLoans 全枚举（共享 schema）。
	if _, err := pool.Exec(ctx, `UPDATE fin_margin_loan SET status='repaid',repaid_at=now() WHERE id=$1`, loanID); err != nil {
		t.Fatal(err)
	}
}

// ③健康闸三档：平静零动作 / warn 升件 / kill flatten→repay 序+终态。
func TestMarginHealthGate(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	cfg := marginCfg(t, pool)
	// 在借仓+现货价行（marginSpotPx 两级——md_ticks.last 源）。
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last)
		VALUES(now(),'okx','HG-USDT',100)`); err != nil {
		t.Fatal(err)
	}
	var loanID int64
	err := pool.QueryRow(ctx, `INSERT INTO fin_margin_loan(venue,ccy,inst_id,amt,apr,status)
		VALUES('okx','HG','HG-USDT',5,0.1,'open') RETURNING id`).Scan(&loanID)
	if err != nil {
		t.Fatal(err)
	}
	// 档一：adjEq 高→ratio<warn 零动作零升件。
	fv := &fakeMarginVenue{adjEq: 100000}
	loans, _ := marginOpenLoans(ctx, pool)
	if err := marginHealthGate(ctx, pool, cfg, fv, loans); err != nil {
		t.Fatal(err)
	}
	var esc int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE esc_class LIKE 'margin_ratio%'`).Scan(&esc)
	if esc != 0 {
		t.Fatalf("平静档误升件=%d", esc)
	}
	// 档二：adjEq 使 ratio∈[warn,kill)——loanUSD=5×100=$500；eq=800→0.625。
	fv2 := &fakeMarginVenue{adjEq: 800, resolveSt: okxexec.StateFilled, fillSz: 5,
		hist: []okxexec.BorrowRepayHistRow{}}
	// repay 历史须有旧行才不会被 10min 窗误判在途——hist 空=发 repay。
	if err := marginHealthGate(ctx, pool, cfg, fv2, loans); err != nil {
		t.Fatal(err)
	}
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE esc_class='margin_ratio_warn'`).Scan(&esc)
	if esc != 1 {
		t.Fatalf("warn 档升件=%d want 1", esc)
	}
	// 档三：eq=500→ratio=1.0≥kill——flatten→repay 序+repaid 终态。
	// O-41350 resolve-first：前序态 absent→发新 flatten→filled→repay。
	fv3 := &fakeMarginVenue{adjEq: 500, resolveSeq: []fakeResolveResp{
		{st: okxexec.StateAbsent}, {st: okxexec.StateFilled, fillSz: 5}}}
	if err := marginHealthGate(ctx, pool, cfg, fv3, loans); err != nil {
		t.Fatal(err)
	}
	// 序钉：place:buy 在 borrowrepay:repay 前；hist 在 repay 前（对账）。
	var buyIdx, repayIdx, histIdx = -1, -1, -1
	for i, c := range fv3.calls {
		switch c {
		case "place:buy":
			buyIdx = i
		case "borrowrepay:repay":
			repayIdx = i
		case "hist":
			if histIdx < 0 {
				histIdx = i
			}
		}
	}
	if !(buyIdx >= 0 && histIdx > buyIdx && repayIdx > histIdx) {
		t.Fatalf("平仓序乱 calls=%v want place:buy→hist→repay", fv3.calls)
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM fin_margin_loan WHERE id=$1`, loanID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "repaid" {
		t.Fatalf("kill 后 status=%s want repaid", status)
	}
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE esc_class='margin_ratio_kill'`).Scan(&esc)
	if esc != 1 {
		t.Fatalf("kill 升件=%d want 1", esc)
	}
}

// ③b flatten 败=flatten_failed 不 repay（序反=事故禁线锚）。
func TestMarginCloseLoanFlattenFailNoRepay(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	var loanID int64
	err := pool.QueryRow(ctx, `INSERT INTO fin_margin_loan(venue,ccy,inst_id,amt,apr,status)
		VALUES('okx','FF','FF-USDT',3,0.1,'open') RETURNING id`).Scan(&loanID)
	if err != nil {
		t.Fatal(err)
	}
	loans, _ := marginOpenLoans(ctx, pool)
	var l marginLoan
	for _, x := range loans {
		if x.ID == loanID {
			l = x
		}
	}
	fv := &fakeMarginVenue{placeErr: fmt.Errorf("venue reject")}
	if err := marginCloseLoan(ctx, pool, fv, l, "test"); err == nil {
		t.Fatal("flatten 败应上抛")
	}
	for _, c := range fv.calls {
		if c == "borrowrepay:repay" {
			t.Fatalf("flatten 败仍 repay——序反事故 calls=%v", fv.calls)
		}
	}
	var status string
	_ = pool.QueryRow(ctx, `SELECT status FROM fin_margin_loan WHERE id=$1`, loanID).Scan(&status)
	if status != "flatten_failed" {
		t.Fatalf("status=%s want flatten_failed", status)
	}
	// 清场——flatten_failed 残留会被 O-41350 扩后枚举窗拾取污染后续测。
	if _, err := pool.Exec(ctx, `UPDATE fin_margin_loan SET status='repaid',repaid_at=now() WHERE id=$1`, loanID); err != nil {
		t.Fatal(err)
	}
}

// ④exec_enabled 缺省 false=执行闸 fail-closed（venue 零调用）。
func TestMarginShortOpenDisabled(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	cfg := marginCfg(t, pool)
	// 缺省 false——迁移种子值；显式置 false 防他测污染。
	if _, err := pool.Exec(ctx, `UPDATE app_config SET value='false' WHERE key='margin.exec_enabled'`); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	gate := risk.NewGate(pool, cfg).WithEquity(func(context.Context, string) (float64, error) { return 10000, nil })
	fv := &fakeMarginVenue{}
	h := hedgeOpener{pool: pool, cfg: cfg, gate: gate, mvenue: fv,
		redeem: func(context.Context, float64) error { return nil }}
	_, err := h.marginShortOpen(ctx, lifecycle.OpenSpec{
		StrategyRef: 999999, Venue: "okx", InstID: "TST-USDT-SWAP",
		Direction: "long_perp_short_spot", NotionalUSD: 100})
	if err == nil {
		t.Fatal("exec_enabled=false 应拒")
	}
	if len(fv.calls) != 0 {
		t.Fatalf("闸拒后 venue 仍被调 calls=%v", fv.calls)
	}
}

// ④b 开仓全序：赎→借→确认→卖→簿记（fin_margin_loan+position+cashflow）。
func TestMarginShortOpenSequence(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	cfg := marginCfg(t, pool)
	// 闸键+证据+价全备。
	for _, kv := range [][2]string{
		{"margin.exec_enabled", "true"}, {"margin.max_open_usd", "500"},
		{"funding.margin_borrow_stale_hours", "6"}, {"funding.margin_borrow_max_apr", "0.5"},
		{"funding.margin_min_loan_usd", "1"},
	} {
		if _, err := pool.Exec(ctx, `UPDATE app_config SET value=$1 WHERE key=$2`, kv[1], kv[0]); err != nil {
			t.Fatal(kv, err)
		}
	}
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_margin_borrow(ccy,inst_id,mgn_mode,borrow_apr,max_loan,ts)
		VALUES('SEQ','SEQ-USDT','cross',0.1,1000,now())`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last)
		VALUES(now(),'okx','SEQ-USDT',50)`); err != nil {
		t.Fatal(err)
	}
	gate := risk.NewGate(pool, cfg).WithEquity(func(context.Context, string) (float64, error) { return 100000, nil })
	// 预期 qty=notional/px——notional=min(100,500,probe_cap,max_cap) 实读
	// 判据键算（cap 值域外不猜）。
	cap1, _ := cfg.GetFloat("exec.probe_notional_usd")
	cap2, _ := cfg.GetFloat("risk.max_notional_usd")
	wantN := min(100.0, 500.0, cap1, cap2)
	wantQty := wantN / 50.0
	var redeemed float64
	fv := &fakeMarginVenue{
		maxLoan: 1000, adjEq: 50000,
		hist: []okxexec.BorrowRepayHistRow{{ // 借后确认行（2min 内 borrow amt=qty）
			Ccy: "SEQ", Amt: fmt.Sprintf("%g", wantQty), Side: "borrow",
			TS: fmt.Sprintf("%d", time.Now().UnixMilli())}},
		resolveSt: okxexec.StateFilled, fillSz: wantQty,
	}
	h := hedgeOpener{pool: pool, cfg: cfg, gate: gate, mvenue: fv,
		redeem: func(_ context.Context, usd float64) error { redeemed = usd; return nil }}
	execID, err := h.marginShortOpen(ctx, lifecycle.OpenSpec{
		StrategyRef: 999998, Venue: "okx", InstID: "SEQ-USDT-SWAP",
		Direction: "long_perp_short_spot", NotionalUSD: 100})
	if err != nil {
		t.Fatal("开仓败:", err)
	}
	if execID == 0 {
		t.Fatal("execID=0")
	}
	if redeemed <= 0 {
		t.Fatal("JIT 赎回未触发")
	}
	// 序锚：borrow 先于 place:sell；簿记三件齐。
	var bIdx, sIdx = -1, -1
	for i, c := range fv.calls {
		if c == "borrowrepay:borrow" && bIdx < 0 {
			bIdx = i
		}
		if c == "place:sell" {
			sIdx = i
		}
	}
	if !(bIdx >= 0 && sIdx > bIdx) {
		t.Fatalf("借卖序乱 calls=%v", fv.calls)
	}
	var loans, pos, cf int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM fin_margin_loan WHERE ccy='SEQ' AND status='open'`).Scan(&loans)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM fin_position WHERE inst_id='SEQ-USDT' AND delta_neutral=false AND closed_at IS NULL`).Scan(&pos)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM fin_cashflow WHERE kind='spot_sell' AND source_table='fin_execution'`).Scan(&cf)
	if loans != 1 || pos != 1 || cf != 1 {
		t.Fatalf("簿记三件 loans=%d pos=%d cf=%d want 1/1/1", loans, pos, cf)
	}
	// 清场——open loan/position 残留污染共享 schema 后续测。
	if _, err := pool.Exec(ctx, `UPDATE fin_margin_loan SET status='repaid',repaid_at=now() WHERE ccy='SEQ'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE fin_position SET closed_at=now() WHERE inst_id='SEQ-USDT' AND closed_at IS NULL`); err != nil {
		t.Fatal(err)
	}
}
