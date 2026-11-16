// coins_test.go —— O-CAP1 桩锚：okxCoinsFor 储蓄 USDT 可赎加算
// （工单#20484——trading avail 之外另计 /finance/savings/balance
// USDT amt 本金面；仅 USDT 计，DOGE 等业主自持资产永不入账；
// 读败/解析败=降级不加算不上抛）。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// coinsGetter authGetter 窄面桩——path→响应/错误分发行。
type coinsGetter func(ctx context.Context, path string) (json.RawMessage, error)

func (f coinsGetter) Get(ctx context.Context, path string) (json.RawMessage, error) {
	return f(ctx, path)
}

// coinsFixture 迁移库+桩 getter 装配（md_account_events 空表=WS 帧径
// 自然落空走 REST 兜底；WS 径用例另 seed 帧行）。
func coinsFixture(t *testing.T, g authGetter) func(context.Context) (map[string]float64, error) {
	t.Helper()
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	return okxCoinsFor(pool, g)
}

const (
	coinsAcctPath = "/api/v5/account/balance"
	coinsSavPath  = "/api/v5/finance/savings/balance"
)

// REST 兜底径：account/balance USDT availBal=100 + savings USDT amt=50
// +DOGE amt=999 → out["USDT"]==150；DOGE 永不入账。
func TestOkxCoinsForSavingsAdded(t *testing.T) {
	g := coinsGetter(func(_ context.Context, path string) (json.RawMessage, error) {
		switch path {
		case coinsAcctPath:
			return json.RawMessage(`[{"details":[{"ccy":"USDT","availBal":"100"}]}]`), nil
		case coinsSavPath:
			return json.RawMessage(`[{"ccy":"DOGE","amt":"999"},{"ccy":"USDT","amt":"50"}]`), nil
		}
		return nil, errors.New("unexpected path " + path)
	})
	out, err := coinsFixture(t, g)(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if out["USDT"] != 150 {
		t.Fatalf("USDT 应=trading 100+savings 50=150: %v", out)
	}
	if _, ok := out["DOGE"]; ok {
		t.Fatalf("DOGE 储蓄=业主自持资产永不入账: %v", out)
	}
}

// savings 端点读败=降级：out["USDT"]==100 原样+err 上抛面零——
// 储蓄读败不盲拒交易户余量（半瞎好过全瞎）。
func TestOkxCoinsForSavingsFailOpen(t *testing.T) {
	g := coinsGetter(func(_ context.Context, path string) (json.RawMessage, error) {
		switch path {
		case coinsAcctPath:
			return json.RawMessage(`[{"details":[{"ccy":"USDT","availBal":"100"}]}]`), nil
		case coinsSavPath:
			return nil, errors.New("savings down")
		}
		return nil, errors.New("unexpected path " + path)
	})
	out, err := coinsFixture(t, g)(context.Background())
	if err != nil {
		t.Fatalf("savings 读败不应上抛（降级径）: %v", err)
	}
	if out["USDT"] != 100 {
		t.Fatalf("savings 读败应保 trading 余量 100: %v", out)
	}
}

// WS 帧径（md_account_events 有 account 帧→len(out)>0 早退点同挂
// 加算——两处 return 全覆盖非只 REST 径）。
func TestOkxCoinsForSavingsWSPref(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_account_events(ts,channel,data)
		VALUES(now(),'account','{"details":[{"ccy":"USDT","availBal":"100"}]}')`); err != nil {
		t.Fatal(err)
	}
	g := coinsGetter(func(_ context.Context, path string) (json.RawMessage, error) {
		if path == coinsSavPath {
			return json.RawMessage(`[{"ccy":"USDT","amt":"50"}]`), nil
		}
		return nil, errors.New("unexpected path " + path)
	})
	out, err := okxCoinsFor(pool, g)(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if out["USDT"] != 150 {
		t.Fatalf("WS 径 USDT 应=帧 100+savings 50=150: %v", out)
	}
}
