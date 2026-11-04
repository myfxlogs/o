# T-0336 PG-md 迁移部署件——md_1087 与 bridge-reverse 五表落 OCI

- 状态：已批（COS 例行代理批 @2026-11-04——常票工程续链）
- 立项：`[角色:决策]` devin 无头拍 @ 2026-11-04（T-0333 部署窗验项②③红项根——结构性发现立票）
- 级别：P2 部署链欠账件——T-0326 返修①合入面生产半侧未活

## 病灶（T-0333 部署窗实录在案）

`scripts/deploy_image.sh` 只船 SG o-core 镜像；**PG-md（OCI 侧）
迁移从不随窗跑**——生产 schema_migrations 43 行无 `md_1087`，
bridge-reverse 五表（fin_trx_vote_apr/fin_trx_account/fin_trx_txs/
fin_cmc_history/fin_cmc_positions）`relation does not exist`。

后果：T-0326 合入的升域五副本面在 PG-md 侧整体缺席——五表对拍
验项无物可对，projector 消费面无表可写（断供/缺席类实案）。

## 任务

1. 查 OCI/PG-md 侧迁移现状：谁跑 migrate（boot 自检/手工/无径）——
   `cmd/migrate` 或部署链内是否有对 OCI DB 的迁移步，欠账定性落票
2. 补 PG-md 迁移部署径：钦定形二择——deploy_image.sh 增 OCI
   migrate 步（同锁同窗）或独立 `scripts/deploy_md_migrate.sh`
   小件（出处闸+schema_migrations 水位断言同纪律）
3. 实跑落 md_1087+五表 DDL，schema_migrations 实录随票

## 验收锚

- [ ] PG-md `schema_migrations` 含 md_1087（生产实录）
- [ ] 五表在 PG-md 实存（`\dt` 或 information_schema 实录）
- [ ] 迁移径文档化（scripts 件或 deploy_image 内步——下次部署自随窗）

## 范围外

- 五表数据回填/追灌——表立后 projector 自然追写，另案处理回填量
- deploy_image.sh 出处闸/锁语义改——只加迁移步不动既有闸
