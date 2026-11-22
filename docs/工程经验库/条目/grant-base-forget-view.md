---
schema_version: 2
id: grant-base-forget-view
kind: pitfall
risk: med
status: verified
keywords: [GRANT, 视图, o_fin_ro, fin_cashflow_domain, migrations, DO块, role_table_grants, 授底忘视, 法证对拍]
scope: [migrations, o_fin_ro 读面, fin_cashflow_domain, 审计劈⑤对拍面]
summary: 迁移 DO 块 GRANT 只发底表 4 件漏授视图本体——fin_cashflow_domain 视图 o_fin_ro 无 SELECT=劈⑤对拍面自身黑盒；授权面验收必逐对象核 role_table_grants（表与视图分行）
first_seen: 2026-11-21
last_seen: 2026-11-21
evidence_refs:
  - type: ops_event
    ref: 升件#28891（审计/法证对拍 RED 实证）
  - type: ticket
    ref: 工单/finro-domain-view-grant
---

## 授底忘视——GRANT 底表齐而视图本体漏授（对象粒度清单缺口）

- **病灶**：`migrations/fin_0226_finro_audit_read.sql` DO 块向 o_fin_ro
  授法证读面——底表 4 件 GRANT 齐，而 `fin_cashflow_domain` **视图本体**
  漏授 SELECT。role_table_grants 视图行=0→对拍面（劈⑤）读自身黑盒。
- **根因形**：授权清单按「底表集合」枚举，视图当「派生物默认随底表」——
  PG 语义否：视图是独立 securable，底表授了视图不随；owner 建视图者
  能查≠授权角色能查。审计断言 `domain-view-readgrant.sh` RED 钉。
- **判则**：迁移含 GRANT 的对象清单必须**逐对象枚举**——表/视图/序列
  分列，验收径 `role_table_grants WHERE table_name IN (全清单)` 按行数
  对拍清单长度；「授底即授视」是语义臆断非实测。
- **排查捷足**：法证读面疑似黑盒先跑
  `SELECT table_name,privilege_type FROM information_schema.role_table_grants
   WHERE grantee='o_fin_ro' AND table_name LIKE '%domain%'`
  ——行缺即漏授实锤，勿先查 RLS/owner。
