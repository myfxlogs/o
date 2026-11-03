# 施工返修①提示词：T-0312 第二刀——venueMsg margin 伪阈

**角色**：`[角色:施工]`　**票**：T-0312 ④件　**分支**：`t/0312-regime-push` 续提
**被驳回 commit**：`a778d65f`（三件对形过，一件语义坑）

## 驳回件 R1：venueMsg 的 margin 阈是伪阈——生产 metrics jsonb 无此字段

**病灶**（`internal/platform/notify/venue_req.go` `venueMsg`）：

```go
if *b.NetAprMid < *b.Baseline+*b.Margin { continue }
```

`board` 结构体（`internal/research/scout/metrics.go`）JSON 序列化字段只有
`family/net_apr_mid/persist/baseline/baseline_source/top_n` 六个——
**`margin` 从不进 metrics jsonb**。生产里 `b.Margin` 恒 nil→`continue` 全灭
→`best<0`→**每条推送必走保底文**，富文永不发。测试被 fixture 里塞的
`"margin":0.05`（生产不存在的字段）掩过去。

**修法（钦定三案，业主点拨后决策席订——丙案标荐）**：

- **丙（荐，业主令 2026-11-03 钦定）proto 为源**：`proto/scoutv1/scout.proto`
  加 `VenueMetrics{repeated MetricBoard boards; string measured_at}`+
  `MetricBoard{family; net_apr_mid; persist; baseline; baseline_source; top_n}`
  →`buf generate`→scout 度量写 metrics 列改用
  `protojson.Marshal(VenueMetrics)`（jsonb 列照存——SQL 可查性保留）；
  `venue_req.go` 删本地解析 struct，`protojson.Unmarshal` 同生成型——
  **schema 唯一源 .proto，字段漂移=gen 后编译炸**。margin 伪阈随删
  （state 已权威），取 persist 最高 board 组文。`VenueRow.metrics` 的
  `google.protobuf.Struct` 临时逃生口同票换 `VenueMetrics`。
- **甲删伪阈**（proto 引型受阻时退路）：`venueMsg` 删 margin 字段+阈检，
  本地解析 struct 照留（漂移根留但本次坑销——须票面注明欠 proto 债）。
- **乙补真阈**（否案留档）：board 加 margin 字段——proto 令下不采。

**测试修**：`TestVenueReqPush` fixture 改生产真形（`top_n`/`baseline_source`
带上、margin 不塞——丙/甲案）或带真 margin（乙案），断言富文含实数
百分比而非仅「请注册」。丙案补「scout.Board 字段演变=编译错」隐含
保障注明（无需测试——编译器即测试）。

## 验收

- `buf generate` 后 `go build ./... && go test ./internal/platform/notify/
  ./internal/research/scout/` 过；
- fixture 与生产载荷形同（`MetricBoard` proto 字段逐字段对拍）；
- metrics jsonb 新写行为 protojson 形（字段名 proto 蛇形——SQL `->>` 径不变）。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | 双修法自由度恐致施工再选错 | 决策残留 | 业主令后钦定丙案——proto 唯一源 |
| 2 | metrics 列历史行为旧手写 JSON 形 | 兼容险 | protojson 读旧字段同名兼容——每拍重写自然收敛，禁为兼容留双解析 |
| 3 | gen 产物 gitignored | 装配险 | 部署/CI 序已含 buf generate（工程纪律钉）；本地验收同令 |
