---
schema_version: 2
id: report-selfreport-missed-call-gate-blind
kind: pitfall
risk: medium
status: retired
keywords: [report.sh, 自报漏调, 零行票, 闸不可见, post-commit钩, ops_event]
scope: [流程机器, report.sh, dm_tick 待裁闸, 收口自报]
summary: 收口自报=人肉顺手旁径非门禁——漏调则 ops_event 零行票，待裁闸以行喂源=票闸不可见永不上裁；根修=post-commit钩按subject含'收口'自动调 report.sh（T-0315漏调/T-0316同session复发案）
first_seen: 2026-11-04
last_seen: 2026-11-04
evidence_refs:
  - type: doc
    ref: scripts/meta/commit_report_hook.sh（机制级解——钩机器化自律条）
  - type: doc
    ref: scripts/meta/dm_tick.sh（待裁闸喂源=ops_event 行——零行票不上闸）
  - type: doc
    ref: docs/tasks/T-0317-自报行断点排查-report写径全生命周期零行案.md
---

# 收口自报漏调→零行票闸不可见（T-0315/T-0316 漏调案根修）

## 坑

`~/bin/report.sh 收口` 自报=commit 旁的**顺手人工旁径**非机闸——施工
漏调一次，`ops_event` 该票零 收口/移交 行。`dm_tick` 待裁闸以 ops_event
为唯一喂源→零行票**永不上裁集**=管道粘滞且无人察觉（T-0315 靠人肉
记忆+relay 代录才捞回）。

漏调无体感：commit 成功、移交文已发，一切像交付了——唯有闸静默。
同 session 断案「漏调」并立「自律同拍绑」后，T-0316 收口一分钟后
**当场复发**——自律兜底对零行案实证无效（返修①驳回锚②）。

## 根修（已落地）

`/opt/o/.git/hooks/post-commit` + 册面 `scripts/meta/commit_report_hook.sh`：
commit subject 含 `T-NNNN`+`收口` → 自动 `report.sh 收口 auto@sha`
——收口即自报，不再依赖记忆。worktree 共享 hooks 目录=装一次全域
生效；败/无匹配零阻 commit。

## 遗留边界

- 钩只认 subject 含「收口」的 commit——返修/其他收口语义不同措辞
  仍走人工（T-0317 返修①自身实证：subject 无'收口'钩正确跳过，
  relay 代录补位）。
- 新 clone/新机须重装钩（hooks 不随库版本化）——装机手册项。
