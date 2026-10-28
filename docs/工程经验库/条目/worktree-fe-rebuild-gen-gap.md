---
schema_version: 2
id: worktree-fe-rebuild-gen-gap
kind: pitfall
risk: low
status: verified
keywords: [worktree, provision, web-src-gen, vite, npm-build, protobuf]
scope: [worktree, frontend, build, tooling]
summary: worktree 前端重建（npm run build）缺 web/src/gen——provision_worktree.sh 只补 internal/gen+dist；从部署树拷或 buf generate 补
first_seen: 2026-10-28
last_seen: 2026-10-28
evidence_refs:
  - type: provenance
    ref: T-0222 施工 session（zcode-glm-c71e）
---

## worktree 前端重建缺 web/src/gen——provision 盲区

- **实证**（T-0222 @ /opt/o-wt/t0222）：`provision_worktree.sh` 只拷 `internal/gen/`+`web/dist/`——过 go 闸够用；但 `cd web && npm run build` 报 `Could not resolve "../gen/dashboardv1/core_pb" from "src/react/api.ts"`（`web/src/gen/` 亦 gitignore 不入库，worktree 天缺）。
- **解法**：`cp -r /opt/o/web/src/gen <wt>/web/src/`（同 commit 基时）；或 web 面走 buf generate 慢径。
- **附带**：目检台架端口会撞他席长驻 vite（5199 被占实证）——台架端口用票号歧义后缀（如 5288），起前 `curl` 探活；vite/stub 起在后台任务防 shell 退出连杀。
