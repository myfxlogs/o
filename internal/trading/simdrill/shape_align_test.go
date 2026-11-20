// shape_align_test.go —— 执行族册 vs 演习腿形映射对齐机检：execfam
// 册内每模板须有已知腿形（legShapeOf 命中），或落在显式「演习不支持」
// 白名单内——新增族忘登映射=矩阵面缺口静默漏（本件把缺口前置到测试期）。
package simdrill

import (
	"testing"

	"o/internal/platform/excell"
	"o/internal/platform/execfam"
)

// drillUnsupported 演习不支持白名单——册内模板经显式裁定不入矩阵腿形
// 面者在此登记（空=零豁免：全册须有形）。新增豁免须同 commit 写明裁处。
var drillUnsupported = []string{
	// 空=零豁免（棒5c：dir_perp_mom 出册——ShapeOKXSingleLeg 演习件接通）。
	"stake-carry",          // 工单#21612：参考模型族——venue 执行臂未建无可演习腿形，凭据伪造不可为
	"grid_spot",            // 工单#25840：venue 原生 bot 容器族——execution.Executor 腿形语义外（okxgrid 适配器径），无可演习腿形
	"sprd-atomic-basis",    // 工单#26268：venue 原子多腿价差容器族——sprd 原子执行臂未接（凭据 403 在案），无可演习腿形勿虚开
	"stake-carry",          // 工单#21612：参考模型族——venue 执行臂未建无可演习腿形，凭据伪造不可为
	"grid_spot",            // 工单#25840：venue 原生 bot 容器族——execution.Executor 腿形语义外（okxgrid 适配器径），无可演习腿形
	"stable-park-usdg",     // 工单#26272：骨架族——真径=okx finance okusd 申赎非 Executor 腿形，probe→live 结构性拦保持
	"staking-defi-parking", // 工单#26272：骨架族——真径=okx finance staking-defi purchase/redeem 非 Executor 腿形，probe→live 结构性拦保持
	"signal-bot-relay",     // 工单#26272：骨架族——真径=tradingBot signal 适配器非 Executor 腿形，probe→live 结构性拦保持
}

func TestExecfamLegShapeAlign(t *testing.T) {
	exempt := map[string]bool{}
	for _, tmpl := range drillUnsupported {
		exempt[tmpl] = true
	}
	var unshaped []string
	for _, tmpl := range execfam.All() {
		if _, ok := excell.LegShapeOf(tmpl); !ok && !exempt[tmpl] {
			unshaped = append(unshaped, tmpl)
		}
	}
	if len(unshaped) > 0 {
		t.Fatalf("execfam 模板腿形未登记且不在演习不支持白名单: %v", unshaped)
	}
}
