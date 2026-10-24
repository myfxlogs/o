---
schema_version: 2
id: gate-script-pipefail-empty-grep
kind: pitfall
risk: medium
status: verified
keywords: [pipefail, grep空集, pre-commit, set -euo pipefail, 静默失败]
scope: [门禁脚本, shell 管道, 一切 grep|xx 组合]
summary: 闸脚本内 grep 空集退出 1→pipefail 杀脚本——钩子静默失败提交被拒且零报错行；过滤用 true 兜底或 rg||true 径
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: []
---
# gate-script-pipefail-empty-grep

## 坑
pre-commit 闸脚本内 `git diff --cached --name-only | grep '\.go$'` 在无 Go 暂存时 grep 退出 1 → `set -euo pipefail` 杀脚本 → 钩子静默失败，提交被拒且无报错行。

## 实证
D-0008 check_no_forbidden_patterns.sh 首版（2026-10-24）：纯 docs 提交被拦，钩子只出前置 WARN 无 BLOCK 行，排查才发现是闸自身空集管道自杀。

## 反证条件
闸脚本内所有「过滤型 grep」管道必须对空集容错（`{ grep ... || true; }` 或 `|| true` 尾），且闸脚本自身改完必须跑一次自测提交验证不误拦。

## 机械化
check_no_forbidden_patterns.sh 已修；本条目作为闸脚本编写时的 checklist 项。
