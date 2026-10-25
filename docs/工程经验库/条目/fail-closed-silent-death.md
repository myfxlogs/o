---
schema_version: 2
id: fail-closed-silent-death
kind: pitfall
risk: high
status: canon
keywords: [fail-closed, 让位, 静默死亡, 429, 限频, 臂静默, 告警缺口]
scope: [research 采集臂, fail-closed 判据径, 限频退避]
summary: fail-closed 让位循环无告警=臂静默死亡——读败让位是正确的，但「每拍都让位+零产出+零告警」=在编探员形同虚设 N 小时无感知
first_seen: 2026-10-25
last_verified: 2026-10-25
skills: []
evidence_refs:
  - type: incident
    ref: SG boot 日志 research-dcd/gap 同打 DCD 端点双 429（2026-10-25 12:25Z）→md_dcd_product 建表以来 0 行
---

## 病灶

DCD 枚举臂 boot 撞 429 → fail-closed 让位 → 下拍 30min 再撞再让——
模板在册+enabled 但从未评估一行；`fin_opp_eval` dcd_* 零产出无告警。
**让位径正确≠告警径在场**——fail-closed 语义只防「假读」，不防「永不读」。

## 修法

①瞬态限频 429/5xx 加指数退避重试（≤3 次）非直接让位；
②臂「上次成功产出」戳记+超 2×节拍告警一次（research_arm_silent）；
③采集节拍错峰/共享件面（同端点双件齐打=限频源头自伤）。
