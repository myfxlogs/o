---
schema_version: 2
id: grant-not-in-migration-silent-dead
kind: pitfall
risk: high
status: verified
keywords: [GRANT, permission denied, 迁移, roster_rw, projector_rw, 静默哑死, "|| echo 0", fin_econ_lesson, md_regime]
scope: [migrations, internal/projector, scripts/meta, SG/OCI 消费凭据面]
summary: 新表迁移只 CREATE 不 GRANT——消费角色（projector_rw/roster_rw）零权，读径 error 被兜底吞成「零行/skip」静默哑死，fin_0100/fin_0102 三犯实证
first_seen: 2026-11-03
last_verified: 2026-11-03
skills: [生产取证, 协议面取证]
evidence_refs:
  - type: doc
    ref: migrations/fin_0100_regime_mirror.sql（漏 projector_rw 四权——T-0312 第一刀返修①补）
  - type: server_fact
    ref: fin_0102 fin_econ_lesson 漏 roster_rw SELECT/UPDATE——verdict_tick 闸询「|| echo 0」吞成 pending=0 永 skip；手工 GRANT 解锁（2026-11-03）
---

## 情境

新消费面表迁移件只写 CREATE TABLE 不写 GRANT——生产者角色（o-core
本体池）能写，但消费角色（projector_rw 投影凭据/roster_rw 席位隧道凭据）
零权。

## 实数

- fin_0100 regime 镜像（T-0312 第一刀）：projector_rw 四权漏→部署即
  permission denied，返修①补条件 GRANT 块；
- fin_0102 fin_econ_lesson（T-0314/第三刀）：roster_rw 零 SELECT/UPDATE
  →`verdict_tick.sh` 闸询 `|| echo 0` 把 error 吞成 pending=0→裁定席
  永不烧——链看着活实际哑死；
- 更早 fin_0098/0099 同款前科在案（票据记）。

## 根因

消费角色授权是「迁移外知识」——DDL 写者盯着表结构不盯着凭据面；
上层兜底（`|| echo 0`/`|| true`/降级静默）把 permission denied 转成
「零数据」语义，故障不炸只哑，最难自查的一类病。

## 判据挂接

迁移自审必查：DDL 块写完即问「**谁在 o 超户以外角色读写这表**」——
每枚消费角色同迁移条件 GRANT 块列权（SELECT/INSERT/UPDATE/DELETE
按件需）；部署验收须用**消费凭据真连真查**（o 超户查过不算过）。

## 配套纪律

手工补权=解锁术不是正本——须立 fin_NNNN 迁移把 GRANT 写回
canonical 链（不入链=下回全新部署又哑）。
