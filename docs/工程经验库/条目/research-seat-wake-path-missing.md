---
schema_version: 2
id: research-seat-wake-path-missing
kind: mechanism-gap
risk: medium
status: verified
keywords: [intake_dry, 源普查, 研究席, 唤起径, 触发档, seat-hosts, 派发层断链, research-intake-spec]
scope: [研究进水口, 唤起链, o-mech事件臂, seat钉位]
summary: intake_dry 升件三因之首=触发档未立——研究席会话无唤醒单元（seat-hosts 钉位只四单元），钦定触发词写得到升件面却唤不起会话,派发层断链。
first_seen: 2026-11-18
last_verified: 2026-11-18
skills: [事实核验, KB维护]
evidence_refs:
  - type: escalation
    ref: 升件#24405
  - type: ticket
    ref: 工单/research-seat-wake-path
  - type: doc
    ref: docs/drafts/research-intake-spec.md
  - type: doc
    ref: cmd/core/supply_esc.go
  - type: doc
    ref: deploy/seat-hosts.tsv
---

## 事实/症状

升件#24405：商业产出/intake_dry——`ops_event seat='研究席' note LIKE
'源普查%'` 7d 窗零新增（末登记 11-11 10:24Z，机制曾转）。
探针 note 自载三因：触发档未立/研究臂未被唤/分诊过严。

## 根因或策略解释

**触发链断在派发层**：research-intake-spec.md 钦定「intake_dry∨breed_dry
升件→源普查循环」是研究席唤起触发之一，skills/研究员同载四触发
（T-0300 venue 达阈/venue 新品/intake_dry∨breed_dry/求知票）。
但 `deploy/seat-hosts.tsv` 钉位仅 o-worker/o-self/o-cognition/o-audit
四单元——**研究席没有唤醒单元**，o-mech 事件臂也不订此唤起；
升件写到 ops_esc_pending 后只到唤知会话手里，唤知按 spec 钦定
「值守/唤知不代采」不能自己跑普查——信号能产、会话唤不起。

selfgap_esc.go 注释已自述「O 环内无『工单→研究席』自发派发件」——
缺口被码面知道但未修，直到 intake_dry 探针 7d 窗点火才账面化。

## 可防一问

器官规格「会话被某信号唤起」必须有唤醒单元物化该径——
「规格档写触发词」≠「触发链通」：钉位表/事件订阅/派发件三者缺一
即断链。修法=立研究席唤起径（工单/research-seat-wake-path：
o-research-dispatch 同款唤知机 ∨ 并入既有派发面按触发词唤研究席
runbook），验收=人为落 intake_dry 触发→研究席会话自起+源普查登记行
落地（机制证非手唤）。

## 排查口诀

「某席未被唤」类信号先查三物：钉位表有无该席唤醒单元、事件臂有无
订阅该触发词、规格档触发词与升件 class/scope 词面是否对得上——
三者对齐才叫径通。
