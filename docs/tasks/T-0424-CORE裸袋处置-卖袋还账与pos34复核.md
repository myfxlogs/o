# T-0424 CORE 裸袋处置——卖袋还账+挂账冲销+pos34 复核（T-0419 DM 裁 A）

- 状态：已批（决策席立票直批 2026-11-07——T-0419 票面件④ DM 裁词=A）
- 领取：待派
- 优先级：P1——钱径处置件（delta 中性破口存续中，敞口虽小章程不谅）
- 收口移交归：决策席

## 背景（DM 裁词在案）

T-0419 生产实证残袋：fin_holding `CORE spot_dust≈1924.61` +
`earn_flexible≈3813.34`——开腿实得 1915.63 平腿只清 24.99 的裸袋
被 Earn 自动收息，=无对冲方向多仓。

**DM 裁词=A 卖袋还账**：市价清袋→USDT，损真化、敞口归零、账实彻底平。
方案 B（留袋续息）被否——delta 中性是章程级破口，$50 级处置成本
换破口闭合为钦定取舍。执行面本票零授权外扩——venue 写面仅限
本票钦定两笔（Earn 赎回+现货卖单），禁搭车。

## 钦定语义

```
处置径：  earn_flexible 3813.34 CORE→赎回(earn_out)→现货市价全清
         →USDT 回笼;滑点/手续费记真损——不许限价单挂等（敞口拖时）
挂账冲销: fin_0130 部署后（bare_bag_* 列实存）——#33 挂账按卖袋
         实得重结:settled=现金净+卖回实额=真实现损,残差=处置真损
pos34:   滞留 open 的平衡对冲对(attempts 耗尽 gave_up 已火)——
         复核去留:正常平持仓归账 or 保留续跑,处置结论落票面+事件行
```

## 范围

1. Earn 赎回+现货市价清袋（venue 写面仅此两笔——工具径按既有
   okxexec/rescuespot 族或新 tool，禁手 API 绕账）；
2. 卖回 cashflow 落账（kind 按账谱——spot_sell/处置真损差额
   cashflow 对齐）；
3. fin_0130 部署后：#33 bare_bag 挂账冲销重结（settled 按实得
   重算=真实现损；冲销迹落 ops_event）；未部署先挂起等窗——
   禁对不存在的列写 SQL；
4. pos34 复核处置+结论落行；
5. 全链路账实对拍：处置后 fin_holding CORE 残尘<阈 vs venue
   实拍一致（earn+spot 两户）。

## 验收

- venue 实拍：CORE earn_flexible→0、spot_dust<处置阈；USDT 回笼额
  与 cashflow 落账一致（对拍误差不超手续费用度）；
- #33 settled 重结=真实现损（非 -24.67 假亏亦非 0）；
- pos34 处置结论落票面+事件行；
- 绿戳对口+全链（涉 venue 写的桩以 fakeExec/DRY 锚——真单不桩）。

## 结果回填

- 状态：待派
