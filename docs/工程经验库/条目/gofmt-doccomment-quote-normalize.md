---
schema_version: 2
id: gofmt-doccomment-quote-normalize
kind: pitfall
risk: low
status: verified
keywords: [gofmt, doc comment, 引号, 弯引号, U+201D, gofmt漂移, pre-commit, Go注释]
scope: [Go, gofmt, 注释规范, pre-commit闸]
summary: gofmt 会把函数 doc comment 里成对的 ASCII 单引号 '' 规范化为 Unicode 弯引号 ”——手写 ASCII 版被判漂移，改回 ASCII 又被改回弯引号的死循环错觉
first_seen: 2026-11-08
last_verified: 2026-11-08
skills: [施工席]
evidence_refs:
  - type: doc
    ref: internal/ledger/holding/sync_test.go:56（T-0478 桩测 doc comment 实证）
  - type: commit
    ref: t/0478-holding-absent-del
---

## 事实/症状

- Go 函数声明前的 doc comment（连续 `//` 块）里写 `venue_ref=''` 这类
  成对 ASCII 单引号，`gofmt -d` 显示漂移、`gofmt -w` 落盘成
  `venue_ref=”`（U+201D 右双引号）；手改回 `''` 后 pre-commit
  gofmt 闸再 FAIL——看似 gofmt 自相矛盾的死循环。
- 函数体内普通行注释里的 `''` 不受影响（gofmt 只重排 doc comment）。

## 根因或策略解释

gofmt 自 Go1.19 起重排 doc comment（doc comment formatting）——其中
排印性引号对被规范化为弯引号；行注释/字符串字面量不在此列。

## 处置

doc comment 里避免成对 ASCII 引号字面量——改写叙述形
（`venue_ref 空`/`venue_ref 为空串`），或直接接受 gofmt 规范化后的
弯引号形态（`gofmt -l` 干净即合法，不必与 ASCII 版纠缠）。

## 证据

- check: `gofmt -d internal/ledger/holding/sync_test.go`——`''`→`”` 漂移实证
- result: 改写「venue_ref 空」叙述后 `gofmt -l` 干净、pre-commit 过
- ref: T-0478 收口 commit（t/0478-holding-absent-del）

## 反证条件

- check: 换 Go 工具链版本重跑同形 doc comment
- falsifies_when: 新版 gofmt 不再规范化引号对（本条目 retired）

## 适用边界

- 仅 Go doc comment（顶层声明前连续 `//` 块）；函数内行注释、
  字符串字面量、其他语言注释不适用。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a（工具链规范化坑，无量化表——gofmt -l 即判据本身）
- class: n/a
- query: n/a
- threshold: n/a
