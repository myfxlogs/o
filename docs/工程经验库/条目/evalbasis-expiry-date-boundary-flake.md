---
schema_version: 2
id: evalbasis-expiry-date-boundary-flake
kind: pitfall
risk: medium
status: observed
keywords: [check_all, flaky 测试, TestEvalBasisMiddleBand, 日界窗, time.Until, 到期天数, basis]
scope: [internal/research/grammar, scripts/check_all.sh]
summary: TestEvalBasisMiddleBand 夹具按 30d 期差定价但 eval 以 time.Until(到期日0点UTC)实算天数——日内流逝使 days<29.63 时 net_apr 越 0.049 断言上界，每日约 09:00–24:00 UTC 窗内确定性假红（master 同败，非负载型）
first_seen: 2026-11-05
last_verified: 2026-11-05
skills: []
evidence_refs:
  - type: commit
    ref: run_test.go:220-242 TestEvalBasisMiddleBand（AddDate(30)命名+0.048–0.049 断言带）
  - type: server_fact
    ref: 2026-11-05 09:31 UTC master /opt/o 复跑 net_apr=0.049048>0.049 FAIL（worktree 同败 0.049048——非本票 diff 致）
  - type: doc
    ref: internal/research/grammar/basis.go:99-115 days=time.Until(exp).Hours()/24
---

## 事实/症状

check_all 全谱 go test 段红：`TestEvalBasisMiddleBand`——
`4.84% basis 应在 4% 档 proposed；got verdict=proposed reason= net=0.04905`。
verdict 方向对（proposed），仅 net_apr 越断言上界 0.049 一线（0.048–0.049 带）。

## 根因

夹具 inst 名 = `time.Now().AddDate(0,0,30).Format("060102")`（30 天后某日），
fut 价按「恰 30 天」基差定价（`0.0032+0.0484*30/365`）；而 eval 内
`days=time.Until(exp).Hours()/24`（basis.go:102）——exp 解析为该日
**00:00 UTC**，故 days 全天 <30 且随日内流逝单调缩。net_apr ∝ 1/days
→ 日内越晚 net 越大。断言窗 0.048–0.049 对应 days∈[29.63,30.5]，
约每日 **09:00–24:00 UTC 确定性假红**，00:00 UTC 翻日（AddDate+1）
后复绿。与负载无关——master 同日同时刻复跑同败实锤（区别于
checkall-parallel-load-flaky-timing 的 CPU 争用型）。

## 处置

- 判型：红时先算窗——`date -u` 在 09:00–24:00 UTC 内 + master 单测
  复跑同败 = 本坑定性，非施工 diff 致（勿误判自身票域）。
- 解阻塞三径（归决策席裁，施工勿搭票超域改）：①绿窗（00:00–约08:50
  UTC）复跑 check_all 补戳；②立项根修——断言带改 days 不变式
  （fut 按 eval 实测 days 反算定价，或断言放宽+窗注记）；③急单
  裁豁免合入（戳缺=半痕，决策席自担）。
- 复发面：凡 `time.Now().AddDate` 命名到期 + `time.Until` 实算天数
  的断言带测试同患——T-0298 三档件（run_test.go 同族 10/90d 档
  离界远暂未发，窗口更窄但同构）。
