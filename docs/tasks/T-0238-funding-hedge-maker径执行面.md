# T-0238 funding-hedge maker 径执行面——post_only 限价双腿摩擦砍半

> 状态：待验 | 领取：devin-t0238 @ 2026-10-29 | 优先级：P1（D-0037 主线条——T-0237 调研裁定的次径可叠件）
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


## 结果回填

**落件**：Leg+Maker/LimitPx 位；Executor 接口+Cancel（OkxExecutor 实装
CancelOrder/MT5 返不支持 err/RouterExecutor 按 venue 透传/fake 桩齐）；
makerAttempt（maker.go 自 flow.go 拆——行棘轮）：exec.maker_enabled 闸+
okx 腿+非 close 腿+md_ticks bid/ask 鲜（<30s fail-closed）→post_only 挂价
（buy→bid/sell→ask，spot quote 腿名义→base 换算）→500ms 轮询至
exec.maker_timeout_s（缺=20s）→时限取消+Resolve 定真态（竞态成交
按成交记）→降级 market 新 clOrdID（改价追单禁绝）；裸奔窗 leg1 maker
成 leg2 未=leg2 走自身 maker→降级闭环，leg2 双径全败仍 LEGGED 回滚
原语义不动；ord_type 列（fin_0087）记每腿实径 post_only/market/cancelled。
分费制：research.funding.maker_fee_bps 软键——在册=funding eval
maker+taker 单腿复合+opener 闸 2（maker+slip)+2(taker+slip) 混合；
缺键=原双 taker 形零行为变。
**桩测**：TestMakerFillAndFallback——maker fill 记 post_only+超时取消
（cancelled≥1）+降级腿 market 三断言绿。
**时限选型**：20s——对冲裸奔窗受窗口界与 taker 原径同级，钦定候 5-60s
区间中值保守；轮询=期界等待合法面（sched-exempt-ok 行内注）。
**灰度**：exec.maker_enabled 键缺=false 默认关——prod 上量键位备而非开；
probe 径共用 Execute 同闸（exec.probe_notional_usd 小仓实发由负责人
审后落键——本票不留参数变更）。
**坑（同票坑注）**：paramKeys 插键中段致后续索引全移判据串位（funding
双 E2E 红实证，已逐位勘正+断言护）；maker 单下单成功后即交 Resolve
轮询故非即时成交态正常（market 与 post_only 异步语义差）。
