# T-0089 PHASE→o 全切换

| 字段 | 值 | |---|---| | 状态 | 审计通过 | | 领取 | devin-t0089 @ 2026-10-25 |
审计席：headless @ 2026-10-31
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

## 审计注记（headless 逐锚对账 @ 2026-10-31，收口链 71ce0b25→9a9d0042）

- 锚①grep=0：收口 commit 9a9d0042 上 `git grep "phase\|PHASE" -- '*.go' '*.yml' '*.toml' 'Dockerfile'`=0 命中 ✓。HEAD 现存 3 处 `phase` 均为后票新增：`tools/hlprobe/main.go`+`internal/gen/dashboardv1/core.pb.go`（T-0263 aee519bc 引入的席「相位」域词，非项目名残留）+未跟踪异树 `kb-tailscale/`（git status `??`，不在仓档）。非本票责任面。
- 锚②镜像/容器/DB：`docker images` 实见 `o-core:latest`+9 SHA tag；`o-postgres` Running healthy、`o-postgres-test{,-b..e}` 在跑；`o-core` 容器在册（`o-core:latest` 镜像、env `DB_DSN=postgres://o:o@o-postgres:5432/o`），10-29 实跑日志含新名进程业务行——切换运行态实证。库列表现存仅 `o`、无 `phase` 库（`docker exec o-postgres \l` 实查）。审计时点 `o-core` 呈 Exited(0)（26h 前净退）——本机运行面后票时漂，非更名未生效。
- 锚③员工行/面客：`fin_employee_registry` 实查 `o-core` active 在册、`%phase%` 行=0；`[O告警]` 在码 `internal/platform/notify/alert.go:87`、页面 `<title>O — 经营面</title>`。`/api/overview` 200 审计时点无现状可复跑（o-core 停、禁起停取证径）——收口回填双端 200 与容器新名运行残留互证。
- 锚④worktree：本机 `git worktree list` 275 条在册；3 条 prunable（`/tmp/o-master-dd5f05b`、`/tmp/t0216-deploy`、`/tmp/t87master`）stat 创建时刻均 2026-10-31 00:39——后于本票收口 6 天，清扫器具新痕，非 repair 失联。
- 结构核对：`go.mod`=`module o`；proto `go_package="o/internal/gen/dashboardv1"`；compose 容器名/卷名 o-*、卷显式 `name: o-pgdata{,-test}`；`deploy_image.sh` o-core tag+SG `/opt/o` 径；origin=`git@github.com:myfxlogs/o.git`；import 方向闸 `sed` 锚定根修为正当修法。
- 注记：回填留尾①~⑤披露属实（档案面未动合票面 archive 口径；README 铭文留裁明示）。
