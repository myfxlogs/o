---
schema_version: 2
id: push-succeeds-after-hook-rejected-commit
kind: pitfall
risk: medium
status: verified
keywords: [git, push, pre-commit, 钩拒, 空支, 假锁, 领单, worktree]
scope: [git 工作流, 领单流程, scripts/hooks]
summary: pre-commit 钩拒掉 commit 后 git push 照常成功——分支引用在、翻单提交缺席，「推送成功=锁单」信号假真；新 worktree（gen 面未 regen→staticcheck 必拦）首 commit 被拒时必踩；推后必须核支上.tip commit 是自己的翻单提交才算锁成
first_seen: 2026-10-26
last_verified: 2026-10-26
skills: [协议面取证]
evidence_refs:
  - type: code_fact
    ref: git push 不跑 pre-commit 钩（仅 commit 跑）；collab.md §3 锁凭据=分支+分支上的翻单提交，单看 push exit 0 判不了锁成
  - type: doc
    ref: T-0158/T-0159 领单现场（2026-10-26 两度实踩：首 commit 被 staticcheck 闸拒（新 worktree 缺 internal/gen），push 照推空支，补 regen+重提后才真锁）
---

# pre-commit 拒 commit 后 push 照常成功——「推送成功=锁单」假信号

## 事实/症状

`git worktree add -b t/NNNN …` 后立即翻单 commit，pre-commit 钩拒
（新 worktree 无 `internal/gen` 生成码→staticcheck 闸报
「package o/internal/gen/mtapi/mt5 is not in std」）。此时随手
`git push -u origin t/NNNN`：**远端分支创建成功、exit 0**——但支上
零新提交（指向 origin/master 原样）。§3「分支 push 成功=锁单」的
判据被满足表象，锁凭据（支上的翻单提交）实际缺席。

## 根因

git 钩的作用域：pre-commit 只挂 commit；push 链上无任何闸核对
「支与 origin/master 有正距离」。两条独立机制（钩拒/推空）相加
=假锁态，且 push 成功的绿输出会主动掩盖 commit 失败的红。

## 处置

- 领单序固定为：**regen 先行**（新 worktree 首提之前
  `buf generate --template buf.gen.mtapi.yaml`）→ commit →
  确认 `git log -1` 是翻单提交 → push → `git log @{u} -1` 复核。
- 若已推空支：补 regen+重提+再推即可，空支被自己的真锁提交覆盖，
  无需删支（分支名占位反而先挡了双抢）。
- 通律：**push 的 exit 0 只证「引用已同步」，不证「提交已落支」**；
  凡「成功信号属于链上后位环节」的流程，前位环节的败因会被后位
  绿输出掩盖——锁凭据必须取自凭据本体（支上的提交），不取自链上
  任一环节的 exit code。

## 证据

- check: T-0158/T-0159 领单各实踩一次；补 regen 后 commit 过闸、
  push 后 `git log --oneline -1` 显翻单提交
- result: 两票均真锁（5ef32ad/fcbaf47）

## 反证条件

- check: pre-push 钩若立（核支上.tip 为本席翻单提交）则本坑闭合
- falsifies_when: 钩拒后 push 非零退出或拒绝建支

## 适用边界

适用于本仓 worktree+分支锁单工作流；对无钩仓库退化为纯纪律问题
（推后复核）。根修（pre-push 闸）属流程机器面，需决策立项，本条
只钉坑形与现场纪律。
