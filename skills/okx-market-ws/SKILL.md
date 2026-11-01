---
name: okx-market-ws
description: okx 公开行情 WS 面能力账——books5/tickers/funding-rate/mark-price/index-tickers/instruments 全谱实测与订阅坑
domain: 工程
---
# okx-market-ws
## 证据
- 实现件：`internal/data/collect/exchange/okx_ws.go`、`okx_ws_books5.go`
- SG 产验（2026-10-23）：syms=7、books5 全谱订阅活、182-271 行/2-3min 落库
- **真户复验（2026-10-24，tools/okxverify 15s 窗）**：tickers 95 帧/books5 124 帧/mark-price 76 帧/funding-rate 2 帧——四频道全流实证
- 退役面实证：funding-rate(-history) REST 轮询清零后 WS 频道持续供数（`facts` 表 `src=okx_ws_v5` `kind=funding` 行流）
## 订阅面（公共频道全谱）
- `books5`：深度五档——原生 instId 键域（如 `BTC-USDT-SWAP`），节流+异步落库（读路径禁同步 DB 写）
- `tickers`：last/bid/ask/vol —— cexscan 价差读 `LatestTickMidWithin(60s)` 吃此面事实
- `funding-rate`：资金费率推送——REST `/public/funding-rate(-history)` 轮询已闸关退役（`NOX_OKX_REST_FUNDING` 豁免键备）
- `mark-price`/`index-tickers`/`instruments`：标记价/指数/合约目录推送
## 坑注记
- **无效符号单点拒不误伤**：`SILVER-USDT-SWAP` 等 venue 不存在合约被单符拒绝、其余符号照流——目录脏符不阻断面（但仍应清目录）。
- **ack≠存活**：订阅 ack 收讫不代表帧在流——存活判据=帧时戳前移（同 binance 族教训）。
- **REST depth 兜底闸关**：`NOX_DEPTH_REST_OKX=1` 才开，默认 fail-closed（缺数据≠降级数据——宁缺交易不静默降级）。
- **WS 即 h1 升级**：okx WS 是 HTTP/1.1 Upgrade（RFC6455 非 h2/h3）——WS 通≠h2 通证；REST 传输病灶须单测（见 `docs/经验库/条目/` h2 感染条目）。
- **端点**：`wss://ws.okx.com:443/ws/v5/public`；代理约定 `wsViaProxyURLs("/okxws/ws/v5/public", 直连URL)`——禁裸写 URL 不挂代理备径。

## funding-rate 帧字段面（T-0084 补记勘正 2026-11-01——预测口径证伪）

- `fundingRate`+`fundingTime`=已结算费率（落 md_funding.funding_rate/ts）。
- `nextFundingRate`=**死字段，勿采勿用**——OKX 官方文档参数表已划线「no longer supported」：2024-03 资金费率机制改制后 `method=current_period` 为唯一支持形，该字段恒发 `""`（官方推送示例实发 `"nextFundingRate":""`；SG 一周全谱实采 `next_funding_rate` 非零行=0 实证）。T-0084 预测前置闸因此作废。
- `nextFundingTime` 仍发值（落 next_funding_ts）——有下一拍时刻、无预测「值」；「帧逐帧推预测值」之前提不成立。
- 重飞注记：`store.go` funding INSERT `ON CONFLICT (venue,inst_id,ts)` 仅 `SET recv_ts`——venue 若回补字段，同结算拍内首见值后永不刷新；且空串→`num()`→0 落库使「缺席」与「真值 0」不可分辨（违 D-616 标缺律——重开此径须改空串→NULL 语义+冲突刷新谓词）。
