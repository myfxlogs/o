---
id: pnl-phantom-one-sided-clear-leg
title: 净产出口径单边计益幻影——账外资产转换腿只计一腿
kind: pit
date: 2026-11-23
source: 工单#34619（fin_cashflow#1621 TRX 清仓卖腿 +9999.08 单边入账，30d 净 -1.78→+9997.30 年化 1125%）
---

# 净产幻影：卖腿计益、买腿记 adjustment（O-34619）

**坑形**：fin_cashflow 损益六 kind（fee/funding_settle/trade_pnl/
rollback_cost/spot_buy/spot_sell）求和的净产出口径，把
`position_ref IS NULL` 的现货腿当策略损益计。杂币清仓/手工转换/
未授权持仓变现的卖腿单边 +额入账，而对应取得腿要么根本不在账
（资产先于系统存在），要么被记成 `recon_adjust`（kind 集外天然
不入）——**一笔资产来回的两腿只计一腿**，净产出凭空跳变。

**判别**：净产/年化报表出现与 NAV 变动严重背离的跳变（本例
+9997 vs NAV 纹丝不动）→ 查窗内 `position_ref IS NULL` 的
spot_buy/spot_sell 腿，逐笔核有无在册取得腿。三种真实形态：
①未授权持仓变现（无取得腿，ops_event 有「白名单外未授权持仓」
升件）②取得腿记 recon_adjust（账平口径不平——禁止再加此类行
抹平，正是病因）③仓腿被 residual_clear TOCTOU 误扫成 NULL 腿
（卖腿 position_ref 丢失但买腿在册——只排 NULL 腿会把一腿
变幻影亏损）。

**修法**：算式侧判别单源化——`fin_pnl_net` 视图（fin_0236）：
六 kind∧venue<>'sim'∧ccy∈USDT/USD（native-ccy 后必须显式币域
闸，否则 BTC 域腿混入 USDT 求和单位错）∧NULL 现货腿排除；
**配对豁免**：decision 源 NULL 腿与在册仓腿（source=
fin_order_leg）intent.ccy↔inst_id base 同币、反向符号、同
venue/ccy、±120s → 误记仓腿按仓腿同口径计入（VVV#1525↔仓86
#1524 实证）。存量行零改动，口径由算式吸收。消费面（盈利报告/
饿探/自观账/死刑梯度年化/账房 earned 对账/策略归因/ops_state
commercial_stall）全走视图，不再各抄 kind 集——**同口径求和
出现第二份 kind 集拷贝=未来同型幻影的温床**。

**推广**：凡「资产转换两腿分属不同 kind/不同 source」的账，
净额口径必须先回答「两腿同口径吗」再谈求和；写侧修不完的
历史形态（手工/对账/竞态误记）一律在算式侧判别，且判别键要
落到能区分「真账外」与「误记仓腿」的粒度（币种+时窗+在册链），
粗粒度 position_ref 判别会把误记腿的对侧变成反向幻影。
