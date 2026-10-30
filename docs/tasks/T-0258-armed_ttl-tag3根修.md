# T-0258 armed_ttl tag3 根修——判据改执行活动戳+退役走 Transition 留痕

> 状态：**在飞（无头）** | 领取：devin-t0258 @ 2026-10-30 03:31Z | 优先级：**P1（复跑前置——MANAGE-151 钦定插队件，
> armed_ttl=4h 闸下 7 条 live funding-hedge 每日进斩杀域）** | 执行径：**无头**
> 立项：决策席 @ 2026-10-30（管理令 20261030-105000-MANAGE-151 裁一前置②判=涉）

## 面况

`internal/portfolio/lifecycle/run.go` `retireStaleArmed` 的 tag3（行 252-259）：
`fin_strategy` 中 `state IN ('live','probe')` 且无未平仓行、且
`COALESCE(max(fin_position.opened_at), entered_at) < now()-ttl` 者置 retired
（`lifecycle.armed_ttl_h`，现值 **4h**）。两缺陷实证：

1. **判据错配**：谓词活动锚=持仓开仓戳。被开仓闸持续拒的活策略（lifetimeOK/
   breakevenOK/dcd_quote 上游拦——拒在开仓前、永不产 position）活动戳冻结于
   entered_at，ttl 一到即被误杀。**全拒≠不活**。10-30 01:00Z 入场的 7 条
   live funding-hedge（BARD/HUMA/BCH/OKB/ZEC/BASED/QTUM，零仓零执行）05:00Z
   起逐批落斩杀域=现行犯。
2. **绕过 Transition 留痕**：tag3 裸 `UPDATE fin_strategy SET state='retired'`
   ——`fin_strategy_transition` 零行（T-0255 取证实证「状态=retired 但零条
   迁移记录，疑绕过 Transition() 直改」即此径）。同函 tag2（提案面）同型
   裸 UPDATE，但提案溯源走 `fin_proposal.retire_reason` 列——策略面无此列，
   只能靠 transition 表留痕，故逃逸更严重。

## 钦定界（MANAGE-151 票面要件）

- **判据改执行活动戳**：退役谓词的活动锚由「持仓开仓戳/entered_at」改为
  「该策略最近一次开仓侧活动证据」——含执行尝试类证据（`fin_execution`
  行戳）与开仓闸拒留痕（`fin_eval_reject` 同 inst 最近 ts 或同义持久
  证据——上游拦拒在开仓前、无 execution 行产出，此子集必须算活动）。
  有 ttl 窗内活动证据者不退役；真·无活动 dormant 照退（语义不丢）。
- **退役走 Transition 留痕**：tag3 退役改经 `Transition()`（machine.go:35，
  tx 内 UPDATE+transition 行）——reason 明写 `armed_ttl_expired` 同因续用；
  批量逐行 Transition 同 degraded 超龄径（run.go:216-223）先例。tag2 提案
  面已有 retire_reason 列溯源，本票只修 tag3 策略面；tag2 顺手改 Transition
  语义化与否由施工裁，偏离落结果段。
- **ttl 现值 4h 不动**（判据错修后语义自正——4h 无活动退役是设计意图）。

## 范围外

armed_ttl 数值调整、提案面 tag2 的 retire_reason 语义变更、其他退役径
（degraded/blacklist/oscillation 各径本已走 Transition）。

## 验收锚

1. 桩测：live/probe 策略近 ttl 窗内有开仓侧活动证据（execution 行 或
   eval_reject 闸拒行）者**不退役**；零活动者退役——两向断言非 skip；
2. 退役落 `fin_strategy_transition` 行（from_state/to_state/reason=
   armed_ttl_expired），无裸 UPDATE 旁路；
3. 现犯场景回放桩：gate 连拒 live 策略超 ttl 仍存活（对应 7 条 funding-hedge
   实例形）；
4. check_all 绿戳+数据流.md 若涉链变同 commit；
5. 票面回填+产坑 KB（若有）。

## 产出

谓词根修+留痕根修+桩测。部署=决策席走 deploy_image.sh 另序（复跑前置件，
收口后急诊部署）。
