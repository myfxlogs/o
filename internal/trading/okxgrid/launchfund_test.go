// launchfund_test.go —— launchGate 前置资金闸四锚（工单#26337）：
// ①余额足零赎回直拉起 ②余额不足赎回→拉起零隙次序 ③赎回败 hold 禁重试
// ④拉起歧态势 hold+定态分类表。清扫赛跑回归注：零隙纪律为何在——
// 实证坑（bills 575/577）清扫臂 2s 把交易户 bot 本金扫回 earn 认购，
// 赎回→划转→Launch 同棒零 sleep 是唯一防送钱回 earn 的形态。
package okxgrid

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
	"o/internal/trading/okxexec"
)

type fundMock struct {
	bal       float64
	balErr    error
	redeemErr error
	launchErr error
	calls     []string // "bal"|"redeem"|"launch" 次序账
	redeemAmt float64
}

func launchFixture(t *testing.T) (context.Context, *Runner, *fundMock) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"okxgrid.live_switch", "okxgrid.max_invest_usd"} {
		if _, err := pool.Exec(ctx, "DELETE FROM app_config WHERE key=$1", k); err != nil {
			t.Fatal(err)
		}
	}
	for _, q := range []string{
		`DELETE FROM fin_bot_order WHERE inst_id='LT-USDT'`,
		`DELETE FROM fin_bot_launch_hold WHERE inst_id='LT-USDT'`,
		`DELETE FROM fin_strategy WHERE inst_id='LT-USDT'`,
		`DELETE FROM fin_proposal WHERE inst_id='LT-USDT'`,
		`DELETE FROM fin_opp_eval WHERE inst_id='LT-USDT'`,
		`DELETE FROM ops_event WHERE seat='okxgrid' AND note LIKE '%LT-USDT%'`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM fin_bot_order WHERE inst_id='LT-USDT'`,
			`DELETE FROM fin_bot_launch_hold WHERE inst_id='LT-USDT'`,
			`DELETE FROM fin_strategy WHERE inst_id='LT-USDT'`,
			`DELETE FROM fin_proposal WHERE inst_id='LT-USDT'`,
			`DELETE FROM fin_opp_eval WHERE inst_id='LT-USDT'`,
			`DELETE FROM ops_event WHERE seat='okxgrid' AND note LIKE '%LT-USDT%'`,
			`DELETE FROM app_config WHERE key IN ('okxgrid.live_switch','okxgrid.max_invest_usd')`,
		} {
			pool.Exec(ctx, q)
		}
	})
	for k, v := range map[string]string{"okxgrid.live_switch": "true", "okxgrid.max_invest_usd": "100"} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	var evalID, propID, stratID int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval
		(inst_id,template,direction,verdict,legs,evidence,net_apr)
		VALUES('LT-USDT','grid_spot','grid_spot','proposed','{}',
			'{"band_lo":10,"band_hi":20,"n_lines":4,"capital":50}',0.10) RETURNING id`).Scan(&evalID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
		VALUES($1,'LT-USDT','grid_spot',0) RETURNING id`, evalID).Scan(&propID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy
		(inst_id,template_ref,proposal_ref,state,notional_usd,venue)
		VALUES('LT-USDT','grid_spot',$1,'live',50,'okx') RETURNING id`, propID).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	m := &fundMock{}
	st := &stubDoer{postRaw: json.RawMessage(`[{"algoId":"999","sCode":"0"}]`)}
	r := NewRunner(pool, cfg, &Adapter{c: launchSeq{nx: st, m: m}},
		func(_ context.Context, _ string) (float64, error) {
			m.calls = append(m.calls, "bal")
			return m.bal, m.balErr
		},
		func(_ context.Context, usd float64) error {
			m.calls = append(m.calls, "redeem")
			m.redeemAmt = usd
			return m.redeemErr
		})
	return ctx, r, m
}

// launchSeq Doer 装饰——post 面记序+可注错（launch 棒次序账）。
type launchSeq struct {
	m  *fundMock
	nx Doer
}

func (l launchSeq) Get(ctx context.Context, p string) (json.RawMessage, error) {
	return l.nx.Get(ctx, p)
}
func (l launchSeq) Post(ctx context.Context, p string, b []byte) (json.RawMessage, error) {
	l.m.calls = append(l.m.calls, "launch")
	if l.m.launchErr != nil {
		return nil, l.m.launchErr
	}
	return l.nx.Post(ctx, p, b)
}

func TestLaunchGateSufficientBalanceNoRedeem(t *testing.T) {
	ctx, r, m := launchFixture(t)
	m.bal = 60 // ≥ quoteSz 50——零赎回
	if err := r.launchGate(ctx); err != nil {
		t.Fatal(err)
	}
	for _, c := range m.calls {
		if c == "redeem" {
			t.Fatalf("余额足仍触发赎回（calls=%v）——零赎回锚破", m.calls)
		}
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_bot_order
		WHERE inst_id='LT-USDT' AND state='launched'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("bot 行缺席 n=%d err=%v", n, err)
	}
}

func TestLaunchGateRedeemThenLaunchZeroGap(t *testing.T) {
	ctx, r, m := launchFixture(t)
	m.bal = 30 // 缺 20——赎回补缺口再拉起
	if err := r.launchGate(ctx); err != nil {
		t.Fatal(err)
	}
	if len(m.calls) != 3 || m.calls[0] != "bal" || m.calls[1] != "redeem" || m.calls[2] != "launch" {
		t.Fatalf("次序账破（%v）——赎回→拉起零隙锚", m.calls)
	}
	if m.redeemAmt != 20 {
		t.Fatalf("赎回额=%v want 20（deficit=quoteSz-avail）", m.redeemAmt)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_bot_order
		WHERE inst_id='LT-USDT' AND state='launched'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("bot 行缺席 n=%d err=%v", n, err)
	}
}

func TestLaunchGateRedeemFailHoldNoRetry(t *testing.T) {
	ctx, r, m := launchFixture(t)
	m.bal = 30
	m.redeemErr = errors.New("earn redeem 划转在途: transport dead") // 歧态/败同组——hold
	if err := r.launchGate(ctx); err != nil {
		t.Fatal(err)
	}
	for _, c := range m.calls {
		if c == "launch" {
			t.Fatalf("赎回败仍拉起（calls=%v）——不拉起锚破", m.calls)
		}
	}
	var stage string
	if err := r.pool.QueryRow(ctx, `SELECT stage FROM fin_bot_launch_hold
		WHERE inst_id='LT-USDT'`).Scan(&stage); err != nil || stage != "redeem" {
		t.Fatalf("hold 行缺席 stage=%q err=%v", stage, err)
	}
	var esc int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE seat='okxgrid' AND esc_scope='okxgrid_launch_ambiguous'
		AND note LIKE '%LT-USDT%'`).Scan(&esc); err != nil || esc != 1 {
		t.Fatalf("升件行缺席 n=%d err=%v", esc, err)
	}
	// 二拍禁自动重试——hold 行在案 launch 不再触。
	m.calls = nil
	if err := r.launchGate(ctx); err != nil {
		t.Fatal(err)
	}
	for _, c := range m.calls {
		if c == "launch" || c == "redeem" {
			t.Fatalf("hold 在案二拍仍触 %s（calls=%v）——禁自动重试补单锚破", c, m.calls)
		}
	}
}

func TestLaunchGateAmbiguousHoldDefiniteEsc(t *testing.T) {
	ctx, r, m := launchFixture(t)
	m.bal = 60                                                         // 资金足——零赎回，歧态在 launch 棒
	m.launchErr = fmt.Errorf("httpStatusError: %w", errors.New("EOF")) // 传输败=歧态
	if err := r.launchGate(ctx); err != nil {
		t.Fatal(err)
	}
	var stage string
	if err := r.pool.QueryRow(ctx, `SELECT stage FROM fin_bot_launch_hold
		WHERE inst_id='LT-USDT'`).Scan(&stage); err != nil || stage != "launch" {
		t.Fatalf("launch 歧态 hold 缺席 stage=%q err=%v", stage, err)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_bot_order
		WHERE inst_id='LT-USDT'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("歧态拉起不得落 bot 行 n=%d", n)
	}
	// 定态（APIError 业务明拒）→esc+continue 不 hold——下拍重试安全。
	if _, err := r.pool.Exec(ctx, `DELETE FROM fin_bot_launch_hold WHERE inst_id='LT-USDT'`); err != nil {
		t.Fatal(err)
	}
	m.launchErr = &okxexec.APIError{Code: "1", SCode: "58350", SMsg: "Insufficient balance"}
	if err := r.launchGate(ctx); err != nil {
		t.Fatal(err)
	}
	var h int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_bot_launch_hold
		WHERE inst_id='LT-USDT'`).Scan(&h); err != nil || h != 0 {
		t.Fatalf("定态拒误落 hold n=%d", h)
	}
}

func TestLaunchDefiniteClassifier(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{&okxexec.APIError{Code: "1", SCode: "58350"}, true},             // 业务明拒
		{&okxexec.HTTPStatusError{Code: 429, Body: "rate"}, true},        // 边缘拒未入业务
		{&okxexec.HTTPStatusError{Code: 502, Body: "bad gw"}, false},     // 5xx=歧态（票钦）
		{fmt.Errorf("httpStatusError: %w", errors.New("EOF")), false},    // 断连=歧态
		{fmt.Errorf("wrap: %w", &okxexec.APIError{Code: "51008"}), true}, // %w 链透传
		{fmt.Errorf("resp parse: %w", errors.New("bad json")), false},    // 解析败=歧态
	}
	for i, c := range cases {
		if got := launchDefinite(c.err); got != c.want {
			t.Fatalf("case %d launchDefinite=%v want %v (err=%v)", i, got, c.want, c.err)
		}
	}
}
