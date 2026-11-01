---
schema_version: 2
id: testutil-dsn-unset-skip-false-green
kind: pitfall
risk: high
status: verified
keywords: [testutil, O_TEST_PG_DSN, skip, 假绿, Migrate, 脚手架]
scope: [test, ci, testutil, fixture]
summary: testutil.Pool 缺 O_TEST_PG_DSN 静默 Skip——本地戳绿≠测试真跑，缺 Migrate 脚手架的测试 CI 才现形；写库表测试后必查「真跑过非 skip」
first_seen: 2026-10-29
last_seen: 2026-10-29
evidence_refs:
  - type: doc
    ref: internal/platform/testutil/testutil.go:24-28
  - type: doc
    ref: cmd/core/cockpit_face_test.go
---

## 事实/症状

T-0239 `TestGetCockpitBlocks` 漏 `pgstore.Migrate(ctx,pool,"fin")` 脚手架
行——施工本地 check_all 绿；CI `go test` 炸
`relation "ops_event" does not exist`（PR#34 run 红）。

## 根因或策略解释

`testutil.Pool` 只建包级空 schema+search_path，**不跑 Migrate**——兄弟测试
全都显式 `pgstore.Migrate(ctx,pool,"fin")`（ops_face 类再加 `"md"`）。而
`O_TEST_PG_DSN` 未设时 `t.Skip` 静默跳过——施工本地无测试 DSN→测试从未
真跑，绿戳=假绿；CI 5434 有 DSN 真跑才现形。

## 处置

- 新增**动 DB 表**的测试：照抄兄弟件脚手架两行
  `pool := testutil.Pool(t)` + `pgstore.Migrate(ctx,pool,"fin")`；
- 本地验真跑：`go test -v` 输出无 `SKIP` 字才算跑过；CI 是唯一可信判官。

## 证据

- check: `grep -n 'pgstore.Migrate' cmd/core/*_test.go`——兄弟件全有、
  cockpit_face_test.go 缺
- result: PR#34 CI `TestGetCockpitBlocks` ops_event 桩 insert 红
- ref: PR#34 build-test run（2026-10-29）

## 反证条件

- check: 设 `O_TEST_PG_DSN` 后本地跑该测试
- falsifies_when: 若 testutil.Pool 某版自带迁移，则缺行无害——核其源码
