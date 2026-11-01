---
schema_version: 2
id: app-config-write-sg-primary-only
kind: pitfall
risk: high
status: verified
keywords: [app_config, SG 正本, PG-md 副本, projector, replica, dConfigGone, 副本收割, config 钉位]
scope: [app_config 写径, internal/projector/replica.go, SG o-postgres, OCI o-postgres-md]
summary: app_config 正本=SG o-postgres——任何直写径必须落 SG；直写 PG-md 副本会被 projector 副本同步收割删键（have-want→dConfigGone），重启/重连 Load 读缺→闸回默认态功能静默回潮（T-0276 实证：research.dcd.enabled 落 PG-md 被收割，o-research 重启即复臂）
first_seen: 2026-10-31
last_verified: 2026-10-31
skills: []
evidence_refs:
  - type: code
    source: internal/projector/replica.go:55-68
  - type: ticket
    source: docs/tasks/T-0276-dcd枚举臂关闭.md（审计注记①②+订正段）
---

## 症状

app_config 键直写 PG-md（`o-postgres-md`）后一度生效（Load 读到→闸落），
随后无影无踪消失——进程靠 boot 快照维持，重启/重连 Load 键缺→
`GetBool` err→默认分支，被关停功能静默复活。

## 根因

projector 副本同步把 `app_config` 当纯镜像面：`want`=SG `o-postgres` 全键集，
`have`=PG-md 行集；`have-want` 的键判陈即 `dConfigGone` 删
（`internal/projector/replica.go:55-68`，删后 `pg_notify('config_changed',
'app_config')` 回灌热载）。副本面不接受正本以外写入——直写副本=写进一个
注定被收割的位面。

## 对策

- **app_config 直写径一律落 SG `o-postgres`**（docker exec o-postgres psql），
  副本同步径自灌 PG-md 双侧成账，跨同步周期+重启双实证存续（T-0276 gen=373 案）。
- 票面/提示词写钉位时写正本座标（SG）不写副本座标（PG-md）——T-0276 票面
  执行要点曾误书 PG-md，已 erratum 订正在案。
- 排障口径：PG-md 键「消失」先查 SG 正本有无——副本消失≠丢数据，=正本无此键。

## 回改点

- `docs/tasks/T-0276-dcd枚举臂关闭.md` 执行要点订正段（决策裁 2026-10-31）。
- 后续任何票面含 app_config 写径按本条目座标书写。
