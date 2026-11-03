---
schema_version: 2
id: loss-attribution-read-guard-trigger-first
kind: pitfall
risk: high
status: verified
keywords: [亏损归因, trigger_kind, fin_guard_close, delta_drift, funding_flip, 持仓时长, 评估器阈值, 1114 单, 叙事漂移]
scope: [internal/guard, internal/research/funding, internal/portfolio/lifecycle, 决策档归因面]
summary: 给 live 亏损归因前先查 fin_guard_close.trigger_kind 分布＋持仓时长＋fin_execution 按 state 分桶——跳过这三查直接归因到评估器/阈值，会修一个没发生过的病并漏掉真病
first_seen: 2026-11-03
last_verified: 2026-11-03
skills: [生产取证, 策略评估与仓位]
evidence_refs:
  - type: server_fact
    ref: SG fin_position⋈fin_guard_close 22 仓 trigger_kind 全=delta_drift（funding_flip=0），avg hold 1.2h；fin_execution 1114=failed 689+ambiguous 380+closed 42；funding_settle 全账 4 行 +0.002
  - type: doc
    ref: D-0043 CTO 会签注记 / T-0297 会签勘正
---

# 亏损归因先读 guard 触发分布（D-0043/T-0297 勘正案）

## 病灶
- D-0043 立案时把「17 仓净 −1.97、1114 单」归因到评估器费用未摊销→阈值形同虚设→guard 翻面即平→反复开平。三步推理每步都合理，但**没有一步查过平仓触发器是什么**。
- 实查：22 仓全由 `delta_drift` 杀（上线期量纲/陈 tick/基线缺陷，已修），funding_flip 零次；1114 单里 1069 单根本没成交。于是票面要加的 `flip_persist_n`/`min_hold_days` 两键是在修一个没发生过的病；真缺陷（费用未摊销）是隐患而非既成亏因。
- 亏损叙事一旦写进决策档就会被下游票面复制（T-0297 原文「勿重复诊断」）——归因错在源头，越往下越难纠。

## 裁决
- live 亏损归因**三查前置**，缺一不写归因句：
  1. `SELECT g.trigger_kind, count(*), avg(closed_at-opened_at) FROM fin_position p LEFT JOIN fin_guard_close g ON g.position_id=p.id GROUP BY 1`——谁平的仓、活了多久。
  2. `SELECT state, count(*) FROM fin_execution GROUP BY 1`——「N 张单」里多少真成交。
  3. `fin_cashflow` 按 kind 聚合＋`funding_settle` 行数——仓有没有活到收益发生。
- 评估器/阈值类缺陷单独成立（码面亲证），但**不得**与既成亏损直接连线，除非三查显示仓是因阈值放行后被收益类触发（funding_flip/max_hold）平掉的。
- 「禁止离场」类方案（最短持仓）须过沉没成本检验：费已付出，禁平不回费；只有再入场成本（冷却/滞回）能治开平振荡。

## 判据
- 决策档/票面出现「X 单磨损」「翻面即平」句式时，必须附 trigger_kind 分布与 execution state 分桶两张表；无表=未归因，退回。
