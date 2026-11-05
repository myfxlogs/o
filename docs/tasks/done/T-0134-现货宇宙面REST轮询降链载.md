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

- 状态：完成（判过并码在案，11-01清账翻面）

## 审计判决（审计席：headless @ 2026-10-30）

逐锚实态对账（收口 db6ca262 全量 diff + 宣称件三证 + SG/OCI 现网只读
实拍三面交叉）：

- **锚① `ens17` RX < 1 Mbps** ✓：SG `/proc/net/dev` 60s 均值实测
  **487 kbps**（10s 短窗样 1023 kbps=SSH 会话自载瞬峰，60s 均值为
  判据）——基线 4767 kbps 贴 5M 上限 → 约 1/10，链路饱和证除。
- **锚② `md_mark_index` 现货行 ts<90s 鲜** ✓：PG-md 正本（OCI 盒）
  index-tickers 通道现货行 **521/521 ts≈2s 鲜**（15:33Z 实拍
  max_recv_ts 同步）——现货价供径今=WS index-tickers 行（OCI 全
  宇宙订，subscribeArgsFull 在码）；spot_last 通道已随 T-0171 快照
  价格全面退役钦定清扫（md_1071：行零存，曾实存=写入径实证）。
  SG 镜像表 serve=活跃集谓词收窄，非活跃键残行 ~12.5h 陈系投影臂
  谓词设计（T-0168/0169）非断供——BTC-USDT 等活跃键 recv_ts 秒级
  鲜，镜像活实证。
- **锚③ 连续 30min 零 `close 1006`** ✓：SG o-core `docker logs` 48h
  窗仅 4 次 1006（末次 12:10:46Z，审计时点 >3h 零）；OCI
  o-collecto（今担全宇宙含 index-tickers ~521 现货标的）journalctl
  48h 同仅 4 次（末次 11:27:53Z，>4h 零）——基线 ~110s/次 vs 现
  ~2/日量级；残余 1006=venue 常态远端断连形，非饱和踢慢消费者谱。
- **锚④ 棘轮/机检全绿** ✓（可考证据界内）：收口期 GitHub Actions
  尚无 run（全仓首跑 2026-10-28）——当时机检=T-0002 已武装
  pre-commit 门禁族+本地测试面；diff 内 parse_test.go 断言同 commit
  改订（funding-rate+mark-price 双面）、史内零 T-0134 修复/回滚
  commit；今 master CI 最新 run success（e70bfcd9）。

宣称件三证（回填 vs diff 逐条）：

- subscribeArgs 现货形 index-tickers 撤 WS ✓——default 分支零订在
  diff（subscribe.go），今 master 瘦身面同形持效；
- spotpoll REST 轮询件 ✓——PublicGet `/api/v5/market/tickers?
  instType=SPOT` 全量解析 instId/last upsert channel='spot_last'、
  60s ticker 热读 `okxws.spot_poll_s`、读败 `rej.Inc` 不造零、
  `restOKAt.Store` 活性戳、AttachSpotFeed/spotFeedFor 装配闭环
  （main.go/universe.go）全在 diff；md_mark_index
  PK(venue,inst_id,channel)+venue DEFAULT 'okx'（fin_0049）与
  ON CONFLICT 对径实证；
- 现货腿判仓不断供 ✓——diff 零触 tickers 订阅径（活跃集动态重订
  面原样）。

判决注记（不阻断，留档报裁）：

1. 修法②举「解析 instId/last/ts」——实装弃响应 ts 用本地 now()
   （recv_ts 同戳），鲜度语义等价、非违锚，注记级；
2. spotpoll 面后随 T-0135 扩 mark-price、T-0171 全撤转 WS 正源——
   属钦定演进非本票病灶；
3. 收口 diff 未滤非 USDT 现货行——验收合入注记在案转 T-0135 顺手
   清，锚面无涉。

判决：**审计通过**。
