---
schema_version: 2
id: mt5-rpc-empty-payload-half-alive
kind: server_fact
risk: medium
status: observed
keywords: ["mt5", "AccountSummary", "空体", "RPC通载荷空", "会话半活", "venueFailGate", "mtapi", "连败闸"]
scope: [mt5, mtapi网关, AccountSummary, venue读面]
summary: mtapi 网关会话/上游瞬断可呈「RPC 正常返回但载荷空」——mt5 facts/summary account 空体实案（43s 自愈）；读面空体≠功能缺席，定性前先分 venue 瞬态 vs 采集病灶。
first_seen: 2026-11-06
last_verified: 2026-11-06
skills: [mt5-account]
evidence_refs:
  - type: doc
    ref: docs/工程经验库/待归纳.md（staging 原档——纪元3归纲转正）
---

## 事实/症状

venue 读败连败闸（T-0352 venueFailGate）首拍真案——mt5 连败 10 拍/43s 升件（ops_event#2603，2026-11-06 08:18Z），败因 `mt5 facts/summary: account 空体 / <nil>`；AccountSummary RPC 正常返回但载荷空，43s 后读常清零自愈（滞回+日限1 未再报）。

## 根因或策略解释

mtapi 网关侧会话/上游瞬断可呈「RPC 通而载荷空」形——传输层活着、业务载荷缺席=会话半活体征。空体是 venue 侧瞬态非采集侧病灶（同窗口 mt5-account skill 复测 AccountSummary ok）。

## 处置

- 读面空体≠功能缺席——定性前先分清 venue 侧瞬态 vs 采集侧病灶
- 连败闸类探测器须把「RPC 通+载荷空」与「RPC 败」分账（半活体单独定性）
- 单发一过性不升格；同 venue 同败因连败>1 次/日再取证对拍根因

## 证据

- check: staging 原档 ops_event#2603 实案+mt5-account skill 复测 ok
- result: AccountSummary RPC 通返回空载荷，43s 自愈
- ref: 待归纳 staging 原文（本条目转正件）

## 反证条件

- falsifies_when: 同 venue 同败因连败复发且网关连接态对拍坐实采集侧根因，或 mtapi 侧修复使空体不再出现

## 适用边界

- mtapi 系 venue（mt5/mt4）RPC 读面；OKX REST 面空体语义不同不直接外推
- 单发观察——再发分布未明，判读按「会话半活」候选非定论

## 伴生量表（T-0332）

- gauge_name: venueFailGate 连败计数（ops_event 升件面）
- class: 断供
- query: SELECT count(*) FROM ops_event WHERE note LIKE '%account 空体%' AND ts>now()-interval '7 days'
- threshold: 同 venue 同败因 >1 次/日=升格复核
