---
schema_version: 2
id: migration-seed-leaks-into-test-fixtures
kind: pitfall
risk: medium
status: observed
keywords: ["迁移种子", "测试夹具", "md_macro_event", "真历", "断言连环炸", "DELETE", "fixture"]
scope: [migrations, 测试库, md_macro_event, 测试断言]
summary: 迁移给事实表种真数据（FOMC 真历等）——测试库同种后「无事件日」断言随真实日历翻页连环炸；夹具清表要清全表非只清自种行。
first_seen: 2026-10-2x
last_verified: 2026-10-2x
skills: []
evidence_refs:
  - type: doc
    ref: docs/工程经验库/待归纳.md（staging 原档——纪元3归纲转正）
---

## 事实/症状

迁移给 md_macro_event 种真历（FOMC 10-28 等官方历），测试库同种——真时间漂过 horizon 窗，「无事件日」断言连环炸（TestBoundaryOnceOnly/TestMacroDailyPush 随日历翻页逐个殉）。

## 根因或策略解释

迁移 seed=生产同源真数据；测试断言隐含「库里没有某类数据」假设=对迁移种子隐形耦合，日期推进后假设自然失效。

## 处置

- 测试夹具清事实表要清**全表**（TRUNCATE/DELETE 无 source 过滤），非只清自种 source 行
- 凡依赖「库里没有某类数据」的断言=对迁移种子隐形耦合，须显式 DELETE 或改判据
- 种子数据带 source 标记便于甄别，但清理仍按全表口径

## 证据

- check: staging 原档实测——断言随真实日历节点逼近批量变红
- result: 「无事件日」断言在真历进入 horizon 后失效
- ref: 待归纳 staging 原文（本条目转正件）

## 反证条件

- falsifies_when: 测试库迁移链禁种真实历数据，或全部相关断言改为显式构造夹具不依赖库内空态

## 适用边界

- 一切「迁移 seed 真实数据 + 测试断言依赖空态/稀疏态」组合；append-only 迁移链内新 seed 同险

## 伴生量表（T-0332）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a（豁免：测试基建件——判据=测试随日历翻页变红体征）
