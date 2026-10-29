---
schema_version: 2
id: pgx-prepared-cache-missing-relation-rows-err
kind: pitfall
risk: medium
status: verified
keywords: [pgx, 预备句缓存, 缺表错滞后, rows.Err, 降级失效, 桩测]
scope: [postgres, testing, cockpit]
summary: pgx pool.Query 预备句缓存命中时，被 DROP 的表/视图不在 Query 层报错——42P01 滞后 rows.Err()，单判 Query err 的「数据缺」降级静默失效零值冒充
first_seen: 2026-10-29
last_seen: 2026-10-29
evidence_refs:
  - type: ticket
    ref: T-0241
  - type: commit
    ref: "294337a"
---
---

## 坑

pgx `pool.Query` 对同一 SQL 文本走预备句缓存：首次 prepare 成功后，后续调用
省 Parse/Describe 直接 Bind/Execute——**被 DROP 的表/视图不会在该层报错**；
错误（SQLSTATE 42P01）滞后到 `rows.Err()`（或 `rows.Next()` 假返回）。

直接后果：`rows, err := pool.Query(...); if err != nil { 降级 }` 形如
T-0241 驾驶舱 missing 判——冷备径（新 prepare）能抓，热径（缓存命中）
静默放行，`Missing` 名录空、「数据缺」降级不触发，零值冒充存活。

## 修

缺数据降级判必双径：`if err != nil` + 迭代后 `if rows.Err() != nil` 两闸
（T-0241 cockpit_face.go seats/tickets 段实证——单查 Query err 桩测
`[]` 空名；补 rows.Err() 判桩绿）。写降级码/桩时默认热径，勿以冷径断言代替。

## 防

- 涉及「查询败→显缺/降级」语义处，一律 `rows.Err()` 收口判；
- 桩测拔径（DROP 表/视图）断言前先暖径（同文本先跑一遍）或直连验错层。
