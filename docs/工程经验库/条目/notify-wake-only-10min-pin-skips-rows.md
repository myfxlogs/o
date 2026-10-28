---
schema_version: 2
id: notify-wake-only-10min-pin-skips-rows
kind: pitfall
risk: high
status: verified
keywords: [LISTEN/NOTIFY, 非持久, 漏帧, 10min 水印, 投影桥, ev_proposal, outbox, 永久跳过]
scope: [internal/platform/events, internal/projector, 跨库投递面]
summary: NOTIFY 只唤醒不持久，投影以 ts 水印消费且被 now-10min 夹逼——触发器漏帧/通知丢失超过窗口的行不再被常规扫描，永久停在旧值（实案：提案 #405143 落库 10min 桥不动，手推 NOTIFY 即通）；修=outbox+ack 持久游标，10min 夹逼只可作过渡止血
first_seen: 2026-10-28
last_verified: 2026-10-28
evidence_refs:
  - type: decision
    ref: D-0036 §12.1（终席实勘）/§12.2 裁3（outbox 双车道）
  - type: doc
    ref: internal/platform/events/events.go:1-6（NOTIFY=唤醒非队列）
  - type: doc
    ref: internal/projector/reconcile.go:191-200（wm=max(wm,now-10min)）
  - type: doc
    ref: docs/tasks/T-0194-ev_ticks触发器双端灭失回填.md（触发器静默丢失实证）
---

# notify-wake-only-10min-pin-skips-rows——NOTIFY 漏帧 + 10min 水印夹逼=永久跳行

## 事实/症状

- 2026-10-28 终席实勘：md 侧提案 `#405143` 已落库，投影桥 **10min 不动**；
  手工对该行发一次 NOTIFY，立刻过桥。说明行在、码在，**唤醒丢了**。
- 故障不报错：监听注册正常、进程活着、健康检查常绿；只有对拍目标表
  才能看出旧值滞留。低频表（目录/宏观/结算拍）可滞留数小时无人知。
- 同类前科：T-0194 实证表重建/迁移会让桥触发器**静默灭失**——没有通知，
  也没有报错。

## 根因或策略解释

1. `internal/platform/events` 的 LISTEN/NOTIFY 只是**唤醒信号**，不持久、
   不重放；断线/半死/漏发都表现为「这次没人叫醒」。
2. 投影消费以源表 `ts` 为游标，liveness 拍把下界夹到 `now-10min`
   （`wm = max(wm, now-10min)`）——只覆「最近 10 分钟」的断窗。
3. 若漏帧行的时间戳早于该窗口，后续任何常规拍都不会再扫到它；
   全表 id 集差/ts 重拉只在启动 `Reconcile(true)` 或手动触发，**不是常规径**。
4. 于是「非持久唤醒 + 有界回看窗」组合出一个**静默永久丢行**语义：
   不是 at-least-once，也不是 at-most-once，而是「窗口内一次，错过即无」。

## 处置

- 正修（D-0036 §12.2 裁3）：桥改 `bridge_outbox(seq)` + 消费端 `ack` 持久
  游标；唤醒只用于降延迟，不再承担「是否送达」语义。重复 apply 幂等。
- 过渡期：删 `now-10min` 夹逼或持久化水印；漏帧时手推 NOTIFY **只止血，
  不是修复**——下一拍照样丢。
- 防触发器静默灭失：桥域「表×触发器」存在性断言进对账闸（T-0223），
  缺即红；另留 arch/源水印日级回退对拍。
- 排障指纹：目标表最新行 ts 远早于源表、监听日志无断连、手推 NOTIFY 即通
  → 先怀疑本条目，不要先怀疑业务码。

## 证据

- check: `SELECT max(ts) FROM <源表>` vs `SELECT max(ts) FROM <镜像表>`；
  对落后行手工 `NOTIFY` 后观察是否立刻追平
- result: 提案 #405143 落库 10min 未过桥；手推后即时通过（2026-10-28 终席）
- ref: D-0036 §12.1；`internal/projector/reconcile.go:191-200`；
  `internal/platform/events/events.go:1-6`；T-0194

## 反证条件

- check: 在**不跑全表对账、不手推 NOTIFY** 的前提下，停监听 >10min 再恢复，
  观察窗口外写入的行是否自动补齐
- falsifies_when: 窗口外行被自动补齐——说明存在本条目未知的持久游标，
  「错过即永久跳过」结论失效，须改写根因段

## 适用边界

- 适用：一切「LISTEN/NOTIFY 唤醒 + 时间水印回看」的跨库/跨进程投递，
  含投影八桥与后续新增桥表。
- 不适用：同事务内同步调用、由消费端全量轮询且无回看窗的路径。
