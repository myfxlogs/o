# T-0336 PG-md 迁移部署件——md_1087 与 bridge-reverse 五表落 OCI

- 状态：待验（施工收口 devin o-worker @2026-11-04——anchors 三项实录在案）
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

## 施工实录（devin o-worker @2026-11-04）

**① 现状定性（任务①欠账定性）**：OCI/PG-md 迁移原有两条半径皆不随
部署窗——(a) `cmd/projector` boot 自带 `pgstore.Migrate(mdPool,"md")`
（cmd/projector/main.go:42），但 OCI 盒 `/opt/o-projector` 二进制为手工
scp 旧件不随 master 进，新登记迁移永远跑不到；(b) `cmd/migrate` 一次性
执行器（T-0167）须 dev 侧 `GOARCH=arm64 go build`+scp+手跑（T-0326
偏离说明产线惯例在案）——无任一步在 deploy_image.sh 内，故 T-0333
窗 PG-md 水位停 1086/43 行（施工前亲证实录：`SELECT max(version),
count(*)` = 1086/43，与票面逐字合）。欠账=部署径结构性缺席非漏跑。

**② 实现（钦定形二择取①）**：`scripts/deploy_image.sh` 增 OCI PG-md
迁移随窗步（段落⑤，锁/出处闸/投件语义零改——diff 只加不减）：
arm64 `cmd/migrate` 交叉建→scp 上盒→`~/o-projector.env` 取
`MD_PG_DSN` 跑 `-domain=md`→水位断言 `live≥登记表`（登记表号自
migrate_table.go md_ 链 grep，同 check_schema_live 判据）。
`O_MD_MIGRATE_SKIP=1` 跳步脱扣（OCI 失联不堵 SG 部署，跳步=欠账
哨兵照红）；`O_MD_HOST` 可覆写盒址。

**③ 实跑实录**：本票即按新步同形手跑一遍——arm64 migrate scp→
`PG_DSN=$MD_PG_DSN /tmp/o-migrate-md -domain=md` 输出 `[migrate]
md 链应用毕`；验 `schema_migrations` version=1087
applied_at=2026-11-04 07:57:04Z、行数 43→44；
information_schema 五表 fin_trx_vote_apr/fin_trx_account/fin_trx_txs/
fin_cmc_history/fin_cmc_positions 全存（\dt 等价实录五行）。

**验收锚对照**：
- [x] PG-md `schema_migrations` 含 md_1087（1087 @07:57:04Z 实录）
- [x] 五表在 PG-md 实存（information_schema 五行实录）
- [x] 迁移径文档化=deploy_image.sh 段落⑤——下次部署窗自随跑

**自测**：`bash -n`/`go build ./...`/`go vet ./...`/`go test ./...`/
`check_migrations.sh`（七断言）全绿。

**范围外守**：五表数据回填=projector 自然追写另案（本票只立表）；
出处闸/锁/投件语义零改（diff 纯新增块）。
