// audit_esc.go —— 审计臂/唤知环唤起败+停摆面探针（escLoop 同拍载）。
// 脱自 meta_esc.go 分拆（check-lines 300 棘轮——同域探件不挤一屉）。
package main

import (
	"context"
	"fmt"
	"strconv"
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
		c.escEdgeUV(ctx, deadSince, "唤知", t.scope, strconv.Itoa(n), n >= 2, func() string {
			tail := ""
			if last != nil && *last != "" {
				tail = "（末因 " + *last + "）"
			}
			return fmt.Sprintf(
				"唤知/%s: %s唤起败 2h 窗 %d 连发%s——会话壳起不来(瞬退占锁饿死他机=断产),定性归唤知会话", t.scope, t.word, n, tail)
		})
	}
	c.spawnOrphanEscCheck(ctx, deadSince) // 唤起孤儿探同拍（唤知面同域;feed_esc 棘轮满线不入册唤点）
	c.frameUseEscCheck(ctx, deadSince)    // 器官闭环账探同拍（同上不入册）
}

// spawnOrphanEscCheck 唤起孤儿探——唤起行超 3h 无 unit 配对果/败/收口=
// 会话吊死无 trap 落行∨宿主亡（SIGKILL/宕机写不了行=残余黑洞,本探即
// 该黑洞的升件径;自我意识「连自己死了不知死」死角补位,自观帧卡旗同源）。
// 三栏判据:
//
//	宽限 3h——RuntimeMaxSec 2h 上限+写行滞后余量(48h 实测最长合法会话
//	108min,3h 外零合法在飞);24h 回看界——孤儿出窗自然转绿销案;
//	自锚——只判「unit 收口契约活后出生」的唤起(锚=首条带 unit= 的
//	会话收口行 ts;契约缺席时锚=now() 判集恒空=探针自闭,链修前老格式
//	唤起永远配不上对不冤判)。
//
// 二元死态 fp="@"——回合销后再孤=新回合照火。
func (c coreService) spawnOrphanEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	var n int
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event e
		WHERE e.event='唤起' AND e.ts>now()-interval '24 hours'
		AND e.ts<now()-interval '3 hours'
		AND e.ts>(SELECT COALESCE(min(x.ts),now()) FROM ops_event x
			WHERE x.event='会话收口' AND x.note ~ 'unit=[^ ]+')
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
			AND x.event IN ('唤起果','唤起败','会话收口')
			AND x.note LIKE '%'||substring(e.note from 'unit=[^ ]+')||'%')`).
		Scan(&n); err != nil {
		return // 读件败本拍让位
	}
	c.escEdgeUV(ctx, deadSince, "唤知", "spawn_orphan", "@", n > 0, func() string {
		return fmt.Sprintf(
			"唤知/spawn_orphan: %d 唤起超 3h 无配对果/败/收口——会话吊死或宿主亡无 trap 落行,认知环自死不知死（自观帧卡旗同源,此探=升件径）", n)
	})
}

// frameUseEscCheck 器官闭环账探——用帧契(runbook收口契「用帧=<键|无>」)两档:
//
//	frame_use_gap:   契活后唤知收口行缺 用帧= 字段=契约漂移(会话没接新契);
//	frame_use_zero:  契行≥3 而全数 用帧=无=器官产出零引用——
//	  「帧产无人用=器官空转」升件径,用法账=器官迭代唯一输入面(无探针
//	  盯=该账又是「写了没人读」表,本探即其消费面)。
//
// 自锚同孤儿探——锚=首条载 用帧= 的唤知收口行;契未上线判集恒空探针自闭。
// 3d 窗:漂移/零引行 append-only 出窗自然转绿销案。
func (c coreService) frameUseEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	var n, missing, zero int
	if err := c.pool.QueryRow(ctx, `SELECT count(*),
		count(*) FILTER (WHERE e.note !~ '用帧='),
		count(*) FILTER (WHERE e.note ~ '用帧=无')
		FROM ops_event e
		WHERE e.event='会话收口' AND e.seat='唤知'
		AND e.ts>now()-interval '3 days'
		AND e.ts>(SELECT COALESCE(min(x.ts),now()) FROM ops_event x
			WHERE x.event='会话收口' AND x.seat='唤知' AND x.note ~ '用帧=')`).
		Scan(&n, &missing, &zero); err != nil {
		return // 读件败本拍让位
	}
	compl, used := n-missing, n-missing-zero
	c.escEdgeUV(ctx, deadSince, "唤知", "frame_use_zero", "@", compl >= 3 && used == 0, func() string {
		return fmt.Sprintf(
			"唤知/frame_use_zero: 3d 窗唤知契行 %d 条全 用帧=无——器官产出零引用(帧产无人用=器官空转),器官迭代输入面断供,判词面定性归唤知会话", compl)
	})
	c.escEdgeUV(ctx, deadSince, "唤知", "frame_use_gap", "@", missing > 0, func() string {
		return fmt.Sprintf(
			"唤知/frame_use_gap: 用帧契活后唤知收口 %d 行缺用帧=字段——契约漂移(runbook 收口格式被绕,会话未接新契)", missing)
	})

	// 自问字段同款自锚 drift 核(零激励缝根修的牙——「必留题」软义务
	// 已改收口必填字段,缺字段=回潮):锚=首条载 自问= 的唤知收口行,
	// 契未上线判集恒空探针自闭。
	var sqAnchor *time.Time
	var sqMissing int
	if err := c.pool.QueryRow(ctx, `SELECT
		(SELECT min(x.ts) FROM ops_event x WHERE x.event='会话收口'
			AND x.seat='唤知' AND x.note ~ '自问='),
		(SELECT count(*) FROM ops_event e WHERE e.event='会话收口'
			AND e.seat='唤知' AND e.ts>now()-interval '3 days'
			AND e.note !~ '自问='
			AND e.ts>(SELECT COALESCE(min(x.ts),now()+interval '1 year')
				FROM ops_event x WHERE x.event='会话收口' AND x.seat='唤知'
				AND x.note ~ '自问='))`).Scan(&sqAnchor, &sqMissing); err != nil {
		return
	}
	c.escEdgeUV(ctx, deadSince, "唤知", "selfq_gap", "@", sqAnchor != nil && sqMissing > 0, func() string {
		return fmt.Sprintf(
			"唤知/selfq_gap: 自问契活后唤知收口 %d 行缺自问=字段——零激励缝回潮(收口格式被绕,自问账断供)", sqMissing)
	})
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
	var inst *time.Time
	if last == nil {
		// NULL 只在「臂未装」时是合法空态——审计机唤起行（唤器 spawn 成功
		// 落账）在案而产出锚仍零行=装臂后死（升件#9572）。
		if err := c.pool.QueryRow(ctx, `SELECT max(ts) FROM ops_event
			WHERE seat='审计机' AND event='唤起'`).Scan(&inst); err != nil {
			return
		}
		dead = inst != nil && time.Since(*inst) > 48*time.Hour
	}
	c.escEdgeUV(ctx, deadSince, "唤知", "audit_stall", "@", dead, func() string {
		var anchor string
		if last != nil {
			anchor = fmt.Sprintf("末审计行 %s 陈 %s",
				last.UTC().Format("01-02 15:04Z"), time.Since(*last).Round(time.Hour))
		} else {
			anchor = fmt.Sprintf("装臂证 %s（陈 %s）而零审计产出",
				inst.UTC().Format("01-02 15:04Z"), time.Since(*inst).Round(time.Hour))
		}
		return fmt.Sprintf(
			"唤知/audit_stall: 审计臂 48h 零产出（%s）——日拍臂死/会话连败/runbook 坏三因其一,定性归唤知会话",
			anchor)
	})
}
