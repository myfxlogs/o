// pulse.go —— eval 活动脉冲桥（T-0485）：md 正本 fin_opp_eval per-template
// 聚合（谓词城外全量面）→SG md_eval_pulse 同键 upsert。rejected-only 族
// 镜像谓词下结构性恒零（D-0030 §三承重墙不动），SG 消费面（族沉默探针/
// eval_zero 量表/eval 段灯）活性判据改读本表——脉冲行新鲜=活、陈旧=真死。
// 量级=模板数（~10 行）。写域=projector 现径（SG 镜像写面），零新开写权。
package projector

import (
	"context"
	"fmt"
	"time"
)

const qEvalPulse = `SELECT template, max(ts), count(*) FILTER (WHERE ts>now()-interval '24 hours')
	FROM fin_opp_eval GROUP BY template`

const uEvalPulse = `INSERT INTO md_eval_pulse(template,last_eval_ts,eval_n_24h,refreshed_at)
	VALUES($1,$2,$3,now())
	ON CONFLICT (template) DO UPDATE SET last_eval_ts=EXCLUDED.last_eval_ts,
		eval_n_24h=EXCLUDED.eval_n_24h, refreshed_at=EXCLUDED.refreshed_at`

// refreshEvalPulse 脉冲聚合写（eval 拍尾+活性对账两径顺带——eval 链死期
// ev_eval 无帧，对账 10min 期界兜底保 eval_n_24h/真死语义不漂）。
func (r *Runner) refreshEvalPulse(ctx context.Context) error {
	rows, err := r.md.Query(ctx, qEvalPulse)
	if err != nil {
		return fmt.Errorf("eval 脉冲聚合: %w", err)
	}
	defer rows.Close()
	var batch [][]any
	for rows.Next() {
		var template string
		var last time.Time
		var n int
		if err := rows.Scan(&template, &last, &n); err != nil {
			return err
		}
		batch = append(batch, []any{template, last, n})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return batchExec(ctx, r.sg, uEvalPulse, batch)
}
