// persist.go —— 分类读面+态账/族提案落表面（dirmom/ctx 同款读写件；
// ops_event 写径全走 escPool fin 池——md 池无 ops_event 面）。
package regime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// bars 末 n 根确认棒 OHLC 序（旧→新；不足 n 返实得数——分类消费
// h/l/c 三件全取）。
func (r *Runner) bars(ctx context.Context, venue, inst string, n int) ([]kline, time.Time, int, error) {
	rows, err := r.pool.Query(ctx, `SELECT h,l,c,ts FROM md_klines
		WHERE venue=$1 AND inst_id=$2 AND bar=$3 AND confirm
		ORDER BY ts DESC LIMIT $4`, venue, inst, bar, n)
	if err != nil {
		return nil, time.Time{}, 0, err
	}
	defer rows.Close()
	var bs []kline
	var last time.Time
	first := true
	for rows.Next() {
		var b kline
		var ts time.Time
		if err := rows.Scan(&b.h, &b.l, &b.c, &ts); err != nil {
			return nil, time.Time{}, 0, err
		}
		if first {
			last, first = ts, false
		}
		bs = append([]kline{b}, bs...) // 倒序→正序（PREPEND——n 小无常数忧）
	}
	return bs, last, len(bs), rows.Err()
}

// regimeOrder dominant 破平定序（等值取固定序首——确定性非 map 序）。
var regimeOrder = []string{"trend", "chop", "quiet", "volatile", "mixed"}

// dominantRegime 占比众数（零分类→""——态账如实记 n=0 不虚构 dominant）。
func dominantRegime(counts map[string]int) string {
	best, bestN := "", 0
	for _, rg := range regimeOrder {
		if counts[rg] > bestN {
			best, bestN = rg, counts[rg]
		}
	}
	return best
}

// regimeMap 映射表读（app_config research.regime.map JSON 对象——
// regime→候选族册钉值 SSOT④；缺键/坏 JSON=error 上抛 fail-closed
// 不提案，态账不受影响）。
func (r *Runner) regimeMap() (map[string][]string, error) {
	raw, err := r.cfg.Get("research.regime.map")
	if err != nil {
		return nil, err
	}
	var m map[string][]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("research.regime.map 解析败: %w", err)
	}
	return m, nil
}

// coverage 族覆盖读（SSOT⑤：fin_opp_template 在册模板集+fin_opp_eval
// 近 thin_days 卡数 per template——两读同池 md 正本面）。
func (r *Runner) coverage(ctx context.Context, thinDays int) (map[string]bool, map[string]int, error) {
	reg := map[string]bool{}
	rows, err := r.pool.Query(ctx, `SELECT template FROM fin_opp_template`)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			rows.Close()
			return nil, nil, err
		}
		reg[t] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	evalN := map[string]int{}
	rows, err = r.pool.Query(ctx, `SELECT template,count(*) FROM fin_opp_eval
		WHERE ts>now()-make_interval(days=>$1) GROUP BY template`, thinDays)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var t string
		var n int
		if err := rows.Scan(&t, &n); err != nil {
			return nil, nil, err
		}
		evalN[t] = n
	}
	return reg, evalN, rows.Err()
}

// writeLedger 态账行（event='态账'——每拍落占比证据；seat=态评估本臂
// 席；unevaluable=thin+stale 跳过台数如实记 SSOT⑩）。
func (r *Runner) writeLedger(ctx context.Context, counts map[string]int,
	classified, thinN, staleN int, dominant string, p Params) error {
	var b strings.Builder
	for _, rg := range regimeOrder {
		share := 0.0
		if classified > 0 {
			share = float64(counts[rg]) / float64(classified)
		}
		fmt.Fprintf(&b, "%s=%.2f/%d ", rg, share, counts[rg])
	}
	// n=宇宙对数——counts 只载五态键，用 classified+unevaluable 合成总数。
	total := classified + thinN + staleN
	note := fmt.Sprintf("态账 n=%d classified=%d thin=%d stale=%d dominant=%s 占比 %slookback=%d",
		total, classified, thinN, staleN, dominant, b.String(), p.Lookback)
	_, err := r.escPool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'态账','',$2)`,
		seat, note)
	return err
}

// maybePropose 族提案一拍（SSOT②⑥）：dominant regime 映射族逐一对覆盖面
// ——未注册=缺族、注册而近 thin_days 卡数<thin_days=稀薄；缺口聚一行
// 族提案（note=占比+缺族+建议）。映射表外/空册=fail-closed 不提案。
// 销抑：未销 族提案@regime_map_gap 在案→抑（anomaly.go 销抑同契——
// 后序行 note 含「族提案#<id>」=销）；票抑：同 scope 开工单在案→抑
// （anomaly.go 票抑 SQL 同契——销票/挂票过期=抑撤自然复火）。
func (r *Runner) maybePropose(ctx context.Context, m map[string][]string,
	dominant string, counts map[string]int, classified int, p Params) error {
	fams, ok := m[dominant]
	if !ok || len(fams) == 0 {
		return nil // 映射表外族/空册——fail-closed 不提案
	}
	reg, evalN, err := r.coverage(ctx, p.ThinDays)
	if err != nil {
		return fmt.Errorf("族覆盖读败: %w", err)
	}
	var gaps []string
	for _, f := range fams {
		if !reg[f] {
			gaps = append(gaps, "缺族="+f+"(映射表在册,fin_opp_template 未注册)")
			continue
		}
		if evalN[f] < p.ThinDays {
			gaps = append(gaps, fmt.Sprintf("稀薄族=%s(近%dd卡数=%d<%d)", f, p.ThinDays, evalN[f], p.ThinDays))
		}
	}
	if len(gaps) == 0 {
		return nil // 覆盖足——零提案
	}
	var open bool
	if err := r.escPool.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM ops_event e WHERE e.event='族提案' AND e.esc_scope='regime_map_gap'
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
			AND x.note ~ ('族提案#'||e.id::text||'([^0-9]|$)')))`).Scan(&open); err != nil {
		return fmt.Errorf("销抑读败: %w", err)
	}
	if open {
		return nil // 未销族提案在案——同窗至多一报
	}
	var tick int
	if err := r.escPool.QueryRow(ctx, `SELECT count(*) FROM ops_event t
		WHERE t.event='工单' AND t.ts>now()-interval '30 days'
		AND t.note ~ ('(^|[^a-zA-Z0-9_])scope='||$1||'([^a-zA-Z0-9_]|$)')
		AND (NOT EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
				AND d.id>t.id AND d.note ~ ('工单#'||t.id||'([^0-9]|$)') AND d.note !~ ('工单#'||t.id||' ?接续'))
			OR (t.ts + COALESCE((substring(t.note from '期限=([0-9]+)h'))::int,24)*interval '1 hour' > now()
				AND EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
					AND d.id>t.id AND d.note ~ ('工单#'||t.id||'([^0-9]|$)')
					AND d.note LIKE '%挂起条件=%')))`,
		"regime_map_gap").Scan(&tick); err != nil {
		return fmt.Errorf("票抑读败: %w", err)
	}
	if tick > 0 {
		return nil // 开工单在案——票管催,探针静默（读败不抑=fail-open 升件优先）
	}
	var b strings.Builder
	for _, rg := range regimeOrder {
		if counts[rg] > 0 {
			share := float64(counts[rg]) / float64(classified)
			fmt.Fprintf(&b, "%s=%.0f%% ", rg, share*100)
		}
	}
	note := fmt.Sprintf("dominant=%s 占比 %sclassified=%d 缺口 %s 建议=人维护映射表补族评估臂/查在册族判词稀薄根因（影子产出零执行——提案升件喊人）",
		dominant, b.String(), classified, strings.Join(gaps, ";"))
	_, err = r.escPool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('研究席','族提案','',$1,'商业产出','regime_map_gap')`, note)
	return err
}
