---
schema_version: 2
id: probe-silent-stream-false-negative
kind: pitfall
risk: high
status: verified
keywords: [能力探针, push-on-change, 流静默, 假阴性, OnOrderProfit, OnOrderUpdate, 空仓静默, capability probe]
scope: [venue 能力面探针, mt5, okx, 一切 push-on-change 流]
summary: push-on-change 流在无活动态下零帧≠能力不存在——「某流/RPC 不存在」结论必须在真实活动态（持仓/成交）复跑，否则假阴性误导架构选型
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: [mt5-orders-streams, mt5-account, 协议面取证]
evidence_refs:
  - type: doc
    ref: tools/mt5stream/main.go
  - type: doc
    ref: tools/mt5ordstream/main.go
  - type: doc
    ref: skills/mt5-orders-streams/SKILL.md
---

## 事实/症状

空仓态对 MT5 全流面探针：OnOrderProfit 60s 窗零帧→误判「无账户摘要推送流」。持仓态复跑同探针：OnOrderProfit ~1Hz 持续推 ProfitUpdate 全账户摘要帧，OnOrderUpdate 在真实 OrderSend 时发 OrderUpdateSummary 帧。前一结论直接翻案。

## 根因或策略解释

push-on-change 语义=「无变动即无帧」，静默是被观测对象无事件而非通道缺席。探针观测窗内若无触发条件（持仓、成交、行情跳动），一切事件流都呈「不存在」假象。该坑对称存在于 OKX（account/balance_and_position 无余额变动时同样静默）与一切事件源。

## 处置

- 「X 流/RPC 不存在」类结论入 skill 前，必须在**真实活动态**复跑：持仓窗、主动下单窗、行情活跃窗至少其一
- 探针设计=先造触发条件再观测（mt5ordstream 范式：流开→订阅→真单 Send→观帧），禁裸开窗干等
- Unimplemented/裸错=传输层确定结论可轻信；**零帧静默=永不可单独作否定证据**

## 证据

- check: tools/mt5stream 空仓 60s 窗 vs 持仓 60s 窗
- result: 空仓 OnOrderProfit 0 帧 → 持仓 60 帧 ~1Hz；OnOrderUpdate 真单触发发 1 帧（UpdateType_MarketOpen 携全摘要）
- ref: skills/mt5-orders-streams/SKILL.md 2026-10-24 翻案行

## 反证条件

- check: 某流在持续活动态（多次成交流）仍零帧
- falsifies_when: 活动态复跑仍静默且传输无错——方可降格判「存在但无效」，而非「不存在」

## 适用边界

仅适用 push-on-change 事件流；定时心跳流（如 OnTickHistory 请求触发型）与拉取 RPC 不在此坑域。协议面取证 skill 的「无能力断言前强制流」已含同源纪律，本条为其实证锚。
