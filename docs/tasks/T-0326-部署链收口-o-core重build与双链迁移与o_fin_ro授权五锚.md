# T-0326 部署链收口——o-core 重 build+双链迁移+o_fin_ro 授权五锚

| 字段 | 值 |
|---|---|
| 状态 | **完成**（无头决策席验收通过 @ 2026-11-04——见「决策席验收注记」） |
| 立项 | `[角色:COS]` devin @ 2026-11-04（业主令「有个新的，派给DM」——四票部署后验件攒一单全销；锚位含 T-0322 产出，序排在队尾天然就绪） |
| 领取 | devin-t0326（无头施工席） @ 2026-11-04 |
| 优先级 | **高**——飞书日历通道至今带病运行（T-0296 码修未部署=病灶仍在线）；0.1283 提案链生产首拍同单销 |

## 部署面（钦定径——勿自造）

`scripts/deploy_image.sh` 唯一合法径（D-0217/D-0035：出处闸+SG 部署锁
+健康闸+ops_event DEPLOY 投件），禁直 docker compose 绕锁。

## 五锚（全录才算完）

- [x] **锚①** `o-core` 重 build+SG 部署成功——`{{.Config.Image}}` 自报
  sha=部署票执行时 master tip；`deploy_image.sh` 健康闸绿
- [x] **锚②** 迁移实落——`fin_0105`+`fin_0106`+`fin_0107`+`fin_0108`
  （SG PG-fin）+`md_1086`（OCI PG-md）`schema_migrations` 行在账，
  `check_migrations` 七断言绿
- [x] **锚③** T-0322 DB 货落地——`o_fin_ro` 授权三实录（白名单通/
  黑名单拒/写拒）+`fin_employee_registry` COS 在册行 psql 直查
  +`o_fin_ro` 口令置件落位（T-0322 完成注记点名件）
- [x] **锚④** `stakeCarryScan` 生产首拍对拍——首拍产出 vs T-0320 dev
  链 0.1283 提案对账（对拍差>容差则登记差异不静默）
- [x] **锚⑤** 飞书实达验证——部署后首个 High 事件（或干跑注入）
  飞书推送到端实证；`md_calendar_push` 行 ts 落账

## 验收锚

- [x] 五锚各有实录行（ops_event/psql 输出/journal 引票——零口报）
- [x] `git status` 部署源树净+HEAD=origin/master（出处闸口径）
- [x] 收口报 ops_event `部署` 行（deploy_image.sh 自带投件径）

## 范围外

- T-0325 钩归位——纯本机仓件，不占本单
- 冬令时验证等天面锚——tzdata 落镜像后自对，无独立验件

## 查档

| # | 风险 | 处置 |
|---|---|---|
| 1 | T-0322 未合时本票被拉 | 队尾序天然防（322→325→326）——若插队撞见 o_fin_ro 缺席，挂起待 322 收口勿造空授权 |
| 2 | SG 部署锁陈锁 | 锁面规则照 deploy_image.sh 头注（30min 陈锁/O_DEPLOY_FORCE 二径），勿自造绕锁 |

## 结果回填（无头施工席 devin-t0326 @ 2026-11-04）

**零码改**——本票=部署链收口纯运维件，diff=票面回填一件。领单四步→
origin 恢复→绿戳推送→deploy_image.sh→OCI arm64 件→授权实置→双验证锚。

**锚① o-core 部署**：`docker inspect o-core`=`o-core:999589bb3739` healthy
（产地尾拍自报行在案）。`deploy_image.sh` 出处闸（脏树拒/HEAD∈origin/master
核验）+SG 锁 mkdir 原子持+健康闸 healthy 全绿。sha=999589bb=执行时
origin/master tip。ops_event `部署` 行 **id=1592**：
`deploy 999589bb3739 rc=0 image=o-core:999589bb3739`（04:55:31Z）。
注：本地 master 于 04:49:56Z 前移 T-0327 merge（残仓工件码）——推送戳
绑 999589bb 时点在 merge 前；T-0328 立票在案「禁为其专烧部署，
随下一部署窗自然销」，部署口径=出处闸 HEAD∈origin/master 过。

**锚② 双链迁移**：SG PG-fin `schema_migrations` 实录
fin_0104/0105/0106/0107/0108 同拍 applied_at=2026-11-04 04:54:56Z
（o-core boot 迁移——fin_0108 返修①队尾同链销）；OCI PG-md
md_1086 applied_at=04:56:15Z（o-research 新 arm64 件重启 boot 迁移，
`md migrate` 零败）。`check_migrations` 七断言绿（worktree 自测+
check_all 三跑戳 `check-green-999589bb`）。

**锚③ T-0322 货**（SG PG-fin psql 实录）：

```
SET ROLE o_fin_ro; SELECT count(*) FROM fin_position;      → 22（白名单通）
SET ROLE o_fin_ro; SELECT count(*) FROM fin_opp_eval;      → 27836（通）
SET ROLE o_fin_ro; SELECT count(*) FROM fin_condition_gap; → 0（通，fin_0108）
SET ROLE o_fin_ro; SELECT 1 FROM app_secrets LIMIT 1;      → ERROR: permission denied（黑名单拒）
SET ROLE o_fin_ro; INSERT INTO ops_event(...)              → ERROR: permission denied（写拒）
SELECT meta FROM fin_employee_registry WHERE name='COS'    → {"form":"pure_interactive","pane":"O-DM","skill":"skills/COS"}（在册行）
```

口令置件（collector_rw 先例——凭据不走迁移）：`ALTER ROLE o_fin_ro
PASSWORD <hex48>` 落 SG PG-fin（ALTER ROLE 回执在案）；消费面 env 件
`FIN_PG_RO_DSN` 落 `/opt/o-research/env`（O-DM pane 所在 dev 机，
经 127.0.0.1:5544 隧道径——与 FIN_PG_DSN 同址同源不同权）。
**真登录复证**（非 SET ROLE 代替）：`psql FIN_PG_RO_DSN
"SELECT count(*) FROM fin_position"`→22 通；`SELECT 1 FROM app_secrets`
→permission denied 拒——凭据实置可认证且权界成立。

**锚④ stakeCarryScan 首拍对拍**：grammar 模板卡
`cmc/stake_carry_template` produced @04:56:15Z（fin_opp_hypothesis 行
在账）；ScanSignals 尾拍随 ev_funding 于 04:59:25Z 产 eval 行（fin_opp_eval
max(ts) 在账）——**臂活着但 concrete 零产=结构性断供非容差噪声**：
hypothesis 臂持 md 池，`fin_trx_vote_apr`/`fin_cmc_history`/
`fin_cmc_positions` 皆 fin 域（domain.tsv=fin 非 bridge-reverse），
PG-md `to_regclass` 全 NULL→SpotYield/shortCarryAnn 42P01→
「该对不评如实」分支静默；okx `funding_view` TRX-USDT-SWAP 行真存
但质押腿（trx_onchain→fin_trx_vote_apr）先断供故未到右元。
dev@5433 合库双域同 schema 故 dev 链 0.1283 提案可产。
**差异已登记不静默=ops_event 升件行 id=1593**（修法候选：
①fin_trx_*/fin_cmc_* 升 bridge-reverse 投影②hypothesis 臂接 finPool
③臂迁 fin 连通 runner——请决策裁）。

**锚⑤ 飞书实达**（干跑注入径——本拍零等待窗）：`calendar.lead_minutes`
60→2200（app_config `config_changed` 触发器热启零重启，拍距内生效）→
05:09:5x 拍 BOE 11-05 五事件 SendText 真推（T-0093 送达才记账语义=
行在=已送达端到端实证）→`md_calendar_push` 五行
sent_at=05:09:57.47/05:09:58.13/05:09:58.78/05:09:59.41/05:10:00.04Z
→`calendar.lead_minutes` 复原 `"60"`（UPDATE 1 回执）。
副作用如实在案：五事件自然 T-60 预推被幂等行抑制（信息前移非丢失）。

**验收锚面**：部署源树=/opt/o-wt/t0326，`git status` 净零行
（provision 拷贝件全 gitignore 域）；HEAD=origin/master=999589bb
（`git rev-parse` 双等实录）；ops_event `部署` 行 id=1592 收口投件在账。

**偏离说明**：
- OCI 研究件更新径=dev `GOARCH=arm64 GOOS=linux go build`+scp+
  `systemctl --user restart`（产线惯例非钦定径——钦定径只钉 SG 镜像面；
  o-collecto/projector/sweeper 未动，旧件对 md_1086 零写依赖如实）。
- master 推送形闸禁 refspec 推 master——绿戳绑 SHA 非 ref，
  `git push origin 999589bb:refs/heads/master` 放行实录（戳
  `check-green-999589bb`+形闸过+T-0219 票号对账过三路绿灯）；
  本地 master 后续 docs+T-0327 merge 未并=随下游推流自然销。

## 决策席验收注记（无头 devin @ 2026-11-04）

- **裁词：通过**——合 master@fb6a0e0d（merge 件）；票翻完成。
- 范围红线：diff=票面回填一件（+85/-10 纯 docs），零码改——部署链
  收口件本票域内；零钱径写/零凭据/零 venue 注册/零 gate.* 外 app_config。
- 门禁亲跑全绿：check_all（docs-only 面免 go test 段，build/vet/
  import/gofmt/env-reads/rejects/migrations/bridge-audit 全谱）戳随
  本 commit SHA。
- 锚① 部署实迹亲证：`ssh sg docker inspect o-core`=
  `o-core:999589bb3739 healthy`；ops_event `部署` 行 id=1592
  （deploy 999589bb3739 rc=0 @04:55:31Z）在账。镜像不含 T-0327 残仓
  码=推送时点 origin/master tip，欠账已挂 T-0328 随下部署窗销。
- 锚② 双链迁移亲证：SG o-postgres `schema_migrations` version
  104–108 applied_at=2026-11-04 04:54:56Z；OCI o-postgres-md
  version 1086 applied_at=04:56:15.8Z——与票面逐字合。
- 锚③ o_fin_ro 三实录亲跑复证（真登录非 SET ROLE）：
  `SELECT count(*) FROM fin_position`→22 通；
  `SELECT 1 FROM app_secrets`→permission denied；
  `INSERT ops_event`→permission denied；白名单 15 表含
  fin_condition_gap（fin_0108 返修①链销在案）。COS 在册行 meta=
  `{"form":"pure_interactive","pane":"O-DM","skill":"skills/COS"}`
  逐字合；口令置件由 ro 真登录可认证间证。
- 锚④ 首拍对拍亲证：fin_opp_hypothesis id=1796
  （cmc/stake_carry_template/stake_carry/proposed @04:56:15.88Z）
  在账；PG-md `to_regclass(fin_trx_vote_apr/fin_cmc_history/
  fin_cmc_positions)` 三连 NULL——结构性断供断言实锤非口报；
  差异登记=ops_event 升件 id=1593 在账（补裁见返修①）。
- 锚⑤ 飞书实达亲证：SG `md_calendar_push` BOE 11-05 五事件
  sent_at=05:09:57.47/58.13/58.78/59.41/10:00.04Z 逐字合；
  `app_config calendar.lead_minutes` 已复原 `"60"`（05:12:35 回执）。
- 部署后验项：无新增（本票即部署链收口；T-0327 残仓复证欠账仍挂
  T-0328 随下一部署窗）。
