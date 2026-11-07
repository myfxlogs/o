---
schema_version: 2
id: close-path-gate-denial-unreachable
kind: pitfall
risk: medium
status: verified
keywords: [close 径, kill_switch 不拦, 闸拒, 风控闸, guard 测试设桩, 敞口只减不增, close_denied, plan:gate]
scope: [internal/risk/check.go, internal/guard, 平仓测试设计, 败因分类]
summary: 风控闸契约 close 径 kill_switch/各上限不拦（敞口只减不增）——测试/推演把「kill_switch 翻位=平仓闸拒」设桩必落空：首拍直接平成，败因断言扫 NULL 炸；close 径可复现败类只有 venue 期败与 sizer 计划期败
first_seen: 2026-11-07
last_verified: 2026-11-07
skills: [施工席]
evidence_refs:
  - type: doc
    ref: "internal/risk/check.go:22"
  - type: doc
    ref: internal/risk/close_test.go
  - type: commit
    ref: "t/0428-close-memory T-0428 红桩 TestCloseFailReasonOverwrites 首版实测炸此坑"
repro_or_symptom: "守卫败因测试首拍置 kill_switch 单引号 true 期待 close_denied→markFailed 落 plan:gate: 败因——实际闸放行、ExecuteClose 成功平仓，fin_guard_close.last_error 为 NULL，Scan NULL 进 *string 即炸 cannot-scan-NULL-into-string"
root_cause: 对 risk/check.go 头注契约「close=平仓径——kill_switch/各上限不拦（敞口只减不增，拦平仓=存风险）」认知缺位——kill_switch 闸拒只存在 open 径；close 的闸拒面（intent_incomplete/equity_missing 类）非常规 fixture 可达
fix_and_prevention: "①守卫败因设桩取 close 径真实可复现败类——venue 期败（fakeExec.fail）与计划期败（fakeSizer.errOn 读败/对齐零）两件足够覆盖分类前缀断言；②「拦平仓=存风险」为闸内铁律，凡见「kill_switch 下应拒绝平仓」类断言/推演一律先对 risk/check.go 头注核契约；③last_error 类可空列断言用 COALESCE 或指针扫，NULL 本身也可能是合法态（未败/平成保留语义）；④条目 frontmatter 纯标量含 ASCII 冒号+空格（如 plan:gate: 后接空格）必被 yaml 误读嵌套 mapping——含冒号值一律双引号包且值内禁裸双引号"
notes: "T-0428 败因分类（plan:/venue: 前缀）设计时踩——plan:gate: 前缀在码面保留（dec.Reason 透传），但 close 径常态不可达，属兜底位非测试可依赖路径"
---

## 事实/症状

T-0428 败因落表红桩首版：第一拍翻 `risk.kill_switch='"true"'` 期待
闸拒落 `plan:gate:` 败因——scanOnce 后 `last_error` 为 NULL，断言扫
炸 `cannot scan NULL into *string`。对照既有 `TestCloseRetryAfterFailure`
才见其 `close_failed` 来自 `fx.fail`（venue 败）而非 kill_switch。

## 根因或策略解释

`internal/risk/check.go` 头注明契：**close=平仓径——kill_switch/各上限
不拦（敞口只减不增，拦平仓=存风险）**。kill_switch 闸拒只存在于开仓
径；平仓被闸拒（`close_denied`）需 intent 不完整/equity 缺失类异常态，
常规 fixture 不可达。

## 处置与预防

- 守卫败因测试桩取真实可复现败类：venue 期败+计划期败（sizer）。
- 见「kill_switch 下平仓应被拒」类断言先核 check.go 契约。
- 可空列断言容 NULL（指针扫/COALESCE）——NULL 可为合法态。
