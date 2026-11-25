---
id: venue-manual-action-invisible-ledger-phantom
title: venue侧手动动作不入簿=ledger_diff幻影残差复发源——deal/bill全谱必须翻译进fin_*
kind: pit
date: 2026-11-25
source: 工单O-33207（升件#39948销案判词「三件机制缺」——MT5出金采集径/手动买单簿记径/GRANT已落）
---

# venue 手动动作不可见病灶（O-33207 根修）

**坑形**：O 只见「自己发起的资金流/仓位」——业主在 venue 侧手动出金/
充值/买卖，对账本结构性不可见：venue 真账动了、fin_cashflow 没腿
→ ledger_diff 幻影残差连升件；手动仓 equity 含浮盈 net 不含 →
NAV 口径假。MT5 侧曾有 balanceOpsLoop 只认 Balance 充提行，交易行
（DealBuy/Sell）静默略、Credit 邻类只注记不入账、邻类金额静默吞差；
OpenedOrders 在仓根本没进簿记面。

**判别**：幻影残差升件若伴「业主近期在 venue 侧动过手」（出金/入金/
手动下单）——先查 venue deal/bill 全谱有没有被静默丢弃的行型
（DealType/OrderType/bill subType 枚举逐值对账采集器分派表），
不是先怀疑对拍公式。

**修法**（O-33207 落地形）：venue 事实面全谱翻译——
①deal/bill 每个枚举值必须有归属（入账 kind∨守恒兜底 unmapped_bill
∨显式注记），**静默略=病灶**；②手动仓=独立事实面
（fin_venue_position）非塞 fin_position（execution_id NOT NULL
装不下无 O 单的仓）；③在仓浮盈必须并入对账 mtm（equity 含它
net 不含=残差）；④O 单排除序先行（comment=clOrdID/instId 命中
在持集=判 O 的不入手动册——双簿防线），判不出=保守判 O 的
（宁缺毋滥不双簿）。

**推广**：新增 venue/新增 deal 型/bill 子型时，采集器分派表必须
全枚举覆盖——「没见过的型」默认落守恒兜底（unmapped_bill）不落
静默；对账残差升件先查「venue 有而簿记无」的行型清单再查公式。
