# T-0009 Dockerfile web 阶段占位修正

| 字段 | 值 |
|---|---|
| 状态 | 施工中 |
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
