---
schema_version: 2
id: nav-usd-aggregate-vs-native-ccy-dual-layer
kind: pitfall
risk: high
status: verified
keywords: [NAV, md_equity_curve, 按币簿记, 聚合读面, 读件缺, nav_stale, idle_capital, GetEquityCurve]
scope: [cmd/core navFn/moneyRows, internal/ledger/report navSnapshot, md_equity_curve 消费面三读件]
summary: 簿记按币化废 ccy='NAV' 单行时把 USD 聚合读面一并删——簿记层与读面层是两件东西;改写者前必须枚举全部消费查询再动。
first_seen: 2026-11-11
last_verified: 2026-11-11
skills: []
evidence_refs:
  - type: commit
    ref: 24e7905ad（按币簿记落地连带删 NAV 行写者）
  - type: commit
    ref: fix/nav-restore（navFn 补写 by["NAV"]=NavTotal）
  - type: esc
    ref: 升件#10438 数据缺/nav_stale
---

# NAV 聚合读面 vs 按币簿记是两层——删写者前查消费面

## 坑形

业主令「资金统计按币计不折换」落地时，实现把 `ccy='NAV'` 单行折换值写者整个换成按币分行。
结果：簿记按币行（USDT/BTC/USD）正常续写，但三个**读面**静默死亡——

- `deployEscCheck` idle_capital 判据查 `ccy='NAV' AND ts>48h` → 恒缺=永不判（fail-closed 设计态，
  无告警无报错——盲了 2 天零觉，nav_stale 探针补位才发现）；
- `owner_beat` 盈利报告年化%vs目标8% 行 → 同缺省省略；
- `GetEquityCurve` 驾驶舱 NAV 日桶曲线 → 11-09 起零点后空。

「簿记按币不折换」管的是**写面口径**（每币原生量各行）；「USD 聚合数」是**读面索引**
（探针判据/目标对比/曲线轴——业主的 8% 年化目标本身就挂在 NAV 数上）。两层并存不冲突：
native 行=簿记真源，NAV 行=聚合读面。

## 根因

写者改造只看「该写什么样」没看「谁在读什么」——消费面三查询都点 `ccy='NAV'`,
写者一停全部 fail-closed 静默跳过，设计上的「陈值=不判」从防呆变成致盲。

## 修法/口径

- **改时间序列写者前：grep 全仓该序列的消费谓词**（ccy/venue/src/kind 键组合），
  逐消费面问「我删的它还需要吗」；fail-closed 消费面最阴险——停写=无报错全盲。
- 簿记口径令与聚合读面分层记：native 行与聚合行共存（navFn `by["NAV"]=NavTotal` >0 才写,
  零值不落防假零）。
- 探针陈值语义区分「序死」与「源死」：nav_stale 探针读 ccy='NAV' 判的是序列活性——
  序列本身被退役的,探针口径要跟着换（或序列恢复——本案选恢复,三读件全复活）。

## 实案

- 2026-11-09 24e7905ad 按币簿记落 → 11-11 10:1x nav_stale 探针升件（断供 2d 零觉）
  → fix/nav-restore 一行补写恢复全读面。
