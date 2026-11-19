---
schema_version: 2
id: ticket-filed-without-cert-anchor
kind: pitfall
risk: medium
status: verified
keywords: [立案闸, 证锚, 施工踩空, desk口径, 可能性层]
scope: [ticket-filing, triage, desk]
summary: 未取证定性即立「工单」——可能性件混进施工候单架,施工位接活先重做分诊的取证(踩空烧工时);根治=desk派发池证锚硬径
first_seen: 2026-11-19
last_seen: 2026-11-19
evidence_refs:
  - type: ticket
    ref: "#24798 dcd-live-settlement-consumer(消费件缺失未核码即立,机制实已在产)"
  - type: ticket
    ref: "#25482 desk-dispatchable-cert-gate(本条目随修件同 commit)"
---

## 未核立案——可能性件顶「工单」名混进施工候单架

- **实证**：#24798「dcd 结算消费件缺失」立案时未核码——施工位接手
  花 ~1h 取证，发现机制早已在产（仓#42 等 12-04 到期自然结算），
  工单本不该立。烧的是施工工时不是分诊工时——漏检成本错付。
- **病灶型**：分诊判据松=「销转工单」门槛不含取证定性——可能性件
  （未证伪未证实）与已定性缺陷同票同面；软径（纪律自觉写证）已被
  此案证败，须机检。
- **判据钉死**：「工单」立案事件必载 `证=<可核锚>`（锚=log行/
  sql行/commit/venue实拍之一，可复核可复算）；无证=可能性层件，
  归唤知分诊销或挂起核实，不占施工道。
- **机检落点**：fin_0212——desk 视图 w CTE 排位谓词加
  `note ~ '证='`：无证工单驻 N 计数但无 rank→dispatchable=false
  （看得见派不出，逼分诊补证或销，不静默丢票）。
- **同族**：方向旗顶工单 tkind 占可拣（立案归类错位）的姊妹形态——
  同属「票桌非施工件污染」家族；归类错位治 tkind，本案治证锚。
