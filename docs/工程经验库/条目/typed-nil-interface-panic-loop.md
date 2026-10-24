---
schema_version: 2
id: typed-nil-interface-panic-loop
kind: pitfall
risk: high
status: verified
keywords: [typed-nil, interface, nil-receiver, panic, 崩环, 装配, 降级路径, Go]
scope: [cmd/core 装配点, internal/platform/notify, 全域接口注入面]
summary: 接口装类型化 nil 指针（(*T)(nil)）时 iface==nil 判 false——降级分支漏进实调径，nil receiver 方法即 panic；容器 Restarting 循环（creds 缺席=必崩路径）
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: []
evidence_refs:
  - type: server_fact
    ref: SG creds 缺席实测崩环 Restarting(2)（T-0032 票面）
  - type: code
    ref: internal/platform/notify/alert.go cliNil（reflect 兜底）
  - type: test
    ref: internal/platform/notify/alert_test.go TestTypedNilNoPanic
---

# 接口装类型化 nil 指针=降级失效崩环（T-0032 三修之一）

## 病灶

`var cli *FeishuClient`（凭据缺席=零值）→ `NewAlerter(cli, ...)` 隐式装进 `Sender` 接口。
Go 语义：**接口值=（类型,指针）二元组**，typed nil 装入后 `iface == nil` 为 **false**——
`if a.cli == nil { return }` 降级防线判不中 → `SendText` 在 nil receiver 上调用 →
方法内解引用 panic → 容器 Restarting 循环（creds 缺席环境=启动必崩路径）。

## 修法（双防线）

1. **主修在装配点**：显式接口变量只装非 nil 具体件——
   `var s Sender; if p != nil { s = p }; NewAlerter(s, ...)`。凭据缺席传**接口 nil**
   而非 typed nil。
2. **兜底在消费面**：`cliNil(s)` 用 reflect 判 `Kind()==Ptr && IsNil()`——防未来
   装配点复发（注入不炸锚 TestTypedNilNoPanic 常绿）。

## 通用律

- 任何「接口字段 + 构造注入 + 缺席降级」面都可能中此坑：降级分支的 nil 判对接口
  写 `== nil` 只挡接口零值，挡不住 typed nil 装入。
- 高危判定：**降级路径恰在凭据/外部依赖缺席时走**——本机有凭据测不出，部署到
  缺件环境（SG 新机）才崩=「本地绿≠生产活」的典型样本。
- 测试锚必须显式构造 `var p *T; var s I = p; s == nil` 前置断言（证语义在案）再注
  入——防 Go 版本/写法漂移悄悄失效。
