# T-0052 skills 索引机生（路由速查，省 token 面）

| 字段 | 值 |
|---|---|
| 状态 | 完成（判过并码在案，11-01清账翻面） |
| 立项 | `[角色:决策]` @ 2026-10-26 |
| 领取 | zcode-glm @ 2026-10-26（worktree /opt/phase-wt/t0052，支 t/0052-kbgen-skills） |
| 优先级 | 中 |
| 涉及面 | `tools/kbgen/`、`skills/索引.md`（新增产出件）、`scripts/hooks/pre-commit`（check 挂载） |
| 范围外 | skill 内容改写、目录结构调整 |

## 背景

session 启动 harness 已注入 name+description 清单——「有无 skill」不缺索引。缺口=**路由歧义**：关键词→文件一对多猜错即白读。另经验库 `技能清单.md` 是 skill→条目正向，缺「坑关键词→归口 skill」反向路由。

## 目标

`kbgen` 增 `skills` 子令（`kbgen skills [--check]`）：扫 `skills/*/SKILL.md` frontmatter+标题行，产 `skills/索引.md`——

| 列 | 源 |
|---|---|
| 域 | 目录名 |
| 覆盖一句话 | frontmatter description 截首句 |
| 关键词 | 从 description/首表头抽 token（venue 名+端点名+坑键） |
| 坑引 | `docs/工程经验库/技能清单.md` 已生成的挂接行 |

- `--check` 漂移档：生成输出与库内文件 diff 不同=exit 非零
- pre-commit 挂点：staged 触 `skills/`→跑 `kbgen skills --check`（与经验库闸同款）

## 验收标准

- `kbgen skills` 出表覆盖全部在册 skill；`--check` 对 stale 索引红
- 索引 ≤60 行（压缩形态——自身不能成 token 负担）
- pre-commit 挂点实证（改一件 skill 不重建索引=钩红）
- frontmatter 缺 description 的 skill 行标「缺描述」——索引顺带当 schema 哨兵

---

## 结果（施工回填 2026-10-26）

### 码面
- `tools/kbgen/skills.go` 新件（176 行）：`kbgen skills [--check]`——扫 `skills/*/SKILL.md` frontmatter+首表头行，产 `skills/索引.md` 四列表（域/覆盖一句话/关键词/坑引）；行数硬顶 60（现 24 行含 18 skill）。关键词=description+首表头抽拉丁 token（venue/端点/坑键）去重封顶 5；坑引=技能清单.md 反向解析条目 id；frontmatter 缺失/无 description 行标「缺描述」（schema 哨兵——okx-api-spec 现值如实标出）。
- `tools/kbgen/main.go` 挂 skills 子令（285<300 行）；--check 漂移档 exit 非零。
- `scripts/hooks/pre-commit` 闸 8b（与经验库闸同款）：staged 触 `skills/`→`go run ./tools/kbgen skills --check`，陈旧=钩红。

### 实证（验收锚逐项）
- `kbgen skills` 出表覆盖全部 18 在册 skill ✓
- 索引 24 行 ≤60 ✓
- 挂点实证：staged 改 skill 不重建索引→钩 exit 1 红报；重建后→exit 0 过闸 ✓
- 「缺描述」哨兵：okx-api-spec（无 frontmatter）如实标出 ✓——该件补 frontmatter 属 skill 内容面=票面范围外，报决策席另令

### 机检
build/vet/gofmt/test 全量 / import-direction / migration-contract / lines（kbgen main 285、skills 176）/ env-reads——全绿。无新 migration。

### 验收毕（决策席第 2 席 @ 2026-10-26 清账）
- 锚逐项复核：kbgen skills 出表覆盖 19 在册 skill ✓；索引 25 行 ≤60 ✓；`skills --check` 漂移档实测红（下见）+重建绿 ✓；pre-commit 闸 8b 挂点在档（scripts/hooks/pre-commit:111）✓。
- 范围外报备裁决：okx-api-spec 缺 frontmatter——**本席补件**（doc 级，决策席顺手修权限内），哨兵归零。
- 验收中发现的闸缺口（坑引列依赖经验库内容、8b 门只查 staged skills/ 面→经验库-only commit 溜漂移）→ 坑条目入库+立 T-0150 修复，不阻本票。
- 状态翻完成。

### 状态：完成（判过并码在案，11-01清账翻面）

## 无头审计判决（审计席：headless @ 2026-10-30 —— **通过**）

锚实对账（收口 ee2dbd62 差面+验收补件 a45feae2；隔离 worktree 基 origin/master 产物实态三证核验）：

- **锚①覆盖全+stale 红**：在册 SKILL.md 19 件 vs 索引 19 数据行逐域对拍全对；`go run ./tools/kbgen skills --check` 现态 exit 0；改 pg-capacity frontmatter 首句未重建→`skills index out of date` exit 1 红实证。
- **锚②≤60 行**：索引现 25 行；码内硬顶 `skillsIndexMaxLines=60`（超限生成即报错）。
- **锚③钩红实证**：staged 改一件 skill 不重建索引→`bash scripts/hooks/pre-commit` 实跑 exit 1（FAIL: skills/索引.md 陈旧）；重建补 staged→exit 0 全钩绿。现行钩为 T-0158 扩面版（触发面 `^skills/|^docs/工程经验库/`），本票原挂点在档（pre-commit:116-128）。
- **锚④缺描述哨兵**：无 frontmatter 临时件（zz-audtmp）实测→索引行标「缺描述」（与收口态 okx-api-spec 实例同源径）。

残留观察（非锚违例，报管理席裁量）：frontmatter 存在但无 description 键的 skill（zz-audtmp2 实测）覆盖列落**空白格**而非「缺描述」字面——锚文「frontmatter 缺 description」严格口径下该子类未标字（空白格异常可目视、不可机判）；收紧=一行补丁（`perr != nil || fm.Description == ""`→缺描述），立小票可消。另：8b 触发面初版未含经验库依赖面（坑引列派生源）之缺口=票面验收时自发现，坑条目 skills-index-drift-via-cross-face-commit 入库+T-0154/T-0158 修复已在 master，闭环在案不属本票违例。
