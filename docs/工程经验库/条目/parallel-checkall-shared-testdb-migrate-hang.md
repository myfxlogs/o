---
schema_version: 2
id: parallel-checkall-shared-testdb-migrate-hang
kind: pitfall
risk: medium
status: verified
keywords: [check_all, go test, O_TEST_PG_DSN, o-postgres-test, 共享测试库, migrate 挂锁, 30m超时, 并行施工]
scope: [工程面, scripts/check_all.sh, o-postgres-test 容器, internal/platform/testutil]
summary: 多 worktree 并行 check_all 时共用 o-postgres-test 容器的 o_test 库——并发 pgstore.Migrate 互持锁致 go test 挂死撞 1800s 闸；开专库 o_test_<wt>+export O_TEST_PG_DSN 隔离即绿
first_seen: 2026-11-06
last_seen: 2026-11-06
evidence_refs:
  - type: ticket
    ref: T-0403
  - type: doc
    ref: scripts/check_all.sh
  - type: doc
    ref: scripts/gc_test_schemas.sh
  - type: doc
    ref: internal/platform/testutil/testutil.go
---

## 病灶

T-0403 收口期 `check_all.sh` go test 全谱撞 `-timeout=1800s` 炸：
`o/internal/earn` TestAutoPurchaseGate 挂于 `pgstore.Migrate` 的
`pgxpool.Exec`（无超时——锁等不到头）。ps 实证同机 t0404/t0409
两 wt 的 check_all 并行在跑——check_all 缺省径把
`O_TEST_PG_DSN` 一律指向共享容器 `o-postgres-test` 的
`o_test` 库（5434），三方并发 Migrate 同库互持表锁=事实死锁，
panic goroutine 堆在 `pgconn.Read`（非连不上，是 query 永等）。

次生坑：`o/internal/platform/events` TestListenSocketStarvationRootFix
为 ~33s 时间敏感受试（PASS 33.15s），三跑并行负载下 29.12s 被裁
`context canceled`——单跑/负载降即绿，判 flaky-under-load 非回归。

## 处置

专库隔离径（本票实证全链绿）：

```bash
docker exec o-postgres-test psql -U o -d o_test \
  -c "CREATE DATABASE o_test_<wt号>"
# timescaledb 扩展随模板自带，无需另建
export O_TEST_PG_DSN="postgres://o:o@localhost:5434/o_test_<wt号>?sslmode=disable"
bash scripts/check_all.sh
```

- check_all 对已设 `O_TEST_PG_DSN` 不覆盖（`-z` 闸）——专库零改机制。
- 时敏包重跑窗收窄=flaky 自愈（go test 缓存使全谱重跑只补红包）。
- 清场：`DROP DATABASE o_test_<wt号>` 随 wt 退役顺手清（或在
  gc_test_schemas.sh 同径扩库级清面——未立票，记此备忘）。

## 证据

- 病跑 panic：`o/internal/earn` 1800s FAIL，goroutine 钉死
  `pgstore.Migrate→pgxpool.Exec→pgconn.Read`；同刻 `ps` 三方
  `go test ./... -timeout=1800s` 在飞（t0403/t0404/t0409）。
- 复跑绿：同 commit 专库径全链绿戳落
  `check-green-aab55864a438e7e31addee429bd3a1ffa9b5e18e`；
  events 单包独跑 33.15s PASS 证 flaky 非码退。
