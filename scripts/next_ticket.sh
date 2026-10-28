#!/usr/bin/env bash
# next_ticket.sh —— 新票取号统一入口（T-0219）：git fetch 后扫
# origin/master:docs/tasks/ 最大号+1。双席并行「本地最大+1」猜测=撞号
# 根因（T-0217 实证）——远端为准；fetch 败降级按本地 origin/master 快照
# 并报 WARN（此时输出仅参考，push 时 --remote 闸仍会复核）。
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
git fetch origin master --quiet 2>/dev/null \
  || echo "WARN: git fetch 败——按本地 origin/master 快照取号（推送前 --remote 闸复核）" >&2
max=$(git ls-tree -r --name-only origin/master -- docs/tasks/ 2>/dev/null \
  | grep -oE 'T-[0-9]{4,}' | sed 's/^T-//' | sort -n | tail -1)
printf 'T-%04d\n' "$((10#${max:-0} + 1))"
