---
schema_version: 2
id: checkall-concurrent-test-pg-migrate-stall
kind: pitfall
risk: medium
status: verified
keywords: [check_all, go test, o-postgres-test, 并发争用, Migrate IO wait, 30m 超时 panic, 测试缓存键, -timeout]
scope: [scripts/check_all.sh, internal/store/pgstore, o-postgres-test 容器, worktree 多席并行]
summary: 多席 worktree 并发 check_all 全谱同打一 o-postgres-test——包级 pgstore.Migrate 相互挤兑/锁等，5 包 30m 超时 panic（栈全停 Migrate→Conn.Exec IO wait）属环境 flake 非码因；错峰/降并发复跑即绿，且 -timeout 入测试缓存键——裸 go test 复跑不给 check_all 同款键暖缓存
first_seen: 2026-11-06
last_verified: 2026-11-06
skills: []
evidence_refs:
  - type: doc
    ref: docs/tasks/T-0404-md_ticks化石清道-死符清源.md
  - type: doc
    ref: scripts/check_all.sh
---

## 事实/症状

T-0404 收口期三席 worktree（t0403/t0404/t0405）并发 `check_all.sh`——
`go test ./... -timeout=1800s` 同打一 o-postgres-test 容器（CPU 145%）。
5 包 FAIL 1800s：panic 栈清一色 `pgstore.Migrate → pgx.Conn.Exec →
IO wait`（o/cmd/core·earn·hunger·lifecycle·projector）——包级超时
被前面兄弟测试的 Migrate 锁等耗尽，非测体本身挂。

## 根因

共享测试库×多席全谱并发：每包 testutil.Pool 各自 schema 隔离是行级
干净，但迁移 DDL 面（数百语句×~90 迁移件×N 席同跑）把单 PG 打到
语句级排队——IO wait 表象是锁/CPU 挤兑非真死锁。go test 缓存坑②：
`-timeout` 值入缓存键——裸 `go test ./pkg`（默认 10m）绿果不暖
`check_all` 的 `-timeout=1800s` 键，全谱重跑该包照样重测。

## 处置

- 判 flake：超时包栈停 Migrate IO wait ∧ 同机他席 `go test ./...` 在跑
  （`ps aux | grep "go test"`）→ 环境争用非码因；
- 错峰复跑：等他席全谱落毕重跑 check_all（已过包缓存命中秒过，原 FAIL
  包单跑全绿实证）；
- 若要定向暖缓存：命令旗标须与 check_all 同款 `-timeout=1800s`，否则
  键异白暖。

## 证据

- 同轮三席并发实证：5 包 panic→错峰 `go test -p 1`+单包复跑全绿
  （earn 223s/hunger 117s/lifecycle 180s/projector 308s/core 276s）；
- 二轮 check_all 复跑全链绿戳
  `check-green-e565681a84ae630f5cc57275679e32153dfadb15`。

## 反证条件

若错峰单跑该包仍超时挂同一 Migrate 点=非本坑（真迁移死锁/回归），
照码病流程查。

## 适用边界

本机 worktree 多席共打一 o-postgres-test 场景；CI 独库面/生产面无关。
