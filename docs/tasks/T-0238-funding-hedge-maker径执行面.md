# T-0238 funding-hedge maker 径执行面——post_only 限价双腿摩擦砍半

> 状态：待施工 | 优先级：P1（D-0037 主线条——T-0237 调研裁定的次径可叠件）
> 立项：决策席 @ 2026-10-29（T-0237 调研报告③裁定的执行面码件）
> **动真钱径——施工产出=实现+桩测+灰度方案，放量由负责人审后落**

## 病灶

T-0237 调研实证：funding-hedge 全谱 market 单（BuildLegs spot=TgtQuote
quote_ccy 市价、perp 市价——零 maker 径），双腿往返 ~12bps 摩擦
（taker 5+滑点 3 判据口径×2 向）。阈值提至 0.20 后 16.4bps carry 仅
**边际 +4bps**——maker 径（post_only ≈1-2bps 返佣面）可将往返摩擦砍至
~5-6bps，同等信号源下边际扩至 +10bps 量级。次径可叠非替代。

## 钦定界

1. **maker 优先双腿**：spot 买/perp 空两腿先 post_only limit（挂盘口
   优价）；**成不=>时限降级 taker**（时限判据施工钦定，防挂死敞口——
   单腿已成另腿未成=对冲裸奔窗，时限+降级+取消逻辑须成对）。
2. **fee 模型同步**：research.funding.taker_fee_bps 判据面加 maker 费率
   键（post_only 实费=负返佣/低费——OKX 费率表实测值回填判据）；eval
   净算分两腿费制（maker 腿×maker 费+降级腿×taker 费）。
3. **probe 径接入**：exec.probe_notional_usd 框架内先小仓 maker 单
   实证成交率/降级率——放量判据=probe 实证数据供负责人审；
   **放量受 D-0038 资金政策阶梯束**（KPI 转正前敞口 10% 硬顶，禁以
   设计面预估收益扩口）。
4. **歧态势**：post_only 挂单超时未成=取消重写不追价（禁 maker→改价
   轮挂死循环）；已成单腿裸奔>时限=另腿市价补或平已腿——选型回填。

## 验收锚

- 桩测：maker 单下单/post_only 拒成交降级/单腿裸奔超时补平三径断言。
- 判据面 maker 费率键在册+eval 净算双腿分费制桩测对。
- 生产灰度：probe 仓 maker 单实发→成交/降级实测数据落票回填
  （放量参数禁施工落——呈负责人审）。
- 闸绿戳+CI 全径（执行面码件无快径豁免）。

## 范围外

- 不动发现面/阈值面（min_net_apr 已定 0.20）；不碰 basis/dcd 族；
  不动 risk 硬线判据。
