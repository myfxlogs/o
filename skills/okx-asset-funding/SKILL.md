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
- **amt 不含 earnings（T-0087 实测）**：`amt=loanAmt+pendingAmt`（出借在途本金），应计利息在 **`earnings` 另列不入 amt**——venue 对拍聚 savings 须 `amt`（本金面）与 `earnings`（利息面）分读：本金入对拍、利息走 interest 高水位入账，混读=双计或恒差。
- **581xx 族**：58110 风控锁（业主配合域）、58111 funding 结算窗撞期=瞬态可重试、58113 币不可划、58114 额≤0。
- **歧态势**：transfer 传输败→transfer-state 按 clientId 定真态，禁盲重发（同 clOrdId 律）。
- **复位径纪律（critical——probe-restore-scope-overreach 条目）**：Earn/资金户归集只许冲销台账内条目，**存量资金禁动**——按现值「有多少抽多少」式调用是越权动业主资产（$2145 事故实证）。

## 消费面
- NAV 基金口径（T-0095 v2，替代 T-0086 三户全币）：交易户=USDT∪**open** 腿基币行 eqUsd 过滤求和（tradingFundEq——不按全账户取）；资金户/Earn=REST 活读零落地（/asset/balances 逐币、/finance/savings/balance 逐行 amt——遍历所有行找 ccy，首行序不定坑在案）但只出基金口径币种行；集合外=托管尘出 NAV 出卡（零遗留概念）；缺户=0 不误报。T-0096：MT5 净值随 NAV 全计（mt5/USD 恒在册）。
- 对账白名单（T-0095 v2）：ledger.FundCcy 唯一真源（USDT∪open filled 腿基币，LedgeredCcy 已退役）——账外尘非「豁免」而是口径内天然排除（venue 侧同口径过滤求和）；fin_known_residual 豁免面退役（0042 清 T-0080 行）。
- 残币活期归口（T-0095 v3）：腿平事件+巡检拍双触发（ev_position+ev_account）——在持腿基币之外交易户非 USDT 余量，≥$1 起投底=交易户→资金户→savings purchase 同币零换手（Earn 申购扩币种，USDT 闭集破）；申购业务拒=确认未动回摆+fin_residual_custody 托管标（**禁兑换径**——摩擦>本金，同复位径纪律：残币处置权在业主）；<底额=托管标幂等（30 天窗）。
- 双币赢枚举消费（T-0102）：internal/research/dcd 臂从 `fin_capability_surface` 注册表读枚举面（注册+fin_opp_template 模板适用声明=探员即枚举，零新代码扩张）；扫 currency-pair→products?baseCcy&quoteCcy&optType 全对在架（三参方言——uly 伪参 50014 实证，勿猜参名）（信封数字 code 已由 okxexec flexCode 兼容、拨号钉 tcp4——50110 IPv6 坑根修）；评估落 fin_opp_eval（dcd_call/dcd_put）→阈闸热启（dist≥8%/term≤30d/notional≤avail×20%/单 uly 方向不叠/apr≥5%）进 fin_proposal 同链；读败 fail-closed 不造零快照。
- 差扫岗（T-0102 v3/v4）：目录探针（种子=dcd currency-pair+earn-flex lending-rate-summary 两实测面，扩面=补行+证据禁猜端点）→对拍注册表→fin_surface_gap 队列；面分类器全机审（yield_bearing→carry / funding_bearing→hedge / swap_sign 正→carry 负→hedge / options→vol 后期）自动生候选走毕业线（provenance=surface_gap，gen_cap=3 防洪水）；瞬态读败不裁死、死面 autorejected+拒因、终态不机器翻案。

## 复跑

`tools/okxverify -mut`（划转往返+Earn 赎购真单谱）；`tools/okxunwind -ccy <X>`（回卷件——限价保护卖+摆回 Earn）。

## 结构化产品面（双币赢 sfp/dcd——2026-10-25 决策席实证补件）

> 权威文档锚：https://www.okx.com/docs-v5/zh/#financial-product-dual-investment

- 端点族 `/api/v5/finance/sfp/dcd/*`：**`currency-pair`/`products?baseCcy&quoteCcy&optType` 读面实测 200**（BTC-USDT/ETH-USDT 的 C/P 全谱——产品 id 形如 `BTC-USDT-261028-83700-C`，载 strike/expiry/annualizedYield/minSize/maxSize）；`quote`/`trade`/`redeem-quote`/`redeem` 为 POST 写面。
- **订单枚举面**：`GET /order-history`（可选 ?uly= 过滤）**实测 200 返全谱订单**（LIVE/SETTLED 含 notionalSz/notionalCcy/strike/settleTime/annualizedYield/yieldSz/settleSz 结算落果）；`GET /order-status?ordId=X` 单查。注意 `orders`/`orders-list` 等非名 500——端点名以文档为准勿猜。
- **资金流水面**：dcd 申购/回款/收益/退款落 bills subType **160/161/162/163**（精简版 207/208/209/210）——查证走 `/account/bills`、`/asset/bills` 按 subType 过滤。
- 响应信封坑：dcd 族 `code` 字段是**数字**非字符串（`{"code":0,...}`）——okxexec.Client.Get 的信封解析（string code）会拒，须裸签名自解或信封兼容改（经验库 `okx-dcd-endpoint-dialect`）。
- **网络坑**：OKX 白名单按源 IP 判——IPv6 egress（无外网 v4 时双栈漂移）触发 50110 间歇拒绝；同机同钥匙间歇 401=查出口栈族，非钥匙坏。
- 定位：双币赢=卖 call 收权利金型 carry（在架 BTC-USDT call 年化 13–72% 实证）——结构产品面候选策略模板，评估须走票面生命周期。生产实持仓：ordId=11448262 0.0207 BTC LIVE（T-0101 入账票）。
