# T-0134 现货宇宙面 index-tickers 撤 WS 转 REST 轮询（阻塞级——链路饱和实证）

## 实证（SG 2026-10-25 ~18:30）

- `ens17` RX = **4767 kbps**（贴 5M 上限）——`close 1006` 断连根因=链路饱和服务端踢慢消费者，T-0132 砍完仍 ~110s/次
- 大头定位：`index-tickers` 订了 ~2300 现货标的（指数秒级推送）；SWAP 双面+活跃集 tickers 占比可忽略

## 修法

1. `subscribeArgs`：default 分支（现货形）**不再发 index-tickers**——宇宙 WS 只剩 SWAP 的 funding-rate+mark-price、季度期货 mark-price
2. 新增现货现价 REST 轮询件：`GET /api/v5/market/tickers?instType=SPOT` 单请求全量返回（分页注意：OKX 该端点单响应全品类），解析 `instId/last/ts` upsert 进 `md_mark_index`（kind='spot_last' 或并入现有键域——施工按表结构定），周期 60s（可配置 `okxws.spot_poll_s`）
3. **现货腿判仓不断供**：活跃集 tickers 不动——在仓现货标的仍有成交价级流；宇宙现货价精度=60s 级，满足评估/发现面
4. REST 轮询走现有 restOKAt 活性戳同通道，失败计 rejects

## 验收锚

- `ens17` RX < 1 Mbps（实测前后对照——现基线 4767 kbps）
- `md_mark_index` 现货行 ts<90s 鲜（轮询+入库时延内）
- 连续 30min 零 `close 1006`（现基线 ~110s/次）
- 棘轮/机检全绿

## 结果回填

- subscribeArgs 现货形 index-tickers 撤 WS（2300 标的秒推=链路饱和实证
  源切除）；宇宙 WS 剩 SWAP funding-rate/mark-price+季度期货 mark-price。
- spotpoll.go REST 轮询件：GET market/tickers?instType=SPOT 单请求全量→
  md_mark_index channel='spot_last'（键域分位不混 WS 面）；60s 周期热启
  okxws.spot_poll_s；读败计 rejects 不造零；restOKAt 活性戳在位。
- 现货腿判仓不断供：活跃集 tickers 面零动（成交价级流保留）。
