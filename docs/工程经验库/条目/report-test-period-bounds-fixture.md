---
schema_version: 2
id: report-test-period-bounds-fixture
kind: pitfall
risk: medium
status: verified
keywords: [期界, BoundsOf, fixture, now-Nmin, 周一, 月1, 测试flake, check_all, weekly, monthly, fin_report]
scope: [internal/ledger/report, 期界窗测试族, go test fixture 锚定]
summary: 测试 fixture 时刻锚「now-Nmin」在周一 00:0x UTC/月 1 号 00:0x 掉出当期窗——net=0 断言漂红；修法=锚 max(各期始)..now 中分均布恒在双窗内。
first_seen: 2026-11-16
last_verified: 2026-11-16
skills: [KB维护, 事实核验]
---

## 事实/症状

`TestWeeklyAnnualized` 在 check_all 2026-11-16 00:04 UTC（周一）
红：`weekly 算式漂移: base=1500 ann=0（期望 1500 / 0.52）`。
fixture 用 `now.Add(-10min..-7min)` 摆入 fin_cashflow，
周一 00:0x 运行时这些时刻落在**上一周**——
`BoundsOf("weekly")` 当期窗 [周一 00:00, now] 内零行 → net=0。

## 根因或策略解释

「相对 now 偏移」≠「期界窗内」——期界窗的起点由日历决定
（周一/月 1/日 0 点），在期界刚开后 N 分钟内运行，
一切 now-Nmin fixture 全掉上一期。同族潜伏面：
monthly 在每月 1 号 00:0x、daily 在每日 00:0x、以及
跨月周（周始>月始，如 Dec 30 周跨 Jan）双窗交错的断言。

## 处置

fixture occurred_at 锚入断言窗而非锚 now：
取 `winStart = max(weeklyStart, monthlyStart)`（双断言时须同进两窗），
行均布于 `winStart + (now-winStart)×(i+1)/6` ——
span 任意小都恒在窗内，跨月周由 max() 覆盖。
实证：report_test.go TestWeeklyAnnualized 改锚后绿。
