# T-0144 跨所 carry 假想→评估→纸面全链（xvenue_carry 模板首立）

## 态：驳回:xvenue-carry未入paperTemplates白名单→admitProposals门闸(run.go:143)先于双腿支路(:147)拦下,admitXvenuePair生产死径,锚②proposal→双腿策略行与锚④p_and_l/funding+swap双账入账永不可达(生产零行实证,单测直调绕闸);重飞=白名单登记(或拆死支路)+proposal→scan→双腿链级桩测
## 审计席：headless @ 2026-10-30

## 判决注记（无头审计逐锚对账——证据=收口链 2ecd0e8+c934134+811ceed/e2f34dd
##   全量 diff + SG fin 库只读实拍 2026-10-30 + 现码 HEAD 复核；T-0124 审计
##   注记 7e0759c 已独立记同款死径——非审计席孤证）

- 锚①（ETH域负funding卡 或 rejected+净年化负值如实行——无机会如实也
  是过）：**过**。生产 fin_opp_eval template='xvenue-carry' 实拍 3 行全
  rejected 如实：BTC short_perp_long_cfd net_ann_economics 净−195.6bps
  （10-26）、ETH 同径 −1188.9bps（10-30 续拍——ev_funding 事件臂仍活）、
  BTC long_perp_short_cfd +433.3bps mt5_leg_capability_missing（能力闸
  如实拒——「看见了做不了」显形）。「或」支成立。
- 锚②（proposal→两腿策略行 venue各正/proposal_ref同引）：**未达——
  驳回**。admitXvenuePair 机制在（主行 venue='okx'+副行 venue='mt5'
  proposal_ref 同引，xvenuepair.go:48-67；TestXvenuePairAdmit 桩绿），
  但 paperTemplates 白名单（run.go:110-112=funding-hedge/dcd_call/
  vol_sell_*/quarter_basis_*/perp_quarter_spread 六件）自收口即不含
  xvenue-carry——门闸 run.go:143 `!paperTemplates[x.tmpl]` 先于双腿
  支路 :147 拦下，支路生产死径（收口 commit e2f34dd 与 HEAD 同形实核，
  非后续回退）。生产 fin_strategy template='xvenue-carry' **零行**。
  桩测直调内函绕闸——真链 proposal(fin_proposed)→scan→双腿行零覆盖，
  「单测绿≠径可达」教科书案。
- 锚③（评估 evidence 携四件分解）：**过**。生产 evidence 键实拍含
  funding_ann_bps/swap_ann_bps/swap_long_ann_bps/spread_bps/
  spread_amort_bps/slippage_buffer_bps/net_ann_bps/capacity_usd——
  四件分解+容量/方向随行，与票面钦定相符。
- 锚④（paper双腿撮合产 p_and_l 行+funding/swap 双账各入列）：**未达
  ——驳回**。生产 fin_paper_fill×xvenue-carry 策略 **零行**、
  swap_snapshot 全表零行——双腿策略行不生（锚②同根），撮合/fill
  无从产。swap_snapshot/p_and_l 机制写在案但属死径内件。
- 锚⑤（armed→open 闸对模板显式拒）：**过**。opener.go:81-86
  xvenue-carry→xvenue_open_forbidden 拒计数+ALERT 日志钉死；拒因门
  run.go:36 预注册挂载一致；TestXvenueOpenForbidden 断言拒计数+1/
  stub Opener 零调用/零持仓行——钉死三证齐。
- 重飞补（不代施工设计，仅指径）：①paperTemplates 登记 xvenue-carry
  或拆死支路——取舍裁归决策/管理席（T-0124 审计注记同款建议另票）；
  ②补链级桩测：fin_proposal 提案行→admitProposals→双腿策略行落，
  让门闸与支路在真链上互证而非旁路直调。
- 附：假想/评估/禁线面（锚①③⑤所涉链环）与收口 diff 相符无欺；
  驳回严格限纸面撮合径死码，非全链否定。

## 领取：zcode-t0144（ZCode 施工席）@ 2026-10-26
## 域：发现链 / 假想模板 / 评估 / 纸面撮合
## 源出：T-0141+T-0143 信号面投产的消费件——carry_diff/xvenue_spread 有燃料无引擎

## 方向锚
D-0022 桥二期终态件。第一性跨所生意模板：**OKX perp × MT5 CFD 对冲 carry**。
负 funding 径（正主）：funding 砸负→空付多→长 OKX perp 收 funding + 空 MT5 CFD
对冲（实锤：BTCUSDm/ETHUSDm swap_short=0 空腿零成本——T-0143 验收在案）。
反向（funding 正大）：空 OKX perp 收 + 长 MT5 CFD——swap_long 成本须入账
（BTCUSDm swap_long=-1786.5pt≈-816bps/yr——通常不划算但评估器如实算）。

## 建模决策（票面钦定）
- fin_strategy/proposal 现行单 venue+inst_id——跨所双腿用**主从策略对**：
  主行 venue='okx' inst=perp；副行 venue='mt5' inst=CFD（BTCUSDm→BTC-USDT-SWAP
  映射走 app_config bridge.xvenue.map 反查——同映射源不另造）。
  两腿对组键=proposal_ref 同引（同提案生双腿）；p_and_l 聚合在提案级。
- 纸面撮合双腿各自市价模拟（OKX 腿吃 md_mark_index、MT5 腿吃 md_ticks
  venue='mt5' mid——双边行情均在库）；funding 入账走资金费结算节拍语义，
  swap 入账按 mt5_swap 信号快照（拍时锁 meta raw 进 evidence 防回改）。

## 链环
1. **假想层**：hypothesis 臂新增 xvenue_carry surface——消费
   carry_diff(|val|≥阈亮牌——**大负值=负 funding 对冲机会**，票面钦定语义钉
   见 T-0143 验收档）+xvenue_spread（进出基差成本入账）。
2. **评估层**：净年化 = −funding_ann − swap_leg_ann − 双边点差摊销 −
   两腿滑点缓冲；容量 = min(OKX depth5 可吸、MT5 contract_size×最大手数、
   MT5 RequiredMargin 实算可承)；verdict 双腿能力账双闸（OKX 腿过
   venue_untradeable 黑名单滤 + MT5 腿过 trade-order 面在册——T-0141 已登记）。
3. **纸面层**：paper 策略双腿虚拟开仓——风控照常（聚合敞口+回撤归因）。
4. **禁线**：本票**零真执行**——提案至 fin_armed 止步（armed→open 闸对
   xvenue_carry 模板显式禁：执行原子性语义未建，抢跑=裸腿风险）。
   真执行=T-0145 另票（两腿序+败一腿处置+venue_untradeable 联动）。

## 验收锚
- ETH 域负 funding 态下 xvenue_carry 假想卡落（或评测器对当下
  carry_diff=+532bps 给出 rejected+净年化负值如实证据行——无机会如实也是过）
- 双腿建模：proposal→两腿策略行，venue 各正、proposal_ref 同引
- 评估 evidence 携四件分解：funding_ann/swap_ann/spread_bps/净年化
- paper 双腿撮合产出 p_and_l 行，funding+swap 双账各入列
- armed→open 闸对模板显式拒（实测验码/日志钉死）

## skills 域
mt5-trading-mutating（腿能力事实源）+mt5-quotes-history（SymbolParams/
contract_size 语义）+okx-exec（腿口径对照）。

## 结果回填（zcode-t0144 @ 2026-10-26）

- **①假想层**：hypothesis 信号臂 xvenueCarryScan 支路——carry_diff |val|≥阈
  （research.signal.xvenue_carry_diff_bps=300，0060 种）亮牌→xvenue_carry 卡
  （surface='xvenue-carry'，legVocab 闭集扩 {"okx_perp_long","okx_perp_short",
  "mt5_cfd_short","mt5_cfd_long"} 随票非运行时；evidence 携
  provenance=hypothesis_officer+signals 源信号+direction 语义钉：大负 diff=
  long_perp_short_cfd 正主/大正=反向）；UNIQUE(venue,surface) 幂等。
- **②评估层**：xvenueCarryEval——净年化=−carry_diff−spread 摊销（|spread|×2
  按持有窗 72h 热启年化占比）−滑点缓冲（taker×2 腿）；evidence 四件分解
  （funding_ann_bps/swap_ann_bps/spread_bps/net_ann_bps+capacity_usd）；容量
  三件=min(okx depth5 顶档×0.5 可吸, mt5 contract_size×max_volume 名义,
  initial_margin 率随 meta)；verdict 双闸=okx 腿 fin_venue_untradeable 滤+
  MT5 腿 trade-order 面在册（master 能力账无执行面→mt5_leg_capability_
  missing 如实拒——「看见了但做不了」显形）；净负→net_ann_negative 如实。
  proposed verdict→fin_proposal 立行（fin_proposed 止）。
- **③纸面层**：admitProposals 模板门闸 xvenue-carry 分支→admitXvenuePair
  主从策略对：主行 venue='okx' inst=perp（funding 端，走既有 paperAccounting
  md_funding 记账）+副行 venue='mt5' inst=CFD（bridge.xvenue.map 反查同映射
  源；向取反 short_cfd/long_cfd）；两腿 proposal_ref 同引（p_and_l 提案级
  聚合）；副行建行时锁 mt5_swap meta 快照→fin_paper_fill kind=
  'swap_snapshot'（防回改——票面钦定）；幂等=双腿恒 2 行。
- **④armed→open 显式禁**：openLive 候选带 template_ref，xvenue-carry→
  xvenue_open_forbidden 拒计数+ALERT 日志钉死+stub Opener 零调用断言——
  零真执行（禁线守）。
- **配套**：0060 迁移（xvenue_carry_diff_bps 阈种+fin_venue_untradeable
  幂等保底〔0059 同构——T-0142 未合 master 的自洽面，合并序无冲突；登记
  {60}，59 位留 T-0142〕+direction 枚举扩 long_perp_short_cfd/
  short_perp_long_cfd〔fin_opp_eval+fin_proposal〕+fin_paper_fill kind 扩
  swap_snapshot）；bridge swapLoop meta 扩容量件（max_volume/initial_margin
  随 mt5_swap 信号携行）。
- **桩测**（PG 实跑全绿）：TestXvenueCarryChain（-532bps 亮牌→卡
  provenance/signals/direction 三断+评估四件+capability_missing 如实拒+
  阈内 BTC 零卡）；TestXvenuePairAdmit（双腿 venue 各正/proposal_ref 同引/
  向取反/armed 翻面/swap 快照/幂等恒 2 行）；TestXvenueOpenForbidden
  （显式拒计数+Opener 零调用+零开仓行）；lifecycle 全包回归绿。
- **机检**：build✓ vet✓ gofmt✓ staticcheck✓ check-lines✓ import 闸 101/108✓
  migration 闸✓（0060 登记；0059 位留 T-0142）。
- **移交面（部署后决策席实证）**：ETH 域负 funding 态下 xvenue_carry 卡/
  评估行（当下 carry_diff 实值驱动——无机会如实也是过）；armed→open 禁
  日志钉死；双腿行 venue 各正实拍。

## 返修回填（zcode-t0144 @ 2026-10-26——正 funding 径双腿经济学归正）

- **①bridge.go 双腿年化**：SwapAnnualBoth（swap_long/short 各经同换算核
  swapAnnualSide——InPoints 正典式/PercCurPrice 分腿字段各算；任一腿不可
  年化=ok false 双端须齐）；mt5_swap meta 携 swap_long_ann_bps/
  swap_short_ann_bps/ann_ok。桩锚 TestSwapAnnualBoth（swap_long=-1786.5pts
  代入 mid=79842→long_ann≈-816bps 精断+short=0 空腿零成本实拍形）。
- **②sigdist carry_diff meta 增携**：双腿年化+容量件
  （swap_long_ann_bps/swap_short_ann_bps/ann_ok/contract_size/max_volume/
  initial_margin）全随行；val 语义不动（仍 funding−short 端）。
- **③xcarry 分径**：xvenueCarryNet 方向感知——diff<0 长perp径
  net=−diff−成本（现行对）；diff>0 短perp径 net=+fundingAnn−|swap_long_ann|
  −成本（swap_long 取 meta.swap_long_ann_bps 非 val）。
- **④方向感知阈**：双向各自 |net|≥th 才起草（正向算不出净正 card 不立）；
  net<0（经济学不通）→评估 rejected/net_ann_economics 如实不撞哑+零卡；
  双腿年化源缺（桥旧版 meta）=不可判 false 如实。
- **⑤清**：xcarry.go 重写去文件尾 var _ 引面锚+无用 import；max_volume 段
  确认 SymGroup 实有域携入非死枝（容量三件 meta 源在案）。
- **桩锚**（返修单三件全绿）：TestSwapAnnualBoth（long_ann≈−816bps）；
  TestXvenueCarryPositiveDiff 幕1（diff=532+funding=200<816→rejected/
  net_ann_economics+零卡不撞哑）幕2（funding=2000>816→卡立+direction=
  short_perp_long_cfd 双腿向反）；hypothesis/mt5/lifecycle/sigdist 全包回归绿。
- **验收判据不变**：mt5_swap.val 非 NULL+carry_diff 落行（部署后实证）；
  新增：正 funding 态下 economics 拒/反向 proposed 双态实证。

## 验收档（决策席 2026-10-26，e2f34dd 合入毕投产）

**正径经济学实证（活判据）**：部署后 `fin_opp_eval template='xvenue-carry'` 落行——
BTC `short_perp_long_cfd` rejected/net_ann_economics 净 −35bps（funding +697bps
vs MT5 长腿 swap −814bps，成本吃光如实拒）；ETH 同径 −299bps rejected。
双腿 swap meta（swap_long_ann_bps/swap_short_ann_bps/max_volume/initial_margin）
全链随行实证。**正 funding 径不再撞哑——方向感知经济学成立**。

**验收窗自捕根修（run.go 双听饥饿）**：验证中抓出 research-hypo-sig 自
T-0140 起零跑——首听同步阻塞占死第二听。首轮返修误把 go 加在死代码行
（病灶原封部署），死臂实证闸（pg_stat_activity LISTEN 计数+评估行复跑）
二次抓出。真修=双听皆 go（811ceed）→部署后 LISTEN ev_funding 2→3 实增、
xvenue-carry 评估行落地、臂日志如实拒——**死臂真活三证齐**。
KB 条目 dual-listenloop-serial-starvation 两度回写（坑加深实证）。

**附带清源**：internal/gen 软链误入库（gitignore 尾斜杠罩不住软链）——
untrack+三处 ignore 洞同堵+buf 真件重生成。

**衍生票**：T-0147（终局分类补面——SHEIN 286078 腿构meta径漏分类+
黑名单存量回扫实证）、T-0148（okxws depth5_no_ctval 门键漏注册洪泛）。

## 态：验收毕（投产）2026-10-26
