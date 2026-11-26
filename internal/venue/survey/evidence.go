// evidence.go —— 普查证据落账+升件面（O-41375——evidence.survey 实载+
// onboard.probe.symbol_params 补缺列+ops_event 拍行/升件行；写面与
// 采集面分件=棘轮拆分）。
package survey

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

// markSurvey evidence.survey 子档合并写（attempt/skip/last_error 标记面——
// 只动 survey 键不碰 symbol_params——幂等补缺列语义）。
func (r *Runner) markSurvey(ctx context.Context, venue string, patch map[string]any) error {
	j, _ := json.Marshal(patch)
	_, err := r.md.Exec(ctx, `UPDATE fin_venue_candidate
		SET evidence = evidence || jsonb_build_object('survey',
			COALESCE(evidence->'survey','{}'::jsonb) || $2::jsonb)
		WHERE venue=$1`, venue, j)
	return err
}

// writeEvidence 全量证据落账：survey 子档全载+onboard.probe.symbol_params
// 补缺列（exness 同形 6 字段子集——触发面/下游消费面同列形对齐）。
func (r *Runner) writeEvidence(ctx context.Context, venue string, d *surveyData) error {
	sv := map[string]any{
		"at": r.now().UTC().Format(time.RFC3339), "surveyed_at": r.now().UTC().Format(time.RFC3339),
		"account": d.Account, "symbols_total": len(d.Symbols),
		"symbols": d.Rows, "partial": d.Partial,
		"spread": d.Spread, "swap": d.Swap, "margin_tiers": d.MarginTiers,
		"quote_health": d.QuoteHealth, "family_verdicts": d.FamilyVerdict,
		"verdict_count": d.VerdictCount, "map_draft": d.MapDraft,
	}
	// onboard.probe.symbol_params 补缺列——exness 证据列形同构 6 字段
	// （params 败行跳过不虚构；trade_mode_id→tradeMode int 对齐）。
	params := map[string]any{}
	for sym, row := range d.Rows {
		p := map[string]any{}
		for _, k := range []string{"digits", "spread_pts", "min_lots", "lots_step", "trade_mode_id", "contract_size"} {
			if v, ok := row[k]; ok {
				p[k] = v
			}
		}
		if len(p) == 0 {
			continue
		}
		entry := map[string]any{
			"digits": p["digits"], "spread": p["spread_pts"],
			"minLots": p["min_lots"], "lotsStep": p["lots_step"],
			"tradeMode": p["trade_mode_id"], "contractSize": p["contract_size"],
		}
		params[sym] = entry
	}
	svJSON, _ := json.Marshal(sv)
	paramsJSON, _ := json.Marshal(params)
	_, err := r.md.Exec(ctx, `UPDATE fin_venue_candidate SET evidence =
		evidence
		|| jsonb_build_object('survey', $2::jsonb)
		|| jsonb_build_object('onboard',
			COALESCE(evidence->'onboard','{}'::jsonb) || jsonb_build_object('probe',
				COALESCE(evidence->'onboard'->'probe','{}'::jsonb)
				|| jsonb_build_object('symbol_params', $3::jsonb)))
		WHERE venue=$1`, venue, svJSON, paramsJSON)
	return err
}

// beatRow 普查拍 ops_event 行（venue普查事件——成/败同面落账）。
func (r *Runner) beatRow(ctx context.Context, venue, note string, fail bool) error {
	esc := interface{}(nil)
	if fail {
		esc = "普查败"
	}
	_, err := r.fin.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,$2,$3,$4,$5,$6)`, seatName, eventName, ticketID,
		fmt.Sprintf("beat=普查 key=%s %s", venue, note), esc, venue)
	if err != nil {
		log.Printf("WARN [venuesurvey] 拍行写败 venue=%s: %v", venue, err)
		r.rej.Inc("write_err")
	}
	return err
}

// escalate 证件齐升件行（spec 钦定产物——「venue=<X> 证件齐：N符/
// 可承接族<list>/优缺判词」；臂只备证不拍板——capital/宇宙纳入归红线面）。
func (r *Runner) escalate(ctx context.Context, venue string, d *surveyData) error {
	var able, part, no, noEv []string
	for f, v := range d.FamilyVerdict {
		m, _ := v.(map[string]any)
		switch m["verdict"] {
		case "能":
			able = append(able, f)
		case "部分":
			part = append(part, fmt.Sprintf("%s(%v/%v)", f, m["covered"], m["total"]))
		case "不能":
			no = append(no, f)
		default:
			noEv = append(noEv, f)
		}
	}
	sort.Strings(able)
	sort.Strings(part)
	sort.Strings(no)
	qh := d.QuoteHealth
	partial := ""
	if d.Partial {
		partial = " partial=预算截断"
	}
	note := fmt.Sprintf("venue=%s 证件齐：%d符（partial=%v qh=%v/%v dead=%.0f%%）可承接族 能[%s] 部分[%s] 不能[%s] 无宇宙实证%d%s——资本金/宇宙纳入决定归红线面",
		venue, len(d.Symbols), d.Partial, qh["alive"], qh["sampled"],
		numF(qh["dead_pct"])*100, strings.Join(able, ","), strings.Join(part, ","),
		strings.Join(no, ","), len(noEv), partial)
	if p, ok := d.MapDraft["doc_path"].(string); ok && p != "" {
		note += " doc=" + p
	}
	_, err := r.fin.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,'venue上岗',$4)`, seatName, ticketID, note, venue)
	if err != nil {
		log.Printf("WARN [venuesurvey] 升件写败 venue=%s: %v", venue, err)
		r.rej.Inc("write_err")
		return r.beatRow(ctx, venue, "升件写败（证据已落——"+err.Error()+"）", true)
	}
	return r.beatRow(ctx, venue, fmt.Sprintf("证件齐 %d符 判词族%d——升件已落", len(d.Symbols), len(d.FamilyVerdict)), false)
}
