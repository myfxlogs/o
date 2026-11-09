---
schema_version: 2
id: bulk-edit-silent-noop
kind: pitfall
risk: medium
status: observed
keywords: ["str.replace", "批量改码", "静默不改", "git diff --stat", "Edit工具", "编辑落空"]
scope: [码面编辑, python批量改, commit前核验]
summary: python str.replace 批量改码目标文本失配时零报错静默不改——commit stat 缺预期文件是唯一体征；批量改后必核 diff 文件清单与意图面一致。
first_seen: 2026-10-2x
last_verified: 2026-10-2x
skills: []
evidence_refs:
  - type: doc
    ref: docs/工程经验库/待归纳.md（staging 原档——纪元3归纲转正）
---

## 事实/症状

python `str.replace` 批量改码，目标文本与实际文件不符时**零报错静默不改**——T-0064 sizing.go quote_ccy 编辑两度落空（第一次 T-0062 反推块失配、commit stat 缺文件才暴露）。

## 根因或策略解释

`str.replace` 找不到子串时返回原串不报错——脚本「跑完了」但零改动，与「改对了」不可分。

## 处置

- 批量改码后必核 `git diff --stat` 文件清单与意图面一致
- replace 前断言计数（`assert old in s` 或 `s.count(old)==N`）
- 或直接用带唯一锚的 Edit 工具（失配即报错）

## 证据

- check: staging 原档两度落空实录（T-0064/T-0062 链）
- result: commit stat 缺预期文件=编辑未生效体征
- ref: 待归纳 staging 原文（本条目转正件）

## 反证条件

- falsifies_when: 所有批量改码路径强制 replace 计数断言/改用 Edit 锚定工具

## 适用边界

- 任何脚本化批量文本替换场景；单文件手工编辑不适用

## 伴生量表（T-0332）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a（豁免：编辑纪律件——判据=commit stat 人工核，无运行面表）
