---
schema_version: 2
id: drill-cleanup-misses-guard-close
kind: pitfall
risk: high
status: verified
keywords: [fin_guard_close, drill, simdrill, closing 僵尸, 假升件, closed_at, reconcile, 仓巡]
scope: [guard, simdrill, 平仓径, 探针, 演练隔离]
summary: 带外关账径（drill 清扫/人工）只写 position.closed_at 不关 guard_close——'closing' 僵尸行被 loopProbe 无限升假 P1
first_seen: 2026-11-09
last_verified: 2026-11-09
skills: []
evidence_refs:
  - type: doc
    ref: docs/tasks/T-0511-两仓closing滞留根修-sizer现读败fail-closed死锁.md
  - type: doc
    ref: internal/guard/loop.go:37
---

## 带外关账径漏关 fin_guard_close——'closing' 僵尸行=假升件放大器

- **实证**：仓#36/#38（fin_exec_drill 演练件 verdict=pass）——drill ⑦步
  跑产线 Warden.ScanOnce：delta_drift 触发→markClosing 落 'closing'
  →sizer 对真 venue 读合成腿败（MUU→XMUU 改名 ccy 缺席/mt5 OpenedOrders=0/
  SWAP ContractPos=0）；drill 清扫只 `UPDATE fin_position SET closed_at`
  →scanOnce 按 `closed_at IS NULL` 滤仓永不再试，loopProbe 查询缺同滤
  →对死行每 2h 升假 P1「平仓循环自残」+熔断挂起演练策略（升件族
  #5398/#5399/#5568/#5569 连刷 16h）。
- **机制**：`markClosed` 原子双写（position.closed_at+guard_close.closed
  同 tx）——凡绕开它写 closed_at 的径（drill 夭折防护、人工 SQL、未来新
  关账径）都会产僵尸 guard_close 行；**关账语义必须在 guard_close 表上
  同样闭环**，否则探针把「已平」当「平仓卡死」报。
- **修法**：双层——①loopProbe 先 reconcile（position 已 closed 的
  'closing' 行落 'closed'，阈键缺件也照扫）+告警查询补
  `p.closed_at IS NULL` 纵深；②drill 双侧清扫径同关 guard_close
  （源头防生——夭折防护注释本义补齐）。
- **教训**：探针查询与扫描查询的谓词必须同源——scanOnce 滤
  `closed_at IS NULL` 而探针不滤=两面对同一行语义分裂（主环弃管
  vs 探针报警）；凡新增「行态投影表」（guard_close/suspend 类），
  其关账语义要随 position/execution 主账一并闭环。
