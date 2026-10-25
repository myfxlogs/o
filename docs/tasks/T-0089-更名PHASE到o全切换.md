# T-0089 PHASE→o 全切换

| 字段 | 值 | |---|---| | 状态 | 完成 | | 领取 | devin-t0089 @ 2026-10-25 |
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

## 结果回填

- 码面：go.mod `phase`→`o`、imports/go_package/proto pkg `o.dashboard.v1` 全换；`PHASE_*` env→`O_*`（allowlist+compose+code 三面对齐）；`[PHASE告警]`→`[O告警]`、页面标题、二进制/镜像/容器/卷名全 o-*；`phase_master_key`→`o_master_key`；`phasekb`→`okb`；~140 文件机械改。
- 门禁根修：`check_import_direction.sh` 模块名剥离 `sed s|o/||g` 误吃 `portfolio/` 中 `o/` 子串——改 `^mod/`+` -> mod/` 锚定。
- 部署链实测（本机+SG 双端）：`ALTER DATABASE phase→o`+`ALTER ROLE phase→o`（会话用户不可自改名——rn_admin 中转；口令 ALTER 重钉 o）；fin_employee_registry `phase-core`→`o-core` UPDATE 1 行×2 端；卷 `phase_phase-pgdata{,-test}` → `o-pgdata{,-test}`（cp -a 物理拷贝，旧卷留作冷备）；SG `phase-postgres:18-tsdb` 薄镜像 retag `o-postgres:`；`o-core` 镜像本机 build→SG load；compose `-p o` 双端 up——`o-core` healthy + `/api/overview` 200 双端实测。
- 目录：`/opt/phase`→`/opt/o`、`/opt/phase-wt`→`/opt/o-wt` + `worktree repair`（97 worktree 双向指针机械修复后 repair 全活）；SG `/opt/phase`→`/opt/phase-retired-20261025`（SG 非 git 树=rsync 面）；SG 侧旧 phase-* 容器/本机旧容器已 rm。
- repo：github `myfxlogs/phase`→`myfxlogs/o`（gh repo rename 实测，旧名重定向）；origin URL 已换 fetch 通。
- 偏离/留尾：①docs/tasks·decisions·经验库·skills·总纲·公司架构·migrations=档案/禁改面未动（票面注明的 archive 口径）；②README 命名源流正文按机械律全改 O（候选谱叙事变今名语境）+铭文 `ex phase-differentia`→`ex o-differentia`——铭文与名绑定应否另定新铭，留业主/决策裁；③`fin_holding.inherited_from` CHECK 值 'phase'=历史出处标记非身份名，未动；④/opt/o 部署树 master 检出待本票合并后 `git pull` 对齐（SG=rsync 面已是新码）；⑤旧卷 phase_phase-pgdata* 双端留冷备未删。
- 停机窗：双端 core 停机≈2–4min（卷拷贝段），秒级 DSN 切换。
- 测试证据：go build/vet/test 49 包 PASS；vite build✓；机检锚 grep go/yml/toml/Dockerfile=0；双端 `pg_isready -U o -d o`+API 200 亲证。
