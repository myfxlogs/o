---
schema_version: 2
id: live-state-not-position-output
kind: pitfall
risk: high
status: verified
keywords: [live 哑弹, 零仓位, 生命周期倒挂, fin_position, 静默零产出, 漏斗断点]
scope: [internal/portfolio/lifecycle, cmd/core, fin_strategy, fin_position]
summary: fin_strategy live=许可态非产出态——863 策略 31 live 90d 仅 2 结算仓（funding-hedge 29/31 哑）；策略产出判据=fin_position 实测行非状态列。
first_seen: 2026-11-10
---

# live 状态绿 ≠ 仓位产出绿——生命周期面与执行面可长期倒挂

**实案（2026-11-10）**：fin_opp_eval 评估闸判词「闸门拒开」被反复引用为
商业停滞主因，实际取证发现更深断点在闸门之后——fin_strategy 863 条
在册、31 条 live，90 天仅 2 笔 fin_position 结算仓。funding-hedge 族
31 条 live 中 29 条从未开过仓（含 #3544 腿缺失案）。策略生命周期
`live` 只表示「获准开仓」，不表示「正在产出」。

**坑形**：

1. **状态语义错觉**——`live` 是许可态不是产出态；看 live 数判断
   系统产出会系统性高估（31 live 听着健康，2 笔仓位才是真相）。
2. **静默倒挂**——评估闸拒绝负 EV 是显式拒绝（有 ops_event）；
   策略 live 但不开仓是静默零产出（无任何事件）——无探针时
   无限期无声。
3. **多层漏斗各断各的**——评估→提案→策略→仓位每一层都可能断：
   闸拒（评估层）、断供（枚举层）、哑弹（执行触发层）、腿缺
   （venue 面）。判「瓶颈在哪」必须分层取证，不可凭单层信号定性。

**对应件**：`cmd/core/supply_esc.go` `liveDudEscCheck`——live≥7d
零仓位→升件；曾开过仓的不在本面（接线已被验过）。

**判词纪律**：策略产出判据=fin_position 实测行，不是 fin_strategy
状态列；「live 在跑」永远要补一句「开过仓吗」。
