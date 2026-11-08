# T-0478 fin_holding 镜像陈行根修——venue 消失币种缺席即删（幻影余额病灶）

- 状态：已批（CTO 立票直批 2026-11-08——T-0424 裁决附带发现病灶立票）
- 领取：常径待派
- 优先级：P2——镜像面缺陷（幻影余额毒诊断取证/earn 汇总读面）
- 收口移交归：决策席

## 背景（T-0424 裁决实证）

fin_holding CORE 行：`spot_dust=3813.34977664`+`earn_flexible=3813.34977664`
stale=f asof=2026-11-07 10:59——venue 实拍 11-07 11:00 全卖后 CORE 余额
退出 venue 响应集，sync 只 UPSERT 不回零→幻影行永续 stale=f 挂陈值。

病灶：`internal/ledger/holding/sync.go` sync() 成功拍零缺席处置——
fetch 返回集外存量行永不收。

毒面实证：

- `internal/ledger/diagnosis.go` 取证查询（无 stale 过滤）——诊断
  草案拿幻影 3813 CORE 当 venue 实存；
- `internal/ledger/report/report.go` earn 汇总 `stale=false` 过滤虽在，
  幻影行 stale=f 照进——earn 停泊额虚增 ~$49。

## 钦定语义

```
修法面：成功拍收尾——fetch 覆盖域(venue='okx'×fetch kind 集)存量行
       不在本次返回集→DELETE（镜像=venue 实存真源非审计留痕；
       审计痕在 venue bills/ops_event 面另存）
护线：  只清 source='api' 且 venue='okx' 且 kind∈fetch 返回 kind 集行——
       bag.go 显形行/手工行（非 fetch kind 域）禁误收；
       fetch 败拍照旧全行 stale=true 不删（拉败非真消失）
```

## 范围

1. holding/sync.go 成功拍收尾加「缺席即删」段（护线谓词钦定）；
2. 桩测四断言：返回集缺 ccy→存量行删 / 同拍返回行照 UPSERT /
   bag 写行（非 fetch kind）不误收 / fetch 败拍全标 stale 不删；
3. 幻影行自洁：本修部署后下拍 fetch 即核销现存 CORE 幻影行——
   收口附部署后直查（自洁生效实证），不另跑手工 SQL；
4. 数据流.md 持账节同 commit 补「缺席即删」口径行。

## 验收

- 桩测四断言绿+check_all 对口；
- 部署后 fin_holding CORE 幻影行核销（行无或=0.0098 实值）；
- diagnosis/earn 报表读面无幻影。
