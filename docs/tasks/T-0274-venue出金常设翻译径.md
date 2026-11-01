# T-0274 venue 出金常设翻译径——type=1 划转行内外向分辨+bills→cashflow 出金面补设

| 字段 | 值 |
|---|---|
| 状态 | 施工中 |
| 立项 | `[角色:决策]` @ 2026-10-31（MANAGE-838 簿记裁决——FIX01 一次性冲正已销恒差，常设径缺口留存） |
| 领取 | claude-kimi @ 2026-11-01（支 t/0274-wdtrans） |
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

**择源亲证（真户探针 2026-11-01，SG 真钥只读 GET——okxprobe 同纪律凭据零落）**：
票面两候选前提均被真户实测推翻——①`/asset/bills` 实响应**无 from/to/subType**
（仅 billId/ccy/balChg/bal/notes/ts/type；`?type=1` 仅 3 行且 14:36Z 案拍缺席）；
②`/asset/withdrawal-history` **全盖外向出金**：链上提币与 OKX 内部户转出
（本案形态：wdId 440849407，amt=10000，state=2，chain/txId 空，to=对方手机号，
areaCode 86）皆在册——票面「内部转出漏网」不成立。**择源=候选②为常设主径**：
行面即外向定性（内转双跳 trading↔funding↔earn 天然缺席此源），零分辨赌猜；
禁全翻坑随源消解（该源不存在内转行）。10-30 案真拓扑（探针+funding 面
/asset/bills notes 字段实证）：14:36:01-03 之 ±10000 = trading↔funding **内转
双跳腿**（funding 侧对腿 type=131"Transferred to trading account"/type=130
"Received from trading account" 分拍在册）；14:36:03 Earn 申购（75）→14:41:09
赎回（76）；**14:42:08 外向真腿=wdId 440849407**（funding 面 type=2
"Withdrawal" -10000 @14:42:09 同秒证）。入金面无缺口：05:47:24 +10000 已由
deposit-history 既有径入账（fin_cashflow id=505，source_ref=1893593）。

**实现**（三件）：①`okxexec/client.go` 增 `WithdrawalHistory` 读件（镜像
DepositHistory 同构）；②`internal/ledger/funding/withdraw.go` 出金巡检件
——deposit.go T-0149 三锚范式镜像：wd-<wdId> 幂等键、账基线截止谓词、
bill+cashflow 同事务原子逐行隔离；state=2 才入账、非 USDT skip
（non_usdt_withdrawal）、amount=-amt（出向负号，FIX01 同号径）；fee 不入本径
（费面归既有 bills fee kind 支——混读=双计）；③`cmd/core/bills.go` wdSrc
装配+ledgerHook 同帧拍接线（读败 warn 下拍重试不炸事件主径）。
**ingest.go 零改动**——type=1 划转行维持 default 支零记账+源行留档（=正确
语义，分辨坑结论即「划转行禁记账」）。

**测试**（PG 实跑全绿 137.8s）：TestWithdrawScanIdempotent（案行 1 入+重放
0 重账+amount=-10000）；TestWithdrawScanGates（非 USDT 滤除+读败上抛零写+
夹毒行序隔离）；TestWithdrawScanBaselineCutoff（pre-baseline 零写——真户
wdId 433855125 等 2026-09/10 旧提币在册实证）；**TestWithdrawTranslationReplay
（仿真锚）**：重放 2001103/2001104 形态→type=1 双腿零 cashflow+案行恰一行
withdraw -10000。

**部署 runbook（决策席执行，决策位裁）**：常设径收编后 FIX01 一次性补账行
须换锚——id=516（withdraw -10000，source_ref=**2001103**，occurred 14:36:02）
锚错内转腿（净额对、锚点错）。部署（新径实落 wd-440849407 withdraw -10000
@14:42:08）后验 wd- 行在册→**DELETE fin_cashflow WHERE id=516**（或 UPDATE
换锚 wd- 源——二选一，删=源锚真径，改=存 id 留痕）；不删=双计 -10000→恒差
假阳。此为簿记数据动作非码件，施工席不动账。

**史核报告（存量 type=1 负行逐行判词——SG o-postgres 只读实拍 2026-11-01）**：
23 行全量（10-24~10-30）：50.25 系清扫对 12 行+100.xx/-187/-108/-124/-100.59
清扫出腿 6 行+ETH 微尘对 2 行（-0.00998388/+0.009983）+-30.97 出腿 1 行+
10-30 万金对 2 行——**判词：23/23 内转**（配对+funding 面 /asset/bills 对腿
+恒差中性三证；FIX01 前 recon 零假阳佐证），零补行零存疑；外部出金唯一
实证=wdId 440849407（非 /account/bills 行——FIX01 行级定性修正见上）。
史核保守界达成：存量 type=1 负行零回灌。

**遗留锚**：「部署后 24h 恒差告警零假阳」待部署后观察（机制已备——扫描器
入账+516 换锚后 recon 面恒差零源）。

**偏离**：①择源反转（②替代①）——票面候选① from/to 前提被真户实测否决，
钦定界原文「施工席择源，docs-v5 先核勿猜方言」授权在案；②/asset/bills
读件未新开（候选①落选——无分辨域无入径价值）；③涉及面实际未触
ingest.go（原票预计改 bill 分派面——源改判后无需）；④withdrawal-history
默认窗口（limit 100）与 deposit-history 同参——窗口溢出假设与 T-0149 同
（事件帧驱动增量，bookmark 幂等切口在 bill 表）。

