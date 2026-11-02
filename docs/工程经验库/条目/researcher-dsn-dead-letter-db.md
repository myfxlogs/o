---
schema_version: 2
id: researcher-dsn-dead-letter-db
kind: pitfall
risk: medium
status: verified
keywords: [fin_opp_hypothesis, 研究员, DSN, 双库漂移, 队列正本, PG-md, 死信]
scope: [researcher-spawn, tools/hypo, OCI PG-md, dev 本地库]
summary: 研究员 spawn 缺省 DSN=localhost:5433（dev 本地研究库）——fin_opp_hypothesis 正本在 OCI PG-md（md 域），落 dev=死信：队列盘/消费臂/度量全盲。
first_seen: 2026-11-02
last_verified: 2026-11-02
skills: [研究员]
evidence_refs:
  - type: doc
    ref: /opt/meta-archive/bin/researcher-spawn.sh
  - type: server_fact
    source: postgres
    query: SELECT id,surface,state,evidence->>'provenance' FROM fin_opp_hypothesis
    observed_at: 2026-11-02
  - type: doc
    ref: docs/tasks/T-0295-研究员落库径归正.md
---

## 病灶

`researcher-spawn.sh` 产线缺省 `DB_DSN=postgres://o:o@localhost:5433/o`
（dev 盒本地研究库）。但 `fin_opp_hypothesis` 是 md 域表（md_1045 迁移、
domain.tsv=md）——分域切换（2026-10-28 研究臂迁 OCI）后队列正本=OCI 盒
o-postgres-md，生产臂（假想官/文法/xcarry）主池同写彼处。落 dev 副本的卡
不进任何评审/消费/度量链——纯死信。

## 实证链

三库对拍（2026-11-02）：研究员卡 #44/#45 只在 dev:5433；OCI-md 有假想官
#15/#16+xcarry #84 无研究员卡；SG 有 grammar 7+假想官 #120（分域前遗存）。

## 泛化律

- 「产线缺省 DSN」移交项悬空=产出静默落错库——验收票面凡含「生产 DSN
  换正」移交项，放行 timer/跑批前必须先核目标库是域正本。
- 分域切换（md/fin 两 PG）后凡新写径件：DSN 目标按 domain.tsv 域册对位，
  不按历史惯性（fin_* 命名≠fin 域——fin_opp_* 全系 md 域）。
- 表正本判据：写它的生产臂主池指向哪（o-research SG_PG_DSN 名残实指
  OCI :5435）+迁移链文件前缀（md_*→PG-md，fin_*→SG）。

## 处置

裁决两卡双端落账（dev 翻态+OCI-md 迁移录 #1690/#1691）+立 T-0295 归正
写径（o_researcher 建 OCI-md+spawn DSN 归正+timer enable）。
