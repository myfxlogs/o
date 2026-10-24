---
name: mt5-orders-streams
description: mtapi MT5 订单/流面——OpenedOrders/OrderHistory 族/Subscribe 族/Streams On* 实测能力+Deal 载荷教训锚（入金走 DealType_Balance=2）
domain: 工程
---

# mt5 订单流面（实测 2026-10-17，证据=docs/能力账/mt5-rpc-probe-2026-10-17.json）

| RPC | 功能 | ICR | EXR | XMR | 关键坑 |
|---|---|---|---|---|---|
| OpenedOrders(Tickets) | 在仓单实读 | ok | ok | ok | Order.Ticket/Profit/Swap——bpMon 持仓卡源 |
| OpenedOrder | 单仓按 ticket 查 | err | err | err | 「OK null response」三户一致——空仓户/未知票=空体非故障，先 OpenedOrdersTickets 取票再查 |
| MarketWatchMany | 行情观察列表批量 | Unimplemented | Unimplemented | Unimplemented | 网关未实现（MarketWatch 面走 SubscribeMarketWatch 同 Unimplemented——观察单功能缺面在案） |
| SubscribeOrderProfit/SubscribeTickValue/SubscribeOrderUpdate | 分型订阅 | ok | ok | ok | 对应 On* 流帧开关——先订后流 |
| SubscribeMarketWatch/SubscribeOpenedOrdersTickets | 观察单/仓票订阅 | Unimplemented | Unimplemented | Unimplemented | 网关未实现（仓票面走 OpenedOrdersTickets 拉取替代） |
| OrderHistory | 已平仓史（充提 Balance 行+平仓盈亏） | ok | **2026-10-24 复测 ok——21 行含 10 笔 Balance 充提行** | ok | **OrderType_Balance=充提行**——mtIORound 只收此型；CloseTimestampUTC 毫秒；XMR 实证入金可滞后 ~48h（退回㉖） |
| OrderHistoryPagination | 分页史 | Unimplemented | Unimplemented | Unimplemented | 网关未实现（三户一致；XMR 取证同理） |
| PendingOrderHistory | 挂单史 | Unimplemented | Unimplemented | Unimplemented | 同上 |
| Subscribe(Many)/UnSubscribe(Many) | 订阅动作（symbol 串） | ok | ok | ok | interval 可选毫秒节流 |
| Streams.Events | 聚合事件流（一切 On* 的合流根） | ok（45s 窗） | ok→**2026-10-24 复测 Unimplemented** | ok | 流帧=OnOrderUpdate/OnQuote/…分型；空仓户零帧常态 |
| OnQuote/OnTickValue | 独立 server-stream | 短窗采样在案 | 同 | 同 | 事件分型流 |
| **OnOrderUpdate** | 独立 server-stream | 短窗采样在案 | **2026-10-24 真户订单事件实证发帧** | 同 | **Deal 载荷教训锚（D-607）**：Reply.Deal 携 DealType 枚举——DealType_Balance=2=入金载荷；充提事件监控正道=订阅此流非轮询 OrderHistory。**2026-10-24 EXR 实单观测**（tools/mt5ordstream：流开→SubscribeOrderUpdate→真户 OrderSend）：市价开单即发 1 帧 `OrderUpdateSummary{Update:{Trans/Deal/OppositeDeal/Order, Type=UpdateType_MarketOpen}, Balance,Equity,Margin,FreeMargin,Profit,MarginLevel, OpenedOrders[]}`——**事件型推送且帧内仍携全账户摘要+在册仓**，与 OnOrderProfit 同摘要字段（事件触发 vs ~1Hz 周期） |
| **OnOrderProfit** | 独立 server-stream | — | **2026-10-24 真仓实测：账户摘要推送流** | — | 持仓态 ~1Hz 推 `ProfitUpdate{Balance,Equity,Margin,FreeMargin,Profit,MarginLevel,Orders[]}`；**空仓静默**（push-on-change）；**不订 SubscribeOrderProfit 也推**（standalone 流自激活，订阅件可能只控 Events 聚合路由）——EXR 真户 60s/60帧+20s/21帧两次实证（tools/mt5stream）；**PHASE 生产接线在案（T-0048）**：`internal/mt5` Collector 会话内订阅+消费此流→`md_account_events` channel='order_profit'（值变才落，ev_account NOTIFY 自动），空仓期无行=无变动语义非缺岗 |
| OnMail/OnMarketWatch/OnOpenedOrdersTickets | 独立 server-stream | — | 2026-10-24 复测 **Unimplemented**（Connect/ConnectEx 双径同） | — | 网关侧缺面稳定复现 |

**坑**：①无 token 时静默零帧（老坑——现网实证已升级为 `id` header 缺失=「Id header is required」硬拒，见 mt5-gateway-admin 修正）；②**能力探针必须带仓态复跑**——空仓窗探不出持仓流（OnOrderProfit 空仓静默曾致「无账户流」误判，EXR 持仓复跑翻案）；③断线自愈=wsrunner 重连+重订阅（adapter 范式）。
**magic number=ExpertId**：写侧 `OrderSendRequest.expertID`（field 10 optional）；读侧 `Order.ExpertId`（21）/`DealInternal.ExpertId`（23）/`OrderInternal.ExpertId`（26）——查单=OpenedOrders/OrderHistory 按 Ticket 过滤取该字段。
**符号方言硬坑**：每 broker 后缀不同（EXR/XMR 户 XAUUSD not exist 实拍、Exness=m 尾 BTCUSDm）——symbol 名一律 venue `SymbolList` 实读禁猜；GETQuote 族传裸名=「Symbol not exist」假故障。
**复跑**：同 mt5-account（--window 控窗）。

<!-- 方法全名索引：OpenedOrders OpenedOrder OpenedOrdersTickets OrderHistory PendingOrderHistory OrderHistoryPagination Subscribe SubscribeMany UnSubscribe UnSubscribeMany SubscribeOrderProfit SubscribeTickValue SubscribeOrderUpdate SubscribeMarketWatch SubscribeOpenedOrdersTickets Events OnOrderUpdate OnQuote OnTickValue OnOrderProfit OnMarketWatch OnMail OnOpenedOrdersTickets MarketWatchMany -->
## 平仓事件仲裁坑（ant 实证移植——nox 事件轨同病预防件）
- **部分平也发 close 事件**：close-kind push 命中≠全平——事件路径必进 RAW 仲裁：票在+量<源=部分平→仅刷快照；票消失=全平→迁移+删快照（ant PARTIAL-CLOSE-EVENT-ARBITRATION 实证）。
- **close push 不携余量**：push 报 remaining=0 不可靠——命中后仍须 OpenedOrders RAW 取余量；RAW 取证败=fail-closed UNKNOWN 禁猜 0（ant CLOSE-PUSH-REMAINING-ZERO 实证）。
- **票号 int32 溢出**：MT5 票域超 int32（Exness 实拍 2.66e9）——下游凡 int32 票槽皆截断回绕为负；VM/表侧票号须 uint64/合成句柄桥（ant MT5-TICKET-INT32-OVERFLOW 实证）。
- **双流归一 fail 语义**：fail 须 `close(done chan)`，send 端 `select{ch<-, <-done}`，Recv 端 `select{ch,done}`——只存 err 不关信号=Recv 永久阻塞；close(ch)=他 pump send panic（ant STREAM-MERGE-FAIL-SHAPE 实证）。

**订阅回执携仓证据（2026-10-24）**：SubscribeOrderProfit/SubscribeOrderUpdate 的 Result 立即回吐当前持仓 Order 全档（含 DealInternalIn 开仓明细）——订阅=拿到仓态首拍，免再拉 OpenedOrders。
