---
schema_version: 2
id: econ-canon-verdictref-not-status
kind: pitfall
risk: medium
status: verified
keywords: [fin_econ_lesson, canon, verdict_ref, 计数器, 矿面, 口径]
scope: [cmd/core/panelv2_trail, dashboard 矿面, 教训矿]
summary: econ 矿 canon 计数器误用 status='canon'——econ 禁写该状态恒零假空架；canon 身份在 verdict_ref 前缀
first_seen: 2026-11-11
last_verified: 2026-11-11
skills: []
evidence_refs:
  - type: doc
    ref: scripts/cognition/runbook.md ③教训矿节（econ 裁定一律 verdicted+verdict_ref 前缀）
  - type: fact
    ref: SG fin 库 2026-11-11 实数——26 verdicted 中 verdict_ref=canon:% 3 件、merge 23 件
---

# econ canon 语义在 verdict_ref 前缀非 status

## 症状
面板/前端把经济经验矿显示为「0 入库·空架」——实际 3 件已入库。

## 根因
`panelv2_trail.go` econ 矿计数器照抄 eng 的 `status='canon'` 过滤；
`fin_econ_lesson` canon 身份制度上禁写 status（runbook：裁定一律
`status='verdicted'`，canon/merge/弃 三态全记 `verdict_ref` 前缀）。
数的是制度上永不出现的状态=恒零假空架。
eng/econ 两表 canon 语义不对称：eng 走 `status='canon'`，
econ 走 `verdict_ref` 前缀制（要装 `merge:ticket:`/`弃——`/`canon:ticket:` 三态）。

## 修法
econ 计数器改 `status='canon' OR verdict_ref LIKE 'canon:%'`（OR 双轨兼容）。
禁统一两表语义——econ 前缀制是故意设计非欠账。

## 教训
凡引用计数前先核「被数的状态制度上可达吗」——
统计输出继承测量 bug（`exit-formula-leg-order-inversion` 先例同款）。
