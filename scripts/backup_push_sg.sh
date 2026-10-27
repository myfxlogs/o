#!/usr/bin/env bash
# backup_push_sg.sh —— dev→SG 备份推送件（T-0176 最优构：dev 纯 SSH 推件零新
# 凭据——凭据主权梯 SG 独占不破；scp 最新 o-md 件→SG:/opt/o-backups/dev-inbox/
# 由 SG 单加密径统推 R2）。用法：dump 后 cron 内链调。
set -euo pipefail
BACKUP_DIR="${BACKUP_DIR:-/opt/o-backups-md}"
SG_HOST="${SG_HOST:-root@192.229.85.35}"
INBOX="${SG_INBOX:-/opt/o-backups/dev-inbox}"
latest=$(ls -1t "$BACKUP_DIR"/o-*.sql.gz 2>/dev/null | head -1)
[ -n "$latest" ] || { echo "[$(date -u +%FT%TZ)] push_sg FAIL: 本机备份件缺" >&2; exit 1; }
# 重命名 o-md- 前缀（R2 dev/ 前缀隔离——sg/o-* 与 dev/o-md-* 键面钦定）
base="o-md-$(basename "$latest" | sed 's/^o-//')"
scp -q -o BatchMode=yes -o ConnectTimeout=10 "$latest" "${SG_HOST}:${INBOX}/${base}"
scp -q -o BatchMode=yes -o ConnectTimeout=10 "$latest.sha256" "${SG_HOST}:${INBOX}/${base}.sha256" 2>/dev/null || true
echo "[$(date -u +%FT%TZ)] push_sg OK $base → $INBOX"
