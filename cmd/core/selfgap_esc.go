// selfgap_esc.go —— 求知滞探（业主令#20930 减法 C 批：热域/做梦/muse/焦点
// 陈腐/自证绿五探撤编后本文件只剩 learnGapEscCheck——「求知:」前缀工单
// 滞工单面零触转唤知/learn_gap 升件，钱环驱动件留）。
package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// learnGapEscCheck 求知滞探(求知环拉起缝根修):「求知:」前缀工单=
// 研究席触发④专用票(处置席契不领),但工单面=可沉默的施工面——
// O 环内无「工单→研究席」自发派发件,票能唤起无主的会话空转轨道。
// 判据:OPEN 求知票(未销∧无活在认领)票龄>2h→唤知/learn_gap 升件
// (升件面=保证注意力的面,滞票转升件即接上注意链——goal_idle 同款
// 传唤语义:本件非新需求,是把无主票推回必读流)。处置写到点名票:
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
	for rows.Next() {
		var id int64
		var h float64
		if rows.Scan(&id, &h) == nil {
			stale = append(stale, fmt.Sprintf("#%d(%.0fh)", id, h))
		}
	}
	c.escEdgeUV(ctx, deadSince, "唤知", "learn_gap", "@", len(stale) > 0, func() string {
		return fmt.Sprintf(
			"唤知/learn_gap: %d 张求知票滞工单面零触（%s）——无主外调票转升件:认领干活(题域内外调→digest/canon)∨escalate转研究席∨证伪销;禁裸销",
			len(stale), strings.Join(stale, "、"))
	})
}
