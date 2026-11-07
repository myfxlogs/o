---
schema_version: 2
id: deploy-oci-before-image-migrations-crash-new-binary
kind: pitfall
risk: medium
status: verified
keywords: [deploy_oci, deploy_image, 发窗序, 迁移未落, inst_type, fin_0137, 活性闸, 两半部署]
scope: [部署径, 发窗序, 两半部署制, 迁移依赖]
summary: OCI 裸二进制写 SG 库——新件若依赖新迁移列/表，先发 oci 面=新件对旧 schema 起即崩（活性闸红回滚）；发窗正序=image 面先落（o-core 起带迁移）→oci 后续
first_seen: 2026-11-07
last_verified: 2026-11-07
skills: [决策席, 外审席]
evidence_refs:
  - type: doc
    ref: docs/collab.md §11 两半部署制（两径锁域分离可并行——并行许可不消 schema 依赖序）
  - type: doc
    ref: scripts/deploy_image.sh（OCI PG-md 迁移随窗段——镜像面部署落 md 链；fin 链由 o-core 起带）
---

## 事实/症状

- 发窗 `faces=image,oci` 双面并发：oci 臂换件在镜像面部署完成前
  先跑，新二进制起即崩——活性闸红回滚；
- 2026-11-07 实证：o-collecto 换 82bca570 起崩两连——
  `[okxscan] 粗扫拍: ERROR: column "inst_type" of relation
  "md_tick_scan" does not exist (SQLSTATE 42703)`——T-0427
  fin_0137 迁移列 SG 库未落（oci 三臂直写 SG 库，迁移随镜像面/
  o-core 起带落）；
- 镜像面部署毕（fin 链水位 138 实查）→同二进制重换→活性闸绿
  写流恢复——码无罪，序有罪。

## 根因或策略解释

- 「两径锁域分离可并行」管的是**部署锁/写面互斥**，不管
  **schema 依赖序**——OCI 臂是 SG 库的客户端，新码对新列/新表
  的依赖由 SG 侧迁移满足，迁移随镜像面落；
- 活性闸红=正确兜住（fail-closed 禁带病上线），但拍序对可零烧。

## 处置

- 发窗双面并发时**序钉**：`deploy_image.sh` 先（含迁移落位断言）
  → `deploy_oci.sh` 后；oci 面活性闸红先查「本臂新码是否依赖
  未落迁移」（起崩日志 SQLSTATE 42703 列缺=迁移序病灶特征）；
- 救场序：让镜像面跑完→`O_DEPLOY_FORCE=1 deploy_oci.sh <臂>`
  强换（谱系自检若被假 .sha 骗跳——见姊妹条目）；
- 判词锚：SG 库直查 `schema_migrations` 水位 ≥ 待发件迁移号。

## 证据

- check: 2026-11-07 collecto 42703 起崩×2→回滚→镜像面部署毕
  schema_migrations=138→重换绿（journalctl o-collecto 实录）
- result: 序对则同件零改动过闸——纯序坑非码坑

## 反证条件

- check: deploy_window/发窗件若把「oci 面待镜像面迁移水位断言」
  机内化后本条 retired
- falsifies_when: oci 臂改读自含 schema 或对缺列降级容错（不
  应——fail-closed 是钦定形）

## 适用边界

- OCI 三臂与 SG 库 schema 依赖面；无新迁移件的发窗不受序影响。

## 伴生量表（T-0332 强制）

- gauge_name: deploy_oci_binary_vs_schema_watermark
- class: 部署序
- query: 发窗时比对 待发 SHA 引入迁移号 vs SG schema_migrations 水位
- threshold: 迁移号>水位=序违规先险（当前无周期探针，人工判）
