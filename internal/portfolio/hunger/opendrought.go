// opendrought.go —— 开仓断流探针（业主令直修/D案：零产出探针自带归因卡）。
// 判据：fin_position.max(opened_at) 龄 > dm.hunger.open_drought_days（键缺
// =fail-closed 不探非病灶）。火时升件 note 直接带漏斗归因卡——「为什么
// 开不了仓」由探针的确定性查询先行给出，归因不再依赖会话空手走查：
//
//	候选宇宙鲜票率 / eval 提案拒谱 / 拒因 top / paper-probe-live-在仓计数 /
//	断点推定（阈内机读派生：上游层/本金层/评估层/闸门层——判归席不归探针，
//	推定只是事实的阈值化标记）。
//
// 去重同族款（esc_class+esc_scope 24h 同窗至多一报）。只报不修。
package hunger

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// droughtEscClass 升件类（判类册同例自由词——产线断流类升件）。
const droughtEscClass = "产线断流"

// droughtDedupHours 同窗去重（silenceDedupHours 同钦）。
const droughtDedupHours = 24

// ProbeOpenDrought 开仓断流探针一拍：断流超阈→归因卡升件；未断流=静默。
func (s *Sensor) ProbeOpenDrought(ctx context.Context) error {
	days, err := s.cfg.GetInt("dm.hunger.open_drought_days")
	if err != nil {
		return fmt.Errorf("断流窗键不可读: %w", err)
	}
	if days <= 0 {
		return fmt.Errorf("断流窗非正（%d——键值坏 fail-closed）", days)
	}
	var last *time.Time
	if err := s.pool.QueryRow(ctx,
		`SELECT max(opened_at) FROM fin_position`).Scan(&last); err != nil {
		return fmt.Errorf("开仓末行读败: %w", err)
	}
	var ageDays float64
	if last == nil {
		ageDays = -1 // 无开仓史=极端断流形
	} else {
		ageDays = s.now().Sub(*last).Hours() / 24
	}
	if last != nil && ageDays <= float64(days) {
		return nil // 未断流——静默
	}

	card, err := s.droughtCard(ctx, ageDays)
	if err != nil {
		return err
	}
	var dup int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class=$1 AND esc_scope=$2
		AND ts>now()-make_interval(hours=>$3)`,
		droughtEscClass, "open_drought", droughtDedupHours).Scan(&dup); err != nil {
		return fmt.Errorf("同窗去重读败: %w", err)
	}
	if dup > 0 {
		return nil
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,$4,$5)`,
		hungerSeat, hungerTicket,
		"开仓断流——"+card+"（探针只报不修——产线归因卡自带，修/裁归席）",
		droughtEscClass, "open_drought"); err != nil {
		return fmt.Errorf("升件行: %w", err)
	}
	return nil
}

// droughtCard 归因卡拼装（全确定性查询——任一段读败整卡不拼，
// fail-closed 走 hungerEscalate 探针自报径，同 ProbeFamilies 族款）。
func (s *Sensor) droughtCard(ctx context.Context, ageDays float64) (string, error) {
	age := fmt.Sprintf("%.1fd", ageDays)
	if ageDays < 0 {
		age = "无开仓史"
	}
	var fresh, total int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FILTER (WHERE ts>now()-interval '1 day'), count(*)
		 FROM (SELECT DISTINCT ON (inst_id) inst_id,ts
		       FROM md_funding ORDER BY inst_id,ts DESC) x`).Scan(&fresh, &total); err != nil {
		return "", fmt.Errorf("鲜票率读败: %w", err)
	}
	var prop, rej int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FILTER (WHERE verdict='proposed'), count(*) FILTER (WHERE verdict='rejected')
		 FROM fin_opp_eval WHERE ts>now()-interval '7 days'`).Scan(&prop, &rej); err != nil {
		return "", fmt.Errorf("eval 谱读败: %w", err)
	}
	var reasons string
	if err := s.pool.QueryRow(ctx,
		`SELECT coalesce(string_agg(split_part(r,' ',1)||'×'||rn,',' ORDER BY rn DESC),'(无)') FROM
		  (SELECT evidence->>'reason' r, count(*) rn FROM fin_opp_eval
		   WHERE verdict='rejected' AND ts>now()-interval '7 days'
		   AND evidence->>'reason' IS NOT NULL
		   GROUP BY 1 ORDER BY rn DESC LIMIT 3) x`).Scan(&reasons); err != nil {
		return "", fmt.Errorf("拒因谱读败: %w", err)
	}
	var paper, probe, live, openNow int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FILTER (WHERE state='paper'), count(*) FILTER (WHERE state='probe'),
		  count(*) FILTER (WHERE state='live'),
		  (SELECT count(*) FROM fin_position WHERE state<>'closed')
		 FROM fin_strategy`).Scan(&paper, &probe, &live, &openNow); err != nil {
		return "", fmt.Errorf("策略池读败: %w", err)
	}
	return fmt.Sprintf(
		"开仓零产 %s|鲜票=%d/%d(%.1f%%)|eval7d=提案%d/拒%d|拒因top=%s|paper=%d probe=%d live=%d 在仓=%d|断点推定=%s",
		age, fresh, total, pct(fresh, total), prop, rej, reasons, paper, probe, live, openNow,
		droughtPoint(fresh, total, prop, rej, reasons, openNow)), nil
}

// droughtPoint 断点推定（阈内派生——顺序=漏斗自上而下，先中先出）：
// 鲜票率<15%=上游枯（采集死∨市场旱——两因同段不分流，分流归席）；
// 拒因首 insufficient_balance=本金层；提案零且拒非零=评估层全拒；
// 提案在而开仓零=闸门层；余=复合未明。
func droughtPoint(fresh, total, prop, rej int, reasons string, openNow int) string {
	if total > 0 && float64(fresh)/float64(total) < 0.15 {
		return "上游层(鲜票率<15%)"
	}
	if strings.HasPrefix(reasons, "insufficient_balance") {
		return "本金层"
	}
	if prop == 0 && rej > 0 {
		return "评估层(提案0/全拒)"
	}
	if prop > 0 && openNow == 0 {
		return "闸门层"
	}
	return "复合/未明"
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b) * 100
}
