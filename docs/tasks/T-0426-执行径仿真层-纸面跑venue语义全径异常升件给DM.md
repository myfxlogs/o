# T-0426 执行径仿真层——纸面跑 venue 语义全径，异常升件给 DM

- 状态：施工中
- 领取：zcode-t0426 @ 2026-11-07（外派施工，O-Dev 径——venue语义钦定全，不占 DM 钱径额度；支 t/0426-exec-sim @ /opt/o-wt/t0426）
- 优先级：P1——执行码验证面结构性缺失（T-0419 实证类）
- 收口移交归：决策席
- 关联：T-0419（单位错配根修=本层首案实证）、T-0420（真钱
  侧监察案——本层=纸面侧镜像）、fin_0130（挂账表同用）

## 病灶（业主令出处）

毕业梯 paper→probe→live 全程虚拟记账（`paper.go:116`
probe 同构吃 md_funding），**执行码第一枪在 live 第 20 秒开
$50 仓**——venue 订单语义类 bug（sz_unit=quote 被当 base、
残袋、假 drift）在现有梯上结构性不可发现。业主判：这类
bug 该在纸面就找到——venue 语义可仿真，执行码全径可在
纸面跑，不必真钱演习。

## 范围

1. **合成 venue 回包面**：按 venue 方言生成 fill——
   `sz_unit`（contract/base/quote）×实得 `qty_base`×部分
   成交×费率；首案=CORE 模板（quote 买入回 qty_base≠sz）
   +既有 contract 腿；venue 方言表自 `okxexec`/`mtexec`
   实约取，禁拍脑袋编；
2. **执行码全径仿真跑**：合成腿进真消费码——drift 判定
   （`guard/decide.go` 族）→平仓计划（`sizing.go` 族）→
   残袋对拍（`bareBags`/fin_0130 族）→settle 账实对平；
   **跑真码不跑影抄**——仿真只替 venue IO 层，消费件
   一律产线同码（禁平行实现）；
3. **异常升件径**：任一环账实不平/drift 假报/残袋非零
   →ops_event 升件行给 DM（禁静默红）+fixture 断言落
   测试面（本票红桩即交付物）；
4. **挂拍形**：先离线/测试拍可跑（o-core 内嵌拍或
   check_all 桩），产线常驻节拍另议不钦定；
5. **venue 决策语义层（HKUDS Vibe-Trading 参照件，
   业主令 11-07 补）**：仿真不止回包，须仿 venue
   「接不接单」的决策——
   - 拒单槽序钦定冻结：lotSz/minSz→价格带→not_
     marketable→insufficient_balance，首命中即拒
     （确定性，顺序写死进测试）；
   - 规则矩阵按 inst_type 分件：contract/spot/future
     各携 lotSz/ctVal/tickSz/minSz（字段源=
     `/public/instruments` 实约——远期由 T-0427
     品种主档供，本票先同径直取不落表）；
   - 部分成交+量上限+订单生命周期状态机；
   - 引擎指纹（RULES_VERSION 族）：回执携规则集
     版本，审计可验「哪套规则判的」；
   - 拒单=一等事件入回执，禁吞；
6. **毕业闸挂钩（业主令 11-07：「每策略毕业前就该
   检查订单会不会出错」）**：lifecycle probe→live
   判据增一项「执行径演习通过」——该策略型的
   开→平→对账全径仿真跑绿才准 live；判据件缺/
   演习败=毕业闸 fail-closed（同经济判据待遇）。

## 验收

- 红桩：quote-unit 腿 fixture 修前=drift 假报/平仓量错/
  残袋盲三病各断言红（T-0419 修后码上应转绿——验证
  仿真层能重放真 bug 非纸面虚构）；
- venue 方言至少 contract+quote 两 unit 覆盖；
- 拒单槽序断言：违 lotSz/minSz 单=拒且原因码对；
- 毕业闸断言：演习未过策略 probe→live 被拒（红桩）；
- 异常升件行断言（ops_event 可读）；
- 绿戳对口+全链。

## 结果回填

- 状态：待派
