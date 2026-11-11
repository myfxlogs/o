---
id: dir-loss-breaker-caliber
title: 方向族亏损口径——腿集同边机械判别+实现浮亏合并
kind: caliber
date: 2026-11-11
source: 工单#10413（业主令⑧）
---

# 方向族亏损口径（工单#10413 钦定先行件）

**族判别**：不凭 template 名册——在仓行腿集全同边（fin_order_leg filled
count(DISTINCT side)=1）即方向位。hedged 结构（funding-hedge/basis/
xvenue-carry 双腿对称）腿集双边自然排除；DCD 类无腿产品不在测域
（其险由 dd_30d 账本线与族级退役闸分守）。族粒度=template_ref。

**亏损口径**：族级累计亏损 = 实现 + 浮亏合并。
- 实现 = fin_cashflow 六损益 kind（fee/funding_settle/trade_pnl/
  rollback_cost/spot_buy/spot_sell，venue<>'sim'），
  position_ref→strategy→template_ref 全史累计。
- 浮亏 = 族在仓方向位逐 inst md_ticks 现读 mark 盯市
  （net_qty×last − Σqty_base×px；mark 缺=该 inst 让位计数不虚构）。
- 本金 = md_equity_curve ccy='NAV' 末行 <48h；缺/陈=fail-closed 本拍不判。

**闸阈**：`guard.dir_loss_budget_pct`（迁移播种 0.10=本金 10%，dm 域热启可改）。
触发=模板禁+族策略合法降级（live/probe/paper→degraded，proposed 不入——
生命周期闭集同律）+升件待业主复审+复用 tend 径全平（闸/反向原子平/节拍重试；
畸形/单腿仓升件载明待人平不盲平）。

**复位**：`guard.dir_broken.<T>` latch 永不自清——解封=显式热改 false+
模板 enabled 回填+策略回迁（业主复审件，dd_30d 同律）。
