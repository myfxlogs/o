---
schema_version: 2
id: migration-number-collision
kind: pitfall
risk: high
status: verified
keywords: [迁移序号撞号, migrate_table, worktree 并行, 合并时序, duplicate version]
scope: [migrations/, internal/store/pgstore/migrate_table.go, 并行施工流程]
summary: 施工分支基于旧 master 立迁移号，并行票先合占位→后到者同号双注册（version 无 DB 级唯一约束）——四犯实证；立号前必查远端最新号，合并验收必核登记唯一性
first_seen: 2026-10-25
last_verified: 2026-10-30
skills: [code-audit]
evidence_refs:
  - type: git_fact
    ref: T-0040/T-0041 撞 0025（merge 6cac7f8 改 0026）；T-0044 撞 0026 改 0027；T-0107/T-0108 撞 0046（T-0108 先合占位，T-0107 续编 0047 处置 f5fc216）；T-0259/T-0263 撞 0088（0263 ops_headless 先合，0259 并码时改 0089+票面/条目文引同修——dm/1030-t0259-merge）
---

## 实案（四犯=系统性）

| 犯 | 撞 | 处置 |
|---|---|---|
| T-0040 vs T-0041 | 0025 | merge 6cac7f8 改号 0026 |
| T-0044 | 0026 | 改号 0027 |
| T-0107 vs T-0108 | 0046 | T-0108 先合占位；T-0107 决策席续编 0047（f5fc216） |

共性：施工立号时点是分支起点，合并时点才是占位时点——并行票在途期间
master 序号面已前移，原立号=陈号。schema_migrations.version 无唯一约束，
撞号后果=同号双注册（重复执行/登记漂移静默）。

## 判据

- **施工**：立迁移号前 `git fetch origin && git ls-tree origin/master
  migrations/ | tail` 查远端最新号；施工期长则合并前再查一次。
- **决策席验收**：合并前 `grep -oP '\d+' migrate_table.go 尾段` 核 version
  唯一；撞号处置=后到者续号重编（文件名+登记行+文件头注记三处同改，
  D-651 下走 worktree 径非部署树直改）。
- **机检**（T-0109 落）：pre-commit/CI 断言 migrate_table version 唯一 +
  文件名号↔登记号一致 + 目录文件与登记行一一对应。

## 根因

序号空间是全局共享资源但分配动作分散在各施工分支——无中心分配器时
唯一真源=**合并时点的 master**。票面防撞注记（T-0108 首用）只能提
醒，不能执行；根修=机检硬闸。
