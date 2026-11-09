# 纪元3 全项目清洁——死件清单（cleanup-inventory）

> 施工：外席会话 @/opt/o-wt/cleanup（t/cleanup）｜令：`docs/cleanup-prompt-epoch3.md`（业主令 2026-11-09）
> 判据：死因三选一（零引用实测 / 语义只属纪元2 / 被替代件取代）+核验后定。
> 「零引用必须实测」——本表每行处置均附核验径；拿不准进文末搁置清单。

## 一、scripts/（45 顶层件+4 子目录）

### 1.1 删——一次性取证/校准件（用毕，双宿主零调用）

| 路径 | 死因 | 处置 | 引用面核验 |
|---|---|---|---|
| scripts/calibrate_recon.sh | T-0055 一次性基线校准（recon_baseline_adjust 已落） | 删 | `rg -l calibrate_recon` 全仓零命中；dev/OCI cron+systemd 零调用 |
| scripts/dualwrite_diff.sh | T-0169 双写对拍一次性件——PG-md 切换已毕，dualwrite 期终结 | 删 | `rg -l dualwrite_diff` 全仓零命中（仅自身） |
| scripts/ledger_hash.sh | T-0169 切换指纹一次性件——切换验证已毕 | 删 | `rg -l ledger_hash` 全仓零命中 |
| scripts/test_tron_apis.sh | T-0192 TRON API 功能证明一次性件——TRX 自营产线已立（63716222c） | 删 | `rg -l test_tron_apis` 全仓零命中 |
| scripts/anchors_snapshot.sh | T-0169 ④ 72h 反证锚快照一次性取证件 | 删 | 仅 docs/容错位清单.md 历史提及（清单=活件但此为在案记录非调用） |

### 1.2 留——活引用实证

| 路径 | 活证 |
|---|---|
| scripts/check_stale_base.sh | docs/总纲.md:68 boot 步⓪a 钦定引用（总纲=业主钦定层禁擅改——删件留悬空引用不可）；容错位清单同引 |
| scripts/audit_app_config_types.sh | internal/platform/config/dmwrite.go:164 注释钉「audit_app_config_types 普查口径」——canon 读器判据件 |
| scripts/deploy_at_sha.sh | 部署管道面（按绿戳发镜像=mirror_push 生态部署半）；决策/链路文档在案 |
| scripts/deploy_sync.sh | **实测活**：SG 宿主 `crontab: 0 3 * * * /opt/o/scripts/backup_pg.sh`——SG /opt/o 源树靠本件同步保鲜（SG 实测 go.mod mtime=今日） |
| scripts/build_host_tools.sh | 总纲/架构约束 canon 引用的宿主工具构建件（工具面仍在长——kbgen/skillgen 等） |
| scripts/{arm,cognition,hooks,oci_ci}/ | arm=OCI /usr/local/bin 在跑备份臂；cognition=唤知机（o-cognition.timer active，o-cognition-dispatch 实跑中）；hooks=git 钩全活；oci_ci=OCI CI runner 在册 |
| scripts/check_*.sh 全族+testdb_lease+config-write-domain.tsv+env-allowlist.txt+check-lines.* | check_all 免疫链+pre-commit/pre-commit/commit-msg/pre-push 钩调用面（逐件 rg 实证） |
| scripts/{backup_pg,backup_push_sg,gc_test_schemas,image_tag_gc,master_push_converge,mirror_push,git_push,oci_bootstrap,oci_ci_*,deploy_image,deploy_oci,provision_worktree,bootstrap_worktree}.sh | 宿主 cron/单元/免疫链/部署管道活引用（dev cron gc_test_schemas 日拍；OCI /usr/local/bin 五臂；o-mirror-push.timer 等） |

## 二、tools/（44 件）

### 2.1 删——一次性取证/迁移/验收件（用毕）

| 路径 | 死因 | 处置 | 引用面核验 |
|---|---|---|---|
| tools/alertprobe | T-0030 告警真推实证件 | 删 | `rg -w alertprobe` 全仓零命中 |
| tools/billsaudit | T-0055 对账差根查件（用毕） | 删 | 零命中 |
| tools/credx | E-1 凭据提取一次性迁移件 | 删 | 仅 SECURITY_MODEL.md 范式引用（件已用毕） |
| tools/dcdprobe | T-0101 字段名取证件（用毕） | 删 | 零命中 |
| tools/feishumig | T-0028 飞书凭据迁移件（用毕） | 删 | 零命中 |
| tools/feishuprobe | T-0028 实证件（用毕） | 删 | 仅 tools/feishusend 注释同族提及 |
| tools/fillsprobe | T-0085 只读取证件（用毕；pre-commit 仅事故注释提及） | 删 | pre-commit:189 注释=历史事故记录非调用 |
| tools/legpxfill | T-0179 px 补录——「补 62/63」已完结 | 删 | 零命中 |
| tools/overviewprobe | T-0086 RPC Overview 验收件（用毕） | 删 | 仅 tools/hlprobe 注释「同形」提及 |
| tools/probe2 | T-0102 验收修补探针（远古用毕） | 删 | depblind_test.go 仅测试失败标签字符串「probe2:」同名巧合非引用 |
| tools/rgen | T-0026 dev 报表实证件（用毕） | 删 | 零命中 |
| tools/scanonce | T-0018 dev scan 实证件（用毕） | 删 | 零命中 |
| tools/keypipe | T-0184 私钥管道一次性迁移件（用毕） | 删 | 仅 SECURITY_MODEL.md 范式引用 |

### 2.2 留——活引用/通用件

reconview（对账自助面，簿记件在跑径）、legarb（今日合入 ambiguous 腿仲裁取证）、okxread（通用读件，reconview 注释同径）、openingequity（T-0034 通用 opening 径——venue 接入首账必录机制件）、mt5probe+mt5wrap+mt5verify+mt5ordstream+mt5stream+mt5close（mt5 skills canon 在用族+runbook 引）、okxverify+okxlev+okxunwind（okx skills canon 在用族）、dmconfig/guardlift/hypo/kbgen/skillgen/skillpatch/capcheck/openfirst/swapsurvey（码面/skills/钩活引用）、secput（秘件入库通用运维件）、poschk（只读仓位快查通用运维件）、mt5verify/mt5wrap（runbook+reconview 引）

### 2.3 搁置（拿不准——不强裁）

| 路径 | 疑点 |
|---|---|
| tools/posprobe | T-0511 今日取证件——closing 滞留根修已合，取证或仍需复跑对拍 |
| tools/mtonboard | T-0379 broker 接入干跑件——首个 broker 已通，后续 venue 接入或复用 |
| tools/dustsweep | 票面自标「一次性运营动作」但 mutating 钱径面+SECURITY_MODEL 在引——钱径邻接慎裁 |
| tools/rescuespot | 事故救助 mutating 件——同上 |
| tools/legrestore | T-0104 事故归位件（用毕）+D-0013 canon 引+D-0013 在引——钱径邻接 |
| tools/okxprobe | E-1 凭据→余额探针——与 okxread/okxverify 能力重叠判据不明 |
| tools/hlprobe | T-0263 HL 验收实证件——HL venue 是否在册未核 |
| tools/moneyinv | 自标「临时诊断件不提交」但在册+COS 卡引（卡本批封存后或零引）——钱径只读面 |
| tools/feishusend | 通知微件——deploy 实走 /opt/meta/scripts/feishu-notify.sh 非此件；通用性存疑 |

## 三、skills/ 席位卡七件+仪式件

### 3.1 封存（纪元2 角色契约——活面零引用，内容保留不删）

| 路径 | 死因 | 处置 | 引用面核验 |
|---|---|---|---|
| skills/{CEO席,COS,CTO席,施工席,分析席,外审席,裁定席}/ | 席位派发整层纪元3 裁撤（D-0045）；卡=epoch2 角色契约 | `git mv` → `docs/纪元2归档/席位卡/`（封存非删——arch tag 另有全量底片） | AGENTS.md/runbook/活脚本对卡文件零引用（席位**词**在码/库存续=事件词表非卡引用）；run.sh 唤知会话只读 runbook.md+skills/ 通用面不指定席卡 |
| skills/阶段交付/ | epoch2 Phase 交付仪式件——纪元3 收口=runbook④ 自查退场，无此仪式 | 同上封存 | `rg -l 阶段交付` 全仓（除自身）零命中 |

### 3.2 留

能力账族全留：mt-*/okx-*/ctx-sources/econ-*/pg-capacity/协议面取证/生产取证/告警分诊/策略评估与仓位/研究员（各有码面/skills 索引/文档活引）；skills/{README,索引}.md 由 kbgen 重建。

## 四、docs/ 考古面

### 4.1 封存 → `docs/纪元2归档/`（epoch2 仪式/产物/被取代设计稿）

| 路径 | 死因 | 引用面核验 |
|---|---|---|
| docs/cos/（packs×2） | COS 秘书席 epoch2 产物 | 仅 skills/COS 卡（同批封存）+migrations 历史行引 |
| docs/design/（PNG×26+论证+皮肤提示词） | epoch2 UI 设计探索件 | `rg docs/design/` 码面零命中 |
| docs/ops/worktree-retire-2026-11-05.txt | 一次性退役记录 | 零命中 |
| docs/prompts/t0348-assets/ | epoch2 提示词资产 | 仅 skills/CTO席 卡（同批封存）引 |
| docs/templates/ticket-draft.md | 票草案模板——票机裁撤（tools/ticketdraft 已删） | 零命中 |
| docs/FIX02-收口.md / docs/checkall-docs-收口.md | epoch2 收口回执件 | 零命中 |
| docs/前端布局设计-v1.md | 被 v2 取代（v2=业主令修订版在案） | 零命中 |
| docs/异常类谱系.md | T-0332 自愈巡检谱系——巡检面 epoch2 裁撤；谱系本身为判据形参考 | 零命中（判：封存保留可查） |

### 4.2 留

法典四件（总纲/工程纪律/架构约束/数据流——总纲业主钦定层）、collab.md（任务三真整理对象）、decisions/、纪要/、infra/（提案+GW-001 在案决策记录）、venue/（md_1088 迁移引 IBKR 候选 canon）、能力账/（8 处活引）、运维/（OCI盒档案 5 处活引）、research/场内产品面盘点（T-0396 策略扩张候选池 canon）、runbooks/T-0356-volsell-revival（volsell 族活策略复役规程）、前端布局设计-v2.md（现行设计稿）、工程链路症结盘点-2026-11-08.md（11-08 链路综览——o-gateway 设计引据，时效内）、容错位清单/机会模板书/能力账本（活引在案）、README/AGENTS/SECURITY_MODEL/docker-compose*/Dockerfile.core（基建面）、cleanup-prompt-epoch3.md（本令原件——untracked 不入 commit）。

## 五、宿主侧（dev+OCI+SG）

| 项 | 死因 | 处置 | 核验 |
|---|---|---|---|
| dev `~/.config/systemd/user/{o-collecto,o-projector,o-research}.service`+`o-collecto.service.d/` | 三单元 disabled+inactive——采集/投影/研究臂纪元3 全归 OCI（OCI 同名单元 active 实证）；dev 副本=迁移前残留 | 删（先 tar 封存 ~/o-retired-units-2026-11-09/） | `systemctl --user is-active/is-enabled` 三件全 inactive/disabled；unit 内容 ExecStart 指 /opt/o-* 本机二进制 |
| dev `~/o-collecto.env`、`~/o-projector.env` | 上述死单元 EnvironmentFile——唯一消费者已死 | 同上封存随件 | rg 引用面：仅死单元文件引用（OCI 侧 /home/ubuntu/o-*.env 另有活件不动） |
| dev `/opt/o-collecto/`、`/opt/o-projector/` | 死单元二进制+bak 群（~8 件 .bak 含 collector.bak-t0191 等） | 删（tar 封存） | 无存活引用——对应单元已死 |
| dev `/opt/o-research/` | **拆半**：`env` 文件=活（cognition run.sh/feishu-notify/deploy_image/eventbus/bridge-registry 五处读 FIN_PG_DSN）；`o-research` 二进制+.bak 群=死（唯一消费者 o-research.service 已死） | 删二进制+baks，**env 留勿动** | 五处活引用实证（scripts/cognition/run.sh:16 等） |
| OCI `~/.config/systemd/user/o-sweeper.service.bak-t0232` | .bak 残件 | 删 | OCI 单元面实测——活件=o-collecto/eventbus/projector/research/sweeper/ci-runner 六件 |
| `/opt/o-wt/*` 陈树 | merge 毕树收（prompt 钦定） | 逐树核：merged→`git worktree remove`；unmerged/脏→留 | `git worktree list` 70 树对账——merged 枝 49、unmerged 20、detached 若干（执行段逐件判） |

## 六、经验库（任务二详表在执行段回填）

| 项 | 预判 | 依据 |
|---|---|---|
| docs/工程经验库/待归纳.md | 删（逐节判：canon 化删行/未 canon 转正/epoch2 专属标弃） | 业主令 10-25 废 staging 层 |
| docs/经济经验库/待归纳.md | 删（空壳） | 同上 |
| docs/工程经验库/前身-nox/（275 件） | **留档原位+README 注归档态** | 138 条 canon 条目 evidence/正文引前身-nox=引用价值实证；kbgen 明示「参考区不进索引」 |
| docs/经济经验库/前身-nox/（10 件+裁定账） | 同上留档+README 注归档态 | 裁定账=分诊链账本；案文件与条目/ 同名件对照留档 |
| 条目/ status=superseded | 实测=0 件——归档面改为「针对物已死的 epoch2 条目」判读面 | `grep status:` 分布 verified344/observed31/mitigated5/canon3/fixed2/draft1/active1/superseded0 |
| 索引/检索物料/技能清单 | `go run ./tools/kbgen gen`+`gen-econ`+skills 全量重建 | 生成面与目录对拍零漂移 |
| README.md | 改写现行组织法（分层/归档规约/新条目流程） | 业主令 10-25「有坑即写条目+kbgen 重建」钦定径在文 |

## 七、collab.md（任务三）

覆盖条判词执行：死段（席位/票/相位/草案/物料化/metabus/收口语/自报/名册权序等 epoch2 节）实质移除或压入文末「纪元2 史证」附录；活段（§6/§7/§10/§11/数据流正本指引及 §4/§9 存活部分）保留正文。**勿 merge sigfix 支 f01ba9945 批注版**——本单真整理取代之。

## 八、搁置清单（拿不准——呈审不强裁）

1. tools/ 九件（§2.3 全列）
2. scripts/build_host_tools.sh——cron 裸奔前提已陈（宿主无 go 二进制 cron 调用实测）但法典引用在案，删留两可
3. docs/异常类谱系.md——谱系判据形或仍有判读价值，封存与留档两可（暂归封存批）
4. dev `/opt/o-research/` 二进制面——若 OCI/他径另有调用未覆盖即误删（已核五处 env 活引，二进制面仅死单元消费——置信高但列此备查）

## 九、验收锚对账（prompt 钦定）

- [ ] check_all mode=full 绿戳（收口前 rebase master 后盖）
- [ ] 本表行数 vs `git rm`/`git mv` 实数对账（执行段回填实数）
- [ ] 每项处置有死因引证（本表核验列即引证）
