# T-0203 carry_diff 方向腿口径伪影——1095bps 是拿错腿的虚高

> 状态：作废（superseded——11-01 清账定性·SETTLE-SWEEP）
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**作废**——superseded：carry_diff方向腿伪影修交付并master（t/0203-carrydir merge在案）——验收债归T-0249
> 域：信号面/经济学口径（D-0034 证伪锚命中）
> 源出：决策席 2026-10-27 实拍出 mt5_swap meta 双腿后定性

## 病灶实证

`carry_diff` = `funding_ann − swap_ann`，swap_ann 取自
mt5_swap.val=0——**该值是「空腿/short 腿」的 swap**（meta:
swap_short_ann_bps=0）。

但经济学对拍：funding>0 时收割方向=**做空 OKX perp 收 funding
(+1095bps) + 做多 MT5 CFD 对冲**，CFD 多头付的是 **swap_long**
（BTCUSDm 实证 -1786.5→年化 **-816bps**）——真净 carry=
**+279bps≈2.8%/yr**，非票面 1095bps≈10.95%。

信号把「与该方向无关的另一条腿」当了成本——**方向腿错配伪影**。
反方向（funding<0 收割）同理：long perp 付 funding+short CFD
付 swap_short=0→净=-|funding|−0。

## 钦定修法

carry_diff 按**方向自洽**产出（二选一，施工可裁）：

- **甲（推荐）**：报「当前最优方向净 carry」——funding>0 时
  val=funding_ann+swap_long_ann（swap_long 为负=成本，相加即净）；
  funding<0 时 val=|funding_ann|+swap_short_ann；meta 钉 direction
  字段（short_perp_long_cfd / long_perp_short_cfd）+双腿原值随行。
- **乙**：拆两行（carry_diff_longcfd/carry_diff_shortcfd）各报
  各向净值——消费面改双键读。

钦定下限：消费面（评估/T-0145 亮牌判）拿到的是**经济学可执行方向**
的净 carry，不是错腿虚高；meta 须携双腿 ann+方向钉死。

## 验收判据

- BTC 当前输入下 val≈+279bps（1095−816）非 1095
- funding 翻负时 direction 翻转、val 符号正确
- 桩锚：swap_long≠swap_short 输入下 val 用对腿（两方向各一测）
- 行史重算后 T-0145 阈判定基线归位（1095 伪影不得进亮牌判据）

## 结果回填

- 甲案收：funding>0→short_perp_long_cfd（val=funding_ann+swap_long_ann）；
  funding<0→long_perp_short_cfd（val=|funding_ann|+swap_short_ann）；
  meta 钉 direction+双腿 ann 原值（f64any 缺件=0 不臆造）。
- 桩锚：TestXvenue 修判据 funding<0→净=|1095|−50=1045 实证绿；
  funding>0 对称径同构。
- T-0145 亮牌判据归位：1095 伪影不再进亮牌——BTC 实净≈279bps 口径
  （spec 钉证属决策席既有取证面）。

## 决策席验收 2026-10-27——销号投产

- 码面核：funding>0→`fundingAnn+sl`(swap_long_ann 负值相加即净)、
  funding<0→`-fundingAnn+ss`——方向腿自洽对钦定；swapMeta 查询置
  双腿取值前无空 map 坑；meta 钉 direction+双腿 ann+透传件。
- 桩：9/10 绿（独败=TestBasisAnn 基线先验，另案记档）。
- **活件实证**：部署后 carry_diff 重写——BTC `val=281.21bps`
  （funding_ann 1095+swap_long_ann −813.79）、ETH 268.35bps；
  meta.direction=`short_perp_long_cfd`、ts 随拍鲜。1095bps 伪影
  死、实净≈2.8%/yr 口径归位——T-0145 亮牌阈（≥8%）未达，候场
  票维持不升派。
