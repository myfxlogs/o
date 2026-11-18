// localwrite.go —— T-0209 SG 开仓闸行情自供谓词（main.go 棘轮瘦身件）。
package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/data/okxws"
)

// attachLocalWritePred 活跃集谓词接件：inst∈fin_position 未平仓∪
// fin_strategy 非 retired∪legset.ActiveSetUnion 订阅集同源臂
// （projector serve 谓词同式自算不赖种子表；tickers→md_ticks、
// funding→md_funding 同键幂等共写——dev 桥恢复后最新赢自动让位，
// 行数不膨胀撕裂对拍）。读败=不落行 fail-closed。
// 工单#24376：谓词本体迁 okxws.LocalWritePredFunc——订阅集
// ActiveSetUnion 并入成「订而不写」根修（SubRequestArm/提案臂
// 入订阅集时写谓词未随行，TRX-USDT-SWAP 日K帧静默丢实证）。
func attachLocalWritePred(ctx context.Context, c *okxws.Collector, pool *pgxpool.Pool, get okxws.ScanGet) *okxws.Collector {
	c.AttachRestGet(get) // 工单#13065③：日K史面回填读件（懒客 ScanGet 注入）
	c.LocalWritePred = okxws.LocalWritePredFunc(ctx, pool)
	return c
}
