# T-0203 carry_diff 方向腿口径伪影——1095bps 是拿错腿的虚高

> 状态：待施工
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
