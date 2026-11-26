---
id: earn-sweep-intent-exempt-purpose-money
title: Earn 停泊环「闲钱」判据不识用途——赎回款/负债担保/在途预备金被秒扫（O-40829）
kind: pit
date: 2026-11-26
source: 工单O-40829 P0 钱径正确性缺陷——margin-short 物理前提+所有「Earn抽钱干活」径通用病灶
---

# 「非白名单内钱的都闲」≠「闲钱」（停泊环意向豁免）

**病灶**：`sweepIdle` 双段（funding 直购+trading 摆渡申购）判据只看
闲余金额>阈——不识「钱有用途」。实证三案：
①Earn 赎 1000 USDT→funding 到账→**1 秒被 depositIdleFunding 再申购**
（asset/bills 双腿 76→75 自杀环在案）；
②margin 借币仓开着时 trading USDT 抵押物+卖币所得=sweep 眼里闲钱→
扫走→adjEq 崩→venue 强平（margin-short 活不过几分钟）；
③T-0462 只修了 funding 段对 dcd 摆渡款让位，「有用途」概念散在个例
没成通则。

**修法（判据源复合，单函数一源入口罩双段）**：
- `fin_risk_decision op='earn_redeem' <15min`（venue 域）——簿记未落行
  窗的唯一信号，覆盖 redeem→book 序间隙；
- `fin_position.state<>'closed'` ∧ 执行腿 `venue=本户`——**venue 域限定
  是硬条款**：MT5/DCD 仓抵押物不在 okx 户，不限定=他 venue 仓永停本户
  停泊环；
- `fin_margin_loan status IN ('open','closing')` venue 域——簿记直证；
- **venue `/account/balance` details liab/crossLiab/isoLiab>0 ∨
  cashBal<0`——权禁线钦定权威源**：venue 有负债账上没有=担保钱
  （手动借/他径借簿记零行全覆盖），禁只看 fin_position 表；
- 任一判据读败=fail-closed 豁免（不虚构「没有用途」）；让位原因落 log
  +`sweep_intent_exempt` 计数门（让位是判词不是静默吞）。

**反向坑（勿过度豁免）**：白名单闸（FundCcy）对 USDT 本位扫面**不可用**
——USDT 恒在 FundCcy 三域在册，查了=永豁免=停泊环死；币级白名单归
sweepResidual 既有径，别混进 sweepIdle。

**回滚**：`earn.sweep_intent_exempt=false` 旧行为全复；
`earn.sweep_liab_check=false` 单降 venue 源（余判据照走）。

**推广**：「钱可动性」判据凡扫/归集/搬家的面——先问「这钱有没有
用途在路上」，判据源优先级=venue 真值>在途簿记>持仓簿记；读败方向
永远向「不动」不向「动」。
