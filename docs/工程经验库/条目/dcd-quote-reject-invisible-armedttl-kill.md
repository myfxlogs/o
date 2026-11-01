---
schema_version: 2
id: dcd-quote-reject-invisible-armedttl-kill
kind: pitfall
risk: high
status: verified
keywords: [dcd_call, armed_ttl_h, retireStaleArmed, 52927, quote, fin_execution, 无痕退役]
scope: [lifecycle, execution, dcd, forensics]
summary: dcd quote 段拒在 fin_execution INSERT 前=exec 零行「live 零成交」假象；armed_ttl 卫生闸以 fin_position 判「未开火」把仍在开火的 live 策略无痕批量退役
skills: [okx-asset-funding]
first_seen: 2026-10-28
last_seen: 2026-10-30
evidence_refs:
  - type: server_fact
    ref: SG fin_execution 10/28 潮 1049 行全零腿+venue order-history 全态零 10/28 单+asset/bills 零动账（T-0255 票面裁决表+handoff 证据件）
  - type: code
    ref: internal/portfolio/lifecycle/run.go:252（tag3 直 UPDATE state='retired' 绕过 Transition()）+ cmd/core/open.go:187-200（DcdQuote 先于 fin_execution INSERT）
  - type: runtime_log
    ref: o-core 10-29 18:19–22:19Z 窗 52927 no-quote×2233+http429×4813+51000×17；22:19:22Z「armed 策略超 4h 退役 11 行」
  - type: ticket
    ref: docs/tasks/T-0255-执行面1028停摆急诊对账.md 结果回填
---

## 「live 零成交」≠断链 + armed_ttl 无痕误杀——10/28 执行面停摆双因链

- **实证链**（T-0255 急诊取证）：48 条 dcd_call 13:37:12Z 同批 probe→live，4h 发起
  1049 笔产品买全在 venue trade 段被恒拒（669 failed=429∪sCode、380 ambiguous=非429
  传输错），venue 全态零落单零动账；17:37Z `lifecycle.armed_ttl_h=4` 触发 run.go:252
  tag3——谓词=`state IN (live,probe)`+无 open 持仓+`COALESCE(max(opened_at),entered_at)`
  超龄——对每笔执行都被拒（零持仓）的活策略判「未开火」直 UPDATE 退役，绕过
  Transition() 零迁移痕（48/48 实测）。10/29 新批 11 条同剧本：18:19Z 晋 live 起
  quote 段被 `52927 "quote provider returned no quote"`×2233 + `http 429`×4813 恒拒，
  22:19:22Z 同闸退役——风暴期 exec 零行。
- **病灶①**：fin_execution「零行」只证明执行未过 INSERT 点——quote 段/gate 拒返回在
  落库前，活策略持续开火对账面无迹。「live 与零成交并存」先查 quote 段 venue 拒码
  （52927=产品不给报价/恒定拒，429=端点节流），勿当执行径断链。
- **病灶②**：retireStaleArmed tag3 用「有无 open 持仓」代理「是否开火」——对全拒
  （ venue 恒定拒）策略是结构性误杀：开火越频、拒得越干净、死得越快。退役动作还
  绕过 Transition()（无痕，审计盲）——判据应取 execution 活动戳，退役须走
  Transition() 留痕（T-0255 前置清单③建议票）。
- **病灶③**：dcdOpen 无腿径把 venue 拒码只写 error→stdout——fin_execution 无 err 列、
  无腿行，容器重建即毁证（10/28 拒码分层不可复原实案）。拒因持久化=前置清单④建议票。
