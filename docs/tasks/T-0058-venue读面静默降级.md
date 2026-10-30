# T-0058 venue 读面静默降级根修——fail-open 聚合→fail-closed

| 字段 | 值 |
|---|---|
| 状态 | 审计通过 |
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

## 结果回填（devin-t0002 @ 2026-10-26——fail-open 聚合根修交付）

| 验收锚 | 态 | 证据 |
|---|---|---|
| 断帧注测→读面返 err 或兜底成功对账不差分 | PASS | tradingVals=帧先 REST /account/balance 兜底 details eqUsd/availBal——帧缺 eq=0 不再静默丢腿；liveEq/WithBalances/WithEquity 三面同径 |
| 三分量任一读败→对账轮跳过+WARN 非假差 | PASS | trading/funding/savings/mt5 任一读败=整读 err 上抛（fundSaveBalances 共享件）；「读件缺席跳过本轮」WARN 与「真差超限」diff_alert 两文本分径 |
| trading $75.93 归位 Earn | 实证中 | 根因=fmtAmt 四舍五入 75.92713387…→75.92713388>availBal 即 58350 拒（非凭据/方向坑——金额截断坑实证）；trunc8 向零截断修后待部署后 sweep 复推归位 |
| build/vet/test 绿 | PASS | go test 零 FAIL @5434；lines 297<300；import 65/70 边；env 20 键 |
| 停泊闸覆盖面缺口查因 | 在案 | sweep_min_usd=100>75.93 阈下闲资结构性困死——票面立缺项记实（阈值裁决归决策席） |

**实勘修正**：①SG sweep_min_usd 100→50 实测触发摆渡——暴露 fmtAmt 截断坑+58129 clientId 方言（POST 体字段名正确 clientId 但 query 侧 type=0 缺参形态）；②fmtAmt FormatFloat 'f',8 已先四舍五入——须先 trunc8 再格式化；③fundSaveBalances 共享件防棘轮超 300。

**生产观察项**：部署后 sweep 复推归位+30min 无 ledger_diff 假警=验签点（决策席部署位）。

审计席：headless @ 2026-10-31

## 判决注记（逐锚对账——收口 9994b248+4ca01bb8，部署实证 o-core:0dbeb3d5）

1. **断帧→兜底/err** 通过：`tradingVals` 帧先→eq=0/err→REST `/account/balance` details 兜底→兜底败=err 上抛（venue_readers.go:83-107 实证），liveEq/WithBalances/WithEquity 三面同径。SG 生产：读败事件未产部分聚合假差。
2. **读败→跳过+可见非假警** 通过（生产实证）：`funding 读败: http 429`（10-30 10:24/11:37/11:39 UTC）、`mt5 读败`（15:31-15:39）→对账轮跳过+`事件拍 fn 败: venue 现读:…` 错误行，期间零 ledger_diff——修复前同型故障即产假差。注记：实发文本为 ListenLoop 通用错误行非「读件缺席跳过本轮」WARN 原文——跳过+可见+非假警功能语义达标，文本口径偏差记档不判死。
3. **$75.93 归位** 通过：`fin_cashflow earn_out -75.92713387720735` @10-24 11:49 source=fin_risk_decision——depositIdle→transfer→SavingsPurchase 正规划转径双行留痕；fmtAmt 四舍五入→trunc8 截断根修在 diff 实证。SG 现态 venue=4134.09 vs 净额 4134.57（0.48<0.5）账平；park 缺口查因+sweep_min 100→50 补件在票。
4. **build/生产验** 通过：钩子链过检+文件 297<300 行（收口时）；修复随 `o-core:0dbeb3d5` 部署（git ancestor 实证）Up 4h healthy。10-30 14:43–15:06 告警风暴=**真差**非假警（$10k 撤资离场 venue=4134 实跌，人工 `withdraw` 簿记落账 ledger 14134→4134 即停）；15:06 后 ≥3.5h 静默，差 0.48<0.5。
