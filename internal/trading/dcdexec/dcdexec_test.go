// dcdexec_test.go —— O-CAP1 桩锚：dcdexec 前置赎回缝（工单#20484——
// trading 不足 deficit 时 Deps.Redeem 先赎储蓄回拢再摆渡；
// 赎回败 fail-closed exec=failed 且 DcdTrade 零调用；
// Redeem=nil 旧径逐字不变）。
package dcdexec

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/risk"
	"o/internal/store/pgstore"
)

// stubVenue dcdexec.Venue 七缝伪件——funding 余/摆渡/trade/status 全控。
type stubVenue struct {
	avail       float64
	availErr    error
	transfers   []transferCall
	transferErr error
	tradeCalls  int
	statusCalls int
}

type transferCall struct {
	ccy, from, to string
	amt           float64
}

func (s *stubVenue) DcdQuote(context.Context, string, string, string) (string, error) {
	return "Q1", nil
}
func (s *stubVenue) DcdTrade(context.Context, string) (json.RawMessage, error) {
	s.tradeCalls++
	return json.RawMessage(`[{"ordId":"O1","sCode":"0"}]`), nil
}
func (s *stubVenue) DcdOrderStatus(context.Context, string) (json.RawMessage, error) {
	s.statusCalls++
	return json.RawMessage(`[{"state":"LIVE"}]`), nil
}
func (s *stubVenue) FundingAvail(context.Context, string) (float64, error) {
	return s.avail, s.availErr
}
func (s *stubVenue) AssetTransfer(_ context.Context, ccy string, amt float64, from, to string) (string, string, error) {
	s.transfers = append(s.transfers, transferCall{ccy, from, to, amt})
	return "cl1", "tr1", s.transferErr
}
func (s *stubVenue) AssetTransferState(context.Context, string, string) (string, error) {
	return "success", nil
}

// redeemFixture 迁移+产品行（notionalCcy=USDT——进摆渡域）+闸放行种子
// （同 open_dcd_reject_test 先例：exposure 阈放宽+BTC tick 供估值径
// ——本桩面只验赎回序非闸判据）。
func redeemFixture(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	pid := "BTC-USDT-261225-95000-C"
	if _, err := pool.Exec(ctx, `INSERT INTO md_dcd_product(ts,venue,surface,uly,product_id,direction,raw)
		VALUES(now(),'okx','dcd','BTC-USDT',$1,'C','{"notionalCcy":"USDT"}')`, pid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,venue,inst_id,last) VALUES(now(),'okx','BTC-USDT',100000)
		 ON CONFLICT(venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO app_config(key,value) VALUES('risk.max_exposure_usd',to_jsonb('1000000000'::text))
		 ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`); err != nil {
		t.Fatal(err)
	}
	return ctx, pool
}

func redeemSpec() Spec {
	return Spec{StrategyRef: 1, Venue: "okx", InstID: "BTC-USDT-261225-95000-C",
		Direction: "long_call", Template: "dcd_call", NotionalUSD: 100}
}

// redeemDeps 闸装配（equity 注入大额放行——同 dcdRejectFixture 先例）。
func redeemDeps(t *testing.T, pool *pgxpool.Pool, redeem func(context.Context, float64) error) Deps {
	t.Helper()
	cfg := config.New(pool)
	if err := cfg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	return Deps{Pool: pool, Gate: risk.NewGate(pool, cfg).WithEquity(
		func(context.Context, string) (float64, error) { return 1e6, nil }), Redeem: redeem}
}

// 锚①②：deficit=100-30=70——Redeem 以 deficit 原值被调（勿自乘），
// 被调后摆渡 AssetTransfer("USDT",70,trading→funding) 照发，全链
// 走到 DcdTrade 定态 open。
func TestDcdExecRedeemBeforeTransfer(t *testing.T) {
	v := &stubVenue{avail: 30}
	ctx, pool := redeemFixture(t)
	var gotAmt float64
	var gotCalls int
	d := redeemDeps(t, pool, func(_ context.Context, usdAmt float64) error {
		gotCalls++
		gotAmt = usdAmt
		return nil
	})
	execID, err := Open(ctx, d, v, redeemSpec())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if execID == 0 {
		t.Fatal("execID 应非零")
	}
	if gotCalls != 1 || gotAmt != 70 {
		t.Fatalf("Redeem 应以 deficit=70 原值被调一次: calls=%d amt=%v", gotCalls, gotAmt)
	}
	if len(v.transfers) != 1 || v.transfers[0].amt != 70 ||
		v.transfers[0].ccy != "USDT" || v.transfers[0].from != AcctTrading || v.transfers[0].to != AcctFunding {
		t.Fatalf("摆渡应照发 USDT 70 trading→funding: %+v", v.transfers)
	}
	if v.tradeCalls != 1 {
		t.Fatalf("DcdTrade 应被调一次: %d", v.tradeCalls)
	}
	var state string
	if err := pool.QueryRow(ctx, `SELECT state FROM fin_execution WHERE id=$1`, execID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "open" {
		t.Fatalf("exec 终态应 open（LIVE 对拍）: %s", state)
	}
}

// 锚③：Redeem 返错→fin_execution state='failed' 且 DcdTrade 零调用
// （fail-closed——钱径半成不摆不盲动）。
func TestDcdExecRedeemFailClosed(t *testing.T) {
	v := &stubVenue{avail: 30}
	ctx, pool := redeemFixture(t)
	d := redeemDeps(t, pool, func(context.Context, float64) error {
		return errors.New("earn redeem 划转败")
	})
	execID, err := Open(ctx, d, v, redeemSpec())
	if err == nil {
		t.Fatal("Redeem 败应上抛")
	}
	if execID == 0 {
		t.Fatal("execID 应非零（intent 行已落）")
	}
	var state string
	if err := pool.QueryRow(ctx, `SELECT state FROM fin_execution WHERE id=$1`, execID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "failed" {
		t.Fatalf("exec 态应 failed: %s", state)
	}
	if v.tradeCalls != 0 {
		t.Fatalf("赎回败 DcdTrade 应零调用: %d", v.tradeCalls)
	}
	if len(v.transfers) != 0 {
		t.Fatalf("赎回败摆渡应零调用: %+v", v.transfers)
	}
}

// 锚④：Redeem=nil→旧径逐字不变——deficit>0 直接摆渡，零赎回调用
// 面（dcdSim 缝/未装配零影响）。
func TestDcdExecRedeemNilOldPath(t *testing.T) {
	v := &stubVenue{avail: 30}
	ctx, pool := redeemFixture(t)
	d := redeemDeps(t, pool, nil)
	execID, err := Open(ctx, d, v, redeemSpec())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if len(v.transfers) != 1 || v.transfers[0].amt != 70 {
		t.Fatalf("旧径摆渡应照发 deficit=70: %+v", v.transfers)
	}
	if v.tradeCalls != 1 {
		t.Fatalf("旧径 DcdTrade 应被调: %d", v.tradeCalls)
	}
	var state string
	if err := pool.QueryRow(ctx, `SELECT state FROM fin_execution WHERE id=$1`, execID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "open" {
		t.Fatalf("exec 终态应 open: %s", state)
	}
}
