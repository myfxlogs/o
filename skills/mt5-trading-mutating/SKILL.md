---
name: mt5-trading-mutating
description: mtapi MT5 mutating 面（OrderSend/Modify/Close）——demo+真户（EXR 3001157，业主令 2026-10-24 授权）双全谱实测在案；ChangePassword 自伤型永禁
domain: 工程
---

# mt5 mutating 面（实测 2026-10-17，证据=docs/能力账/mt5-rpc-probe-2026-10-17.json）

| RPC | 功能 | demo 实测（2026-10-17，BTCUSDm） | 关键字段 |
|---|---|---|---|
| OrderSend | 市价开仓（0.01 BUY）/挂单（BuyLimit 深价） | **ok**：ticket=2701577885（市价）/2701577888（BuyLimit@1000 深价挂）；**2026-10-24 复验 ok**：ticket=2741191278/2741191280（tools/mt5verify -demoenv） | operation=OrderType_OrderType_* 枚举；volume；price*（pending 必）；sl/tp*；comment* |
| OrderModify | 改 SL/TP（仓）/改价（pending） | **ok**：两态均成功 | ticket+stoploss/takeprofit（仓）；price（pending 改价） |
| OrderClose | 平仓/撤挂 | **ok**：市价仓平+pending 撤 全周期闭环 | ticket（pending 撤同 RPC） |

**连接正径（demo）**：`Service.Search(company="Exness")` → Results[].Access 列表（**host:port 形态须拆分**——首版整串当 host 致 Connect disposed/timeout）→ Connect 凭据=/opt/nox-secrets/mt5-demo.env（MT5_DEMO_USER/SERVER/PASS）。实拍 endpoint=18.163.85.196:443。
**隔离钉（原禁，业主令 2026-10-24 授权开）**：真户 Trading RPC 原禁调——业主令「已有持仓本来就是测试单，可以做任何测试」后 **EXR 3001157 真户全谱实测**（在案）；ChangePassword **自伤型永禁**（改密断访——唯一永未实测项）；demo 数据禁写生产表。

## 真户 mutating 实证谱（2026-10-24，EXR 3001157，tools/mt5verify -realmut）
| 操作 | 实测 | 证 |
|---|---|---|
| OrderClose 在册真仓(2567600779) | **ok** | 真仓平仓全周期实证 |
| OrderModify SELL 仓 SL=1.0 | apierr `Invalid stops` | **方向校验在岗**——SELL 仓 SL 须高于现价，错向被正确拒（测试参数坑非 venue 病） |
| OrderSend 市价 BUY 0.01 BTCUSDm | ok ticket=2567601166 | 真户开仓实证 |
| OrderModify BUY 仓 SL/TP | ok | — |
| **OrderClose 开完即平** | **apierr「Request processing error」瞬态拒** | **真户新坑**：开仓后秒级内平=处理中竞态被拒；隔秒+带 `Lots` 字段重试即成（tools/mt5close 实证 att1）——**平必带 Lots，败退避重试，禁裸重试** |
| BuyLimit 深价挂→改价→撤 | ok→**同款瞬态 err**→ok | 挂单改价同竞态坑 |
| ETHUSDm 0.01 | apierr `Invalid volume` | 符号 minLots 异构——**量须 SymbolParams.SymGroup.MinLots 实读禁字面量** |
| OrderSend 市价 BUY（mt5ordstream 复跑） | 连拒 2 次「Request processing error」栈卡 `MarketOpenWaiterAsync`→第 3 次 ok ticket=2567601594 | **会话开+报价流正常仍瞬态拒**——MarketOpenWaiter 等网关侧成交确认超时；退避 ~10s 重试即过（发送侧同竞态族） |
| OrderClose 同一新仓（ordstream 内即平） | 「Request processing error」栈卡 `MarketCloseWaiterAsync`；**mt5close att1 带 Lots 即成** | 开/平两侧同族瞬态——**败后必须 OpenedOrders 仲裁残仓**，禁信报错=未成交（该次确未平，残仓 2567601594 后清） |
**符号方言硬坑**：demo 户同 Exness 系=BTCUSDm/XAUUSDm 型 m 尾（XAUUSD 裸名 not exist 实拍同 EXR 真户）——**symbol 名一律 venue SymbolList 实读禁猜**；周六仅加密面开（外汇对停盘勿试）。
**复跑**：nox `go run ./tools/mtprobe --dsn <dsn> --demo-env /opt/nox-secrets/mt5-demo.env`；PHASE `tools/mt5verify -demoenv <env>`（mutating 谱 6 拍：市价开→改 SL/TP→平→挂→改价→撤，2026-10-24 全绿）。
**请求字段速查**：OrderSend{symbol,operation(OrderType 枚举),volume,price*(pending),slippage,sl,tp,comment}；OrderModify{ticket,stoploss,takeprofit,price*}；OrderClose{ticket,lots*,price*,slippage*}。
## 错误码→动作分类表（errclass.ClassifyMT5——真钱重试语义层，重建必录）
| 码串 | Action | 语义 |
|---|---|---|
| REQUOTE / PRICE_CHANGED | RetryFresh | 重报新价再试（价漂非终败） |
| OFF_QUOTES / NO_PRICES / TOO_MANY_TRADE_REQUESTS | Retry | 瞬态退避重试 |
| MARKET_CLOSED / TRADE_DISABLED | Halt | 市场态停机——停试等会话 |
| NO_MONEY / INVALID_VOLUME / INVALID_PRICE / INVALID_STOPS / INVALID_TRADE_VOLUME / INVALID_TRADE_REQUEST | Abort | 入参/资金终败不试 |
| 未列码 | Abort 兜底 | CEX 侧无标准码表同 Abort |
**配套**：`IsFundingRejectMsg`——「margin insufficient/liq-distance」拒文归资金语义类（透出块/dashboard 单源件）。码串源 `mt5grpc.ErrorCode.String()`。
