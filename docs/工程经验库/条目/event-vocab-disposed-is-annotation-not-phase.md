---
schema_version: 2
id: event-vocab-disposed-is-annotation-not-phase
kind: pitfall
risk: medium
status: verified
keywords: [ops_event, 生命周期词集, 处置, 说明类事件, ops_ticket_phase, dm_queue, dispatched, 被踢票, 相位投影]
scope: [scripts/meta/dm_queue.sh, migrations/fin_0147_ops_ticket_phase.sql, ops_event 事件词表, 队列/相位投影类推导]
summary: 事件名直觉≠票相位——「处置」在票域=CEO 对升件告警的处置注记非票本体相位；词集凭名收入生命周期=派单未拉起票被踢出补拉队，词集设计必须先对真实票史尾行取证
first_seen: 2026-11-08
last_verified: 2026-11-08
skills: [协议面取证]
evidence_refs:
  - type: commit
    ref: GW-001 段B fin_0147_ops_ticket_phase.sql
  - type: doc
    ref: docs/infra/GW-001-内循环机械轨-o-mech.md 段B判据①
---

## 症状

GW-001 段B 设计 `ops_ticket_phase` 生命周期投影时，词集初稿凭事件名直觉
把 `处置` 收入侧态相位（处置=终了语义在 dm_handoff 告警面成立）。落地前
对真实票史取证即翻案：被踢票 T-0443/0444/0446/0449/0452 尾行全部同构——

```
派单(CTO)「依赖闸即派」→ spawn rc75 水位满 → 升件「断链:派单」→
处置(CEO)「断链=水位/额度候派在径…调优归CTO」
```

`处置` 的 note 实证=**对升件告警的处置注记**，票本体仍候派未拉起。
处置入相位 → 最新相位=处置 → `dispatched` 判据落空 → 候派票被踢出补拉
队——正是段B要根修的病灶换个位置复发，且判据①钦定形「已批→派单→处置」
直接不成立。

## 根修

- `处置` 归说明类不进相位（fin_0147 词集实证修订在案）；`升件/候水/
  送达/退班/等锁/先例适用/部署` 等旁证行同例。
- 词集设计纪律：**先抽真实票史尾行分布再定词集**——事件名的领域语义
  在不同消费面（告警配对 vs 队列相位）可以不同义，同一事件在
  dm_handoff 是终态、在 dm_queue 是注记。
- 桩测钉法：判据钦定形做成夹具（T-9050 已批→派单→处置→dispatched
  双面同 IN），真票尾行同构形不靠想象造。

## 教训

- 「处置/完成/驳回」类词在告警/配对面=终态，在队列/相位面未必——
  投影词集按消费面语义收，不按事件名字典义收。
- 新 VIEW/投影上线前跑「词集×真实票尾行」对账：哪些票会进/出各队列
  变了，逐类归因（说明类踢队=预期回队 vs 生命周期推进=正确出队）。

## 关联

- GW-001 段B（ops_ticket_phase fin_0147——本坑在案修订）
- dm_queue.sh 头注释（处置归说明类的判据①实证链）
