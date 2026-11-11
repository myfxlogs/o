---
schema_version: 2
id: organ-state-fact-feed-class-blind
kind: pitfall
risk: medium
status: verified
keywords: [ops_event, 事件类, 推导面, 采样偏差, 毛坯工位, ghostSt, floor]
scope: [web/src/react/pages/floorFocus, cmd/core/panelv2_trail, dashboard floor]
summary: 工位态推导只喂升件类尾流——器官事实散在工单/处置/会话收口四类，升件窗外全盲；同类病灶=统计继承测量 bug
first_seen: 2026-11-11
last_verified: 2026-11-11
skills: []
evidence_refs:
  - type: ops_event
    ref: "#10507 会话收口（法典已部署销——旧禁条作废完工证不在升件类）"
  - type: ops_event
    ref: "#10349 工单（A-D 四件包——B/C/D 开工证不在升件类）"
  - type: ops_event
    ref: "#10503/#10505 处置（自态账/四器官领命——不在升件类）"
---

## 病灶

floor 毛坯工位态推导初版喂 `GetDecisionTrail.trail`（升件类最新 12 行）——
器官事实实际散在 `工单`/`升件`/`处置`/`会话收口` 四类事件：

- 旧禁条作废完工证 #10507 是 `会话收口`——永不在推导面里，恒显在建；
- B/C/D 开工证 #10349（工单）/#10503（处置）随升件流滚出 12 行窗即退化毛坯。

**事件类切片=采样偏差**：对 ops_event 推导时先问「事实会落在哪个 event 类」，
单类截取只在确知全集同类时才合法——与「统计继承测量 bug」同族。

## 修法

- `GetDecisionTrailResponse.facts`：ops_event 全事件类近窗行（LIMIT 600），
  前端 ghostSt 改吃 facts 不吃 trail；
- 判词取**关键词 ±25 字邻域**——一行事实可带多器官不同定性
  （#10507 同注「法典已部署销」∧「自态账领命」），整行 DONE_RE 会串味；
- 聚焦卡/悬停挂「据 #N」事实锚——推导态可指回 ops_event id。

## 复发信号

- 某工位态与其 ops_event 事实不符 → 先查事实行的 event 类在不在数据源切片里；
- 一行多对象的行级判词 → 改词邻域局部判。
