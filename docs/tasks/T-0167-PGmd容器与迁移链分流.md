# T-0167 PG-md 容器+迁移链分流（D-0030 §六/§八——分域施工①）

## 态：完成（判过并码在案，11-01 清账翻面·SETTLE-SWEEP）
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**完成**——审计通过落票在案，漏网翻面
## 立项：决策席 @ 2026-10-26
## 领取：zcode-glm 施工席 @ 2026-10-26（支 t/0167-pgmd-container-migration-split，前置于 b0223d0——T-0165 已翻投产）
## 决策档：D-0030

## 施工
0. **隧道钥匙收窄（D-0030 §八 r4 改判——Tailscale 非必要）**：SG sshd
   `PermitRootLogin prohibit-password`+fail2ban；隧道走专用低权用户 `tunnel@`
   （`Match User tunnel`→`ForceCommand /bin/false`+`PermitOpen 127.0.0.1:15433`）
   或钥匙行加 `restrict,port-forwarding,permitopen="127.0.0.1:15433"`；root 钥匙
   从隧道退役；`alert-bridge-tunnel` 同收窄。验证：以隧道钥匙 `ssh SG id`→拒；
   PG 隧道通；SG 24h 暴力认证计数降。
1. 开发机 `docker-compose.dev.yml` 起 PG-md（timescale 同版），数据卷与施工树
   目录隔离；systemd `o-prod.slice`（collector/research/projector/PG-md 归入）。
2. `migrations/` 分链：`md_NNNN_*`/`fin_NNNN_*`；桥表镜像 DDL **归一化断言**
   （D-0030 §六——弃「同源 include」措辞，D-0031 六-⑥）。
3. `check_migrations.sh` 扩三断言（两链各单调/桥表 DDL 归一化同形/域表禁串链
   ——域表判定用显式表清单，不按 md_/fin_ 前缀）。
4. `pgstore.Migrate(-domain=md|fin)` 选链；SG 走 fin、开发机走 md。

## 桩锚
- 两链各自 Migrate 全过；串链表→check 红；桥表 DDL 漂移→check 红。
## 验收锚
- 开发机 PG-md 空库 Migrate md 链全过；SG fin 链 Migrate 幂等零变更。

## 复审注记（2026-10-26→D-0032）

- 候派前置补一条：T-0165 返修（report 留 SG）须验收合入 master——现 8037071
  仅在 `t/0165-research-node`，master 票态仍「已批」，report 留 SG 未定。
- 派单时票面须并入 md_macro_event 跨机读面裁法：D-0032 裁默认=投影第八桥表
  （§三/§四/§六「七桥表」改八）；若复议改选「report 读 PG-md」，须补第二
  DSN+隧道失败静默语义，并回 D 档撤回锚③结论。
- 本票「桥表 DDL 同形」实现须比列定义全项+NOT NULL/DEFAULT/CHECK/UNIQUE/FK/
  索引集（D-0032 §三-5）——只比列名/类型/PK 不满足回滚「DDL 相同」前提。

## 态：待验
## 结果回填（施工席 zcode-glm @ 2026-10-27，支 t/0167-pgmd-container-migration-split @ 4dbba85+）

### 逐项施工

- **⓪ 隧道钥匙收窄（D-0030 §八三行收窄全落）**：SG sshd 主文件
  `PermitRootLogin prohibit-password`（.bak-t0167 备份+sshd -t+reload，sshd -T
  亲证 without-password）；`Match User tunnel` 文件尾追（ForceCommand /bin/false
  +PermitOpen 127.0.0.1:15433+PermitListen none+禁 tty/X11/agent+local-only——
  sshd -T -C user=tunnel 亲证五项）；`tunnel@` 低权用户（nologin）+
  dev 专用钥 o_tunnel_pg_ed25519（SG 侧钥行 restrict,port-forwarding,
  permitopen="127.0.0.1:15433"）；dev `o-collecto-tunnel.service` 切
  tunnel@+IdentitiesOnly——**root 钥退役出 PG 隧道**；alert-bridge 钥行升
  `command="/bin/false",restrict,port-forwarding,permitlisten="127.0.0.1:8099"`
  （kill 断线重连实证：SG 127.0.0.1:8099+172.20.0.1:8099 双听、dev 桥活）；
  fail2ban 装机+sshd jail（maxretry5/findtime10m/bantime1h 递增，active）。
  验证：隧道钥 `ssh sg id`→拒（exit1 零输出）；PG 隧道通（collector_rw 经
  5544 实读 md_mark_index=2724 行）；collector 重启全量订阅 1014 syms 恢复。
  暴力基线 2,840 次/24h（fail2ban 前）——**24h 复核=观察项**。
- **① PG-md 容器**：`docker-compose.dev.yml`（timescale 2.30.2-pg18 同版/
  127.0.0.1:5435/named volume o-pgdata-md/cgroup_parent o-prod.shice→
  **o-prod.slice**）；`deploy/o-prod.slice`（CPUWeight 300+双面装配注记）——
  用户面 slice 已装 active；PG-md 容器 healthy 落 slice。
- **② 分链**：migrations 90 件（md 22/fin 68）+`domain.tsv` 59 表三值白名单。
  拆分原则=每表一归属（D-0030 §三/§五：md_account_events/md_equity_curve/
  md_calendar_push 名 md 实 fin 归 fin 链；研究域 fin_ 前缀表归 md 链）；
  app_config 种子/fin 域 DDL 全数归 fin 链件；八桥表 DDL 双链同形落。
  **md 链版本=历史号+1000**（与 fin 键空间全局不相交——同号拆分在并集态实证
  schema_migrations 撞号静默跳件，经验库条目在案）。
- **③ check_migrations 七断言**：原四断言分链化（链内唯一/序号实存/零孤儿/
  md_* 键域）+两链登记单调+八桥表 DDL 归一化同形（列定义全项+CHECK/UNIQUE/FK
  约束+索引集+create_hypertable/保留期；归一化=去注释/去 IF NOT EXISTS/折空白
  后语句集 diff）+白名单（缺册红/串链红；触表面=CREATE/ALTER/INDEX/DROP/
  INSERT/UPDATE/DELETE/trigger ON/hypertable/retention）。
- **④ Migrate(-domain)**：`pgstore.Migrate(ctx,pool,domain)` 前缀选链+非法域
  fail-closed；cmd/core→"fin"；`cmd/migrate` 一次性执行器（开发机载体——
  PG_DSN 入 env 白名单，报备）。
- **⑤ md_funding UNIQUE(venue,inst_id,ts) 双链同加**（fin_0066/md_1066 逐字
  同形）：先去重（保留 recv_ts 最新，D-0018 同键最新赢）再落键。
- **⑥ fin_eval_reject 拆表**（fin_0068）：UNIQUE(inst_id,template,direction,
  reason)；opener.go 235/241 改读写（read 1h 幂等窗 reason 显式列，write
  'funding-hedge' 字面量保真+reason/evidence upsert）；保留账 audit_keep 90d。
- **⑦ fin_active_set 建名**（fin_0069）：inst_id PK+fin_position
  closed_at IS NULL ∪ fin_strategy 非 retired 双触发器行级维护+ev_active_set
  NOTIFY；保留账 state。附 md_macro_event 第八桥表镜像（fin_0067，DDL 与
  md_1020 归一化同形，保留账 state）。

### 桩锚与测试

- `TestMigrateChains`（两链各空库全过+二次幂等）/`TestMigrateChainSurface`
  （域面在位+他域缺席）/`TestMigrateFundingUnique`（键在+拆键造重重放 0066
  去重留最新）/`TestFinActiveSetTrigger`（入集/出集四相）/`TestFinEvalRejectKey`
  （同因覆写异因各行）——O_TEST_PG_DSN 5434 全绿。
- 串链红/DDL 漂移红=check_migrations 断言本体（⑥⑦活体：fin_0044 漏桥 DDL
  与 md_books5 串链在开发中被闸实时拦截修正——闸有效性实证）。
- 跨域测试包（earn/research 五包/review/retention/mt5）改双链并集态
  （md 先 fin 后=历史重放方向；守卫双侧同形——经验库条目钉法）。

### 机检

build✓ vet✓ gofmt✓ check_migrations 七断言✓ check_migration_contract✓
check_env_reads（白名单 26 键）✓ check_rejects_keys✓ check-lines（main.go 300
棘轮内）✓ 经验库 kbgen 重建✓；全仓 go test（-p 4）结果见收口行。

### 验收锚对账

- **开发机 PG-md 空库 Migrate md 链全过：PASS**——live 5435 实证
  （`cmd/migrate -domain=md`：22 件 apply、schema_migrations max=1066、
  16 表=PG-md 全集且零 fin 域表零 app_config）。
- **SG fin 链 Migrate 幂等零变更：PASS（桩面）**——TestMigrateChains/fin
  二次幂等+0066 重放零变更；live SG 新件（0066-0069）随下次 o-core 轮换
  apply（全加法幂等），轮换后复核=交接决策席（§11 四步）。

### 偏离/报备（交决策裁）

1. `cmd/migrate` 件名=票面④执行载体推定（票面未点名件名）。
2. md 链版本+1000 错位=票面未规定的版本方案——并集态撞号根修（经验库
   migration-chain-split-version-keyspace-collision）。
3. env 白名单增 PG_DSN（头注明注「新键须过评审」——D-520 份内先决，报备复核）。
4. **条件请求**：系统面 o-prod.slice 装配（/etc/systemd/system+systemctl
   start）需 root——本席 sudo 无密码缺席；deploy/o-prod.slice 头注双面装配单
   待决策/运维执行（用户面已装，PG-md 已落 slice cgroup）。
5. **下游观察项**：a) app_config 归 fin 而 collector/research/projector
   config.Load 读本池——T-0169 DSN 切换后 PG-md 缺 app_config（测试以并集态
   绕开，生产须裁：投影第 9 桥 or config 二 DSN）；b) review 考据面
   （review/run.go:94-97）与 report rejected 计数（report.go:43）在拆表后
   不再见 SG 钱径拒——覆盖与否归决策（本单票面仅 opener 读写）；c) 清道夫
   双实例+PG-md 保留账面归 T-0169；d) ev_proposal 双侧触发器归 T-0168
   （本票未含）；e) SG 24h 暴力计数复核（基线 2,840）。

## 验收（决策席 2026-10-27 @merge 后落）

**判：PASS——合入 master。**

独立取证对账（非票面转述）：
- PG-md live：`o-postgres-md` healthy 5435、卷 `o-pgdata-md`、容器在
  `/o.slice/o-prod.slice` cgroup；`schema_migrations`=1003~1066（22 件，
  +1000 键空间实证零撞号）；16 表=md 域+八桥正本全集、零 fin 域表零
  app_config（与 domain.tsv 白名单一致）。
- md_funding UQ 在案（venue,inst_id,ts 唯一索引+30d retention policy）。
- check_migrations 七断言重跑绿；build（core/migrate/collector/research）
  +pgstore 测试绿。
- 隧道收窄实物全落：tunnel@ uid=1000+Match 块五项、钥行 restrict+
  permitopen 单口、root authorized_keys 无隧道钥（指纹对拍）、
  PermitRootLogin prohibit-password、fail2ban active、隧道进程以
  tunnel@ 实测在跑、o-collecto/o-research 服务活。
- env 未切（仍 5544→SG）——切换界守 T-0169 ✓。

**要害三笔裁**：①+1000 错位=票面外方案但为撞号根修，收（KB 条目在案）；
②保留账入册行=预修非超单，收；③o-prod.slice 系统面=条件请求成立
（transient cgroup 现态可用但 CPUWeight 未生效+重启漂失——装配令随报发业主）。

**承接观察项**：五条全数钉入 T-0169 票面（含本席新捕 fin_0069 无种子
回填——触发器只随行变，首日集空，切轮换前补种一行）。

票态：✅ 验收合入。

审计席：headless @ 2026-10-30

## 审计注记（headless——merge ff684df 全量 diff 142 件逐锚对账+本机实物面亲核）

**判：审计通过。**

- 验收锚「开发机 PG-md 空库 Migrate md 链全过」：**成立**——码面
  `pgstore.Migrate(ctx,pool,domain)` 前缀选链+非法域 fail-closed
  （migrate.go:16-18）；`cmd/migrate` 一次性执行器在案（-domain 必填、
  PG_DSN 白名单第 35 行登记含理由注）；登记表 90 行=fin 68+md 22
  （md=历史号+1000 键空间不相交）；`TestMigrateChains` md/fin 各空库
  全过+二次幂等桩证在案。实物面本机亲核：`o-postgres-md` 容器实存
  （image timescale/timescaledb:2.30.2-pg18 与票面同版钉、
  HostConfig.CgroupParent=o-prod.slice）、`o-pgdata-md` 卷实存——
  容器当前 exited(0)（2026-10-28T19:16Z 停）属运行态事实，卷在=
  数据面持久，不否定锚所验收之「空库全过」能力与落地件。
- 验收锚「SG fin 链 Migrate 幂等零变更」：**成立（构造性+桩证）**——
  fin 链沿用原版本键 1-69，schema_migrations 以 version 为幂等键，
  改名不动键→已装库重放全跳过；旧文件 1-65 每号 fin_NNNN 为其 fin 域
  子集（md 域子集迁 md_1NNN，SG 不执行），真正新增仅 fin_0066-0069
  全加法件（UQ/镜像/拆表/建集）随轮换一次性补——幂等由 Migrate
  applied-check 构造保证+fin 二次幂等桩证；cmd/core 显式选 "fin"。
- 桩锚三件套：**全成立**——两链各全过=TestMigrateChains；串链红=
  断言⑦（domain.tsv 59 表显式三值白名单判域非前缀，触表缺册/串链
  皆红）；桥表漂移红=断言⑥（八桥表 create/alter/index/hypertable/
  retention 归一化语句集跨链 diff——含列全项+约束+索引集，满足
  D-0032 §三-5 复审注记口径）；七断言脚本 100755 且 pre-commit
  76-77 行实挂载。
- 施工①-④落地件逐一在案：docker-compose.dev.yml（5435 环回绑/
  named volume/cgroup_parent/健康检查）+deploy/o-prod.slice
  （CPUWeight 300+双面装配注记+条件请求明标）；migrations/ 分链
  全量重命名+桥表双链镜像；check_migrations.sh 扩断言；Migrate
  域参签名+两调用方（core→fin、migrate→入参）。
- 施工⓪隧道收窄仓内可核面=deploy/o-collecto/tunnel.service 切
  tunnel@ 低权户+IdentitiesOnly+专用钥+BatchMode（root 钥退役径）；
  SG sshd/fail2ban=主机侧件不入仓 diff，验证径（sshd -T/ssh 拒）
  在施工回填在案——按仓外主机件计，不以 diff 缺位论违。
- 非锚注记（报管理席/决策席备查，不阻塞判）：票面外捆绑件——
  fin_0064-0069 新增迁移、opener.go fin_eval_reject 拆表读写、
  env 白名单+PG_DSN——施工「偏离/报备」五条明列披露非瞒报，
  且均为分链成立的配套面；o-prod.slice 系统面装配（root 依赖）
  票内已自标条件请求项。

### 本席复核补证（headless 重飞 @ 2026-10-30——判词维「审计通过」不变）

- 验收锚①升级实测：临时库 o_audit_md（5434 测试实例新建）
  `cmd/migrate -domain=md` 空库全过（33 版 1003-1076、23 表、二次
  重放零变更幂等）；o_audit_fin `-domain=fin` 同（88 版 1-90、56
  表）；`-domain=both` 非法域拒 rc=1——fail-closed 亲证。
- 验收锚②升级实证：隧道径 SG 只读（FIN_PG_DSN@5544）——
  schema_migrations 含 66,67,68,69（count=90/max=90），
  fin_eval_reject/fin_active_set/md_macro_event/md_funding/
  fin_proposal/fin_opp_eval 六件全在——fin 链幂等键零变更成事实。
- 桩锚负测双红亲跑：fin_ 件触 md 域表→断言⑦「串链」FAIL；
  md 侧独改桥表 md_ticks→断言⑥「DDL 归一化漂移」FAIL；
  本基线 check_migrations 绿。
- 实物补证：PG 隧道进程以 tunnel@+o_tunnel_pg_ed25519 钥在跑
  （root 钥退役径实证）；o-postgres-md 仍 exited、5435 现被
  o-postgres-test-b 占用——运行态漂移照原判不否能力锚，
  归运维承接（与 T-0169+ 谱系运行面一并复核）。
