---
schema_version: 2
id: arch-import-audit
kind: map
risk: high
status: verified
keywords: [开源架构, 引进审计, 成本模型, venue连接器, 状态机, level-trigger, saga, Temporal, hummingbot, 裁定]
scope: [架构引进, 钱径, 执行径, 调度环, 审计图]
summary: 工单#25771（扩件令#25778 三候选→五）——开源架构引进审计：①统一成本模型②venue连接器抽象③订单/持仓FSM收口④K8s水平触发调协⑤Temporal/saga，各出adopt/reject裁定+最小引进设计；采纳件派施工票在案
first_seen: 2026-11-20
last_verified: 2026-11-20
skills: [施工席, 复审席]
evidence_refs:
  - type: code
    ref: internal/trading/execution/execute.go:50（Executor iface=Place/Resolve/Cancel+EarnMover）
  - type: code
    ref: internal/portfolio/lifecycle/friction.go（lifecycle.friction_bps.<venue> 校准成本模型）
  - type: code
    ref: internal/trading/mtexec/mtexec.go（MT5 腿走 execution.Executor 包装——接口先行已事实完成）
  - type: code
    ref: migrations/fin_0009_execution.sql+fin_0082（fin_execution/fin_order_leg/fin_position 状态词表）
  - type: code
    ref: ops_event#17957（River/pg_timetable/Temporal 重框架先例否决账）
---

# 开源架构引进审计（工单#25771——业主令「引进优架构非搬码」+僵尸单14天事故教训面）

口径：每候选一裁定（adopt/reject/部分采纳）+最小引进设计；采纳件各派
施工票（ops_event 工单行在案）。证=业主令扩到五件+非基金开源方法论引进旨。
审计基线=origin/master@2a33a19f1。

## ① 现实模型件——费率/滑点/资金费统一成本模型（评估径∥执行径共享）

| 列 | 内容 |
|---|---|
| 现状证据 | 三处成本口径并存互不读同一源：`lifecycle/friction.go` 有校准摩擦模型（`lifecycle.friction_bps.<venue>` app_config 键，frictionDefaultBps=20/frictionBufferBps=5/frictionCalEvery=6h 校准环）；各研究臂自带 fee 参数（gridfit fee_rt、regime/dirmom 各族自裁）；执行径用 venue 实返（`OrderResult.Fee` 真账，工单#11438 净额锚）。纸面判词（eval 估费）与实盘成本（exec 实费+friction 校准）无共享读面——paper-vs-live 一致性从未审计过。 |
| 裁定 | **adopt（有界）**——先行件=纸面vs实盘一致性审计（本票已粗审：三口径并存实锤），引进=统一成本读面非统一模型本身。 |
| 最小引进设计 | 不新立模型：抽 `internal/platform/costmodel`（或并入 config 域）单源读面——venue→{fee_bps, slip_bps, funding_apy} 三元组，lifecycle friction 校准环改写此源、eval 臂 fee 参数改读此源、exec 侧 OrderResult.Fee 实账对拍此源出偏差告警。范围=读面统一+对拍探针，不动任何判据阈值。 |
| 施工票 | `工单/arch-cost-model`——一致性审计表（eval臂×exec径×lifecycle三面对拍）+单源读面落地+偏差升件阈。 |

## ② venue 连接器抽象——MT5 执行径接口先行（对照 hummingbot connector 形）

| 列 | 内容 |
|---|---|
| 现状证据 | **接口先行已事实完成**：`execution.Executor` iface（Place/Resolve/Cancel 三方法，execute.go:50）+`EarnMover`（RedeemToTrading）；okxexec.Client 与 mtexec.Executor 双实现在案（mtexec 注释明载「包 execution.Executor——跨 venue 双腿的 cfd 腿走此径」）；execwiring 为装配缝单源（CloseExec/EmergencyExec/OkxSizer，simdrill 同槽消费防影子缝漂移）；exec.xvenue 先例在 sigdist。对照 hummingbot connector：其价值面=能力协商（每 venue 声明支持订单型/改单/预算检查）+connector 健康面；我方 Executor 无能力查询面，venue 方言经 Leg 字段携。 |
| 裁定 | **reject（框架层）+窄件缓议**——抽象层已存在且双 venue 验证，重写=纯搬码违业主令；能力协商面无真实需求触发（MT5 mutating 已 demo 全谱实测在案，无第三 venue 在途）。待第三 venue 立项或改单需求实发时再按需扩 Executor 能力查询方法，不预建。 |
| 最小引进设计 | 不引进。记录对照结论备查：若未来扩 venue，先例=照 mtexec 形（包 execution.Executor+venue 方言收口在 adapter 内+execwiring 装配缝登记），能力矩阵按需随立项票带。 |

## ③ 订单/持仓状态机收口——fin_execution↔fin_position FSM 耦合修

| 列 | 内容 |
|---|---|
| 现状证据 | 三层状态词表各自为政：fin_execution（open/legged/rolled_back/closed/failed/rollback_failed，fin_0082 增 ambiguous）；fin_order_leg（placed/ack/filled/failed/ambiguous/rolled_back）；fin_position **无 state 列**——open/closed 经 closed_at 推导，与 leg/execution 态无单源派生关系。僵尸单 #1117/#1134 事故=持仓态与腿态脱钩 14 天未收敛，age-gate 是补丁非根治（超龄才动，脱钩窗口仍在）。execution 包内已有回滚机器（rolled_back/rollback_failed 16 处引用）但状态迁移无单一转移表约束。 |
| 裁定 | **adopt**——僵尸单实证+age-gite 补丁论证成立，是五件中证据链最硬者。 |
| 最小引进设计 | 学 FSM 收口法不引框架：①fin_position.state 物化列（由 fin_order_leg 终态+fin_execution.state 单源派生，写径收口在 execution 落账同事务）；②状态转移表落 `docs/` 或包内 const 表（合法迁移集显式枚举，非法迁移=落账拒）；③reconcile 拍=每巡对 fin_position open 集重推期望态 vs 实际腿态全差（与④水平触发同法复用）。范围=状态单源化+转移合法性闸，不改业务语义。 |
| 施工票 | `工单/arch-fsm-converge`——fin_position.state 物化+转移表+reconcile 巡（含僵尸单回归用例：脱钩注入→巡内收敛）。 |

## ④ K8s 调协环——扫描件边沿触发→水平触发（每拍重推期望vs实际全差）

| 列 | 内容 |
|---|---|
| 现状证据 | 扫拍族枚举（roster ~25 席+独立 binary）：①研究臂族（dcd.go researchArms ~10 臂，dirmom/gridfit/regime 等 ticker 拍）；②lifecycle scan+admitProposals+retireStaleArmed；③hunger Detect/Respond+miner+param_intake；④okxws 采集族（klinesub/collector/stalewatch/klineback/scan 五 ticker）；⑤平台探针族（pgguard/hostmetrics/roster/eventbus/projector/feed_esc/retention/event_push/trxvote ~10 ticker 点）。**关键判读：现有拍族绝大多数本就是水平触发等价物**——每拍重读 DB 全态重推（scan() 全量重估），「边沿」仅存在于拍内 diff 判定（新提案/新信号才动作），漏拍=下拍自然补，非致命。真正边沿致命面=事件订阅消费径（WS 推送/notify 直推）若有漏收不重推机制——本审计未发现此类钱径消费点（okxws 订阅面有 stalewatch 兜底重拉）。 |
| 裁定 | **部分采纳（方法论）**——采纳「期望vs实际全差重推」为审计判据与新件设计律；逐件改型**不批量立项**：现存拍族已是水平等价，改造=零收益搬码。唯一实改候选=③FSM reconcile 巡（已并入③施工票同法落地）。新增扫描件立项时强制按水平触发设计写入施工单模板。 |
| 最小引进设计 | 不引 K8s controller/framework：水平触发表=「每拍 SELECT 全集→与期望态 diff→全差动作」五行模板进施工单规范；漏通知不致命性=拍周期上界即恢复上界（现状 ticker 30s-6h 已满足）。 |

## ⑤ Temporal 持久化工作流/saga 补偿——钱径多步流审计

| 列 | 内容 |
|---|---|
| 现状证据 | 钱径多步流枚举：①execution Intent open=两腿原子+回滚（fin_order_leg placed→ack→filled/failed/ambiguous→rolled_back 全链在案，fin_execution rolled_back/rollback_failed 终态）；②close 径同构；③EarnMover RedeemToTrading→开仓前置赎回（败=fail-closed 不开仓）；④dcdexec preRedeem+到期结算消费；⑤划转/扫额（okxpriv/okxasset transfer 族）。**断腿面盘点：腿级补偿态已存在**（ambiguous/rolled_back/rollback_failed 三态词表），缺的是**流程级补偿编排**——多步流跨表跨 venue 时中途断点的恢复靠各 Runner 巡巧合对齐，无显式 saga 步骤表。先例账：#17957 已否决 Temporal/River/pg_timetable 重框架（最重成本=ops_event ticket_id 列+事件语义收紧）。 |
| 裁定 | **adopt 方法、reject 框架**——学 saga 补偿形（步骤表+补偿动作+断点续跑），Temporal 框架否决维持 #17957 判词（钱径多步流量级不配 workflow 引擎，PG+ops_event 已是持久层）。 |
| 最小引进设计 | `internal/platform/saga`（或并入 execution 域）最小件：Saga 表={flow_id, step_seq, step_name, venue, idem_key, state(pending/done/compensated/failed), compensate_ref}；每多步流（open两腿/close/Earn赎回链/划转）登记步骤表，断点恢复=按 flow_id 续跑未完成步+对 done 步逆序补偿。首批只覆盖 execution open/close 两流（断腿面最痛处），不追全覆盖。 |
| 施工票 | `工单/arch-saga-journal`——Saga 表迁移+execution open/close 两流步骤表登记+断点续跑径（注入中途断→重启→补偿/续跑验证）。 |

## 裁定汇总

| # | 候选 | 裁定 | 施工票 |
|---|---|---|---|
| ① | 统一成本模型 | adopt（有界——读面统一+对拍，先一致性审计） | 工单/arch-cost-model |
| ② | venue 连接器抽象 | reject 框架（接口先行已完成）；能力协商面缓议待真实需求 | — |
| ③ | FSM 收口 | adopt（证据链最硬——僵尸单实证） | 工单/arch-fsm-converge |
| ④ | 水平触发调协 | 部分采纳方法论（现有拍族已水平等价，不批量改型） | 并入③reconcile 巡 |
| ⑤ | saga 补偿 | adopt 方法、reject Temporal 框架 | 工单/arch-saga-journal |

采纳件施工票三枚已派（ops_event 工单行）；②缓议不立票（无触发件）。
审计方法论留档：引进判据=「现有件是否已含等价物→最小增量是否解决
实证痛点→框架级引进须过 #17957 重框架否决账复审」。

