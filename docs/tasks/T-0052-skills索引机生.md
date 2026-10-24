# T-0052 skills 索引机生（路由速查，省 token 面）

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:决策]` @ 2026-10-26 |
| 领取 | （空=未领） |
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
