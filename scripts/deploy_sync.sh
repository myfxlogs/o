#!/usr/bin/env bash
# deploy_sync.sh — O 本机→SG 源码同步唯一入口（NOX 同型继承——坑判据照搬）。
# 只推 git 跟踪文件：.env/bin/logs/产物一律不碰 SG 侧。
# 用法：./scripts/deploy_sync.sh [-n 干跑]
set -euo pipefail
cd "$(dirname "$0")/.."
SG="${O_SG_HOST:-root@192.229.85.35}"
DEST="/opt/o/"
EXTRA=(); [ "${1:-}" = "-n" ] && EXTRA=(-n)
# --files-from=git 跟踪清单（core.quotepath=false CJK 原样——NOX 实证）。
rsync -az "${EXTRA[@]}" --files-from=<(git -c core.quotepath=false ls-files) ./ "$SG:$DEST"

# 陈件 prune：git 已删 .go/migrations 差集定点删——**禁 rsync --delete**
# （坑正典 deploy-sync-stale-files：--files-from 下 --delete 扫删 .env/logs 活件）。
# T-0054：internal/gen/** 排除——gitignore 产物不在跟踪清单，计入差集=SG 侧生成码
# 每轮被误删（镜像内 buf regen 自洽后宿主/SG gen 面均免依赖，此处只防误删活件）。
git -c core.quotepath=off ls-files '*.go' 'migrations/*.sql' > /tmp/o-sync-tracked.$$
ssh "$SG" "cd $DEST 2>/dev/null && find . \\( -name '*.go' -o -name '*.sql' -path './migrations/*' \\) -type f -not -path './internal/gen/*'" | sed 's|^\./||' | sort > /tmp/o-sync-sg.$$ 2>/dev/null || true
sort -u /tmp/o-sync-tracked.$$ > /tmp/o-sync-tracked-sort.$$
comm -13 /tmp/o-sync-tracked-sort.$$ /tmp/o-sync-sg.$$ > /tmp/o-sync-stale.$$ || true
stale_n=$(wc -l < /tmp/o-sync-stale.$$)
if [ "${1:-}" = "-n" ]; then
	[ "$stale_n" -gt 0 ] && { echo "[deploy_sync] 干跑差集 $stale_n 件:"; cat /tmp/o-sync-stale.$$; } || echo "[deploy_sync] 干跑零差集"
else
	[ "$stale_n" -gt 0 ] && ssh "$SG" "cd $DEST && xargs -d '\n' rm -v" < /tmp/o-sync-stale.$$ | tail -5
fi
rm -f /tmp/o-sync-tracked.$$ /tmp/o-sync-sg.$$ /tmp/o-sync-tracked-sort.$$ /tmp/o-sync-stale.$$
echo "[deploy_sync] done→$SG:$DEST"

# T-0181 漂移哨兵挂验收序：部署尾拍对拍 fin/md 链 live version vs 登记表
# （落后即红——「已合并≠已执行」灰色态自此由机器报不靠撞出来）。
if [ -x scripts/check_schema_live.sh ]; then
	bash scripts/check_schema_live.sh || echo "[deploy_sync] WARN schema 漂移哨兵报红（见上——补迁=显式动作非自动）"
fi
