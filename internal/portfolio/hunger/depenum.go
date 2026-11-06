// depenum.go —— T-0394 依赖对枚举面（depblind.go 对账拍的供对侧）：
// 枚举器实扫注册表/归一链产依赖对（零硬编码清单——票面红线），
// 谱面清单落账供人工基线。枚举源：
//
//	①信号对=kind→eval template 两链派生（0384 同款假想卡 data_needs
//	  注册表链+template 命名规约链）+两链皆不属的 orphan kind；
//	②腿-行情对=legset.ActiveSetUnion 同源表达式∩订阅宇宙；
//	③表读写对/④配置对 deferred（票面钦可选臂——family 槽预留）。
package hunger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"o/internal/platform/legset"
)

// enumerateDeps 依赖对枚举（实扫注册表/归一链——零硬编码清单）：
// ①信号对=kind→eval template 两链派生+消费册外 orphan kind；②腿对=
// legset.ActiveSetUnion∩订阅宇宙。③④ deferred（票面钦——见 depblind.go）。
func (s *Sensor) enumerateDeps(ctx context.Context) ([]depPair, error) {
	var pairs []depPair
	kindSet, err := s.signalKindSet(ctx)
	if err != nil {
		return nil, fmt.Errorf("信号对 kind 谱读败: %w", err)
	}
	needs, err := s.hypothesisNeeds(ctx)
	if err != nil {
		return nil, fmt.Errorf("信号对假想卡需求读败: %w", err)
	}
	templates, err := s.evalTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("信号对模板全集读败: %w", err)
	}
	claimed := map[string]bool{}
	for _, t := range templates {
		for _, k := range deriveKinds(t, kindSet, needs) {
			claimed[k] = true
			pairs = append(pairs, depPair{
				sig: "signal:" + k + "→" + t, family: "signal",
				producer: k, consumer: t})
		}
	}
	var orphans []string
	for k := range kindSet {
		if !claimed[k] {
			orphans = append(orphans, k)
		}
	}
	sort.Strings(orphans)
	for _, k := range orphans {
		pairs = append(pairs, depPair{
			sig: "signal:" + k + "→∅", family: "signal",
			producer: k, consumerAbsent: true})
	}
	legs, err := s.depLegKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("腿对键集读败: %w", err)
	}
	for _, inst := range legs {
		pairs = append(pairs, depPair{
			sig: "leg:" + inst, family: "leg",
			producer: inst, consumer: inst})
	}
	return pairs, nil
}

// depUniverse 订阅宇宙（okx.symbols 名单——腿对所需键的合法性边界；
// '*' 放形 fail-closed 让位：宇宙扩件=venue REST 实读闭包，本探针不虚构）。
func (s *Sensor) depUniverse() (map[string]bool, error) {
	raw, err := s.cfg.Get("okx.symbols")
	if err != nil {
		return nil, err
	}
	var all []string
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, fmt.Errorf("okx.symbols JSON 解析: %w", err)
	}
	out := make(map[string]bool, len(all))
	for _, sym := range all {
		if sym == "*" {
			return nil, errors.New("okx.symbols=* 宇宙扩件探针未挂——fail-closed 不虚构标的集")
		}
		out[sym] = true
	}
	return out, nil
}

// depLegKeys 腿-行情对所需键集（归一链=legset.ActiveSetUnion 同源表达式
// ——订阅集/镜像谓词同串禁第二套；∩订阅宇宙+IsUSDTInst 闸=派生合法性滤，
// 虚构键宇宙外不入所需集——同 syncActiveSet 加订闸判据语义）。
func (s *Sensor) depLegKeys(ctx context.Context) ([]string, error) {
	universe, err := s.depUniverse()
	if err != nil {
		return nil, fmt.Errorf("宇宙读败: %w", err)
	}
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT inst_id FROM (`+legset.ActiveSetUnion+`) t`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		if legset.IsUSDTInst(k) && universe[k] {
			out = append(out, k)
		}
	}
	return out, rows.Err()
}

// depCensus 枚举器产出对清单落账（首跑人工基线材料——「依赖对清单」行
// 携全谱对签名；谱面不变不重复落行——最新行 note 全等比对，零膨胀）。
func (s *Sensor) depCensus(ctx context.Context, pairs []depPair) error {
	sigs := make([]string, 0, len(pairs))
	for _, p := range pairs {
		sigs = append(sigs, p.sig)
	}
	sort.Strings(sigs)
	note := fmt.Sprintf("pairs=%d sigs=%s", len(pairs), strings.Join(sigs, ","))
	var last string
	if err := s.pool.QueryRow(ctx, `SELECT COALESCE((SELECT note FROM ops_event
		WHERE event='依赖对清单' ORDER BY id DESC LIMIT 1),'')`).Scan(&last); err != nil {
		return err
	}
	if last == note {
		return nil // 谱面未变——不重复落行
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'依赖对清单',$2,$3)`,
		hungerSeat, depblindTicket, note)
	return err
}
