// evalmap.go —— 提案 eval_id 译码面（T-0211）：自然键 upsert 后 SG eval 行
// 持自有 id（md 发号不落）——fin_proposal.eval_id FK 须按 eval 自然键译
// SG 发号（fin_0053 重指钦定径：提案行自带 inst/direction/net_apr，
// eval 引用=溯源面非载荷）。
package projector

import (
	"context"
	"fmt"
	"log"
)

// remapEvalIDs 提案批 eval_id 译 SG 发号。mdEval=md eval_id→自然键底稿
// （投影前 eval upsert 批的副产）。键缺席=源侧 eval 已删的异常行——
// 跳过+WARN（SG 无行=下轮 id 集差自重试，不毒化整批提案管）。
func (r *Runner) remapEvalIDs(ctx context.Context, mdEval map[int64][3]string, batch [][]any) ([][]any, error) {
	var ins, tpls, dirs []string
	seen := map[[3]string]bool{}
	for _, k := range mdEval {
		if !seen[k] {
			seen[k] = true
			ins = append(ins, k[0])
			tpls = append(tpls, k[1])
			dirs = append(dirs, k[2])
		}
	}
	rows, err := r.sg.Query(ctx, `SELECT u.i,u.t,u.d,e.id FROM
		unnest($1::text[],$2::text[],$3::text[]) AS u(i,t,d)
		JOIN fin_opp_eval e ON e.inst_id=u.i AND e.template=u.t AND e.direction=u.d`,
		ins, tpls, dirs)
	if err != nil {
		return nil, fmt.Errorf("eval_id 译码查询: %w", err)
	}
	sgByKey := map[[3]string]int64{}
	for rows.Next() {
		var i, t, d string
		var id int64
		if err := rows.Scan(&i, &t, &d, &id); err != nil {
			rows.Close()
			return nil, err
		}
		sgByKey[[3]string{i, t, d}] = id
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var kept [][]any
	for _, p := range batch {
		mdID := p[1].(int64)
		k, ok := mdEval[mdID]
		if !ok {
			log.Printf("[projector] 提案 id=%v 引用 eval=%d md 侧缺席——跳过（id 集差自重试）", p[0], mdID)
			continue
		}
		sgID, ok := sgByKey[k]
		if !ok {
			log.Printf("[projector] 提案 id=%v 引用 eval 键 %v SG 侧缺席——跳过", p[0], k)
			continue
		}
		p[1] = sgID
		kept = append(kept, p)
	}
	return kept, nil
}
