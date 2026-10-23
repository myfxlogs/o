#!/usr/bin/env bash
# git_push.sh —— 带重试的推送包装器（D-588：裸 git push 瞬时失败静默滞留病灶修，自 nox 誊入）。
# 用法：scripts/git_push.sh [remote] [refspec]——缺省 origin HEAD:master。
# 3 次指数退避；终败非零退出+醒目报错（调用方/复审席须处置，禁吞）。
# PHASE 仓唯一远端=origin；请求远端不存在而 origin 在→告警自动改推 origin；皆缺→显式失败。
set -uo pipefail
REMOTE="${1:-origin}"
REFSPEC="${2:-HEAD:master}"
if ! git remote get-url "$REMOTE" >/dev/null 2>&1; then
  if [ "$REMOTE" != "origin" ] && git remote get-url origin >/dev/null 2>&1; then
    echo "⚠️  远端 '$REMOTE' 不存在（本仓唯一远端=origin）——自动改推 origin。" >&2
    REMOTE="origin"
  else
    echo "❌ 远端 '$REMOTE' 不存在且无 origin 可回落——本仓远端清单：" >&2
    git remote -v >&2
    exit 1
  fi
fi
for i in 1 2 3; do
  if git push "$REMOTE" "$REFSPEC" 2>&1; then
    exit 0
  fi
  echo "⚠️  git push 第 $i 次失败，$((i*3))s 后重试…" >&2
  sleep $((i*3))
done
echo "❌ git push $REMOTE $REFSPEC 连续 3 次失败——本地 commit 滞留。" >&2
echo "   处置：查网络/远端权限；恢复后重跑本脚本。pre-commit 会持续 warn 直至推送成功。" >&2
exit 1
