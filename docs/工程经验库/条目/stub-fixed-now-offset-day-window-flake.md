---
schema_version: 2
id: stub-fixed-now-offset-day-window-flake
kind: pitfall
risk: medium
status: verified
keywords: [日界 flake, UTC 日窗, now()-interval 桩, 固定位偏移, 当日谓词, Truncate(24h), 钉点, 桩测时钟绑架, 午夜跨日竞态]
scope: [一切桩测时戳 × 生产面日窗谓词的组合（trx_rollup 类日界函数为高发）]
summary: 桩测用 now()-N hours 固定位造数、生产面按 UTC 日窗（Truncate(24h)+[day,day+24h)）过滤时，UTC 00:00–(offset) 窗内桩数据漂过日界被谓词滤掉→断言恒红；修法=桩时戳钉当日日内（day.Truncate(24h)+偏移，与谓词同域）；甄别=红窗恰为 [00:00, 偏移) 且窗外必绿
first_seen: 2026-10-26
last_verified: 2026-10-26
evidence: [D-0026, T-0147 验收撞见 UTC 06:50 红, 48h 逐秒扫描旧桩违域 57600 时点=精确 8h 红窗负控制, 07:53 UTC 红窗内新桩实跑绿]
---

# stub-fixed-now-offset-day-window-flake——固定位 now()-N h 桩 × 当日日窗谓词 = UTC 晨窗必红

## 症状
- 桩测只在 UTC 特定晨窗红（本例 00:00–08:00），窗外必绿；红的不是业务断言竞态，是数据整个缺席（`voting=0（应 2）`）。
- 同一测试在 CI 与本地不同时区跑结果漂移；「昨天绿今天红」与 UTC 时钟位置强相关。

## 根因
桩造数用 `now() - interval '8 hours'` 固定位回看，生产面 rollup 按 `time.Now().UTC().Truncate(24h)` 取当日窗 `[day, day+24h)` 过滤。UTC 时刻 < 偏移量时 `now()-8h` 落前一日→被当日谓词滤掉→同窗桩行数不足→Δ 类断言（相邻快照差分）缺席。测的是 rollup 语义，被时钟位置绑架。

## 修法
- 桩时戳**钉当日 UTC 日内**：`day := time.Now().UTC().Truncate(24*time.Hour)` + 固定偏移（如 `day.Add(2h)`/`day.Add(8h)`），与谓词取域同一表达式——两边对「今天」的定义同源，永不漂日界。偏移落日内即可，时间戳未来于 now 无碍（谓词只看窗不看先后于钟）。
- 同族一并清：喂日窗谓词表的桩用裸 `now()` 也有午夜跨日竞态（插入 23:59:59.x、读在 00:00:00.y 出窗）——一并钉点。
- **非病灶勿动**：龄期语义桩（TTL 过期、staleness 阈值如 `now()-5min`/`now()-30h`）与读面无时间谓词的桩（如全量 UNION 流水）不患此病，勿超范围改。

## 甄别
红窗恰为 `[00:00, 最大偏移)` UTC 且窗外恒绿 ⇒ 本类。锚证三板斧：①SQL 在事故时点确定性复现出域（`timestamptz '…06:50+00' - interval '8 hours'` 对当日窗比布尔）；②全日界逐秒扫描新旧桩对照（旧桩违域时点数=红窗宽×3600，新桩零违域——负控制实证）；③红窗内实跑新桩绿。
