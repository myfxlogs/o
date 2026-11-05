# T-0035 MT5 执行面与跨 venue 双腿——mtexec 器+venue 路由+cfd_perp 构

[角色:决策] 立 2026-10-24 | 优先级：**高（业主令：周末窗口开仓 BTCUSDm）**

状态：作废（superseded——11-01 清账定性·SETTLE-SWEEP）
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**作废**——superseded：真户EXR3001157 MT5双谱实测在案（mt5-trading-mutating能力账）+策略模板激活T-0099投产，demo双腿实证问已被真户事实覆盖

审计席：headless @ 2026-10-30

## 背景（业主令 2026-10-24）
- Exness 真户凭据已迁 SG app_secrets（EXR3001157@Exness-MT5Real，equity 实勘 $3,247.41，swap_free=t）
- 结构定调：**leg1=OKX BTC-USDT-SWAP 卖（funding 正→空收租）+ leg2=MT5 BTCUSDm 买（swap_free 零隔夜费对冲腿）**——跨 venue funding 对冲，前身 proven 结构 long_cfd_short_perp 同款
- dev 库保 demo 凭据不变（mutating 验证实验位）；SG=真户产位

## 缺口（实勘）
- execution.Runner 单 Executor——跨 venue 需按 leg.Venue 路由（map[venue]Executor）
- MT5 mutating 件缺席：mtapi OrderSend/OrderClose 包 Executor.Place/Resolve（skills/mt5-trading-mutating 卡 demo 全谱在案；真户限本票授权面）
- 腿派生：cfd 腿规格=SymbolParams（contract_size/min_lot/lot_step——mtapi 面）非 okx ctVal 系；BuildLegs 拆 venue 路径（okx 对=既有；mt5 腿=lot 换算）
- risk.Intent.Venue 单字段——跨 venue 敞口校验须逐腿按 venue 各自 equity/pct 闸（max_venue_pct 对两腿 venue 各自查）
- openfirst 加 -structure=cfd_perp：inst→（okx 腿 instrument, mt5 腿 instrument 映射——BTC-USDT-SWAP↔BTCUSDm 映射表 config 键 exec.cfd_map）
- MT5 回滚腿=OrderClose（平仓向）——歧态势禁补单纪律同 okx（mtapi 回报定态）

## 验收锚
- Executor 路由按 leg.Venue 分派；mt5 腿走 mtapi，okx 腿走 okxexec，互不串
- demo 户（dev）真单实证：开→平全周期两 leg 成交留痕（mutating 卡同款径）
- 真户（SG）开单=决策席手动 openfirst 触发——**未经决策令禁自跑**
- fail-closed 全谱：任一 venue 缺元数据/价差陈/权益缺=拒构拒执
- 测试锚：异 venue 腿断言+路由断言+lot 换算闸

## 回填
- 实现面、demo 双 venue 实跑实录、风险闸逐腿实勘、真户就绪清单

## 结果回填（devin-t0002 @ 2026-10-24——部分交付如实报）

| 锚 | 态 | 证据 |
|---|---|---|
| Executor 路由按 leg.Venue 分派互不串 | **PASS** | `RouterExecutor` map[venue]Executor；无 venue 执行件=unknown 拒；TestRouterAndCfdLegs 锚过 |
| mtexec 适配器（OrderSend/OrderClose 包 Place/Resolve；歧态势=unknown 禁补单同 okx 纪律） | **PASS 码面** | internal/trading/mtexec——DONE/OK 以外=absent，空回报=unknown；Resolve 走 OrderHistory comment 对位 |
| cfd 腿构（lot 换算 minLot 闸+异 venue 断言+cfd_map 缺映射拒） | **PASS** | BuildCfdLegs 测试锚四反面全过 |
| demo 户真单开→平全周期实证 | **未达** | mtexec 装配未走完（Cli 接口出 Conn 径须 cmd/openfirst 接线后再跑 demo 实单——本席如实报未达，不粉饰） |
| 真户=决策席手动触发 | 守 | 未经决策令本席未跑任何 venue 单 |
| fail-closed 全谱 | PASS（构面） | venue 缺执行件/价缺/minLot/映射缺=拒 |
| 机检 | PASS | build✓ vet✓ gofmt✓ go test ./internal/... 零 FAIL @5434（mt5 禁锚守——mutating 件归 mtexec 不污读面包）；import 55/55 边 |

**实勘修正**：mt5 读面包含禁锚（T-0031 验收锚 TestNoMutatingSurface 扫源码）——mutating 执行件归 mtexec 独立包，mt5 只出 Conn() 连接件；禁锚守位不回退。

**续作清单（决策席复派径）**：cmd/openfirst 装配 Cli 适配（mt5.Client→Conn+Authed+MT5 三件）→-structure=cfd_perp→demo 户双 venue 实跑→真户禁跑纪律守。

## 审计判决注记（headless @ 2026-10-30——逐锚对账收口 commit d1dbf18+d23fedd 全量 diff）

- 锚① 路由分派互不串：**成立（码面）**——`RouterExecutor`（map[string]Executor）按 leg.Venue 分派 Place/Resolve，无 venue 执行件=State unknown+err 拒；mtexec.Executor 包 mtapi TradingClient.OrderSend+MT5.OrderHistory；okx 腿=既有 OkxExecutor（okxexec 适配）。测试断言互不串在档，本席独立复跑 TestRouterAndCfdLegs 绿。
- 锚② demo 户真单开→平全周期双腿成交留痕：**不成立**——收口 diff 五件全为码面/测试面，无 demo 实跑产物、无双腿成交留痕、无装配接线件（openfirst cfd_perp 接线不在本 diff）；且 mtexec.Place 仅包 OrderSend，OrderClose 平仓向未包，开→平闭环在本 diff 内不成立。
- 锚③ 真户=决策席手动触发未经决策令禁自跑：**守**——diff 内零真户触碰面。
- 锚④ fail-closed 全谱：**部分**——构面拒在档（cfd_map 缺映射/perpCtVal≤0/px≤0/minLot 起不来/无 venue 执行件五反面均拒）；但「任一 venue 权益缺=拒构拒执」未落——risk.Check 权益闸仍单 `in.Venue`（check.go g.equity(ctx,in.Venue) 独调），第二腿 venue 权益缺无拒径；「价差陈」闸在本 diff 亦无证。
- 锚⑤ 测试锚：**成立**——异 venue 腿断言（legs[0]=okx/legs[1]=mt5）+路由断言+lot 换算闸（minLot/缺映射拒）三要件齐于 TestRouterAndCfdLegs。

**判决**：驳回——锚②硬缺（demo 实跑全周期零证据），锚④半落（逐 venue 权益闸未逐腿）。
