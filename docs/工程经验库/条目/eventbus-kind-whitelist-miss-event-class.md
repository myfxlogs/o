---
schema_version: 2
id: eventbus-kind-whitelist-miss-event-class
kind: pitfall
risk: medium
status: verified
keywords: [eventbus, mechwake, 工单, 白名单, ops_event, 唤起, dispatch]
scope: [cmd/eventbus, scripts/cognition, 唤知环]
summary: eventbus dm/mech 帧按事件词白名单分帧；纪元3新增事件类（工单等）漏进名单时 INSERT 不出帧、唤臂收不到信号，票只能蹭 15min 兜底拍——事件类新增必过帧谱名表审
first_seen: 2026-11-13
last_verified: 2026-11-13
skills: []
evidence_refs:
  - type: doc
    ref: cmd/eventbus/main.go
  - type: doc
    ref: scripts/cognition/dispatch.sh
---

# eventbus 帧白名单漏事件类——「工单落板不唤」静默缝

**坑形**：eventbus 唤醒谱按事件词表白名单分帧（d 源 dm=收口/移交/收口待审/升件/
诊断草案；e 源 mech=已批/派单/裁决/处置/部署/起草）。纪元3 新增 `工单` 事件类后
未进补录——票 INSERT 不产生 mech 帧，`o-mech` 唤臂收不到信号，只能靠 15min
timer 兜底拍。实测 r10 工单 00:48 立、00:52 唤是碰巧临近拍点，最坏延迟一拍。

**判别**：某类事件落账后唤件「不立即动但迟早动」＝漏名单而非唤臂死。
对账法：现存 event 类全集 ∖ 帧谱词集 = 静默类清单。

**修法**：白名单补事件类（本次：e 源加 `工单`）。唤件幂等——dispatch.sh
阈面/单飞闸自挡，误唤=空拍毫秒 SQL，漏唤才是延迟债。

**推广**：新增 ops_event 事件类时必须同时审两张词表：dispatch.sh 阈面
（算不算触发条件）与 eventbus 帧谱（出不出帧）。不唤的类要显式注明裁定，
不许默认缺席。
