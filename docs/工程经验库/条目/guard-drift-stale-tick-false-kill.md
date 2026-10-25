---
schema_version: 2
id: guard-drift-stale-tick-false-kill
kind: pitfall
risk: critical
status: canon
keywords: [guard, delta_drift, md_ticks, 陈数据, 状态表化, 误杀, 新鲜度闸]
scope: [guard 判据链, 事实读面, 状态表模型]
summary: 状态表化后行恒在内容可陈——「行在即有效」读法在断流期拿陈行当真值，一腿陈一腿新=价时差型假失衡误杀真仓
first_seen: 2026-10-25
last_verified: 2026-10-25
skills: []
evidence_refs:
  - type: incident
    ref: 仓 23 delta_drift 误杀——现货 tick 冻 15:58/SWAP 新鲜，17:00:02 被平（真腿量仍平衡基线 1.53bps）
---

## 病灶

md_ticks 状态表模型化（T-0123）后行恒在——`SELECT last` 断言「行存在」
不断言「行新鲜」。断流期一腿陈一腿新=价时差型假失衡；超龄闸判据拿时间
当代理变量同族——事实读面年龄是有效性本身。

## 修法

① 事实读面统一带 ts 回读+age 断言（tick=exec.tick_max_age_s 缺省 120s；
funding=guard.funding_max_age_min 缺省 540min 结算窗尺度）；
② 陈行=缺同 facts_missing 计数径——fail-closed 不判非硬杀；
③ 面级断流（订阅标的齐陈）=ALERT 件非单键静默。
