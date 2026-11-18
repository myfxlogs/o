// localwritepred.go —— T-0209 SG 活跃集写谓词本体（工单#24376 根修：
// 订阅集 ActiveSetUnion 与写谓词键域错位=订而不写机制空转——
// SubRequestArm/提案臂入订阅集时写谓词未随行，fin_sub_request open
// 的 TRX-USDT-SWAP 订了日K帧被谓词闸静默丢，md_klines recv_ts 冻
// 2026-11-17 20:02Z 实证）。
package okxws

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/legset"
)

// localWritePredSQL 写谓词集——在仓∪非退役策略两臂（T-0209 原义保留）
// ∪订阅集同源臂 legset.ActiveSetUnion（T-0401 提案/晋级臂+三族派生腿
// 全谱——订阅集与写谓词同源，后加臂自动生效；upsert 幂等同键与桥写
// 共存 T-0209 注释已钦，宽写无膨胀险）。读败=不落行 fail-closed。
var localWritePredSQL = `SELECT count(*) FROM (
	SELECT inst_id FROM fin_position WHERE closed_at IS NULL
	UNION SELECT inst_id FROM fin_strategy WHERE state<>'retired'
	UNION ` + legset.ActiveSetUnion + `) t WHERE t.inst_id=$1`

// LocalWritePredFunc SG 活跃集写谓词接件（Collector.LocalWritePred 槽位
// 注入件——kind 形参各面共用同集，T-0209 原语义不动）。
func LocalWritePredFunc(ctx context.Context, pool *pgxpool.Pool) func(kind, instID string) bool {
	return func(kind, instID string) bool {
		var n int
		if err := pool.QueryRow(ctx, localWritePredSQL, instID).Scan(&n); err != nil {
			return false // 读败=不落行（fail-closed 不写僵尸）
		}
		return n > 0
	}
}
