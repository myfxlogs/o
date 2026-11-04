---
schema_version: 2
id: fixedzone-fallback-dst-drift
kind: pitfall
risk: high
status: verified
keywords: [tzdata 缺席, LoadLocation 兜底, FixedZone 固定 offset, DST 冬令时漂移, alpine 镜像, America/New_York, 静默错 1h]
scope: [alpine 容器镜像, notify/calendar, research/ctx, Go time.LoadLocation]
summary: LoadLocation 兜底 time.FixedZone 固定 offset 对 DST 时区=定时炸弹——alpine 镜像缺 tzdata 使兜底成主径，冬令切换(-4→-5)后全部事件时间静默错早 1h 零报错；兜底件只能是缺件备位，生产镜像必须 tzdata 真命中
first_seen: 2026-11-03
last_verified: 2026-11-04
skills: [ctx-sources]
evidence_refs:
  - type: ticket
    ref: T-0296
  - type: doc
    ref: Dockerfile.core
  - type: doc
    ref: internal/platform/notify/calendar.go
---

## 事实/症状

`Dockerfile.core` `FROM alpine:3.20` 无 tzdata 包（容器内
`/usr/share/zoneinfo` 缺席实证）→ `calendar.go` etLoc 兜底
`FixedZone("ET", -4*3600)` 成主径。美国 2026-11-02 入冬令（EST=-5）后
FF 历表全部事件时间解析错早 1h——推送提前 1h 且错开 [T-lead,T] 预推窗
可致漏报。10-30 前推送准时只因彼时尚 EDT(-4) 恰合兜底值——**兜底正确
是巧合非保证**，DST 切换即爆。

## 根因或策略解释

`time.FixedZone` 固定 offset 本质是「冻结的时区快照」——对实行 DST 的
zone 名（America/New_York 等）只在快照季内偶然正确。镜像缺 tzdata 时
`LoadLocation` 静默走兜底（err 分支不报警），病灶零报错零告警纯靠业主
发现推送时间不对——静默兜底比直接炸更凶险。同调用点
`internal/research/ctx/collect.go:85` 同病（兜底 -5h 则夏令错晚 1h）。

## 处置

T-0296 根修：`apk add --no-cache tzdata` 进 alpine 镜像（裁 apk 非
`_ "time/tzdata"` embed——双调用点分属 notify/research/ctx 包，apk 整
镜像面一件罩全，embed 只罩 import 它的二进制）；etLoc 兜底按票面保留
作缺件备位。单测 `TestFFTimeDSTBoundary` 钉冬令 -5h/夏令 -4h 边界+
etLoc offset=-18000 断言防兜底回潮。

## 证据

- check: `TestFFTimeDSTBoundary`——LoadLocation 真命中，2026-11-05
  8:30am ET=13:30 UTC（-5 冬令），07-15=12:30 UTC（-4 夏令对照），
  etLoc 冬令 offset=-18000。
- result: 全绿。
- ref: T-0296 票面件③（容器内 /usr/share/zoneinfo 缺席实证在案）。

## 反证条件

- check: 部署后容器内 `ls /usr/share/zoneinfo/America/New_York` 存在；
  下一 ET 事件推送时间与票面正确窗对拍（如 AUD RBA 07:30–08:30 UTC）。
- falsifies_when: tzdata 在镜像内仍走 -4h 兜底=LoadLocation 调用序有别
  病灶；DST 边界用例败=ffTime 解析径另伤。

## 适用边界

凡 `time.LoadLocation` 使用面（DST zone 尤甚）+一切 scratch/alpine/distroless
系最小镜像——缺 tzdata 是默认态非异常，FixedZone 兜底只能兜「zone 文件
缺件」不能兜「DST 规则」。
