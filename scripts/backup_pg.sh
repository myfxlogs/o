#!/usr/bin/env bash
# backup_pg.sh — O 库备份（pg_dump+gzip+sha256+滚动 14 份——NOX D-159 同款）。
# 仓库外 /opt/o-backups/（600，不进 git/chat）；R2 三落点凭据缺席=首失 marker
# 幂等降级记（凭据业主配合项——E-5 径留位，在案则 rclone+age 加密推）。
# 用法：scripts/backup_pg.sh [PG_DSN] [BACKUP_CONTAINER]；默认容器 o-postgres。
# T-0176 泛化：BACKUP_CONTAINER 环境变量/第二参（o-postgres-md 等同形——
# dev PG-md 备份链同径不重抄）。
# crontab：0 3 * * * /opt/o/scripts/backup_pg.sh >> /opt/o-backups/backup.log 2>&1
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-/opt/o-backups}"
MARKER="$BACKUP_DIR/.r2_missing_marker"

if [ $# -ge 1 ]; then
	PG_DSN="$1"
else
	BACKUP_CONTAINER="${BACKUP_CONTAINER:-${2:-o-postgres}}"
	PG_PORT="$(docker port "$BACKUP_CONTAINER" 5432 2>/dev/null | head -n1 | awk -F: '{print $NF}')"
	[ -n "$PG_PORT" ] || { echo "[$(date -u +%FT%TZ)] backup FAIL: $BACKUP_CONTAINER 端口解析失败" >&2; exit 1; }
	PG_DSN="postgres://o:o@localhost:${PG_PORT}/o?sslmode=disable"
fi

mkdir -p "$BACKUP_DIR" && chmod 700 "$BACKUP_DIR"
ts="$(date -u +%Y%m%dT%H%M%SZ)"
f="$BACKUP_DIR/o-${ts}.sql.gz"
# 走容器内 pg_dump——宿主客户端版本与服务端可能错位（SG 18 服务/16 客户端实证）。
BACKUP_CONTAINER="${BACKUP_CONTAINER:-${2:-o-postgres}}"
if docker exec "$BACKUP_CONTAINER" true 2>/dev/null; then
	docker exec "$BACKUP_CONTAINER" pg_dump -U o -d o | gzip -9 > "$f"
else
	pg_dump "$PG_DSN" | gzip -9 > "$f"
fi
sha256sum "$f" > "$f.sha256"
echo "[$(date -u +%FT%TZ)] backup OK $f ($(du -h "$f"|cut -f1))"

# 滚动 14 份
ls -1t "$BACKUP_DIR"/o-*.sql.gz 2>/dev/null | tail -n +15 | xargs -r rm -f
ls -1t "$BACKUP_DIR"/o-*.sql.gz.sha256 2>/dev/null | tail -n +15 | xargs -r rm -f

# R2 三落点（凭据在案才走——$BACKUP_DIR/backup-r2.env 仓库外 600：
# AGE_PUBKEY/R2_REMOTE/R2_BUCKET；缺席=首失 marker 幂等记，续日静默）
if [ -f "$BACKUP_DIR/backup-r2.env" ]; then
	. "$BACKUP_DIR/backup-r2.env"
	age -r "$AGE_PUBKEY" -o "$f.age" "$f" \
		&& rclone copyto "$f.age" "$R2_REMOTE:$R2_BUCKET/o-$(basename "$f" .sql.gz).age" \
		&& rm -f "$f.age" \
		&& echo "[$(date -u +%FT%TZ)] R2 pushed（三落点）" \
		|| echo "[$(date -u +%FT%TZ)] R2 FAIL——本地全保留" >&2
	rm -f "$MARKER"
else
	if [ ! -f "$MARKER" ]; then
		touch "$MARKER"
		echo "[$(date -u +%FT%TZ)] R2 凭据缺席——本地双份降级（首失记 marker；业主配合项）" >&2
	fi
fi
