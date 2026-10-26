# T-0144 跨所 carry 假想→评估→纸面全链（xvenue_carry 模板首立）

## 态：施工中
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
