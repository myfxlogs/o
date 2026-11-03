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

- **丙（荐）共享类型杀漂移根**：`scout.board` 导出为 `scout.Board`（或挪
  公共 types 位置——依赖方向以编译不环为准），`venue_req.go` 删本地解析
  struct 改用 `scout.Board`——**生产/消费同一份型，字段漂移编译期炸而非
  运行时静默降级**。这就是同 module 里的「protobuf 级 schema 保障」零
  工具链版。消费面仍删 margin 阈（state 已权威）+取 persist 最高组文。
- **甲删伪阈**：若 import 向有环/PKG 边界争议——`venueMsg` 只删 margin
  字段+阈检，仍本地解析 struct（漂移根留但本次坑销）。
- **乙补真阈**：`board` 加 `Margin float64 json:"margin"`+`measure`
  写值（载荷自描述）——metrics 每拍重写，recommended 那拍必然新载荷，
  历史行无碍；venueMsg 保留阈检但 fixture 全真形。

**测试修**：`TestVenueReqPush` fixture 改生产真形（`top_n`/`baseline_source`
带上、margin 不塞——丙/甲案）或带真 margin（乙案），断言富文含实数
百分比而非仅「请注册」。丙案补「scout.Board 字段演变=编译错」隐含
保障注明（无需测试——编译器即测试）。

## 验收

- `go test ./internal/platform/notify/` 过；
- fixture 与生产载荷形同（`board` 结构体 json tag 逐字段对拍）。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | 双修法自由度恐致施工再选错 | 决策残留 | 钦定甲荐——state 已权威，删伪阈最净 |
| 2 | 乙案涉 scout 度量码改动面更大 | 范围险 | 甲案只动 notify 件——标荐即限范围 |
