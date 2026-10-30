---
schema_version: 2
id: wiring-hardcode-nil-masks-config-domain
kind: pitfall
risk: high
status: verified
keywords: [装配点硬码, rental_payers, app_config, 移植回归, 分类降级, TRX, 静默空集, wiring]
scope: [cmd/core, internal/data/trx, config, trx]
summary: T-0108 移植把 nox「锁定默认名单」装配实参落成硬码 nil——rental_income 分类静默降级 deposit 三天（收益年化低估，业主当场实证）；装配点值必须出配置域+空集断言告警
first_seen: 2026-10-30
last_verified: 2026-10-30
skills: []
evidence_refs:
  - type: ticket
    ref: T-0265
  - type: commit
    ref: t/0265-rental-classify
---

## 事实/症状

- O `cmd/core/trx.go:64` `NewTxsCollector(..., nil, 0.10)`：rentalPayers
  装配实参硬码 nil + floor 硬码 0.10（与 app_config `trx.floor_pct`=8.0
  脱节）。nox 原型 `wire_trx.go:45` 该位=锁定默认 `TBtvnRB`（业主定性
  租金付款方）+env 可覆写——移植时语义丢成「永远空集」。
- **SG 真库实证**：`TBtvnRBgrVE9zZWkWfPvDSiuo65es55555` 同址 84 笔
  `rental_income`（8-01→10-26 迁库旧分类）→10-27 起 3 笔误落 `deposit`
  （82.68 TRX ≈27.5/天）——收益雷达 7d 年化剔该付款方显著低估，
  业主当场实证报障（MANAGE-242 → T-0265）。

## 根因

移植逐参对拍只核「形参位存在」未核「实参语义」——`nil` 编译通过、
运行零报错、分类静默走兜底分支；配置域键 `trx.rental_payers` 从未播种，
nil 又不显形=三天静默错账。硬码值绕过 DB 单源律（无硬编码域值）。

## 处置

T-0265 四件：装配点改读 app_config `trx.rental_payers`（逗号分隔、
getter 每 Poll 热读非 boot 捕获）+floor 同读 `trx.floor_pct` 撤 0.10+
fin_0090 播种键域+名单空→ops_event 告警断言（不静默空跑）+SG 错账
UPDATE 回改 3 行（SELECT 先核对）。

## 律

- 移植/装配参数对拍必须核**实参语义**（默认值/名单/阈值随语言一起搬），
  形参位对拍通过≠语义等价；
- 配置域新键要随票播种（migration ON CONFLICT DO NOTHING），否则读面
  空集静默；
- 凡外部名单/阈值类装配，实效空集必须显形（WARN+告警行），
  「没找到配置」不是「没有配置」。

## 证据

cmd/core/trx.go:64（改前硬码 nil/0.10）、internal/data/trx/collector.go
（getter 化）、migrations/fin_0090_trx_rental_payers.sql、SG
`UPDATE fin_trx_txs` 3 行回改（10-27→10-29）。
