# T-0274 venue 出金常设翻译径——type=1 划转行内外向分辨+bills→cashflow 出金面补设

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:决策]` @ 2026-10-31（MANAGE-838 簿记裁决——FIX01 一次性冲正已销恒差，常设径缺口留存） |
| 领取 | （session 标识 @ 时间；空=未领） |
| 优先级 | 中（现恒差已消=0.38<容差；缺口仅在业主下次外部划动时复发警——排 T-0272 波后施工） |
| 涉及面 | `internal/ledger/funding/ingest.go`（bill 分派面）、`internal/ledger/funding/deposit.go`（镜像范式）、`internal/trading/okxexec`（/asset/bills 读件若新开）、`docs/数据流.md` |
| 执行径 | 无头 |

## 背景（产侧亲证在案——勿重复诊断）

- `ingest.go` record 分派面只翻 173/174（funding）、160-163（dcd）、fee——**type=1
  划转行全弃记**（源行留档，cashflow 不写）。
- 2026-10-30 实录：bill 2001103（type=1 sub_type=12 **-10000** 14:36:02Z）=
  业主外部划转出金，无翻译→ledger-venue 恒差 $10000 循环警；
  FIX01 一次性补 fin_cashflow.id=516（withdraw -10000）销警（Dev-843 在案）。
- **分辨坑（禁全翻）**：史查 type=1 行 sub12 负向多为**内部划转**
  （funding→trading/earn 扫款，-30/-100/-187 各档，venue 净零——全翻=虚出金）；
  2001104 sub11 **+10000** 同拍=内转进腿（trading→funding，venue 净零）；
  bill 行 raw={} 不带 from/to——**行面自身无法分辨内外向**。

## 钦定界

- 出金常设径（施工席择源，docs-v5 先核勿猜方言）：
  - 候选①`/asset/bills`（funding 户账单，带 from/to 户号域）——to=外户=
    外部出金→cashflow withdraw；from/to 皆本户系（6/18/earn）=内转双跳免记。
  - 候选②`/asset/withdrawal-history` 镜像 deposit.go——只盖链上提币
    （state 完成态），OKX 内部转出（本案形态）漏网，作补充不独当。
  - 幂等键沿用 bill_id/source_ref=源行范式（dep-/wd- 前缀如开新源）；
    bill+cashflow 同事务原子（deposit.go T-0149 范式）。
- **史核保守界**：存量 type=1 负行**禁批量回灌**——逐行与 venue 侧/内转对
  证据比对，仅可证实外部出金者补行（2001103 已记=幂等跳过）；内转行一律
  不记。历史存量判不明=挂 rejects 留档不硬记。
- 内转对识别：同 ccy±同额近时窗（≤数秒）成对=内转候选，但外向分辨以
  from/to/venue 证据为准——**配对存在≠内转定性**（本案 ±10000 成对而
  sub12 外向）。
- 更新 docs/数据流.md 记账面小节。

## 验收锚

- [ ] 桩测：外部出金行→withdraw 入账；内转对行→零 cashflow；幂等重放→
  不双记；读败→不炸主径。
- [ ] 仿真锚：重放 2001103/2001104 形态→恰一行 withdraw（-10000），
  sub11 行零记。
- [ ] 史核报告：存量 type=1 负行逐行判词（内转/外部/存疑挂起）附票面。
- [ ] 全链机检绿+部署后 24h 恒差告警零假阳。

## 范围外/禁线

- 不改既有 173/174/160-163/fee 翻译语义；不动 deposit.go 入金径；
  不碰恒差告警阈值（0.50 容差）与对账环。
- 禁凭 sub_type/符号猜内外向——from/to 或 venue 证据为据；禁推断记账。

## 参考

- 病灶：`internal/ledger/funding/ingest.go` record default 支；范式：
  `internal/ledger/funding/deposit.go`（幂等/基线/原子三锚）；
  冲正案：`docs/簿记-FIX01-1万过境.md`@70e7090、Dev-843 报；
  参数字典纪律：skills/okx-api-spec（bills subType 表）。

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）
