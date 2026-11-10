// audit_esc.go —— 审计臂/唤知环唤起败+停摆面探针（escLoop 同拍载）。
// 脱自 meta_esc.go 分拆（check-lines 300 棘轮——同域探件不挤一屉）。
package main

import (
	"context"
	"fmt"
	"time"
)

// cognitionSpawnFailEscCheck 唤知环断产面探——唤起败连败=会话壳起不来。
// 2026-11-10 实坑实证：SG 第二 dispatcher 临时 unit env 漂→devin list 败
// →瞬死占跨机锁不释(唤起行在案无败迹)→好机 30min 让位连败×4,交互会话
// 处置行遮蔽 cognitionEscCheck「唤知面死」(其判据=24h 无处置)——探针全盲
// 2h。2h 窗 ≥2 败=结构性(单败=瞬态让位不扰);探针只述败数+末因,定性归
// 唤知会话。败迹源=run.sh EXIT trap 释锁行(dispatch spawn 落败同义)。
// spawnFailTargets 唤起败探针面——seat=败迹写者（dispatcher/run.sh trap）,
// scope=升件域,word=文案词；唤知环与审计臂同构判据同探一件两扫。
var spawnFailTargets = []struct {
	seat, scope, word string
}{
	{"唤知机", "cognition_spawn_fail", "唤知环"},
	{"审计机", "audit_spawn_fail", "审计臂"},
}

func (c coreService) cognitionSpawnFailEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	for _, t := range spawnFailTargets {
		var n int
		var last *string
		if err := c.pool.QueryRow(ctx, `SELECT count(*),
			(SELECT note FROM ops_event
				WHERE seat=$1 AND event='唤起败' ORDER BY id DESC LIMIT 1)
			FROM ops_event WHERE seat=$1 AND event='唤起败'
			AND ts>now()-interval '2 hours'`, t.seat).Scan(&n, &last); err != nil {
			continue // 读件败本拍本面让位
		}
		t := t
		c.escEdgeU(ctx, deadSince, "唤知", t.scope, n >= 2, func() string {
			tail := ""
			if last != nil && *last != "" {
				tail = "（末因 " + *last + "）"
			}
			return fmt.Sprintf(
				"唤知/%s: %s唤起败 2h 窗 %d 连发%s——会话壳起不来(瞬退占锁饿死他机=断产),定性归唤知会话", t.scope, t.word, n, tail)
		})
	}
}

// auditStallEscCheck 审计臂停摆面探——seat='审计席' 产出锚 48h 零行=死。
// NULL=语义空态不死：臂从未产出=装臂前合法态（装臂后首场即落行,落地
// 即进判据面）;非 NULL 陈窗=日拍臂死/会话连败/runbook 坏三因其一。
func (c coreService) auditStallEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	var last *time.Time
	if err := c.pool.QueryRow(ctx, `SELECT max(ts) FROM ops_event
		WHERE seat='审计席' AND event IN ('审计','审计疑点','升件')`).Scan(&last); err != nil {
		return // 读件败本拍让位
	}
	dead := last != nil && time.Since(*last) > 48*time.Hour
	c.escEdgeU(ctx, deadSince, "唤知", "audit_stall", dead, func() string {
		return fmt.Sprintf(
			"唤知/audit_stall: 审计臂 48h 零产出（末审计行 %s 陈 %s）——日拍臂死/会话连败/runbook 坏三因其一,定性归唤知会话",
			last.UTC().Format("01-02 15:04Z"), time.Since(*last).Round(time.Hour))
	})
}
