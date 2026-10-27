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

## 结果回填（施工席取证 2026-10-27）

- MT5 swap 信号链实活：mt5_swap BTCUSDm/ETHUSDm val=0、swap_type=1、
  ts=10-27 13:33 鲜——链取径正常（非死数据）。
- **venue spec 原值未对拍**：本席会话内 MT5 venue spec 读径
  （swap_long/swap_short 原值）未取——凭据面在 SG（readcred 闸内），
  「免息实证」未能钉；swap=0 是「信号链报 0」而非「venue 证实 0」
  ——此为本票最大未决锚，判伪风险即在此层。
- 方向/年化/单位三审：carry_diff=1095bps=funding_ann(0.0001×3×365)
  −swap_ann(0)——funding 单边主导；若 venue spec 免息实证则亮牌真
  （OKX short perp 收 10.95% 年化），若非则「取错字段/换面错」立案。
- 行史分布：今日仅 2 行（当值期判据下稀疏再写=预期——T-0199 语义）；
  「持续亮牌」窗拍数实证不足（2 行难钉持续阈）。
- 定性：**亮牌存疑待 venue spec 钉**——方向/换算/单位自洽无伪；
  唯一未钉=MT5 侧免息真实性。建议决策席派「MT5 spec swap_long/
  swap_short 原值实拍」小票（readcred 取凭→venue 读面）定夺 T-0145。
