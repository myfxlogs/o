---
schema_version: 2
id: docker-pg-readiness-pg-isready-too-early
kind: pitfall
risk: medium
status: verified
keywords: [docker, postgres, pg_isready, entrypoint 临时服务器, POSTGRES_DB, docker inspect 空行, 就绪判, shell]
scope: [docker postgres/timescaledb 容器自动化, 起停编排脚本, bash]
summary: PG 容器两连坑——①pg_isready 在 entrypoint 临时服务器相即绿（POSTGRES_DB 未建完即放写径撞 init 并发）；②docker inspect 失物仍吐空行使 `|| echo` 产出 "\n<词>" 穿 case 分支。就绪判须真连目标库 select 1，状态判须空串兜底。
first_seen: 2026-10-30
last_verified: 2026-10-30
skills: [KB维护]
evidence_refs:
  - type: commit
    ref: T-0266
  - type: doc
    ref: scripts/testdb.sh
---

## 现象（T-0266 每席测试库机化实建两坑连踩）

**坑① inspect 失物空行穿 case**：`st=$(docker inspect -f '{{.State.Status}}' <缺席容器> 2>/dev/null || echo missing)`——docker inspect 对不存在对象 stdout 仍吐一空行，`|| echo` 追加后 `st`="\nmissing"，case 匹配不上 missing 落 `*` 分支走 `docker start` → "No such container" 假败。

**坑② pg_isready 早于 POSTGRES_DB 就绪**：docker-entrypoint 起临时服务器相 pg_isready 已返绿，此刻 POSTGRES_DB=o_test 尚未由 init 脚本建出；即放 `DROP DATABASE o_test` 扑空（NOTICE skipping）+ `CREATE DATABASE` 与 init 并发撞 `pg_database` 唯一键（duplicate key violates pg_database_datname_index）。

## 修法/防法

- 状态判：inspect 失败输出为空串而非干净错误——`st=$(docker inspect ... || true); st=${st:-missing}`，禁 `|| echo` 直接追加。
- 就绪判：pg_isready 只证「server 收连接」不证「目标库存在」——自动化写径前须 `psql -d <目标库> -tAc 'select 1'` 真连成功才放（T-0266 scripts/testdb.sh ensure_container 已按此修）。
- 通式：凡「等容器就绪再动手」的编排，就绪判口径=业务目标真实可读写，非存活探针。

## 证据

- check: T-0266 无头施工实跑 `testdb.sh alloc` 首轮——e slot 新容器就绪判用 pg_isready 过闸后 reset_db 撞上述 ERROR 原文；改真连判后同径秒过。
- result: 修复后 alloc 全生命周期（起库/租约/重置/回收）实测绿，双席并发盖戳双绿零锁阻。
