#!/usr/bin/env bash
# check_stale_base.sh —— stale 基探针（提示件非阻断件，自 nox 誊入改 origin）。
# 席位本地基落后 origin/master 时读的是陈票：fetch 败=如实报非静默；落后>0→非零退出+warn。
set -uo pipefail
cd "$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
if ! git fetch origin master --quiet 2>/dev/null; then
	echo "[stale-base] 远端 origin 不可达（fetch 败）——如实报出非静默" >&2
	exit 0 # 只报不拦：网络域不可达非基陈旧证据，不阻断
fi
base=$(git merge-base HEAD origin/master 2>/dev/null || true)
if [ -z "$base" ]; then
	echo "[stale-base] merge-base 缺（基线未接）" >&2
	exit 0
fi
behind=$(git rev-list --count "$base..origin/master" 2>/dev/null || echo 0)
if [ "$behind" -gt 0 ]; then
	echo "[stale-base] 本地基落后远端 $behind commits——先 pull 再动（陈票复派/冲突双病灶源）" >&2
	exit 1
fi
exit 0
