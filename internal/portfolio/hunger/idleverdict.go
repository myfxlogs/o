// idleverdict.go —— 有理闲置判词质量闸探针（工单O-36962——业主令「钱闲着」
// 机会成本语义化：live_drought/idle_capital 升件的「有理闲置」处置判词
// 必备三字段=停泊基线值+无候选过线原因+到期日，缺任一=判词不合格件）。
// 机判口径：处置行含「有理闲置」且回引 live_drought/idle_capital 升件
// （note 内「#N」锚→ops_event 升件行 esc_class/esc_scope 对拍），
// 三字段词面闸：停泊 / 过线 / 到期|复审|有效期——缺任一=不合格。
// 闸起点=fin_0245 部署日 2026-11-24——此前旧判词不追（判据生效前
// 事实不溯）。只报不修：不合格件聚合升件（esc_class=idle_verdict_gate
// 惧值 65 在册），修/裁归处置席。
package hunger

import (
	"context"
	"fmt"
)

// idleVerdictGateSince 判词质量闸起点（部署日钉死——旧判词不溯）。
const idleVerdictGateSince = "2026-11-24"

// ProbeIdleVerdictGate 判词质量闸一拍：不合格有理闲置判词>0→聚合升件
// （24h 同窗去重）；零=静默。败=上抛归 hungerTick 自报位非静默。
func (s *Sensor) ProbeIdleVerdictGate(ctx context.Context) error {
	var bad int
	var sample string
	if err := s.pool.QueryRow(ctx,
		`WITH refs AS (
		   SELECT v.id vid, (m)[1]::bigint ref_id
		   FROM ops_event v, LATERAL regexp_matches(v.note,'#([0-9]+)','g') m
		   WHERE v.event='处置' AND v.note LIKE '%有理闲置%'
		     AND v.ts > $1::timestamptz),
		 bad AS (
		   SELECT v.id FROM ops_event v JOIN refs r ON r.vid=v.id
		   JOIN ops_event e ON e.id=r.ref_id
		   WHERE e.event='升件' AND (e.esc_class='live_drought'
		     OR (e.esc_class='资金闸' AND e.esc_scope='idle_capital'))
		     AND (v.note !~ '停泊' OR v.note !~ '过线'
		       OR v.note !~ '到期|复审|有效期'))
		 SELECT count(*), COALESCE(min(id)::text,'') FROM bad`,
		idleVerdictGateSince).Scan(&bad, &sample); err != nil {
		return fmt.Errorf("判词质量闸读败: %w", err)
	}
	if bad == 0 {
		return nil
	}
	var dup int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='idle_verdict_gate' AND esc_scope='判词质量'
		AND ts>now()-make_interval(hours=>$1)`,
		droughtDedupHours).Scan(&dup); err != nil {
		return fmt.Errorf("同窗去重读败: %w", err)
	}
	if dup > 0 {
		return nil
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,$4,$5)`,
		hungerSeat, hungerTicket,
		fmt.Sprintf("有理闲置判词质量闸——%d 件判词缺必备字段（停泊基线/无候选过线原因/到期日三缺一；首件#%s）——O-36962：判词必载停泊=$X@Y%%/yr+无候选过线原因+到期日，不合格件归处置席补正（探针只报不修）",
			bad, sample),
		"idle_verdict_gate", "判词质量"); err != nil {
		return fmt.Errorf("升件行: %w", err)
	}
	return nil
}
