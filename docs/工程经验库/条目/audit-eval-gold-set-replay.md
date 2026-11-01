---
schema_version: 2
id: audit-eval-gold-set-replay
kind: method
risk: low
status: verified
keywords: [审计回放, gold集, 剥态防抄答, 无头审计, 计分机判, T-0245]
scope: [audit, evaluation, meta-platform]
summary: 审计信任闸构建法——剥态票面副本（状态/判决/回填段全除）+收口commit+gold判决录三件成集，无头回放对 gold 机读一致；终态词引证=抄答嫌疑机判打折——判决文禁含 git log/票面终态词推证
first_seen: 2026-10-29
last_seen: 2026-10-29
evidence_refs:
  - type: ticket
    ref: T-0245
  - type: path
    ref: /opt/meta/eval/
---

## 法

- 三件套成 gold：剥态副本（awk 除『状态』行/审计席署名/回填|驳回段——
  判决泄露面全剥离，审计仅见钦定界+验收锚）+收口 commit hash+gold 判决录
  （expected+驳回理由原文 history）。
- 回放=devin-headless-audit.sh 逐票单飞——判决写回副本『状态』字段读出。
- 防抄答机判硬条款：判决文含终态词（闭环|终过|已并|销号|merged）→
  score=0.5 flag=cheat 自动标——机器裁非人判；可读判词（审计通过|驳回：)
  =1；不可读=0。
- 剥离桩测必跑：副本零判决面残漏为放行前置。

## 坑位

- 判决文引用 git log/票面终态字段=反推捷径——prompt 条款+词表机判双闸；
- gold 史须含驳回原文——归因三分类（判据理解/格式/真错判）锚。

## 首跑实绩（2026-10-29 M1 试跑）

- 5 票回放：3/5 一致（T-0236/T-0243/T-0244 通过=gold）；2 歧=T-0238（分费制零桩测+maker_fee_bps≤0 吞负返佣合法值）+T-0241（sec 未定义白屏真病——T-0248 急修）——**皆真阳性：无头审计捕原人审漏判真病灶**。
- 归因三分类落位：两歧归『真错判(gold 侧)』——回放审计超原判深度，gold 判决录须注记 caveat 非改 expected。
- 教训：判决一致性≠审计正确性——回放驳回须先核真伪再修 runner（真金发现≠计分败）。
