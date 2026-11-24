// lessongate.go —— 教训矿消费环（工单O-38942）：verdicted 验尸案从死档
// 接入提案消费点——同 (template_ref,inst_id,direction) 且同 reason 尸体
// ≥lessonDeadSigMin 且该组净额合计<0 → 新提案 fin_retired
// （retire_reason='dead_signature'）+ops_event「教训消费」审计行。
// 上游定性（外席取证坐实）：矿链三站产 pending→唤知裁定→verdicted
// 死档；S0 量化=48 死签 114 复生 107 再死（94%）——消费点值建。
// 读面纪律：统计面读败=fail-open 放行（统计缺≠抑制——抑制须可证有据，
// spec 禁线）；闸读 app_config lifecycle.lesson_dead_sig 默认开（可关
// =回滚不撤码）。不动裁定语义/不生新策略——只抑制+审计。
package lifecycle

import (
	"context"
	"fmt"
	"log"
	"time"
)

// lessonDeadSigKey 消费闸开关（app_config——默认开：键缺席=开；
// 置 false=全闸停摆回滚不撤码）。lifecycle 域=DM 写域册在册。
const lessonDeadSigKey = "lifecycle.lesson_dead_sig"

// lessonDeadSigMin 同签同因尸体计数阈（spec 建议 N=2；真库实证
// 1023 verdicted 尸按签名粒度 max=1——N=2=死闸零覆盖验收不可达，
// 取 N=1：单具净亏尸签即抑。reason 分组结构保留——矿化成熟后
// 调阈收紧只动常量）。
const lessonDeadSigMin = 1

// lessonStatsEvery 族尸统计面刷新节流（v_lesson_family_stats 物化
// 视图——scan 事件驱动拍内节流，1h 粒度对审计直读足够）。
const lessonStatsEvery = time.Hour

// deadSigHit 尸签命中载（reason=同签同因死法、n=该因尸数、net=净额合计）。
type deadSigHit struct {
	reason string
	n      int
	net    float64
}

// deadSigGateOn 闸开关读——键缺席/读败=默认开（spec 默认开语义；
// cfg nil=桩测直通）。
func (r *Runner) deadSigGateOn(ctx context.Context) bool {
	if r.cfg == nil {
		return true
	}
	on, err := r.cfg.GetBool(lessonDeadSigKey)
	if err != nil {
		return true // 键缺席=默认开（可禁=显式 false）
	}
	return on
}

// deadSignature 尸签判据——同 (template_ref,inst_id,direction) 尸案按
// reason 分组，任一组 count≥lessonDeadSigMin 且净额合计<0=命中。
// 净额=realized_usd+paper_net_usd 合计（纸仓死 realized≈0——判词
// 「realized 净额」按全簿净额口径取两腿和，判词行载明口径）。
// 读败返 err——调用方 fail-open 放行（禁「查尸败=全抑」反向病）。
func (r *Runner) deadSignature(ctx context.Context, tmpl, inst, dir string) (deadSigHit, bool, error) {
	rows, err := r.pool.Query(ctx, `SELECT numbers->>'reason' AS reason, count(*) AS n,
		COALESCE(sum(COALESCE((numbers->>'realized_usd')::float8,0)
			+ COALESCE((numbers->>'paper_net_usd')::float8,0)),0) AS net
		FROM fin_econ_lesson
		WHERE kind='retire_postmortem' AND status='verdicted'
			AND numbers->>'template_ref'=$1 AND numbers->>'inst_id'=$2
			AND numbers->>'direction'=$3
		GROUP BY 1`, tmpl, inst, dir)
	if err != nil {
		return deadSigHit{}, false, err
	}
	defer rows.Close()
	for rows.Next() {
		var h deadSigHit
		if err := rows.Scan(&h.reason, &h.n, &h.net); err != nil {
			return deadSigHit{}, false, err
		}
		if h.n >= lessonDeadSigMin && h.net < 0 {
			return h, true, rows.Err()
		}
	}
	return deadSigHit{}, false, rows.Err()
}

// deadSigSuppress 提案消费点闸——命中→提案 fin_retired
// （retire_reason='dead_signature'）+「教训消费」审计行同事务落账
// （消费动作必须可审计——先审后抑不留无证抑制）；未命中/闸关/
// 读败=放行（fail-open）。返 true=调用方 continue 跳过开户。
func (r *Runner) deadSigSuppress(ctx context.Context, proposalID int64, tmpl, inst, dir string) bool {
	if !r.deadSigGateOn(ctx) {
		return false
	}
	hit, ok, err := r.deadSignature(ctx, tmpl, inst, dir)
	if err != nil {
		log.Printf("[lifecycle] 尸签判据读败——fail-open 放行 proposal=#%d %s/%s/%s: %v",
			proposalID, tmpl, inst, dir, err)
		return false
	}
	if !ok {
		return false
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		log.Printf("[lifecycle] 尸签抑制事务起败——fail-open 放行 proposal=#%d: %v", proposalID, err)
		return false
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE fin_proposal SET state='fin_retired',
		retire_reason='dead_signature',retired_at=now()
		WHERE id=$1 AND state='fin_proposed'`, proposalID)
	if err != nil {
		log.Printf("[lifecycle] 尸签抑制提案退役写败——fail-open 放行 proposal=#%d: %v", proposalID, err)
		return false
	}
	if tag.RowsAffected() == 0 {
		return false // 并发已迁态——不重复审计不重复抑
	}
	note := fmt.Sprintf("proposal=#%d tmpl=%s inst=%s dir=%s 命中死签 reason=%s 尸×%d 净额=%.2fUSD → fin_retired(dead_signature)",
		proposalID, tmpl, inst, dir, hit.reason, hit.n, hit.net)
	if _, err := tx.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('探测席','教训消费','',$1)`, note); err != nil {
		log.Printf("[lifecycle] 教训消费审计行落败——事务回滚不抑 proposal=#%d: %v", proposalID, err)
		return false
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[lifecycle] 尸签抑制事务提交败——fail-open 放行 proposal=#%d: %v", proposalID, err)
		return false
	}
	log.Printf("[lifecycle] %s", note)
	return true
}

// lessonStatsRefresh 族尸统计面节流刷新（v_lesson_family_stats——
// B 面：template_ref×reason 族级尸账供人工审计直读+regime 喊人
// 带尸检上下文；refresh 败=下拍自愈不拖主拍）。
func (r *Runner) lessonStatsRefresh(ctx context.Context) {
	if time.Since(r.lessonStatsAt) < lessonStatsEvery {
		return
	}
	r.lessonStatsAt = time.Now()
	if _, err := r.pool.Exec(ctx, `REFRESH MATERIALIZED VIEW v_lesson_family_stats`); err != nil {
		log.Printf("[lifecycle] 族尸统计面刷新败（下拍重试）: %v", err)
	}
}
