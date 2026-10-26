// cmd/core bills 源适配——T-0044 funding 记账件装配（funding.Bill 域自有型
// 转换——账本域不反向依赖交易域，cmd 不受方向闸）。
package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	legfunding "o/internal/ledger/funding"
	"o/internal/trading/okxexec"
)

// billsSrc okxexec bills 源适配（cmd 装配层——funding.Bill 域自有型转换）。
type billsSrc struct{ c *okxexec.Client }

func (s billsSrc) Bills(ctx context.Context, instType, after string) ([]legfunding.Bill, error) {
	rows, err := s.c.Bills(ctx, instType, after)
	if err != nil {
		return nil, err
	}
	out := make([]legfunding.Bill, len(rows))
	for i, r := range rows {
		out[i] = legfunding.Bill{BillID: r.BillID, Type: r.Type, SubType: r.SubType,
			InstID: r.InstID, InstType: r.InstType, Ccy: r.Ccy, TsMs: r.TsMs,
			BalChg: r.BalChg, Fee: r.Fee, Pnl: r.Pnl, Sz: r.Sz, Px: r.Px}
	}
	return out, nil
}

// depSrc 充币史源（T-0149——装配层转件：okxexec RawMessage→ledger DepositRow）。
type depSrc struct{ c *okxexec.Client }

func (s depSrc) DepositHistory(ctx context.Context) ([]legfunding.DepositRow, error) {
	raw, err := s.c.DepositHistory(ctx)
	if err != nil {
		return nil, err
	}
	var rows []legfunding.DepositRow
	return rows, json.Unmarshal(raw, &rows)
}

// ledgerHook bills 钩装配（T-0044 记账+T-0149 ①入金检测同帧拍——余额事件
// 帧驱动：bills 记账 Ingest→deposit-history 补漏扫（无推送面=REST 补漏
// 正当面；depId 幂等防双记——重拍/重放不双账）。
func ledgerHook(pool *pgxpool.Pool, mk func() *okxexec.Client) func(context.Context) {
	return func(ctx context.Context) {
		legfunding.NewIngester(pool, billsSrc{mk()}).Hook()(ctx)
		if n, err := legfunding.NewDepositScanner(pool,
			depSrc{mk()}).Scan(ctx); err != nil {
			log.Printf("[ledger] 入金巡检败（下帧重试）: %v", err)
		} else if n > 0 {
			log.Printf("[ledger] 入金巡检落 %d 行", n)
		}
	}
}
