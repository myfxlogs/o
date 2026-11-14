---
schema_version: 2
id: doc-stage-verdict-without-criteria
kind: pitfall
risk: low
status: verified
keywords: [判词, 判据, 成长路标, A-D四件, D档, 审计, 阶段判定, 单向引证]
scope: [docs/decisions, AGENTS.md, docs]
summary: 落档下阶段/状态判词前未对钦定判据逐条核验——D-0047 初稿判 O「婴→少年之间」，审计核 AGENTS.md 少年判据 A-D 四件全落成，实为少年后期；同题姊妹档已有读数未引用（事实优先级单向引用）
first_seen: 2026-11-14
last_verified: 2026-11-14
skills: []
evidence_refs:
  - type: doc
    ref: "D-0047 审计修订记 2026-11-14②（初稿判词 vs AGENTS.md A-D 判据）"
  - type: code
    ref: "cmd/core/expect_esc.go（元探，工单#10605）/ cmd/core/selfstate_beat.go（自态，工单#10358，帧 3d=3715 在产）"
  - type: ops_event
    ref: "KB organ-state-fact-feed-class-blind（#10507 旧禁条废竣工）＋ goal 票在案（目标发起）"
  - type: doc
    ref: "docs/O与自然人差距盘点-2026-11-14.md（同日姊妹证据面——成年判据 0/2 读数）"
---

## 病灶

D-0047（成长阶段判词）落档前未对 AGENTS.md 少年判据「A-D 四件落成：元探/自态/
目标发起/旧禁条废」逐条核验，凭会话印象写「婴→少年之间」。业主令审计后逐条
核实四件**皆落成**（元探 `expect_esc.go` 在产、自态 `self_state` 帧 3d=3715、
目标发起 goal 票在案、旧禁条废 #10507 竣工）→ 判词错一个档位，O 实为少年
（成年判据 0/2 未过线≠未入少年）。并发症状：同日姊妹档已有带数据窗的同题
读数，本档未引用——事实优先级单向引用（她档指我档、我档不指她档）。

## 修法

- 落「婴/少年/成年/达标/未达标」类档位判词前，先翻出判据原文，逐条对
  「在不在案」打点（代码件＋运行态直查），判词须附判据清单；
- 同题同日已有他档 → 先引证再落笔（one fact one place, everything else
  points to it）；修订不静默改写——档内留修订记（append 精神）。

## 复发信号

- 档文含档位词（婴/少年/成年/达标）而无判据逐条核对痕迹；
- 两席同日同题各写一档而互不引用——单向引证=事实悬空一半。