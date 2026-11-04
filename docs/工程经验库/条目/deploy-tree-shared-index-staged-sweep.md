---
schema_version: 2
id: deploy-tree-shared-index-staged-sweep
kind: pitfall
risk: medium
status: verified
keywords: [/opt/o 部署树, shared index, git commit, staged sweep, 并发席, 归属错置]
scope: [/opt/o 部署树, 多席并发, git index]
summary: 多席共用 /opt/o 部署树=共用同一 .git index——A 席 stage 的改动会被 B 席随后的 `git commit`（不带 pathspec）整锅扫入提交，归属错置且审读面污染；实证 T-0316 裁决件扫入并发域界清除 24 删除
first_seen: 2026-11-04
last_verified: 2026-11-04
skills: []
evidence_refs:
  - type: commit
    ref: 42824a9a（T-0316 裁决注记件混入 24 件他席 stage 删除）
  - type: commit
    ref: 01203ff1（域界大清除真属件——同组删除的本家提交）
repro_or_symptom: /opt/o 上 `git add <自己件>` + `git commit` 提交后 `git show --stat` 出现陌生文件改动——并发席在 commit 间隙 `git add/git rm` 入同一 index 所致
root_cause: git commit 默认提交整个 index 非仅本进程 add 的面；/opt/o 为全体席位共用工作树（§9 钦定决策台面），index 竞态无互斥——检出纪律管分支不管道具暂存区
fix_and_prevention: 部署树提交一律 `git commit -o -- <pathspec>`（only 模式只提指定径）或 `git commit <files>`；提交前 `git status` 核 index 含他席件→先 `git stash push --staged` 隔离或等其提交；混扫已发生不 rewrite——并发席在途改史更险，handoff 记档即足
notes: 与 [[precommit-pipeline-race-content-loss]]、[[deploy-tree-uncommitted-wip-stash-swept]] 同族（部署树并发竞态族）——本条=index staged 面被扫，非 WIP 未暂存被卷
---

## 现象

T-0316 裁决件 `git commit` 后 `git show` 出现 24 件 docs/tasks 删除——非本席所为；并发的域界大清除席 stage 在途删除被整锅扫入，其本家提交 01203ff1 随后落顶。

## 根因或策略解释

`git commit` 不区分 index 内容谁 stage 的——整个暂存区一锅出；/opt/o 单 index 多席共享，§9 只约分支/编辑不约 index 时序。

## 处置

- 预防：部署树提交用 `git commit -o -- <path>` 只提自件；或先 `git status` 见陌生 staged 件→`git stash push --staged` 推开再提。
- 已发生：不重写（并发席在途，改史风险>归属瑕疵）；handoff/坑条目记档，真属 commit 在顶即明。
