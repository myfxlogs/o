---
schema_version: 2
id: test-shard-second-pool-must-pin-database
kind: pitfall
risk: high
status: verified
keywords: [O_TEST_SHARDS, testutil, 分库, o_test_s, ShardDatabase, 第二池, advisory锁, T-0468]
scope: [工程面, internal/platform/testutil, internal/venue/onboard, cmd/core]
summary: T-0468 族内分库后，凡绕 testutil.Pool 直读 O_TEST_PG_DSN 自建第二池的测试必须钉 cfg.ConnConfig.Database=testutil.ShardDatabase()——schema 落分库 o_test_s<N>，第二池连基库 o_test=schema 找不到断言必败；pg_advisory_lock 锁域按库隔离（锁标含 databaseOid）是分库消 herd 的机制根
first_seen: 2026-11-08
last_seen: 2026-11-08
evidence_refs:
  - type: ticket
    ref: T-0468
  - type: doc
    ref: internal/platform/testutil/testutil.go
  - type: doc
    ref: internal/venue/onboard/writefail_test.go
  - type: doc
    ref: cmd/core/pipeline_face_test.go
---

## 坑面

`testutil.Pool` 在 `O_TEST_SHARDS>1` 时按调用方包目录 fnv1a32 散列把
schema 落到 `o_test_s1..sK` 分库（`pg_advisory_lock(860223)` 等 advisory
锁的锁域含 databaseOid——跨库同键互不阻塞，实测 32ms vs 同库 2028ms
lock_timeout，T-0468 G5 桩证）。**基库 `o_test` 不再是默认落点**。

凡测试「先 `testutil.Pool` 拿 schema，再 `os.Getenv("O_TEST_PG_DSN")`
自建第二池（异用户/异 search_path）」的配对面，第二池若不钉
`cfg.ConnConfig.Database = testutil.ShardDatabase()` 就连回基库——
schema 在分库里，基库查无此 schema=断言必败。T-0468 实证两处存量
命中：`internal/venue/onboard/writefail_test.go`（deny 角色池）、
`cmd/core/pipeline_face_test.go`（拆表后 search_path 钉死池）。

## 判据/修法

- 新增「Pool+裸 DSN 第二池」测试：第二池必钉
  `cfg.ConnConfig.Database = testutil.ShardDatabase()`（须由测试文件
  直调——runtime.Caller 取帧=测试文件目录，与 Pool 同目录同库）。
- `ShardDatabase()` 契约：测试文件直调 + 先有 Pool/等价确保（分库
  未建时裸连败——正常序 fixture 先行）。
- grep 面：`O_TEST_PG_DSN` 出现处若邻近 `ParseConfig`+自定义
  User/Password/AfterConnect=疑似第二池，逐一核。
- K<=1（O_TEST_SHARDS 缺省/非法）时 ShardDatabase() 恒回 `o_test`——
  旧行为零回归，钉库写法向后兼容。

## 证据

- T-0468 桩测 G3/G6：两处钉库断言+pgstore 端到端分库实建
  （o_test_s6）绿；不钉则 writefail deny 池连 o_test 查无
  t_onboard_* schema 必败（机理直证）。
