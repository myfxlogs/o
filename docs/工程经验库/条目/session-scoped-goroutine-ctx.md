---
schema_version: 2
id: session-scoped-goroutine-ctx
kind: pitfall
risk: medium
status: verified
keywords: [goroutine 泄漏, 会话域, ctx 派生, 重连, ticker, WithCancel]
scope: [internal/mt5, 连接循环, 伴生 goroutine]
summary: 重连循环内的伴生 goroutine（ticker 拍/流消费）须绑会话域派生 ctx（WithCancel+defer cancel）——直传外层运行 ctx 则断连不 cancel，每次重连累积一个永不退出的 goroutine（T-0143 swapLoop 实案）
first_seen: 2026-10-26
last_verified: 2026-10-26
skills: [策略评估与仓位]
evidence_refs:
  - type: doc
    ref: docs/tasks/T-0143（返修回填——bridgeSession sessCtx 派生+TestSwapLoopStopsWithSession 零泄漏桩锚）
  - type: code_fact
    ref: internal/mt5/bridge.go bridgeSession——Run 的 ctx 跨重连共用，swapLoop 直传则重连 N 次=N 个死 goroutine
---

# 重连循环伴生 goroutine 绑错 ctx 层=重连累积泄漏

## 病灶形

bridgeSession 每次重连跑一遍，会话内 `go swapLoop(ctx, cl)` 直传的却是
Run 的外层 ctx（跨重连共用）——会话终（OnQuote 流断返）无人 cancel，
swapLoop 的 ticker select 永不命中 Done：**每次断连重连净增一个死
goroutine**（tick 在跳、RPC 在打、连接已换——幽灵拍）。

## 判据挂接

- 会话/连接生命周期内起的伴生 goroutine：`sessCtx, cancel := WithCancel(ctx)`
  + `defer cancel()`——流断返与 panic 返双径皆触发。
- 桩锚：cancel→goroutine 退出断言（done 通道+期界等待），非只看功能对。

## 处置

T-0143 返修：sessCtx 派生+TestSwapLoopStopsWithSession。
