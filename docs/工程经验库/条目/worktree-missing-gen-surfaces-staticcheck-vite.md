---
schema_version: 2
id: worktree-missing-gen-surfaces-staticcheck-vite
kind: pitfall
risk: medium
status: verified
keywords: [worktree, buf generate, internal/gen, web/src/gen, staticcheck, vite, gitignored, 生成面, pre-commit]
scope: [worktree, pre-commit, go build, web, codegen]
summary: 新 worktree 缺 gitignored 双 gen 面（internal/gen+web/src/gen）→ pre-commit staticcheck 报「package o/internal/gen/... is not in std (compile)」、npm run build 报「Could not resolve ./gen/dashboardv1/core_pb」；处置=buf generate（正典）或部署树 cp（master 对齐时等价）
first_seen: 2026-10-26
last_verified: 2026-10-28
skills: []
evidence_refs:
  - type: provenance
    ref: T-0155（领单+施工双撞——pre-commit 闸与 vite build 各炸一次，cp 后双绿）
  - type: doc
    ref: docs/工程纪律.md:69（前端构建前置 buf generate——gen 产物 gitignored 不入库）
---

## worktree 缺 gitignored 生成面——staticcheck/vite 双闸假红

- **实证**：新建 worktree 后首次 commit，pre-commit 闸报 `internal/mt5/bridge.go:23:2: package o/internal/gen/mtapi/mt5 is not in std (compile)`（staticcheck 因 gen 包缺席连编译都过不了）；补 Go 侧后 `npm run build` 再炸 `Could not resolve "./gen/dashboardv1/core_pb" from "src/main.ts"`。两炸均与改动面无关（纯领单 commit、HTML 文本改动），是 worktree 环境缺件假红。
- **根因**：proto 生成面 `internal/gen`（Go）与 `web/src/gen`（TS）均 gitignored（.gitignore:8-9），不经 git 检出传播——新 worktree 天然缺。与 [[worktree-web-dist-embed-gate]]（web/dist 侧，go:embed 门）同族：worktree 生成物断链族，共三面。
- **规则**：worktree 建好后、首次 commit/build 前，先补齐生成面——正典 `buf generate`（proto 唯一源，产物与 master 对齐）；部署树在同 master 提交时 `cp -r /opt/o/internal/gen <wt>/internal/` + `cp -r /opt/o/web/src/gen <wt>/web/src/` 等价。改动面不含 proto 时产物内容无关紧要，实存即可。

## 证据

- check: 新 worktree 直接 commit → pre-commit staticcheck 炸（见 summary 报错原文）；`npm run build` → vite resolve 炸
- result: cp 双 gen 面后 pre-commit 绿（`OK: pre-commit gate passed`）、`npm run build` 绿（exit 0，dist 五件产物）
- ref: T-0155 施工分支 t/0155-瓦片坐标字母化（0daf968 领单提交过闸即证）

## 反证条件

- check: 若未来 gen 面改为入库（.gitignore 移除）或 pre-commit 改为容缺生成
- falsifies_when: 新 worktree 不补生成面也能过 staticcheck 与 vite build

## 适用边界

- 仅新 worktree 首次构建/提交前适用；部署树 gen 面常在（部署链自产）
- cp 法前提=部署树与 worktree 同源 master（proto 未漂）；漂了就走 buf generate
- T-0218 实证补：`scripts/provision_worktree.sh` 只补 internal/gen+web/dist
  两槽——web/src/gen 不在其补面，前端施工票须另行 cp 或 buf generate
  （脚本扩槽属另单面）
