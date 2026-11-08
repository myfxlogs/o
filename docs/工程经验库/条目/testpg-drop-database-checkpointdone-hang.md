---
schema_version: 2
id: testpg-drop-database-checkpointdone-hang
kind: pitfall
risk: medium
status: verified
keywords: [test-pg, DROP DATABASE, CheckpointDone, scratch库, 并发测试, 超时, 清库, FORCE]
scope: [scripts/meta/tests/*_test.sh, test-pg 共享池, PostgreSQL 管理面]
summary: 共享 test-pg 忙时 DROP DATABASE 可阻塞在 CheckpointDone 分钟级（checkpointer DataFileSync 排队）——桩测清库走 postgres 维护库+WITH (FORCE)+timeout 包裹，勿在测中途 DROP 在飞库死等
first_seen: 2026-11-08
last_seen: 2026-11-08
evidence_refs:
  - type: commit
    ref: GW-001 段E case_writeback_test.sh
---

## 现象

桩测 scratch 库清理/换库时 `psql -c "DROP DATABASE xxx"` 挂起不返回——
`pg_stat_activity` 见后端 wait_event=`IPC/CheckpointDone`，checkpointer
自身 `IO/DataFileSync`——共享池高负载下 checkpoint 排队，DROP DATABASE
等不到检查点完成。

## 根因

DROP DATABASE 须等 checkpointer 完成挂起的 checkpoint 请求才落盘清理；
多 worktree 并行 check_all/桩测共用同一 test-pg 实例时 IO 排队可达
分钟级——不是连接未断（FORCE 前），是 IO 闸排队。

## 规避（已在案写法）

- 清库连接一律打 **postgres 维护库**（勿连待删库自身——DROP 自库必败
  且会挂起）；
- `DROP DATABASE IF EXISTS <db> WITH (FORCE)`——顺带终结残留连接；
- 整条 psql 包 `timeout 60`——checkpoint 慢不拖死套件，库名带 $$
  PID 天然不撞下轮；
- **勿在测中途 DROP 在飞库**——要空库就 `CREATE DATABASE <db>b`
  另起（CREATE 不等 checkpoint），在飞库留到 trap/末尾清。

## 对拍

`parallel-checkall-shared-testdb-migrate-hang` 同域不同灶——那条是
并发 Migrate 持锁挂死，本条是 DROP DATABASE 等 checkpoint IO。
