---
schema_version: 2
id: migration-unique-index-dirty-data
kind: pitfall
risk: critical
status: verified
keywords: [迁移, 唯一索引, 脏数据, dedupe 前步, 23505, 部分索引]
scope: [migrations/, pgstore.Migrate, 上线数据态]
summary: 新唯一索引迁移缺 dedupe 前步——在既有脏数据上 CREATE UNIQUE INDEX 23505 炸；迁移须自带 dedupe 前步或显式决策跳过
first_seen: 2026-10-23
last_verified: 2026-10-23
skills: []
evidence_refs:
  - type: server_fact
    ref: dev 实证 fin_strategy_active_dedupe 在 480+189 重复 paper 行上 23505
  - type: doc
    ref: migrations/0014_strategy_dedupe.sql
---

# 新唯一索引迁移缺 dedupe 前步（T-0024）

## 病灶

`0014_strategy_dedupe` 直接 `CREATE UNIQUE INDEX`——dev 库既有
480+189 同键重复 paper 行→SQLSTATE 23505 迁移炸。
dev 撞过一次，SG 上线同款数据态必复炸。

## 判据（钉死）

- **新增唯一/排他约束的迁移必须自带 dedupe 前步**（先合并/退役重复行
  再建索引），或迁移头注释显式记录「决策跳过 dedupe」——两者皆无=退回。
- 前步范式：`DELETE 保留最新行`或`UPDATE 旧行置 retired`，
  同事务内再 CREATE INDEX。
- 迁移 review 必问：该索引在**既有数据**上能否建成——空库能建≠产线能建。

## 同族教训

- 版本槽撞号（T-0023 0015 与 T-0019 0015 撞）——迁移版本唯一纪律，
  合并冲突时后到者让号重编；dev 库手填 schema_migrations 须与
  migrate_table 登记名一致，否则正牌迁移被已占版本号跳过。
