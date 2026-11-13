---
schema_version: 2
id: ops-user-no-fin-read-gauge-parse
kind: pitfall
risk: low
status: verified
keywords: [权限, fin_position, FIN_PG_DSN, 量表, self_state, armed_distance, 监工哨, 指标口径]
scope: [tools/, scripts/, 外席工具面]
summary: ops_event 读写户（FIN_PG_DSN）对 fin_* 钱径表无 SELECT 权——工具/哨件直查
  fin_position 等表=permission denied 静默 -1；活性指标须改从 ops_event 量表行解析
  （self_state 仓N/armed_distance 近阈%/归因行 赚[]），既免授权面又与量表同口径
first_seen: 2026-11-13
last_verified: 2026-11-13
skills: []
evidence_refs:
  - type: doc
    ref: tools/overwatch/main.go
---

# ops 户无 fin_* 读权——工具面指标改从量表行解析

**坑形**：写工具/哨件时用 `FIN_PG_DSN` 户直查 `fin_position`/`fin_cashflow` 等
钱径表——ops_event 读写户对这些表**无 SELECT 授权**，查询 permission denied；
若错误被吞（如计数函数返 -1）则指标静默错值，哨面显「-1」而无人知查询败。

**实案**：overwatch 首版 `SELECT count(*) FROM fin_position WHERE opened_at>…`
→ permission denied → 指标 -1。手动复跑同样报错确证。

**修法**：活性/状态指标**不从钱径表直取**——ops_event 量表行就是同源口径：
- 在仓数：self_state 末行 `仓(\d+)`（`live34/probe16/仓1`）
- armed 距阈：armed_distance 末行 `近阈 \S+ (\d+)%`（36%）
- 赚面：归因行 `赚\[([^]]*)\]`（空=策略面无正产）

**辅坑**：`ts>now()-interval $n::interval` 参数化区间 PG 不收（interval 字面
语法非函数）——抑窗改字符串拼接 `interval '24 hours'` 或 `make_interval(secs=>$n)`；
且 dedup 查询失败返 -1 时**必按「已报」处理=旗标静默全灭**——查询错误要响 stderr。

**判定**：工具/哨件跑通后必抽查「指标值≠-1 且旗标实插行」——两处静默源都踩过。
