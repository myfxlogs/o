// cmd/core bills 源适配——T-0044 funding 记账件装配（funding.Bill 域自有型
// 转换——账本域不反向依赖交易域，cmd 不受方向闸）。
package main

import (
	"context"

	legfunding "phase/internal/ledger/funding"
	"phase/internal/trading/okxexec"
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
