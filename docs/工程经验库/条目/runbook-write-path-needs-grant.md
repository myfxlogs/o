---
id: runbook-write-path-needs-grant
title: runbook 钦定写径必须核对处置角色授权面——建表零授权=臂自始死
domain: 簿记/权限面
tags: [grant, roster_rw, fin_known_residual, 豁免, runbook, 权限缺口, 探针暴报]
---

# runbook 钦定写径必须核对处置角色授权面——建表零授权=臂自始死

## 坑形

runbook 处置档钦定「写某表」为处置动作（如 ledger_diff→fin_known_residual
豁免行），但该表建表迁移未向处置角色（FIN_PG_DSN=roster_rw）授任何权限
——处置臂从建表第一天起就机械不可达，探针因底层条件不消而**每拍暴报**，
销件只能压当下拍、下一拍同值复燃成纯噪声。

五连实证：升件 #9255→#9283→#9288→#9291→#9294（ledger_diff_okx_USDT
$0.62 同值），定性早已完成（spot 净残+尘=口径残差），缺的是落豁免行的
写径。`information_schema.role_table_grants` 查 roster_rw 在
fin_known_residual 上**零行**——表存在而没有任何角色获授。

## 判定法（处置前必核）

凡 runbook 钦定「INSERT/UPDATE 某表」为处置动作，动手前先：

```sql
SELECT table_name,privilege_type FROM information_schema.role_table_grants
WHERE grantee=current_user AND table_name='<目标表>';
```

零行=授权缺口——处置路径不是「销件+接续」而是**补授权迁移**（同
fin_0139 判词闸箱/fin_0151 票号序列先例：`IF EXISTS (SELECT FROM
pg_roles WHERE rolname='roster_rw')` guard + GRANT）。

## 授权三件套

处置席位写表所需最小集（fin_0159 形）：

1. `GRANT SELECT` ——处置前读既有行查重/算净差；
2. `GRANT INSERT` ——写处置行；
3. `GRANT USAGE ON SEQUENCE <t>_id_seq` ——bigserial 主键 nextval。

**UPDATE/DELETE 不授**：簿记面表一律 append-only 同 ops_event 律——
状态翻转靠新行+期到自动复报，不靠改行（fin_known_residual.expires_at
设计即此）。

## 伴生坑：销件不抑探针

「定性销案≠清偿」的极端形：探针条件持真+无销件抑制面+无豁免行压制
→ **每 tick 升件**（秒级）。销件前必须确认「底层条件消」或「有机制压」，
否则本轮销=下轮噪声+处置簿被同源信号刷量。
