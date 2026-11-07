---
schema_version: 2
id: testdb-stale-advisory-lock-timeout-fanout
kind: pitfall
risk: high
status: verified
keywords: [测试库, advisory lock, pg_advisory_lock, 超时扇出, 600s, 假红, 陈旧事务, schema 堆积, VACUUM FULL, 基线对拍]
scope: [o-postgres-test, go test 全谱, pgstore.Migrate, testutil.Pool]
summary: 共享测试 PG 一旦有陈旧会话/事务持 pg_advisory_lock(860223)（Migrate 闸锁），全谱 go test 逐测试排队等锁——单测试 ~25s、包级 600s 默认超时全线假红（FAIL 无断言红形）；遗留 t_* schema+死元组膨胀放大慢面
first_seen: 2026-11-07
---

## 病灶

共享测试库 `o-postgres-test`（check_all 自举 DSN）上：

- 某次测试进程异常退出后遗留**后端事务未灭**（4 天 16 小时 xact_age），持着
  `pg_advisory_lock(860223)`（pgstore.Migrate 每连接/每 setup 取的闸锁）；
- 之后**所有**走 `pgstore.Migrate` 的包测试逐测试排队等锁：
  - 单测试 ~25s（应亚秒）、包级撞 go test 默认 10m 超时——`FAIL <pkg> 600.0xxs`
    **无任何断言红形**；
  - 全谱并行时连带拖慢不取锁的包（ledger 457s），假象=「环境炸了」；
- 叠加面：229 个遗留 `t_*` 测试 schema（每 run 新建从不回收）+死元组把库
  撑到 10GB——放大慢面但非根因（VACUUM FULL 后基线仍红，锁才是根因）。

## 定位法（master 基线对拍是关键一步）

1. 失败形全为 `FAIL <pkg> 600.0s` 整齐超时、无 `--- FAIL` 断言行 → 先疑环境；
2. **干净 master worktree 单跑同包同红** → 排除本支 diff（基线对拍定责）；
3. 库内取证：
   ```sql
   SELECT pid, now()-xact_start, query FROM pg_stat_activity
     WHERE xact_start < now()-interval '1 hour';            -- 陈旧事务
   SELECT l.pid,a.state,a.query FROM pg_locks l
     JOIN pg_stat_activity a USING (pid)
     WHERE l.locktype='advisory' AND l.granted;              -- 持锁者
   SELECT wait_event_type, count(*) FROM pg_stat_activity
     WHERE wait_event_type='Lock' GROUP BY 1;                -- 等锁队列
   ```

## 处置

```bash
SELECT pg_terminate_backend(<陈旧持锁 pid>);               -- 锁随会话灭
# schema 卫生（无测试在跑时）：
psql -tAc "SELECT 'DROP SCHEMA IF EXISTS '||quote_ident(nspname)||' CASCADE;'
  FROM pg_namespace WHERE nspname LIKE 't\_%'" | psql -q
VACUUM FULL;                                               -- 10GB→15MB
```

## 预防（待立票面）

- 测试库巡检键/脚本：`xact_age > 1h` 持 advisory lock=告警/自动清；
- t_* schema 回收：测试 Teardown 或定期 drop（同上 SQL）；
- go test 全谱跑加 `-timeout` 显式值+失败形「整齐 600s」写进分诊第一眼。
