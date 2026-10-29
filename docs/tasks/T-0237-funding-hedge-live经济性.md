# T-0237 funding-hedge live 经济性——费率倒挂修至净利转正

> 状态：待验 | 领取：devin-t0237 @ 2026-10-29（**动真钱参数——施工只调研+出方案，参数变更由负责人审后落**）
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
贡献）。maker 径改造=T-xxxx 另单立（执行面码件非参数）。
