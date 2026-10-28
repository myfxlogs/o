---
schema_version: 2
id: worktree-provision-missing-web-gen
kind: pitfall
risk: medium
status: verified
keywords: [provision_worktree, web/src/gen, vite, 台架, 前端空白, core_pb, ERR_ABORTED, 生成件缺口]
scope: [worktree, web, vite, 前端台架, codegen]
summary: provision_worktree.sh 只拷 internal/gen+web/dist 两件，漏 web/src/gen（proto 生成 TS）——后端全链闸绿但 vite dev 前端空白（core_pb 解析 500），症状在前端根因在台架缺件
first_seen: 2026-10-28
last_seen: 2026-10-28
evidence_refs:
  - type: ticket
    ref: T-0224
  - type: doc
    ref: scripts/provision_worktree.sh:15
  - type: doc
    ref: .gitignore:8
---

## worktree 台架缺 web/src/gen——前端空白误诊坑

- **实证**：T-0224 目检台架（vite dev+stub）在新鲜 worktree 起服务，
  页面 serve 正常、stub/proxy 全通，但 SPA 渲染空壳（body 134 字符、
  零 chip 零 dot）；浏览器 console 两组信号：`Failed to load resource:
  500` + `src/react/stream.ts / api.ts net::ERR_ABORTED`；vite 日志真因
  `Failed to resolve import "../gen/dashboardv1/core_pb"`。
- **病灶**：gitignored 生成件共三处——`internal/gen`（Go 面）、`web/dist`
  （构建件）、`web/src/gen`（proto→TS 面）。provision_worktree.sh 的
  need 断言与拷贝清单只覆盖前两处（L15/L23-26）；Go 闸（build/vet/
  staticcheck/check_all）全绿给出「环境齐」的假信，前端缺件直到
  vite 转译才爆，且症状（页面空白）与根因（台架缺件）隔着 proxy/
  stub/React 三层。
- **律**：worktree 起前端台架前先验 `ls web/src/gen/dashboardv1/
  core_pb.ts`；缺则 `cp -r /opt/o/web/src/gen web/src/`（部署树恒有面）。
  诊断次序：vite 日志 grep "Failed to resolve" 先于一切前端码排查。
  provision 清单补该件属单外改动，须走票（本条目即事实归位）。
  **provision 时点=紧随 worktree add、先于领单首 commit**——pre-commit 闸
  同跑 staticcheck 全树编译（T-0228 实证：缺 `internal/gen/mtapi/mt5` 时
  首个领单 commit 即被拦，非推支时才爆）。

> 相关条目：[`worktree-fe-rebuild-gen-gap`](worktree-fe-rebuild-gen-gap.md)——同根因的首次实证（T-0222 前端重建踩坑）；两条目待归纳合并，根定位以本条为准。

> **根修在案**：T-0229（2026-10-28，业主直令抢修）——`provision_worktree.sh` 三处补 `web/src/gen`（need 检/快径拷/慢径 buf+终检），dist 快照语义注明（仅供闸/dev，生产唯一径=deploy_image.sh）。第四次显形=他席部署镜像混装（当前后端+陈旧前端），部署链病灶实证补记。
