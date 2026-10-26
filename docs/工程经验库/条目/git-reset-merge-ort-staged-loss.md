---
schema_version: 2
id: git-reset-merge-ort-staged-loss
kind: pitfall
risk: high
status: verified
keywords: [git reset, merge ort, 暂存连坐, merge 空壳, 工作树脏码, master 假基]
scope: [一切 merge 冲突解径, 部署树/工作树一致性]
summary: merge 冲突解跑裸 git reset（本意撤未跟踪件）→连坐撤回 ort 已暂存的全部合入码面→merge commit 只含冲突件成空壳——merge 在世系、代码不在树；部署链从工作树 build 故生产跑着未提交码，master 成假基，下游 merge 再撞「local changes would be overwritten」才暴露
first_seen: 2026-10-26
last_verified: 2026-10-26
evidence: [T-0160 merge 1a2157f 空壳实案（HEAD:open.go 零 dcd 码+生产跑脏码 12h）, 复原径 stash→worktree→77e553f ff 补录]
---

# git-reset-merge-ort-staged-loss——裸 reset 连坐 ort 暂存，merge 空壳吞码

## 症状
- merge commit 在 master 世系、但 `git show HEAD:<码件>` 零目标实现——
  merge=空壳，票验收锚在 diff 层「假过」。
- `git status` 恒见一批 ` M` 码件（工作树脏）——易被当「无关脏件」略过
  （本席如此漏检一轮）。
- 部署链若从工作树 build=生产跑着未提交码（master/生产双源背离）；
  后续基于 master 的施工/审计全看假基。
- 下游 merge 报「local changes would be overwritten」反复拒——暴露信号。

## 根因链
1. ort 合入时无冲突码面已**自动暂存**于 index。
2. 解冲突者跑裸 `git reset`（想撤一个不想提交的未跟踪件 `tools/`）——
   连坐撤回全部 ort 暂存件回工作树未暂存态。
3. `git add 冲突件 && commit` 产出的 merge commit 树=旧基+冲突件——
   码面实物全漏。
4. 部署链从工作树 build 正常出镜像——生产真 master 假，背离隐形。

## 处置
- 已发生：工作树脏件 `git stash -u`→worktree pop 提交→部署树
  `git merge --ff-only` 补录（部署树 D-651 禁直写码，补录也须 worktree 径；
  本实案复原 commit=77e553f）。
- merge 完提交前机械自检：`git diff --cached --stat` 应含全部预期码面；
  merge commit 落地后 `git status` 应净（` M`=信号非背景）。
- 预防：解冲突只想排除某件=只 `git reset -- <path>` 精准撤，
  **禁裸 `git reset`**。

## 证据
- check: `git show HEAD:cmd/core/open.go | grep -c dcdOpen`（T-0160 实案）
- result: 0——merge 1a2157f 在世系但 open.go 零 dcd 码；生产实跑脏码 12h+
- ref: T-0160/2068548 补录链；reflog HEAD@{8} merge→HEAD@{7} 间撤暂存实录

## 反证条件
- check: merge 落地后 `git diff --cached --stat` 与 `git status` 双净对拍
- falsifies_when: 裸 reset 后 merge commit 树仍含全部码面（git 版本行为异）

## 同族附注
- `git stash` 默认不带 `??` 未跟踪件——须 `stash -u`；pop 到他树后源树
  `??` 件即失（本实案 open_dcd_test.go 一度「消失」虚惊）。
- `git merge --ff-only` 报「Already up to date」时**不触工作树**——
  工作树停留旧版须 `git checkout -- .` 手动归位（本实案二次踩点）。
