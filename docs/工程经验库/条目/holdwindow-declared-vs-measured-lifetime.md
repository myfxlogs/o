---
schema_version: 2
id: holdwindow-declared-vs-measured-lifetime
kind: pitfall
risk: high
status: verified
keywords: [摊销窗, max_hold, funding_flip, 实测持久度, breakeven, 声明窗, 收租段]
scope: [lifecycle, opener_gate, 判据口径]
summary: 摊销窗用声明上限（max_hold）而仓实际寿命被平闸/反转件先终止=
  乐观定价病灶；正解=min(声明上限,实测收租段中位)——摊销分母必实测
first_seen: 2026-11-10
last_seen: 2026-11-10
evidence_refs:
  - type: ticket
    ref: T-0521
  - type: commit
    ref: 42de87894
---

## 摊销窗声明上限 vs 实测持仓寿命——乐观 4 倍定价病灶

- **实证**：breakevenOK 回本窗判据把往返摩擦摊到 `guard.max_hold_hours`
  =72h 上——但 funding_flip 平闸在案，费率反转即平仓。md_funding 实测
  （17d×511 inst）：收租号连续段 p25=0.3d 中位=0.7d p75=1.3d——真实
  持仓寿命≈段长非上限，72h 摊销=成本年化被压 ~4 倍乐观。
- **根因型**：判据分母用「制度上限」而非「事实寿命」——上限是天花板
  不是预期值；凡摊销/年化/寿命预测口径，先问「什么事件会先终止这个
  仓」（平闸/反转/到期），分母取 min(声明上限, 该终止事件的实测周期)。
- **修法**：`measuredHoldDays`——md_funding 该 inst 近 30d 收租号
  连续段天数中位（正号段=short_perp 收租期,负号段=long_perp）；
  `holdDays=min(max_hold/24, 实测中位)`；该 inst 无该号段=史缺
  fail-closed 拒开（不虚构摊销窗）。右删失在飞段计入=保守向偏。
- **同族警讯**：所有「假设仓活满窗」的判词同险——paper 收益折算/
  Earn 摊息/DCD 权利金摊销,凡写 hold 窗先核终止事件表。
