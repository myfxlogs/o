---
schema_version: 2
id: stack-identity-unpinned-false-evidence
kind: pitfall
risk: medium
status: observed
keywords: [双栈, dev 栈, SG 栈, 栈身份, 伪证, 考古, fin_host_metrics, o-postgres, 同名表]
scope: [运维考古, 观测面, 环境拓扑]
summary: dev 机与 SG 各持一套同名栈（o-postgres/o-core/同名表）——观测证据不钉「哪个栈哪个库」即成伪证：T-0434 票面尸体行（450G/04:18）实为 dev 栈库行，SG 栈库同期两行俱鲜；混对拍把考古拖入小时级弯路。
first_seen: 2026-11-07
last_verified: 2026-11-07
skills: []
evidence_refs:
  - type: doc
    ref: docs/tasks/T-0434-hostmetrics监控自监-写器死活根修与前端陈旧硬闸.md（票面三查数据=dev 栈库行值亲证吻合：449.6G/252.0G/04:18:44 唯一行）
  - type: provenance
    ref: SG o-postgres fin_host_metrics 两行俱鲜（dev 44.07G/sg 57.97G，2026-11-07 06:44 UTC 实拍）vs dev o-postgres 唯一冻结行
---

## 事实/症状

T-0434 死因考古中：票面证据（唯一行 'sg'、450G 盘值、df 97%）与 SG 生产栈现状全面对不上（SG 盘 58G、表有两行、行鲜）；逐步排查后发现同名表在 dev 栈库（o-postgres@dev :5432）中另有——票面所有数值与 dev 栈库行精确吻合。两个栈的 o-postgres/o-core/表名完全同名，跨栈对拍=伪证。

## 根因或策略解释

dev 机=/opt/o 部署树+完整 compose 栈（开发环境），SG=生产栈——**同名异命**。观测证据（行值/df/RPC 应答/页面）不先钉「栈+库+主机」三身份即引用，会让后续考古者把 A 栈事实当 B 栈病灶查（本次烧约两小时才收敛）。附带发现：dev 栈 o-core 进程 10-30 04:18:44 UTC 死后无人复活——dev 栈观测面 8 天无人知（与 self-report-writer-death-unmonitored 同案异面）。

## 处置

**纪律（记坑即修的用法面）**：任何票面的生产数字/表值/df 证据，必须携带「栈（dev/SG/OCI）+ 库实例 + 主机」三身份锚；跨栈引用前先 `hostname`+DSN 亲证。dev 栈长期死进程是否复活/退役=ops 裁量件（留外审）。

## 证据

- check: dev 机 `docker exec o-postgres psql -U o -d o -c 'SELECT host,ts,disk_total_gb FROM fin_host_metrics'` → 唯一行 449.6G @ 10-30 04:18:44
- check: SG 同查询 → 两行俱鲜（44G/58G）
- result: 两库同名表内容分叉，票面证据=dev 侧

## 反证条件

- falsifies_when: 出现栈身份三锚齐备仍对不上的观测证据——另有未知实例在案

## 适用边界

只覆盖 O 现有 dev/SG/OCI 三面；新栈接入时本条即出生律：栈名入证据锚。
