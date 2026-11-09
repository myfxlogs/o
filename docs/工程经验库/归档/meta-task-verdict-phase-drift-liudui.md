---
schema_version: 2
id: meta-task-verdict-phase-drift-liudui
kind: pitfall
risk: medium
status: retired
keywords: [meta_task, task_transition, 相位, 审计驳回, 通过误标, 留队, 已立, 候派, stale催办, EDR升级]
scope: [meta仓, mbus-dispatch, SG o-postgres, meta.meta_task, 审计链]
summary: 审计判决落票面不回写 meta_task.phase——相位可滞留『通过』触发自动并码路由；留队处置落点=已立（非候派——候派在 stale 4h 催办面会循环误报）
first_seen: 2026-11-01
last_verified: 2026-11-01
skills: []
evidence_refs:
  - type: commit
    ref: 95e28129
  - type: server_fact
    source: postgres
    query: SELECT ticket,phase,verdict_commit,note FROM meta.meta_task WHERE ticket='T-0118'
    observed_at: 2026-11-01T07:55:00Z
---

## 事实/症状

T-0118 审计链 14:32 相位落『通过』、14:34 票面落驳回判决（commit 95e28129）——判决事件（report.sh 裁决）只落 ops_event 流水与票面『状态』字段，不回写 meta_task.phase，相位滞留『通过』。dispatcher `task_scan_round` 对 `phase='通过' AND kind='o_repo'` 有自动并码路由（`task_scan_pass_one`），滞留相位=错误路由敞口。管理席裁「未处置」→ L1（MANAGE-3990，30min 到点）→ L2（DM-4056）→ obl727 门2 唤起处置。

## 根因或策略解释

判决链与相位机读面异源：票面『状态』=判决权威，meta_task.phase=调度驱动源，二者无自动对账——审计席落票面判决后无人回写相位（同族病灶=T-0292『票面终态但行照派』，派前否决闸只兜已派一径）。

留队态用词坑：票面词『已批』在 meta_task 词表无对应词。`候派`（重设计停泊位）语义虽近，但在 `task_stale_sweep` 扫描名单内——永久泊挂会每 TASK_STALE_TTL（4h）折叠窗后循环发 taskstale 催办 MANAGE，对「持续在跑」母票=噪声源。`已立`不在催办面、无自动消费面（已派→spawn／收口待审→审计／驳回→返修／通过→并码各路由均不可触），是『已批留队』正确落点；误派时票面状态=驳回经 T-0292 否决闸泊回挂起兜底。

## 处置

- 相位失配处置径：`meta.task_transition` CAS 转相（p_from=现相原子守）→已立，verdict_commit 落判决 commit 留痕，note 记裁决出处；边界=不改票面锚文／不重跑审计／不动实现码。
- 根治候选（未落，候裁）：判决落票面同拍回写相位（审计收口脚本加 task_transition 钩），或否决闸覆盖面扩到通过相。

## 证据

- check: `SELECT ticket,phase,verdict_commit,note FROM meta.meta_task WHERE ticket='T-0118'` + ops_event 流水对拍 + `task_scan_round` 消费面 grep `phase='通过'`
- result: T-0118 相位 挂起→已立（verdict_commit=95e28129）2026-11-01 07:54 UTC 在案；ops_event 触发器镜像行同拍自落
- ref: 95e28129（票面驳回判决）；mail 20261101-145009-MANAGE-3990 / 20261101-152011-DM-4056 / 回报 20261101-155456-DM-4121

## 反证条件

- check: 查审计收口链是否已加 task_transition 回写钩、stale 名单词集是否重排
- falsifies_when: 票面判决→相位出现自动对账机制后，「判决不回写相位」前提失效；stale 名单移除候派后落点论证减半

## 适用边界

- o_repo 票调度面；ext 票『通过→待部署』另有 deploy_sweep 直转径同理暴露
- 「已立=留队落点」适用无活跃 owner 的定性保留票；`候派`仍是短泊候重派裁决的正确泊位
