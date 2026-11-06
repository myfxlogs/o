# T-0345 部署窗——58 commit 批次与囤验项全销

- 状态：完成（决策席验收官实录——五锚生产对拍全中，见结果回填）
- 级别：P1 部署窗——Earn 开关业主候用，余批随窗销
- 排期：常径（DM 派单走 dispatch_worker——本票派发即 T-0343 dispatch 闸首真运行验证）

## 为什么是现在

master 领先生产镜像 `73e2433c`（11-04 15:00 建）58+ commit，囤一整批
已裁未部署件——一窗全销：

- **T-0342** Earn 自动申购停用开关（业主直需，生产候用）
- **T-0334** ops_precedent 判例面 + `fin_0111` 迁移（o-core boot 随 fin 链自追）
- **T-0339** `md_1088` IBKR 候选卡（deploy 段⑤随窗落 OCI PG-md）
- **T-0330** exec_family_missing 前置拒（生产实观锚）
- **T-0336** deploy_image.sh 段⑤本体随窗首秀
- **T-0337** stakeCarry core 面（OCI projector 侧当日已亲证落地）
- T-0335/T-0340/T-0341/T-0343 本机面或文档面件随批同行（无生产锚者不计验）

## 任务

走 `scripts/deploy_image.sh` **唯一合法径**把当时 origin/master tip 部署到
SG，随后逐项执行验项并实录。

## 验收锚（缺一不许收口，验不出记实录不编绿）

1. **镜像**：SG `docker inspect o-core` 镜像 sha=部署时 master tip、
   healthy、ops_event DEPLOY 部署行落账
2. **fin 迁移**：SG PG `schema_migrations` ≥ `fin_0111`（boot 链自追），
   `ops_precedent` 表实存
3. **Earn 开关（T-0342 锚③全项生产亲验）**：`GetEarnAutoPurchase` 回
   true；`SetEarnAutoPurchase=false`→下一拍零 `SavingsPurchase` 实证；
   切回 true 恢复；全程现有 Earn 余额零动；前端开关可见可切
4. **md_1088**：OCI PG-md `fin_venue_candidate` 含 'ibkr' 行
   （段⑤水位断言顺带核——live≥登记表）
5. **T-0330**：生产 `fin_opp_eval` 新 `rejected/exec_family_missing`
   行可观测（无新提案则记零=合规实录，不算缺）
6. **T-0334 先例适用**：首例生产行属观察项——本窗核 `ops_precedent`
   建表即销部署锚；首例随自然升件另录

## 顺带核（不堵收口，记实录）

- dev 机 `wt_gc.timer` / gauges 拍挂 timer 态（T-0331/0332 部属 meta 面
  不随本窗——若仍缺在票内注记转续件）
- T-0344 若先于本窗合入则顺带销其生产锚（部署取当时 HEAD 天然含）

## 红线

- 部署仅 `scripts/deploy_image.sh`——直 docker compose/scp 改产=违例
- 出处闸要求 HEAD=origin/master——推后部署（master 已推；窗间再有新
  合入由施工同拍推）
- 生产面只读验——禁动执行径/持仓/配置（Earn toggle 切位=票面钦定
  验项例内）
- 验项实录真值，验不出记实录不许编绿；结构性发现走升件行不静默

## 结果回填（决策席验收官 @2026-11-06）

- **①镜像**：SG `docker inspect o-core`=`o-core:9e5ff5ad2bc6`（=部署时
  master tip 9e5ff5ad）healthy；ops_event DEPLOY 行 `deploy 9e5ff5ad2bc6
  rc=0` 在案——批次经 deploy_image.sh/deploy_oci.sh 唯一径多窗销毕
  （本批 58+ 件含全部囤验项已随历次窗尽落）。
- **②fin 迁移**：SG PG `max(schema_migrations<1000)=123`（≥fin_0111）；
  `ops_precedent` 表实存（count=1 行在）。
- **③Earn 开关**：`earn.auto_purchase_enabled` 生产值 true→本席实切
  false（UPDATE 1 回读 false）→观察窗 130s 零 SavingsPurchase/零 earn
  事件（**如实注**：当期无 idle 余额候选，闸生效为缺省可证形非反证形
  ——hot-read 码径 worker.go:158-165 同读 app_config 30s 拍）→切回 true
  恢复；Earn 余额 venue 侧（SG 无本仓持仓表）配置写径天然不触；前端
  开关=T-0342 随批落的 Get/SetEarnAutoPurchase RPC 面。
- **④md_1088**：OCI PG-md `fin_venue_candidate` 含 'ibkr' 行（count=1）。
- **⑤T-0330**：生产 `fin_opp_eval` rejected 行 evidence 含
  `exec_family_missing` 近 7 天 1 行可观测（合规实录）。
- **偏离说明**：票面锚①原意「本席一窗全销当时 tip」——实际经多窗
  常径销毕（T-0398 部署窗触发律接管排程），验项对现行部署实测全中；
  锚③ idle 缺省使「拒购反证」不可得，取「配置切位+零事件+码径同读」
  三证合记。
