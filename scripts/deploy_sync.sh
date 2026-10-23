#!/usr/bin/env bash
# deploy_sync.sh — 本机→SG 源码同步唯一入口（自 nox 誊入；2026-10-09 xasset 断写事故后钉死）。
# 只推 git 跟踪文件：.env/bin/logs/运行时产物一律不碰 SG 侧。
# 用法：./scripts/deploy_sync.sh [-n 干跑]
# PHASE 现状：SG 部署为「待切流」——脚本就位不实际执行（T-0002 范围外：不切流生产）。
set -euo pipefail
cd "$(dirname "$0")/.."
SG="${PHASE_SG_HOST:-root@192.229.85.35}"
DEST="/opt/phase/"
EXTRA=(); [ "${1:-}" = "-n" ] && EXTRA=(-n)
# --files-from=git 跟踪清单：非跟踪文件不推不删。
# core.quotepath=false：CJK 文件名原样输出（默认 C 引用转义会让 rsync link_stat 失配）。
rsync -az "${EXTRA[@]}" --files-from=<(git -c core.quotepath=false ls-files) ./ "$SG:$DEST"

# 陈件 prune：git 已删 .go 永留 SG 进目录编译面炸 build。差集=SG find '*.go'
# − git ls-files '*.go'，按清单定点 rm。**禁 rsync --delete**（--files-from
# 下会扫删非跟踪活件）。干跑=差集只报不删。
git -c core.quotepath=false ls-files '*.go' > /tmp/deploy-sync-tracked-go.$$
ssh "$SG" "cd $DEST && find . -name '*.go' -type f" | sed 's|^\./||' | sort > /tmp/deploy-sync-sg-go.$$
sort -u /tmp/deploy-sync-tracked-go.$$ > /tmp/deploy-sync-tracked-go-sort.$$
comm -13 /tmp/deploy-sync-tracked-go-sort.$$ /tmp/deploy-sync-sg-go.$$ > /tmp/deploy-sync-stale-go.$$
stale_n=$(wc -l < /tmp/deploy-sync-stale-go.$$)
if [ "${1:-}" = "-n" ]; then
	[ "$stale_n" -gt 0 ] && echo "[deploy_sync] 干跑差集 $stale_n 件:" && cat /tmp/deploy-sync-stale-go.$$
else
	if [ "$stale_n" -gt 0 ]; then
		ssh "$SG" "cd $DEST && xargs -d '\n' rm -v" < /tmp/deploy-sync-stale-go.$$ | tail -5
		echo "[deploy_sync] prune 陈件 $stale_n 件"
	fi
fi
rm -f /tmp/deploy-sync-tracked-go.$$ /tmp/deploy-sync-sg-go.$$ \
	/tmp/deploy-sync-tracked-go-sort.$$ /tmp/deploy-sync-stale-go.$$

# 宿主侧二进制随源码重建——rsync 只推 git 跟踪面，不重建则 cron 跑旧码。
ssh "$SG" "cd $DEST && bash scripts/build_host_tools.sh" >/dev/null
