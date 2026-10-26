# D-0026 TestTrxRollupIdempotent UTC 日界 flake 根修（T-0147 验收撞见）

## 态：待施工
## 原编号：T-0152（2026-10-26 前后台分系改编——后台 D 系/前端 T 系）
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
