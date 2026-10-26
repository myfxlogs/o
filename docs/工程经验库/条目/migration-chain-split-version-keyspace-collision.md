---
schema_version: 2
id: migration-chain-split-version-keyspace-collision
kind: pitfall
risk: high
status: verified
keywords: [迁移分链, schema_migrations, 版本键空间, md_1000, 撞号, 双链并集, create_hypertable, add_retention_policy, D-0030]
scope: [migrations/, internal/store/pgstore/migrate.go, 分域迁移, check_migrations]
summary: PG 分链（md_/fin_）后同链双跑于一个 schema 时 schema_migrations 版本号键空间相撞——fin_0044 已录版本 44 会把 md_0044 整件跳号（表缺位炸在后面的迁移）；修=md 链版本整体 +1000 与 fin 键空间全局不相交；同坑族：跨链重放时序（瞬态超表化撞状态表末态、add_retention_policy 重复调用即红）
first_seen: 2026-10-27
last_verified: 2026-10-27
skills: [生产取证]
evidence_refs:
  - type: code_ref
    ref: migrations/ 文件名 md_NNNN（NNNN=历史号+1000）与 md 链头注；internal/store/pgstore/migrate.go Migrate(domain) 前缀选链
  - type: server_fact
    ref: o_test 实证——md/fin 同 schema 双跑：md_0051 报 fin_capability_surface 不存在（fin_0044 已占版本 44 致 md_0044 被跳）；timescale add_retention_policy 重复调用报 "retention policy already exists"
---

# 迁移分链后版本键空间相撞——同链双跑必撞，md 链 +1000 错位根修

## 坑形

T-0167 把单一迁移链分拆成 md_/fin_ 双链（D-0030 §六分域）。设计时用**同名同号**
拆分件（md_0044/fin_0044 各持本域语句）——单库单链语义下没问题，但**测试面跨域
包（研究/复盘/earn）需要两链并集态落同一个 schema**，此时 `schema_migrations`
的版本号是**全局键空间**：fin 链先跑录下版本 44，md 链跑到自己版本 44 时按
「已应用」跳过——**md_0044 的建表语句从未执行**，表缺位炸在后面的迁移
（md_0051 INSERT 报 relation does not exist，病灶却在早先的静默跳号）。

## 根修

**md 链版本整体 +1000**（md_1003=历史 0003……md_1066=新件），两链键空间全局
不相交：

- 单库单链（生产 PG-md/SG）语义零变化；
- 跨链并 schema（测试并集态=SG 现库形态）零撞号；
- 未来任何「按版本号」的工具（对账/审计/比对）不再有跨链歧义。

check_migrations 断言⑤（链内单调）+①（链内唯一）随之分链化；断言④（md_* 键域）
判阈用历史号（md 链号-1000）。

## 同坑族：跨链重放时序三刺客

并集态双跑还有时序病（fin 历史件 vs md 链尾态互踩）：

1. **瞬态超表化撞状态表末态**：0003 的 `create_hypertable('md_ticks')` 在
   0049 已把 md_ticks 转状态表（PK 不含分区列）后重放=红
   （cannot create a unique index without the column "ts"）——双侧同形 DO 守卫
   `IF to_regclass(...) IS NULL THEN PERFORM create_hypertable(...)` 根修
   （状态化表的瞬态超表层本就不影响末态）。
2. **add_retention_policy 重复调用即红**（retention policy already exists，
   实证）——双侧同形 DO 守卫查 timescaledb_information.jobs 后再 PERFORM。
3. **重放方向**：md 链先、fin 链后（历史重放方向——md 链的瞬态超表层先落，
   fin 链的状态化/守卫后到）；测试注释里钉死。

## 教训

- 迁移登记表的版本号一旦成为**多链共享键空间**，任何「同号拆分」都是定时雷；
  分链即错位键空间，一步到位。
- 跨库/跨链机制的同形断言（DDL 归一化）会让守卫代码也必须**双侧逐字同形**——
  改守卫时两链文件一起改，check_migrations ⑥ 会拦单侧改。
- 幂等迁移（IF NOT EXISTS/DROP IF EXISTS/OR REPLACE/DO 守卫）不只是重跑保险，
  是**并集态可组合**的前提。

## 关联

- D-0030 §六（分链设计）、D-0032 §三-5（前缀断言不可实施→白名单）
- T-0167 票面②③④
