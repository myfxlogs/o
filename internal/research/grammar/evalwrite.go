// evalwrite.go —— 评估行写径拆分件（行数闸拆分——T-0215 volsell.go 同款先例；
// writeEval=全族通用落卡径 T-0205 重飞列级闸在此）。
package grammar

import (
	"context"
	"encoding/json"
	"math"
)

// writeEval 幂等评估行（同 inst+template+reason 1h 窗不重生——节拍防抖）。
func (r *Runner) writeEval(ctx context.Context, inst, template, direction, verdict, reason string, ev map[string]any) error {
	// T-0205①硬闸（重飞·列级口径）：行级不变量=proposed 行 net_apr 列 NOT
	// NULL（fin_0094 DB 闸同维钉死）。evidence 缺 APR 估计（键缺席或值非法）
	// →降级 unevaluable+apr_est_missing；估计在场→同拍落列——首飞闸=键面
	// 被列洞绕行（writeEval 从不写列），闸与不变量同维为本飞根修轴。
	apr, aprOK := evidenceApr(ev)
	if verdict == "proposed" && !aprOK {
		verdict, reason = "unevaluable", "apr_est_missing"
	}
	var aprArg any
	if aprOK {
		aprArg = apr
	}
	ev["reason"] = reason
	b, _ := json.Marshal(ev)
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_eval
		WHERE inst_id=$1 AND template=$2 AND evidence->>'reason'=$3
		AND ts>now()-interval '1 hour'`, inst, template, reason).Scan(&n); err != nil || n > 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES($1,$2,$3,$4,$5,'[]',$6)
		ON CONFLICT (inst_id,template,direction) DO UPDATE SET ts=now(),
		net_apr=EXCLUDED.net_apr,verdict=EXCLUDED.verdict,legs=EXCLUDED.legs,evidence=EXCLUDED.evidence`,
		inst, template, direction, aprArg, verdict, b)
	if err != nil {
		r.rej.Inc("eval_write_err")
	}
	return err
}

// evidenceApr evidence 面 APR 估计提取（net_apr 优先、apr 次之——两键同拍
// 同值小数比率域）；非 float64/NaN/Inf 一律=缺席——首飞「键在值非法不拦」
// 残留窄洞并堵（类型不识=保守降级，不虚构估计）。
func evidenceApr(ev map[string]any) (float64, bool) {
	for _, k := range []string{"net_apr", "apr"} {
		f, ok := ev[k].(float64)
		if !ok || math.IsNaN(f) || math.IsInf(f, 0) {
			continue
		}
		return f, true
	}
	return 0, false
}
