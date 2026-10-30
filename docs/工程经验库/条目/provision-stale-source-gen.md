---
schema_version: 2
id: provision-stale-source-gen
kind: pitfall
risk: medium
status: verified
keywords: [provision_worktree, internal/gen, GetHeadless 未定义, buf generate, 陈旧快照, detached HEAD]
scope: [worktree, scripts/provision_worktree.sh, internal/gen, web/src/gen]
summary: provision 从部署树拷 gen/dist 快照——部署树 detached@origin/master 的 gen 不随 fetch 重生，可陈旧于 master proto→新树 build 即炸 GetHeadlessRequest undefined；修=树内 buf generate 重产（web es 插件缺时用 Go-only 模板）
first_seen: 2026-10-30
last_verified: 2026-10-30
skills: []
evidence_refs:
  - type: commit
    ref: dm/1030-t0259-merge（check_all go build 红 GetHeadlessRequest undefined→buf 重生后全链绿）
  - type: doc
    ref: ~/devin-headless-T-0265.log（施工席同坑：拷贝的 gen 陈旧缺 T-0262/0263 新面——临时 Go-only 模板重产）
  - type: doc
    ref: scripts/provision_worktree.sh:15
---

## 事实/症状

新鲜 worktree 跑 provision_worktree.sh 后 `go build` 即红：
`undefined: dashboardv1.GetHeadlessRequest/HeadlessRow`（cmd/core
headless_face.go、cockpit_face.go、tools/hlprobe 同族多处）——而 master
proto/码面明明定义齐全。

## 根因

gitignored 生成件三件（internal/gen、web/dist、web/src/gen）不随
`git fetch/checkout` 重生。部署树 /opt/o 常驻 detached@origin/master，
master 前移（PR#100 引入 GetHeadless RPC）后其 gen 快照仍停在旧 proto
时点；provision 逐字拷贝旧快照 → 新树拿到「比 master 旧」的 gen。
即：provision 治「缺」，不治「陈」。

## 处置

- 树内重生：`buf generate --template buf.gen.core.yaml`（Go 面；
  web es 插件缺=node_modules 未装时，Go-only 模板即够 check_all
  全链——check_all 不跑 web build；需 TS gen 再装依赖走 buf.gen.web.yaml）。
- 远期根修（若复发）：provision 增「快照新旧探针」（proto mtime vs
  core.pb.go mtime 对拍，陈则自动 buf 重生）。

## 证据

- check: `grep -c GetHeadlessRequest internal/gen/dashboardv1/core.pb.go`
  ——拷后=0，`buf generate --template buf.gen.core.yaml` 后=12。
- 同日两席中：DM 并码支（dm/1030-t0259-merge）+无头施工席 T-0265
  （log 自述「拷贝的 gen 陈旧缺 T-0262/0263 新面」）。
