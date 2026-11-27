---
id: brokerboard-beatfail-overrides-live-phase
title: 接入板接入态被事件史二次判据盖活性面——beatFailed 并入 chip 致假死显形
kind: pit
date: 2026-11-27
source: 工单O-41377（okx/okx_read 板面 beat=ws-login close 1006 败行冻结显接入败，私有会话实活）
---

# 读面「判态单源」被前端第二判据凿穿（接入板四刀根修实证）

**坑形**：`ListBrokerOnboard` 的 `phase` 字段本就是服务端单源派生
（roster 户=`brokerPhaseRoster(roster.state)`——roster.state=会话环
自报活性面，okxpriv/mt5 reportVenueState 同源；非名册行=beat 末拍
fallback）。但 `brokeronboardlist.tsx` chip 判据写成
`r.phase === "failed" || r.beatFailed`——`beatFailed` 只是
ops_event「venue接入」末行带 esc_class 的**事件史标志**，不是态。
会话环恢复 active 不补写「venue接入」愈行 → 败行永远是末行 →
beatFailed 永真 → 活会话被冻结败行盖成「接入败」假死。

**判别**：板面显「接入败/未识别态」而对应会话环日志显示活——
先查 phase 派生源是否真读了活性面（roster.state/会话心跳），
再查渲染层有没有把事件史标志（beatFailed/末拍注记）并进态判据。
「注记/历史」与「态」是两栏两源，混判=假死/假活双向病。

**修法**：态栏=phase 单源纯渲染（前端零判据纪律本来就钉死）；
beatFailed 降历史注——末拍列显「末败@t」+stale 龄。活性源缺席
（非名册行）才轮到末拍 fallback。词表侧同坑：roster 六态
registered/connecting/active/degraded/disabled/removed 须全谱进
人话词表，「未识别态」只作未知兜底——态映射把六态并成两态
（registered/connecting 同显「验连中」）=词表全开也掩同态。

**推广**：凡「服务端派生态 + 前端补判据」的读面，补判据必须是
同一真源的只读投影，绝不许引入第二事实源（事件史/缓存快照）。
新事件类落账时先问：它会不会成为某行的「末行」并冻结显形？
