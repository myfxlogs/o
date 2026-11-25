// desklife.go —— 票桌生命周期探针（工单O-39400——「只进不出」票桌
// 根修细胞，hungerTick 同槽 ProbeBreedStall 后）：
//
// ①delivered-sweep 交付实据全域销：未销∧未挂起∧非裁定票存在契形回执
//
//	（完工|候审|施工收口|工单完工|回执事件族 ∨ 处置行@sha——谓词与
//	land_queue.sh CLOSESET 逐字同源）→ 落地件同形销行
//	（seat=探测席 处置「工单#<id> 销——…」→ops_act_parse act='close'→
//	desk.closed 派生真）。CLOSESET 只在列车落成拍对当批票号集生效，
//	交付实据早于机制/走旁路落成的票永挂——本件=同谓词全域连续版。
//
// ②premise-death 前提死亡（O-39410 锚优先径——判据码在
// desklife_premise.go）：anchor_ref='venue:<v>' 结构化硬引用→
// candidate.state='rejected' 硬终态直销/锚指对象缺席升件；无锚票
// note 解析兜底只升件不直销；retired template_ref 命中→升件交裁。
//
// ③decay 陈腐衰减阀：deadline+dm.hunger.desk_decay_days 零活动
//
//	（last_active<deadline——死后无动作）→ desk_stale 升件一次
//	（无窗去重=单次，不刷屏）。
//
// 禁误杀硬条款：claimed/rulable/susp_until 非空票一律不碰（在办/待裁/
// 挂起面归原径）；delivered-sweep 与 CLOSESET 同闸（不查 claimed——
// 交付实据在=工作已完，claim 语义已兑现）。fail-closed：
// dm.hunger.desk_decay_days 缺/非正=报错不探（迁移 fin_0254 落键）。
package hunger

import (
	"context"
	"fmt"
)

const (
	deskPremiseEscClass = "premise_dead"
	deskStaleEscClass   = "desk_stale"
)

// ProbeDeskLifecycle 票桌生命周期一拍：三面顺序扫，单面败上抛归
// hungerTick 自报位（同 ProbeBreedStall 败情隔离约定）。
func (s *Sensor) ProbeDeskLifecycle(ctx context.Context) error {
	days, err := s.cfg.GetInt("dm.hunger.desk_decay_days")
	if err != nil {
		return fmt.Errorf("票桌衰减窗键不可读: %w", err)
	}
	if days <= 0 {
		return fmt.Errorf("票桌衰减窗非正（%d——键值坏 fail-closed）", days)
	}
	if err := s.deskDeliveredSweep(ctx); err != nil {
		return err
	}
	if err := s.deskPremiseDeath(ctx); err != nil {
		return err
	}
	return s.deskDecay(ctx, days)
}

// deskDeliveredSweep ①交付实据销——谓词与 land_queue.sh CLOSESET 的
// EXISTS 支逐字同源（完工|候审|施工收口|工单完工|回执 事件族 ∨
// note 头 40 字内完工回执/完工推支候审/候审回执/施工收口 ∨
// 处置行^O-<id>…@<sha>；复审过 sha=/复审席/落地件 豁免；票号三源
// O-<id>/工单#<id>/ticket 列）。全域版=不限当批 TSET。
func (s *Sensor) deskDeliveredSweep(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `SELECT d.id FROM ops_ticket_desk d
		WHERE d.closed IS NOT TRUE AND d.susp_until IS NULL AND d.rulable IS NOT TRUE
		  AND EXISTS (SELECT 1 FROM ops_event r
		    WHERE ((r.event ~ '完工|候审|施工收口|工单完工|回执'
		      OR r.note ~ '^.{0,40}(完工回执|完工推支候审|候审回执|施工收口)'
		      OR (r.event='处置' AND r.note ~ '^O-[0-9]+[^@]*@[0-9a-f]{9,40}'))
		      AND r.note !~ '^复审过[[:space:]]*sha=' AND r.seat NOT IN ('复审席','落地件')
		      AND (r.note ~ ('O-'||d.id||'([^0-9]|$)') OR r.note ~ ('工单#'||d.id||'([^0-9]|$)')
		        OR r.ticket ~ ('^(O-|工单#)?'||d.id||'$'))))`)
	if err != nil {
		return fmt.Errorf("交付实据面读败: %w", err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("交付实据扫败: %w", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("交付实据迭代败: %w", err)
	}
	for _, id := range ids {
		if _, err := s.pool.Exec(ctx,
			`INSERT INTO ops_event(seat,event,ticket,note)
			 VALUES($1,'处置','工单',$2)`,
			"落地件",
			fmt.Sprintf("工单#%d 销——交付实据销（契形回执/处置@sha 在案——desklife全域版）", id)); err != nil {
			return fmt.Errorf("交付实据销行写败 id=%d: %w", id, err)
		}
	}
	return nil
}
