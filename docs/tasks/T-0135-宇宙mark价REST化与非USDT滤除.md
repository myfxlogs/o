# T-0135 宇宙 mark-price 撤 WS 转 REST 轮询+非USDT滤除（阻塞级——链路仍饱和实证）

## 实证（SG netmeter 分频道账，T-0134 部署后 7min）

- `okxws/mark-price` = **93.7MB ≈ 1.8Mbps**（507 SWAP 秒级推送）——T-0134 砍掉 index-tickers 后的新任链路大头
- `okxws/tickers`（活跃集~16 标的）= 59.9MB ≈ 1.1Mbps；funding-rate 1.6MB 可忽略
- `ens17` RX 仍 4763 kbps 贴顶——宇宙 mark-price 不撤，1006 断连必然复发

## 修法（T-0134 同法）

1. `subscribeArgs`：SWAP 形只留 `funding-rate`；`mark-price` 退出全宇宙（含季度期货——到期期货标量少同撤）
2. REST 轮询扩件：`GET /api/v5/public/mark-price?instType=SWAP` + `?instType=FUTURES` 两请求全量→`md_mark_index` 写 **`channel='mark_price'`**（复用 WS 原键域——读面零改无缝衔接；与 spotpoll 同周期同热启键）
3. `writeSpotLast` 补 `isUSDTInst` 滤——T-0134 留的非 USDT 死行滤除（含 mark-price 轮询同滤）
4. 活跃集 tickers 不动——钱径实时价流保留

## 验收锚

- `ens17` RX **< 1.5 Mbps**（活跃集 tickers+REST 轮询量级）
- `md_mark_index` mark_price 行 ts<90s 鲜、spot_last 同鲜
- 连续 30min 零 `close 1006`
- netmeter `okxws/mark-price` 字节账归零（只剩 REST 轮询行）
- 棘轮/机检全绿
