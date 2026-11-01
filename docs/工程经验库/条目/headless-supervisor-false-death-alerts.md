---
schema_version: 2
id: headless-supervisor-false-death-alerts
kind: pitfall
risk: high
status: mitigated
keywords: [headless-supervisor, 监理器, 无头, shell优先级, pgrep, SAID, 假死亡告警, 冤报, grep锚定]
scope: [ops, headless-supervisor, meta, bash]
summary: 监理器三缺陷致假死亡告警潮——完工判定 (A||B)&&C 左结合、pgrep 漏 audit 式、SAID 内存态重启清零重报；同条件另揭裸 grep 收口待审匹正文=漏报盲区
first_seen: 2026-10-30
last_seen: 2026-10-30
evidence_refs:
  - type: ticket
    ref: docs/tasks/T-0260-监理器三缺陷根修.md
  - type: commit
    ref: 03a1f54（meta 仓）
  - type: log
    ref: ~/devin-supervisor.log 10-30 08:05~11:47 冤报串
---

## 监理器假死亡告警潮——三缺陷根修+同条件第4坑（T-0260）

- **实证（10-30 晨）**：「无头单死亡」告警潮实查=误报潮，5 单全活；监理器
  10:10 重启后 T-0246/0249/0250 三连冤报；T-0257 10:13 审计在飞被报死。
- **根因三叠**：
  ①完工判定 `A || ls票面 && grep收口待审`——shell `||`/`&&` 同优先级左结合
    =`(A||B)&&C`，.done 在仍要求票面收口待审，否则照监死报（T-0246/0250 冤案）；
  ②存活 pgrep `headless-<TID>-prompt` 匹不到 `headless-audit-<TID>-prompt`
    ——审计单在飞=「死亡」；
  ③SAID 去重键纯内存态（declare -A）——监理器重启清零→全量重报。
- **第4坑（同条件内另揭，本票票面自证）**：裸 `grep -q '收口待审'` 匹票面
  **正文**即判完工——T-0260 票面验收锚自带「收口待审」串→监理器对在飞单
  误判完工=监控盲区漏报。锚定 `状态：\**收口待审`（审计包装 devin-headless-
  audit.sh 同款锚式）。
- **根修（meta 03a1f54）**：完工判定改 `A || { B && C }`+收口待审锚定状态
  字段；pgrep 宽匹 `headless-(audit-)?<TID>-prompt` 两式；SAID→落盘标
  `/opt/meta/mail/.said/`（重启读回已报键，gen 代际再武装语义不变）；干跑
  六态×4轮+生产 restart 零冤报全实证。
- **模式教训**：①可重启守护件的去重状态禁纯内存态——落盘标或首扫静默；
  ②shell 复合条件 `||`/`&&` 同优先级左结合，跨条件分组必须显式 `{}`；
  ③文档面判定 grep 必锚字段名（状态：前缀），裸词匹正文=双向误中。
