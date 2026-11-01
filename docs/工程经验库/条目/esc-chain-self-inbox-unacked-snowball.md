---
schema_version: 2
id: esc-chain-self-inbox-unacked-snowball
kind: pitfall
risk: medium
status: verified
keywords: [escalation链, 滚雪球, 自家升级件, ack, 账面残链, EDR, L1-L8]
scope: [mbus, deadline, diagnosis]
summary: 实体处置毕≠账清——发回自家的升级件（DM→DM esc 链）未 ack 则每 30min 滚一级直到 L8 唤醒；处置回执投出后必须回头消费自家 inbox 全链
first_seen: 2026-11-01
last_seen: 2026-11-01
evidence_refs:
  - type: mail
    ref: 20261101-105854-MANAGE-3606（飞书应答代分诊根件）
  - type: mail
    ref: 20261101-142924-DM-3890（L8 到点唤醒件）
---

## EDR 升级链自反件不消费=滚雪球唤醒

- **实证**：2026-11-01 飞书应答代分诊件（MANAGE-3606）实体面 14:16-14:18 已毕（T-0283 并合@23825ba2+完成事件已报、T-9224 沙箱套件收口+spawn 停止+隔离 quarantine），回执 DM-3766 MANAGE 亦 ack。但处置席未消费自家 inbox 内的升级件——链仍按 deadline 每 ~30min 滚一级：L1(3649)→L8(3890) 七件连发，15:00 deadline 到点唤起 CTO 会话清账。
- **语义根**：esc 链每级都是独立 armed 账行+独立 .mb 投递件；下级「源账到点」只核上级行消费态，与实体处置是否完毕**无关**——做完事不 ack 自家升级件=账面永远「未处置」。
- **教训**：处置升级件的标准收尾=**实体处置→回执投出→自家 inbox 同链全件 ack**（`mbus.sh ack DM <id>` 逐件，ack 同拍级联 disarm `esc:<id>` 账行）。只报不 ack=下个 session 被同链再唤醒。同理适用一切「账=X deadline 到点」唤醒件：先核实体是否已毕，毕则清账消费残链，勿把残链当新事重做。
- **判别式**：升级件 ref 里一串「源：…」件若全在 MANAGE/acked=实体已结，本件=账面残链；处置动作只有 ack 链+一条回执，不重复实体工作。
