---
schema_version: 2
id: ops-event-verdict-row-manual-insert-duplicate
kind: pitfall
risk: medium
status: retired
keywords: [ops_event, 裁决行, 驳回, 通过, 重行, meta.task_transition, report.sh, 决策席, append-only, INSERT-only权限]
scope: [ops_event 账本写径, 决策席裁决落行, report.sh 四事件双写面]
summary: report.sh 裁决调用经 meta.task_transition 已同事务落「裁词行」（驳回/通过）——决策席再手 INSERT 同名裁词行=永久重行（ops_event INSERT-only 权限 DELETE 物理被拒）。
first_seen: 2026-11-06
last_verified: 2026-11-06
skills: [KB维护]
evidence_refs:
  - type: server_fact
    source: postgres
    query: SELECT id,ts,seat,event FROM ops_event WHERE id IN (2284,2285,2286)
    observed_at: 2026-11-06T01:36:00Z
  - type: doc
    ref: ~/bin/report.sh（四事件双写段 meta.task_transition）
  - type: doc
    ref: ops_event id 2284/2285（report.sh 单事务裁决+驳回）vs 2286（手 INSERT 重行）
---

## 事实/症状

T-0377 裁决驳回落行：照 T-0375 判例「裁决+驳回双行同 ts」形态，误认
双行=席方手 INSERT 两件，遂 report.sh 裁决后再补一行手 INSERT 驳回
——结果同票三行（裁决×1+驳回×2）。真相：`report.sh 裁决`（note 以
「驳回」起头）在单事务内①INSERT ops_event 裁决行 ②调
meta.task_transition(审计中→驳回)——**转相函数自身落裁词行**，同
ts 双行是这么来的，非席方手写。事后 DELETE 重行：`permission denied
for table ops_event`——append-only 由权限层钉死，误写不可撤。

## 根因或策略解释

1. 判例形态倒推写径——看见「双行同 ts」当惯例复刻，未核双行各自的
   产出者（一在 report.sh SQL 面、一在 PG 函数面）。
2. 双写函数行为不可见于调用方——report.sh 脚本面只见 ops_event
   INSERT 一条，task_transition 的内部落行要追到函数体才见。
3. 无闸拦重行——event 无唯一约束，重行合法入库；INSERT-only 权限使
   事后无修复径。

## 处置

- 裁决落行唯一径=`report.sh 裁决 <T> "驳回——…"/"通过——…"`一次——
  裁决行+裁词行+相位转相+feishu/relay 全带，**禁再手 INSERT 裁词行**；
- 手 INSERT 仅用于裁词外的正交事件（如本席补落的 参数调整 审计行——
  函数不代写的那类）；
- 重行已留账（ops_event#2286）——下游判据只认最新刻事件面不数行数，
  功能无害，不再补救；返修轮次按 docs/prompts/T-XXXX-返修N.md 件数计
  非驳回行数（行数计数面不存在，重行不放大轮次）。

## 证据

- check: ops_event id 2284/2285 同 ts（report.sh 单事务）vs 2286 晚
  3.6s（手 INSERT 自事务）——三行同票驳回语义；
- result: DELETE id=2286 → `permission denied for table ops_event`
  ——重行不可撤实证；
- ref: report.sh 双写段（四事件同事务 ops_event+task_transition）。

## 反证条件

- falsifies_when: meta.task_transition 改为不落 ops_event 行（则
  report.sh 裁决只产单行，判例行形态变、本条作废）——查函数体现状。

## 适用边界

- 全席 report.sh 四事件（领单/收口/移交/裁决）同构——相位转相函数
  落事件面行的形态通用；凡「判例有双行」先查函数体再定手写集。
- ops_event 一切写=append-only——误写预案=写对行前核对产出者链，
  无事后 DELETE 径。
