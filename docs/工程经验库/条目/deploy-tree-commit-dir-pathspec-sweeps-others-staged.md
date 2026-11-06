---
schema_version: 2
id: deploy-tree-commit-dir-pathspec-sweeps-others-staged
kind: pitfall
risk: medium
status: verified
keywords: [部署树, /opt/o, git commit, pathspec, staged, 多席并发, 目录形参]
scope: [工程纪律, git 操作, 多席并发]
summary: 部署树多席并发期 commit 用目录 pathspec 会把他席同目录 staged 改动卷入本席提交——精确到文件
first_seen: 2026-11-06
last_seen: 2026-11-06
evidence_refs:
  - type: provenance
    ref: T-0378 归done 提交 86bb67d7（卷入 COS staged T-0379 票面改）
---

## 部署树 commit 目录 pathspec 卷他席 staged——精确到文件不精确到目录

- **实证**：T-0378 归done 提交时 COS 席同树并发中（其 T-0379 票面改
  已 staged）。本席 `git commit -m ... -- docs/tasks/` 意只收归档改名，
  结果目录 pathspec 把 COS staged 的 `docs/tasks/T-0379-*.md` 修改一并
  卷入 86bb67d7——归 done 提交里夹带他席票面语义改（外派→常径待派+
  CEX 附注）。事后修正被 reference-transaction 非 ff 闸拒（COS 已在
  其上续 commit，重写成本高于保留），内容零失但归属错位在案。
- **根因**：`git commit -- <pathspec>` 提交「匹配路径的全部 staged+worktree
  改动」——多席共用 /opt/o 单 index，目录形 pathspec 视野含他席 staged。
- **教训**：部署树并发期提交三律——① pathspec 精确到**文件**不给目录
  （rename 两侧路径都列）；② commit 前 `git status` 看一眼他席 staged
  在不在自己 pathspec 内；③ 归档改名用 `git commit -- 旧径 新径` 双
  pathspec（`docs/tasks/` 目录写法=踩坑形）。master 非 ff 闸在，卷错
  后想回头重写=被拒——一次写对比事后修便宜。
- **收口**：本班处置=保留合史（COS 续 commit 接纳零冲突）+条目入册；
  后续同类 commit 一律文件级 pathspec。
