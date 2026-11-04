---
schema_version: 2
id: absence-detector-missing-report-rows
kind: pitfall
risk: high
status: verified
keywords: ["自报漏调", "缺席发现器", "post-commit钩", "收口字样匹配", "对拍清扫", "死人闸", "report.sh", "T-0317", "T-0323"]
scope: [report.sh, post-commit 钩, ops_event, metabus, 席位编排链]
summary: 「有信号才动」的机制件（钩/踢/帧）看不见信号缺席——收口 commit 存在但无自报行时全链静默，靠人肉发现。凡自动化关节必配缺席对拍闸（死人闸）：扫在飞分支 commit 对账 ops_event，缺行即代录+踢人。五次漏调五病灶，共性唯此一条。
first_seen: 2026-11-04
last_verified: 2026-11-04
skills: []
evidence_refs:
  - type: doc
    ref: docs/tasks/T-0317-自报行断点排查-report写径全生命周期零行案.md
  - type: server_fact
    ref: 2026-11-04 T-0323 返修commit 86d82229 钩洞二次咬人（subject无「收口」字样）；同日钩匹配放宽收口|返修|移交
---

# 「有信号才动」的机制件看不见信号缺席——缺席对拍闸必修

## 病灶

自动化链每个关节都防「错信号」，但**防不了「无信号」**：
report.sh 漏调→无行；钩字样不匹配→无行；踢径未配→无行。
无行=无帧=无席醒——全链静默，从外面看一切正常，实际活已断。
五案实证（T-0315/0316/0317/0323 各漏调形态）：病灶不同，共性唯此。

## 正形

每个自动化关节配**缺席死人闸**（对拍清扫非信号驱动）：

| 关节 | 死人闸 |
|---|---|
| 收口 commit→自报行 | 扫在飞 t/* 分支新 commit 对账 ops_event，缺行超时→代录+踢人 |
| 流帧 | metabus 45s 看门狗（已在） |
| 派单→拉起 | dm_tick 漏拉清扫闸（T-0324 钦定） |
| 自报→ops_event | report.sh 自身失败=静默——靠上条对拍发现 |

通则：**信号驱动件管「到了」，对拍件管「没到」——两层缺一必漏**。
人肉搬运是第三层兜底，不是替代方案。
