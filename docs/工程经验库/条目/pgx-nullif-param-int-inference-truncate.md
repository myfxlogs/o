---
schema_version: 2
id: pgx-nullif-param-int-inference-truncate
kind: pitfall
risk: medium
status: verified
keywords: [pgx, NULLIF, 参数推断, integer, numeric, 小数截断, fixture, qty_base]
scope: [go, pgx, postgres, test-fixture, fin_order_leg]
summary: pgx 参数 `NULLIF($n,0)` 里 $n 被 PG 推断为 integer——float64 小数静默截断（1915.63→1915）；小数参数须显式 `::numeric` 转型
first_seen: 2026-11-07
last_verified: 2026-11-07
skills: []
evidence_refs:
  - type: doc
    ref: internal/guard/guard_test.go
---

## 事实/症状

测试 fixture 写 `INSERT ... VALUES(...,NULLIF($8,0))` 传 Go `float64`
1915.63，库中 `qty_base numeric` 实存 **1915**——小数部被静默截断，
且整条链路零报错（INSERT 成功、读出 float64 仍是 1915），断言层才
暴露（期望 1915.63 实得 1915）。

## 根因

`NULLIF($8, 0)` 第二参是**无类型整数字面量 0**，PostgreSQL 对未声明
类型的参数按另一参推断 `$8=integer`；pgx 据此把 float64 值按整型
编码，1915.63 在**驱动编码层**就被截成 1915，根本没到 numeric 列。

## 对策

- 小数参数进 `NULLIF`/比较位：显式转型 `NULLIF($8::numeric,0)`
  （或 `$8::float8`）——参数类型断言钉死，不交给 PG 推断；
- 同类隐形面：任何 `$n` 与整数字面量同现于类型推断位（NULLIF/
  COALESCE/CASE WHEN）都有截断风险——fixture 里小数字段一律带
  `::numeric` 保险；
- 生产径不受影响：schema 列类型上下文（直接 `VALUES(col)` 位）本身
  给参数提供 numeric 推断，本坑只吃「推断位无列上下文」的表达式参数。
