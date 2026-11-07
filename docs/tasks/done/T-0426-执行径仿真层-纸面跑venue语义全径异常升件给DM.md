# T-0426 执行径仿真层——纸面跑 venue 语义全径，异常升件给 DM

- 状态：完成（ops_event 完成行 2026-11-07 在案——机械轨票翻）
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

- 状态：待验
- 实现（支 t/0426-exec-sim，全部在 origin/master 之上）：
  - **`internal/trading/simvenue`**——合成 venue（只替 IO 缝）：实现
    `execution.Executor`（Place/Resolve/Cancel）+guard.VenueSizer 三方法
    （结构化隐式满足，不 import guard——方向闸零边）。方言词汇源：
    sz_unit×qty_base=fin_0047/flow.legPx 实约；Instrument 字段=
    /public/instruments（okxexec.InstrumentMeta 同词汇）；拒码 51008/51001
    仓内实证（okxexec/order.go+skills/okx-exec），51005/51006/51007/51020=
    OKX docs-v5 错误码表域。**拒单槽序钦定冻结**：lotSz/minSz/max→价格带
    →not_marketable→insufficient_balance 首中即拒（TestRejectSlotOrderFrozen
    钉死顺序）。订单生命周期态机（live→partially_filled→filled/cancelled，
    Plan 可编程位+OnlyClose 开平异计划）+RULES_VERSION="simvenue/1" 指纹
    （回执携版本——OrderResult 加法性可选字段，真 venue 留空）。费面：
    基准币内扣（canon cex-close-sz-fee-in-base）+quote 买实扣=成交额锚
    （T-0116 实证口径——账实可对平）；SWAP 费 USDT 扣+Fill.FeeCcy 显形。
  - **`internal/trading/simdrill`**——全径演习 harness（跑真码不跑影抄）：
    risk.Gate→execution.Runner→guard.Warden→ledger settle 全产线同码；
    合成腿（quote 买现货+SWAP 反向）进真 flow.legPx/legCash/bareBags/markClosed/
    settle。异常升件径：账实对拍四锚（⓪未平=假绿禁线/①drift 假报/②残袋
    非零未挂账/③合约残仓/④现金对平 venue Δ==Σcashflow）任一破→ops_event
    升件行（seat='演习'，esc_class='执行径演习'，禁静默红）+verdict=fail
    落 fin_exec_drill；费流水=bills 钩同径（kind='fee'/USDT 域/归仓）。
    夭折防护：演习仓位幂等关账防产线仓巡拾起。
  - **`internal/trading/execwiring`**——cmd/core/close.go 装配缝单源拆立
    （CloseExec/CloseIntent/CloseLegOf/OkxSizer）：演习与 cmd 同槽消费同
    一装配径——装配缝复制=影子缝漂移源（T-0043 Close=true 类语义位错一环
    演习即失真）。cmd/core/main.go 换引，close.go 删。
  - **毕业闸挂钩**（migrations/fin_0133_exec_drill.sql+domain.tsv 入册+
    pgstore 登记）：probe→live 判据增「执行径演习通过」——lifecycle.promote
    读 fin_exec_drill latest verdict，判据件缺（无行）/演习败=fail-closed
    不晋（同经济判据待遇，rej 键 exec_drill_missing/failed 已注册）；
    迁移留痕携「执行径演习绿」词。凭据携 RULES_VERSION 审计「哪套规则判的」。
  - **跨包驱动缝**：guard.ScanOnce+ledger.ScanOnce 导出包装（产线 Run/
    ListenLoop 径不变——演习机外部驱动同一拍）。
  - **cmd/simdrill**——演习 CLI（DB_DSN+`-strategy=<id>`；verdict 打印，
    fail exit 1）。产线常驻节拍另议不钦定（票面④）——本件=离线/手动拍口。
- 红桩（票面验收①）：
  - 修前红**已实证**：scratch 树 @ `063df3b0^`（fa228da4，T-0419 施工前）
    贴 sim 三包+RulesVersion/ScanOnce 镜像贴片，同一演习测试
    TestDrillQuoteUnitFullPath 跑红——`verdict=fail detail=残袋非零未挂账
    ——venue 实余 1973 CORE>挂账 0（closed 了但货还在）`（1998−25=美元额
    当 base 只清 1.3%——仓#33 病灶精确重放，仿真层重放真 bug 非纸面虚构）。
  - 修后绿：本支同测试 PASS（真 PG o_test 容器）——平仓腿量=腿实得 1998
    非美元额 25（closeWant 量纲消歧经真消费码可重放验证）。
- 测试证据（O_TEST_PG_DSN=o-postgres-test:5434，全 -count=1）：
  - `internal/trading/simvenue` ok（槽序冻结/量纲语义/合约量纲/态机 partial
    →Resolve 补成→filled/cancelled/撤挂错/Sizer fail-closed——7 test 面）；
  - `internal/trading/simdrill` ok（CORE 案全径绿/残袋升件/平仓拒单未平
    fail 禁假绿+ack 留痕/partial 开仓 ack×exec failed/开腿违网格拒码入
    leg 行 venue_err_code=51005 禁吞——5 test 面）；
  - `internal/portfolio/lifecycle` TestGraduationGateExecDrillFailClosed
    PASS（缺凭据不晋/败不晋/绿晋 live+留痕携演习词）；
  - `internal/trading/execwiring` ok（T-0043 锚随缝迁驻）。
- 偏离说明：
  - cmd/core/close.go 拆立 execwiring=范围必要件（禁影子实现要求装配缝
    单源——复制装配径=演习失真源）；T-0043 锚测试随缝迁驻，语义零改。
  - execution.OrderResult 加 RulesVersion 可选字段（加法性，真 venue 留空，
    消费面零改）——RULES_VERSION「回执携版本」票面要求的承载位。
  - guard/ledger 各 +3 行导出 ScanOnce 包装（演习跨包驱动缝，产线径零改）。
  - 环境坑（非本票码面）：o-postgres-test 有 4 天陈旧事务持 advisory lock
    860223（pg_terminate_backend 清）+229 个遗留测试 schema+10GB 死元组
    膨胀（清+VACUUM FULL→15MB）——master 基线同红实证非本支引入；
    测试库卫生面另票建议（会话尾经验库条目）。
- 涉及面：internal/trading/{simvenue,simdrill,execwiring}（新）、
  internal/trading/execution/execute.go（+RulesVersion 字段）、
  cmd/core/{main.go,main_test.go}+删 close.go/close_test.go、
  internal/guard/guard.go+internal/ledger/run.go（ScanOnce 缝）、
  internal/portfolio/lifecycle/{paper.go,run.go,execdrill_test.go}、
  migrations/{fin_0133_exec_drill.sql,domain.tsv}、
  internal/store/pgstore/migrate_table.go（133 登记）、cmd/simdrill/（新）、
  scripts/check_import_direction.sh（+11 边白名单）、docs/数据流.md（同
  commit 链路更新）。
- 范围外确认：未动 mtexec/MT5 方言（接口留位——演习面限 SWAP 键策略）；
  未建 T-0427 品种主档（venue 册直取留待换源）；未动产线常驻演习节拍
  （另议）；未动 deploy 面与任何 app_config 键。
