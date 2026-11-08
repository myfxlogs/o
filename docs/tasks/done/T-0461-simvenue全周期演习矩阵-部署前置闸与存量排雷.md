# T-0461 simvenue 全周期演习矩阵——T-0145 部署前置闸+存量排雷

- 状态：完成（决策席裁决通过 2026-11-08——判词在案·机械轨票翻）
- 领取：常径待派
- 优先级：P1（T-0145 部署令硬前置——裁定+凭据未绿前部署令不发；
  业主令「把已毕业的，都拿来纸面过一次」存量排雷同批）
- 收口移交归：决策席
- 立项：T-0145 票面「部署前置补件」CTO 裁定四件（见裁定节——矩阵格集/
  凭据落点/排雷口径/本票归属钦定）

## 背景锚

- 业主令（COS 立壳在案）：simvenue 全周期演习=T-0145 部署前置闸，
  open→hold→强平全周期、双腿残留=0 才算绿（pos33/34 裸袋案实证：
  开仓绿≠平仓绿）；粒度=模板×venue×方向矩阵不逐策略重跑。
- 现有基建（T-0426）：simdrill.Drill 全径（Gate→Runner→Warden→ledger
  四锚对拍）+fin_exec_drill 凭据表——现仅 funding-hedge 腿形单格。
- T-0145 钦定不变件：exec.xvenue_open 暗发闸+真户微额彩排判据照旧
  （sim 闸不替真户彩排——两闸串行）。

## CTO 裁定（T-0145 补件节同款——本票范围钦定源）

① **演习矩阵格集**（格=template×venue对×direction，面值=格内两档）：

| 格 | venue 对 | direction/腿形 | 在产占用 |
|---|---|---|---|
| funding-hedge@okx | okx 单所 | short_perp_long_spot（现货多+perp空） | live34+probe1 |
| dcd_put@okx | okx 单所 | dcd_put（产品申购非双腿径） | paper43+armed11 |
| quarter_basis_long@okx | okx 单所 | basis_quarter（BuildBasisLegs 双腿） | paper1+armed1 |
| perp_quarter_spread@okx | okx 单所 | perp_quarter_spread（spread 双腿） | paper1+armed1 |
| xvenue-carry@mt5×okx | mt5×okx | long_perp_short_cfd（mt5卖+okx买） | T-0145 新开 |
| xvenue-carry@mt5×okx | mt5×okx | short_perp_long_cfd（mt5买+okx卖） | T-0145 新开 |

- 格集机器枚举禁手编：fin_strategy(state∈live|paper|probe|degraded)
  ∪fin_proposal(state∈fin_proposed|fin_armed，template 经 eval 对位)
  template×venue×direction 去重∪xvenue-carry 双向钦定格；词表零占格
  =免演登记（枚举式落票面注释——新增格自动入谱）。
- 面值档：每格两档=格内最小在产名义（粒度/单位错配咬 pos33/34 类）
  +中位名义；零占用格（xvenue）=首单规模钦定 $50 单档。
- 判据：双腿族=open→hold→哨戒平→对账全径+venue 双腿残留=0+现金
  对平（现四锚+双腿残留断言扩）；DCD 族=申购→到期结算→账实平+
  锁仓余额=0（产品生命周期全径——残留定义随腿形）。

② **凭据落点**：fin_exec_drill 同表加 `cell_key` 可空列（迁移）
   ——格键='template@venue对/direction#面值档'机读；strategy_ref=
   演习 ephemeral 策略 id（凭据归属可溯）；毕业闸读径（latest verdict
   by strategy_ref）零影响；部署闸读径=格集 latest verdict=pass 全覆盖。
   不立新表。

③ **存量排雷**：格级非提案级——13 张 armed 分属 dcd_put/
   perp_quarter_spread/basis_quarter 三格全在矩阵内，格绿即排雷毕；
   armed→open 消费格凭据 latest=pass 同构毕业闸 fail-closed——格凭据
   未绿格内提案 hold 不放行+计数。

④ **实现归属**：本票（非 T-0145 返修续件）。

## 范围（钦定）

1. 格集枚举器+演习批跑器（cmd/simdrill matrix 模式——格集遍历+
   凭据落账+任一格 fail=升件 esc_class='执行径演习' 沿用）。
2. 模板→腿形注册（funding-hedge/basis 两模板/xvenue-carry 双腿原子
   径/dcd 产品径）；每格两档面值跑法。
3. **mt5 侧 sim**（xvenue 格必演前置）：cfd 乐器模型（lots/
   contractSize/minLot/TradeMode 闸）+ticket 回写+OrderClose 全周期
   +OpenedOrders 形位读；Runner venue 路由（okx-sim/mt5-sim 双腿分
   派）——跑产线同码 xvenueOpen→Execute LEGGED→rollback→
   Close(ticket) 全链。仿真方言/拒码=mtexec 实约族（禁拍编——同
   simvenue 方言冻结律）。
4. fin_exec_drill `cell_key` 可空列迁移+migrate_table.go 登记。
5. armed→open 格凭据闸（提案格 latest verdict=pass 缺/败=hold+计数
   ——fail-closed 同构毕业闸消费径）。
6. 桩测谱：格枚举断言/双腿残留=0 断言/mt5 腿序+回滚+ticket 全周期
   断言/格凭据闸 hold 断言/DCD 全周期断言。

## 验收锚

- 六格全谱跑绿凭据在案（fin_exec_drill cell_key 各行 verdict=pass
  +rules_version 携）；
- xvenue 双腿序/败一腿回滚/OrderClose ticket 全周期在 sim 谱咬中；
- 格凭据闸 fail-closed 桩（无凭据格提案 hold 断言）；
- check_all 全绿+先红桩（部署闸缺失断言修前红）；
- 部署令序钦定：矩阵凭据全绿→真户微额彩排（原判据不动）→部署令。

## 红线

- 纸面域——零真 venue 写/零凭据读域（sim 凭据一切禁入）；
- 演习 ephemeral 行自洁（24h TTL 退役同现径）；
- 不超格集外加格；不动 simvenue 拒码冻结域/既有四锚语义；
- 钱径/凭据域/真单径语义禁碰（mt5 sim 只仿方言面禁接真会话）。

## 结果回填

### 实现摘要（施工 2026-11-08）

- **格集枚举器** `internal/platform/excell`：机枚举唯一真源——
  fin_strategy 占用格 ∪ execfam 词表零占免演登记 ∪ xvenue-carry
  双向钦定格（mt5xokx，空格 $50 单档）；占用格两档=min+median
  active notional 去重；`CellKey` 唯一编码
  `template@venuepair/direction#usd=N`；`Normalize` 副腿坐标归并；
  `GreenKeys`/`CellGreen` latest verdict fail-closed 全档齐绿判。
- **execfam.All()** 词表导出（零占免演登记消费面+矩阵全覆盖断言锚）。
- **mt5sim** `internal/trading/mt5sim`：CFD 仿真 venue——
  OrderSend/OrderClose/OpenedOrders/OrderHistory/SymbolParams/
  OrderModify/Cli/Dialect/Meta/ContractPos/LotSz/SpotAvailBal 全缝；
  ticket 序列 700001+、TradeMode=4、拒序=乐器缺→交易态→量纲→
  平仓票缺；方言/拒码=mtexec 实约族同形（禁拍编律守）。
- **simdrill matrix 模式**（`cmd/simdrill -matrix`）：模板→腿形
  注册表 `legShapeOf`（funding/basis→okx_two_leg、xvenue-carry→
  xvenue、dcd_*→dcd；未登记=显式 fail 凭据不静默漏）；每格每档
  ephemeral 策略行（template_ref=`simdrill:cell:<cell_key>` 防
  dedupe 撞）→开→持→哨戒强平→结算→venue 残留=0 对拍→凭据落
  `fin_exec_drill.cell_key`；任一格 fail=ops_event 升件+Failed 计数。
- **xvenue 腿构**直用 `execution.BuildXvenueLegs` 产线同径（mt5 腿
  先 okx 腿后钉死）；execTwoLeg 走 `execution.Runner`+`RouterExecutor`
  venue 分派+`mtexec.Executor` ticket 全周期+`Warden` 哨戒 venue 路由。
- **DCD 抽立** `internal/trading/dcdexec`：cmd/core dcdOpen 产线同码
  单源（quote→risk→intent→trade→status→exec/position 态迁；venue
  三步缝注入式；cmd 侧仅规格映射+缝别名）。
- **迁移** `fin_0144_exec_drill_cell.sql`：`fin_exec_drill.cell_key`
  可空列+migrate_table.go {144} 登记。
- **armed→open 格凭据闸** `internal/portfolio/lifecycle/cellgate.go`：
  openLive 拍首装载格集+latest 判词，候选格归一坐标→CellGreen 全档
  齐绿才放行；缺/败=hold+`exec_cell_hold` 计数（rejects 注册表登记）；
  xvenue 副腿行豁免；读败 fail-closed 全 hold。
- **import direction** 白名单 9 边登记（excell/execfam、lifecycle→
  excell、dcdexec→risk+okxexec、mt5sim→gen/mtapi/mt5、simdrill→
  excell+dcdexec+mt5sim+mtexec）。

### 测试证据

- `go build ./...`、`go vet ./...` 全绿。
- `go test -count=1 ./internal/trading/simdrill` 绿——矩阵全径：
  funding 两档/basis/DCD/xvenue+残留=0+凭据落账断言。
- `go test ./internal/portfolio/lifecycle` 绿——格闸 hold→fail-hold→
  pass 放行三段断言+既有诊断闸序回归（TestXvenueLifetimeDirectionSign
  等）。
- `go test ./internal/trading/mt5sim ./internal/platform/excell
  ./internal/trading/dcdexec` 绿——ticket 生命周期/拒序/枚举/态映射。
- `check_migrations` 七断言过；`check_import_direction` 182/203 边 OK；
  `check-lines` 全绿（行数棘轮拆件：matrix.go→matrix_exec/legs/
  venue.go、opener.go→cellgate.go、mt5sim.go→mt5sim_seam.go）。

### 偏离说明

- `fin_exec_drill.inst_id` 列载本格代表乐器（`cell_key` 列载机读格键
  ——语义分记不混）。
- 格闸判位=r.op.Open 前（诊断闸序不变）而非候选前置过滤——保
  lifetime/breakeven/振荡诊断与 fin_eval_reject 落账面不变；
  TestXvenueLifetimeDirectionSign 序回归实证。
- 验收锚「六格全谱跑绿凭据在案」：桩测断言凭据落账机制在案；真库
  全谱跑批=部署前置运维动作（`simdrill -matrix` 一键产凭据）。
- 范围外发现：无（勘察期未见须另立票病灶）。
