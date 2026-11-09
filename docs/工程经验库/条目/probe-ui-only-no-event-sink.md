---
schema_version: 2
id: probe-ui-only-no-event-sink
kind: pitfall
risk: high
status: verified
keywords: [ops_event, 升件, 探针, 前端零信任, 唤知环, 事件面, feedesc]
scope: [cmd/core/ops_face.go, cmd/core/feed_esc.go, 健康面/探针类组件]
summary: 健康探针结果只喂 RPC/UI 不落 ops_event=唤知环对面死全盲——
  一切「供人看」的判态都必须有「供机判」的后端事件落水口。
first_seen: 2026-11-09
---

# 探针只上 UI 不落事件面——展示≠发现

## 案情
ops_face `GetSystemHealth` 逐件评 healthComps 鲜度、前端渲染红字——
但判态**仅存活于 RPC 回包**。唤知环（吃 ops_event 升件的唤知机→
认知会话→处置链）对「面死」全盲：xvenue_spread 停写 61min 直到业主
看板报红才有升件——而那条升件还是另一探针路径产出的，非 ops_face。

## 根因
把「给人看的判态」与「给机器处置的信号」混为一谈。前端是展示层、
后端对前端零信任：UI 红字不是发现，事件落水才是。

## 判别式（同类对象一致性审）
写任一「健康/鲜度/异常」判态组件时自问：
1. 判态出了 RPC 回包，有没有一条 append-only 事件落水径？
   （ops_event 升件 / 本域事实表）
2. 状态机在内存（转死一次报）还是每请求重报？——去重同窗必须在库，
   进程重启不丢忆。
3. 复绿有没有自闭径（携升件#id 引证——唤知环 dedup 视=已处置，
   自愈案例零认知开销）？
4. 死而复死同窗去重窗几何？（本件=24h，同 hunger 覆盖失明契）

## 定式
- 升件：`esc_class` 判类册登记词（断流/观测盲…）+`esc_scope`=组件名
  +24h 同窗 dedup（`ops_event` 同窗查）。
- 自闭：恢复行 `升件#<id>` 引证（唤知环未决查询 NOT EXISTS 引证行
  即未决——见 dispatch.sh dedup 径）。
- 修复类动作另落动作账表（md_repair_log 式——journal 会轮转，
  处置席核证要账不要日志）。

## 实证
feed_esc.go（升件环）+ internal/repair/watch.go（修复环）
+ runbook 断流档，2026-11-09 同 commit 落地。
