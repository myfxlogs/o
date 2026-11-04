# T-0342 前端 Earn 自动申购停用开关

| 项 | 值 |
|---|---|
| 状态 | **完成**（决策席验收通过 devin @ 2026-11-04——合 master@4955fad1，判词在案） |
| 优先级 | P2 |
| 域 | 钱径控制面/前端 |
| 立票 | 2026-11-04 COS |
| 领取 | devin-o-worker-t0342 @ 2026-11-04（无头施工 run——前拍 rate-limit 断线，工树存未提交进度续作收口） |

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

## 结果（施工回填）

- 分支：`t/0342-earn-autobuy-switch`；涉及面=`internal/earn/`（worker+residual+测试）+`cmd/core/`（main/service/earn_face+测试）+`proto/dashboardv1/core.proto`+`internal/store/pgstore/migrate_table.go`+`migrations/fin_0110_earn_auto_purchase.sql`+`web/src/react/`（api/Overview/EarnToggle/styles）。
- 实现：
  - 键 `earn.auto_purchase_enabled`（JSON bool）：fin_0110 播种 `true` 保持现行为；缺键/读败=fail-closed 停申购（worker `autoPurchaseOn` 热读+`auto_purchase_off` 计数门）。
  - 三闸钦定面：`sweepIdle` 起始全段让位、`depositIdleFunding` 防线内再闸（绕过上游直调亦不发）、`sweepResidual` 残币归口实际申购前让位；`RedeemToTrading` 开仓赎回不经闸。
  - RPC 面：`GetEarnAutoPurchase`（enabled/updated_at/updated_by/note——app_config 库面直查审计列+ops_event 末次翻位理由；缺键=enabled=false 停用显形不假开）+`SetEarnAutoPurchase` 窄写（只许写此一键，`config.Client.Set` actor=`dashboard-ui`；reason 落 `ops_event` seat=`dashboard-ui` ticket=空——工单板零污染；回包=库面重读真值非回显）。
  - 前端：`EarnToggle` 挂 Overview「钱在哪」卡贴 Earn 展示处；按钮 `window.confirm` 二次确认，停用文案钉「只停止 O 自动转入，不赎回现有 Earn；开仓需要时仍可自动赎回」——不声称已赎回；写径裸调不入 dedup 合并窗（写非幂等），写后回读真态。
  - Origin 校验口径落字：connect-go v2 服务端 `CallInfo.RequestHeader` 由 connecthttp handler 注入 `Host`（`handler.go` `setHeaderCanonical(request.Header,"Host",request.Host)` 源码实证）——闸齿=Origin scheme+host 对请求 Host 等值、scheme 钉 http/https、TLS 直连面必 https；Origin 缺席/跨源/不可解析=`CodePermissionDenied`，无「无校验裸写」退化。
- 测试证据：
  - `internal/earn` `TestAutoPurchaseGate`：键=false 下 sweepIdle（funding 500+交易户闲余 800 双源）/depositIdleFunding 直调/sweepResidual（FIL 残币归口）三径零 `SavingsPurchase` 零摆渡；同态下 `RedeemToTrading(100)` 照发 `SavingsRedeem`+回拢摆渡；删键热载 fail-closed 同停用——锚①。
  - `cmd/core` `TestEarnAutoPurchaseOriginGate`（connecthttp 真挂载：缺席/跨源/坏 scheme/坏 Origin 全 403+键仍播种 true+审计行零）、`TestEarnAutoPurchaseWriteAndReadback`（同源写 200→app_config `updated_by=dashboard-ui`+ops_event reason 行+cfg 热读 false+Get 回读停用显形+同源恢复往返）、`TestEarnAutoPurchaseAbsentKey`（缺键读面 enabled=false）——锚②。
  - `check_all.sh` 全链绿（build/vet/import/gofmt/env/rejects/migrations/bridge-audit/go test 全谱）——锚④；前端 `npm run typecheck`+`vite build` 绿。
  - 前端测试口径注记：web/ 无测试框架基建（package.json 无 test 脚本、全仓前端测试件零在案）——本票不新立框架（超票面+新依赖），验证口径=tsc+build+RPC 面 Go 桩测。
- 部署后验项（锚③留生产亲验）：切关→下一拍零申购实证；切回→恢复；现有 Earn 余额零动。
