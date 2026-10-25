# T-0098 degraded 态死锁根修——策略浴火重生径

> 实证（2026-10-25）：fin_strategy #1 BTC 卡 degraded 数月无人复盘——同键去重
> `state<>'retired'` 把它当 active → 860 个 BTC 提案全部被吸退役零新开。
> ETH #101 同类降级靠人工工单救回——机制缺「degraded 到期自动退役」自疗径。

## 钦定修法（两选一，施工可带方案）

A. **超龄退役**：lifecycle 循环加 degraded 行龄检——超 N 时（票面建议 24h，
   或配 config 键 strategy.degraded_ttl_hours 默认 24）自动 transition→retired，
   reason 写「degraded 超龄自动退役——莫环重生窗」
B. **dedup 收窄**：`state<>'retired'` 改 `state IN ('paper','probe','live')`——
   degraded 不再锁键，新提案照常开 paper（degraded 行留档不删）

建议 A：degraded 语义=「暂停待复盘」，有界的暂停才符合莫环自净——
无限期暂停=墓碑。决策定 A，施工若遇阻报回。

## 验收锚

- degraded 行 >TTL 后自动 retired+transition 留痕
- 新提案同键可重新开 paper（复用 #1 场景桩测）
- live/probe/paper 行不受波及
