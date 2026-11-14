// Package excell —— simvenue 全周期演习矩阵格集唯一真源（T-0461）。
//
// 格=模板×venue对×方向（机枚举禁手编——占用格自 fin_strategy
// live|paper|probe|degraded ∪ fin_proposal fin_proposed|fin_armed 现查，
// 钦定 xvenue-carry 双向格无条件在册）；面值档=格内最小在产名义+上中位
// 名义两档（零占用钦定格=首单规模 $50 单档）；免演登记双源：①词表
// 零占格 ②paper-only 裁定族占用格（Exempt=true——不产凭据不静默漏）。
// cell_key 机读格式唯一真源：
//
//	template@venuepair/direction#usd=<面值>
//
// venuepair：单所=venue 名（okx）；跨所对=mt5xokx（票面「mt5×okx」ASCII
// 折——x=跨）。消费面双侧同源：simdrill matrix 遍历跑格落 fin_exec_drill
// .cell_key；lifecycle openLive armed→open 闸按本册判格绿 fail-closed。
package excell

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/execfam"
)

// 钦定跨所格（T-0145 CTO 裁定六格之⑤⑥——无生产行也在册：空格按 $50
// 单档跑，跑绿=部署前置凭据；占用格按在产名义两档跑）。
const (
	VenuePairXvenue     = "mt5xokx"
	DirLongPerpShortCFD = "long_perp_short_cfd"
	DirShortPerpLongCFD = "short_perp_long_cfd"
	// XvenueNotional 空占用格钦定面值（票面「首单规模钦定 $50」）。
	XvenueNotional = 50.0
)

// Cell 演习矩阵格（占用格+钦定格+免演登记三源归一）。
type Cell struct {
	Template  string
	VenuePair string
	Direction string
	Occupied  bool   // 有策略/提案占用（false=钦定空格或免演登记）
	Exempt    bool   // 免演登记双源——①词表零占格 ②paper-only 裁定族占用格（不产凭据不静默漏）
	Tiers     []Tier // 面值档（min 在产+上中位；空占用钦定格=$50 单档）
}

// Tier 面值档（Notional=演习名义额；InstID=代表性乐器——占用格取占用行
// inst_id，空占用格="" 由演习侧按 venue 映射解析）。
type Tier struct {
	Notional float64
	InstID   string
}

// CellKey 格键唯一编码（凭据/闸判共用——禁各处自拼）。
func CellKey(template, venuePair, direction string, notional float64) string {
	return fmt.Sprintf("%s@%s/%s#usd=%s", template, venuePair, direction,
		strconv.FormatFloat(notional, 'f', -1, 64))
}

// Keys 格全档 cell_key 集（序定排——断言/审计面防漂移）。
func (c Cell) Keys() []string {
	out := make([]string, 0, len(c.Tiers))
	for _, t := range c.Tiers {
		out = append(out, CellKey(c.Template, c.VenuePair, c.Direction, t.Notional))
	}
	return out
}

// Normalize 占用行→格坐标归一（venue/方向方言折格键——xvenue 副腿行
// venue='mt5' 向位折回 pair 向：short_cfd↔long_perp_short_cfd 对位）。
// 返回 (venuePair, direction)。
func Normalize(template, venue, direction string) (string, string) {
	if template == "xvenue-carry" {
		switch direction {
		case "short_cfd":
			direction = DirLongPerpShortCFD
		case "long_cfd":
			direction = DirShortPerpLongCFD
		}
		return VenuePairXvenue, direction
	}
	if venue == "" {
		venue = "okx"
	}
	return venue, direction
}

// Enumerate 矩阵格集机枚举（禁手编——占用现查+钦定格并集+词表零占免演）。
// 占用源：fin_strategy state∈(live|paper|probe|degraded) ∪ fin_proposal
// state∈(fin_proposed|fin_armed)（模板经 fin_opp_eval 归位）；simdrill
// ephemeral 行 template_ref='simdrill%' 排除——演习自身不产占用（防自激）。
func Enumerate(ctx context.Context, pool *pgxpool.Pool) ([]Cell, error) {
	type occ struct {
		tmpl, vp, dir string
	}
	cells := map[occ]*Cell{}
	// 名义→代表 inst 对（档位键=名义值——同名义异 inst 取首见序定）。
	reps := map[occ]map[float64]string{}
	notions := map[occ][]float64{}
	addCell := func(tmpl, vp, dir string) *Cell {
		k := occ{tmpl, vp, dir}
		c, ok := cells[k]
		if !ok {
			c = &Cell{Template: tmpl, VenuePair: vp, Direction: dir, Occupied: true}
			cells[k] = c
			reps[k] = map[float64]string{}
		}
		return c
	}
	// ① 策略占用（名义+inst 双收——档位代表源）。
	rows, err := pool.Query(ctx, `SELECT template_ref, COALESCE(venue,'okx'),
		COALESCE(direction,''), notional_usd::float8, inst_id
		FROM fin_strategy
		WHERE state IN ('live','paper','probe','degraded')
		  AND template_ref NOT LIKE 'simdrill%'
		ORDER BY CASE state WHEN 'live' THEN 0 WHEN 'paper' THEN 1
		  WHEN 'probe' THEN 2 ELSE 3 END, id`)
	if err != nil {
		return nil, fmt.Errorf("excell 策略占用读败: %w", err)
	}
	for rows.Next() {
		var tmpl, venue, dir, inst string
		var notional float64
		if err := rows.Scan(&tmpl, &venue, &dir, &notional, &inst); err != nil {
			rows.Close()
			return nil, err
		}
		vp, d := Normalize(tmpl, venue, dir)
		k := occ{tmpl, vp, d}
		addCell(tmpl, vp, d)
		notions[k] = append(notions[k], notional)
		if _, ok := reps[k][notional]; !ok {
			reps[k][notional] = inst
		}
	}
	rows.Close()
	// ② 提案占用（模板经 eval 归位——无名义贡献仅占格）。
	prows, err := pool.Query(ctx, `SELECT e.template, p.inst_id, p.direction
		FROM fin_proposal p JOIN fin_opp_eval e ON e.id=p.eval_id
		WHERE p.state IN ('fin_proposed','fin_armed')
		ORDER BY CASE p.state WHEN 'fin_armed' THEN 0 ELSE 1 END, p.id`)
	if err != nil {
		return nil, fmt.Errorf("excell 提案占用读败: %w", err)
	}
	for prows.Next() {
		var tmpl, inst, dir string
		if err := prows.Scan(&tmpl, &inst, &dir); err != nil {
			prows.Close()
			return nil, err
		}
		vp, d := Normalize(tmpl, "", dir)
		k := occ{tmpl, vp, d}
		c := addCell(tmpl, vp, d)
		_ = c
		if _, ok := reps[k][0]; !ok && len(notions[k]) == 0 {
			reps[k][0] = inst // 提案-only 格代表 inst（名义位 0=占位键）
		}
	}
	prows.Close()
	// ③ 档位装配（min 在产+上中位名义两档——同值去重；提案-only 格回退
	// invest.paper.notional_usd 钦定纸面名义，键缺=50 兜底不虚构闸）。
	paperNotional := 50.0
	var pn float64
	if err := pool.QueryRow(ctx,
		`SELECT (value#>>'{}')::float8 FROM app_config WHERE key='invest.paper.notional_usd'`).Scan(&pn); err == nil && pn > 0 {
		paperNotional = pn
	}
	for k, c := range cells {
		ns := notions[k]
		if len(ns) == 0 {
			inst := ""
			if r, ok := reps[k][0]; ok {
				inst = r
			}
			c.Tiers = []Tier{{Notional: paperNotional, InstID: inst}}
			continue
		}
		sort.Float64s(ns)
		seen := map[float64]bool{}
		for _, n := range []float64{ns[0], ns[len(ns)/2]} { // 下界 min+上中位 med
			if seen[n] {
				continue
			}
			seen[n] = true
			inst := ""
			if r, ok := reps[k][n]; ok {
				inst = r
			}
			c.Tiers = append(c.Tiers, Tier{Notional: n, InstID: inst})
		}
	}
	// ③b paper-only 裁定族占用格标免演（真单径未建跑格=凭据伪造不可为；
	// 免演=显式知缺非静默漏——族出 PaperOnly 册即自动转真演习格）。
	for _, c := range cells {
		if execfam.PaperOnly(c.Template) {
			c.Exempt = true
		}
	}
	// ④ 钦定 xvenue 双向格并入（T-0145 CTO 裁定——无占用=$50 单档在册）。
	for _, dir := range []string{DirLongPerpShortCFD, DirShortPerpLongCFD} {
		k := occ{"xvenue-carry", VenuePairXvenue, dir}
		if _, ok := cells[k]; ok {
			continue // 占用格档位已装——不动
		}
		cells[k] = &Cell{Template: "xvenue-carry", VenuePair: VenuePairXvenue,
			Direction: dir, Tiers: []Tier{{Notional: XvenueNotional}}}
	}
	// ⑤ 词表零占免演登记（execfam 册内模板全格集空=免演——不产凭据）。
	occupiedTmpl := map[string]bool{}
	for k := range cells {
		occupiedTmpl[k.tmpl] = true
	}
	for _, tmpl := range execfam.All() {
		if occupiedTmpl[tmpl] {
			continue
		}
		cells[occ{tmpl, "", ""}] = &Cell{Template: tmpl, Exempt: true}
	}
	out := make([]Cell, 0, len(cells))
	for _, c := range cells {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Template != b.Template {
			return a.Template < b.Template
		}
		if a.VenuePair != b.VenuePair {
			return a.VenuePair < b.VenuePair
		}
		return a.Direction < b.Direction
	})
	return out, nil
}

// GreenKeys 全格档最新判词读面（cell_key→latest verdict=='pass'）。
// 缺行/非 pass=false——fail-closed 同构（armed→open 闸消费面）。
func GreenKeys(ctx context.Context, pool *pgxpool.Pool, cells []Cell) (map[string]bool, error) {
	var keys []string
	for _, c := range cells {
		if c.Exempt {
			continue
		}
		keys = append(keys, c.Keys()...)
	}
	green := map[string]bool{}
	if len(keys) == 0 {
		return green, nil
	}
	// 凭据鲜度界（升件#10281 同口径）：ts>=now()-7d——能力债探针凭据窗同界；
	// 无界收跨哨段陈凭据=闸收而侦测报缺互斥根修。陈档=缺档 fail-closed。
	rows, err := pool.Query(ctx, `SELECT DISTINCT ON (cell_key) cell_key, verdict
		FROM fin_exec_drill WHERE cell_key = ANY($1)
		  AND ts >= now() - interval '7 days'
		ORDER BY cell_key, ts DESC, id DESC`, keys)
	if err != nil {
		return nil, fmt.Errorf("excell 格凭据读败: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		green[k] = v == "pass"
	}
	return green, rows.Err()
}

// CellGreen 格绿判（全档 latest=pass 齐绿才真——缺档/任一档败=false
// fail-closed；Exempt 格=true 免演不拦——零占用无提案可放）。
func CellGreen(c Cell, green map[string]bool) bool {
	if c.Exempt {
		return true
	}
	for _, k := range c.Keys() {
		if !green[k] {
			return false
		}
	}
	return true
}

// IndexByCoord 格坐标索引（template|venuepair|direction→Cell——闸判
// 候选行归格消费面）。
func IndexByCoord(cells []Cell) map[string]*Cell {
	out := map[string]*Cell{}
	for i := range cells {
		c := &cells[i]
		out[c.Template+"|"+c.VenuePair+"|"+c.Direction] = c
	}
	return out
}

// CoordKey 坐标键（Normalize 产出对→索引键）。
func CoordKey(template, venuePair, direction string) string {
	return template + "|" + venuePair + "|" + direction
}
