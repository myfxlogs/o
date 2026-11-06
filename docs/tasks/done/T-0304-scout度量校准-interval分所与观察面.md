# T-0304 scout 度量校准——interval 分所 + rejects 观察面 + binance interval 归真

## 状态

**施工中**（devin-t0304 @ 2026-11-03；原态=已批）

## 背景（码面实证）

`internal/research/scout/metrics.go:95` `p.SettlesPerDay` 统一读 config
`research.funding.settles_per_day`（=3），carry/跨所两族全所同值年化。
`series.go loadVenue` 只回 rate 不读 `interval_h`——采集列存而不用。

失真实证：
- hyperliquid 费率=**小时结**（adapter 自存 interval_h=1），真 settles=24/日——
  统一×3 低估 carry 净 APR **8 倍**；跨所价差把 HL 小时率与 OKX 8h 率同×3 错比。
- binance `interval_h=0`（premiumIndex 无此字段——T-0302 实证）；真源=
  `fapi/v1/fundingInfo`（盒上已验 820 行带 fundingIntervalHours）。
- 回算无损：rate+interval_h 已存事实表，修法改派生径非补数。

## 钦定目标

1. **settles 分所**：`loadVenue` 行载 (rate, interval_h)；settles=
   `24/interval_h`（>0 时），缺/0→config 回退（binance 8h→3 恰与回退同值）。
   carry 用本所 settles；跨所=`(rate_v×settles_v − rate_okx×settles_okx)×365`。
   OKX 侧 md_funding 无 interval 列→config 回退（8h=3 如实）。
2. **binance interval 归真**：`collectBinance` 增拉 `fapi/v1/fundingInfo`
   （公开零凭据）按 symbol join `fundingIntervalHours` 落 interval_h；
   端点挂=venue_down 同径拒。
3. **rejects Door 出观察面**：scout `rej` Snapshot 接现有消费面（metrics
   字段写入 fin_venue_candidate.metrics.rejects 或 scoutv1 Board 行——择一
   不并增），venue_down 不再运行期盲（T-0302 实证坑）。

## 钉注

- 不改推荐判据/阈键/基线式；`top_n`/`persist_ratio`/`recommend_margin_apr` 不动。
- 跨所族判据的「双腿费」口径不变；settles 分列只动年化量纲。
- interval_h 已存历史行照旧（0=回退径兜住）；新采 binance 行起归真值。
- 测试：分所 settles 桩（HL=1→24、binance=0→3 回退）、跨所差式、fundingInfo join。

## 施工回填（devin-t0304 @ 2026-11-03，分支 t/0304-scout-metrics）

- ①settles 分所：loadVenue 行载 {rate,iv}（dayRow）；settles(iv)=24/iv（iv>0），
  缺/0→config settles_per_day 回退；carry 用本所 settles，OKX md_funding NULL
  →回退 8h=3 如实。
- ②跨所差式分列：canon×day join 后 `|rate_v×settles_v − rate_okx×settles_okx|
  ×365 − 双腿费年化`——各带本所结算频次，同×3 错比病灶修。
- ③binance 归真：collectBinance 增拉 fapi/v1/fundingInfo 按 symbol join
  fundingIntervalHours 落 interval_h；端点挂=同径 venue_down 拒。
- ④rejects 出观察面：rej.Snapshot() 随 metrics jsonb 写回
  fin_venue_candidate（择 metrics 面非 Board——票面择一不并增）。
- 判据/阈键/基线式零动；历史 0 行由回退径兜住。
- 测试：+2 桩（TestSettlesPerVenue HL=1→24 vs binance=0→3 回退 8× 对照；
  TestXvenueSplitSettles 分列差式手算锚；fundingInfo join=4 归真）6 桩全绿@5434。
