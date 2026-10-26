# T-0158 skills 索引闸触发面补依赖面（坑引列跨面派生漏检）

| 字段 | 值 |
|---|---|
| 状态 | 已批（原 T-0154，撞号让号重编——池守卫票已全链投产不动，未施票让号，T-0054 先例成本判版） |
| 立项 | `[角色:决策]` @ 2026-10-26 |
| 领取 | （空=未领） |
| 优先级 | 低 |
| 涉及面 | `scripts/hooks/pre-commit`（闸 8b 触发面一行） |

## 背景

T-0052 验收实案（2026-10-26）：skills/索引.md 坑引列取自
`docs/工程经验库/技能清单.md`，而 pre-commit 闸 8b 触发面仅
`^skills/`——经验库-only commit（坑条目+kbgen gen）不触闸，
坑引列漂移静默上 master（当日五枚新坑在案，--check 红实证）。
坑条目已入库：`skills-index-drift-via-cross-face-commit`。
通律：**派生依赖面 ⊆ 闸触发面才闭环**。

## 目标

闸 8b 触发面扩为 `^skills/|^docs/工程经验库/`——坑引列的输入面
（经验库技能清单）入 staged 集时同样跑 `kbgen skills --check`。

## 验收标准

- [ ] staged 含 `docs/工程经验库/` 件且索引陈旧→钩红（实证：暂存一枚
      新坑条目不重建 skills 索引→pre-commit exit 非零）
- [ ] staged 仅 `skills/` 面→原行为不变（回归不破）
- [ ] 重建后过闸 exit 0；`kbgen gen --check`/`skills --check` 双绿
- [ ] 坑条目 `skills-index-drift-via-cross-face-commit` evidence_refs
      补修复 commit 坐标

## 范围外

- kbgen 本体逻辑不动（生成/检查语义已对）
- gen/gen-econ 闸触发面不动（经验库闸已含自身面）

## 参考

- `scripts/hooks/pre-commit:111`（闸 8b 现触发面）
- `tools/kbgen/skills.go`（坑引列=技能清单.md 反向解析——依赖面出处）
- 坑条目：`docs/工程经验库/条目/skills-index-drift-via-cross-face-commit.md`
