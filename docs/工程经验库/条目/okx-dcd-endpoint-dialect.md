---
schema_version: 2
id: okx-dcd-endpoint-dialect
kind: pitfall
risk: medium
status: verified
keywords: [OKX, sfp/dcd, 双币赢, 信封 code 数字型, order-history, 50110 IP 白名单, IPv6 egress, bill subType]
scope: [internal/trading/okxexec, tools 探针族, venue 集成面]
summary: OKX sfp/dcd 端点族三坑——响应 code 为数字型（okxexec 信封 string 解析拒）、端点名猜错回 500 非 404（真名 order-history/order-status）、IP 白名单按出口栈族判（IPv6 egress 漂移即 50110）；dcd 资金流水走 bills subType 160-163/207-210
first_seen: 2026-10-25
last_verified: 2026-10-25
skills: [okx-asset-funding]
evidence_refs:
  - type: server_fact
    ref: "决策席裸签名探针 2026-10-25：currency-pair/products/order-history 三返 200；orders/order/orders-list 等非名全 500；IPv6 出口三端点连 401(50110)"
  - type: doc
    ref: https://www.okx.com/docs-v5/zh/#financial-product-dual-investment
---

# OKX sfp/dcd 端点方言坑（T-0101 探针实证）

## 病灶四件

1. **信封 code 数字型**：dcd 族返 `{"code":0,...}`——`code` 是 number
   非 string。`okxexec.Client.Get` 的信封解析（`Code string`）unmarshal
   即拒：「cannot unmarshal number into Go struct field .code of type
   string」——**200 真数据被信封方言误杀**。接 dcd 面须信封兼容
   （json.RawMessage 或 number|string 联合型）。
2. **端点名猜错=500 非 404**：`orders`/`order`/`orders-list`/`order-list`
   全 Spring 式 500「Internal Server Error」——像服务崩其实是**名不对**。
   真名：`GET /order-history`（枚举全谱）、`GET /order-status?ordId=`（单查）。
   教训：端点名以文档为准勿猜——500≠路由不存在。
3. **IP 白名单按出口栈族判**：OKX key 白名单记 IPv4；本机双栈时 happy-eyeballs
   漂移 IPv6 egress→50110「IP 不在白名单」间歇 401——同机同钥同时段
   一半请求过一半拒，表象像钥匙坏。诊断锚：报错体里的 IP 是 v6 形即中此坑；
   修法=出站钉 tcp4 或把 v6 也录白名单。
4. **dcd 流水面=bills subType**：双币赢申购/回款/收益/退款=
   subType 160/161/162/163（精简版 207-210）——查 dcd 资金流不查
   finance 表，走 `/account/bills`/`/asset/bills` 按 subType 过滤。

## 经验

- venue 新族接入三查：信封方言（code 型别）、端点真名（文档钉死）、
  资金流水 subType 表（账单枚举页）。
- 白名单 401 先看报错 IP 的栈族——间歇性 401=egress 漂移特征非凭据病。


## 追补（2026-10-25 T-0102 部署日实证）
- `products` 参数名=**baseCcy+quoteCcy+optType 三必参**（官方文档钦定）——
  施工曾猜 `?uly=` 全 50014；`uly` 只存在于 `order-history` 过滤参。
- 响应字段真名：`optType`（非 direction）、`expTime`（非 expiryTime）、
  `productId/strike/annualizedYield/minSize/maxSize/notionalCcy`——伪字段名
  静默解析零行，不报错难察觉。
- 响应形歧：`currency-pair` data=裸数组；`products` data=`{"products":[...]}`
  对象包络——同族端点 envelope 不统一，按裸数组解析=静默零行。

## 追补（2026-10-25 T-0117 返修③实证）
- `products` 的 `uly` 字段=**标的指数名非可采 inst_id**：`BTC-USD`、
  `BTC-USDG`（USDG=USDG 稳定币计价族，标的仍 BTC）——md_ticks 采的是
  `BTC-USDT`/`BTC-USDT-SWAP`，按 uly 直查价=零命中全 spot_missing。
  取价须归一链 `uly→uly+T→uly+T-SWAP→uly-SWAP` 外再补 `USDG→USDT` 变体。
- **`quoteCcy` 计价币族=USDT/USDG/USDC/RLUSD**（官方 2025-12 上线 USDG/USDC
  计价双币赢）：P 产品锁的是 quoteCcy 非 USDT——锁仓余量判据硬写 USDT=
  错币过闸，产品行必须解析 quoteCcy 各归各币。
