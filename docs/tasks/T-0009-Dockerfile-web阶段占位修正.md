# T-0009 Dockerfile web 阶段占位修正

| 字段 | 值 |
|---|---|
| 状态 | 待验 |
| 提出 | `[角色:决策]` 2026-10-23 |
| 类别 | 骨架补漏（P1 尾） |
| 领取 | claude-glm @ 2026-10-23 |

## 背景

`docker compose up -d --build phase-core` 开发机自验失败：Dockerfile web-build 阶段 `COPY --from=web-build /src/web/dist` 在 `web/dist` 不存在时炸（前端占位 build 只 echo 不产物——P4 才真做前端）。

## 验收标准

- [ ] web-build 阶段在无前端时产占位 `dist/`（如 `mkdir -p dist && echo 占位 > dist/index.html`）或 COPY 容忍缺件——`//go:embed web/dist` 语义不破
- [ ] `docker compose build phase-core` 本地过；`compose up -d` 三服务全健康（含 phase-core）
- [ ] CI 不红

## 范围外

- 前端本体（P4）；SG 部署

## 结果回填（`[角色:施工]` claude-glm 2026-10-23）

### 实现摘要

- `Dockerfile.core` web-build 阶段：`npm run build` 之后补占位步——`dist/index.html` 缺件才写占位页，真前端落位后零触碰；目录有文件即满足未来 `go:embed web/dist` 语义。
- 拦路面第二断点（施工中实测发现）：`internal/gen/` 亦 gitignore 不入库，build 阶段 `go build` 找不到生成码。补 `buf.gen.core.yaml`（Go-only 模板，仿 buf.gen.web.yaml 隔离先例，out/opt 与 buf.gen.yaml 同号）+ build 阶段 `go install buf@1.74.0`/`protoc-gen-go@v1.36.12`/`protoc-gen-connect-go@v2.0.1`（与 CI/go.mod 同号）+ `buf generate`，与 CI「generate 先于 go build」同序。

### 验收锚对账

- 占位 dist / embed 语义不破：**PASS**（锚=Dockerfile.core:16-18 占位步；修复前 `COPY --from` 报 `"/src/web/dist": not found`，修复后该层过）
- `compose build` 过 + 三服务全健康：**部分未达（缺口:phase-postgres-test）**
  - `compose build phase-core`：PASS（镜像 `phase-core:latest` 构建成功）
  - phase-core：PASS（`Up (healthy)`；`curl 127.0.0.1:50051/`→ok，`/md/status`→活流 frames=14907、rejects 全零，行情巡在跑）
  - phase-postgres：PASS（healthy）
  - phase-postgres-test：**未达**——127.0.0.1:5434 被 arb 域在役容器 nox-postgres-test（/opt/arb-wt/d165-testenv-debt，Up 7 天）占用，容器停在三万 Created。业主已裁「不动 arb」——端口仲裁请决策席处置（改 compose 端口或与 arb 侧协调，均超本单面）
- CI 不红：**PASS**（CI 不建镜像、不读 buf.gen.core.yaml，本单未触 CI 路径；go build/vet/gofmt 本地全绿）

### 机检

build✓（go build ./... + 镜像构建）vet✓ gofmt✓（零漂移）import-direction✓（9/10 边）env-reads✓（白名单 18 键）check-lines✓（exit 0）test：本单零 Go 行为改动，全量 go test 归 CI（DSN 5434 面被 arb 占用）

### 偏离与交接说明（请决策核）

1. **归属缠结**：本单第一处修复（web-build 占位步）施工中被并发 session 以 `de8a95f`（[角色:决策][T-0011] 立单）一并扫入并已推上 master——内容正确、commit 标签错挂 T-0011。本分支 de8a95f 与 master 同物；验收合并时建议知悉此痕。
2. **孤儿容器清理两笔**（均已获业主批准执行）：t0007 遗留 phase-postgres/phase-postgres-test 容器已 rm（卷未删）；arb 侧容器**未触碰**（业主裁）。
3. 并发期教训已由决策吸收为协议 §9（工作面隔离），本单施工中段起全程在 worktree /opt/phase-wt/T-0009 内。
