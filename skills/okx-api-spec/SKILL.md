# okx-api-spec —— OKX v5 参数字典（docs-v5/zh 全量核证 2026-10-24）

> 用途：设计/施工碰 OKX 参数前**先查本表不猜**。源=官方 docs-v5 中文站全量页（71k 行抓档核证）。
> 本账与 `okx-exec`（下单径）、`okx-market-ws`/`okx-private-ws`（流面）、`okx-rest-boundary`（REST 界桩）分工：本件=参数字典层。

## 1. 数量/金额计量总表（币本位 vs U本位——业主令级）

| 场景 | sz 计量 | 面值公式 |
|---|---|---|
| 币币现货 限价单 | 交易货币（base_ccy）数量 | sz×px=计价币额 |
| 币币现货 **市价买单** | **默认计价币 USDT 金额** | `tgtCcy=base_ccy` 改按币量 |
| 币币现货 市价卖单 | 默认 base_ccy 币量 | tgtCcy=quote_ccy 改按 USDT 额 |
| 永续/交割/期权 | **张数** | 正向=`sz×ctVal×markPx`；反向=`sz×ctVal`（USD 固定） |
| 币币杠杆 | 同现货规则 | 同上 |

- **区分币本位/U本位**：看 instruments 的 `ctValCcy`+`ctType`（linear=正向 USDT 本位/inverse=反向币本位），禁按币种名猜。BTC-USDT-SWAP 实测=ctVal 0.01 BTC 本位面值。
- `tgtCcy` 仅币币市价单适用：buy 默认 quote_ccy、sell 默认 base_ccy。
- 现货市价单 `slippageBps`（0~0.05，默认 0）=拒价保护面。

## 2. 下单参数字典（POST /trade/order 与 WS 下单同参）

| 参数 | 规则 |
|---|---|
| instId | 原生键 `BTC-USDT-SWAP`/`BTC-USDT` |
| tdMode | `cash` 现货 / `cross` 全仓 / `isolated` 逐仓（开平仓模式永续逐仓=品种+持仓方向粒度） |
| side/posSide | net 户免 posSide；long_short 户 SWAP 必填：开 sell→short/buy→long，平 buy→short/sell→long |
| ordType | market/limit/post_only/fok/ioc/optimal_limit_ioc（仅合约） |
| clOrdId | 英数 1-32 位，**禁连字符**（51000 实证） |
| reduceOnly | 单向户减仓声明；**long_short 户平仓单天然只减仓**（传不传都生效）；sz 超持仓=整单拒非截断 |
| attachAlgoOrds | 附带止盈止损（独立于母单须走 order-algo） |
| stpMode | cancel_maker/cancel_taker/cancel_both（后者不支持 FOK） |

**预检端点**：`POST /api/v5/trade/order-precheck`——同参不建单验证，真单前可零成本试参（T-0042 后接入议）。

## 3. instruments 字段（GET /public/instruments?instType=&instId=）

`ctVal` 面值币量 / `ctValCcy` 面值币种 / `ctType` linear|inverse / `lotSz` 数量步长 / `minSz` 最小量 / `tickSz` 价步长 / `lever` 最大杠杆 / `state` live|suspend|preopen。张数必落 lotSz 网格+≥minSz。

## 4. 账户面关键字段

- `GET /account/config`：posMode（net_mode|long_short_mode）、acctLv、level、kycLv、perm、ip、uid——**首单实读禁猜**。
- `GET /account/balance`：`adjEq` 调整权益 / `availEq` 可用 / `details[].{ccy,eq,availBal,cashBal,eqUsd}`；`notionalUsd`=衍生品 USD 名义总（线性=数量×ctVal×markPx，反向=数量×ctVal）。
- `GET /account/positions`：`availPos` 可平量 / `avgPx` 均价 / `upl`：线性=(markPx−avgPx)×持仓×ctVal；反向=(1/avgPx−1/markPx)×持仓×ctVal。posId 满 30 天平尽清除。
- `GET /account/fee-rates`：手续费率（maker/taker 逐品种）。
- `GET /trade/order?instId&clOrdId` 定态：51603=确定单未建（absent）非查询败。

## 5. 私有 WS 面（事件驱动正解——禁 REST 轮询）

- 登录后订阅：`orders`（订单状态推）、`positions`（持仓推）、`account`（余额推）、`balance_and_position`（合并推）——详见 okx-private-ws 卡。
- 公开流：`instruments` 频道推产品配置变更（lotSz/minSz 变更事件源——facts 乐器层可改事件驱动）。
- WS 限：480 次/时·连接公共订阅；私有频道同连接多订阅合并计数。
- **订单频道推送=成交真相源**——REST 查单只用于 clOrdId 歧义定态（okx-exec 卡三分支）。

## 6. 限速与运维面

- 限速按「端点×UID/子账户+IP」双层；下单类 60 次/2s 量级（端点逐则，查 GET /account/rate-limit 实表）。
- 批量下单/撤单上限 20 张/请求。
- 时间戳=UTC ISO8601 毫秒；偏差=50102 最常见因（装表核 NTP）。
- 模拟盘=同端点+`x-simulated-trading:1` 头。

## 7. 错误码族（详见 okx-exec 卡——本表只列结构）
- envelope `code`+单条 `sCode/sMsg`；业务拒在 data[].sCode≠0；envelope≠0=APIError 类型化按码定态（51603→absent）。
- 5xxxx 参数/交易族；501xx 鉴权族。
