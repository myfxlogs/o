---
name: mt5-account
description: mtapi MT5 账户面 RPC——Account/AccountSummary/ServerTimezone/RequiredMargin/Mails 实测能力与坑
domain: 工程
---

# mt5 账户面（实测 2026-10-17，证据=docs/能力账/mt5-rpc-probe-2026-10-17.json）

| RPC | 功能 | ICR 15032313 | EXR 3001157 | XMR 110660304 | 载荷关键字段 |
|---|---|---|---|---|---|
| Account | 登录户全档（login/balance/credit/leverage） | ok | ok | ok | result.Balance/Credit/Leverage（Login uint64） |
| AccountSummary | 实时 equity/free_margin | ok | **2026-10-24 复测 ok——外加 `result.Currency`="USD" 实证**（账户币种现读位，overview USD≡USDT 核证用） | ok | result.Equity/FreeMargin/Currency——mt_frame/capitalview 源 |
| ServerTimezone | 服务端时区 offset（小时，含 DST） | ok | ok | ok | result（double）——D-337 swap 计费零点/brokertz 缓存源 |
| RequiredMargin | 挂单保证金试算 | ok | err: Symbol not exist | err: Symbol not exist | result（double）——**符号方言坑**：EXR/XMR 户 XAUUSD 裸名不存在（XAUUSDm 型尾缀），须 symMap 原生名 |
| Mails | 信箱列举 | Unimplemented | Unimplemented | Unimplemented | 网关未实现（三户一致定谳）；流侧 OnMail 同 Unimplemented |

**坑**：①token 走**请求体 Id 字段**（每 Request 必填）——metadata header 径网关不读（首版 harness 全 RPC "Value cannot be null" 实拍）；②Account 无参但 Id 必填；③ServerTimezone 返回 FixedZone 可用 offset 非 tz 名（DST 由 24h TTL 重取自愈，D-337）。
**复跑**：`go run ./tools/mtprobe --dsn <dsn> --out docs/能力账`（SG）。

## 流面实测补档（EXR 3001157 真户，2026-10-24，tools/mt5stream 探针）

| 流 RPC | 实测 | 载荷 |
|---|---|---|
| Events（通用事件） | **Unimplemented** | — |
| OnMarketWatch | **Unimplemented** | — |
| OnOpenedOrdersTickets | **Unimplemented** | — |
| OnOrderUpdate | 实现，15s 窗零帧（空仓静默） | OrderUpdateSummary——订单事件非账户摘要 |
| OnOrderProfit | **实现，账户摘要推送流**——持仓时 ~1Hz 推 `ProfitUpdate{Balance,Equity,Margin,FreeMargin,Profit,MarginLevel,Orders[]}`；空仓静默（push-on-change 语义） | 类型化 protobuf 全字段 |
| OnTickValue | 实现，零帧 | tick 值 |

**定谳（当日证伪修正）**：mtapi **有**账户摘要推送流——`OnOrderProfit` 推 `ProfitUpdate`（equity/margin/freeMargin/profit/marginLevel+订单明细全量）；**持仓期 ~1Hz 活动，空仓静默**（push-on-change——无推送即无变动，帧龄不判陈）。双轨：`OnOrderProfit` 流=持仓态唤醒+载荷源，`AccountSummary`=判时点现读/空仓态校验径——与 OKX 双轨同构（流推+REST 现读）。

**证伪修正**：原坑①「metadata header 径网关不读」失效——2026-10-24 实拍 `id` **header 必带**（Connect 用临时 sid，登录后所有调用 header `id`=<token> **且** 请求体 Id=<token>，双位同填）；缺 header=「Id header is required」全拒。
