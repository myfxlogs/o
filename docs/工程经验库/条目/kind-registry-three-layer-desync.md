---
schema_version: 2
id: kind-registry-three-layer-desync
kind: pitfall
risk: medium
status: verified
keywords: [fin_cashflow, kind 闭集, fin_cashflow_kind_class, fin_cashflow_kind_check, 册载零写者, 三层闭集, unmapped_bill, bill 子型映射]
scope: [新增 cashflow kind 的注册径, bill ingest 分派面, 闭集约束多面同步]
summary: cashflow kind 存在三层独立闸——class 册(fin_cashflow_kind_class)/Go Post() kinds map/DB CHECK(fin_cashflow_kind_check)。fin_0075 只注 class 册未更另两层→dust_in/dust_out/supply_income/airdrop 四 kind 册载而物理不可写=「册载零写者」病灶；审计升件#14671 以此为现金守恒断根因之一。修 kind=三层同更+迁移链登记(migrationTable)齐位。
first_seen: 2026-11-13
last_verified: 2026-11-14
skills: []
evidence_refs:
  - type: doc
    ref: migrations/fin_0174_bill_kind_close_set.sql（DB CHECK 补五 kind）
  - type: doc
    ref: internal/ledger/ledger.go kinds map（Go 闭集同步）
  - type: doc
    ref: internal/ledger/funding/ingest.go 子型映射+unmapped_bill 残差支
  - type: server_fact
    source: postgres
    query: SELECT kind,class FROM fin_cashflow_kind_class
    observed_at: 2026-11-14T03:50:00Z
    server_revision: ""
    response_hash: ""
---

## 事实/症状

fin_0075 在 fin_cashflow_kind_class 册载 dust_in/dust_out/
supply_income/airdrop 四 kind（「正为此域生」），但写侧两层闸未同步：
- Go `ledger.Post()` kinds map 无此四键→Post 径拒写；
- DB `fin_cashflow_kind_check` CHECK 无此四键→直写 SQL 亦拒。
结果：册载语义完备而物理零写者——审计二巡实测 bill ingest 未映射
子型 balChg≠0 静默零覆盖=对账 phantom diff 源（升件#14671 P1）。

## 根因

kind 的「合法」由三层各自为政判：class 册（语义层）、Go map（API 层）、
DB CHECK（物理层）。新增 kind 只注其一即语义/物理断裂——册面说合法
而写径全拒，无任一面告警，静默型病灶。

## 修法/判据

①新 kind=三层同更：kind_class INSERT（class 归属钦定——unmapped_bill
归 adjustment 待归因非 income/transfer）+ Go kinds map + DB CHECK
（DROP/ADD 迁移，免疫面 trailer）。
②迁移文件还须登记 migrationTable（internal/store/pgstore/
migrate_table.go）——未登记文件不执行=第四层隐闸（本修复首跑即中：
迁移在册外致 class 册未补、trigger 产 NULL class 拒写）。
③配对判据防漏：新 kind 若有写者分派面（ingest 子型映射），子型号
以 venue 权威典为准（/api/v5/account/subtypes 实拉）不猜。

## 通用化

凡「册载合法集合」被多层闸各自判定的域（kind/事件词/角色/seat），
增删集合成员=全层盘点——只更一层=册载零写者或写者无册两种断裂。
check 面已有桥审/md 键域断言同款思想；新增闸面时把「成员同步」写成
断言优于事后追。
