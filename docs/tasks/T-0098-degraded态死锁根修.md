# T-0098 degraded 态死锁根修——策略浴火重生径

| 字段 | 值 |
|---|---|
| 状态 | 完成（判过并码在案，11-01清账翻面） |
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

审计席：headless @ 2026-10-31

**判决注记（逐锚对账收口 commit 749ec186，diff=run.go+38/machine_test.go+61/票面+15）**：
- 锚①「degraded>TTL 自动 retired+留痕」成立：`retireStaleDegraded`（run.go:195-225）挂 `scan` 内 admitProposals 前（85→88 行序实对）；筛 `state='degraded' AND entered_at<now()-make_interval(hours=>$1)`，TTL=`strategy.degraded_ttl_hours` 缺/非正默认 24h=票面钦定形；`Transition` 同事务翻 state+`entered_at=now()`+落 `fin_strategy_transition`（machine.go:51-55），reason 原文逐字=票面钦定串；entered_at 每次迁移重置→龄=degraded 入龄语义正；degraded→retired 在合法闭集（machine.go:19）。事件驱动+boot 首拍触发，未新增周期原语（合 D-0008 禁线）。
- 锚②「同键新提案可重开 paper（#1 桩测）」成立：去重=部分唯一索引 `(venue,inst_id,direction,template_ref) WHERE state<>'retired'`（fin_0050），retired 行出索引即解锁；`TestDegradedTTLRetire` 夹具行键（venue 默认 okx/direction 默认 short_perp_long_spot 见 fin_0008/fin_0014/template funding-hedge）与新提案冲突键全对位——不退役则 ON CONFLICT 吸收无 paper=测试反例可辨，确系真同键复刻非旁路。
- 锚③「live/probe/paper 不波及」成立：新径 WHERE 钉死 `state='degraded'`，他态构造上不可入选；桩断言龄内 degraded+paper 行零波及。
- 纪律面：run.go 260 行<300、测试件免门禁、gofmt 净、无 `_ =` 吞错新增、write_err 计门注册键内、无 env 直读/禁三件套/秘钥形态；回填三件齐备。小注（不阻断）：rows.Scan 败径未挂 rej.Inc——与邻 admitProposals 同款约定一致。
