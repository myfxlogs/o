// selfgap_esc.go —— 求知滞探（业主令#20930 减法 C 批：热域/做梦/muse/焦点
// 陈腐/自证绿五探撤编后本文件只剩 learnGapEscCheck——「求知:」前缀工单
// 滞工单面零触转唤知/learn_gap 升件，钱环驱动件留）。
package main

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// learnGapEscHotH/learnGapEscTicketN 分级复升阈（工单O-32295 裁案a）——
// 写死常量可调改此：计数带=票龄(2h,learnGapEscHotH]∧滞票<learnGapEscTicketN
// 张→日计照记不升件（#22578①降级语义保留——噪声窗内静默）；任一票龄
// >learnGapEscHotH∨滞票≥learnGapEscTicketN 张→真滞升件（降级全禁言
// 实证#26142滞44h+/32020/32021滞4h+只日计无人知——双盲叠加病灶根修，
// 「什么算真滞」沉为判据非会话判断）。
const (
	learnGapEscHotH    = 12.0 // 单票滞龄升件阈（小时）——可调改此
	learnGapEscTicketN = 3    // 滞票张数升件阈——可调改此
)

// learnGapCountEdge 计数带死沿——与 deadSince 升件态分键：计数带只
// 日计不产恢复行（复绿只清计数沿）；升件态复绿走 escEdgeUV 恢复径。
// 包级原子量同 selfEscMeter 先例（coreService 值接收器不可携锁,
// escLoop 单实例常驻=单写者）。
var learnGapCountEdge atomic.Bool

// learnGapEscCheck 求知滞探(求知环拉起缝根修):「求知:」前缀工单=
// 研究席触发④专用票(处置席契不领),但工单面=可沉默的施工面——
// O 环内无「工单→研究席」自发派发件,票能唤起无主的会话空转轨道。
// 判据:OPEN 求知票(未销∧无活在认领)票龄>2h 分级输出(O-32295):
// 计数带(单票≤12h∧<3张)日计一火不升件(#22578①降级语义保留);
// 热带(任一票>12h∨滞票≥3张)转唤知/learn_gap 升件(升件面=保证
// 注意力的面,真滞票转升件即接上注意链——goal_idle 同款传唤语义:
// 本件非新需求,是把无主票推回必读流)。处置写到点名票:
// 认领干活(题域内外调→digest/canon 交付)∨escalate 转研究席派发
// ∨证伪销票;禁裸销本件。
func (c coreService) learnGapEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	rows, err := c.pool.Query(ctx, `SELECT t.id,
		EXTRACT(EPOCH FROM (now()-t.ts))/3600
	FROM ops_event t
	WHERE t.event='工单' AND t.ts>now()-interval '30 days'
		AND t.note ~ '求知[:：]'
		AND t.ts<now()-interval '2 hours'
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='处置'
			AND x.id>t.id AND x.note ~ ('工单#'||t.id||'([^0-9]|$)'))
		AND NOT EXISTS (SELECT 1 FROM ops_event cl WHERE cl.event='认领'
			AND cl.id>t.id AND cl.ts>now()-interval '2 hours'
			AND cl.note ~ ('工单#'||t.id||'([^0-9]|$)'))
	ORDER BY t.id LIMIT 5`)
	if err != nil {
		return // 读件败本拍让位
	}
	defer rows.Close()
	var stale []string
	hot := 0
	for rows.Next() {
		var id int64
		var h float64
		if rows.Scan(&id, &h) == nil {
			stale = append(stale, fmt.Sprintf("#%d(%.0fh)", id, h))
			if h > learnGapEscHotH {
				hot++
			}
		}
	}
	note := func() string {
		return fmt.Sprintf(
			"唤知/learn_gap: %d 张求知票滞工单面零触（%s）——无主外调票转升件:认领干活(题域内外调→digest/canon)∨escalate转研究席∨证伪销;禁裸销",
			len(stale), strings.Join(stale, "、"))
	}
	if hot > 0 || len(stale) >= learnGapEscTicketN {
		// 热带——learn_gap 已移出降级册(#22578①册件见 selfesc_meter.go
		// O-32295 注),escEdgeUV 走真升件径(值指纹+未销抑簇折叠照旧)。
		c.escEdgeUV(ctx, deadSince, "唤知", "learn_gap", "@", true, note)
		return
	}
	if len(stale) > 0 {
		// 计数带——日计照记不升件:降级语义保留但只盖噪声窗
		// (死沿计一火,复绿清沿——同册内降级件口径)。
		if learnGapCountEdge.CompareAndSwap(false, true) {
			selfEscMeter.bump("唤知/learn_gap")
		}
		return
	}
	learnGapCountEdge.Store(false)
	c.escEdgeUV(ctx, deadSince, "唤知", "learn_gap", "@", false, note) // 复绿——仅升件态产恢复行,计数带静默
}
