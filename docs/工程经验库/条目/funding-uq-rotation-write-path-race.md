---
schema_version: 2
id: funding-uq-rotation-write-path-race
kind: pitfall
risk: high
status: verified
keywords: [md_funding, UNIQUE, 轮换, 结算拍, ON CONFLICT, READ COMMITTED, 迁移竞态, LOCK TABLE, fin_0066]
scope: [internal/data/okxws/store.go, migrations/fin_0066_md_funding_uq.sql, 迁移轮换, 分域切换]
summary: 给 append 表补 UNIQUE 键的两层暗雷——①写径裸 INSERT 撞键（funding-rate 帧 ts=结算拍结算前恒值，逐拍写同键行正是 878k 行体的成因，UQ 一上 collector 写批即 23505）；②去重 DELETE 与建索引两语句间 READ COMMITTED 新快照可见并发新重复行=建索引 23505（迁移重试环）——修=写径 ON CONFLICT DO NOTHING + 迁移内 LOCK TABLE SHARE ROW EXCLUSIVE 原子去重建键
first_seen: 2026-10-27
last_verified: 2026-10-27
skills: [生产取证]
evidence_refs:
  - type: server_fact
    ref: "SG 轮换首挫——o-core boot FAIL（migration 66 fin_0066 建索引 23505）；预评估 GROUP BY 3445 组重复、挫后复测 3948 组（00:00 UTC 结算拍 600+ 行/组持续增长）；回滚 tx 实证 DELETE 877934 行后同 tx SELECT 仍见 184 组新重复（READ COMMITTED 语句级快照）"
  - type: code_ref
    ref: "internal/data/okxws/store.go funding 写裸 INSERT（59 行带）；migrations/fin_0066 修后形（LOCK TABLE+DELETE+CREATE UNIQUE INDEX）"
---

# 给 append 表补 UNIQUE：写径撞键与迁移竞态双层雷

## 坑形

md_funding 878k 行体=病灶本身：funding-rate WS 帧 ts=**下次结算时刻**（结算前
恒值），collector 逐帧裸 INSERT——同键 (venue,inst_id,ts) 每分钟写一行×600 inst
×双写源。fin_0066 补 UNIQUE(venue,inst_id,ts) 时连环炸：

1. **写径层**：collector 裸 INSERT 无 ON CONFLICT——UQ 一落，下一结算拍写批
   整批 23505（采集断流）。
2. **迁移层**：同迁移内 DELETE 去重→CREATE UNIQUE INDEX——两语句间 READ
   COMMITTED 各取新快照，并发写流的新重复行对建索引可见→23505→迁移回滚
   →boot 重试环（core 停摆）。

## 根修

- 写径：`ON CONFLICT (venue,inst_id,ts) DO NOTHING`（D-0030 §六 per-table 键表
  「同键同拍即同值」——设计裁定本就要求写方幂等，T-0167 迁移票漏配写径）。
- 迁移：`LOCK TABLE md_funding IN SHARE ROW EXCLUSIVE MODE` 首语句——去重与
  建键同锁内原子（读不阻、写阻秒级让位）；此后写方新码键在无感。

## 教训

- **给 append 表补键=双面工程**：迁移面（原子去重建键）与写径面（幂等写）
  必须同票同轮换落地，漏一面=轮换即挫。
- **评估迁移影响要用「增长中」的表实测**：静态 GROUP BY 计数会骗（3445 组是
  快照，结算拍下持续膨胀）；回滚 tx 内 DELETE+复测才能暴露语句间竞态。
- 预评估口径写进票面承接段≠机制保障——「去重成本预估」类承接项要在轮换前
  用真实写流演练（本例若预演一次即早暴露）。

## 关联

- D-0030 §六/D-0032 §三-2（md_funding UQ 裁①）
- T-0166 fin_0066 去重成本承接段（本坑即其兑现）
- T-0169 承接段钉序
