# T-0342 前端 Earn 自动申购停用开关

| 项 | 值 |
|---|---|
| 状态 | 已批（业主令 2026-11-04：「这个停止 Simple Earn功能，帮我做到前端页面上去控制，我有时候需要临时停用」） |
| 优先级 | P2 |
| 域 | 钱径控制面/前端 |
| 立票 | 2026-11-04 COS |

## 需求

业主需要一个前端开关，临时停用 O 对 OKX Simple Earn 的**自动申购/自动停泊**，不必再停整个 `o-core`。

## 钦定语义边界（不得扩大）

1. 关的是 **O 发起的自动买入 Earn**：`sweepIdle` 内交易户闲余回存、funding 户闲资直购、`sweepResidual` 残币归口申购。
2. **不关**：开仓前 `RedeemToTrading`；否则停开关会误伤开仓资金可用性，违本票目的。
3. **不关**：`pollRate` 现率采集、`refreshParkingSnapshot` 停泊余额读取、Cockpit/资金页 Earn 展示。
4. **不赎回、不移动现有 Earn 持仓**；停开关后钱留在原账户。

## 钦定实现形

- 配置键：`earn.auto_purchase_enabled`（JSON bool；生产缺键=fail-closed 停申购，播种默认 `true` 保持现行为）。
- Earn worker 在所有自动申购入口前热读该键：`sweepIdle` 起始、`depositIdleFunding` 防线内、残币归口实际申购前。读败/缺键按 fail-closed 处理并留计数。
- 新窄写 RPC：`SetEarnAutoPurchase(SetEarnAutoPurchaseRequest{enabled bool, reason string}) returns (...)`；只许写这一键，走 `config.Client.Set(..., actor="dashboard-ui")`，回包含当前真值/updated_at/updated_by。禁通用 config 写面。
- 新读 RPC 或并入现有响应：返回 `enabled/updated_at/updated_by/note`，前端显示「自动转入 Simple Earn：开/临时停用」。
- 前端放在驾驶舱或资金页贴近 Earn 展示处；按钮需二次确认，文案明确「只停止 O 自动转入，不赎回现有 Earn；开仓需要时仍可赎回」。
- 写面边界：生产 `o-core` 对外只绑 `127.0.0.1:50051`，首版按**同源页面窄写**落——`SetEarnAutoPurchase` 必须校验浏览器 `Origin` 的 scheme+host 与请求 Host 同源；`Origin` 缺席/跨源=拒。禁放行通用 config 写、禁接受任意 key。若实现时发现 Connect/浏览器实际不发 `Origin`，改成等价的同源校验，并在票面回填口径；不得退化成无校验裸写。

## 验收锚

1. `earn.auto_purchase_enabled=false` 时，fake/测试面三自动申购路径零 `SavingsPurchase`；`RedeemToTrading` 仍可调 `SavingsRedeem`。
2. RPC 写后 `app_config` 值、`updated_by` 留痕；前端回读显停用。
3. 生产部署后亲验：切关→下一次自动拍不再发申购；切回→恢复。不得动现有 Earn 余额。
4. `check_all.sh` 全绿；前端测试/Go 测试随件。

## 风险

- 前端无鉴权写面：宁可首版只显状态，也不得裸开放写接口。
- 禁用期间持仓 Earn 继续计息展示；停用文案不得声称已赎回。
