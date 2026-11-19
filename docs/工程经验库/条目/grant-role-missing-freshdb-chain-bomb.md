---
schema_version: 2
id: grant-role-missing-freshdb-chain-bomb
kind: pitfall
risk: high
status: verified
keywords: [GRANT, SQLSTATE 42704, CI 炸链, roster_rw, 顶格GRANT, IF EXISTS 门, INVOKER 触发器, 测试自建角色, 共享簇假绿]
scope: [migrations, internal/platform/opschain, scripts/check_migrations.sh, CI 测试库]
summary: 顶格 GRANT TO 链外角色（roster_rw 等生产手工角色不在迁链内建）——fresh 测试库无此角色即 SQLSTATE 42704 炸迁移链；五犯实证后 check_migrations 断言⑧ 机检封死。姊妹坑=grant-not-in-migration-silent-dead（反向：漏授权→哑死）
first_seen: 2026-11-19
last_verified: 2026-11-19
skills: [生产取证]
evidence_refs:
  - type: commit
    ref: d017b7ca2/f1a9bf3ea/739d19450——fix/roster-grant-guard 三连：四门补+闸⑧+测试补权
  - type: server_fact
    ref: CI run 44439770827(0161炸)→44449854063(0191炸)→44458094780(测试触面拒)——逐层揭纱三案
---

## 情境

迁移件写 `GRANT ... TO <角色>` 顶格裸授。生产手工角色（roster_rw/
research_rw/review_rw/analyst_rw/projector_rw 等）**不在迁链内建**——
迁链只 CREATE 了 o_fin_ro(fin_0106)/condreq_resolver(fin_0165)。
CI/fresh 测试库无这些角色，迁移跑到该句即 SQLSTATE 42704 炸链。

## 实数（五犯）

- fin_0161/0190/0198/0208 四处裸 `TO roster_rw`——CI 红位 0161；
- fin_0191:100-101 漏网（首轮普查按 `pg_roles` 出现数判门，误把
  DROP VIEW 的 IF EXISTS 当授门）——二轮 awk DO块追踪全谱扫抓出；
- 三层揭纱：迁好→测试自建 roster_rw 缺授权面→INVOKER 触发器逐层拒
  （先 ops_event_sem_matrix 语义闸读，补后 ops_event_act 解析臂写）。

## 根因

① GRANT 目标是「迁外知识」——角色存在性不可静态推；
② **验证掩蔽**：本地共享测试簇（o-postgres-test）全局角色真在——
迁移重放假绿，CI 裸容器才真。环境差下"本地过≠CI 过"；
③ 测试自建角色=零迁移真授，INVOKER 触发器触面（ops_event 上
sem_gate 读矩阵/act_ai 写臂/precedent 先例臂）须逐表对齐
prod 在册权，漏一手=插行撞下一层。

## 判据挂接

- 迁移写 GRANT 必走 `DO $$ IF EXISTS(pg_roles) ... EXECUTE` 门
  （fin_0098/0210 形）——断言⑧ 已机检：DO块外顶格 GRANT 只许
  链内自建角色或 PUBLIC；
- 测试自建角色的授权面=该角色 prod 在册权逐字镜像（触发器触面
  全含，含 fail-open 臂）；
- 环境差验证=裸容器复刻（timescale 镜像+仅 postgres+o 角色），
  全链重放+单测定向——共享簇结果不可作 CI 证。

## 配套纪律

逐错补丁=揭纱式打地鼠（修161→炸190→炸191→炸测试）——正径=
全谱静态扫（DO块追踪）+CI 同构环境一次验全，不在错误信息里
迭代收敛。
