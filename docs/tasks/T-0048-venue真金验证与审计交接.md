# T-0048 附属：MT5+OKX 真金全谱验证报告与复审交接（2026-10-24）

> 用途：给**独立复审席**的冷启动包。本文件自包含——读完此篇+按指针抽查即可独立裁决，无需原 session 上下文。
> 性质：T-0048 同票续作的验证工作报告；skills/=结论 canon，本件=过程与证据锚。
> 复审纪律（契约定）：施工不能验收自己的产出——本报告作者=施工席，结论待复审席独立复跑确认。

## 0. 一句话总览

MT5 五件 skill + OKX 七件 skill 的全部能力面声明，用**真实账户真金白银**逐项实测对拍；产出 6 件探针工具、2 件新建 skill、5 件经验库条目；过程暴露并修复 1 起真钱事故+4 枚探针缺陷簇。venue 事实面已实证到顶，**待复审的是探针台账算术与同族残留扫尾**（§6）。

## 1. 授权与账户背景

- 业主令：「全套跑一遍，读取，开仓，平仓，读各种数据……这是我拿真金来做测试每一项」；「已有持仓本来就是测试单，可以做任何测试」；「MT5 不需要 demo，全部真实账户」。
- MT5 真户：Exness login 3001157，币别 USD，末态 equity≈3247.49、**0 持仓**（在册测试仓 ticket=2567600779 已按授权平仓）。
- OKX 真户：posMode=long_short_mode（双向持仓户）、cross lever=3；末态 positions ∅、挂单 ∅、trading/funding 各 ≈$0.2、Earn USDT≈2132.91（含事故往返净耗 ≈$7-15）、PENGU 残尘 0.1856（<minSz 不可交易）。
- 凭据径：`app_secrets` 密文列+`/opt/phase/secrets/phase_master_key`（mode 600）解密；工具零回显。

## 2. 探针工具（tools/，全部 `-h` 可查）

| 件 | 用途 |
|---|---|
| `tools/mt5verify` | MT5 五 skill 44 项声明+9 流对拍（`-login/-pass/-server` 真户；`-demoenv` demo；`-realmut` 真户 mutating 谱） |
| `tools/mt5stream` | MT5 流面探针（60s 窗全流监听+OpenedOrders/AccountSummary 基线对拍） |
| `tools/mt5ordstream` | OnOrderUpdate 事件流探针（流开+订阅+真单贯穿） |
| `tools/mt5close` | 残仓清理件（OpenedOrders 实读→逐票带 Lots 重试平） |
| `tools/okxverify` | OKX 全谱（21 行 REST 读矩阵+4 公开 WS+4 私有 WS+`-mut` mutating 谱；台账制复位） |
| `tools/okxunwind` | 应急回卷件（限价保护卖 floor=bid×0.998+台账制摆回） |

运行环境：OKX 探针需 DB（SG `127.0.0.1:15433`，容器内 `postgres://phase:phase@phase-postgres:5432/phase`）+master key 文件；MT5 探针 flag 传凭据不落盘。

## 3. 核心实证锚（复审对拍基准——关键原值在案）

- **MT5 流面**：OnOrderProfit 持仓态 ~1Hz 推 `ProfitUpdate{Balance,Equity,Margin,FreeMargin,Profit,MarginLevel,Orders[]}`（60s/60 帧实测；空仓静默=push-on-change 语义）；OnOrderUpdate 真单事件发帧实证 `Type=UpdateType_MarketOpen ticket=2567601594 State=Filled` 携全账户摘要；Events/OnMail/OnMarketWatch/OnOpenedOrdersTickets=Unimplemented。
- **MT5 mutating**：OrderSend BUY 0.01 BTCUSDm → ticket 2567601166；**瞬态拒族实证**——开完即平/挂单即改秒级内 `Request processing error`（栈 MarketOpen/CloseWaiterAsync），隔秒+带 `Lots` 重试即成；`Invalid stops` 错向 SL 正确拒；ETHUSDm `Invalid volume`（minLots 异构）。
- **MT5 venue 回归（NOX 期 ok→现病）**：PriceHistoryToday/Month=`Unable to cast List[Bar]`、GetTickValueMany=`Unable to cast Dictionary`、MemorySnapshot=挂起至死线（绕行件各 skill 已标）。
- **OKX mutating 单号在案**：perp BTC-USDT-SWAP 0.01 张 ordId=4037394526527279104 开→filled→positions 实读→reduceOnly 平→归零；划转 clientId=826135644425077/826137848337689 双向 `success`；挂单 lid=vfp* 挂→pending→cancel→canceled。
- **OKX 方言实证**：transfer 写读两侧键名都是 `clientId`（非 clTId——旧码被静默吞，已修）；envelope `code=1 All operations failed` 壳码真因在 `data[].sCode/sMsg`（APIError 已透传）；现货市价买默认 quote 计价、`tgtCcy=base_ccy` 换口径；clOrdId 限 alnum 1-32；availBal 在 `details[]` 非行顶。
- **OKX WS**：私有频道 account/balance_and_position/positions/orders 全活——orders 频道推含自家 clOrdId 真单帧、b_and_p 推 `eventType:transferred`；公开 tickers/books5/funding-rate/mark-price 全有数据帧（修复后复跑 92/132/2/74 帧/15s）。

## 4. 事故实录（自造，已回卷——复审应知情全貌）

okxverify 第二跑复位段把 **Earn 内业主存量 2139.96 USDT 全额抽走买成 PENGU**（误判：savings 只扫首行 BTC 尘→以为无 USDT；且复位逻辑假设「Earn USDT 全属本探针」——无台账）。处置：okxunwind 限价保护卖 278,765.1856 PENGU @0.0076476 单轮 filled→摆回 Earn=2132.91，净耗 ≈$7-15（0.33%），残尘 0.1856 PENGU 在册记实。教训条目= `docs/工程经验库/条目/probe-restore-scope-overreach.md`（critical）。

## 5. 自审已修缺陷（一审记录——复审验修不验有）

| # | 缺陷 | 位置 | 修法 |
|---|---|---|---|
| 1 | 复位段越界（事故根因残留） | okxverify §⑧+transfer_back+repurchase | 台账制：soldUSD/parked 双封顶，Earn 抽回段删除，`-restoremax` 封顶显式复位 |
| 2 | 同族越界 | okxunwind 摆回段 | tra0 基线+proceeds/parked 封顶 |
| 3 | 订阅 ACK 当数据帧计数 | okxverify wsCollect | 判 `data` 字段存在才计数（复跑实证 funding-rate 2 帧为真数据帧） |
| 4 | PASS 构造性灌水 | bills_rows 硬 ok=true、transfer_out 空 clTId 也 PASS | 改真实断言 |
| 5 | description 判词陈旧 | mt5-trading-mutating frontmatter「真户永久禁跑」 | 改真户实测在案口径 |
| 6 | %s/%d 打 float64 | okxunwind:63、mt5stream:90 | 格式化修正（vet 全绿） |

## 6. 复审任务单（建议裁定面）

**已闭环无需重验**：venue 能力事实（真金实证最高证据级）；wsCollect 修复（真户复跑 27 PASS/0 DIFF/0 FAIL 在案）；空集语义分型。

**必须独立审**：
1. **台账算术 diff 审**——`tools/okxverify/main.go` mutateAll §①⑧（soldUSD/parked 流向）与 `tools/okxunwind/main.go` 摆回段（tra0/proceeds/parked）：逐行走查「任一运行态下可动资金是否严格 ≤ 本进程引入资金」。验收判据：枚举无资/部分资/存量大资三种初态，推演每段 cap 取值。
2. **同族残留扫尾**——`grep -rn "AvailBal\|availBal" tools/ internal/ | 查所有「余额→转账/购买」无封顶模式`；判据：任何读余额即全额动用=复发。
3. **机检独立复跑**——`cd /opt/phase-wt/t0048 && go build ./tools/... && go vet ./tools/... && go test ./internal/trading/okxexec/`；okxverify 只读谱（无 `-mut`）：`go run ./tools/okxverify -dsn 'postgres://phase:phase@127.0.0.1:15433/phase?sslmode=disable' -key /opt/phase/secrets/phase_master_key`（需 `ssh -L 15433:127.0.0.1:15433 -N -f sg` 隧道）。
4. **skill 转录保真抽样**——抽 5-8 条实测声明对本文 §3 原值对拍（ticket/ordId/clientId/帧率/错误串逐项）。

**诚实边界**：台账制复位径**从未实跑**（编译/vet 绿≠行为对）——行为验证=下次 `-mut` 执行本身，复审可判「码面正确+待首跑实证」而非「已验」。MT5 面 `GetDemo`/`ChangePassword` 永禁未验、`ConnectProxy` 无凭据——skill 已标缺。

## 7. 文件索引（本 worktree /opt/phase-wt/t0048，随 T-0048 提交）

- 探针：tools/{mt5verify,mt5stream,mt5ordstream,mt5close,okxverify,okxunwind}/main.go
- skills 新建：okx-account、okx-asset-funding；更新：okx-exec、okx-private-ws、okx-market-ws、okx-rest-boundary、mt5-account、mt5-orders-streams、mt5-gateway-admin、mt5-quotes-history、mt5-trading-mutating
- 生产码修复：internal/trading/okxexec/{client.go(APIError 透 sCode/sMsg+clientId), order.go, savings.go(clientId 方言)}
- 经验库新条目：probe-restore-scope-overreach(critical)、exec-error-ambiguous-arbitrate-raw(critical)、probe-silent-stream-false-negative(high)、unwind-limit-floor-protected(high)、verify-harness-verdict-semantics(medium)

## 8. 独立复审裁决（[角色:复审] claude-glm 2026-10-24——四 mandate 全执行）

### mandate 3 机检独立复跑——PASS
build✓ vet✓ okxexec 9/9✓；okxverify 只读谱经 SG 隧道独立实跑：**27 PASS / 0 DIFF / 0 FAIL，与票面声称逐字吻合**（独立取样非转录）。

### mandate 4 skill 转录抽样——PASS（8/8）
clientId 方言（okx-asset-funding:3,20）、envelope sCode/sMsg（okx-exec:34）、tgtCcy 口径（okx-exec:23,26）、availBal details[] 层（okx-account:3,15）、MT5 瞬态拒族+Lots 重试（mt5-trading-mutating:25-29，ticket 2567601166/2567601594 与 §3 逐字同）、OnOrderProfit 1Hz push-on-change（mt5-orders-streams:23，60s/60帧同）、PriceHistory/GetTickValueMany cast 回归+MemorySnapshot 挂起恶化（mt5-quotes-history:15,16+gateway:15）——抽样全部转录保真，ID/帧率/错误串逐项对上。

### mandate 1 台账算术 diff 审——**两处 fail-open，打回补丁**
三初态枚举结论：「任一运行态可动资金 ≤ 本进程引入资金」的保证**在两条分支下不成立**：

1. **okxverify §⑧ transfer_back**：封顶卫 `if soldUSD > 0`——无尘币路径（soldUSD==0，如存量 trading 足额起跑）时封顶失效，`back=tra2-0.5` 全额 trading→funding，且 `parked=back` 后 repurchase `pu=fu-0.5`（封顶=parked=全额）→**存量 trading 净搬入 Earn**。此即原事故形态经 transfer 径的复现——同族残留（mandate 2 判据命中）。
2. **okxunwind 摆回段**：封顶卫 `if tra0>0 && proceeds>0`——零所得路径（无货可卖 proceeds<=0）时 `back=tra-0.2` 无封顶 → 同款存量 trading→funding→Earn 搬运。
3. 边缘：dust 路径 soldUSD 实测 Δ≤0 时（dustInst 已设）买回封顶卫同样失效——并入 1 修。

修法（两处同款，量小）：封顶改**无条件** `back = min(back, max(proceeds 或 soldUSD, 0))`——引入额度为零=零摆回，fail-open 分支物理消失；补锚：无资/零所得初态跑摆回段断言零 transfer 零 purchase。

### mandate 2 同族残留扫尾——命中即上两条
`internal/`（生产码）零命中——余额→动钱模式不存在于产线（产线金额全走 app_config 键非余额扫读）；MT5 侧定张不下余额。残留全部收敛于上列两探针两处。

### 诚实边界确认
台账制修复径未实跑=本席同判「码面待修+修后首 `-mut` 实证」；当前真实账户 trading≈$0.2，两处 fail-open 现实危害≈0——**打回理由是保证语句不成立，非现账损失**。

**裁定：段一二三事实面+机检+转录全过；台账制两处 fail-open 打回补丁（单点、量小），修复合入后本席复审即翻完成。**
