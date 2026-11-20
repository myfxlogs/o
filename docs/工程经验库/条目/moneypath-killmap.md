---
schema_version: 2
id: moneypath-killmap
kind: map
risk: high
status: verified
keywords: [钱径, 致死面, 耦合点, check_all, 迁移双链, app_config, 部署谱系, 审计图, killmap]
scope: [交付流, 钱径隔离, 迁移闸, 配置闸, 部署闸]
summary: 工单#25759 业主令钱径隔离三件套 T2——「什么动作杀得死钱径」审计图：同仓编译/双链迁移/app_config 同册/部署谱系四耦合点逐一列现有闸门锚+破口面+修补建议；破口三枚立工单在案
first_seen: 2026-11-20
last_verified: 2026-11-20
skills: [施工席, 复审席]
evidence_refs:
  - type: code
    ref: scripts/check_all.sh:87-88（go build/vet ./... 无条件全谱）
  - type: code
    ref: scripts/check_migrations.sh:227（八断言——DDL 同形不管数据面）
  - type: code
    ref: scripts/check_config_write_domain.sh（集同/硬界/覆盖/裸Get 四断言）
  - type: code
    ref: scripts/deploy_watch.sh:61-112（image_dep_clean+o-core-clean 水位径）
---

# 钱径致死面审计图（工单#25759——业主结构性裁问「反复建设对钱径影响」收闸三件套 T2）

口径：钱径=o-core 钱径全谱（internal/{risk,execution,ledger,trading}、
cmd/core）+其依赖闭包。致死面=「建设侧动作（编译/迁移/配置/部署）
使钱径停摆或行为变异」的全部敞口。四耦合点逐一审计，三列制
（闸门锚/破口/修补），破口立工单不私了。

## ① 同仓编译——建设码与钱径码同仓同编译

| 列 | 内容 |
|---|---|
| 现有闸门 | `scripts/check_all.sh:87-88` `go build ./...`+`go vet ./...` **无条件全谱**——任何包编译败全链红，与 merge-scoped/scoped/docs-only 分档无关（分档只裁 go test 段）。绿戳绑 HEAD SHA（`:12` STAMP=check-green-$(rev-parse HEAD)），pre-push 闸验戳放行；land_queue 机械合要求「鲜戳+复审过」双件。 |
| 破口面 | (a) go test 是 **merge-scoped 定向谱**（`:110-167`）——变更包+反向依赖闭包+恒测册（risk/ledger/trading/guard/lifecycle 实证在包集）；**不在闭包不在恒测册的包零测试**——钱径喂数侧（internal/data/*、internal/mt5 采集件）语义破坏只在反向闭包命中时被测，纯数据面件（collector 专属包）破坏=编译过测试零覆盖直进主干。(b) docs-only/mode 豁免档跳 go test 段——分类误判（把 .go 变更判成 docs-only）则编译仍过（build/vet 无条件）但测试全免；分档判据在 check_all.sh:117-167。(c) 脏树跑戳=dirty-ok 附记戳（`:326-331`）——戳键=HEAD sha 而测面含未提交件，复审席须自查戳面 mode 附记。(d) 绕过面=纪律非物理：人工 merge/直推 master 由协作纪律禁，无分支保护物理闸（远端仓库侧未见 required-check 配置实证）。 |
| 修补建议 | ①(a) 立工单：钱径依赖闭包全集入恒测册（`go list -deps ./cmd/core` 实算集 vs 恒测册 diff 入册）——喂数侧包测试零覆盖=编译绿≠行为绿。①(d) 远端 required status check/分支保护=业主仓库设置面件，立工单转业主面（工程侧无权自设）。①(b)(c) 现状可接受——build/vet 全谱兜底+戳附记可审计，暂不立票。 |

## ② fin_/md_ 双链迁移同形——坏 DDL 两链同炸

| 列 | 内容 |
|---|---|
| 现有闸门 | `scripts/check_migrations.sh` 八断言（`:227` 汇总行）：链内唯一/序号实存/零孤儿/md_* 键域/两链单调/**八桥 DDL 同形**（`:188` 桥表须以 bridge\|bridge-reverse 入册对齐）/域白名单/裸GRANT 白名单。`internal/store/pgstore/migrate_table.go` 双链登记序列；`pgstore.Migrate(ctx,pool,"md"|"fin")` 分链应用。 |
| 破口面 | (a) **同形断言只管 DDL 形不管数据语义**——INSERT 参数种/UPDATE 回填在两链各自独立（设计如此：fin 册正本 md 桥副本），但**对存量数据的破坏性 DDL 无闸**：测试库 migrate 跑在空表上，`ADD CONSTRAINT CHECK/NOT NULL/新 UNIQUE` 对生产存量行的违例**编译期不可见、部署期才炸**——迁移应用径（cmd/migrate 或部署钩）若在换件窗内失败=钱径半迁移态。(b) **序号跳空跨支乱序**：在途支各占号段（实证 fin_0215/0216、md_1100 并存在途），后落支序号小者后应用——`ON CONFLICT DO NOTHING` 幂等键种无虞，但**带顺序依赖的 DDL 对（如先 DROP 后 ADD）跨支倒挂=静默语义反转**，八断言不管跨支时序。(c) 桥表「同形」归一化比对——两链各自合法但**语义分叉**（同名列不同默认值/不同 CHECK 域细节归一化吃掉）断言不报。 |
| 修补建议 | ②(a) **立工单**：迁移生产预演闸——部署前对最近快照/影子库跑 `pgstore.Migrate` dry-run（或 lint 禁令：大表 `ADD CONSTRAINT` 必须 `NOT VALID`+分离 `VALIDATE` 形）。证=check_migrations.sh:227 断言清单无「存量数据适用性」项+测试 migrate 恒空表。②(b) 立工单：跨支迁移序号依赖声明（migration 文件头 `depends_on=` 或 land_queue 合流时序校验）——现状靠人肉避让。②(c) 并入②(a) 同票——同形断言加列级默认值/CHECK 域原文比对项。 |

## ③ app_config 同册——配置写域与钱径参数同表

| 列 | 内容 |
|---|---|
| 现有闸门 | `scripts/check_config_write_domain.sh` 四断言全绿实证（`dm=24 行 owner=23 行 universe=108 键 bare=14 读点`）：集同（键集对拍）/硬界（写域分册——`config-write-domain.tsv` owner/dm 域外禁写）/覆盖/裸Get 白名单（标量键禁 `.Get` 裸读走类型化读器）。app_config 单表+generation+LISTEN 热启（`internal/platform/config/client.go:33-60`）。 |
| 破口面 | (a) **闸管结构不管值域**——键的写域归属/读器纪律全覆盖，但**值合理性零校验**：DM 域内写错型/错量值全闸绿过（GetFloat 类型化读器 fail-closed 兜底=消费方报错跳过，爆炸半径=功能静默停非钱径直丢）；**但 gate.kill.*/风控阈值类键写坏=守卫/闸主动停摆**——kill 闸语义=「kill=true 停功能」，写域闸不审值=误写 true 即停钱径守卫件。(b) app_secrets 分表+cryptoutil 解密（`client.go:131-145`）——密钥面独立已隔离 ✓。(c) 写径审计=ops_event 账可回放，但**无值快照对拍闸**——事后只能查「谁写了」不能查「写对没」。 |
| 修补建议 | ③(a) **立工单**：高危键值域闸——`gate.kill.*`/仓位上限/风控阈类键建值域 schema（bool-only/正值域/range），config.Set 写径过校验或 DMSet 域收窄至逐键白名单值域。证=check_config_write_domain.sh 断言清单无值域项+app_config 无值 CHECK（jsonb 列结构性不可加）。③(c) 并入同票——ops_event 配置写事件带值指纹便于回放对拍。 |

## ④ 部署谱系——镜像换件判据与钱径水位

| 列 | 内容 |
|---|---|
| 现有闸门 | `scripts/deploy_watch.sh:61-112`：`image_dep_clean`（`:62-90`，`go list -deps` 实算 cmd/core 依赖闭包——diff 件全在闭包外=不换）+`o-core-clean` 水位行（`:103-112`，谱系证空落 DEPLOY/部署 水位行跳本拍，水位不抵实部署）。换件本体=`deploy_image.sh`（SG o-core 镜像）；OCI 侧=`deploy_oci.sh`（collector/research 臂）。 |
| 破口面 | (a) **o-core-clean 跳部署判据只看「镜像依赖闭包」**——`migrations/*.sql`、`deploy/`、`scripts/`、docs 全在闭包外：**纯迁移 commit 合流后镜像零 diff→跳部署→迁移本体谁应用？**若迁移应用绑在镜像换件链（cmd/migrate 随镜像或部署钩），迁移-only 变更=**迁移饿死**（库结构落后于码期望=钱径读新列炸/写旧表丢）——此为本图最实破口，证=deploy_watch.sh:101-105 跳部署分支不查 pending migration 态。(b) deploy_watch 自身存活无闸——watcher 死=谱系判不触发=merge 后钱径跑旧码无告警（ops_event DEPLOY 水位可查但无人盯 watcher 心跳）。(c) o-core-clean 水位以「最近一次证空 SHA」记——**证空后同 SHA 再有闭包内变更**（理论上不可能同 SHA 变内容，SHA 绑定已防）✓。(d) 双链部署谱系：SG(o-core) 与 OCI(collector/research) 分谱——钱径只在 SG，但 **app_config 同册**（③）意味着 OCI 侧配置写可影响 SG 钱径消费（跨机同库）——部署谱系隔离≠配置面隔离。 |
| 修补建议 | ④(a) **立工单（最急）**：deploy_watch 跳部署判据补 pending-migration 咨询——合流面含 migrations/*.sql 时即使镜像证空也须走迁移应用径（或独立迁移 watcher）。证=deploy_watch.sh:101-112+本图②(a) 联动：迁移饿死×存量数据炸=双杀链。(b) 并入同票——watcher 心跳行+watchdog（参照 roster.BeatLoop 先例）。④(d) 记档不立票——跨机配置同库是设计事实，值域闸（③a）覆盖其风险面。 |

## 破口立票汇总（施工侧已核 ops_event 无在案同题——立票归复审/裁定席径）

| # | 破口 | 立票建议题 |
|---|---|---|
| B1 | 钱径喂数侧包（internal/data/*、internal/mt5）在 merge-scoped 测谱恒测册外——编译绿≠行为绿 | `工单/moneypath-test-closure：恒测册补 go list -deps cmd/core 依赖闭包全集` |
| B2 | 迁移对存量数据破坏性 DDL 无预演闸（测试恒空表） | `工单/migration-rehearsal：部署前迁移 dry-run/影子库预演或 NOT VALID 强制形` |
| B3 | 跨支迁移序号顺序依赖无声明机制 | `工单/migration-order-declare：depends_on 声明或 land_queue 时序校验` |
| B4 | 高危 app_config 键（gate.kill.*/风控阈）值域零校验 | `工单/config-value-domain：高危键值域 schema+写径校验` |
| B5 | **迁移-only 变更 o-core-clean 跳部署→迁移饿死**（本图最急破口） | `工单/deploy-migration-watch：deploy_watch 跳部署判据补 pending-migration 咨询` |
| B6 | deploy_watch 自身存活无心跳闸 | `工单/deploy-watch-heartbeat：watcher 心跳+watchdog` |

## 总评

四耦合点中①③为「结构闸已罩、值域/测试闭包残面」级破口（B1/B4）；
②为「DDL 形闸罩、数据语义/跨支时序残面」（B2/B3）；④(a) 为
**唯一「闸判据本身漏一类变更」级破口**——o-core-clean 谱系证空
判据的完备性假设（「镜像依赖闭包外=零影响」）不成立于迁移面，
与②(a) 串成「迁移既可能炸存量、又可能根本不被应用」双杀链，
B5 应最先处置。钱径隔离三件套 T1/T3 落地后，本图破口面即
残余风险全谱——复审时按 B1-B6 逐条销账。

证据锚：scripts/check_all.sh:87-88,110-167,326-331；
scripts/check_migrations.sh:188,227；scripts/check_config_write_domain.sh
（四断言输出行）；scripts/deploy_watch.sh:61-112；
internal/platform/config/client.go:33-60,131-145；
internal/store/pgstore/migrate_table.go。
实证环境：origin/master@2a33a19f1 核读（2026-11-20）。
