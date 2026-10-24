---
name: okx-account
description: OKX 账户/交易读面能力账——balance/config/positions/max-size/leverage/orders 史/fills/bills 真户实测字段级口径（availBal 在 details[] 层不在行顶）
domain: 工程
---

# okx-account —— OKX 账户/交易读面（真户实测 2026-10-24，EXR 关联户 uid=44303245937614848，tools/okxverify）

> 分工：读面能力账。下单写径→okx-exec；参数全集→okx-api-spec；asset/finance 钱径→okx-asset-funding。

## 端点实测表（真户全通——GET /api/v5/）

| 端点 | 实测 | 关键字段/口径 |
|---|---|---|
| /account/balance | ok | `totalEq`=交易户总权益 USD；**币种明细在 `data[0].details[]` 层**——{ccy,eq,availBal,availEq,cashBal,eqUsd}；行顶层无 availBal（读行顶字段=空串假零坑，实测踩过） |
| /account/config | ok | `posMode`=`long_short_mode`（**本户双向持仓**——SWAP 单必带 posSide=long/short）；`acctLv` 账户等级 |
| /account/positions | ok | 空仓=空集常态；持仓行 `pos` 数量/`avgPx`/`upl` |
| /account/max-size?instId&tdMode | ok | maxBuy/maxSell 张数（0=无可开额度常态） |
| /account/max-avail-size?instId&tdMode | ok | availBuy——闲余额度实读位 |
| /account/leverage-info?instId&mgnMode&posSide | ok | `lever`——本户 BTC-USDT-SWAP cross=3x；**long_short_mode 下 data[] 双侧行，data[0] 恒=long 行**——读 short 侧必带 posSide=short 查参或读 data[1] |
| /trade/orders-pending?instId | ok | 挂单面；空集=无挂常态 |
| /trade/orders-history?instType | ok | SPOT/SWAP 分页史——ordId 全档 |
| /trade/fills-history?instType | ok | 成交流水 tradeId/fillPx/fillSz/fee |
| /account/bills?instType&limit&after | ok | **记账真源**——billId 幂等键；subType 173=funding 支出/174=funding 收入；fee 负=扣正=返 |
| /account/fee-rates | 在册未跑 | maker/taker 逐品种费率（api-spec 有口径） |

## 写面实测（tools/okxlev，真户 2026-10-26）

| 端点 | 实测 | 口径 |
|---|---|---|
| POST /account/set-leverage | ok | cross 往返 3→5→3 实证；响应体 `data[0].lever` 回显=写确认位 |
| cross 参集 | ok | `instId+mgnMode+lever` 三参即成——**无需 posSide**（cross 档位管双侧） |
| isolated 参集 | ok | **posSide 必填**（long_short_mode）——缺=`51000 Parameter posSide error`(http400)；long/short 双侧独立各设 |
| 边界 | 拒码在案 | 超档（200x>档顶）=`59102`（envelope 码 http200 壳！）；`lever=0`=`51000 Parameter lever error`(http400) |

## 坑

- **availBal 层级坑**：/account/balance 的币种可用额在 `data[0].details[].availBal`——取 `data[0].availBal` 顶层=空串（现货卖回读错层曾致跳测）。取件统一走 details[] 遍历。
- **∅≠病**：positions/orders-pending 空集是常态（空仓/无挂）——探针判 PASS 语义=errfree 非「有行」。
- **long_short 户语义**：`posMode=long_short_mode` 时平仓单天然只减仓（sz 超持仓=整单拒非截断）；perp 开 buy→long / sell→short，平反向 posSide+reduceOnly。
- **leverage-info 双侧行坑**：long_short_mode 下 `data[]` 返回 long/short 两行、`data[0]` 恒为 long——单读 data[0] 拿 short 侧档位=拿错行。查参 posSide=short 或遍历 data[]。
- **set-leverage 拒码双层**：超档位是 `59102` envelope 码（http 200 壳内）非 http 4xx——只检 http 状态会漏拒。
- **51008 余额不足**：内层 sCode 才见真码——envelope code=1「All operations failed」是壳（okx-exec 卡）。
- **50004/传输歧义**：下单超时/传输败→`/trade/order?instId&clOrdId` 定态三分支（filled/live-or-partial/absent-可同键重试）；查询亦败=unknown 禁补单。

## 复跑

`tools/okxverify -dsn <dsn> -key <masterkey>`（读面+双流）；`-mut` 加 mutating 实单谱。杠杆写面：`tools/okxlev -dsn <dsn> -key <masterkey> [-inst BTC-USDT-SWAP]`（往返实测，自动复原 3x）。
