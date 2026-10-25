# OKX mutating 面双端并发审计表（T-0104 @ 2026-10-25——同户双活竞态逐面定性）

| mutating 面 | 代码径 | venue 端点 | 双跑后果 | 幂等/竞态判 | 裁 |
|---|---|---|---|---|---|
| **残币归口** | `earn.sweepResidual`→transfer+SavingsPurchase | POST /asset/transfer、/finance/savings/purchase-redempt | **断腿级实爆**（2026-10-25 盲库搬 #23 活仓 ETH 腿→裸腿+对账假差+平仓径险炸） | 判据源=本节点库 FundCcy——分账部署下必盲 | **禁双跑**（已裁） |
| 闲资停泊 | `earn.sweepIdle`→transfer+purchase | 同上 | 同余量双读双摆——后者不足额拒重试（良性但双倍打 venue+在途窗互扰） | 幂等近真但双倍 REST | 禁双跑（只读端撤岗） |
| **执行双腿** | `execution.Runner.Execute`→PlaceOrder ×2 | POST /trade/order | **双开仓=双倍敞口**——armed 提案同键双端各开一份 | 非幂等（clOrdId 各自生） | 禁双跑 |
| 平仓 | `closeExec.ExecuteClose`→PlaceOrder | POST /trade/order | 双端评同仓各自平——第二平单空向或超卖 | 非幂等 | 禁双跑 |
| 开仓缝 | `lifecycle.openLive`→hedgeOpener | 同上（间径） | 同 ②——晋升双跑即双开 | 非幂等 | 禁双跑 |
| Earn 赎回 | `earn` 赎回岔→SavingsRedeem | POST purchase-redempt side=redempt | 同余量双赎——后者余额不足拒 | 幂等近真 | 禁双跑 |
| **策略晋升** | lifecycle probe→live | —（库内迁移） | 本机 fin_strategy 零行=现状安全；晋升径若在→armed 晋 live 即双开仓（连②） | — | 禁双跑（只读端 opener 不接线） |
| 划转摆渡 | `transfer()` trading↔funding | POST /asset/transfer | recon 30s 在途窗被对端划转反复触发 skip——良性但对账频次衰减 | 幂等（transId 追溯） | 禁双跑；debounce 扰=已知非病 |
| convert-dust | — | — | 本仓无此面（dust 未接） | — | N/A |

**结构性根修**：`O_NODE_ROLE=readonly` 只读镜像——`okxexec.NewClientReadOnly`
do() 签名前硬闸非 GET 全拒（ErrReadOnly），mutating 岗全不接线
（earnW/execution/liveOpener/closer）——写径从「双跑」降格「结构不跑」。
读径（balances/savings/dcd/order-history/bills/positions）全同。
