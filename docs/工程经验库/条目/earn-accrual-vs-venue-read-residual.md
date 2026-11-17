---
id: earn-accrual-vs-venue-read-residual
title: 对账残差=未实现Earn应计利息——accrual入账 vs settlement读面口径差（#15636 机制定案）
kind: pit
date: 2026-11-18
source: 工单#15636 accrete-residual-audit（五代认领接力实证——增生~0.8/日谜团终解）
---

# accrual 口径入账 × settlement 口径读面 = 未实现应计残差逐日增生

**坑形**：`ledger_diff_okx_USDT` 残差逐日增生 ~0.8（豁免链 #4→#11
滚动续命五代）。分解到机制层：

- **账本侧（accrual 口径）**：`postInterest`（internal/ledger/run.go）
  按 savings `earnings` 字段高水位 delta 把 Earn 利息**逐拍应计入账**
  ——fin_cashflow `kind='interest'` 行随 accrual 实时进账本净额。
- **venue 侧（settlement 口径）**：`fundSaveBalances`（cmd/core/
  restlive.go）读 `/finance/savings/balance` 只取 **`amt`**——
  OKX 实测语义 `amt=loanAmt+pendingAmt`（出借在途本金），应计利息
  在 **`earnings` 另列**（累计 lifetime 值，不入 amt）。
- **结果**：每行 interest 把「账本净−venue权益」差推宽≈利率额；
  利息**实现**（复利入 amt/派发 funding 户 bills subType381→
  supply_income/赎回回款超本金）时 venue 侧才认账，差回缩。
  差值=**未实现应计利息存量**，随 accrual 率逐日增生、赎回/派发
  时回缩——机制永续不除非改口径。

**实证锚**（2026-11-18 reconview+psql）：累计 interest 入账
13.1318（531 行自 10-24）vs 现 diff 12.88 ≈1:1；11-09~14 accrual
~0.85-0.9/日≈增生~0.8/日 1:1（#20213 已锁），11-14/15 earn 率断崖
（日 0.88→0.07）增生同停——增生源=accrual 率非外部移动。
USDT savings 实测 amt=964.42476114=loanAmt+pendingAmt 精确成立，
earnings=85.36（lifetime 累计，含记账起点前存量——基线前 accrual
双侧同不计不入差）。

**判别**：对账差增速≈某 cashflow kind 日聚合额（本案=interest
日行和）且 venue 侧对应资产面有「账外应计字段」——先查 venue
读件取的是本金面还是全值面（amt vs amt+earnings 类字段分列
=高危口径点）。

**修法候选**（设计权归唤知/决策——钱径口径非施工自裁）：
①venue 侧加「未实现应计」分量=earnings_now−基线（基线=记账起点
  earnings 值，须 ops_state_row 类机制态存；earnings 是 lifetime
  累计值，实现事件是否回减语义未实证——盲加=实现后双计）；
②账本侧改 realization 入账——accrual 不记，实现时记（funding
  bills 381→supply_income 已在册自动命中；复利入 amt 须侦测
  amt 无购买增量；赎回息差部分须拆——侦测面不 trivial）；
③认领为已定性残差——fin_known_residual 机制豁免+机制写因
  （现状即此径滚动豁免，增生不停则豁免滚动续命）。

**推广**：凡「账本 accrual 入账 × venue settlement 读面」对拍架构，
未实现应计=恒差增生源——设计时二选一钉死（读面含应计∨账本延到
实现），混用=逐日残差。另坑：`fin_ledger_snapshot`/daySnapshot
按 ts 窗外累计和——**backfill 行（occurred_at≪ts）会追溯改写
历史日净额**，日差分解禁用快照差分当流量（本案 +6.16 跳变实为
存款史/对账 backfill 落地非当日流量）。
