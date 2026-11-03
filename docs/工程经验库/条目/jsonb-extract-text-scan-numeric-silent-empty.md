---
schema_version: 2
id: jsonb-extract-text-scan-numeric-silent-empty
kind: pitfall
risk: medium
status: verified
keywords: [jsonb, ->>, text列, Scan型不配, pgx, 静默空集, 读面, fin_venue_candidate, persist]
scope: [cmd/core, internal/store, jsonb 列读面]
summary: jsonb->>/-> 提取列返 text——Go Scan 进 *float64 型不配，首行 Scan 败 break→整类静默缺席；SQL 直跑正常掩盖真实败点——生产端点实返是唯一终验
first_seen: 2026-11-03
last_verified: 2026-11-03
skills: [协议面取证]
---

## 坑

`metrics->'boards'->0->>'persist'` 返回 `text`——Go 侧 `rs.Scan(&persist *float64)`
型不配，循环首行 Scan 即 break，整类工件行静默缺席（fail-soft 外衣下零日志
可查——`if rs, err := pool.Query; err == nil` 只兜 Query 败不兜 Scan 败）。

## 三教训

1. **SQL 直跑正常 ≠ 代码路径正常**——psql 同款查询 4 行全返，败点藏在
   Scan 型配层；终验锚必须钉**端点实返行数**（GetFlowPulse work_items=4），
   不能钉「SQL 能出数」。
2. `jsonb->>`/`->>'x'` 恒 text；`->'x'` 恒 jsonb。扫数值/时间型前 cast
   `::float8`/`::timestamptz` 或扫 *string 自 Parse——勿裸 Scan。
3. 循环内 `Scan err → break` 静默吞行错误——err 应 log 一次或上行汇总，
   单型不配不该拖死整类且无痕。

## 实证

T-0309 二轮：venue measuring=4 工件行端点实返 0；psql 直跑同款 SQL 正常；
`->>'persist'` text 撞 *float64——`::float8` 一行销。
