---
schema_version: 2
id: large-table-dedup-needs-fk-index-first
kind: pitfall
risk: high
status: verified
keywords: [大表去重迁移, FK 无索引, NOT EXISTS 嵌套循环, 迁移长跑, 容器 unhealthy 排队]
scope: [migrations/, 任何对被外键引用大表的 DELETE/去重迁移]
summary: 对被引用大表做去重 DELETE 的迁移，若引用侧 FK 列无索引，NOT EXISTS 引用检查退化为「每删一行全表扫引用表」——523k×181k 嵌套循环实测 33min+ 未竟；先补 FK 列索引再跑迁移=分钟级落账
first_seen: 2026-10-25
last_verified: 2026-10-25
skills: [生产取证]
evidence_refs:
  - type: production_fact
    ref: "T-0126 部署实证：fin_opp_eval 523k 行 dedup 迁移（fin_proposal.eval_id 无索引、18.1万提案行）——迁移器 33min 未竟、o-core boot 排队 unhealthy；停容器→补 fin_proposal(eval_id) 索引→重跑 ~90s 落账，表 306MB→4.8MB"
---

## 病灶

迁移 `0053_eval_state_table.sql` 的 DELETE 形态：

```sql
DELETE FROM fin_opp_eval e
WHERE ... AND NOT EXISTS (SELECT 1 FROM fin_proposal p WHERE p.eval_id = e.id);
```

`fin_proposal.eval_id` **无索引**且提案表 18.1 万行——候选删除行 52.3 万，
逐行做引用检查=嵌套循环 523k×181k。pg_stat_activity 实测：迁移事务
active 33 分钟未竟（无锁等待=纯 CPU/IO 在烧）；同文件排队的 o-core
boot 迁移锁等 28 分钟→容器 unhealthy（是排队不是崩）。

## 判据

- **写迁移**：凡迁移含「大表 DELETE + 引用存在性检查」或「FK 重指
  UPDATE」，先查引用侧列有无索引——无则迁移内/前置先补索引，或改用
  hashable 写法（`IN (subquery)` 物化集合 / join 反连接）。
- **诊断**：大事务长跑 ≠ 卡死——先 `pg_stat_activity` 看 wait_event
  （Lock=被挡、NULL=在烧 CPU），再核执行计划假设（EXPLAIN 同形语句）。
- **排障序**：容器 boot unhealthy 先查其迁移会话是 active 还是 Lock
  等待——勿直接 kill/restart（本会话实证：先补索引再放行=90s 收工）。

## 根因

FK 引用检查是声明式约束的暗面：删侧表时数据库用引用侧列反查，无索引
即把 O(n) 的删放大成 O(n×m)。迁移文件评审时习惯性只看被改表自身，
引用侧索引需求隐形——本法典条把「引用侧索引」升为迁移评审必查项。
