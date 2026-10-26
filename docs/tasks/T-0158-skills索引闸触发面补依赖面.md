# T-0158 skills 索引闸触发面补依赖面（坑引列跨面派生漏检）

| 字段 | 值 |
|---|---|
| 状态 | 待验 |
| 立项 | `[角色:决策]` @ 2026-10-26 |
| 领取 | zcode-glm @ 2026-10-26 |
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

## 结果回填（施工：zcode-glm @ 2026-10-26）

实现摘要：`scripts/hooks/pre-commit` 闸 8b 触发面 grep 扩为
`'^skills/|^docs/工程经验库/'`（原仅 `^skills/`），门禁注释与 FAIL
文案同步扩接自述；kbgen 本体零触碰（范围外守约）。commit 链：
领单 5ef32ad → 闸修 e7dffc2 → 坑条目 evidence_refs 补坐标 19bb02f。

验收锚对账：

- 验收①（经验库件 staged+索引陈旧→钩红）：PASS——造临时坑条目
  `t0158-gate-test-temp` + `kbgen gen` 重建经验库索引后 staged 三件
  （条目+索引.md+技能清单.md），skills 索引不重建，直跑
  `bash scripts/hooks/pre-commit` → 闸 8b 红
  「skills index out of date」exit 1（修复前此集静默过闸）。
- 验收②（仅 skills/ 面原行为不变）：PASS 双向——②a 纯
  `skills/索引.md` staged+手漂 → 8b 红 exit 1（原红路径在）；
  ②b 纯 skills/ 面 staged（SKILL.md 描述改+重建索引）→ 8b 绿
  exit 0（原绿路径在）。
- 验收③（重建后过闸双绿）：PASS——同①staged 集 `kbgen skills`
  重建后钩子 exit 0；独立复跑 `kbgen gen --check` exit 0、
  `kbgen skills --check` exit 0。另 19bb02f 为扩接后 8b 首次真实
  经验库件提交，钩内双检真跑过。
- 验收④（坑条目补修复坐标）：PASS——evidence_refs 新增
  `type: commit ref: e7dffc2`，code_fact 同步改「修复前触发面」
  过去时态；见 19bb02f。

机检：pre-commit 全闸绿（env-reads✓ migrations✓ rejects_keys✓
gen --check✓ skills --check✓）；本票无 Go 面改动，gofmt/vet/
staticcheck 不涉。测试造数（临时条目+②b SKILL.md 描述改动）测毕
已全部还原，工作树净。

偏离说明：无超单。范围外双条（kbgen 本体、gen/gen-econ 触发面）
零触碰；闸注释/FAIL 文案属闸 8b 自述面随行修订。工作面备注：本
worktree 依 §11 需 buf regen mtapi gen 面（新树无生成码，staticcheck
闸所需，未入 commit）。

## 验收毕（决策席第 2 席 @ 2026-10-26）

- 锚亲证（worktree 合规造数实测）：经验库-only commit+索引陈旧→钩红报
  「skills/索引.md 陈旧」exit 1✓；重建→exit 0✓；触发面 grep 码面
  `^skills/|^docs/工程经验库/`✓；KB 条目 evidence_refs 补修复坐标
  （19bb02f）✓。
- 注记：首测造数不合规被 gen --check 先拦（schema 闸正当履职）——
  两闸序位无冲突，8b 精测以 schema 合规造数复跑为准。
- 状态翻完成。
