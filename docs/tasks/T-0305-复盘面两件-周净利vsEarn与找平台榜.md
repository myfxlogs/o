# T-0305 复盘面两件——周净利 vs Earn 指标 + scout 找平台榜卡

## 状态

**施工中**（devin-t0305 @ 2026-11-03；原态=已批）

## 背景（实证）

- `docs/运维/周报口径.md` 钉死「报一个数=周净利−Earn基线」，但 `fin_report` 现
  metrics 无此行；`net` 按 `class='income'` 算——`trade_pnl`/`rollback_cost`
  两 kind 尚无行致 class 归属未验，口径应按 kind 钉死而非 class 顺带。
- `fin_venue_candidate.metrics` 已桥 SG（boards+rejects+measured_at），
  `scoutv1.Board` 服务已挂——前端无消费件，30d 度量窗业主无面可看。
- 带宽/OKX 自校=一次性验收锚（判不上——票面证据定格）。

## 钦定目标

1. **report arm**：weekly（+monthly 同径）metrics 加 `net_vs_earn`——
   `Σfin_cashflow(kind∈{fee,funding_settle,trade_pnl,rollback_cost,spot_buy,
   spot_sell,interest},窗)` − `avg(fin_ledger_snapshot.net_usd,okx,USDT,7snap)
   ×最新md_earn_rate.rate×7/365`（口径文档钉式 SQL 逐字搬，禁改写）。
   `net` 键改同 7-kind 钉式（语义归位——现值巧合等价不算数）。
2. **前端复盘页**：METRIC_NAMES 加 `net_vs_earn`→「净利−Earn基线」；
   新卡「找平台」消费 `scoutv1.Board`——行=venue×family：
   净APR/持续占比/baseline_src/state/measured_at，measuring 所前排。
3. 验收锚：复盘页周报 tab 出 `net_vs_earn` 行且与 SQL 手算一致（对账值截图落票）；
   榜卡出 ≥4 所行含 boards 字段。

## 钉注

- scoutv1.Board 只读面——前端零写径；榜卡纯渲染不引入新状态。
- 不新建 proto/服务（Board 现成）；若前端列宽不够 venue 截断>12 字显示。
- Earn 基线读 md_earn_rate 经 SG 镜像（桥表已有）；快照缺 7 行=如实降精度
  （metrics 加 `earn_baseline_snaps` 回执快照数）。
