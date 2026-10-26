# T-0052 skills 索引机生（路由速查，省 token 面）

| 字段 | 值 |
|---|---|
| 状态 | 完成 |
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

### 状态：完成
