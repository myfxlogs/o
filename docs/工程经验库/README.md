# 工程经验库

> **域=工程面坑**：码/设施/工具链/venue API 行为的故障与陷阱。经济面案例归 `docs/经济经验库/`。
> 建制自 nox `docs/经验库/`（schema v2，门禁=索引机生禁手编）。

## 结构

| 位置 | 内容 |
|---|---|
| `条目/` | canon 条目——schema v2 格式见 `模板.md`（复制改字段） |
| `归档/` | 退役条目（`status: retired`，不进索引仍可 grep） |
| `待归纳.md` | staging——中低危坑先停此，高危直入条目+同 commit 索引（D-618 分级） |
| `索引.md` | **机生索引，禁手编**——生成工具待迁（nox 原形 `tools/noxkb`） |
| `前身-nox/` | nox 274 条库条目原文（@b43c02d6），参考料非 canon——升格=复制进 `条目/` 按现 schema 校对（T-0003 分拣） |

## 入库与消化

- **时点=事件驱动**：坑现即写（私忆清扫纪律——session 死不得带走坑）；修复类 commit 未触本库→commit-msg warn 提醒（D-585）。
- **高危/生产事故/系统性坑**：直写 `条目/` + 索引再生成，同 commit；staging 行不进检索=半隐形，禁积压。
- **中低危**：可停 `待归纳.md`，复审/决策席周期消化（周检）。
- **条目退役判据**：被取代 / 针对物已移除 / 反证条件命中——`git mv` 到 `归档/` + `status: retired` + 索引同 commit。

## 条目契约（schema v2 要点）

- `id` kebab-case 全库唯一；`kind`=pitfall/strategy/invariant/ops；`risk` 四级
- `evidence_refs` 五类：decision（D#）/commit/doc（file:line）/review_signature/server_fact（只读 query，禁 DSN 凭据）
- 正文六节：事实/症状 · 根因 · 处置 · 证据 · **反证条件**（必填——什么结果让结论失效）· 适用边界
- 动态事实条目 `last_verified` 须滚动更新
