package hunger

import (
	"context"
	"fmt"
)

// monoMinEvals 样本地板——窗内判词量以下单模板浓度=噪声不判
// （低窗期非饥荒语义；统计有效性下限非策略阈不落键）。
const monoMinEvals = 20

// detectMonoFamily ③mono_family 多样性饥荒（工单#25649）：窗内判词集中
// 单模板≥阈 pct=评估光谱塌缩——量饱掩质荒两旧判只见「有没有」不见
// 「是不是只有一种」（dir_perp_mom 97.9% 实证：提案量饱∧因子一）。
// 样本地板以下浓度=噪声不判；键缺/坏 fail-closed 报错同 zero_proposal 键径。
func (s *Sensor) detectMonoFamily(ctx context.Context, rep *Report) error {
	monoDays, err := s.cfg.GetInt("dm.hunger.mono_family_days")
	if err != nil {
		return fmt.Errorf("多样性窗键不可读: %w", err)
	}
	if monoDays <= 0 {
		return fmt.Errorf("多样性窗非正（%d——键值坏 fail-closed）", monoDays)
	}
	monoPct, err := s.cfg.GetInt("dm.hunger.mono_family_pct")
	if err != nil {
		return fmt.Errorf("多样性阈键不可读: %w", err)
	}
	if monoPct <= 0 || monoPct > 100 {
		return fmt.Errorf("多样性阈域外（%d——键值坏 fail-closed）", monoPct)
	}
	rows, err := s.pool.Query(ctx, `SELECT template,count(*) FROM fin_opp_eval
		WHERE ts>now()-make_interval(days=>$1) GROUP BY template`, monoDays)
	if err != nil {
		return fmt.Errorf("多样性面读败: %w", err)
	}
	var top string
	var topN, total int
	for rows.Next() {
		var t string
		var n int
		if err := rows.Scan(&t, &n); err != nil {
			rows.Close()
			return fmt.Errorf("多样性行读败: %w", err)
		}
		total += n
		if n > topN {
			topN, top = n, t
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("多样性面读败: %w", err)
	}
	if total >= monoMinEvals && topN*100/total >= int(monoPct) {
		rep.Triggered = true
		rep.Reasons = append(rep.Reasons,
			fmt.Sprintf("mono_family:%s(%d%%/%d判词@%dd窗)", top, topN*100/total, total, monoDays))
	}
	return nil
}
