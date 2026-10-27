---
schema_version: 2
id: test-schema-per-test-no-gc-instance-suffocation
kind: pitfall
risk: high
status: verified
keywords: [测试, testutil.Pool, schema, GC, o_test, 5434, advisory lock, autovacuum, 套件拖死, out of shared memory]
scope: [test, testutil, CI, o-postgres-test 实例]
summary: testutil.Pool 每测试建 t_<pkg>_<rand> schema 且无 GC——O 实例两日堆 18365 个/32GB，目录膨胀+autovacuum 风暴拖死全仓套件（单包绿/全跑挂超时）；前身 GC 脚本按 arb_* 库命名+arb 角色连接，与 O schema 泳道完全脱钩且每日 auth failed
first_seen: 2026-10-27
last_verified: 2026-10-27
skills: [KB维护]
evidence_refs:
  - type: doc
    ref: /opt/arb-archive/scripts/test_db_gc.sh（前身遗物——arb 角色连接 o_test 5434，cron 04:45 日志连日 psql auth failed；匹配面=arb_* 数据库非 t_* schema）
  - type: server_fact
    source: postgres
    query: SELECT count(*) FROM pg_namespace WHERE nspname LIKE 't\_%'
    observed_at: 2026-10-27T07:05:00Z
    server_revision: timescale/timescaledb:2.30.2-pg18
    response_hash: "18365 schemas / 32GB（清理前实测）"
repro_or_symptom: 单包测试绿（core 66s）、全仓 ./... 并发跑则包级 10m 超时/25 分钟零包完成；PG 侧 pg_advisory_lock(860223)（迁移锁）11+ 等待者、autovacuum ANALYZE 风暴常驻；DO 单事务批量 DROP SCHEMA CASCADE 报 out of shared memory（max_locks_per_transaction 爆）
root_cause: ①testutil.Pool 每测试函数建新 schema（t_<pkg>_<nanos>）用后不弃不清 ②O 换测试拓扑（单库多 schema）后未建对应 GC，前身 test_db_gc.sh 的 arb_* 库匹配面+arb 角色双双脱钩，cron 日跑日败 ③批量清理用单事务 DO 循环=全量锁挤爆 max_locks_per_transaction ④放大器：后台任务「停」只杀包装 shell 不杀进程树——go test 父孤儿化后继续 spawn 包二进制，多套件并发不可见地叠加（本例 15:19/15:44 两套孤儿套件+09:55 六小时僵尸 exe/core 实证）
fix_and_prevention: ①应急=逐条自动提交 DROP（sed 生成逐行 DROP SCHEMA，单 psql 会话顺序跑，锁随语句释放）②根治=testutil 面补 schema GC（testutil 注册名或同前缀+期限清；或每测试 t.Cleanup 自清）——立单交决策 ③全仓套件前查 pg_namespace 计数，>千即先清再跑 ④跑套件前后 `ps aux | grep "go test"` 断言单实例——孤儿父进程不查则「一次跑」实际是 N 次跑
notes: 误诊链：先判「并发资源争用」（部分真）→ 再判「DROP 工人与套件互踩」（真）→ 终判「孤儿 go test 父进程多套件隐形叠加」（主放大器）。诊断顺序应先 `ps aux` 断言进程单例，再谈库与锁
---

## 现象

全仓 `go test ./...` 单包全绿、并发全跑必挂：包级 10m 超时（earn）、或 25 分钟零包完成且 PG 见迁移 advisory lock 十余等待者+autovacuum ANALYZE 风暴。实例 o_test 达 18365 个 `t_*` schema / 32GB。

## 根因或策略解释

testutil.Pool 每测试建 schema 用后不清；O 项目改测试拓扑（前身=多库多席位，O=单库多 schema）后前身 GC 脚本（cron 每日）匹配面与连接角色双双脱钩——日跑日 auth failed，schema 无限堆积。目录与死元组膨胀放大一切迁移/查询成本，并发只加速窒息。

## 处置

应急：单 psql 会话逐条 `DROP SCHEMA ... CASCADE`（自动提交，锁随语句释放——DO 单事务批量会爆 max_locks_per_transaction）。根治（立单交决策）：testutil 面补 schema GC 或 t.Cleanup 自清。

## 证据

- check: `SELECT count(*) FROM pg_namespace WHERE nspname LIKE 't\_%'`
- result: 清理前 18365；逐条 DROP 递减零错；tail /opt/arb-backups/test_db_gc.log 连日 `password authentication failed for user "arb"`
- ref: docs/tasks/T-0174（同 session 实证）

## 反证条件

- check: O 若落地 schema 级 GC（脚本/testutil 自清）后长期计数稳定
- falsifies_when: GC 落地后计数仍无限增长，或证实另有清理机制在本条目盲区

## 适用边界

O 测试拓扑（o-postgres-test 5434 单库多 schema）；前身多库拓扑沿用旧脚本不在此列。逐条 DROP 速率 ~1000 schema/分钟（本实例实测）。
