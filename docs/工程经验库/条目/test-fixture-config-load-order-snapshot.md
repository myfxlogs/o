---
schema_version: 2
id: test-fixture-config-load-order-snapshot
kind: pitfall
risk: low
status: verified
keywords: [config.Client, Load, 快照, app_config, 测试桩, GetBool, fixture, UPDATE]
scope: [testutil, config, go-test, fixture]
summary: 测试桩先 config.Load 再 UPDATE app_config=GetBool 仍见旧值——快照一次性读；桩内置键后必须再 Load
first_seen: 2026-11-06
last_verified: 2026-11-06
skills: []
evidence_refs:
  - type: doc
    ref: internal/platform/config/client.go:33
  - type: doc
    ref: internal/portfolio/hunger/feedstale_test.go:36
  - type: doc
    ref: docs/tasks/T-0410-宏观事件提醒径断更——feed死闸与seed转提醒件.md
---

## 事实/症状

桩测里先 `pgstore.Migrate` → `config.New+Load` → 再 `UPDATE app_config` 置开关
→ `Sensor`/`GetBool` 读到的仍是 Load 时旧值：T-0410 feedstale 桩按
「UPDATE 后置开」写，病灶形三测全哑（feed断更=0 升件=0）——探针没坏，
是桩的置键序错。

## 根因或策略解释

`config.Client` 是快照读器非按次查库器：`Load` 把全表 app_config 读进
`snap` map，此后 `GetBool/Get/GetFloat` 只查内存快照（D-0005 热启契约——
消费方按次读快照，变更靠 config_changed LISTEN 驱动重载）。测试里裸
`UPDATE app_config` 不触发 LISTEN 重载（桩径无 Watch 环），快照恒旧。

## 处置

桩内置键序钉死：**UPDATE/INSERT app_config 后→对被测对象所持 cfg 显式
`Load(ctx)` 再断言**（feedstale_test.go `calSwitch` 一式：UPDATE→s.cfg.Load）。
或先在 Migrate 前/后写库再 New+Load——两形皆可，唯「Load 后改库不重载」
必哑。审计桩断言「开关开而件不动作」先查此序再疑实现。

## 证据

- check: T-0410 feedstale 病灶三测（UPDATE 后无 Load）→feed断更=0
- result: 加 `s.cfg.Load(ctx)` 后同测全绿（升件/病灶行如期落账）
- ref: internal/portfolio/hunger/feedstale_test.go calSwitch

## 反证条件

- check: 某测试 UPDATE app_config 后不 Load 而 GetBool 见新值
- falsifies_when: config.Client 改成按次查库（契约翻转——D-0005 快照制
  废止即本条失效）

## 适用边界

仅测试桩/工具单发径——生产 Watch 环有 LISTEN 重载无此坑；用
`config.New` 直持读键的件（非 Scan 内闭包 func() 取值形——装配面
`func(){v,_:=cfg.GetBool(k);return v}` 每次拍重读快照，快照由生产
Watch 刷）同样受制。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a
（豁免由：桩写法纪律坑非生产量面——无运行时可量对象，红绿分界在测试序）
