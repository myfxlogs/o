---
schema_version: 2
id: core-hookspath-dead-install-face
kind: pitfall
risk: high
status: verified
keywords: ["core.hooksPath", "scripts/hooks", "post-commit", "钩兜底", "收口自报", "commit_report_hook"]
scope: [git hooks, report.sh, T-0317 兜底链, T-0323, T-0324]
summary: 仓库 core.hooksPath=scripts/hooks 时，装在 .git/hooks/ 下的钩件永不触发——兜底机制「看着在装实则死信」。核钩生效性先查 `git config --show-origin --get core.hooksPath` 指向，钩件须落生效径（仓内 scripts/hooks/）而非默认 .git/hooks/。
first_seen: 2026-11-04
last_verified: 2026-11-04
skills: []
evidence_refs:
  - type: server_fact
    ref: 2026-11-04 无头施工席实测 `git config core.hooksPath`→scripts/hooks（/opt/o/.git/config）；/opt/o/.git/hooks/post-commit 存在可执行但 git commit 零触发（收口 ops_event/relay 双无行）；scripts/hooks/ 目录无 post-commit 件
  - type: doc
    ref: docs/tasks/T-0324-* 返修①补件回填「钩兜底失配根因坐实」段
---

# core.hooksPath 指向仓内径——.git/hooks 下钩件永不触发（兜底死信）

## 坑形

`core.hooksPath=scripts/hooks`（/opt/o/.git/config 在案）使 git 只认
**仓内** `scripts/hooks/`；T-0317 装的收口自报兜底钩写进
`.git/hooks/post-commit`——路径存在、位可执行、词集够宽，唯 git
永不读该径。T-0323 收口三行 ops_event 缺、决策席记「返修字样不
匹配」实为表象：**钩自始未跑**，在案三行均手动调（relay 证）。
比「钩逻辑写错」更隐蔽——装的面都不对，任何词集修订皆空转。

## 修法

- 核钩生效性第一拍：`git config --show-origin --get core.hooksPath`
  ——非空则 `.git/hooks/` 全目录死信，钩件必须落指向径。
- 兜底钩归位：`scripts/meta/commit_report_hook.sh`（repo 源件）须以
  `scripts/hooks/post-commit` 入仓方生效——**已归位（T-0325 @7a3e6b3c
  干跑实证：ops_event id=1612 `收口 auto@7a3e6b3c` 钩发自报行在案）**；
  部署树 `/opt/o/.git/hooks/post-commit` 死件已清场。
- 同类闸件自查清单：pre-commit/pre-push/commit-msg 已仓内径在装
  （pre-commit 门禁实证触发在案）；凡新钩件一律入 `scripts/hooks/`。

## 回改点

- `skills/施工席/SKILL.md`「『收口』字样 commit 另有 post-commit 钩兜底」
  一行——**钩已归位生效（T-0325），该句自此为真**；人工 `report.sh 收口
  +移交` 必调口径不变（钩=兜底非替代，auto@ 备注区分两源）。
