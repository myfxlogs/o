---
schema_version: 2
id: push-ledger-write-before-send-confirm
kind: pitfall
risk: high
status: observed
keywords: ["幂等锚", "先记账后发送", "fin_push_ledger", "md_calendar_push", "重试丢失", "送达率", "push ledger"]
scope: [fin_push_ledger, md_calendar_push, 推送面, 通知面]
summary: 推送幂等行「已尝试」误当「已送达」——先插行后发送发败=永久丢报零重试（2026-10-24 报表瞬时发败实案）；幂等行只在送达确认后写。
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: []
evidence_refs:
  - type: doc
    ref: docs/工程经验库/待归纳.md（staging 原档——纪元3归纲转正）
---

## 事实/症状

幂等锚行「已尝试」语义误当「已送达」——fin_push_ledger 成败不分照写（2026-10-24 报表瞬时发败=永久丢报零重试）；md_calendar_push 同款先插行后发送、发败占位注释还自称「下拍重试」。

## 根因或策略解释

「查账→动作→记账」三段序被写成「查账→记账→动作」——记账先于送达判定。防重推的占位行反而吞掉失败重试权：行在=认为已处理，发败无下文。

## 处置

- 推送幂等行只在送达后写（Sender 返 msg_id+ok 判成败）
- 「先占位防重推」换「崩溃重发」丢「失败重试」——送达率优先防重率
- 配套降级计数须出观测面（内存计数=静默丢报）

## 证据

- check: staging 原档三处同病实录（fin_push_ledger/md_calendar_push 等）
- result: 发败占位行使重试径失效
- ref: 待归纳 staging 原文（本条目转正件）

## 反证条件

- falsifies_when: 全部推送面幂等行改送达后写且发败可重试，或 venue 侧送达语义保证 at-least-once

## 适用边界

- 一切「先查账防重→执行外发→记账」的推送/通知/上报面；纯内部幂等（无外发）不适用

## 伴生量表（T-0332）

- gauge_name: n/a
- class: 静默吞错
- query: n/a
- threshold: n/a（豁免：送达语义件——失效形=丢报由推送面观测行兜底，gauges.tsv 无直接表）
