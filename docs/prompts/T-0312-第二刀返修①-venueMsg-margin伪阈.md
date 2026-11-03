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

**修法（钦定二选一，施工择一并在回填写明所择+理由）**：

- **甲（荐）删伪阈**：`state='recommended'` 本身就是达阈认证——推送面不该
  二次裁决阈值（双真相源风险）。`venueMsg` 删 margin 字段+达阈再检，直接
  取 `persist` 最高 board 组文。Margin 字段一并删出解析 struct。
- **乙补真阈**：`board` 结构体加 `Margin float64 json:"margin"` 字段+
  `measure`/`boardFor` 写值（让载荷自描述）。metrics 每拍重写所以历史行
  无 margin 无碍——recommended 翻牌那拍必然是新写载荷。若择此，venueMsg
  保留阈检但测试 fixture 改全真字段形。

**测试修**：`TestVenueReqPush` fixture 改生产真形（`top_n`/`baseline_source`
带上、margin 不塞——甲案）或带真 margin（乙案），断言富文含实数百分比
（如 `12` 字样）而非仅「请注册」。补一条「无 margin 字段的 metrics 也出
富文」（甲案直接覆盖）。

## 验收

- `go test ./internal/platform/notify/` 过；
- fixture 与生产载荷形同（`board` 结构体 json tag 逐字段对拍）。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | 双修法自由度恐致施工再选错 | 决策残留 | 钦定甲荐——state 已权威，删伪阈最净 |
| 2 | 乙案涉 scout 度量码改动面更大 | 范围险 | 甲案只动 notify 件——标荐即限范围 |
