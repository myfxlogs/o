# T-0326 部署链收口——o-core 重 build+双链迁移+o_fin_ro 授权五锚

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:COS]` devin @ 2026-11-04（业主令「有个新的，派给DM」——四票部署后验件攒一单全销；锚位含 T-0322 产出，序排在队尾天然就绪） |
| 领取 | （空=未领） |
| 优先级 | **高**——飞书日历通道至今带病运行（T-0296 码修未部署=病灶仍在线）；0.1283 提案链生产首拍同单销 |

## 部署面（钦定径——勿自造）

`scripts/deploy_image.sh` 唯一合法径（D-0217/D-0035：出处闸+SG 部署锁
+健康闸+ops_event DEPLOY 投件），禁直 docker compose 绕锁。

## 五锚（全录才算完）

- [ ] **锚①** `o-core` 重 build+SG 部署成功——`{{.Config.Image}}` 自报
  sha=部署票执行时 master tip；`deploy_image.sh` 健康闸绿
- [ ] **锚②** 双链迁移实落——`fin_0105`（SG PG-fin）+`md_1086`
  （OCI PG-md）`schema_migrations` 行在账，`check_migrations` 七断言绿
- [ ] **锚③** T-0322 DB 货落地——`o_fin_ro` 授权三实录（白名单通/
  黑名单拒/写拒）+`fin_employee_registry` COS 在册行 psql 直查
- [ ] **锚④** `stakeCarryScan` 生产首拍对拍——首拍产出 vs T-0320 dev
  链 0.1283 提案对账（对拍差>容差则登记差异不静默）
- [ ] **锚⑤** 飞书实达验证——部署后首个 High 事件（或干跑注入）
  飞书推送到端实证；`md_calendar_push` 行 ts 落账

## 验收锚

- [ ] 五锚各有实录行（ops_event/psql 输出/journal 引票——零口报）
- [ ] `git status` 部署源树净+HEAD=origin/master（出处闸口径）
- [ ] 收口报 ops_event `部署` 行（deploy_image.sh 自带投件径）

## 范围外

- T-0325 钩归位——纯本机仓件，不占本单
- 冬令时验证等天面锚——tzdata 落镜像后自对，无独立验件

## 查档

| # | 风险 | 处置 |
|---|---|---|
| 1 | T-0322 未合时本票被拉 | 队尾序天然防（322→325→326）——若插队撞见 o_fin_ro 缺席，挂起待 322 收口勿造空授权 |
| 2 | SG 部署锁陈锁 | 锁面规则照 deploy_image.sh 头注（30min 陈锁/O_DEPLOY_FORCE 二径），勿自造绕锁 |
