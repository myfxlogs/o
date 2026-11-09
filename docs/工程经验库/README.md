# 工程经验库

> **域=工程面坑**：码/设施/工具链/venue API 行为的故障与陷阱。经济面案例归 `docs/经济经验库/`。
> 建制自 nox `docs/经验库/`（schema v2）。**纪元3 现行法**：业主令 2026-10-25
> 「有坑即写条目+kbgen 重建」——不待归纳，排队=下个 session 盲区。

## 结构

| 位置 | 内容 |
|---|---|
| `条目/` | canon 条目——schema v2 格式见 `模板.md`（复制改字段） |
| `归档/` | 退役条目（`status: retired`，不进索引仍可 grep） |
| `前身-nox/` | nox 前身 274 条原文——**归档参考区**（README 在案：非 canon 不进索引，canon 引用 138 处故留档原位） |
| `索引.md` / `检索物料.tsv` / `技能清单.md` | **机生面，禁手编**——`go run ./tools/kbgen gen` 统一重建（检索物料 gitignored 再生物） |

## 分层与归档规约

- **分层**：`kind`=pitfall 主面（369/390）；`risk` 四级 critical>high>medium>low；
  `status` 在册 observed/verified/disputed/retired（历史遗留 mitigated/fixed/draft/active/canon 值随用随归一）。
- **退役判据**（README 旧约沿用，满足其一即退，存疑不退）：
  被取代 / **针对物已移除**（纪元3 裁撤面首犯——scripts/meta、票机、
  席位拍、metabus 等死件的条目批量归档 34 件在案）/ 反证条件命中。
- **归档操作**：`git mv 条目/<id>.md 归档/` + frontmatter `status: retired`
  → `go run ./tools/kbgen gen` 重建 → 同 commit。
- **检索**：归档条目不进索引；`grep -r docs/工程经验库/` 仍可命中全文。

## 新条目流程（现行）

1. 坑现即写 `条目/<id>.md`（私忆清扫纪律——session 死不得带走坑）；
   修复类 commit 未触本库→commit-msg warn 提醒（D-585 闸在位）。
2. `risk≥high` 或生产事故/系统性坑直入条目；中低危**同样直入**
   （status=observed 起步）——staging 层已废，待归纳.md 不再存。
3. `go run ./tools/kbgen gen` 索引+检索物料+技能清单同 commit 重建；
   `--check` 在 CI/pre-commit 面验鲜（物料缺席=合法，陈旧=红）。
4. 唤知会话教训矿径（scripts/cognition/runbook.md③）：fin_eng_lesson
   pending 逐条判→canon 落条目+`status='canon'` 回写，弃→`status='verdicted'`。

## 条目契约（schema v2 要点）

- `id` kebab-case 全库唯一；`kind`=pitfall/strategy/invariant/ops；`risk` 四级
- `evidence_refs` 五类：decision（D#）/commit/doc（file:line）/review_signature/server_fact（只读 query，禁 DSN 凭据）
- 正文六节：事实/症状 · 根因 · 处置 · 证据 · **反证条件**（必填——什么结果让结论失效）· 适用边界
- 动态事实条目 `last_verified` 须滚动更新；伴生量表节（T-0332 格式）照模板带
