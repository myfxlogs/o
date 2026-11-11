// armed_beat.go —— armed 距阈量表探（escLoop 同拍载，业主令 2026-11-11）：
// 「N 件 armed、各自差阈多远」定期落 ops_event 量表行——哑弹探只管
// 「不开仓」存在性，本件答「差多少」：拒件证据面（fin_eval_reject 含
// gross/cost/safety）逐 armed 取末拒，毛收/需阈比值近 1=近开仓。
// 发射闸=值指纹变（拒面新现实）∨ ≥20h 基线（表鲜度心跳）——
// 常设态照 escEdgeUV 值指纹契约不 churn；探测席/量表 行 relay digest
// 推飞书=日行营运报告。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

// armedBeatEvery 基线重报窗（指纹同也重报——表鲜度心跳）。
const armedBeatEvery = 20 * time.Hour

// armedRow 一 armed 策略的距阈照。
type armedRow struct {
	inst   string
	reason string  // 末拒因；空=未落拒件（凭据闸/冷窗等不入 fin_eval_reject 面）
	ratio  float64 // 毛收/需阈（breakeven 类才有——近 1=临阈）；负=毛收负
	hasR   bool
}

// armedDistanceBeat 量表拍——armed 集逐取末拒→指纹闸→量表行。
func (c coreService) armedDistanceBeat(ctx context.Context) {
	rows, err := c.pool.Query(ctx, `SELECT s.inst_id,
		COALESCE(r.reason,''), COALESCE(r.evidence::text,'')
		FROM fin_strategy s
		LEFT JOIN LATERAL (SELECT reason,evidence FROM fin_eval_reject e
			WHERE e.inst_id=s.inst_id ORDER BY e.ts DESC LIMIT 1) r ON true
		WHERE s.state='live' AND NOT (s.template_ref='xvenue-carry' AND COALESCE(s.venue,'')<>'okx')
		AND NOT EXISTS (SELECT 1 FROM fin_position p
			WHERE p.strategy_ref=s.id AND p.closed_at IS NULL)
		ORDER BY s.inst_id`)
	if err != nil {
		return // 读件败本拍让位
	}
	defer rows.Close()
	var as []armedRow
	for rows.Next() {
		var a armedRow
		var ev string
		if rows.Scan(&a.inst, &a.reason, &ev) != nil {
			return
		}
		if a.reason == "breakeven_beyond_hold" {
			var e struct {
				Gross  float64 `json:"gross_apr"`
				Cost   float64 `json:"cost_apr"`
				Safety float64 `json:"safety"`
			}
			if json.Unmarshal([]byte(ev), &e) == nil && e.Cost*e.Safety > 0 {
				a.ratio, a.hasR = e.Gross/(e.Cost*e.Safety), true
			}
		}
		as = append(as, a)
	}
	note, fp := armedNote(as)
	if gaugeDue(ctx, c, "armed_distance", fp, armedBeatEvery) {
		if _, err := c.pool.Exec(ctx,
			`INSERT INTO ops_event(seat,event,ticket,note) VALUES('探测席','量表','armed_distance',$1)`,
			note); err != nil {
			log.Printf("[beat] armed距阈写败: %v", err)
		}
	}
}

// armedNote 量表文+值指纹（近阈序——有 ratio 者降序居前）。
func armedNote(as []armedRow) (string, string) {
	hist := map[string]int{}
	for _, a := range as {
		k := a.reason
		if k == "" {
			k = "无拒件"
		}
		hist[k]++
	}
	var hs []string
	for k, n := range hist {
		hs = append(hs, fmt.Sprintf("%s×%d", k, n))
	}
	sort.Strings(hs)
	var rats []armedRow
	for _, a := range as {
		if a.hasR {
			rats = append(rats, a)
		}
	}
	sort.Slice(rats, func(i, j int) bool { return rats[i].ratio > rats[j].ratio })
	var tops, fps []string
	for i, a := range rats {
		fps = append(fps, fmt.Sprintf("%s:%.3f", a.inst, a.ratio))
		if i < 4 {
			tops = append(tops, fmt.Sprintf("%s %.0f%%", a.inst, a.ratio*100))
		}
	}
	for _, a := range as { // 无 ratio 件入指纹（reason 变亦算新现实）
		if !a.hasR {
			fps = append(fps, a.inst+":"+a.reason)
		}
	}
	sort.Strings(fps)
	note := fmt.Sprintf("armed距阈 N=%d | %s", len(as), strings.Join(hs, " "))
	if len(tops) > 0 {
		note += " | 近阈 " + strings.Join(tops, " ")
	}
	fp := fpHash(fps)
	return "v=" + fp + " " + note, fp
}

// fpHash 值指纹（拒面签名——集成员+比值集合的稳定摘要）。
func fpHash(fps []string) string {
	h := uint32(5381)
	for _, s := range fps {
		for _, b := range []byte(s) {
			h = h*33 + uint32(b)
		}
	}
	return fmt.Sprintf("%08x", h)
}

// gaugeDue 量表发射判（armed_distance/owner_ledger 双探共用——禁第二套
// 纹闸）：末行同 ticket 量表指纹异 ∨ ≥every 陈 → 报；同纹且鲜 → 抑。
func gaugeDue(ctx context.Context, c coreService, ticket, fp string, every time.Duration) bool {
	var last string
	var ts time.Time
	if err := c.pool.QueryRow(ctx, `SELECT note,ts FROM ops_event
		WHERE event='量表' AND ticket=$1 ORDER BY id DESC LIMIT 1`, ticket).
		Scan(&last, &ts); err != nil {
		return true // 无末行=首报
	}
	if !strings.HasPrefix(last, "v="+fp+" ") {
		return true // 指纹异=新现实
	}
	return time.Since(ts) >= every
}
