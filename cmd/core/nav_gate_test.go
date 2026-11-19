package main

// nav_gate_test.go —— 工单#24802 锚：NAV 日拍完备性闸。腿读败缺腿→
// nav() 返 err 本拍让位不写（md_equity_curve 零新增由 report 读败
// 让位径兜——err 即 skip,本件断言闸面本身）；腿复常→正常 map 出账。

import (
	"context"
	"errors"
	"strings"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
	"o/internal/trading/okxexec"
)

func TestNavPartialLegsGate(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	// FundCcy 无条件种 USDT×3 户+mt5 USD。交易户腿=md_account_events
	// 最新帧——空表=ErrNoRows→读件缺行（真缺腿,闸该拦）,种一帧供
	// 复常态。
	if _, err := pool.Exec(ctx, `INSERT INTO md_account_events(ts,channel,data)
		VALUES(now(),'account','{"details":[{"ccy":"USDT","eqUsd":"10","eq":"10"}]}')`); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	okMap := func(v float64) func(context.Context) (map[string]float64, error) {
		return func(context.Context) (map[string]float64, error) {
			return map[string]float64{"USDT": v}, nil
		}
	}
	errMap := func(context.Context) (map[string]float64, error) {
		return nil, errors.New("venue read down")
	}
	mt5OK := func(context.Context) ([]mt5Acct, error) {
		return []mt5Acct{{Login: "1", Equity: 10}}, nil
	}
	dcdOK := func(context.Context) ([]okxexec.DcdOrder, error) {
		return nil, nil
	}
	mkNav := func(fund, earn func(context.Context) (map[string]float64, error)) func(context.Context) (map[string]float64, error) {
		_, navFn := assemble(pool, cfg, fund, earn, mt5OK, dcdOK, nil, nil, nil)
		return navFn
	}
	// ① funding 腿读败 → nav() err（缺腿闸）——下拍复常自写。
	if _, err := mkNav(errMap, okMap(5))(ctx); err == nil ||
		!strings.Contains(err.Error(), "资金户") {
		t.Fatalf("funding 读败应拦本拍, got %v", err)
	}
	// ② earn 腿读败 → 同拦。
	if _, err := mkNav(okMap(5), errMap)(ctx); err == nil ||
		!strings.Contains(err.Error(), "Earn") {
		t.Fatalf("earn 读败应拦本拍, got %v", err)
	}
	// ③ 全腿复常 → map 出账（交易10+资金5+Earn7=USDT 22；NAV 另+mt5
	// 10=32）。
	by, err := mkNav(okMap(5), okMap(7))(ctx)
	if err != nil {
		t.Fatalf("复常应出账: %v", err)
	}
	if by["USDT"] != 22 || by["NAV"] != 32 {
		t.Fatalf("账不对: %+v", by)
	}
	// ④ dcd 腿读败 → 同拦（结构性持仓缺腿同险）。
	_, navFn := assemble(pool, cfg, okMap(5), okMap(7), mt5OK,
		func(context.Context) ([]okxexec.DcdOrder, error) {
			return nil, errors.New("dcd read down")
		}, nil, nil, nil)
	if _, err := navFn(ctx); err == nil {
		t.Fatal("dcd 读败应拦本拍")
	}
}
