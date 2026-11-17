// Package evalgate —— fin_opp_eval 写径指纹闸（工单#22904：md_1097
// insert-only 版本史落地后，eval 环同 (inst,template,direction) 指纹未变
// 仍每拍 INSERT——md 侧 89 行/s≈7.7M 行/日≈2-4GB/日盘烧，OCI 盘余
// 数日撑爆）。同键最新行 verdict+net_apr 未变=信念未迁不产新版：
// 跳 INSERT 复用行 id 续提案径（upsert 时代同义——行 id 稳定、判词/
// 净额变迁仍留版本史）。全族写径单点闸——8 写面同消费防判据漂移。
package evalgate

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// Querier pgxpool.Pool/pgx.Tx 公共最小读面（写拍批内合流事务与散写
// 同吃——指纹判定与 INSERT 同事务面，批内自见）。
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// SameFingerprint 查 (inst,template,direction) 最新评估行（id DESC——
// fin_opp_eval_key_latest 索引同键序）指纹是否与本拍一致：verdict+
// net_apr 双同=信念未迁。返 (行id,true)——调用方跳 INSERT、复用 id
// 续下游提案/呈批径；指纹异或无历史行返 (0,false)——走原 INSERT 径
// （判词/净额变迁=新版史，版本化 insert-only 语义不破）。
// net_apr 比对走库内 IS NOT DISTINCT FROM——numeric 精确域判（参数
// 经 ::numeric 显式转；NULL↔NULL 同指纹——gapscan 面 NULL 族同闸）。
func SameFingerprint(ctx context.Context, q Querier, inst, template, direction, verdict string, netAPR any) (int64, bool, error) {
	var id int64
	var same bool
	err := q.QueryRow(ctx, `SELECT id,
		(verdict=$4 AND net_apr IS NOT DISTINCT FROM $5::numeric) AS same
		FROM fin_opp_eval
		WHERE inst_id=$1 AND template=$2 AND direction=$3
		ORDER BY id DESC LIMIT 1`,
		inst, template, direction, verdict, netAPR).Scan(&id, &same)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if same {
		return id, true, nil
	}
	return 0, false, nil
}
