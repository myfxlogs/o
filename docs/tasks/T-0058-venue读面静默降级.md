# T-0058 venue 读面静默降级根修——fail-open 聚合→fail-closed

| 字段 | 值 |
|---|---|
| 状态 | 施工中 |
| 优先级 | 高 |
| 涉及面 | `cmd/core/venue_readers.go` `internal/ledger` `internal/trading/okxexec` |
| 证据日 | 2026-10-24（ledger_diff 假警实证） |

## 病灶实证

2026-10-24 10:59:44 告警 `对账差 okx/USDT venue=2057.5736 ledger=2133.4036`——
venue 实态：trading cashBal **75.9271 USDT** + savings 2057.3787 + funding 0.2007 = **2133.51**（≈ledger，真差<0.1）。
读面却丢整条 trading 腿：`frameVals` 在账户帧缺席/eqUsd 未到时返 eq=0 → `err == nil && eq != 0` 静默跳过（`venue_readers.go:159`）→ 读值只剩 savings+funding=2057.57 → 假差 $75.8。
同型 fail-open 还在 `AssetBalances`/`SavingsBalance` 两处 `err == nil` 静默跳过——任一 REST 读件失败都会产假差。
此前不可见：trading 户长期为 0，丢腿=丢 0；T-0050 事故 unwind 回款留 $76 在 trading 后病灶现形。

## 根修要求

- `frameVals` 读败/缺席→**REST /account/balance 兜底**；兜底再败→整读返回 err，对账环跳过本轮（fail-closed），禁部分聚合。
- `AssetBalances`/`SavingsBalance` 同规：读败→err 上抛非静默跳腿。
- 告警文本区分「读件缺席跳过本轮」与「真差超限」——ledger_diff 只在真差时发。
- trading 户 $75.93 闲资归停泊位（Earn）：走正规划转径并落 `fin_cashflow`（earn_in），禁工具绕行——顺带核「停泊闸为何不扫 trading 户闲资」（park 覆盖面缺口，同票查因补件或立缺项）。

## 验收锚

1. 断帧注测（mock frameVals 返 0/err）→ 读面返 err 或兜底成功，对账不差分，无 ledger_diff 误警。
2. 三分量任一读件注入失败→对账轮跳过+WARN，非假差告警。
3. trading $75.93 归位 Earn：fin_cashflow 有对应行，venue 三户实拍与账平。
4. build/vet/test 绿；生产跑 ≥30min 无 ledger_diff 假警（读件全好时差<容差）。
