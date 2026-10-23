#!/usr/bin/env bash
# check-deploy-tree.sh —— D-651 部署树写禁执行器（自 nox 誊入，PHASE 部署树=/opt/phase）
# 判据：
#   - 非部署树（git root realpath != /opt/phase）→ 直通（worktree/开发树零干预）；
#   - MERGE_HEAD 在途 → 放行（部署合并=部署树合法写径）；
#   - 当前分支 != master → 拦（t/* 分支在部署树禁 commit）；
#   - 暂存面含 docs/** 以外路径 → 拦（代码/迁移/配置改动一律走 worktree）；
#   - master + 纯 docs/** 暂存 → warn 放行（部署树台账簿写是常态运维径）。
set -uo pipefail

root=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
[ "$(realpath "$root" 2>/dev/null)" = "/opt/phase" ] || exit 0

if git rev-parse -q --verify MERGE_HEAD >/dev/null 2>&1; then
  exit 0
fi

branch=$(git branch --show-current 2>/dev/null || true)
if [ "$branch" != "master" ]; then
  echo "FAIL(D-651): 部署树 /opt/phase 检出分支=$branch（非 master）禁 commit——" >&2
  echo "  开发走 worktree：git worktree add /opt/phase-wt/<票> -b t/<票>" >&2
  exit 1
fi

staged=$(git -c core.quotepath=off diff --cached --name-only 2>/dev/null || true)
[ -z "$staged" ] && exit 0
bad=$(printf '%s\n' "$staged" | grep -v '^docs/' || true)
if [ -n "$bad" ]; then
  echo "FAIL(D-651): 部署树 master 仅容 docs/** 台账写——以下路径须 worktree 径：" >&2
  printf '%s\n' "$bad" | head -10 >&2
  exit 1
fi
echo "WARN(D-651): 部署树纯 docs 提交放行——代码/配置改动请走 /opt/phase-wt/<票>" >&2
exit 0
