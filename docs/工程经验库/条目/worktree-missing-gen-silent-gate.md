---
id: worktree-missing-gen-silent-gate
title: 新 worktree 缺 internal/gen 生成件——go list 哑死致 import 闸静默退 1（零输出）
tags: [worktree, go-list, import-gate, 生成件, gitignore]
last_verified: 2026-11-01
---

# 新 worktree 缺 internal/gen 生成件——go list 哑死致 import 闸静默退 1

## 现象

`git worktree add` 新支树里跑 `check_import_direction.sh`（或含 `go list`
的前置检查）→ **零 stdout 零 stderr 退出码 1**，像闸误报，实为前置命令哑死。

## 根因

`internal/gen/`（protoc 生成面：mtapi/dashboardv1 等）在 `.gitignore` 内——
主树有生成产物，新 worktree checkout 天然缺席 → `go list ./internal/...`
对引用 gen 的包报 "package ... is not in std" 非零退 → 脚本 `set -euo
pipefail` 下命令替换 `RAW=$(...)` 失败即死，verdict 段未到=零输出。

## 处置

新 worktree 首件事：`cp -r /opt/o/internal/gen <wt>/internal/`（或重跑
buf generate 产径）——go list 复绿后各闸正常。诊断诀：脚本静默退 1 先
`bash -x` 看死在哪条命令替换，别先疑闸逻辑。

## 回改点

- [ ] 新 worktree 建后跑 check_* 脚本前确认 internal/gen 在场
