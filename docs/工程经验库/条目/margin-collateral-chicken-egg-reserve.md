---
id: margin-collateral-chicken-egg-reserve
title: margin 借币额度鸡生蛋死锁——抵押金被 sweep 扫光 eval 恒拒额度不足（O-41364）
kind: pit
date: 2026-11-26
source: 工单O-41364 P1 钱径——O-41363 修通采集臂后实证 max_loan≈0 全表尘级
---

# 「额度可证」需要抵押物在场——扫光抵押金=能力面结构性空转

**病灶**：OKX cross 借币额度=抵押物比例制——`max-loan` 对
`adjEq≈$0.003`（sweep 按设计把钱全扫进 funding/Earn）的 trading 户
全币返回尘级（BTC 0.000000177≈$0.015）。于是：
无 armed margin 提案→无意向登记→O-40829 意向豁免四类源全属**已承诺态**
（在途赎回/在持仓/在借负债/venue负债——无一条盖「待评估抵押金」）
→sweep 照扫→adjEq≈0→额度≈0→`borrowQuote` $500 下限恒拒
`borrow_quota_insufficient`→提案永不 armed。**每一环都实证，环环咬死。**

**修法（钦定——venue 额度唯有抵押物在场才可证，无「假想抵押」API）**：
- 新键 `margin.collateral_reserve_usd`（缺省 $500——probe$50 级量测
  足矣；额度随抵押物线性，实测值低估满额能力）；
- **判据⑥入意向豁免复合源**：`margin.exec_enabled=true ∧ trading 可用
  <reserve`→让位（原因 `collateral_reserve`）；exec=false→失效不豁免
  （休眠不留钱——回滚径=闸本身）；reserve 键缺=fail-closed 豁免
  （宁停泊不误扫同缺省律）；
- **截量非全豁免**：`sweepable=idle−floor`——avail≥floor 时超部照扫
  不误（禁无限豁免——全豁免=扫光→泊资→再扫光死循环）；
- **泊资径=既有臂复用**：`collateralReserveTopup`→`RedeemToTrading`
  （sweep 反向=redeem——禁另造划转件）摆回短额，15min 冷却窗防
  读面滞后/连败锤 venue；scan 拍 sweepIdle 前置（钱先到判据才有据）。

**坑①（复合判据降级早退=后续判据死码）**：④venue 负债源
`w.liab==nil` 时原写法 `return "",false` 早退——末位判据无碍，
**插到它后面的判据⑥在测试面/源缺席径整段死码**（桩测实证：
funding 段照跑 purchases=[500]）。降级语义=「跳过本判据」非「判全
不豁免」——复合源扩位时必须把早退改成条件包裹。

**坑②（「可用」选源）**：reserve 比对用 `w.idle`（USDT availBal
≈USD）非 adjEq——sweep 只动计价币 USDT 本位，availBal 恰=可被
扫走的量；adjEq 含冻结仓位保证金会高估可动抵押物。

**回滚**：`margin.exec_enabled=false`→reserve 判据+泊资双双失效
（休眠不留钱）；`margin.collateral_reserve_usd` 调大=更多停泊
（机会成本=Earn 收益率×差额，spec 规模界钉 ~$0.1-0.3/天@500）。

**验收机械待验**：部署后 exec_enabled 开闸→trading 可用≥reserve
停扫→泊资首拍 Earn→trading 402 级摆回→md_margin_borrow 锚币行
max_loan 非尘量级→负费率提案从 borrow_quota_insufficient 推进到
净额判词。
