---
schema_version: 2
id: test-shared-db-namespace
kind: pitfall
risk: high
status: verified
keywords: [测试隔离, 共享测试库, schema 隔离, app_config 键空间, 并发包测, fixture 干扰]
scope: [internal/platform/testutil, 全域 *_test.go, go test 并发]
summary: 共享测试库无包级名空间——跨包 fixture（config 键/快照行/策略行）并发互扰翻判；修法=testutil.Pool 每包独占 schema+search_path
first_seen: 2026-10-23
last_verified: 2026-10-23
skills: []
evidence_refs:
  - type: doc
    ref: internal/platform/testutil/testutil.go
  - type: server_fact
    ref: master 裸跑 risk TestAllowDenyAnchors/TestTokenMechanics 全量并发翻判（T-0021 票面）
---

# 测试共享库无包级名空间（T-0021 缺口二）

## 病灶

`testutil.Pool` 返共享 phase_test 库裸连接——`go test` 包级并发下：
- guard 测试翻 `risk.kill_switch=true`→risk 包 Check 读快照误 deny；
- 他包 fin_strategy/snapshot fixture 行顶过敞口/集中度阈翻 risk 判；
- ledger 锚持存 `execution_id=1` 字面假设。
单包跑全绿、全量并发翻——**本地绿≠隔离绿**（CI 红同族）。

## 修法

- testutil.Pool 每调用包建 `t_<pkg>_<rand>` schema+AfterConnect
  `search_path`——Migrate/查询全非限定名随 schema 落位，
  包间 config 键/事实行/令牌行零干扰（一处修全域受益，非逐测补）。
- 测试断言禁持存字面 id（执行行/仓行 PK 取回值非假设）。

## 判据

新测试包直接用 testutil.Pool（隔离自带）；凡断言共享键/行状态者
默认并发脆——修复方向是隔离层而非断言放宽。
