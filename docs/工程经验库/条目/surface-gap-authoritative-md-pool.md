---
id: surface-gap-authoritative-md-pool
title: fin_surface_gap 权威队列在 PG-md——fin 池探针读=死表谎报（O-39330 偏差实证）
kind: pit
date: 2026-11-25
source: 工单O-39330 施工（族卡断流探针 spec D 偏差裁定）
---

# 跨域表名≠同库可达——探针挂错池=读死表（O-39330 实证）

**坑形**：spec 附带件要求 hunger 探针（fin 池=SG）顺手盯
`fin_surface_gap` 非终态行龄——按表名直觉以为同库。实证三层：

- `domain.tsv` 钉 `fin_surface_gap → md`；唯一建表件 `md_1044` 在
  **md 链**（PG-md=oci `o-postgres-md`），fin 链零建表零写家；
- 三个写家（`gapscan` 差扫岗/`classify` 分类器/`hypothesis` 假想官）
  全跑 oci research 节点、全走 md 池（`SG_PG_DSN`→127.0.0.1:5435）；
- SG fin PG 上的同名表=**分域前孤儿残留**（schema_migrations 只记
  fin 链 252 版无 md_1044；行 last_seen=2026-10-27 死表一个月）。

对死表写探针=探针「述事实」变「述幽灵」——比不探更坏。

**判别法**：动表前先三查——`domain.tsv` 域归属→建表迁移在哪条链
（`grep -l "CREATE TABLE.*<表>" migrations/{fin,md}_*.sql`）→
生产写家走哪个 DSN（`cmd/*/main.go` 的 pool 来源+部署 env）。
「两库同名表」在分域架构=常态（projector 反向镜像也会造同名影子），
行数相同≠同一张表——抽查行内容/last_seen 分辨真伪。

**连带教训**：跨域需求（fin 池探针要读 md 表）在本架构无现成径——
o-core 只持 fin 池是钦定形（「SG o-core 撤臂钱径独占」）；正解=探针
挂到拥有该域池的节点/器官（gap 断流探针应挂 research 侧 md 池件），
不是给 fin 探针接第二根 DSN。
