// ops.go —— 投影执行面（proposal/eval/macro/回写/帧分派）：批量 upsert（SG
// 镜像）+反向回写+反向桥唤醒分派。水印推进使 ev_* 重复唤醒零代价（空集即返）；
// ts 判桥表在 tsops.go。
package projector

import (
	"context"
	"fmt"
	"time"
)

const (
	propIDOverlap = 500              // 提案 id 水位回读重叠（窗内 upsert 幂等）
	evalOverlap   = 10 * time.Minute // eval ts 水位回读重叠（对齐 replayTsTables 惯例）
)

// mdWake PG-md 侧帧分派（ev_*→对应桥表谓词切片投影）。
func (r *Runner) mdWake(ctx context.Context, ch string) error {
	switch ch {
	case "ev_ticks", "ev_markidx", "ev_funding", "ev_signal", "ev_dcd":
		if ch == "ev_funding" { // T-0301/T-0303：台账+两幽灵表桥频道顺带（小表全量镜像——ev_funding 帧节拍顺带搬运零成本）
			if err := r.projectVenueCandidates(ctx); err != nil {
				return err
			}
			if err := r.projectGhostTables(ctx); err != nil {
				return err
			}
			if err := r.projectRegime(ctx); err != nil { // T-0312 第九桥顺带
				return err
			}
			if err := r.projectTwoTier(ctx); err != nil { // T-0401 两级采集制桥面对顺带
				return err
			}
		}
		return r.projectAllTs(ctx) // T-13230：唤醒名只作叫醒源——全 ts 道过闸（cap1 塌陷道饿死根修）
	case "ev_eval":
		return r.projectEvals(ctx)
	case "ev_proposal":
		return r.projectNewProposals(ctx)
	case "": // boot 首拍（无 channel）——水印零值全量过一遍谓词面
		if err := r.projectAllTs(ctx); err != nil {
			return err
		}
		if err := r.projectVenueCandidates(ctx); err != nil {
			return err
		}
		if err := r.projectGhostTables(ctx); err != nil {
			return err
		}
		if err := r.projectRegime(ctx); err != nil { // T-0312 boot 首拍顺带
			return err
		}
		if err := r.projectTwoTier(ctx); err != nil { // T-0401 boot 首拍顺带
			return err
		}
		return r.projectEvals(ctx)
	}
	return nil
}

// sgWake SG 侧帧分派（活跃集刷+proposal 回写+反向桥 diff）。
func (r *Runner) sgWake(ctx context.Context, ch string) error {
	switch ch {
	case "ev_active_set":
		if err := r.refreshSets(ctx); err != nil {
			return err
		}
		return r.projectMacro(ctx) // ~30 行全表 diff——廉价顺带（事件级表无独立写帧）
	case "ev_proposal":
		return r.repairProposalStates(ctx)
	case "ev_subreq": // T-0412 意图晋级帧（fin_0128 触发器——user_intent 行变即发）
		if err := r.refreshSets(ctx); err != nil { // 谓词集先纳新提名符——其后 tick 即过桥
			return err
		}
		return r.reverseSubRequests(ctx)
	case "ev_strategy": // 反向桥帧（fin_0028 在册——策略行变即发，T-0172）
		if err := r.reverseStrategies(ctx); err != nil {
			return err
		}
		return r.reverseFinMirrors(ctx) // T-0326：五 fin 副本群顺带（小表全表 diff 廉价——无 SG 触发帧随既有帧径+期界兜底）
	case "config_changed":
		return r.syncConfig(ctx)
	case "":
		if err := r.refreshSets(ctx); err != nil {
			return err
		}
		return r.syncConfig(ctx)
	}
	return nil
}

// repairProposalStates 反向回写（SG state 列组=正本——D-0030 写权矩阵）：
// SG 活跃/近期退役行对拍 PG-md，三列差即回写。前向投影不写 state 列组=
// 不触发 SG ev_proposal_sg（UPDATE OF state 按 SET 清单）——零回环构造性成立。
func (r *Runner) repairProposalStates(ctx context.Context) error {
	sgRows, err := r.sg.Query(ctx, `SELECT id,state,retire_reason,retired_at FROM fin_proposal
		WHERE state IN ('fin_proposed','fin_armed') OR retired_at>now()-interval '24 hours'`)
	if err != nil {
		return fmt.Errorf("回写面查询: %w", err)
	}
	type st struct {
		state  string
		reason *string
		at     *time.Time
	}
	sg := map[int64]st{}
	defer sgRows.Close()
	for sgRows.Next() {
		var id int64
		var s st
		if err := sgRows.Scan(&id, &s.state, &s.reason, &s.at); err != nil {
			return err
		}
		sg[id] = s
	}
	if err := sgRows.Err(); err != nil {
		return err
	}
	if len(sg) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(sg))
	for id := range sg {
		ids = append(ids, id)
	}
	mdRows, err := r.md.Query(ctx, `SELECT id,state,retire_reason,retired_at FROM fin_proposal WHERE id=ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("回写对拍: %w", err)
	}
	md := map[int64]st{}
	defer mdRows.Close()
	for mdRows.Next() {
		var id int64
		var s st
		if err := mdRows.Scan(&id, &s.state, &s.reason, &s.at); err != nil {
			return err
		}
		md[id] = s
	}
	if err := mdRows.Err(); err != nil {
		return err
	}
	var batch [][]any
	for id, s := range sg {
		if m, ok := md[id]; !ok || m.state != s.state || !reasonEq(m.reason, s.reason) || !timeEq(m.at, s.at) {
			batch = append(batch, []any{id, s.state, s.reason, s.at})
		}
	}
	return batchExec(ctx, r.md, uProposalReverse, batch)
}

func reasonEq(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func timeEq(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}
