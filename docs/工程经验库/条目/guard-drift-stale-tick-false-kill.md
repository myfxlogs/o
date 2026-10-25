---
schema_version: 2
id: guard-drift-stale-tick-false-kill
kind: pitfall
risk: high
status: verified
keywords: [delta_drift, 误杀, tick 陈旧, 新鲜度检查缺失, guard, 现货腿]
scope: [internal/guard/, md_ticks, 平仓判据链]
summary: guard 漂移判据读两腿名义=qty×tick，只查 tick 存在不查新鲜度——现货腿 tick 陈死 1h+SWAP 腿新鲜时混读出 ~48bps 假失衡超 20bps 闸带，真仓 #23 被误杀（两腿实量仍平衡）；陈数据比缺数据更危险
first_seen: 2026-10-25
last_verified: 2026-10-25
skills: [生产取证]
evidence_refs:
  - type: production_fact
    ref: "仓 23（ETH funding-hedge $50）17:00:02 被 delta_drift 平——取证：现货 tick 死於 15:58（全现货行齐冻）而 SWAP tick 新鲜，混读价失衡 ~48bps 触发；开时基线 1.53bps、两腿量 0.01/0.009993 ETH 实仍平衡。前 3 秒日志『全判据无触发』→tick 一刷即杀，时间链吻合"
---

## 病灶

`internal/guard/guard.go` 漂移判据：`notionalNow` 只判 tick **存在**，
不判**新鲜度**。md_ticks 是状态表原地覆盖——陈行合法存在，读败才走
facts_missing 跳过。一腿陈一腿新时失衡=(价时价差)，价格动几个 bp 就
越过 hedge_band → 误杀。

本案第二层病灶：T-0122 全宇宙订阅后**现货 tick 面整体死亡**（全部现货
行冻结 15:58:02）——疑 spot 品种订阅渠道改造后断流（index-tickers 与
tickers 分工错位或批帧丢失），guard 却对陈现货价照单全收。

## 判据

- **写判据**：凡用行情价算风控判据，tick 必带 age 断言（如 <120s），
  陈则等同缺失走 facts_missing 跳过——fail-closed 语义含「陈=缺」。
- **评审**：判据链只查存在性=半 fail-closed；md_* 状态表模型下「行在」
  只证明「曾收到过」，不证明「现在有效」。
- **诊断**：误杀疑案先比两腿 tick 时间戳再推码——fin_guard_close 只有
  trigger_kind 无实测值，取证靠 tick 快照侧写。

## 根因

fail-closed 设计成「缺失拒判」但没覆盖「陈旧」形态——追加表模型下
行缺=真缺，状态表模型下行恒在、内容可陈，判据的失效面随之从「无行」
搬到「行龄」，判据代码没跟上模型迁移。
