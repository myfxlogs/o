---
schema_version: 2
id: signal-fresh-gate-source-cadence
kind: pitfall
risk: high
status: verified
keywords: [carry_diff, mt5_swap, Fresh闸, 源节拍, 消费失明]
scope: [sigdist, hypothesis, md_signal, 派生信号, 消费闸]
summary: 统一帧龄消费闸对慢源派生信号恒陈失明——闸值<最慢源拍=行随血统戳写即陈，评估臂全史仅撞运气窗；界须=源拍×2边按 kind 自适
first_seen: 2026-11-06
last_verified: 2026-11-06
skills: []
evidence_refs:
  - type: doc
    ref: docs/tasks/T-0383-carry_diff消费闸120s失明-xvenue臂DOA根修.md
  - type: doc
    ref: internal/research/hypothesis/xcarry.go
  - type: doc
    ref: internal/research/sigdist/run.go
  - type: doc
    ref: docs/工程经验库/条目/ts-watermark-vs-stamped-ts-order.md
---

## 事实/症状

xvenue-carry 评估臂全史仅 3 行 fin_opp_eval（10-26/10-29/10-30 皆
rejected），且 3 次恰好撞 mt5_swap 刚写后 <120s 的「运气窗」；同期
md_signal carry_diff 现值 BTC -515bps/ETH +455bps vs 阈 250bps——
机会在桌面，评估臂看不见。表象=机会缺席，实=鲜度闸错配。

## 根因或策略解释

T-0164 血统闸钦定派生行 ts=最旧上游观测（carry_diff.ts=least(fundingTs,
swapTs)）。最慢源 mt5_swap 为 4h 拍（mt5/bridge.go swapEvery）→
carry_diff 行落地即带≤4h 陈戳——消费侧统一 `ts>now()-interval
'120 seconds'` 闸对 4h 源=恒陈视缺席，每 4h 仅 <120s 窗可见。
同族口径自相矛盾实证：写侧读闸 xvenue.go mt5_swap 已 8h，消费侧 120s。

**判则**：凡消费闸守「ts 帧龄」的派生信号，闸值界=最慢上游源拍×2
安全边按 kind 自适，禁全仓统一帧龄——统一闸对慢源派生行=设计性
失明（与 ts-watermark 坑同源：血统戳≠实时戳的下游处处要对拍）。

## 处置

- xvenueCarryScan carry_diff 闸 120s→8h（mt5_swap 4h×2 边，与读侧
  8h 闸同根——派生径注释引证非魔法数）；
- sigdist Fresh/freshByKind 种类自适闸扩 carry_diff→8h（CASE kind
  派生——T-0199 funding_view 当值期同款）；
- 全仓 120s 闸逐一审计（清单落票面——xvenue_spread/depth5/mark/tick
  源拍皆秒级不配修；funding_view 已当值期自适）；
- 写侧零动：血统戳语义不破；超界（9h>8h）仍视缺席——闸收窄非拆。

## 证据

- check: `go test ./internal/research/hypothesis -run TestXvenueCarryStaleVisible`
- result: PASS——carry_diff ts=2h 陈行→评估落行（修前恒零行）；
  ts=9h→零行（界守）；TestXvenueLineage 血统戳断言原样绿
- ref: docs/tasks/T-0383（翻转断言桩锚 + 全仓闸审计清单）

## 反证条件

- check: 派生信号源拍提速（如 mt5_swap 拍改 <1h）
- falsifies_when: 源拍×2 边 < 闸值时统一闸亦不失明——界定义仍须随源
  拍复核（闸值注释引证链断=再失明险）

## 适用边界

适用一切「行 ts=血统/业务语义回填」的派生信号消费闸（md_signal 谱系
为主）；写侧上游闸（mark/tick 秒级流 120s）不受本判则约束——那是对
采集活性的判据非消费界。新慢源 kind 入库时须在 Fresh CASE 登记界。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: xvenue_carry_eval_zero
- class: 断供
- query: SELECT count(*) FROM fin_opp_eval WHERE template='xvenue-carry' AND ts>now()-interval '24 hours'
- threshold: -1（<1=断供——先查 carry_diff 行龄 vs mt5_swap 4h 拍+闸界，再对拍 bridge.xvenue.map 覆盖）
