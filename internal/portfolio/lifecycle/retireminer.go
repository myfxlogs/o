// retireminer.go —— 退役验尸产线（fin_0172，业主令 2026-11-13「经验管
// 接通」件①）。病灶：策略退役=最大经验蒸发口——transition 账 2175 死全
// 留痕而 fin_econ_lesson 策略级零产线（26 案全仓位级）；举一反三有矿无
// 料。本件扫「退役无验尸案」的 transition 行→产 pending 案入经济矿
// （kind='retire_postmortem' entity_ref=strategy#<id>——一策略一案
// UNIQUE 幂等）。只产不裁——判词归裁定席（fin_econ_lesson 裁定语义全在
// verdict_ref 前缀，status 恒 verdicted）；35d 窗（史前死案据陈矿值递减）
// +pending<20 自节流（对齐裁定消化节奏不淹矿）+单拍 ≤5+新死优先
// （id DESC——鲜证先喂）。全死径通吃（armed_ttl/paper_ttl/degraded/
// 黑名单/工具径全走 Transition 留痕）——补产幂等天然自愈，写败下拍再喂。
package lifecycle

import (
	"context"
	"log"
)

// retireMinePendingCap 矿面水位闸——pending 验尸案 ≥20 停喂（裁定席消化
// 节奏对齐；判完水位落自续喂，不淹矿面）。
const retireMinePendingCap = 20

// retireMineWindowDays 扫窗——35d 退役补产（史前死案据陈不补；
// 考古价值案由裁定席直查 transition 账，不淹 pending）。
const retireMineWindowDays = 35

// retireLessonMine 退役验尸产线一拍：transition to_state='retired' 行无
// 配套验尸案者产 pending（numbers=策略档+死法+一生账机读面全载）。
// 非致命——记账面同例：产败 log+拒标不拖开户/晋升主拍（下拍自愈补产）。
func (r *Runner) retireLessonMine(ctx context.Context) {
	tag, err := r.pool.Exec(ctx, `INSERT INTO fin_econ_lesson(kind,entity_ref,numbers)
SELECT 'retire_postmortem','strategy#'||t.strategy_id,
	jsonb_build_object(
		'template_ref',s.template_ref,'inst_id',s.inst_id,
		'venue',COALESCE(s.venue,'okx'),'direction',s.direction,
		'notional_usd',s.notional_usd,'from_state',t.from_state,
		'reason',t.reason,'died_at',t.ts,
		'age_days',round(EXTRACT(EPOCH FROM (t.ts-s.created_at))/86400.0,2),
		'proposal_ref',s.proposal_ref,
		'positions',(SELECT count(*) FROM fin_position p WHERE p.strategy_ref=s.id),
		'realized_usd',(SELECT COALESCE(sum(c.amount),0) FROM fin_cashflow c
			JOIN fin_position p2 ON p2.id=c.position_ref
			WHERE p2.strategy_ref=s.id),
		'paper_net_usd',(SELECT COALESCE(sum(f.amount_usd),0)
			FROM fin_paper_fill f WHERE f.strategy_id=s.id),
		'execs',(SELECT count(*) FROM fin_execution e WHERE e.strategy_ref=s.id))
FROM fin_strategy_transition t JOIN fin_strategy s ON s.id=t.strategy_id
WHERE t.to_state='retired'
	AND t.ts>now()-make_interval(days=>$1)
	AND NOT EXISTS(SELECT 1 FROM fin_econ_lesson l
		WHERE l.kind='retire_postmortem'
		AND l.entity_ref='strategy#'||t.strategy_id)
	AND (SELECT count(*) FROM fin_econ_lesson
		WHERE kind='retire_postmortem' AND status='pending')<$2
ORDER BY t.id DESC
LIMIT 5
ON CONFLICT (kind,entity_ref) DO NOTHING`, retireMineWindowDays, retireMinePendingCap)
	if err != nil {
		r.rej.Inc("write_err")
		log.Printf("[lifecycle] 退役验尸产败: %v（下拍补产——幂等径自愈）", err)
		return
	}
	if tag.RowsAffected() > 0 {
		log.Printf("[lifecycle] 退役验尸 pending ×%d（retire_postmortem——策略级死法入经济矿）", tag.RowsAffected())
	}
}
