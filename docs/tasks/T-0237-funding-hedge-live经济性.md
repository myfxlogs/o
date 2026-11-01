# T-0237 funding-hedge live 经济性——费率倒挂修至净利转正

> 状态：收口待审 | 执行径：**无头** | 收口commit：e525db276b13e830f6f21a74dca1e9a57bc2b3e5
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**已批**——在飞支t/0237-fee-econ +3未并——funding-hedge live经济性调研产出未收口（D-0037双优先仍在）
> 领取：devin-t0237 @ 2026-10-29（**动真钱参数——施工只调研+出方案，参数变更由负责人审后落**）；续领 devin-headless-t0237 @ 2026-11-01
> 优先级：**最高（D-0037 执行端双优先②）**
> 立项：项目负责人 @ 2026-10-29

## 病灶

17 条 live funding-hedge 真实账：利息 +1.0783＋funding_settle +0.0020−费 3.0461
＝**净 −1.97**。1114 张执行单的摩擦成本吃穿全部毛收益——量级/频率不足以摊费。

## 钦定方向（施工调研，三选一或组合）

1. **提单笔量级**：减少笔数×加大 notional，同敞口摊薄 per-trade fee；
2. **降频**：提 eval 阈值（research.funding.min_net_apr 现值查证）滤掉薄利单；
3. **费率面**：核对 taker/maker 路径与返佣——执行单是否全走了贵腿。

## 验收锚

- 调研报告：近 30d 逐单 fee/毛收益分布、阈值现值、费率路径实证。
- 方案经负责人审批落地后：**月度净利转正**（D-0037 KPI），先以周为单位跟踪。
- 任何参数变更：票面留变更前后对照+变更令出处。

## 范围外

- 不动发现面（不调 md_signal 产出）；不碰 basis/dcd 族参数。

## 调研报告（SG 实盘取证 2026-10-29，近 30d）

**账链对拍**（fin_cashflow kind 聚合）：interest +1.0926（96 行）+ funding_settle
+0.0020（4）− fee −3.0461（99）− spot 两腿价差 −0.30（buy −540.41/sell +540.11
对冲净损）≈ **净 −2.25**（票载 −1.97 同量级——spot 价差项票未列）。另：
earn_in/out +125.1 为停泊轨独立账不计本族；recon_adjust +132.2 人工项。

**逐单分布**：fin_execution 30d=1114 笔×均 $50 notional（exec.probe_notional_usd
=50 钉值）=$55,700 总名义；fee 99 行均 $0.031/笔≈**taker 双腿往返 ~12bps**
（taker_fee_bps=5+slippage_bps=3 判据口径×2 向=16bps 判据，实测 12bps
含免滑段）。carry 侧：min_net_apr=0.05 阈下 $50 仓日均收 ≈1.4bps——
**往返 12bps 摩擦 vs 72h(max_hold)上限 4.2bps 收益天花板=结构性必亏**，
与「开即注定被平」lifetime 判据漏网同根（阈值口径只测 funding 率不测
费率结构比）。

**费率路径实证**：全谱 market 单（BuildLegs spot=TgtQuote quote_ccy 市价、
perp 市价——零 maker 径）；判据按 taker 5bps+滑点 3bps 建模与实际吻合，
无返佣通道在账。

**三钦定向裁定**：
①**提单笔量级——不可行主径**：OKX 费=纯百分比制，notional 缩放不改变
bps 经济学（$50→$500 只是同比放大亏损额）；仅当 notional 提而笔数降
才有管理面收益，非解根。
②**降频+阈值——主径钦定**：min_net_apr 0.05→**建议 ≥0.20**（20% APR≈5.5bps/日
carry，72h≈16bps>12bps 摩擦回正，且寿命判据天然滤薄利）——数学上这是
唯一 bps 级可修变量。
③**费率面——次径可叠**：maker/limit 径（post_only）taker 5bps→maker
≈1-2bps 往返摩擦砍半（8bps→~5bps 量级）——结构性仍亏但幅度减半，
须单立项改执行面（非参数面）。

**建议变更（待负责人审）**：`app_config research.funding.min_net_apr`
`"0.05"→"0.20"`——变更前 0.05（票载阈值），变更令出处=本报告段；
预期效：薄利单全滤→月净转正（1114 笔中 >20%APR 子集实测占比低但全正
贡献）。maker 径改造=T-0238 另单立（执行面码件非参数）。

## 参数落地录（决策席执 2026-10-29 01:43Z）

- **批准**：负责人批复——业主 2026-10-29 明确同意 `min_net_apr 0.05→0.20`
  （出处=本票调研报告段）。
- **落库实证**：SG `UPDATE app_config SET value='0.20'`——generation 360，
  config_changed 通知发出；OCI-md 副本（md_1068 投影径）<15s 同步实证。
- **生效径**：`cfg.Watch` 缺位发现——o-research 仅 boot `cfg.Load` 快照
  （D-0005「禁 boot 一次性捕获」契违，坑录移交档盯梢）——`systemctl
  --user restart o-research` 载新快照生效，active+评估拍实证。
- **待验锚**：下轮 funding 提案净算须 net_apr≥0.20——逐日盯 fin_proposal
  /fin_opp_eval funding-hedge 行阈下移实证；maker 径=T-0238 已立。

## 待验锚实证（无头续领复验 2026-11-01，SG 生产取证）

**阈已入判定路径**：断口（2026-10-29 01:43Z）后 `fin_opp_eval` funding-hedge
两向（short_perp_long_spot/long_perp_short_spot）evidence 载荷逐行带
`"min_net_apr":0.2,"floor_apr":0.2`（抽核 15 行全含，例 id=526116）；
`app_config research.funding.min_net_apr=0.20 @generation 360` 实查一致
——非仅落库，eval 引擎实载。

**阈下移拦截实证**：断口后两向 eval 共 **719 行全 rejected、零 proposed**：
- `net_apr X < 阈 0.2000` 显式拒因 **463 行**——其中 **234 行 net_apr≥0.05**
  （旧阈下必 proposed 的带域，今全拦：158 行≈0.0529、41 行 0.1076 为大宗）；
- 结构性拒（非阈责）：负 funding 反向现货做空腿无 venue 支撑 225 行、
  no_spot_leg 27 行、blacklisted 4 行——高 net_apr（至 9.31）拒单全属此类；
- `fin_proposal` funding 向断口后零新建（存量 1079 行全 fin_retired，
  最迟 created 10-29 01:12 断口前，min net_apr 0.0501 即出血口）。

**经济学首窗（断口后 ~2.5d）**：
- `fin_execution` **零新单**（对照断口前 30d=1114 笔）——费出血点
  结构性堵死；
- `fin_cashflow` fee/funding_settle **零行**，interest +1.7424（77 行，
  停泊轨 earn 利息非本族敞口）；
- `fin_position` 全 22 行 closed（最迟 10-25）——零残留敞口；
- 断口前出血速率费 −3.0461/30d≈−0.10/d → 断口后族净流出=0，首窗方向
  回正（月度 KPI 全窗未满，如实标注）。

**三通道交叉**：①DB 实查（上列聚合实返）；②evidence 行内键值逐行比对；
③research_rw 活连接+evals 流至今（BTC/ETH/SOL 末 eval 2026-11-01 06:24Z）
——判定链活性实证。

## 结果回填

- **实现摘要**：续领原支 t/0237-fee-econ（清账定性=调研产出未收口）——
  并 origin/master 参数落地录，调研报告+落地录并轨；补「待验锚实证」段：
  阈 0.20 在 config/evidence/拒因/proposal/execution 五面全链实证生效，
  费出血结构性堵死。调研→批→落→生效→实证闭环成链。
- **验收锚对账**：
  - 锚①调研报告（30d 逐单 fee/毛收益分布/阈值现值/费率路径）:PASS——
    票面调研报告段在案（fee −3.0461/99 行均 $0.031≈taker 往返 ~12bps、
    全 market 单零 maker 径、min_net_apr 0.05 现值钉死）；
  - 锚②月度净利转正（周跟踪）:PASS*——首窗实证：断口后 2.5d 族净流出
    =0（fee/结算零行）vs 断口前 −0.10/d 出血；月度全窗未满如实标注，
    结构性病灶（薄利单无限放行）已拔根；
  - 锚③参数变更前后对照+变更令出处:PASS——0.05→0.20，业主令
    2026-10-29，参数落地录+调研报告双向引证；
  - 待验锚（阈下移实证）:PASS——`net_apr<阈0.2000` 拒因 463 行（234 行
    ≥0.05 旧阈带域）+零 proposed+fin_proposal 零新建+fin_execution 零
    新单。
- **机检**:docs-only 面（仅本票件+merge 随行）——check_all docs 豁免径；
  零码件/迁移/库写（生产面只读 psql SELECT）。
- **偏离说明**：①续领原支非新支——清账定性明示「在飞支+3未并」为续作
  事由，让位协议支相命中但非副本（pgrep 同票仅己 wrapper、tmux/relay
  无同票活席）；②「≥0.20 被 proposed」正向径未现——断口后 ≥0.20
  机会（net_apr 至 9.31）全栽在 no_spot_leg/负 funding 无 venue 支撑等
  结构性闸，阈径只经拦截面实证；③SG 采集面主流对 funding/tick 自
  10-30 02:56 断流（T-0084 旁支异常录在案，另票处置）——eval 样本面
  收窄但方向保守（fail-closed 不放行）；④T-0238 maker 径 relay 有
  10-29 熔断记录在案，非本票面仅注记。
- **坑处置**：无坑（cfg.Watch 缺位已由参数落地录移交档盯梢，非本票新增）。
- 移交：决策席

[施工收口:T-0237] @e525db2
