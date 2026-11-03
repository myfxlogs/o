---
schema_version: 2
id: research-pool-md-canonical-bridge-not-fin
kind: pitfall
risk: high
status: verified
keywords: [研究臂, o-research, PG-md, 桥表, bridge, domain.tsv, relation does not exist, scout, fin_venue_candidate]
scope: [internal/research, cmd/research, migrations, OCI盒]
summary: 研究臂主池=PG-md（OCI盒:5435，DSN名残SG_PG_DSN）——研究域自产表须钉bridge正本PG-md，钉fin域=SG单边建表、臂体relation not exists
first_seen: 2026-11-03
last_verified: 2026-11-03
skills: []
evidence_refs:
  - type: server_fact
    source: postgres
    query: SELECT relname,relkind FROM pg_class WHERE relname='fin_venue_candidate'
    observed_at: 2026-11-03T05:26:00Z
    server_revision: "OCI盒 PG-md :5435"
  - type: doc
    ref: docs/数据流.md:32
  - type: commit
    ref: 10fc3edc
---

## 事实/症状

T-0300 把 `fin_venue_candidate` 钉 fin 域（fin_0098 在 SG 建表+seed），但
scout 臂的 `pool`=研究主池（OCI 盒本机 PG-md :5435——env 键名残
"SG_PG_DSN" 实指盒库）。2026-11-03 部署重启后 scout 每拍败：
`relation "fin_venue_candidate" does not exist (SQLSTATE 42P01)`。

## 根因或策略解释

研究臂（o-research/cmd/research 全家：funding/grammar/scout/dcd/gapscan/
hypothesis）主池=PG-md；研究域自产表（fin_opp_eval/fin_opp_template/
venue_candidate 类）正本钉 `bridge`（PG-md 正表，SG 投影镜像给 core 面），
**不是** `fin`（fin=钱径域正本，SG o-core `Migrate(fin)` 单边落）。fin 域表
研究臂只可经 FIN_PG_DSN 双池读（T-0172 review 范式）或桥投影（projector
码面枚举频道，加表须改码）。

## 自检

新表进研究臂读径前：`rg 'FROM <table>' internal/research/` 对 domain.tsv
域判——bridge/md 域=主池可达；fin/bridge-reverse=需双池或桥投影。

## 处置

T-0301（scout fin_venue_candidate fin→bridge 返修单，含 seed 同源+projector
频道）。部署实证坑——迁移域判勿望表名"fin_"生义（fin_opp_eval 也是 bridge）。
