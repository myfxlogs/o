# T-0089 PHASE→o 全切换

| 字段 | 值 | |---|---| | 状态 | 施工中 | | 领取 | devin-t0089 @ 2026-10-25 |
> 业主令：项目更名 PHASE→o（域名 o.myfxlogs.org 对齐）。一票原子切，禁半改态。

## 三层命名律

| 层 | 值 |
|---|---|
| 代码/基建（go module、二进制、镜像、compose、容器、DB、目录、员工行） | 小写 `o` |
| 面客显示（页面标题、`[O告警]` 前缀、文档抬头） | 大写 `O` |

## 切换清单（顺序钉死）

1. **Go module**：`go.mod` `phase`→`o` + 全库 `import "phase/..."` 机械改
2. **运行面**：二进制名、Dockerfile 镜像名 `phase-core`→`o-core`、compose 服务/容器名、`deploy_image.sh`、`mtapi` 镜像相关引用
3. **DB**：`ALTER DATABASE phase RENAME TO o` + 全容器 DSN 同改同重启（停机窗秒级）
4. **员工行**：`fin_employee_registry` `phase-core`→`o-core`（UPDATE）
5. **面客**：`[PHASE告警]`→`[O告警]`、页面标题、文档抬头
6. **目录**：`/opt/phase`→`/opt/o`、`/opt/phase-wt`→`/opt/o-wt` + `git worktree repair` 全量
7. **文档**：AGENTS.md/collab.md/路线图/任务单引用全改（`grep -rn "phase\|PHASE"` 扫尾）
8. **repo**：github `phase`→`o`（最后一步，remote 一换）

## 机检锚

- `grep -rn "PHASE\|phase" --include="*.go" --include="*.yml" --include="*.toml" --include="Dockerfile"` = 0（除 archive/历史票文档注明不改）
- `docker images | grep o-core`、容器 `o-core`/`o-postgres` Running、`psql -l | grep ' o '`
- `/api/overview` 200、员工行 `o-core` 在册、告警前缀 `[O告警]` 下一拍真值
- `git worktree list` 全活（repair 后无失联）
