---
schema_version: 2
id: sem-matrix-offmigration-drift-newschema-gate
kind: pitfall
risk: high
status: verified
keywords: [ops_event_sem_gate, sem_matrix, 迁移外入册, schema 异形, 处置拒写, check_all 测试红, migrations]
scope: [ops_event_sem_gate, ops_event_sem_matrix, fin 迁移链, check_all cmd/core 测试面]
summary: ops_event_sem_gate 只闸「event 已入矩阵」的写——往矩阵加 event 的迁移若不同迁补该 event 生产存量全集，新 schema（只跑迁移链）缺迁外行=闸异形全拒；夹具座亦须归位在册座。
first_seen: 2026-11-11
last_verified: 2026-11-11
skills: []
evidence_refs:
  - type: commit
    ref: fix/intake-matrix fin_0166 首轮 12 测红→补存量对全绿
---

# sem-matrix 迁移外入册差集——新事件入闸致新 schema 与生产闸异形

## 坑形

`ops_event_sem_gate` 触发器只对「event 出现在矩阵」的写行生效（matrix 内 event 才查 seat×note_like 白名单）。
fin_0154 基集不含 `处置` 行——生产的 `(处置,处置席,'')`/`(处置,复审席,'')` 对经非迁移径入册。

后果链：任何新迁移向矩阵插入首条 `处置` 行（如 fin_0166 研究席源普查对）→ `处置` 瞬间成为闸内事件
→ 生产上存量放行对在，无事；**但新 schema（测试库/重建库）只跑迁移链，缺迁移外行** → 全部合法处置写被闸拒
→ check_all 内 12 个 cmd/core 测试同时红，报文是「拒写非法 seat=处置席」——闸在做对的事，差集在矩阵册。

对称坑：测试夹具用从未注册的虚拟座（`'会话'`）写闸内事件——闸前不查随便过，闸后须归位真实座。

## 根因

矩阵真源 = 迁移链（新 schema 只跑迁移）+ 迁外手工行（生产独有）——两面不齐时，
「event 是否入闸」在两 schema 间异形：同一 INSERT 生产过、测试拒。

## 修法/口径

- **往矩阵加 event 的迁移，必须同迁补齐该 event 的生产存量全集**（fin_0166 先例：
  研究席新对 + 处置席/复审席存量对迁外补链，ON CONFLICT DO NOTHING）。
- 加闸前列生产该 event 全行：`SELECT seat,note_like FROM ops_event_sem_matrix WHERE event='<e>'`——
  凡迁外入册者同迁补链（basis 注「存量对迁外入册——本迁补链」）。
- 测试夹具座归位：闸内事件禁虚拟座——用生产在册座（处置写=处置席），
  「闸前不查能过的夹具」在闸后是隐患非豁免。

## 实案

- 2026-11-11 fin_0166 首轮：单行研究席对 → cmd/core 12 测全红（处置写全拒）；
  补两行存量对 + 夹具座归位 → 全绿。check-all 报文模式：`ops_event_sem_gate: 拒写非法 seat=<合法座>`——
  见合法座被拒先查「该 event 的生产矩阵行是否全在迁移链内」。
