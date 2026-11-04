# T-0326 返修①——stakeCarryScan 质押腿断供根修：五 fin 表升 bridge-reverse（升件补裁件，非驳回）

> 事由：T-0326 锚④ 升件行1593——stakeCarryScan 生产首拍模板卡
> cmc/stake_carry_template@04:56:15 已产（fin_opp_hypothesis#1796），
> concrete cmc:TRX 零产=**结构性断供**非容差噪声：研究臂持 md 池，
> `fin_trx_vote_apr`/`fin_cmc_history`/`fin_cmc_positions` 皆 fin 域
> （domain.tsv 非 bridge-reverse），PG-md `to_regclass` 三连 NULL→
> SpotYield/shortCarryAnn 42P01→「该对不评如实」静默。okx
> funding_view TRX-USDT-SWAP 行真存但质押腿先断供故未到右元。
>
> **补裁（决策席无头拍 devin @ 2026-11-04）**：采纳修法①——fin 域表
> 升 `bridge-reverse` 副本入 PG-md（现成反向臂径：domain.tsv 入册+
> md_*_replica 迁移建副本+reverse.go 全表 diff——md_1070 先例同形）。
> 汰②「臂接 finPool」=OCI 持 fin 凭据新爆面+跨 WAN 每拍查询；
> 汰③「臂迁 fin 连通 runner」=研究臂撕裂跨节点，机制搬家成本高。
>
> **范围补裁 3→5 表**：升件点名三表外，亲证读面再发两条**同类静默
> 断供**——`fin_trx_account`/`fin_trx_txs`（rental 腿 err==nil 吞
> 42P01→rental 恒 0 退化，spotyield.go:96-103）——同病灶一并根修。
> 五表体量亲证全小（vote_apr 2409/trx_account 3008/trx_txs 813/
> cmc_history 39/cmc_positions 8390 行），全表 diff 廉价。

## 工作树/分支

`git worktree add /opt/o-wt/t0326r -b t/0326-stakecarry-replica master`——
原树 `/opt/o-wt/t0326` 票毕可收（`git worktree remove`）。

## 须落六件（勿扩）

1. `migrations/domain.tsv`——五行 `fin`→`bridge-reverse`：
   `fin_trx_vote_apr`、`fin_trx_account`、`fin_trx_txs`、
   `fin_cmc_history`、`fin_cmc_positions`。
2. `migrations/md_1087_stake_carry_fin_replica.sql`——五表
   `CREATE TABLE IF NOT EXISTS` 副本 DDL（照 md_1070_fin_strategy_replica
   先例；与 fin 正本同形——断言⑥ DDL 归一化钉）。
3. `internal/store/pgstore/migrate_table.go`——`md_1087` 登记。
4. `internal/projector/reverse.go`——五表 reverse 函数（全表 diff 照
   `reverseStrategies`/`reverseEarnRate` 形；勿造 CDC/触发器新范式）。
5. `internal/projector/reconcile.go` `Reconcile` 挂线+
   `internal/projector/ops.go` 拍点按既有帧径接（无 SG 触发帧者随
   Reconcile 周期对账即可——**勿为本次新增 SG NOTIFY 触发器**，超域）。
6. `docs/数据流.md` 反向桥面补五表（链路变更同 commit 硬律）。

## 验收锚（亲跑实录贴回填段，禁口报）

- [ ] dev 双链迁移落账：fin 侧无新 fin_*（本件零 fin 迁移）+md 侧
  `md_1087` 在账；PG-md `to_regclass` 五表全非 NULL。
- [ ] 副本对账：md 侧五行数与 SG 正本 `SELECT count(*)` 一致
  （正本行数上文亲证值可对拍）。
- [ ] `check_migrations` 七断言绿（含⑥五新副本 DDL 同形）+
  `bridge-audit` 授权/存在性列绿（reverse 表面 3→8 随册动）。
- [ ] dev 链复跑 stake_carry 路径到右元：断供 42P01 消失——
  concrete eval 行落账或「该对不评」拒因换成真数据判据（dev@5433
  合库双域本通，验的是 42P01 静默臂消灭+投影后行在账）。
- [ ] `check_all` 全谱绿戳随收口。

## 停线

- 只这五表升域——他表禁动（`fin_trx_vote_apr_top` 等未读面不入）；
- 零钱径写/零 venue 注册/零 gate.* 外 app_config/零凭据面；
- 反向臂勿造新范式——全表 diff 照既有三函数形。

## 结果回填（无头施工席 devin @ 2026-11-04）

### 实现摘要（须落六件+机械随件）

1. `migrations/domain.tsv`——五行 `fin`→`bridge-reverse`：fin_trx_vote_apr/
   fin_trx_account/fin_trx_txs/fin_cmc_history/fin_cmc_positions；
   `fin_trx_vote_apr_top` 等未读面留 `fin`（停线守）。
2. `migrations/md_1087_stake_carry_fin_replica.sql`——五表
   `CREATE TABLE IF NOT EXISTS`+索引逐字照 fin_0046/fin_0081 正本（断言⑥
   归一化同形钉）+`fin_data_retention` 五行 class='state'（副本不删行——
   sweeper 删行与全表 diff 互搏=churn 环，md_1070 同口径）。
3. `migrate_table.go` 登记 `{1087}`（md 链尾单调）。
4. `internal/projector/reverse_fin.go`（新件——reverse.go 450 行硬线
   分拆，同包同 Runner）——五表 reverse 函数共走 `revTableDiff[T]` 泛型
   双侧全表 diff（行判等 `reflect.DeepEqual`：指针列按所指值比——`h!=s`
   结构体判等含指针列恒不等=全表恒覆写浪费，坑条目
   struct-pointer-field-equality-always-diff 同 commit 入库）+
   `reverseFinMirrors` 汇总径；numeric `::text` 搬运保精度（mirror.go 同律）。
5. `reconcile.go` `Reconcile` 挂 `reverseFinMirrors`（earn_rate 后）；
   `ops.go` `ev_strategy` 帧顺带（既有反向桥帧径——五表无 SG 触发帧，
   期界对账兜底；零新 NOTIFY 触发器）；`projector.go` `New()` 反向
   搬运面对册清单 3→8。
6. `docs/数据流.md` 反向桥三处补五表（链路变更同 commit）。
   机械随件：`check_migrations.sh` 断言⑥ `BRIDGE` 清单 +5 表
   （验收锚「含⑥五新副本 DDL 同形」的机检载体——不入清单=断言不覆盖）；
   `projector_test.go` 增 `TestReverseFinMirrors`（五表 diff/覆写/幂等三拍）。

### 验收锚亲跑实录

- [x] **dev 双链迁移落账**：fin 侧 `migrate -domain=fin`→`fin 链应用毕`
  （基线齐，零新 fin_* 在案）；md 侧 dev@5433 `schema_migrations` 1087=
  `md_1087_stake_carry_fin_replica.sql` 在账；`to_regclass` 五连=
  `fin_trx_vote_apr|fin_trx_account|fin_trx_txs|fin_cmc_history|fin_cmc_positions`
  全非 NULL。**环境注记**：dev 合库 md 链账停 1011 为既有漂移
  （md_1015 `create_hypertable` 撞非空 md_earn_rate 唯一键——fin 链镜像
  建的 PK(venue,ccy) 与 ts 分区不兼容），md_1087 走 psql 直落+账行补登
  同 md_1086 dev 先例（行在账未链销），不改旧迁移不碰他表数据。
- [x] **副本对账**：dev 合库同表故 md 侧行数=正本行数——
  vote_apr 225/trx_account 765/trx_txs 784/cmc_history 34/
  cmc_positions 672（生产正本亲证值 2409/3008/813/39/8390 部署后对拍
  由生产验证锚兜）；真实 diff 语义证=`TestReverseFinMirrors` PASS
  （独立 schema 双侧——五表各 fixture 落副本+行变覆写 min_sr NULL+
  幂等零漂移，`go test -run 'TestReverseFinMirrors|TestReverseBridge'`
  双 PASS 实录）。
- [x] **`check_migrations` 七断言绿**（亲跑 `OK: check_migrations`——
  含⑥五新副本 DDL 归一化语句集相等）；**`bridge-audit` 四列全绿**：
  `授权列: 23 桥表×projector_rw.SELECT + 8 reverse×research_rw.SELECT 齐`
  （reverse 表面 3→8 随册动实证）+存在性/模板/水印三列同绿。
  **SG 生产授权实录**：`GRANT SELECT ON 五表 TO projector_rw,research_rw`
  →`GRANT` 回执+实查五表×两角色 SELECT 全在（`role_table_grants`
  分组对拍输出在案——手工补权同 fin_0104 先例，本件零 fin 迁移票面钦定）。
- [x] **dev 链复跑 stake_carry 到右元**：临时 scanonce-stake 跑一拍
  `ScanSignals`（用完即删未入库）→`SCAN-OK
  stake_carry_eval_rows_last_2min=1`；实查 `fin_opp_eval` 新行=
  `cmc:TRX|stake_carry|proposed|net_apr=0.12833773338730112|
  legs={"spot":"long_staked","short":"cmc"}`@05:35:55Z——concrete eval
  行落账=42P01 静默臂消灭（净 1283bps 与 T-0320 标定值 0.1283 逐位合）。
- [x] **`check_all` 全谱绿**：build/vet/import 129-138 边/gofmt/env-reads/
  rejects/migrations/bridge-audit 四列/go test 全包 ok——戳落
  `check-green-11d2541f`（收口 commit SHA 重跑绑定）。

### 偏离说明

- `ops.go ev_strategy` 顺带五表 diff——票面「按既有帧径接」落形（五表
  无 SG 触发帧，反向桥唯一 SG 帧=ev_strategy；小表全表 diff 廉价顺带，
  Reconcile 期界仍兜底）。
- `projector_test.go` 增测试一件+`check_migrations.sh` BRIDGE 清单扩
  ——验收锚要求的机检覆盖载体，属机械随件非扩域。
- dev@5433 md 链账停 1011 既有漂移（md_1015 hypertable 冲突）如实报
  不修——修=改旧迁移/动 dev 表键超票域；md_1087 psql 直落+账行补登
  与 md_1086 dev 先例同径。
