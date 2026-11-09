---
schema_version: 2
id: event-ts-not-liveness-source
kind: invariant
risk: high
status: observed
keywords: ["事件时间", "liveness", "max(ts)", "md_funding", "活性判据", "伪造绿灯", "水位"]
scope: [md_funding, 活性判定, 数据新鲜度, PG-md]
summary: 数据列的「事件时间」语义≠「行写入时间」——md_funding.ts 可未来值（费率事件时刻先于写入 ~6.5h），拿它做通道 liveness=伪造绿灯；活性判据只用真流心跳列。
first_seen: 2026-10-2x
last_verified: 2026-10-2x
skills: []
evidence_refs:
  - type: doc
    ref: docs/工程经验库/待归纳.md（staging 原档——纪元3归纲转正）
---

## 事实/症状

md_funding.ts 是费率**事件时间**非行写入时间——可呈未来值（T-0088 实测 max=08:00Z 先于当前 ~6.5h）。若拿它做通道 liveness 判定，未来值恒「新鲜」=伪造绿灯。

## 根因或策略解释

事件时间列（业务事实发生时刻）与落库时间列（观测时刻）语义不同—— funding 费率是按未来结算时刻预告的，事件 ts 天然可大于 now()。混入 max(ts) liveness 聚合即污染。

## 处置

- 活性判据只用真流心跳列（md_ticks/last_beat 类写入时刻列）
- 事件时间列禁入 liveness 聚合；展示面可用但须标语义
- 新增「该列是事件时还是写入时」判读入库纪律

## 证据

- check: staging 原档 T-0088 实测 max(ts)=08:00Z 先于当前时刻
- result: 未来值在 liveness union 中恒判新鲜
- ref: 待归纳 staging 原文（本条目转正件）

## 反证条件

- falsifies_when: 全部活性判据源审计确认只用写入时刻列，或 md_funding.ts 语义改为写入时刻

## 适用边界

- 一切「max(某 ts 列）判活」的读径；凡列名带事件语义（effective/settle/event/happened）须先辨义

## 伴生量表（T-0332）

- gauge_name: n/a
- class: 缺席
- query: n/a
- threshold: n/a（豁免：语义判据件——失效形=活性假绿灯由探针/水位闸兜底，gauges.tsv 无直接表）
