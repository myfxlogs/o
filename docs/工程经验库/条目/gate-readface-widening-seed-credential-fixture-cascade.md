---
schema_version: 2
id: gate-readface-widening-seed-credential-fixture-cascade
kind: pitfall
risk: medium
status: verified
keywords: [闸, fail-closed, dcd_baseline, fin_risk_decision, 种子行, 测试夹具, exposure_cap, md_ticks, T-0355]
scope: [internal/risk, cmd/core 测试夹具, 统一持仓读面]
summary: 闸读径扩面（新事实族入判据）时，迁移种子行会进入每个测试 schema——凡 schema 内走闸的测试夹具必须同步补齐新判据所需输入（本案=dcd_baseline BTC 种子行须配 BTC-USDT tick+放宽 max_exposure_usd），否则全拒暴毙
first_seen: 2026-11-05
last_verified: 2026-11-05
skills: []
evidence_refs:
  - type: commit
    ref: t/0355-exposure-cap-read
  - type: doc
    ref: internal/risk/exposure_dcd_test.go
---

## 事实/症状

T-0355 把 `fin_position_view kind='dcd'`（dcd_baseline 凭据聚合）计入
`exposure_cap`，估值需 ccy-USDT 最新 tick，缺价=fail-closed 拒放。落地后：

- `internal/risk` 全部 allow 锚测试瞬间变 deny——迁移 fin_0043 种子行
  （BTC 0.0207 allow）在每个 Migrate 过的测试 schema 都在册，无 tick→拒；
- `cmd/core` `TestDcdTradeRejectPersist` 三子例同灭——quote 成功径走到
  gate.Check，种子 dcd 估值 ~2070 USD 顶破迁移种值
  `risk.max_exposure_usd=1000`；
- 表面像「闸坏了」，实为**判据扩面×种子行×缺夹具**三件套传导。

## 根因或策略解释

`pgstore.Migrate` 把迁移内 INSERT（fin_0043 dcd_baseline 种子、fin_0008
risk.* 种子阈）带进每个包级测试 schema——种子行=全局事实不是夹具。
闸判据每扩一族事实，所有「期望放行」的测试夹具都要为新族备齐输入
（价/阈/凭据），否则 fail-closed 语义把缺件放大成全拒。反向教训同样成立：
**改闸前先 `SELECT` 各测试 schema 的判据输入存量**（本案=dcd_baseline
行+max_exposure_usd 种值），夹具补齐=setup() 一处 upsert 而非逐测补丁。

## 处置

- 夹具集中收口：`internal/risk/check_test.go setup()` 与
  `cmd/core/open_dcd_reject_test.go dcdRejectFixture()` 各加
  `md_ticks BTC-USDT` 幂等 upsert（risk 侧另把 max_exposure_usd 显式
  拉回测试阈——core 侧种子阈 1000 被 dcd 估值顶破故放宽至 1e9）。
- 新判据族测试独立文件 `internal/risk/exposure_dcd_test.go`——清场
  （DELETE dcd_baseline）后自控变量，边界断言用实测 cur+目标值卡
  max_exposure 双向（压线拒/放宽放）。

## 证据

- check: `go test ./internal/risk/` 修复前 7 测全 deny exposure_cap→
  fixture 后 9/9 绿；`go test -run Dcd ./cmd/core/` 修复前 3 子例
  `闸拒: exposure_cap`→12/12 绿。
- result: 闸语义扩面票的标准夹具传导面=迁移种子行全集+判据输入全集。
- ref: T-0355 收口支；check_test.go:29-35、open_dcd_reject_test.go:55-67。

## 反证条件

- check: 未来闸扩面票若某测试 schema 判据输入天然齐备（如新族无种子行），
  本条目「必补夹具」结论降级为「按需核」。
- falsifies_when: pgstore.Migrate 改为不跑迁移内数据 INSERT（纯 DDL），
  种子行不再入测试 schema。

## 适用边界

凡「闸/判据读径纳入新事实表或新聚合」的施工票；纯阈值/参数调整不适用
（无新输入需求）。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a（豁免由：测试夹具传导面坑非产线数据面；监测面=check_all
  全谱测试即闸——夹具缺件当场红）
