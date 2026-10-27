# T-0201 carry regime 亮牌核查——carry_diff 1095bps 真实性对拍

> 状态：待施工
> 域：评估面/venue 对拍（D-0034 行动②）
> 性质：取证+定性，禁改码；**先证后派**——若实证亮牌则 T-0145 升派

## 病灶/疑点（决策席 2026-10-27 实拍）

`carry_diff` 现报 BTC/ETH **1095bps≈10.95%净年化**——
= funding_ann(0.0001×3×365×10000=1095bps) − mt5_swap_ann(0)。
两疑问必须先钉死，否则「亮牌」可能是口径伪影：

1. **MT5 swap=0 真实性**：mt5_swap 行 BTCUSDm/ETHUSDm val=0
   （swap_type=1）——venue 侧该标的 swap_long/swap_short 是否真的
   免息？对拍 MT5 contract spec（symbol properties 面：
   swap_mode/swap_long/swap_short 原值）——零是真零还是读径
   没取到值（val=0 与「字段缺席」必须区分）。
2. **funding 侧符号方向**：OKX funding_rate>0 时多头付空头收——
   carry_diff 表达式（funding−short 端）的符号/方向经济学解释
   复核一遍（1095bps 是「做空 OKX perp 收 funding+做多 MT5 免 swap」
   的净 carry，对否）。

## 施工面

1. MT5 contract spec 实拍：BTCUSDm/ETHUSDm 的 swap_long/swap_short/
   swap_mode 原值（readcred 取凭据后 venue 读面）——若 spec 本就 0
   →免息实证；若非 0→信号计算链取错字段立案。
2. funding 方向核对：xvenue.go carry_diff 表达式 vs 双腿经济学
   （OKX short perp 收 funding vs MT5 long CFD 付 swap——方向、
   年化换算、单位三审）。
3. **持续亮牌判**：carry_diff 行史分布（近 7d val 序列）——
   是稳定 >8% 还是噪声尖峰；T-0145 候场阈「净年化 ≥8% 持续」的
   持续窗拍数实证。

## 产出要求

- 结论行：亮牌真/伪+证据指针
- 若伪：坑位定性（哪个环节出伪——spec 读径/换算/方向）
- 禁动生产语义/代码/配置

## 验收判据

- venue spec 原值实拍入票
- 方向/年化/单位三审结论在案
- carry_diff 7d 分布钉值在案

## 决策席验收（2026-10-27 深夜）——亮牌判伪收口

施工三审+我自拍的 mt5_swap meta 双腿合判：
- spec 面：链报 swap_long=-1786.5/swap_short=0 是 venue 原值级
  （meta 直出 venue 读件）——「免息」只在空腿侧为真。
- **判伪**：正 funding 收割向=short perp+long CFD，成本腿=
  swap_long(-816bps)——carry_diff 用 swap_short(0) 虚高了 816bps。
  真净≈+279bps<8% 阈——**T-0145 候场阈未达，不升派**。
- 口径伪影根治转 T-0203（方向腿自洽化）。销号。
