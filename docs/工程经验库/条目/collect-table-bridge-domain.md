---
schema_version: 2
id: collect-table-bridge-domain
kind: pitfall
risk: high
status: verified
keywords: [domain.tsv, bridge, md 域, fin 链, 采集写表, md_klines, 双链镜像]
scope: [migrations, domain.tsv, collector, SG 写库]
summary: 采集器写表必须 bridge 域——md 域只落 md 库而 SG collector 写库=fin 链，域注错=产线 relation missing write_err
first_seen: 2026-11-13
last_verified: 2026-11-13
evidence_refs:
  - type: ticket
    ref: 13065
  - type: commit
    ref: 625a51d2d
  - type: doc
    ref: migrations/domain.tsv
---

## 采集写表域注律：SG 写库=fin 链，md 域=只落 md 库

- **实证**：md_klines 初注 `md` 域——本地桩迁 fin 域即报
  relation does not exist（fin 链无此表）。SG 侧 okxws collector
  跑在 o-core 内，pool=fin 链库；`md` 域表只随 md 迁移链落
  PG-md 正本，SG 写径必撞 42P01→write_err。采集写表全谱先例
  （md_ticks/md_funding/md_mark_index/md_signal/md_tick_scan）
  无一非 bridge——双链各落同形 DDL。
- **教训**：新采集写表域注判定三问——①谁写（SG o-core 内采集
  器=fin 链库；桥 collector=md 库）②谁读（SG 钱径读=fin 链需
  在场）③bridge=双链 fin_XXXX 镜像迁+DDL 逐字同形（断言⑥
  钉）。「表名 md_ 前缀」与「domain=md」是两件事——md_ticks
  叫 md_ 但域=bridge。测桩迁域应与生产写侧同域（SG 写表桩
  迁 fin 不迁 md）。
- **关联病灶**：projector 镜像谓词侧同理——桥表在 fin 链的
  行是 SG 采集写，md 侧行是桥投影写，同键共写幂等靠
  upsert 同键谓词钉（最新赢/recv_ts 面）。
