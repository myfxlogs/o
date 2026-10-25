---
schema_version: 2
id: swallow-sql-err-dead-guard
kind: pitfall
risk: high
status: verified
keywords: [debounce, 在途窗, 列名漂移, 错误吞噬, 下划线等号, fail-open, 守卫静默失效, SQL 报错丢弃, guard dead]
scope: [internal/ledger 对账环, 一切 `_ =` 吞错判定面, 跨表查询列名]
summary: 判定守卫的探测查询引用不存在列→QueryRow 恒报错被 `_ =` 吞→守卫自出生即死且零痕——吞错+列漂移复合=生产守卫静默失效，必须错误显性化+测试锚对真实 schema 断言
first_seen: 2026-10-25
last_verified: 2026-10-25
skills: []
evidence_refs:
  - type: doc
    ref: internal/ledger/run.go
  - type: doc
    ref: docs/tasks/T-0087-对拍公式不对称根修.md
---

## 事实/症状

ledger 对账在途窗 debounce（T-0062）探测查询写 `fin_order_leg WHERE created_at`、`fin_execution WHERE opened_at`——两表真列是 `ts`/`created_at`（0009 schema）。查询恒报 `column does not exist`，返回值被 `_ =` 丢弃，`inflight` 恒 false：**debounce 自 T-0062 合入起从未生效过**。生产实证：T-0083 迁移回填窗内对账每 5s 连报（守卫本应罩 30s）；修复前无任何报错/计数痕。

## 根因或策略解释

复合坑：①判定面探测错误被 `_ =` 吞=失败退化成 fail-open（守卫语义=跳过不判，死守卫=永远「不跳」→漏判面反过来又让误判全量漏出）；②测试对真实 migrations 跑也吞同一错误——测试绿因断言的是「告警发生」路径，窗内路径无锚。两个事实叠加使死守卫在生产存活多日且不可观测。

## 处置

- 判定面探测查询**禁 `_ =` 吞错**：错→计数门（rej.Inc 预注册键）+WARN+本轮让位不判（fail-closed），错误必须可观测
- 跨表查询列名以 migrations 为准写完后，用真实 schema 跑一遍（psql \d 或 EXPLAIN）再交付；禁凭记忆拼列
- 窗内/窗外**双向测试锚**：窗内跳过锚+窗外判定锚各一——只测单侧时死守卫可从另一侧漏过（窗外判定锚是列名勘误的直接判据：列错→探测败→永判=锚红）

## 证据

- check: git show T-0062 合入版 run.go vs migrations/0009 列集；生产 SG 2026-10-25 00:44-01:04 报警窗日志（每 5s 连报）
- result: 旧查询 `fin_order_leg.created_at`/`fin_execution.opened_at` 两列均不存在；修复列名+错误显性化后 T-0087 双向锚全绿
- ref: internal/ledger/run.go reconcile；internal/ledger/ledger_test.go TestReconcileDebounceWindows

## 反证条件

- check: `_ =` 吞错的探测查询在真实 schema 上无报错且守卫行为有窗外判定锚覆盖
- falsifies_when: 查询列全存在且双向锚在位仍出现死守卫——另立新根

## 适用边界

一切「探测查询决定是否执行判定」的守卫面（debounce/在途窗/快照鲜度）；纯读展示面的吞错（错=显示空）不在本条高危域，按各面纪律处置。
