# T-0152 TestTrxRollupIdempotent UTC 日界 flake 根修（T-0147 验收撞见）

## 态：✅ 验收毕投产（8835b1d 合入·决策席 2026-10-26）
## 领取：zcode-glm（施工席）@ 2026-10-26（支 t/0152-trx-stub-day-boundary）
## 域：测试基建 / TRX rollup
## 源出：T-0147 验收跑桩窗实案——TestTrxRollupIdempotent 红
  `voting=0（应 2=Δreward 正增量）`。UTC 06:50 实拍：桩插入
  `now()-8h` 快照落进前一日（22:50 UTC），`ListTRXAccountByDay`
  当日谓词滤掉→同日只剩 1 行→Δ 缺席→voting=0。

## 病灶
桩固定位 `now() - N hours` 而非「当日 UTC 日内钉点」——
UTC 00:00–08:00 窗口内必红，窗外必绿。经典日界 flake：
测的是 rollup 语义，被时钟位置绑架。

## 施工
- 桩插入时间钉当日 UTC 日内（`day.Truncate(24h)+2h/+8h` 形）——
  与 rollup 谓词同域，不再漂过日界。
- 同族普查：grep 桩测里 `now() - interval` 配 day 谓词的其他锚，
  同病灶一并清（trx_rollup 类日界函数桩为普查重点）。
- trxRollup 生产语义勿动——bug 在桩不在件。

## 验收锚
- 桩在 UTC 任意时刻跑皆绿（可 faketime 或边界时刻锚证）。
- 全仓 day 谓词桩零同类 flake。

## 结果回填（2026-10-26 zcode-glm 施工席）

### 实现摘要
- `cmd/core/trx_test.go` TestTrxRollupIdempotent：桩时戳全数钉当日 UTC 日内——
  `day := time.Now().UTC().Truncate(24h)`（与 trx_rollup.go:17 谓词取域同一表达式）；
  account 快照两行 `day+2h`/`day+8h`（原 `now()-8h`/`now()-2h` 固定位），同测试
  fin_trx_txs 行 `now()`→`day+8h`（午夜跨日竞态同族清）。
- **生产件零触碰**（trx_rollup.go/pgstore/trx.go 未动——病在桩非件）。

### 同族普查结论（全仓）
- 生产面 now 派生日窗谓词仅 trxRollup 一径（`SumTRXTxRentalByRange(day,day+24h)`
  + `ListTRXAccountByDay(day)`）；喂表桩测仅 trx_test.go 一件。
- TestHedgeFlowUnionAndBandwidth 的 `now()` 桩非病灶（hedge_flow.go 读面 UNION
  零时间谓词）；全仓其余 `now()-interval` 桩皆为龄期语义（TTL/staleness）无日窗
  断言——零同类残留。
- 坑条目已立：`docs/工程经验库/条目/stub-fixed-now-offset-day-window-flake.md`
  （+kbgen 索引重建）。

### 锚证（faketime 缺位，走边界时刻锚证）
1. 红窗内实跑绿：07:52–07:53 UTC（旧病灶必红窗 00:00–08:00 内实测）
   `go test ./cmd/core/ -run TestTrx -count=1` 全 PASS；全包 27.99s ok。
2. 旧病 SQL 确定性复现：事故时点 06:50 UTC 旧桩 `now()-8h`→22:50 前日，
   出 `[day,day+24h)` 域（谓词布尔 f）；新桩 day+2h 在域（t）。
3. 48h 逐秒扫描 172,800 时点（跨两午夜）：旧桩违域 57,600 时点=精确 8h 红窗
   （首现 00:00:00 UTC，负控制实证）；新桩违域 0。

### 机检
gofmt 全树 CLEAN ✓ / build ✓ / vet ✓ / test（cmd/core 全包+锚证三件）✓

### 偏离说明
- 无超单。支名沿用原号 `t/0152-trx-stub-day-boundary`（改编前已锁单，锁凭据连续）。
