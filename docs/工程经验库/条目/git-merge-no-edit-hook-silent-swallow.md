---
id: git-merge-no-edit-hook-silent-swallow
title: git merge --no-edit 钩红静默吞合并提交——管道吃 rc 呈"无输出成功"假象
tags: [git, merge, pre-commit-hook, silent-fail, 手工合并]
last_verified: 2026-11-20
---

# git merge --no-edit 钩红静默吞合并提交——管道吃 rc 呈"无输出成功"假象

## 现象

手工合并径 `git merge --no-edit -m "..." <branch> 2>&1 | tail -N`（或
`git merge --no-edit -q`）：pre-commit 钩子红（D-585 缺 `坑处置:` 行、
gen 件未生成、gofmt 漂移等）→ 合并**停在 staged 未提交态**，而管道
下游只看 tail 输出——现象=「无报错也无 Merge branch 输出，rc 看似
正常」，易误判合并已落成。同型撞过两次：一次停在 `MERGE_HEAD` 悬挂，
一次零输出直接错过提交。

## 根因

`--no-edit` 的自动提交同样过 pre-commit 钩——钩红则 commit 不生；
`2>&1 | tail` 管道把钩的 FAIL 输出截断吞掉，pipeline rc 由 tail 决定
（恒 0），真实失败被管道语义掩埋。

## 处置

手工合并序：`worktree → bootstrap_worktree.sh（先生成 gen 面防 gofmt/
staticcheck 钩红）→ merge → git status 确认 staged → 显式 git commit -m
（钩子输出全量可见）→ git log -1 --format='%h parents=%P' 核双亲`。
禁在 merge 命令上挂 `| tail`——要么不接管子，要么管道后必核
`git log -1 --format='%P'` 双亲数=2 再续跑。另：合并 commit 信息必带
`坑处置:`（D-585）与免疫件 `免疫件:` trailer——两闸文案即修法。

## 回改点

- [ ] 手工合并跑批判完成依据=双亲=2 非「无红字」
- [ ] merge/commit 命令不搭 tail 管道（要截输出先落盘再 tail 文件）
