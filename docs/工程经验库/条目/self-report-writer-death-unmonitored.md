---
schema_version: 2
id: self-report-writer-death-unmonitored
kind: pitfall
risk: high
status: verified
keywords: [hostmetrics, fin_host_metrics, 自报写器, 停拍, 静默死, 尸体数据, 观测断供, pool nil, Upsert 失败, 对机互探, selfprobe]
scope: [hostmetrics, 监控自监, 观测面, 前端渲染, ops_event 升件]
summary: 自报写器（hostmetrics.Run）死亡=观测断供零告警——'sg' 行冻结 8 天（10-30 04:18→11-07）前端照渲尸体数据，靠业主人工对拍发现；写器三设计缺陷（pool nil 一次永退/Upsert 败仅 log/无陈旧探针）+渲染契约缺硬闸。T-0434 四件根修。
first_seen: 2026-10-30
last_verified: 2026-11-07
skills: []
evidence_refs:
  - type: doc
    ref: internal/platform/hostmetrics/hostmetrics.go（T-0434 根修后：poolFn 退避环+连败升件+ProbeStale）
  - type: doc
    ref: internal/platform/selfprobe/selfprobe.go（自报表陈旧→升件公共模子——feedstale/ctx/hostmetrics 三实例归一）
  - type: doc
    ref: docs/tasks/T-0434-hostmetrics监控自监-写器死活根修与前端陈旧硬闸.md
---

## 事实/症状

fin_host_metrics 'sg' 行 ts 冻结 2026-10-30 04:18:44 UTC（dev 栈库），8 天无新拍零告警；前端 system 页照常渲染尸体数值；业主人工对拍 df 发现（「前端不能展示真实数据，这是很大的一个灾难」）。同期同库 ops_event/业务面正常——唯观测自报死。

## 根因或策略解释

死因实锤（T-0434 考古）：**写器宿主进程死亡且无重启**（dev 栈 o-core 10-30 04:18:44 UTC 消失，8 天无进程复活）——非 goroutine 级死亡。但票面点名的三个设计缺陷全部实证为真，任一命中即同形静默死：

1. `pool==nil→return` 一次永退——启动时序抖动即永世断供（防死码，但形在）；
2. Upsert 失败仅 log——「观测面缺不拖业务面」钦定静默的副作用=断供无感；
3. 本表无陈旧探针——「盯人的探头没人盯」（feedstale/ctx-probe 同族第三面缺席）；
4. 前端渲染契约无硬闸——服务端 stale 标已回，尸体照渲（T-0173 起码 stale 藏值，但断流僵尸帧/旧 dist 不设防）。

静默死族谱：dispatch-spawn-silent-failure → feed 断更（T-0410）→ ctx 断供（T-0429）→ 自报停拍（本条）——第五条实证。

## 处置

T-0434 已修（verified）：

1. 写器 poolFn 惰性取池+退避重试环（未就绪不永退）；
2. Upsert 连败阈（3 拍）→ops_event 观测盲升件（24h 同窗去重）；
3. fin_host_metrics 停拍探针（阈 10min，selfprobe 公共模子第三实例）——o-core/collector 双挂对机互探：任一写者进程死=对机活者代吼，进程死不再逃逸观测；
4. 前端渲染硬闸：stale 行数值区整块置灰+红标「数据陈旧·非实时」+无行显式无数据态+客户端 2min 龄复核（僵尸帧同闸）。

**出生律候选（留外审判）**：新自报机/自报表无停拍探针行=交付不齐。

## 证据

- check: dev 栈库 `SELECT * FROM fin_host_metrics` → 唯一行 'sg' ts=2026-10-30 04:18:44，disk 449.6/252.0（dev 盘值）
- check: SG 栈 o-core 容器日志 02:12 起零 [hostmetrics] 失败行（写器健康对照面）
- check: `go test ./internal/platform/hostmetrics/ -run 'TestRunPoolNilThenRecover|TestRunUpsertFailEscalation|TestProbeStaleEscalationOnRolledBackTs'`（三断言桩）；web/scripts/hostmetrics-domtest.mjs（渲染契约桩）

## 反证条件

- falsifies_when: 出现写器死亡且 12min 内 ops_event 落 HOSTMETRICS-* 升件行——探针网住即非坑

## 适用边界

进程死+对机活=可吼；**全栈俱死（dev 栈无 collector 对机）仍盲**——栈级监察归 systemd/compose restart 策略+外审巡检面，本条不覆盖。
