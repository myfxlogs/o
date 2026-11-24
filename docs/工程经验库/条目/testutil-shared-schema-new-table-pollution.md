---
schema_version: 2
id: testutil-shared-schema-new-table-pollution
kind: pitfall
risk: medium
status: verified
keywords: [testutil, ShareMain, SeedRestore, 共享schema, 跨测渗漏, fin_venue_roster, 定向清表]
scope: [test-infra, cmd/core, pgstore]
summary: testutil 共享 schema 定向清表=SeedRestore/Reset 名单制——新状态表漏列=行跨测持久渗漏（roster upsert 残行劫持后续测读面断言）
first_seen: 2026-11-24
last_verified: 2026-11-24
skills: []
evidence_refs:
  - type: code
    ref: cmd/core/main_test.go
  - type: code
    ref: internal/platform/testutil/shared.go
---

## 现象

O-37450 给 `fin_venue_roster` 加写径后，cmd/core 共享 schema 测试连环红：
`TestListBrokerOnboardPhase` 读 exness 行得 `state=connecting phase=cred_missing`
——上一测 `SubmitBrokerCred` 留下的 roster 行跨测存活，且 app_secrets 被
SeedRestore 清掉后 cred_ns 必备键核判缺件。

## 根因

`testutil.ShareMain` 共享 schema 的 per-test 清场=`SeedRestore`（快照还原）
+`Reset`（DELETE 名单）**显式名单制**——`applyReset` 只清登记表。新表
`fin_venue_roster` 未登记 → 测试写入的行永不清，跨测渗漏。对照：
`fin_venue_candidate` 早已在册所以无此坑。

## 处置（O-37450 同 commit）

- `cmd/core/main_test.go` SeedRestore 补 `"fin_venue_roster"`（快照=迁移后
  基线，per-test 还原零渗漏）。
- 判据：**凡新增测试可写状态表，ShareSpec 必须同步登记**——表在
  SeedRestore（要还原种子基线）或 Reset（纯清即可）二选一；漏列=跨测
  渗漏隐性债。
- 读面主源化类改造（roster 替代派生）放大此坑：旧派生源（凭据枚举/
  候选册）天然被清，新真源表不在册=新旧真源跨测打架。

## 伴生量表

- 触发信号：测试断言读到「不可能的行态」（state=connecting 但本测未提交）。
- 排查序：先怀疑共享 schema 残留（`SELECT * FROM <表>` 直查），再查代码逻辑。
