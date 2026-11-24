---
schema_version: 2
id: gateway-enum-unit-mismatch-granularity-pollution
kind: pitfall
risk: high
status: verified
keywords: [timeFrame, 枚举单位, mtapi, PriceHistoryExMany, md_klines, 周棒误标日K, 陈棒界, 静默零产出, mt5_dir_mom]
scope: [internal/mt5, internal/research/dirmom, md_klines]
summary: mtapi 网关 timeFrame 是分钟制非 MT5 原生 ENUM_TIMEFRAMES——按 16408(PERIOD_D1) 请求产周棒落行全周日戳署 '1D'；下游按日粒度设计的陈棒界(56h)对周棒结构性永不满足=评估臂史上零产出且全程静默
first_seen: 2026-11-25
last_verified: 2026-11-25
skills: [mt5-quotes-history]
evidence_refs:
  - type: ticket
    ref: O-38593
  - type: code
    ref: internal/mt5/klines.go
  - type: migration
    ref: md_1107_mt5_klines_d1_purge.sql
---

## 现象

`mt5_dir_mom` 影子评估臂接线进 o-research、输入 `md_klines venue='mt5'
bar='1D'` 有 67011 行，`fin_opp_eval` 却史上 0 行、journalctl 三天零该臂
日志零拍败——臂在跑、数据在、产出恒零。

## 根因（三层叠加，缺一不可复现全病灶）

1. **枚举单位错配**：`internal/mt5/klines.go` 按 MT5 原生
   `PERIOD_D1=16408` 请求 `PriceHistoryExMany.timeFrame`，但 mtapi 网关
   实测为**分钟制**——tf=1440→24h 日棒、tf=10080→168h 周棒、tf=1→1m
   棒（OCI 共享会话实测 2026-11-25）。16408≈11.4 天被归周棒。
2. **粒度污染落库**：周棒（ts 全周日戳，67011 行 dow=0 逐行实证）
   upsert 进 `bar='1D'`——表内无粒度列，下游读者无法自查。
3. **新鲜度界结构性失效+观测面缺席**：臂 `staleBars=56h` 按日粒度设计
   （末确认棒常龄≤40h）；周棒确认棒 ts=周日开盘时刻，成为「末确认棒」
   时恒 ≥7 天=168h>56h——**任何时刻全宇宙全陈棒**，scan 空卡提交成功
   →`watch.OK()` 每拍照常→armwatch 永不告警；rejects.Door 纯内存计数
   无持久面；scan 无摘要日志——三层观测全哑=「死寂臂不可再生」病灶。

## 处置（O-38593 落地形态）

- 采集端 `mt5FrameD1=1440`（分钟制日棒）+ `nonDailyBars` 落行间距自验
  （末 6 对棒距 >120h 过半=非日签名 WARN 现形——头注「间距自验」承诺件
  补齐）。
- `md_1107` 清污：`DELETE … venue='mt5' AND bar='1D' AND dow=0`——
  周签名谓词只削误标行，日棒（即便先于迁移落库）不误伤。
- 臂侧 `scan` 每拍摘要日志（宇宙/薄棒/陈棒/零波动/判词分布）——
  死寂臂从此 journal 可见。

## 验收

- 桩测：`TestNonDailyBarsWeeklySignature`（周距判非日/日距不判/单道
  长隙不误伤/样本不足不判）+`TestScanE2EMt5Venue`（mt5 宇宙产
  `mt5_dir_mom` eval 行+`cfd_long` 腿原语）。
- 生产实证（修复前）：tf 对拍 1440→24h/10080→168h/16408→168h；
  md_klines 全行 dow=0。

## 教训

- **第三方网关枚举≠上游原生枚举**：proto 注释/example（"240"）与
  skill 能力账都提示单位存疑——接入新 venue 枚举面先做「请求值→落行
  粒度」实测对拍再写常量，注释承诺的自验必须有代码落地。
- **粒度是数据的一等公民**：bar='1D' 只承诺名字不承诺粒度——写径
  应自验落行间距签名（本条 nonDailyBars 即最小实现）。
- **「拍成≠产出」**：armwatch.OK() 挂在 scan 完成而非 eval 行产出——
  全拒/全空拍照样清标；评估臂的活性锚需要「产出分布可见」的摘要行
  兜底（本票补齐 dirmom；同仓其他臂同型风险在案待逐臂核）。
