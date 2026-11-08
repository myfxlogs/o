// depsigfam.go —— T-0447 信号消费册钉定谱（depenum.go enumerateDeps
// 补位链供给件）：两链派生（假想卡 data_needs+template 命名规约）
// 零命中但语义消费在册的 signal kind→模板族登记。谱本体与解析面
// 独立成件——depenum.go 行数棘轮（check-lines >300 硬阻断）拆分径，
// 判据语义零动。
package hunger

import (
	"context"
	"sort"
)

// depSignalFamilies 信号消费册钉定谱（T-0447）——kind→fin_opp_template
// 族 kind（params->>'kind'）登记：两链派生零命中但语义消费在册的信号
// kind 对补位。解析面=族内 enabled+state='enabled' 且在评估全集
// （fin_opp_eval DISTINCT——评估留痕族才是观测对象同 evalTemplates
// 口径）模板逐一入对 signal:<kind>→<template>；族内全禁=kind 仍落
// orphan 照报不误。首谱 basis_ann→basis：
//
//	basis_ann=季度基差年化蒸馏信号（sigdist T-0139/D-0022——链=信号→
//	  假想官→评估器→提案）——消费语义=basis 族评估臂（evalBasis
//	  直读 md_mark_index/md_ticks 原始行自算基差——蒸馏信号=派生面
//	  非事实源，消费侧必过 Fresh 闸不作判定源）；假想官证据附据面
//	  sigdist.SurfaceSignals basis 面→basis_ann 绑定在册。
//	  两链零命中根因：模板名不派生（首词元 quarter/perp 非 basis
//	  前缀）+假想卡 requirements 无 data_needs 键（grammar 文法卡
//	  仅 executor/kind/settle 三键面——卡行 INSERT-only 径存量冻结）。
var depSignalFamilies = map[string]string{
	"basis_ann": "basis",
}

// appendSignalFamilyPairs depSignalFamilies 展开入列（enumerateDeps
// 补位链——depsigfam.go 独立成件=depenum.go 行数棘轮拆分径）：kind
// 不在册=产者缺席对不入列（谱面实扫纪律）；与派生对撞签名跳过
// （登记册只补派生面够不着的链）；登记对照常活性对账，kind 入
// claimed 不落 orphan；族模板零在册=kind 照旧 orphan 如实。
func (s *Sensor) appendSignalFamilyPairs(ctx context.Context, kindSet map[string]bool,
	templates []string, claimed, derived map[string]bool, pairs []depPair) ([]depPair, error) {
	famTemps, err := s.signalFamilyTemplates(ctx, templates)
	if err != nil {
		return nil, err
	}
	fams := make([]string, 0, len(depSignalFamilies))
	for k := range depSignalFamilies {
		fams = append(fams, k)
	}
	sort.Strings(fams)
	for _, kind := range fams {
		if !kindSet[kind] {
			continue
		}
		for _, t := range famTemps[depSignalFamilies[kind]] {
			sig := "signal:" + kind + "→" + t
			if derived[sig] {
				continue
			}
			claimed[kind] = true
			pairs = append(pairs, depPair{
				sig: sig, family: "signal",
				producer: kind, consumer: t})
		}
	}
	return pairs, nil
}

// signalFamilyTemplates depSignalFamilies 解析面：fin_opp_template
// enabled+state='enabled' 行按 params->>'kind' 归族 ∩evalTemplates
// 评估全集（族模板未产评估行=非观测对象不入册——消费册口径同
// evalTemplates 钦定）。族 kind→模板名集定序去重（同名跨 venue
// 行一对只入一次）。
func (s *Sensor) signalFamilyTemplates(ctx context.Context, templates []string) (map[string][]string, error) {
	inEval := make(map[string]bool, len(templates))
	for _, t := range templates {
		inEval[t] = true
	}
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT template, params->>'kind'
		FROM fin_opp_template WHERE enabled AND state='enabled'
		AND params->>'kind' IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]map[string]bool{}
	for rows.Next() {
		var t, k string
		if err := rows.Scan(&t, &k); err != nil {
			return nil, err
		}
		if inEval[t] {
			if out[k] == nil {
				out[k] = map[string]bool{}
			}
			out[k][t] = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	res := make(map[string][]string, len(out))
	for k, set := range out {
		for t := range set {
			res[k] = append(res[k], t)
		}
		sort.Strings(res[k])
	}
	return res, nil
}
