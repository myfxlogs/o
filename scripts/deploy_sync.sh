#!/usr/bin/env bash
# deploy_sync.sh — PHASE 本机→SG 源码同步唯一入口（NOX 同型继承——坑判据照搬）。
# 只推 git 跟踪文件：.env/bin/logs/产物一律不碰 SG 侧。
# 用法：./scripts/deploy_sync.sh [-n 干跑]
set -euo pipefail
cd "$(dirname "$0")/.."
SG="${PHASE_SG_HOST:-root@192.229.85.35}"
DEST="/opt/phase/"
EXTRA=(); [ "${1:-}" = "-n" ] && EXTRA=(-n)
# --files-from=git 跟踪清单（core.quotepath=false CJK 原样——NOX 实证）。
rsync -az "${EXTRA[@]}" --files-from=<(git -c core.quotepath=false ls-files) ./ "$SG:$DEST"

# 陈件 prune：git 已删 .go/migrations 差集定点删——**禁 rsync --delete**
# （坑正典 deploy-sync-stale-files：--files-from 下 --delete 扫删 .env/logs 活件）。
git -c core.quotepath=false ls-files '*.go' 'migrations/*.sql' > /tmp/phase-sync-tracked.$$
ssh "$SG" "cd $DEST 2>/dev/null && find . -name '*.go' -o -name '*.sql' -path './migrations/*' -type f" | sed 's|^\./||' | sort > /tmp/phase-sync-sg.$$ 2>/dev/null || true
sort -u /tmp/phase-sync-tracked.$$ > /tmp/phase-sync-tracked-sort.$$
comm -13 /tmp/phase-sync-tracked-sort.$$ /tmp/phase-sync-sg.$$ > /tmp/phase-sync-stale.$$ || true
stale_n=$(wc -l < /tmp/phase-sync-stale.$$)
if [ "${1:-}" = "-n" ]; then
	[ "$stale_n" -gt 0 ] && { echo "[deploy_sync] 干跑差集 $stale_n 件:"; cat /tmp/phase-sync-stale.$$; } || echo "[deploy_sync] 干跑零差集"
else
	[ "$stale_n" -gt 0 ] && ssh "$SG" "cd $DEST && xargs -d '\n' rm -v" < /tmp/phase-sync-stale.$$ | tail -5
fi
rm -f /tmp/phase-sync-tracked.$$ /tmp/phase-sync-sg.$$ /tmp/phase-sync-tracked-sort.$$ /tmp/phase-sync-stale.$$
echo "[deploy_sync] done→$SG:$DEST"
