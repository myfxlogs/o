---
schema_version: 2
id: pnl-face-divergent-caliber-sum-null-collapse
kind: pitfall
risk: high
status: verified
keywords: [盈亏口径, 三面三数, fin_cashflow, spot_leg, bare_bag, sum_null, T-0422]
scope: [ledger, report, cockpit, overview, caliber]
summary: 业主盈亏面各写各的口径 SQL 必漂→三面三数；spot 腿按腿发生日计=开仓置换流出当亏；多子查询 sum() 空集 NULL 塌全式为 0
first_seen: 2026-11-07
last_seen: 2026-11-07
evidence_refs:
  - type: ticket
    ref: docs/tasks/T-0422-盈亏口径三处不符-日报净额累计净值与前端累计亏对账.md
  - type: commit
    ref: t/0422-pnl-caliber
---

## 盈亏面口径各写各漂+leg 归属错位+sum() NULL 塌式三坑合体

- **实证（生产对拍 2026-11-07）**：日报净额 -48.79/累计净值 -46.89/前端累计亏
  -21.86 三面三数。逐仓逐笔拆：11-06 两笔 CORE 开仓买腿 -49.99——pos33
  假亏件（T-0419）+pos34 未平仓置换流量；cum_nav 按 class='income' 无上界
  吞未来行+rollback_cost 缺册（class=''）漏计；前端 4-kind+已平仓腿不冲回
  bare_bag；驾驶舱 NetMonth/Daily 5-kind 全腿=最差第四面。
- **坑①口径多源**：同一「盈亏」概念三面各写 SQL，集合/归属/时界三轴全漂。
  治=单源函数（report.Realized）四面同读，面侧禁另写口径 SQL。
- **坑②腿归属错位**：spot 腿按腿发生日计入日净=开仓置换流出（资产仍持、
  NAV 已计值）被当亏损。钦定归属=平仓日（closed_at 窗——实现时点），
  未平仓腿不入实现盈亏；裸袋挂账冲回同窗（未回资产非实现损）。
- **坑③sum() 空集 NULL 塌式**：多子查询相加 `a + (SELECT sum(x) ...)`
  ——子查询空集 sum=NULL 致全式 NULL（COALESCE 外层兜 0 反吞真值）。
  每个 sum 子查询须各自 COALESCE((SELECT sum...),0)。桩测锚：
  TestWeeklyAnnualized 抓现行（现金 15+腿 NULL+袋 NULL=0 而非 15）。
- **收口**：fin_0133（rollback_cost 入 income 册+class 漂移重刷+fin_report
  派生行按统一口径重算）+caliber.go+口径定义节 docs/运维/周报口径.md。
  统一口径实证：11-06 日净 +0.83/累计 +3.34（原三面读数全系假亏）。
