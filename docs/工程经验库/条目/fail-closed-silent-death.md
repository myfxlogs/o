---
schema_version: 2
id: fail-closed-silent-death
kind: pitfall
risk: high
status: verified
keywords: [fail-closed, 静默死亡, 臂静默, 限频 429, 让位循环, 告警通路, research_arm_silent]
scope: [internal/research, 周期/事件臂, 告警面]
summary: fail-closed 让位若无业主通道告警=静默死亡——「引擎活着但从未产出」与「引擎死了」观测上不可分；在编臂必须有 lastOK 锚+超窗 research_arm_silent 告警（T-0117 DCD 枚举臂 0 产出实案）
first_seen: 2026-10-25
last_verified: 2026-10-25
skills: [策略评估与仓位]
evidence_refs:
  - type: server_fact
    ref: "SG 实拍 2026-10-25 12:25Z：md_dcd_product 0 行（建表以来）；boot 日志 [research-dcd] 面 dcd 读败 fail-closed: http 429——两臂 boot 同打 DCD 端点撞限频，让位循环无告警"
  - type: doc
    ref: docs/tasks/T-0117（回执①三缺：Alert 通路/四臂覆盖/uly 归一）
---

# fail-closed 让位无告警=静默死亡

## 病灶形

DCD 枚举臂 boot 即 429（两臂同拍同端点撞限频）→fail-closed 让位→下一节拍
再撞再让——**循环静默死亡**。模板在册、岗在编、心跳在跳（BeatLoop 30s 独立
于产出），`md_dcd_product` 建表以来 0 行；业主 2K USDT 唯一自然受位引擎
（dcd_put 年化 8%+）从未产出过一条评估。系统对「一只在编探员 0 产出 N 小时」
零报警。

## 根因三层

1. **fail-closed 让位=正确判据，错在无产出告警**——让位是单拍正确语义，
   让位循环是系统性死亡，两者观测面必须分开。
2. **log 前缀≠告警**——`log.Printf("ALERT ...")` 不进飞书；业主通道=
   `notify.Alerter.Alert(ctx,kind,payload)`+kindSpec 注册。静默告警自己
   走 log=告警静默（回执①打回第一缺）。
3. **修一臂不修全家**——四臂（funding/dcd/gapscan/hypothesis）同构弱点，
   单点修=下一臂复发（回执①打回第二缺）。

## 处置（已落）

- `internal/research/armwatch.Watch` 共享件：OK()=成功产出拍；
  Check()=超 2×beat 零产出→`Alert("research_arm_silent")` 一次抑刷屏，
  恢复产出清标可再报；kindSpec 注册（title 研究臂静默，actable=true）。
- 四臂同挂：拍首 Check+产出末 OK（beat=各臂节拍键：dcd/gap=scan_interval、
  funding=freshnessMax、hypo=日窗）。
- 桩测锚：TestSilentAlertOnceAndRecover（窗内零警/超窗恰一次/持续静默不刷屏/
  恢复清标再报资格）+TestArmSilentFlagOwnerChannel（业主通道载荷含 humanize
  标题）。

## 坑谱外延

- boot 撞 vs 持续限频不可分（容器重建丢日志）——429/5xx 退避重读件（≤3 次
  指数）先于 fail-closed 让位，瞬态自愈不进让位循环。
- 伴随方言坑：DCD `uly`=指数名（BTC-USD）≠行情 inst（BTC-USDT）——spot 查价
  归一序 uly→+T→+T-SWAP→+-SWAP（okx-api-spec 符号方言同族），否则臂活了
  评估功能仍死（全产品 spot_missing）。
