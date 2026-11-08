---
schema_version: 2
id: psql-age-single-arg-midnight
kind: pitfall
risk: medium
status: verified
keywords: ["psql", "age函数", "单参", "午夜", "负龄", "时钟歪斜假象", "事件龄对拍", "ops_event", "T-0459"]
scope: [psql, ops_event, 事件龄审计, 哨兵桩测对拍]
summary: psql `age(ts)` 单参=自当日午夜（current_date−x）非自 now()——午后查上午行得负值，形似「行落在未来/时钟歪斜」假警报。事件龄一律用 `now()-ts` 或 `extract(epoch from now()-ts)`；age(ts,now()) 双参才=now−x。
first_seen: 2026-11-08
last_verified: 2026-11-08
skills: []
evidence_refs:
  - type: server_fact
    ref: 2026-11-08 T-0459 干跑对拍：`age(ts)` 报 T-0447 派单行 −01:34:42 误判未来时刻；`extract(epoch from now()-ts)` 同行=+2329s 与哨兵扫描一致（DB now()=02:13 UTC，行 ts=01:34 UTC）
---

# psql age(ts) 单参是自午夜不是自 now——负龄=时钟歪斜假象

## 病灶

事件龄对拍写 `SELECT age(ts) FROM ops_event ...`，午后查询上午落的事件行得负区间
（如 `-01:34:42`），形态与「写入方时钟快 1.5h」的歪斜假象完全一致——
验哨兵干跑命中时差点据此误判生产数据异常。单参 `age(x)` 语义= `current_date − x`
（自当日 00:00），非 `now() − x`。

## 正形

```sql
-- 事件龄（与哨兵/量表判据同式）
SELECT extract(epoch from now()-ts)::bigint FROM ops_event WHERE id=3758;
-- 或
SELECT now()-ts FROM ops_event WHERE id=3758;   -- 返回 interval 自 now
```

## 判别锚

同行的两种查法互相矛盾（age 负 vs extract 正）且 `now()` 与 `ts` 时段肉眼可比对
（now=02:13，ts=01:34， plainly 过去）→ 是查询语义错不是数据错。
