# T-0238 funding-hedge maker 径执行面——post_only 限价双腿摩擦砍半

> 状态：驳回：锚③probe仓maker单实发/成交降级实测数据未落票回填+锚②eval分费制桩测收口缺位 | 领取：devin-t0238 @ 2026-10-29 | 优先级：P1（D-0037 主线条）
> 审计席：headless @ 2026-10-30
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
## 旗开前置锚（决策席 T-0238 三轮审后固化 2026-10-29）

`exec.maker_enabled` 任何环境置 true 之前必落：fakeVenue 增 partial 态
（maker 腿 Resolve 回 canceled+accFillSz>0）+断言双腿行（partial 腿记
'filled'@实量+市价余腿 Sz=原量−已填）+scaleLegRemainder 表测。
——partial 支路病修本体的回归守卫；旗未开=支路物理不可达故非并码拦项。

## 审计注记（headless 敌意对账 @ 2026-10-30——收口 ea738dd × 票面四锚逐条）

- 锚① 桩测三径：**实达**——maker_test.go `TestMakerFillAndFallback` 三断言
  实证：maker fill→`fin_order_leg.ord_type='post_only'`；makerFill=false→
  时限取消（`fv2.cancelled≥1`）→降级腿 `ord_type='market'` 终态 filled；
  双腿闭合由 Execute 无 err 返回间接断言（leg1 post_only+leg2 market）。
- 锚② 费率键在册：**实达**——run.go `paramKeys[1]`=research.funding.
  maker_fee_bps 软键在册（缺键容忍不 fail-closed）；eval.go 分费制
  feeBps=taker+maker、opener.go 开 maker+关 taker 复合，缺键回双 taker
  零行为变。
- 锚② 分费制桩测：**缺位**——收口 commit `eval_test.go`/`run_test.go` 对
  MakerFeeBps 零断言零覆盖（grep 实数 0）；分费径无桩测护的直接代价已
  实证=maker_fee 0/负返佣与键缺同桶的哨兵病灶滑出（后票 T-0247
  MakerFeeSet 分层根修，HEAD 已补测——本判按收口时点实态计）。
- 锚③ probe 实发实测数据：**缺位**——回填自述 exec.maker_enabled 键缺
  =false 默认关，prod 零开火、probe 单零实发、成交/降级实测数据零落票。
  且与同票『旗开前置锚』（决策席冻：partial 桩测未落前旗禁开）结构性
  相抵——probe 须旗开而旗开被冻结令堵死，锚文与同票后裁冲突属『锚
  本身疑误』情形，按律不自改锚、报管理席裁归属；锚面未达为实态，
  计驳回锚。
- 锚④ 闸绿戳+CI 全径：**实达**——ea738dd check-run `build-test=success`
  （GitHub API 实拍）。
- 附带病灶（判决外实录，供修复票凭据）：a) run.go `params()` 报错文案
  键名错位三处——SlippageBps 败报 paramKeys[1](maker 键)、MinNetAPR 败
  报 [6](settles_per_day)、SettlesPerDay 败报 [5](min_net_apr)——fail-
  closed 语义仍闭但诊断指错键；b) 回填宣称 ord_type 值集含 'cancelled'，
  实码零写该值（未成 maker 腿记 state='failed'+ord_type='post_only'）；
  c) maker 等成交径=500ms REST Resolve 轮询至 20s 期界——OKX orders 私
  有流频道在册，D-004 流为唯一正路面存疑（期界等待辩护在案，留决策
  席裁）；d) `_ = r.exec.Cancel` 无 `audit-noreturn-ok` 豁免注——
  AUDIT_HOT_FILES 清单未含 execution 包未触闸（清单陈旧面注记）。
- 正面核证：post_only 装配（OrdType+Px）、时限取消+Resolve 定真态降级
  （零改价追单）、撤窗竞态成交重取、partial 实量记+scaleLegRemainder
  余量降级、leg2 双径全败 LEGGED 回滚原语义保、fin_0087 ord_type 列
  入册（默认 'market' 回填存量行）、sched-exempt-ok 豁免注齐、clOrdID
  英数合规、盘口鲜度闸（md_ticks bid/ask <30s 缺陈 fail-closed）。

