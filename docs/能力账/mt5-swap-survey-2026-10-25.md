# MT5 swap/carry 全集调查（T-0099，EXR 3001157 真户实采）

> 采集：tools/swapsurvey 2026-10-25 ~12:44Z，SymbolList→SymbolParams→GetQuote→RequiredMargin
> 逐符号实拉（366/366 零失）；原始 JSON=`mt5-swap-survey-2026-10-25.json`。
> 户头：login=3001157、leverage=2000、ccy=USD、equity=freeMargin=balance=$3247.59（空仓）。

## 结论：CFD swap-carry **无货**

- 366 符号 swap 三值分布：**SwapNone 112 件**（不记隔夜费，双向皆 0）+
  **InPoints 254 件**（点数型收费，**swap_long/swap_short 全部 ≤0**）。
- 全库无一件「持仓被付息」方向——Exness 双向皆收或皆免，carry 收割模板原料不存在。
- 最贴近零的收费行仍负：XAGGBPm L=-4.7pt、CHFSEKm L=-100.5pt、NOKJPYm S=-14.1pt、
  USDDKKm S=-43pt、US30_x10m L=-105.1pt——正收益侧无。
- 123 件 InPoints 行有一侧为 0=「免费持仓」非「付息持仓」——carry 收入型模板不可用，
  但**零成本腿**可作跨 venue 价差的持仓腿候选（备查，不在本单激活）。

## 保证金分层（$3247.59 权益对照）

| margin_1lot 档 | 件数 |
|---|---|
| ≤$100 | 192 |
| ≤$500 | 75 |
| ≤$1623（权益 50%） | 72 |
| >$1623 | 26 |
| 读缺 | 1 |

- 名义额小/保证金低件密集（crypto USD 对 margin 一件 ~$5–50）——短线/价差型
  模板保证金面不构成约束；约束在**收入型缺失**非资金量。
- 杠杆 2000=户头上限非策略建议；保证金试算列=venue 实返值。

## 模板裁决（走票钦定口径，不硬造单）

- **MT5 swap carry 模板**：激活否决——全品种正 swap 方向缺失（步骤②候选池空集）。
- **后续候选（本单范围外报备）**：贵金属短线（点差面可查 spread_pts）、
  跨 venue 价差（OKX↔MT5 同标的映射——BTCUSDm margin=$40.4/lot、ETHUSDm=$1.18/lot
  且 short 腿 swap=0=零成本持仓腿）、SwapNone 品种趋势型——另立评估单。
- 闲资 $3247 承接缺口仍开——CFD 内无 carry 货则回报决策席另议部署面。
