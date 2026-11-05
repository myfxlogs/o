# T-0322 业主面减负替代包——o_fin_ro 窄读 + COS 秘书席 + 草案开放立票

| 字段 | 值 |
|---|---|
| 状态 | **完成**（无头决策席验收通过 @ 2026-11-04——见「决策席验收注记」；合并 3347c391） |
| 立项 | 业主令 @ 2026-11-04（`[角色:讨论]` session 代笔——原取 T-0321 与他席撞号让号重编——业主明批秘书席方向并钦定英文缩写 **COS**=Chief of Staff） |
| 领取 | devin-t0322 @ 2026-11-04 |
| 优先级 | 中 |
| 涉及面 | `migrations/`（PG 角色授权）、`docs/collab.md`（席位章+立票规则修订）、`docs/员工名册.md`（COS 职卡）、`docs/数据流.md`（凭据面同 commit 更新）、`fin_employee_registry`（在册行） |

## 背景

11-04 讨论 session 实证三件摩擦：①分析/讨论席因最小权限读不到钱面表（fin_position/ledger_snapshot 等），对账讨论只能空手；②非决策席立票须业主逐件转发；③业主被逐件追问=真瓶颈。业主否决「业主席位」案（不可逆三权归 LLM=爆炸半径问题），钦批**替代包**：秘书席 COS——**准备决策不做决策**。

## 目标

业主从「逐件转发追问」变「批一页打包件」；分析席拿到钱面只读事实；草案入口对全席开放——三件事一次落。

## 内容三件套

**A. `o_fin_ro` 只读角色**
- SELECT-only（无 INSERT/UPDATE/DELETE），白名单：`fin_ledger_snapshot`/`fin_position`/`fin_strategy`/`fin_opp_eval`/`fin_eval_reject`/`fin_proposal`/`ops_*` 全族；
- 黑名单：`app_secrets` 及一切凭据物料列——结构性不授；
- 授予对象：分析/讨论类只读席（COS 亦挂此角色读事实）。

**B. COS 秘书席（Chief of Staff）**
- 职卡：盯请求箱/条件请求/待业主裁决项→每件打包「选项+建议+证据」一页上行→业主批复即转对应席执行；
- **写面唯一**=往「待业主裁决」队列/通知径投递；动钱/立票/审批/部署一概不沾；
- 读面=o_fin_ro（含钱面只读——打包决策要事实）；
- 禁线：不得以任何形式代业主拍板——打包件必须含「业主裁」空位。

**C. 草案开放立票**
- `collab.md` §2 修订：草案（draft）态允许任意角色立（审批闸 `草案→已批` 仍归决策席独占）——把「立票」拆成「提案入口全开放/批准入口不开放」；
- 配套：票面「立项」字段记实际立票席，出处可追溯。

## 验收标准

- [ ] `o_fin_ro` 角色建册在 migrations，白名单表 SELECT 通、黑名单表 SELECT 拒——实测贴出
- [ ] COS 职卡入 `docs/员工名册.md` + `fin_employee_registry` 在册行（职责/产出/合格判据/越权线四件齐）
- [ ] `collab.md` §2 草案条款修订落字（开放立票+决策独占审批双写）
- [ ] `docs/数据流.md` 凭据面同 commit 更新（o_fin_ro 行入）
- [ ] COS 首件交付实证：一页「待业主裁决」打包件产出（真实条目或演练皆可）

## 范围外

- 业主席位/业主级凭据一律不建——不可逆三件（资金扩投/新平台/席位立换）永久归人；
- COS 不接告警分诊/研究/裁定任何现职——纯业主办公室功能；
- 不改 ops_event/roster 机制本体（COS beat 走既有件）。

## 参考

- 11-04 对话录（业主令原文+否决业主席位理由）；
- D-0040 权序（CEO 四扳手先例——权力按 scoped instrument 授予的先例）；
- `docs/数据流.md` 凭据面（最小权限设计原文）。

## 结果回填

**实现摘要**（分支 `t/0322-cos-ro-draft-open`）：
- A `o_fin_ro`：`migrations/fin_0106_o_fin_ro.sql`——LOGIN 角色（口令产出面手工置，collector_rw 先例）；SELECT 白名单 fin_ledger_snapshot/fin_position/fin_strategy/fin_opp_eval/fin_opp_hypothesis/fin_opp_template/fin_eval_reject/fin_proposal/fin_capability_surface/ops_* 全族（含两视图）；`app_secrets` REVOKE ALL 结构性钉死，写权零授。fin_capability_surface=md 域表（fin 链独立 schema 测面缺席）→条件授权（存在面必授）防迁移炸。
- B COS：`migrations/fin_0107_cos_registry.sql` 在册行（dept=platform，meta.form=pure_interactive）；`docs/员工名册.md` 新节「公司席」职卡四件齐；`skills/COS/SKILL.md` 职卡（收件箱三源 SQL/打包件模板/写面窄白名单）；`docs/cos/packs/2026-11-04-o_fin_ro-condition-gap-grant.md` 首件打包实证（真实条目——本单新证）。
- C `docs/collab.md` §2 立票开放段落+生命周期表 `草案→已批` 动作方改 COS（§1 写面判据衔句）；`docs/数据流.md` 凭据面 o_fin_ro 行入；`skills/索引.md` kbgen 重建。

**测试证据**（本机 dev 库 127.0.0.1:5433，SET ROLE o_fin_ro 实录）：
```
SET ROLE o_fin_ro; SELECT count(*) FROM fin_position;   -- whitelist-ok | 0（通）
SET ROLE o_fin_ro; SELECT count(*) FROM fin_opp_eval;   -- opp-ok | 6577（通）
SET ROLE o_fin_ro; SELECT 1 FROM app_secrets LIMIT 1;   -- ERROR: permission denied（拒）
SET ROLE o_fin_ro; INSERT INTO ops_event(...) VALUES... -- ERROR: permission denied（写拒）
SET ROLE o_fin_ro; SELECT 1 FROM fin_condition_gap;     -- ERROR: permission denied（白名单外拒——缺口实证）
registry: COS|platform|COS 秘书席——业主办公室（准备决策不代拍板）|active|pure_interactive
```
机检：`scripts/check_all.sh` 二跑**全谱绿**（戳在案）——build/vet/import/gofmt/env/rejects/migrations（七断言）/bridge-audit/go test；首轮 events 包 `TestListenSocketStarvationRootFix` 曾红（33s 超时敏感受并行载波动），单跑 PASS——本 diff 零涉该面。

**偏离说明**：
- 批准闸措辞：票面「仍归决策席独占」与 §1 席面分家写面判据（`草案→已批` 只属 COS 交互形态）冲突——按施工提示词钦定句+§1 落 COS 闸（提案开放/批准不开放双写语义不变）。
- **缺口升件**：`fin_condition_gap`（COS 收件箱源②）不在钦定白名单——停线禁自扩，已落 ops_event `升件` 行（施工席写面）+打包件上呈请决策补裁；SKILL 源② SQL 已按钦定口径落字并注「授权落地前缺席=如实记读面缺」。
- 角色口令与 DSN 供给=产出面手工件（同 collector_rw 径），seat 上线清单已随升件行带出。

审批注记（决策交互形态 @2026-11-04）：**已批**。A o_fin_ro 窄读+黑名单结构性不授=最小权限正法；C 草案开放立票与新章咬合（批准闸仍本席独占）；B COS 职卡方向业主钦批在案、四件落字归施工件，收件箱即升件队列兜底面（collab §1 两形态定章同语义）。施工提示词另立。

## 决策席验收注记（无头拍 devin @ 2026-11-04）

裁词=**通过**，合 master@3347c391（分支尖 fea91eeb=收口 2c51bd3c+机检终态回填一行，票面文档件域内）。

- 范围红线过：diff 10 件 ⊆ 涉及面——migrations/fin_0106+0107、migrate_table.go 册登记（迁移必载机件，先例同形）、collab §2/员工名册/数据流/skills/COS+索引、docs/cos/packs 首件打包实证、票面自回填。
- 门禁绿在案：check_all 戳 `check-green-fea91eeb`（戳只写全绿后——build/vet/import/gofmt/env/rejects/migrations 七断言/bridge-audit/go test 全谱）。
- 验收锚实迹（本席 dev 库 o-postgres@5433 亲跑复跑，非口报）：
  ① 白名单通：SET ROLE o_fin_ro SELECT fin_position=0 / fin_opp_eval=6577（与回填实录逐字同）；
  ② 黑名单拒：app_secrets SELECT=permission denied；写拒：ops_event INSERT=permission denied；
  ③ registry 行 COS|platform|active|pure_interactive 实录在案；
  ④ fin_condition_gap 拒=白名单缺口实证（升件行1564 同源）。
- 停线纪律正面在案：白名单外表不授（条件授权 DO 块防迁移炸——fin_capability_surface 生产真存，to_regclass 全 14 物亲证）；app_secrets REVOKE 显式钉死非静默漏。
- 升件补裁（本席份内，非红线三件）：升件行1564+打包件 docs/cos/packs/2026-11-04-o_fin_ro-condition-gap-grant.md——采纳建议A，白名单补 `fin_condition_gap` SELECT（COS 源②请求盒承载表正是秘书席该读面）；落 `docs/prompts/T-0322-返修①-fin_condition_gap白名单补授.md` 常径派单。
- 部署后验项：fin_0106/0107 生产迁移+o_fin_ro 口令产出面手工置件——归 T-0326 部署链收口五锚销，本票不判红。

### 决策席验收注记②——返修①（o_fin_ro 补授 fin_condition_gap，无头拍 devin @ 2026-11-04）

裁词=**通过**，合 master@199c475c（分支 `t/0322-condgap-grant` 尖 02673f7e）。

- 范围红线过：diff 恰 4 件 ⊆ 返修①提示词清单——migrations/fin_0108、migrate_table.go 册登记 {108}（fin 链 107 后 md 链前）、docs/数据流.md 凭据面白名单串、skills/COS/SKILL.md 源②注改「已授」。
- 门禁绿在案：本席于 worktree t0322r 亲跑 check_all 全链绿——戳落 check-green-02673f7e（build/vet/import 129-138边/gofmt/env 30键/rejects/migrations 七断言/bridge-audit 四列/go test 全谱）。
- 验收锚实迹（本席 dev 库 o-postgres@5433 亲跑复跑，非口报）：
  ① schema_migrations 108 在列；SET ROLE o_fin_ro; SELECT count(*) FROM fin_condition_gap=1（通）；
  ② 回归钉：app_secrets SELECT=permission denied；ops_event INSERT=permission denied——补授不伤黑名单/写禁。
- 停线纪律正面在案：单行 GRANT SELECT 一表，他表零授；o_fin_ro 写权仍零、凭据域结构性仍拒（②实证）。
- 瑕疵注记（非判红）：票面未落返修①回填段——锚摘要走 ops_event 收口行（id 1580/1581）可稽，本注记补位。
- 部署后验项：fin_0108 生产迁移随行——归 T-0326 部署链收口五锚同销，本件不判红。
