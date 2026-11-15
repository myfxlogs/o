---
schema_version: 2
id: cognition-dual-lane-worker-seat
kind: design
risk: high
status: verified
keywords: [双车道, 施工席, o-worker, seat, sem_matrix, 销票权, 认领TTL, 兄弟面谓词]
scope: [scripts/cognition/runbook.md, migrations/, cmd/core/feed_esc.go, ops_event]
summary: 双车道编制——唤知=信号道+工单验收销号权,施工席(o-worker flash-high)=工单专办道:唤知限时件不再被长活施工烧窗;两坑——新seat须fin迁移显式入册sem_matrix授权矩阵(隐式=行被拒),谓词修一处忘兄弟处(stall探s.ts改锚后TICKET_N仍锚t.ts)。
first_seen: 2026-11-15
---

# 双车道施工席——信号/工单分离编制与两坑

## 病灶

唤知限时会话被长活施工烧窗：工单认领→干不完→挂起→滞留探误报
（codework_stall 把「在修」当「滞留」）→再烧窗处置误报——信号面与
施工面挤同一条车道，长活短件互相饿死。

## 钦定制（业主裁面 2026-11-15）

- **唤知**=信号道（升件/矿尸/hotdom 全归）+**工单验收销号权独占**；
- **施工席**（o-worker 单元，flash-high 档）=工单专办道：认领→干→
  `施工席/处置` 完工回执；
- 分工契：①OPEN 桌同读——认领行<2h=在办租约，施工席认领中票
  勿抢勿重复处置；②销票=唤知独占——施工席完工回执是销证候选
  非销据：证据行（sha/部署 ts/实测值）验真→销，散文「已修」→
  追修行打回；③施工席不进信号面。

## 坑一：新 seat 须显式入册 sem_matrix

授权矩阵（fin_0154 seat×event 矩阵）不认隐式席——新 seat 落行
全靠迁移显式 INSERT 入册。施工席上线若无配套迁移行，其处置/
回执写径全被拒，表面看是「席位没干活」实为「行没资格落」。
**新席=迁移件**，不是约定件。

## 坑二：谓词修一处忘兄弟处

stall 探把挂起判定锚从 `t.ts` 改 `s.ts`（挂起行时戳起算缓期）
后，TICKET_N 同族判定面仍锚 `t.ts`——同语义谓词散在多处，
修一处留兄弟=新旧口径并立，一半面认缓期一半面不认。
**兄弟面谓词=同修清单**——凡谓词改动先枚举同语义锚点全改。

## 判据

- 新 seat 上线 checklist：sem_matrix 迁移行 + 该席写域事件名实测；
- 谓词类改动 checklist：grep 同语义锚点（ts 锚/状态词/期限式）全改。
