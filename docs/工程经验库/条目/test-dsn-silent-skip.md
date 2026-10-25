---
schema_version: 2
id: test-dsn-silent-skip
kind: pitfall
risk: high
status: verified
keywords: [O_TEST_PG_DSN, 桩测 skip, 夹具债, t.Skip, 静默跳测, fixture]
---

# 桩测 DSN 缺席静默 skip——夹具债潜伏通道

## 现象

`O_TEST_PG_DSN` 未设时 `testutil.Pool(t)` 走 `t.Skip`——`go test` 输出
「ok/No tests found」一片绿，**真实一条没跑**。坏夹具（schema 漂移、
新闸拒旧夹具值）在仓库里潜伏数票无人觉。

## 实证

- T-0111 breakeven 闸合入时本地无 DSN 全 skip；T-0124 验收期带 DSN 实跑，
  opener_test 三案齐红（夹具 carry 45bps 撞 48bps 新闸）——债潜伏三票。
- 同 session `go test` 无 DSN 输出「No tests found」与「真过」不可分。

## 修法/防法

- **验收/交付跑测必须显式带 DSN**：
  `O_TEST_PG_DSN="postgres://o:o@localhost:5434/o_test?sslmode=disable" go test ./...`
  ——跑前 `docker ps` 确认 o-postgres-test 在。
- skip 不算过：包内 0 run 0 fail 的「绿」按未验处理，复审期认
  `-v` 输出或 `-json` 的 pass 计数。
- 收口纪律（业主令 2026-10-25 起）：**坑不排队——有坑立即写条目，
  不待归纳**（待归纳层=下个 session 的盲区）。

## 证据

T-0111 尾注/T-0124 验收回执（3 案 breakeven_beyond_hold 拒旧夹具）。
