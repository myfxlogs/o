#!/usr/bin/env bash
# backup_pg.sh — O 库备份（pg_dump+gzip+sha256+滚动 14 份——NOX D-159 同款）。
# 仓库外 /opt/o-backups/（600，不进 git/chat）；R2 三落点凭据缺席=首失 marker
# 幂等降级记（凭据业主配合项——E-5 径留位，在案则 rclone+age 加密推）。
# 用法：scripts/backup_pg.sh [PG_DSN] [BACKUP_CONTAINER]；默认容器 o-postgres。
# T-0176 泛化：BACKUP_CONTAINER 环境变量/第二参（o-postgres-md 等同形——
# dev PG-md 备份链同径不重抄）。
# T-0234 前缀归位（业主令：前缀=数据产地）：收罗段 dev-inbox→dev/ 改
# oci-inbox→oci/（OCI 推件落地）；R2 dev/ 存量历史键冻结留原地（dev 段退役）
# ；陈化哨兵检查径随改名；escrow 推件滚动段落位（/root/oci-secrets/escrow 案
# 14 份）；R2_DISABLED=1=本机纯本地段显式跳过（OCI 侧用——R2 凭据主权梯 SG
# 独占，凭据缺席属设计而非失格，不记 marker）。
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
# AGE_PUBKEY/R2_REMOTE/R2_BUCKET；缺席=首失 marker 幂等记，续日静默）。
# T-0176 单加密径统前缀——sg/o-*.sql.gz.age（本机件）；T-0234 收罗面改
# oci/o-md-*.sql.gz.age（oci-inbox 收罗件）；滚动 14 份/前缀（lsl+deletefile
# ——只推不转=无限胀病灶根修）；oci-inbox 陈 >26h=推链死→CRITICAL（响非默）。
if [ "${R2_DISABLED:-0}" = "1" ]; then
	echo "[$(date -u +%FT%TZ)] R2 段跳过（R2_DISABLED=1——纯本地段，R2 由 SG 单加密径统推）"
elif [ -f "$BACKUP_DIR/backup-r2.env" ]; then
	. "$BACKUP_DIR/backup-r2.env"
	age -r "$AGE_PUBKEY" -o "$f.age" "$f" \
		&& rclone copyto "$f.age" "$R2_REMOTE:$R2_BUCKET/sg/$(basename "$f" .sql.gz).age" \
		&& rm -f "$f.age" \
		&& echo "[$(date -u +%FT%TZ)] R2 pushed（sg/ 前缀）" \
		|| echo "[$(date -u +%FT%TZ)] R2 FAIL——本地全保留" >&2
	# oci-inbox 收罗（OCI scp 推件——凭据主权梯 SG 独占不破）
	if [ -d "$BACKUP_DIR/oci-inbox" ]; then
		for df in "$BACKUP_DIR/oci-inbox"/o-md-*.sql.gz; do
			[ -f "$df" ] || continue
			age -r "$AGE_PUBKEY" -o "$df.age" "$df" \
				&& rclone copyto "$df.age" "$R2_REMOTE:$R2_BUCKET/oci/$(basename "$df" .sql.gz).age" \
				&& rm -f "$df.age" "$df" "$df.sha256" \
				&& echo "[$(date -u +%FT%TZ)] R2 oci/ pushed $(basename "$df")" \
				|| echo "[$(date -u +%FT%TZ)] R2 oci/ FAIL $(basename "$df")——inbox 保留" >&2
		done
		oldest=$(find "$BACKUP_DIR/oci-inbox" -name 'o-md-*.sql.gz' -mmin +1560 -print -quit 2>/dev/null)
		if [ -n "$oldest" ]; then
			echo "[$(date -u +%FT%TZ)] CRITICAL oci-inbox 陈 >26h $(basename "$oldest")——推链死告警" >&2
		fi
	fi
	# R2 滚动 14 份/前缀（lsl 正序+deletefile 斩尾——前缀隔离互不打；
	# dev/ 历史键冻结不转——T-0234 前缀归位，存量留原地）
	for pfx in sg oci; do
		rclone lsl "$R2_REMOTE:$R2_BUCKET/$pfx/" 2>/dev/null | awk '{print $NF}' | sort \
			| head -n -14 | while read -r oldobj; do
			[ -n "$oldobj" ] && rclone deletefile "$R2_REMOTE:$R2_BUCKET/$pfx/$oldobj" \
				&& echo "[$(date -u +%FT%TZ)] R2 $pfx/ 滚动清 $oldobj"
		done
	done
	rm -f "$MARKER"
else
	if [ ! -f "$MARKER" ]; then
		touch "$MARKER"
		echo "[$(date -u +%FT%TZ)] R2 凭据缺席——本地双份降级（首失记 marker；业主配合项）" >&2
	fi
fi
# o-md escrow 滚动（OCI 推件落 SG /root/oci-secrets/escrow/——T-0234 件③，
# 盒级配置信托锚；案 14 份与 dump 同节奏裁尾。OCI 侧无此目录=自然跳过。
# find 径=空匹配 exit 0——grep/ls 空匹配非零 × pipefail=静默炸链实证）
ESCROW_DIR="${ESCROW_DIR:-/root/oci-secrets/escrow}"
if [ -d "$ESCROW_DIR" ]; then
	find "$ESCROW_DIR" -maxdepth 1 -name 'o-md-escrow-*.tar.age' | sort -r | tail -n +15 \
		| while read -r old; do rm -f "$old" "$old.sha256"; done
fi