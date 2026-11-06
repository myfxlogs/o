// xshadow.go —— 影子评估层蒸馏臂（T-0408②③）：xvenue 数学扩域——map 外
// 候选对产影子行，同数学同事实域（writeSpread/writeCarry 共享件——防冗余
// 断言禁第二套评估数学）。
//
//	候选对提名=okx.symbols 宇宙×broker 自报目录（鲜 mt5_swap 键——目录
//	  粗扫 mt5/catalog.go 落面）按 base 匹配（mt5.MatchSymbol 单真源——
//	  onboard 草案同匹配器）−bridge.xvenue.map 已映射集：不落闸只产
//	  候选（提名≠判词——发现层≠评估层界，判词归假想官臂）；
//	影子行=独立 kind xvenue_shadow_spread/xvenue_shadow_carry（可辨——
//	  禁混喂真评估/真开闸径，真消费者 kind 精确键天然免疫）；
//	mid 源=md_ticks 鲜行→目录快照 meta.mid 回落（快照 ts=swap 行 ts——
//	  血统闸语义不动，慢源 4h 拍→影行 ts 随之）。
package sigdist

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"o/internal/mt5"
)

// xvenueShadow 影子域一拍（Xvenue 尾调——xm=map 内已映射集）。宇宙/目录
// 缺=让位不虚构；候选零=静默如实（全宇宙已映射或未注册 broker——非病）。
func (r *Runner) xvenueShadow(ctx context.Context, xm map[string]string) error {
	if r.cfg == nil {
		return nil
	}
	universe := r.okxUniverse(ctx)
	if len(universe) == 0 {
		return nil // okx.symbols 缺——影子域让位（fail-closed 不虚构候选）
	}
	// 鲜目录=mt5_swap 源节拍界（8h——粗扫 4h 拍×2 边同根；扫死→目录缩→
	// 影子不产 fail-closed）。
	rows, err := r.pool.Query(ctx, `SELECT key FROM md_signal
		WHERE kind='mt5_swap' AND ts > now()-interval '8 hours'`)
	if err != nil {
		return err
	}
	var catalog []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return err
		}
		catalog = append(catalog, k)
	}
	rows.Close()
	if len(catalog) == 0 {
		return nil // broker 自报目录缺席（桥未起/凭据缺）——让位不虚构
	}
	cand := 0
	for _, inst := range universe {
		if _, ok := xm[inst]; ok {
			continue // map 内——真径照管（不双产）
		}
		base := strings.SplitN(inst, "-", 2)[0]
		mt5Sym := mt5.MatchSymbol(catalog, base)
		if mt5Sym == "" {
			continue // broker 无对应符——非候选
		}
		cand++
		// spread 影行：okx mark 120s 鲜×mt5 mid（tick 鲜行优先→目录快照
		// meta.mid 回落——快照 ts=swap 行 ts 血统钉）。
		okxMark, markTs, okM := r.okxMark(ctx, inst)
		mid, midTs, midSrc, okMid := r.mt5Mid(ctx, mt5Sym)
		if okM && okMid {
			if err := r.writeSpread(ctx, "xvenue_shadow_spread", inst, mt5Sym,
				okxMark, mid, leastTs(markTs, midTs),
				map[string]any{"shadow": true, "mid_src": midSrc}); err != nil {
				return err
			}
		}
		// carry 影行：同 writeCarry 数学（funding_view×mt5_swap 双腿净
		// carry+direction 钉——单边陈/缺=缺席如实）。
		if err := r.writeCarry(ctx, "xvenue_shadow_carry", inst, mt5Sym,
			map[string]any{"shadow": true}); err != nil {
			return err
		}
	}
	log.Printf("[sigdist] xvenue_shadow 拍：%d 候选对（提名≠判词——判词归假想官臂）", cand)
	return nil
}

// mt5Mid 候选符 mid 双源（tick 鲜行优先——瞬订阅/已订阅符正道；缺→目录
// 快照 meta.mid 回落，ts=swap 行观测戳）。返 mid 源标签留痕。
func (r *Runner) mt5Mid(ctx context.Context, mt5Sym string) (mid float64, ts time.Time, src string, ok bool) {
	if mid, ts, ok = r.mt5TickMid(ctx, mt5Sym); ok {
		return mid, ts, "tick", true
	}
	err := r.pool.QueryRow(ctx, `SELECT (meta->>'mid')::float8, ts FROM md_signal
		WHERE kind='mt5_swap' AND key=$1 AND ts > now()-interval '8 hours'`,
		mt5Sym).Scan(&mid, &ts)
	if err != nil || mid <= 0 {
		return 0, ts, "", false // 快照缺/零价——缺席如实
	}
	return mid, ts, "catalog_quote", true
}

// okxUniverse OKX inst 宇宙（okx.symbols owner 册——候选匹配源；缺/坏=
// 空集让位不虚构，onboard.okxUniverse 同键同源）。
func (r *Runner) okxUniverse(ctx context.Context) []string {
	raw, err := r.cfg.Get("okx.symbols")
	if err != nil {
		return nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}
