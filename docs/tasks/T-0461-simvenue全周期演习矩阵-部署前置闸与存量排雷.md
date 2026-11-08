# T-0461 simvenue 全周期演习矩阵——T-0145 部署前置闸+存量排雷

- 状态：已批（CTO 自立票 2026-11-08——T-0145 部署前置补件四件裁定之实现票）
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

（施工写：实现摘要 / 测试证据 / 偏离说明）
