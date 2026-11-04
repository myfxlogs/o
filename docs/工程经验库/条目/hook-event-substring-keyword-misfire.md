---
schema_version: 2
id: hook-event-substring-keyword-misfire
kind: pitfall
risk: medium
status: verified
keywords: [post-commit钩, report.sh, ops_event, 词面误触, 子串匹配, 收口错报]
scope: [流程机器, scripts/hooks/post-commit, scripts/meta/commit_report_hook.sh, ops_event 流水]
summary: 钩事件判定按 subject 裸子串匹配（收口|返修|移交）——任意角色 commit 含触发词即误报收口行（T-0331/T-0332 驳回件实证）；根修=正向锚「[角色:施工]前缀+收口词」，负向排除不可采因真返修收口件自体含返修子串
first_seen: 2026-11-04
last_seen: 2026-11-04
evidence_refs:
  - type: commit
    ref: 76d09e6c
  - type: commit
    ref: 6ac51f5f
  - type: doc
    ref: scripts/hooks/post-commit
  - type: doc
    ref: docs/tasks/T-0340-post-commit钩词面收紧-返修子串误触auto收口.md
---

# 钩事件词面误触——subject 子串匹配把驳回/立票 commit 错报成收口行

## 事实/症状

`scripts/hooks/post-commit`（T-0317 落地，底稿 `scripts/meta/commit_report_hook.sh`）
原判定：`subject 含 T-NNNN 且含 '收口|返修|移交' 任一子串 → report.sh 收口`。
凡带 T-NNNN 的 commit 只要 subject 出现触发词即落 `收口` ops_event 行——
决策裁决件（「裁决驳回——返修①件」「裁决通过——返修①收口合master」）、
COS 立票件（「T-0340立并批——…返修子串误触auto收口」）全数误报。
实案：T-0332 驳回 commit `76d09e6c`、T-0331 驳回 commit `6ac51f5f`
均多落一行错 `收口`——ops_event 为业主可见流水，错报=语义污染。

## 根因或策略解释

裸子串匹配无角色面/语义面隔离——触发词表把「词出现」当「事件发生」。
收紧选型：**负向排除不可采**（含返修/驳回不报收口）——真返修收口件
`[角色:施工][T-0332] 返修①收口——` 自体含 `返修` 子串，排之=漏报，
过紧丢报比错报更坏（缺席类病灶，票面红线明禁）。钦定形=**正向锚**：
`[角色:施工]` 前缀 + `收口` 词——角色前缀把决策/COS/审计件天然隔开
（其 subject 常含 收口/返修 字样但前缀非施工），提交格式 `[角色:X][T-NNNN]`
为仓内强制约定故锚可靠。

## 处置

判定改两行（两钩同源同步改）：

```bash
echo "$SUBJ" | grep -qF '[角色:施工]' || exit 0
echo "$SUBJ" | grep -qF '收口' || exit 0
```

触发时机/远端路径/ops_event schema 不动（票面红线）。

## 证据

- check: 干跑五场景（scratch repo + HOME stub report.sh）——返修件/裁决件/无词件零触，
  真收口件+返修①收口件触
- result: 全谱逐值实录见 T-0340 票面回填
- ref: T-0340

## 反证条件

- check: `git log --format=%s` 找 `[角色:施工]` 前缀外存在真收口语义 commit
- falsifies_when: 出现非施工角色发的真收口 commit 需钩补报（如决策席自施工收口），
  或施工收口件改用非「收口」措辞成惯例——此时正向锚漏报，须改判定面

## 适用边界

只覆盖 subject 触发型钩件（post-commit 收口自报同类）；commit-msg/pre-push
等词面判定的钩件同受此教训管（词面判定一律角色前缀锚定，禁裸词表）。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a（豁免由：流程机器词面坑非产线数据面——无适配 gauges.tsv 表，
  复发监测=ops_event 流水人工抽阅）
