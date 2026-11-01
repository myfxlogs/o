---
schema_version: 2
id: bridge-ddl-constraint-needs-md-twin
kind: pitfall
risk: medium
status: verified
keywords: [桥表, CHECK约束, 迁移孪生, DDL归一化, check_migrations, 八桥]
scope: [migration, schema, bridge]
summary: 桥表（八桥+反向桥）新增 CHECK/UNIQUE 等约束只落 fin 链=check_migrations 桥表归一化断言红——DDL 双链同形是机检契约，约束件须 fin+md 孪生（md 镜像存量同清后加同款 ALTER），且 md 号走 md 链自身次序非 fin 号+1000
first_seen: 2026-11-01
last_seen: 2026-11-01
evidence_refs:
  - type: ticket
    ref: T-0205
  - type: file
    ref: scripts/check_migrations.sh
  - type: file
    ref: migrations/md_1078_eval_proposed_net_apr_gate.sql
---

## 桥表加约束漏 md 孪生=迁移机检红

T-0205 重飞给 fin_opp_eval 加 CHECK 约束（proposed 行 net_apr NOT NULL），
fin_0094 落妥后 check_all 在 migrations 闸红：`桥表 fin_opp_eval DDL 归一化
漂移（md 链正表 vs fin 链镜像语句集不等）`——桥表 DDL 双链同形是机检契约
（回滚径「SG 镜像表即正表（DDL 相同）」前提），check_migrations ⑥ 逐语句集
diff 双链。

**修法**：fin 件+md 孪生件同 commit（fin_0094+md_1078，slug 同构）；md 件
内 UPDATE 清 md 镜像存量（镜像行与 SG 同形违例）→同款 ALTER。md 号走
md 链自身次序（1077 后=1078），**非** fin 号+1000（+1000 是键空间隔离规则
非配对规则——md_1077 对 fin_0085 同 slug 在案）。

DML（UPDATE）不入归一化对比集（语句类型过滤 create/alter/index/hypertable/
retention）——孪生件 DML 可按各侧存量实况写，ALTER 语句须逐字同形。

**第二坑（同修法根因）**：测试 schema=双链并库（testutil+pgstore.Migrate
md/fin 两链灌同 schema）——孪生 ALTER 无幂等守卫时后到链 42710 撞号
（constraint already exists），全包测试 setup 级连炸（cmd/core 83 处同源）。
约束件 ALTER 必须前置 `DROP CONSTRAINT IF EXISTS` 再 ADD（md_1074/fin_0083
direction_check 孪生既有范式）——双链各库（SG fin 链/OCI md 链）与并库
（测试 schema）两态下均幂等。
