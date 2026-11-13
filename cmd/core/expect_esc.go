// expect_esc.go —— 元探通用形（工单#10605 业主令④B件）：「预期产出 vs 实产」
// 差分不变式面。断流探（healthComps 注册表）只管采集 feed 面;本件管
// 「自己产没人看」的环内生产面——beat/账页件死了无人替它报丧
// （navStale 先例同律:读件「陈值不判」≠「断供不可见」）。
// 声明=(面名,产出谓词,实产询,阈,断供注) 一件一册;探针只述实数,
// 判词归唤知会话 runbook 预期产出档。增面=增册行,勿再写专属探针。
package main

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// expectDecl 生产面期望产出声明——实产询返回窗内实数（标量行数）;
// 窗长按面拍节钦定（≈1.5× 拍长——误报防宽不误判）。
type expectDecl struct {
	name, prod, query string
	need              float64
	note              string
}

// expectDecls 期望产出注册表——量表 beat 五件（20h/20h/20h/6h/20h 拍
// →30h/30h/30h/9h/30h 窗）：beat 内件查询败=静默零行,此前无探看得见。
var expectDecls = []expectDecl{
	{"armed_distance", "量表 armed_distance(20h拍)",
		`SELECT count(*) FROM ops_event WHERE event='量表'
			AND ticket='armed_distance' AND ts>now()-interval '30 hours'`, 1,
		"armedBeat 断供——armed 距阈拒面账页停更"},
	{"owner_ledger", "量表 owner_ledger(20h拍)",
		`SELECT count(*) FROM ops_event WHERE event='量表'
			AND ticket='owner_ledger' AND ts>now()-interval '30 hours'`, 1,
		"ownerLedgerBeat 断供——业主令账+点破率曲线停更"},
	{"profit_daily", "量表 profit_daily(20h拍)",
		`SELECT count(*) FROM ops_event WHERE event='量表'
			AND ticket='profit_daily' AND ts>now()-interval '30 hours'`, 1,
		"profitDailyBeat 断供——日产出账页停更"},
	{"self_state", "量表 self_state(6h拍)",
		`SELECT count(*) FROM ops_event WHERE event='量表'
			AND ticket='self_state' AND ts>now()-interval '9 hours'`, 1,
		"selfStateBeat 断供——冷启动账页停更"},
	{"books_daily", "量表 books_daily(20h拍)",
		`SELECT count(*) FROM ops_event WHERE event='量表'
			AND ticket='books_daily' AND ts>now()-interval '30 hours'`, 1,
		"booksBeat 断供——账房日对账停更"},
	{"pnl_attr_daily", "量表 pnl_attr_daily(20h拍)",
		`SELECT count(*) FROM ops_event WHERE event='量表'
			AND ticket='pnl_attr_daily' AND ts>now()-interval '30 hours'`, 1,
		"pnlAttrBeat 断供——策略盈亏归因账页停更"},
	{"self_aware", "量表 self_aware(6h拍)",
		`SELECT count(*) FROM ops_event WHERE event='量表'
			AND ticket='self_aware' AND ts>now()-interval '9 hours'`, 1,
		"selfAwareBeat 断供——自我意识器官帧停更"},
}

// expectProdEscCheck 期望产出面探——注册表册逐件对拍:实产<阈=升件
// expect_*。值指纹=实数（变值复燃=新现实;销后同值 30d 内不火）。
func (c coreService) expectProdEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	// 部署冷启宽免（工单#13830——升件#13763 self_aware 死窗1m39s 复绿实证：
	// 新册面部署零窗行必先火后自愈=噪音源）。末次部署龄<宽免窗=首产拍
	// 未完本拍全册让位；读败照常评（读径病灶不得借宽免暗灭）;真断供
	// 宽免期后下一拍照升。
	var depTs time.Time
	if err := c.pool.QueryRow(ctx, `SELECT ts FROM ops_event WHERE seat='DEPLOY'
		AND event='部署' AND note LIKE '%o-core:%' AND note LIKE '%rc=0%'
		ORDER BY id DESC LIMIT 1`).Scan(&depTs); err == nil &&
		time.Since(depTs) < 10*time.Minute {
		return
	}
	for _, d := range expectDecls {
		var n float64
		if err := c.pool.QueryRow(ctx, d.query).Scan(&n); err != nil {
			continue // 读件败本册让位（全谱同律）
		}
		d := d
		c.escEdgeUV(ctx, deadSince, "预期产出", "expect_"+d.name,
			strconv.Itoa(int(n)), n < d.need, func() string {
				return fmt.Sprintf(
					"预期产出/expect_%s: %s 窗内实产 %g<期望≥%g——%s（定性归唤知会话预期产出档）",
					d.name, d.prod, n, d.need, d.note)
			})
	}
}
