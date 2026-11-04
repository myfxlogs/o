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
