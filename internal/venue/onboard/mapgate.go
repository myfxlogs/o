// mapgate.go —— ②映射草拟+③④map 闸/激活拍（T-0379）。
package onboard

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"o/internal/mt5"
	"o/internal/platform/condreq"
)

// ── ②映射草拟（probed→mapped）───────────────────────────────────────

// beatDraft broker 符号表×OKX inst 匹配→map 草案 proposal 件（不落闸——
// 产出物=evidence.onboard.map_draft+condreq payload）。零命中=如实记不跃迁。
func (r *Runner) beatDraft(ctx context.Context, b credFamily, ob map[string]any) (bool, error) {
	syms := strSlice(ob["probed_symbols"])
	universe := r.okxUniverse(ctx)
	if len(syms) == 0 || len(universe) == 0 {
		return false, nil // 探测符号/宇宙缺席——候下拍（不配阈不虚构）
	}
	pairs := map[string]string{}
	var unmatched []string
	for _, inst := range universe {
		base := strings.SplitN(inst, "-", 2)[0]
		if s := mt5.MatchSymbol(syms, base); s != "" {
			pairs[inst] = s
		} else {
			unmatched = append(unmatched, inst)
		}
	}
	login := b.fields["login"]
	if l, _ := ob["login"].(string); l != "" {
		login = l
	}
	draft := map[string]any{"venue": b.venue, "login": login, "pairs": pairs,
		"unmatched": unmatched, "drafted_at": r.now().UTC().Format(time.RFC3339),
		"note": "备选=bridge.xvenue.map.<login> 子键降 dm 类+硬校验（符号须 broker SymbolParams 自报表+swap 可核）——降域另票议"}
	ob["map_draft"] = draft
	if len(pairs) == 0 {
		return false, r.trans(ctx, b.venue, login, "", "草案", ob,
			"零命中——broker 符号表×okx 宇宙无配对（如实记不跃迁）", true)
	}
	// 已有 map 在库不重复落闸（EXR 首案钦定）——闸已满足即免请求盒行。
	if hit, _, _ := r.gateHits(ctx, login, pairs, ob); len(hit) > 0 {
		return true, r.trans(ctx, b.venue, login, "mapped", "草案", ob,
			fmt.Sprintf("草案 %d 对——map 已在库 %d 对，免 condreq 重复落闸",
				len(pairs), len(hit)), false)
	}
	payload, _ := json.Marshal(draft)
	if err := condreq.Request(ctx, r.fin, "permission", b.venue,
		"bridge.xvenue.map", string(payload)); err != nil {
		return false, fmt.Errorf("map 闸请求盒写败: %w", err)
	}
	return true, r.trans(ctx, b.venue, login, "mapped", "草案", ob,
		fmt.Sprintf("草案 %d 对（未配 %d）——condreq 携草案推 O 群候业主单次批",
			len(pairs), len(unmatched)), false)
}

// ── ③④map 闸+激活（mapped→paper）────────────────────────────────────

// beatGate map 落闸检测：bridge.xvenue.map.<login> 子键优先（多账户面）→
// 缺则 bridge.xvenue.map 平铺——草案对内值须 ∈ broker 探测符号自报表
// （备选注硬校验同义——owner 写值不自报=不落闸）。落闸→paper+ops_event；
// 未落=stay（condreq pending 在位不重复发）。
func (r *Runner) beatGate(ctx context.Context, b credFamily, ob map[string]any) (bool, error) {
	draft, _ := ob["map_draft"].(map[string]any)
	pairsRaw, _ := draft["pairs"].(map[string]any)
	pairs := map[string]string{}
	for k, v := range pairsRaw {
		s, _ := v.(string)
		pairs[k] = s
	}
	if len(pairs) == 0 {
		return false, nil
	}
	login, _ := ob["login"].(string)
	if login == "" {
		login = b.fields["login"]
	}
	hit, bad, src := r.gateHits(ctx, login, pairs, ob)
	if src == "" {
		return false, nil // 闸未落——stay（条件请求盒行已在位）
	}
	if len(bad) > 0 {
		return false, r.trans(ctx, b.venue, login, "", "map闸", ob,
			fmt.Sprintf("落闸值越自报表 %v——fail-closed 不激活（%s）", bad, src), true)
	}
	if len(hit) == 0 {
		return false, nil
	}
	// ③map 闸落证实行（四拍各落一行——闸拍与激活拍分行）。
	ob["map_source"] = src
	ob["map_hit"] = hit
	if err := r.trans(ctx, b.venue, login, "", "map闸", ob,
		fmt.Sprintf("%s 落闸证实（命中 %d 对——owner 单次批行）", src, len(hit)), false); err != nil {
		return false, err
	}
	ob["activated_at"] = r.now().UTC().Format(time.RFC3339)
	return true, r.trans(ctx, b.venue, login, "paper", "paper激活", ob,
		"paper 评估开放——admitXvenuePair 提案有机径复用（零新模板零新径）；live 晋升照生命周期判据+live_switch", false)
}

// gateHits map 闸核对（子键 bridge.xvenue.map.<login> 优先→平铺
// bridge.xvenue.map）：草案对 inst 有落值且值 ∈ broker 探测符号自报表=
// 命中须=草案值；值在表外=bad 硬校验拒。src=""=无图（闸未落）。
func (r *Runner) gateHits(ctx context.Context, login string, pairs map[string]string, ob map[string]any) (hit, bad []string, src string) {
	probed := map[string]bool{}
	for _, s := range strSlice(ob["probed_symbols"]) {
		probed[s] = true
	}
	src, xmap := "bridge.xvenue.map."+login, r.xvenueMap(ctx, "bridge.xvenue.map."+login)
	if len(xmap) == 0 {
		src, xmap = "bridge.xvenue.map", r.xvenueMap(ctx, "bridge.xvenue.map")
	}
	if len(xmap) == 0 {
		return nil, nil, ""
	}
	for inst, dv := range pairs {
		mv, ok := xmap[inst]
		if !ok {
			continue
		}
		if !probed[mv] {
			bad = append(bad, inst+"→"+mv)
			continue // owner 写值不在 broker 自报表——硬校验拒（fail-closed）
		}
		if mv == dv {
			hit = append(hit, inst+"→"+mv)
		}
	}
	return hit, bad, src
}

// xvenueMap app_config 键→{okx_inst: mt5_sym}（键缺/形坏=空集不虚构——
// 闸检测空集=未落闸如实）。
func (r *Runner) xvenueMap(ctx context.Context, key string) map[string]string {
	raw, err := r.cfg.Get(key)
	if err != nil {
		return nil
	}
	m := map[string]string{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

// okxUniverse OKX inst 宇宙（okx.symbols owner 册——草案匹配源；缺=空集）。
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

func (r *Runner) cfgStr(ctx context.Context, key string) (string, error) {
	raw, err := r.cfg.Get(key)
	if err != nil {
		return "", err
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", err
	}
	return s, nil
}
