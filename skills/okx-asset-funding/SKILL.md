---
name: okx-asset-funding
description: OKX 资金户/划转/Earn 钱径能力账——/asset/* 族+/finance/savings 族真户实测、clientId 幂等键方言、581xx 错误码族
domain: 工程
---

# okx-asset-funding —— OKX 资产/停泊钱径（真户实测 2026-10-24，tools/okxverify+okxunwind）

> 分工：funding 资金户、账户间划转、Simple Earn 停泊、充提史。账户读面→okx-account；界桩裁定（何面 REST/WS）→okx-rest-boundary。

## 账户编码方言

`from`/`to` 参数：`6`=funding 资金户、`18`=trading 交易户（Earn 申购只吃 funding，赎回回 funding——交易户须划转摆渡）。

## 端点实测表（真户全通——GET/POST /api/v5/）

| 端点 | 实测 | 关键字段/口径 |
|---|---|---|
| GET /asset/balances | ok | funding 户逐币 {ccy,bal,availBal,frozenBal}——**funding 户无 WS 推送频道，唯此 REST 面**（界桩钦定） |
| POST /asset/transfer | **ok 真钱往返** | {type:"0",ccy,amt,from,to,**clientId**}——trading↔funding 双向实证（clTId 826xxx 在案） |
| GET /asset/transfer-state?type&ccy&clientId | **ok** | 按 clientId 定划转真态 success/failed——**真单查证在案** |
| GET /asset/deposit-history | ok | 充币史 {ccy,amt,state,ts} |
| GET /asset/withdrawal-history | ok | 提币史同构 |
| GET /asset/currencies?ccy | ok | 链信息（USDT-Arbitrum One 等在案） |
| POST /finance/savings/purchase-redempt | **ok 真钱** | side=`purchase`/`redempt`——赎回 T+0 到 funding；**大额实证 2139 USDT 赎+购全链** |
| GET /finance/savings/balance | ok | Earn 持仓行 {ccy,amt,earnings,pendingAmt}——**遍历所有行找 ccy**（首行序不定，取首行=漏读坑实证） |
| GET /finance/savings/lending-rate-summary?ccy | ok 公开 | avgRate=24h 平均年化（USDT 实测 0.035=3.5%）——Earn 现率底取值源 |

## 坑

- **clientId 方言（2026-10-24 真单实证）**：transfer POST 体幂等键字段=**`clientId`**（写 `clTId` 被静默吞——划转成功但键没存）；transfer-state 查询参名同为 `clientId`（写 clTId=→50015 缺参拒、clientId=不存在值→51000 校验拒=端点在岗非可达性证）。
- **savings/balance 多行遍历**：rows 按 ccy 多行，取首行=可能拿到非目标币尘行（曾致「Earn 无 USDT」误判→2139 USDT 停泊仓差点被漏看）。
- **581xx 族**：58110 风控锁（业主配合域）、58111 funding 结算窗撞期=瞬态可重试、58113 币不可划、58114 额≤0。
- **歧态势**：transfer 传输败→transfer-state 按 clientId 定真态，禁盲重发（同 clOrdId 律）。
- **复位径纪律（critical——probe-restore-scope-overreach 条目）**：Earn/资金户归集只许冲销台账内条目，**存量资金禁动**——按现值「有多少抽多少」式调用是越权动业主资产（$2145 事故实证）。

## 消费面
- NAV 三户口径（T-0086）：资金户/Earn=REST 活读零落地（/asset/balances 逐币、/finance/savings/balance 逐行 amt——遍历所有行找 ccy，首行序不定坑在案）；缺户=0 不误报。
- 对账白名单：账外币种按身份豁免（在册=fin_cashflow 有账币种），粉尘/杂币涨跌不报警。

## 复跑

`tools/okxverify -mut`（划转往返+Earn 赎购真单谱）；`tools/okxunwind -ccy <X>`（回卷件——限价保护卖+摆回 Earn）。
