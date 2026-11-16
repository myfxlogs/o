---
schema_version: 2
id: worker-claim-scan-ticket-events
kind: ops
risk: low
status: observed
keywords: [施工席, 工单, 认领, ops_ticket_desk, dispatchable, 作废, 勿施工, 撤编, capacity挂起]
scope: [ops_ticket_desk, ops_event, scripts/cognition/worker_runbook.md]
summary: 拾桌机读态≠实质态——dispatchable/workable 只查机推导列（未闭∧近2h无认领∧挂起类=capacity不挡拣选），「作废/勿施工/探针已撤编」等实质判词只活在 ops_event 散文行不入视图；认领后动手前必扫该票近期事件行，见作废标记即还票不施工。
first_seen: 2026-11-16
last_verified: 2026-11-16
skills: []
evidence_refs:
  - type: server_fact
    source: postgres
    query: SELECT id,closed,claimed,susp_cls,dispatchable FROM ops_ticket_desk WHERE id=17112
    observed_at: 2026-11-16T14:23:00Z
  - type: commit
    ref: 78cd6cf45
  - type: doc
    ref: scripts/cognition/worker_runbook.md
---

## 事实/症状

施工席按 runbook⓪ 拾序首 OPEN 票 #17112（frame-probe-unit-scope），
认领成功后查证发现：票面要修的 `frame_use_gap`/`selfq_gap` 探针已随
业主令#20930 C 批撤编（master HEAD=78cd6cf45，部署
o-core:78cd6cf452b8@13:00Z，audit_esc.go 零残留）；外席认领行
#20933 明示「所修探针在撤除表内……期间勿施工」，续票 #17136 已销。
即：**票还在桌上可拣，活已死**。

## 根因

`ops_ticket_desk` 是机读视图，`workable` 谓词=未闭∧近 2h 无认领∧
（无挂起∨挂起类=capacity）。capacity 挂起不挡拣选是设计本意
（窗满还票候容量，窗有位即复拣）；但「作废/勿施工/标的物已撤」
这类**实质判词只写在 ops_event 散文 note 行**，视图不索引不推导
——机读态与实质态之间没有桥。

## 处置

认领后、动手前必扫该票事件流：

```sql
SELECT id,seat,ts,event,left(note,200) FROM ops_event
WHERE note ~ '工单#<id>([^0-9]|$)' ORDER BY id DESC LIMIT 10
```

见「作废/勿施工/已撤编/被#X接管/销」类最新处置或认领标记 →
写处置行还票（引证据），不施工不占位。`dispatchable=f` 且非己
认领所致时先自查是否他席在办租约。

## 证据

- 工单#17112 实案：认领行#21227（施工席 14:21Z）→查证→处置行
  作废还票（引 78cd6cf45/#20933/#21118/#17136 closed=t）。
- `ops_ticket_desk` 视图定义：workable 谓词见
  `pg_get_viewdef('ops_ticket_desk')`，susp_cls='capacity' 不排除。

## 反证条件

若 desk 视图日后把实质作废纳入机读推导（如扫事件行作废关键词
置 closed），或工单面改为显式 status 列，本坑失效。

## 适用边界

施工席拾桌流程专属。唤知/处置席本就扫事件流处置信号，不受此限；
本条防的是「机读桌面直接驱动动手」的工单道会话。
