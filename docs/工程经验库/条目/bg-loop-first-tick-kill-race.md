---
schema_version: 2
id: bg-loop-first-tick-kill-race
kind: pitfall
risk: medium
status: verified
keywords: [bash, 后台环, 心跳, 首拍, kill, 赛跑, 锚行丢失, 桩测假红, run.sh]
scope: [scripts/cognition/run.sh, 任何「后台循环首迭代产事实行+主径返回即kill」同构件]
summary: 后台心跳/脉冲环 i=0 首拍与主径 kill 赛跑——会话秒亡时首行事实（在办锚/首心跳）可能根本没写就被 kill，桩测呈「行缺」假红像产品 bug；锚行须同步先落再起环
first_seen: 2026-11-23
last_verified: 2026-11-23
skills: []
evidence_refs:
  - type: code
    ref: scripts/cognition/run.sh pulse() 起跑锚同步落+HB 环 i=1 起拍（工单O-32268①）
  - type: experiment
    ref: "run_test 案④：devin 桩秒退→kill 先于后台环首拍 psql 落库→'在办'行缺 FAIL；改同步锚后 PASS=8"
---

# 后台环首拍与主径 kill 赛跑——锚行须同步先落（O-32268 桩测实证）

## 病灶

`run.sh` 原心跳环 `( while :; do sleep 60; hb alive; done ) &` 起后台后
主径立即跑 devin——会话正常长跑无碍。但给环加「i=0 即产首行事实」
（在办脉冲锚）后，**桩测/秒级会话**场景暴露：devin 桩秒退 → 主径
`kill $HB_LOOP_PID` 先于后台 subshell 完成 fork→tail→printf→psql
全链 → 首行 `在办` 永不落库。症状=桩测 `FAIL 在办脉冲行缺`，
trace 里能看到 `+ pulse` 被调但 psql 日志零 INSERT——像 pulse 函数
本身坏，实为**时序竞态**。

## 判据

凡「后台循环负责产事实行 ∧ 主径结束即 kill 循环」同构件：
**第一行（锚/基线）必须同步写在起环之前**，环内只管周期续行。
异步首拍的存活窗口=主径剩余时长——主径越快死，锚越容易丢；
生产长会话掩盖此坑，桩测必踩。

## 修法（O-32268 落地形）

```bash
pulse                      # 起跑锚同步落——秒亡会话也留首行事实
( i=1; while :; do sleep 60; hb alive; [ $((i % N)) -eq 0 ] && pulse; i=$((i+1)); done ) &
```

环内计数从 i=1 起、条件 `i%N==0` → 周期行仍在 t=N×拍距 准点落，
与同步锚零重零漏。

## 教训

- 桩测「行缺」先怀疑**时序**再怀疑**逻辑**——bash -x trace 里
  `+ pulse` 出现≠函数体跑完，subshell trace 与主流交错易误读。
- 事实账的「首行」语义=存在性锚（会话起跑即有在修证据），
  异步化会把锚的存活绑定到会话寿命——锚恰恰是为死会话准备的，
  语义上就必须同步。
