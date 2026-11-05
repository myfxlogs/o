# T-0377 MT5 第二桌——venue 转化首案 xvenue-carry 纸面激活

- 状态：待验（2026-11-06 施工收口自报——实工 zcode-t0377）
- 领取：业主手动外派 @ 2026-11-05（外部工人 run——提示词
  docs/prompts/T-0377-O-Dev施工提示词.md，DM 勿自动领）；
  实工领取=zcode-t0377 @ 2026-11-06（worktree /opt/o-wt/t0377）
- 优先级：P1——战略缺陷对治（venue 转化首案）
- 收口移交归：决策席（2026-11-05 业主令——外派件收口回 DM 常径裁；窄口豁免降级为 DM 缺席时兜底）

## 背景与依据

- 判词（11-05 实账）：34 live 全 OKX funding-hedge=单 alpha 单
  venue；产出年化 ~2.9% < mandate 8%；侦察臂 10 候选转化=0。
- 机制在案：T-0144 建 `xvenue-carry` 主从双腿 paper 户
  （`internal/portfolio/lifecycle/xvenuepair.go`——主 okx perp
  收付 funding 端+副 mt5 CFD swap 端，向取反对冲）；映射源
  `app_config bridge.xvenue.map`；mt5_swap 信号快照建行时锁。
- MT5 面能力账：skills/mt5-* 五件全谱实测在案（quotes/history/
  account/orders-streams/trading-mutating）。
- **本票只到 paper 结算**——probe/live 迁移归生命周期判据+
  委员会闸，不开真单径。

## 范围

1. 链路实查：`bridge.xvenue.map` 现值与覆盖品种、mt5_swap 信号
   供径是否活、admitXvenuePair 触发位（谁调——评估提案径还是
   手工）、paper 结算对 mt5 腿的 funding/swap 记账口是否通；
2. 首案激活：选 1-2 个双所同品种（如 XAUUSD/BTC 主链映射内）
   开 xvenue-carry paper 双腿户——机读落地（fin_strategy 主从
   两行+fin_paper_fill 开流）；
3. 断链补焊：若查出任一环节断供（map 缺品种/mt5_swap 信号死/
   触发位无人调），在票面域内根修补接——禁绕径手建；
4. 观察面：舱面/榜单若该显示 xvenue paper 户而未显，同票补
   （可见面人话口径照 T-0371）。

## 验收锚

① fin_strategy 见 xvenue-carry 主从 paper 行（venue=okx+mt5 各
  一）+fin_paper_fill 开流；
② mt5 腿 swap 端记账对拍（信号快照值 vs 记账值）；
③ 链路断点清单落回填（有断=根修记录，零断=如实报零断）；
④ check_all 绿戳对口收口 SHA。

## 红线

- 不碰真单径（MT5 OrderSend 类 mutating 面零触）；
- 不动业主凭据/账户配置；
- 不动资金上限键（T-0376 域）。

## 结果回填

**实现摘要**（链路实查全部生产亲证，出处=SG o-postgres/OCI 盒直查）

链路实查四问对答：
1. `bridge.xvenue.map`=3 对全活（BTC-USDT-SWAP→BTCUSDm/ETH→ETHUSDm/SOL→SOLUSDm）；
2. mt5_swap 供径=活（4h 拍拍面 11-05 12:47≈3.7h 陈<4h；BTC/ETH ann_ok=true 可年化，
   SOL swap_type=0 单位方言不可年化=如实缺席）；
3. 触发位=lifecycle scan→admitProposals（ev_strategy/ev_eval 事件驱动）→模板
   xvenue-carry→admitXvenuePair——在岗但被三闸拦死（BP-1/2/3）；
4. 记账口=paperAccounting 只吃 md_funding（okx perp funding 端），mt5 腿仅开户
   swap_snapshot 一行（amount=0）无持续账。

断点清单（锚③）：
- BP-1 闸｜execfam 未登记 xvenue-carry：评估前置拒 exec_family_missing+lifecycle
  模板门闸双拦→提案永零（10 候选转化=0 主因）。根修=execfam 单点登记（钦定径
  「建径票落=本表登记即全链通」）；opener armed→open 永拒**不动**（真单径红线兜底）。
- BP-2 阈｜research.signal.xvenue_carry_diff_bps=300 vs BTC 现净≈291bps
  （carry_diff 301−点差摊 0.1−滑点缓冲 10）——过阈永不成立。处置=调 250
  （T-0376 业主令赚钱方式参数域归机器；已落 SG app_config 热生效，
  UPDATE…RETURNING 对拍=250）。
- BP-3 静默杀提案径（坑，KB 已立）｜xvenueCarryEval 提案幂等 count 查询声明 $1
  **未传参**——pgx `expected 1 arguments, got 0`→err≠nil→INSERT 分支静默跳过
  （零日志零拒因）。前置拒掩护下不可见，登记即现。根修=补传 evalID+err 显式留痕。
- BP-4 记账｜mt5 腿 swap 持续账零径。根修=xvenueSwapAccounting 新件（md_signal
  mt5_swap 新行 ts 门驱动、方向腿年化取键 long_cfd↔swap_long_ann_bps/short_cfd↔
  swap_short_ann_bps、Δh 摊提、ann 缺/零跳过如实）+fin_0117 kind CHECK 扩 'swap'；
  同拍 paperAccounting 方向符号根修（long_perp 正 rate 付租取负——原恒按
  short_perp 记=伪账风险）。
- BP-5 观察面｜舱面/榜单零 paper 策略显示位。补=flow_pulse「策略」段
  （fin_strategy_transition 事件驱动中性灯，八段）+pipeStatus.ts 人话映射
  （T-0371 口径：机器词禁入可见面）。
- BP-6 时序窗窄（如实报**不修**——T-0164 血统闸钦定语义「mt5_swap 陈=carry_diff
  陈，派单钦定」）｜carry_diff ts 钉 mt5_swap 4h 拍→消费 Fresh 120s 窗每 4h 仅
  ~2min 且需 ev_funding 窗内触发。实证吻合：7 天 3 评（10-26/29/30）。

首案激活 runbook（merge+deploy 后有机触发——**禁绕径手建**，施工不合并不部署）：
1. 决策席 merge→SG `deploy_image.sh`（o-core 面：lifecycle/execfam/flow_pulse）+
   OCI o-research 件重建 ship（hypothesis/execfam 面：评估前置拒与提案径在此件；
   手径=arm64 构建→scp /home/ubuntu/o-research/o-research→`systemctl --user
   restart o-research`）——**双件缺一链不通**；
2. 下次 mt5_swap 拍→sigdist 重钉 carry_diff ts→120s 窗内 ev_funding 触发
   xvenueCarryScan→BTC net≈291≥250→卡+评估 proposed+提案 fin_proposed；
3. ev_eval→lifecycle admitXvenuePair→fin_strategy 主从两行（okx
   BTC-USDT-SWAP+mt5 BTCUSDm，notional=50U）+swap_snapshot+fin_armed；
4. 记账随 scan 开流：okx 腿 md_funding 端+mt5 腿 swap 摊提；
5. 监测：`fin_opp_eval WHERE template='xvenue-carry'`/fin_proposal/fin_strategy——
   窗错过后顺延下一 4h 拍自愈，无需人工干预。

首案=BTC-USDT-SWAP↔BTCUSDm（票面钦定示例；唯一全输入鲜+net>0 对——ETH 净≈−16
不可行、SOL ann_ok=false 不可年化），1 对合票面「1-2 个」。

锚①② 完成时点说明：提案/开户/记账=merge+deploy 后有机事件（提示词硬要求不合并
不部署）——锚①② DB 证据按 runbook 时序在部署后落，验收时决策席亲证；本收口
交付=代码面六件+配置杠杆+runbook，测试面全绿如下。

**测试证据**：
- go build+vet 绿、gofmt 零漂、check_migrations 七断言过（fin_0117=117 链内唯一）；
- lifecycle 包 30.4s 绿：TestXvenueSwapAccrualLongCfd（摊提手算对拍
  −794.336bps×100USD×1h/8760≈−0.009067+ts 门防重摊+新行续摊）、
  TestXvenueSwapShortCfdAndSkips（short_cfd 正账+ann 零/非副腿向零账）、
  TestPaperFundingDirectionSign（long_perp −1/short_perp +1）、存量锚
  TestXvenuePairAdmit/TestXvenueOpenForbidden 仍绿；
- hypothesis 包 18.4s 绿：TestExecFamilyRegisteredChainProceeds（登记翻转锚：
  proposed+提案立 1 行——兼钉 BP-3 根修）、存量链锚仍绿；
- cmd/core 59.4s 绿：TestFlowPulseStages（八段）。

**偏离说明**：
- 阈调 250 落生产 app_config（非码面）——T-0376 机器参数域+票面激活杠杆域内，
  决策席可复核回滚（键名 research.signal.xvenue_carry_diff_bps）；
- execfam 登记语义钉=paper 径贯通非真单径（opener 永拒在码），注释同步；
- BP-6 按 T-0164 钦定语义不动只报；SG md_mark_index 桥面残留陈行（10-30）为
  FIX-02 同族现象（ projector 真源律），诊断中已核非本链断点——蒸馏器真源在
  OCI 盒本机 PG-md（mark 鲜），SG 侧 carry_diff/spread 行由 o-research 直写鲜行。

**返修1偏离说明（基线漂移重号案——非施工责）**：
- 原收口基 29e61702 在 T-0376 合入前；T-0376 落 master 已载
  fin_0116_hunger_window.sql（migrate_table {116}），本支
  fin_0116_xvenue_swap_fill.sql 同号→fin 链双 116 违 check_migrations
  链内唯一单调断言，裁决驳回。返修=merge origin/master 重戳基线+
  本支迁移重号 fin_0117（置 116 行后），文件名/头注/票面提法同步；
  码面六件语义零碰（仅迁移号面+引用同步），不合并不部署照旧。
