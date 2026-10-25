# T-0098 degraded 态死锁根修——策略浴火重生径

| 字段 | 值 |
|---|---|
| 状态 | 完成 |
| 领取 | devin-swe2-1025 @ 2026-10-25 |

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

## 结果回填

**实现摘要**（devin-swe2-1025 @ 2026-10-25）——方案 A 钦定径：
- `retireStaleDegraded` 挂 `scan` 首步（admitProposals 前——同拍解锁即复评）：`fin_strategy state='degraded' AND entered_at < now()-TTL` 逐行走 `Transition→retired`，reason=「degraded 超龄自动退役——莫环重生窗」同事务留痕。
- TTL=`strategy.degraded_ttl_hours` 热启键，缺/非正=默认 24h（自疗径不配键不瘫——与 critKeys fail-closed 不同类：判据缺=停拍，TTL 缺=按票面默认值跑）。

**测试证据**：新锚 `TestDegradedTTLRetire`（超龄 30h 行退役+留痕断言；龄内 degraded/paper 行零波及；#1 场景复刻——退役后同键新提案开 paper 解锁 860 被吸径）——本机 O_TEST_PG_DSN 未设按约定 skip；`go build/vet/test` lifecycle 包绿，gofmt 净，`npm run build` 过。

**偏离说明**：无（方案 B dedup 收窄未采用——决策钦定 A）。BTC #1 存量 degraded 行部署后首拍 scan 即退役解锁，无需人工工单。
