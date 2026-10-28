---
schema_version: 2
id: mirror-upsert-conflict-key-must-match-natural-key
kind: pitfall
risk: high
status: verified
keywords: [projector, fin_opp_eval, ON CONFLICT, 自然键, 跨写者, 宿主迁移, id 发号域]
scope: [projector, mirror, state-table, upsert]
summary: 状态表镜像 ON CONFLICT 只对齐 PK 不对齐唯一约束全集=跨写者同键异 id 必撞（宿主迁移/重灌即发）；冲突判=行身份全集（自然键），id=发号域非身份
first_seen: 2026-10-28
last_verified: 2026-10-28
skills: []
evidence_refs:
  - type: doc
    ref: docs/tasks/T-0211-投影uEval跨写者同键冲突根修.md
---

## 镜像 upsert 冲突判须对齐全表唯一约束（自然键=身份，id=发号）

- **实证**：fin_opp_eval 状态表化后（UNIQUE=(inst_id,template,direction)），
  投影臂 `uEval` 仍 `ON CONFLICT (id)`——OCI 迁期同键异 id 行批插撞
  `fin_opp_eval_key` 崩环 41 次；研究臂上量后 509 行再撞——手工 id 归位
  各救一次，迁一次犯一次的结构性病灶。
- **教训**：表上每多一个唯一约束=镜像多一重冲突面。ON CONFLICT 目标
  不是「主键」是「行身份」——状态表身份=自然键全集，序列 id 只是发号
  域产物；跨写者（新宿主/重灌序列）对同身份发新号是必然事件不是异常。
- **连带面**：FK 引用方（fin_proposal.eval_id）随镜像侧 id 漂移——
  投影时须按身份键回查镜像侧 id 译码，源侧 id 不可直落。
- **判据**：凡「镜像/同步/upsert 写径」改动，先查目标表
  `pg_constraint` 全唯一面——冲突目标须覆盖会被同表写入方触发的
  每一个唯一约束，或显式论证其不可达。
