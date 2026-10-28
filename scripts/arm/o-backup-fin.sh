
#!/usr/bin/env bash
# o-backup-fin —— PG-fin（SG 账链）经 o-sg-tunnel 隧道备份落 ARM（T-0173 业主令：
# PG-fin 备份落点跨机跨域）。PG18 客户端出自 timescale 镜像（宿主 PG16 客户端
# 对 SG PG18 服务端版本错位——容器径规避）。gzip+sha256+滚动 14 份。
# 密码：PGPASSWORD 由 root env 文件提供（/etc/o-backup-fin.env，600 非仓库件）。
set -euo pipefail
[ -f /etc/o-backup-fin.env ] && . /etc/o-backup-fin.env
BACKUP_DIR="${BACKUP_DIR:-/opt/o-backups}"
mkdir -p "$BACKUP_DIR" && chmod 700 "$BACKUP_DIR"
ts="$(date -u +%Y%m%dT%H%M%SZ)"
f="$BACKUP_DIR/fin-${ts}.sql.gz"
docker run --rm --network host -e PGPASSWORD="${PGPASSWORD:?PGPASSWORD 缺}" \
  timescale/timescaledb:2.30.2-pg18 \
  pg_dump -h 127.0.0.1 -p 15433 -U o -d o | gzip -9 > "$f"
sha256sum "$f" > "$f.sha256"
echo "[$(date -u +%FT%TZ)] backup OK $f ($(du -h "$f"|cut -f1))"
ls -1t "$BACKUP_DIR"/fin-*.sql.gz 2>/dev/null | tail -n +15 | xargs -r rm -f
ls -1t "$BACKUP_DIR"/fin-*.sql.gz.sha256 2>/dev/null | tail -n +15 | xargs -r rm -f
