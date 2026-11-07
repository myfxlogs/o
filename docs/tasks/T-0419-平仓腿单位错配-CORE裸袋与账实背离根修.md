# T-0419 平仓腿单位错配——CORE 裸袋与账实背离根修

- 状态：已批（业主令直批 2026-11-07——「立」）
- 领取：t/0419-close-unit-mismatch @ /opt/o-wt/t0419
- 优先级：P1——钱径执行 bug（已产 -$24.67 假亏+真裸袋）
- 收口移交归：决策席

## 背景（生产实证 2026-11-07 对拍）

fin_position #33 CORE-USDT-SWAP：20:00:25 开→20:00:40 平（15s）：
- 开现货腿 sz=24.999 **sz_unit=quote**（$24.999→qty_base=1915.63 CORE）✓
- 平现货腿 sz=24.99 **sz_unit=base**（只卖 24.99 CORE≈$0.33）✗
  ——同一 24.99 数从 quote 翻成 base，现货腿只清 1.3%
- 账记 settled_amount=-24.67（假亏——成本未回记账当损）
- 残袋实证：fin_holding `CORE spot_dust=1924.61` +
  `earn_flexible=3813.34`（残袋被 Earn 自动收息——
  资产在，**但=无对冲方向多仓，违 delta 中性章程**）
- position #34 21:11 又开新对冲对——新对是平衡的，
  但旧袋独立裸露

三处病灶（fin_guard_close 实证升级）：
①平仓腿 qty 计算错径——目标美元额被直接当 base qty 发；
②**delta_drift 守卫幻觉环**——pos33 开仓 15s 即触发
  delta_drift（两腿配平 1923 张 vs 1915 个不该漂）；
  pos34 同因 state=closing **attempts=5 循环中**——
  疑似守卫单位换算同病（拿 quote 额比 contract 数）
  或按库存全账算漂移（残袋让真对永远漂）；
③账实背离——仓标 closed 记全损，资产实留 venue 未对账。
**循环性质：开→幻觉裸→错单位平→产袋→drift更深→
再开再平→袋更厚——正在烧手续费的活循环。**

## 钦定语义

```
平仓腿:  qty必须=持仓腿实得qty_base(读腿/读持仓余额),
        禁拿开仓sz的美元额当base卖——对不上的平=错平
裸袋处置: 现有3813 CORE earn袋——归置决策:
        方案A卖袋还账(市价清,记真损)
        方案B留袋续息(明示接受方向敞口,风险上案)
        ——施工定径要DM裁,先票内探明选项
账实:    平仓后残袋必须落行(裸袋显形)——不许「closed了
        但货还在」的静默背离;settled_amount语义:
        未回的资产在账记「裸袋挂账」非「实现亏损」
```

## 范围

1. 平仓腿根修：qty=腿实得 qty_base（fin_order_leg 回读）
   或 venue 余额实查——禁止 sz 额直接当 base；
2. 裸袋显形：平仓后 venue 余额对账——非零残留→
   ops_event「裸袋」行+holding 标，禁静默；
2b. 守卫环探测：fin_guard_close `state='closing'` 且
   attempts>N∨滞留>Xh→升件（本案 pos34=5次循环
   零告警实证——探针面缺「守卫自残」枚举）；
3. settled 语义修：未回资产记裸袋挂账（ unrealized ）
   非实现损——账实一致；
4. 现有袋处置：3813 CORE earn 袋——票面呈 DM 裁 A/B
   （卖袋还账 vs 留袋担险）；
5. 平仓复发桩：unit-mismatch 场景红桩。

## 验收

- 红桩：模拟 unit-mismatch→平仓腿不再拿美元额当 base 卖；
- 裸袋闸：平仓后残留→ops_event 行+holding 标可查；
- 生产对拍：3813 CORE 袋处置案 DM 裁定落行；
- 绿戳对口+全链。

## 结果回填

- 状态：待验
- 实现摘要：
  1. **平仓腿根修**（guard/facts.go）：`closeWant` 量纲消歧——SWAP=sz
     张数；qty_base>0=venue 实填币量唯一真源；sz_unit=base=sz；quote 腿
     qty_base 缺/量纲未消歧=SzKnown false→`sizedCloseLegs` 备源 venue
     实余（钦定「读腿/读持仓余额」双径）。美元额绝不当 base 发。
  2. **残袋显形**（guard/bag.go 新件）：ExecuteClose→markClosed 间
     两真源对拍——spot 腿 venue 实余×腿账差（开腿实得−平腿实卖
     qty_base，平腿行链自 close exec）→残袋=min(实余,腿账差)；估值≥
     `guard.bare_bag_min_usd`（种子 1）→fin_position.bare_bag_* 挂账+
     ops_event「裸袋」行+fin_holding spot_dust 显形；SWAP 残仓=
     ContractPos>0→同事件+Alert 敞口升件非资产挂账。平腿行缺=
     facts 异常本腿跳判（实卖无真源不猜）。
  3. **settled 语义修**（ledger/run.go）：settled_amount=Σcashflow+
     bare_bag_usd=实现额——未回资产记裸袋挂账非实现损；无价袋挂
     币量不挂虚值（COALESCE(0) 保守）。
  4. **fin_0130 迁移**：bare_bag 三列+阈键种子+已平仓仓腿账残袋回填
     （#33 式：开腿 side='buy' qty_base−spot_sell 平腿 qty_base 对拍，
     settled 按挂账口径重算，幂等护栏）+ops_event 显形行。
  5. **坑条目** pgx-nullif-param-int-inference-truncate（fixture
     NULLIF($8,0) 整数推断截小数 1915.63→1915 实证——本条红桩期
     中抓出）+索引重建；数据流.md 守仓段补挂账口径行。
- 测试证据：红→绿四桩——`TestCloseLegUnitMismatch`（quote 腿美元额
  25→平仓量=qty_base 1915.63 非 25）；`TestCloseLegQuoteMissingQty
  FallbackAvail`（腿账不可定→venue 实余 777 非 25）；
  `TestBareBagSurfaced`（残袋 115.63≈$1.50→挂账列+裸袋行+holding 标
  三面齐；碎尘<阈不挂反向锚）；`TestSettleBareBagSuspense`（现金净
  −24.669+挂账 24.5→−0.169 实现额非全损）。check_all 全链绿戳对口
  （`check-green-fa228da483f4837cc502c9c99080fd8e51a26073`）。
- 偏离说明：`TestSizedCloseLegs` 字面量补 `SzKnown:true`（新位语义—
  —false=venue 备源径）；bag 归因 sold 取平腿 qty_base 真源行非
  avail 差分（凭据面钦定径）。
- **DM 裁呈（票面件④——3813.34 CORE earn_flexible 袋处置）**：
  - **方案A 卖袋还账**：市价清 3813 CORE→USDT（现货卖单），回冲
    #33 挂账——损真化、敞口归零、账实彻底平；代价=市价滑点+手续费
    （$25 级单滑点≈小）。
  - **方案B 留袋续息**：维持 earn_flexible 收息（residual_park 凭据+
    对拍白名单/台账已在纳——账面一致不破），明示接受方向敞口——
    风险上案：CORE 价动盈亏全担，delta 中性章程破口事实存续（量小
    但破口在）。
  - 施工中立：A/B 归 DM 裁；裁 A 则处置径=另立票市价清+挂账冲销
    （本票零 venue 写面——凭据域外不动钱径）。
