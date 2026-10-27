---
schema_version: 2
id: psql-c-autocommit-test-row-pollutes-prod
kind: pitfall
risk: critical
status: verified
keywords: [psql -c, autocommit, 逐语句提交, 测试数据污染, 生产库, ROLLBACK 失效, fin_cashflow]
scope: [生产取证, 测试, psql, 账链]
summary: psql -c 逐语句自动提交——测试 INSERT 打生产库后跟独立 -c ROLLBACK 零回效；测试复现必须 dev 影子库或单 -c 内显式 BEGIN/ROLLBACK 包裹
first_seen: 2026-10-27
last_verified: 2026-10-27
skills: []
evidence_refs:
  - type: commit
    ref: 6cdbe1f
---

## psql -c 自动提交：测试插行打生产=真账污染

- **实证**：T-0173 复盘页验收期，为复现「venue 筛 42P18」在 SG 生产库（o-postgres 15433 容器）用 `psql -c "INSERT ..."(source_ref=99,source_table='t')` 后跟独立 `-c "ROLLBACK"`——**psql 无显式事务时逐语句自动提交**，INSERT 即刻落账，ROLLBACK 空转。假行（interest/mt5/+2 USDT）被总览「最近动作」实读显形，业主客户端对账无此入账——真账污染实锤（fin_cashflow id=435）。
- **害**：interest 属经营性四件——当日盈亏/累计盈亏实时求和多计 $2；若留到 00:00Z NAV 日拍即固化进净值曲线。
- **教训**：①测试插行的合法落点=dev 影子库/testutil 隔离 schema，**生产库零写**（生产取证 skill 硬线「只读」的应有之义——取证=读）；②必须事务性复现时，同一 `-c` 内 `BEGIN;INSERT;ROLLBACK` 包裹或 `-f 事务文件`——跨 `-c` 的 ROLLBACK 是安慰剂；③污染自证指纹：测试行带 `source_table='t'` 类标记，污染枚举 `WHERE source_table='t'` 一查即全。
- **收口实证**：`DELETE FROM fin_cashflow WHERE id=435 AND source_table='t' AND source_ref=99`（双条件防误删真账），残量=0，pnl_today 回落 0.2292 真值。
