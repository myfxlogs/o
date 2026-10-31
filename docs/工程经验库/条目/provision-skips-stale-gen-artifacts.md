---
schema_version: 2
id: provision-skips-stale-gen-artifacts
kind: pitfall
risk: medium
status: verified
keywords: [provision_worktree, gen陈旧, buf generate, protoc-gen-es缺失, go build红]
scope: [build, codegen, worktree, provisioning]
summary: provision_worktree「已在免补」只看目录存在不看新鲜度——陈 gen(proto新字段缺)致 build 红；buf generate 本机径需 web/node_modules
first_seen: 2026-10-31
last_seen: 2026-10-31
evidence_refs:
  - type: provenance
    ref: 清孤会话 fix-mt5-swap-outage/t0236 worktree check-all build 红实录
---
---

## provision_worktree 免补判据=存在性——陈 gen 静默致 build 红

- **实证**：worktree 内 gen 目录存在但陈旧（proto 已加 Headless/TotalTokens
  等字段）→ `provision_worktree.sh` 报「环境件已在免补」→ check_all
  go build 红（cockpit_face.go 字段 undefined）。buf generate 本机径
  双坑：默认模板含 web ES 插件需 `web/node_modules/.bin/protoc-gen-es`
  （未 npm ci 即 stat 败）+ 双 buf 并发/上下文挤占报 context canceled。
- **解法**：`cp -r /opt/o/internal/gen/* $WT/internal/gen/` +
  `web/src/gen` + `web/dist`（部署树恒有最新构建产物——provision 兜底
  源同径），或先 cd $WT && npm ci 再 buf generate 双模板。
- **通律**：gitignored 生成物「存在≠新鲜」——免补判据须含新鲜度锚
  （如 HEAD proto 文件 mtime vs gen mtime），provisioner 改进票待立。
