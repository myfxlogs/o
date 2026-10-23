#!/usr/bin/env bash
# backup_pg.sh：PostgreSQL 每日备份（pg_dump + gzip + sha256 + 滚动 14 份，自 nox 誊入改 PHASE）。
# 仓库外目录+权限 600+checksum+不进 git/docs/LOG/聊天输出。
# 第三落点：age 加密 → rclone 上传 R2 → R2 侧滚动 14 份，消除双机同灭 DR 缺口。
#   配置缺失=降级「仅本地」；加密/上传失败=alerts critical（D-033 自盯）+本地全保留；
#   fail-open 只许在上传段，本地链路任何情况下不阻不断。
#   降级降噪：marker（$BACKUP_DIR/.r2_degrade_state）——首失必发+落 marker，续日静默，
#   恢复清 marker，再失按首失再发。
#
# 用法：scripts/backup_pg.sh [PG_DSN]
# 默认 PG_DSN 端口取 phase-postgres 运行容器发布值（宿主拓扑权威源，勿写死端口）。
# 备份目录：/opt/phase-backups/（权限 700，仓库外；env BACKUP_DIR 可覆盖——仅测试/演练用）
# R2 外推配置：$BACKUP_DIR/backup-r2.env（仓库外 600，含 AGE_PUBKEY/R2_REMOTE/R2_BUCKET；私钥业主离线持有）
# crontab：0 3 * * * /opt/phase/scripts/backup_pg.sh >> /opt/phase-backups/backup.log 2>&1
# 还原 runbook（10-20 演练实证——裸容器还原=hypertable 数据静默全丢）：
#   ①目标实例=tsdb 镜像+ -c shared_preload_libraries=timescaledb,pg_stat_statements
#   ②角色预建：psql -c "CREATE ROLE phase_ro LOGIN"（dump 含其 GRANT——不建=角色错噪音）
#   ③对账锚：restore 后核心表 count vs 生产差≤采样窗增量；scratch 清收 docker rm -v <ctr>

set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-/opt/phase-backups}"
emit_alert() {
	if psql "$PG_DSN" -q -v ON_ERROR_STOP=1 \
		-c "INSERT INTO alerts (ts, level, message, acked) VALUES (now(), 'critical', '$1', false);" >/dev/null 2>&1; then
		echo "[$(date -u +%FT%TZ)] alert critical inserted: $1"
	else
		echo "[$(date -u +%FT%TZ)] ERROR: ALERT(critical, psql 不可达兜底): $1" >&2
	fi
}

if [ $# -ge 1 ]; then
	PG_DSN="$1"
else
	PHASE_PG_PORT="$(docker port phase-postgres 5432 2>/dev/null | head -n1 | awk -F: '{print $NF}')"
	[ -n "$PHASE_PG_PORT" ] || { echo "[$(date -u +%FT%TZ)] backup FAIL: phase-postgres 端口解析失败（传 PG_DSN 显式旁路）" >&2; emit_alert "[backup] phase-postgres 端口解析失败（pg_dump 前置死）"; exit 1; }
	PG_DSN="postgres://phase:phase@localhost:${PHASE_PG_PORT}/phase?sslmode=disable"
fi
RETENTION_DAYS=13
R2_KEEP=14
STATUS=0

mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"

R2_DEGRADE_MARKER="$BACKUP_DIR/.r2_degrade_state"
notify_r2_degrade() {
	if [ -f "$R2_DEGRADE_MARKER" ]; then
		local day
		day="$(cat "$R2_DEGRADE_MARKER" 2>/dev/null || true)"
		case "$day" in ''|*[!0-9]*) day=0 ;; esac
		day=$((day + 1))
		if echo "$day" > "$R2_DEGRADE_MARKER" 2>/dev/null; then
			echo "[$(date -u +%FT%TZ)] INFO: r2 degrade ongoing day=$day, critical suppressed (state-transition dedup)" >&2
		else
			echo "[$(date -u +%FT%TZ)] WARN: r2 degrade ongoing (day write failed), critical suppressed (state-transition dedup)" >&2
		fi
	else
		emit_alert "[backup] R2 外推降级 critical 首报：config missing→local-only（三地失守单地 DR 在案）"
		if ! echo 1 > "$R2_DEGRADE_MARKER" 2>/dev/null; then
			echo "[$(date -u +%FT%TZ)] WARN: r2 degrade marker write failed — next run re-alerts (fail-noisy)" >&2
		fi
	fi
}

# 读 R2 外推配置。文件缺失/不可读/源失败/任一变量空 → 降级：跳过加密与上传，本地照常。
# 配置在而 age/rclone 不可用不算降级——走 critical（DR 缺口无声打开比降级更危险）。
R2_ENV_FILE="$BACKUP_DIR/backup-r2.env"
R2_CONFIGURED=0
AGE_PUBKEY="" R2_REMOTE="" R2_BUCKET=""
if [ -f "$R2_ENV_FILE" ] && . "$R2_ENV_FILE" \
	&& [ -n "$AGE_PUBKEY" ] && [ -n "$R2_REMOTE" ] && [ -n "$R2_BUCKET" ]; then
	R2_CONFIGURED=1
	echo "[$(date -u +%FT%TZ)] r2 offload enabled: bucket=$R2_BUCKET keep=$R2_KEEP"
	if [ -f "$R2_DEGRADE_MARKER" ] && ! rm -f "$R2_DEGRADE_MARKER"; then
		echo "[$(date -u +%FT%TZ)] WARN: r2 degrade marker clear failed" >&2
	fi
else
	notify_r2_degrade
fi

# R2 侧滚动：phase_*.sql.gz.age 按对象时间序保留最新 R2_KEEP 份，删最旧。
r2_rotate() {
	local listing total excess obj deleted=0
	if ! listing=$(rclone lsl "$R2_REMOTE:$R2_BUCKET/" | awk 'NF>=4 && $NF ~ /^phase_/ && $NF ~ /\.sql\.gz\.age$/'); then
		echo "[$(date -u +%FT%TZ)] WARN: rclone lsl failed, skip R2 rotation" >&2
		return 0
	fi
	total=0
	if [ -n "$listing" ]; then
		total=$(printf '%s\n' "$listing" | wc -l)
	fi
	[ "$total" -gt "$R2_KEEP" ] || return 0
	excess=$((total - R2_KEEP))
	while read -r obj; do
		[ -n "$obj" ] || continue
		if rclone deletefile "$R2_REMOTE:$R2_BUCKET/$obj"; then
			deleted=$((deleted + 1))
		else
			echo "[$(date -u +%FT%TZ)] WARN: rclone deletefile failed: $obj" >&2
		fi
	done < <(printf '%s\n' "$listing" | LC_ALL=C sort -k2,2 -k3,3 | head -n "$excess" | awk '{print $NF}')
	echo "[$(date -u +%FT%TZ)] r2 rotation: total=$total kept=$((total - deleted)) deleted=$deleted (limit $R2_KEEP)"
}

TIMESTAMP=$(date -u +"%Y%m%d_%H%M%S")
DUMP_FILE="$BACKUP_DIR/phase_${TIMESTAMP}.sql.gz"
SHA_FILE="$DUMP_FILE.sha256"

echo "[$(date -u +%FT%TZ)] backup start: $DUMP_FILE"

# pg_dump | gzip（管道避免明文落盘）
if ! pg_dump "$PG_DSN" | gzip > "$DUMP_FILE"; then
	echo "[$(date -u +%FT%TZ)] backup FAIL: pg_dump error" >&2
	emit_alert "[backup] pg_dump error（本地备份链断）"
	rm -f "$DUMP_FILE"
	ls -1t "$BACKUP_DIR"/phase_*.sql.gz 2>/dev/null | tail -n +15 | xargs -r rm -f
	exit 1
fi

chmod 600 "$DUMP_FILE"

sha256sum "$DUMP_FILE" > "$SHA_FILE"
chmod 600 "$SHA_FILE"

# 滚动 14 份（RETENTION_DAYS=13+当日）
find "$BACKUP_DIR" -name "phase_*.sql.gz" -mtime +${RETENTION_DAYS} -delete
find "$BACKUP_DIR" -name "phase_*.sql.gz.sha256" -mtime +${RETENTION_DAYS} -delete
find "$BACKUP_DIR" -name "phase_*.sql.gz.age" -mtime +${RETENTION_DAYS} -delete

SIZE=$(stat -c%s "$DUMP_FILE" 2>/dev/null || stat -f%z "$DUMP_FILE")
echo "[$(date -u +%FT%TZ)] backup OK: $DUMP_FILE (${SIZE} bytes), sha256=$SHA_FILE"

# ---- 第三落点：age 加密 → rclone 上传 R2 → R2 滚动（失败本地全保留，STATUS 只反映外推段）----
if [ "$R2_CONFIGURED" -eq 1 ]; then
	AGE_FILE="$DUMP_FILE.age"
	if ! age -r "$AGE_PUBKEY" -o "$AGE_FILE" "$DUMP_FILE"; then
		echo "[$(date -u +%FT%TZ)] encrypt FAIL: age error, local dump kept" >&2
		emit_alert "[backup] age encrypt failed, local dump kept"
		STATUS=1
	else
		chmod 600 "$AGE_FILE"
		if ! rclone copy "$AGE_FILE" "$R2_REMOTE:$R2_BUCKET/"; then
			echo "[$(date -u +%FT%TZ)] upload FAIL: rclone error, local kept" >&2
			emit_alert "[backup] rclone upload R2 failed, local kept"
			STATUS=1
		else
			AGE_SIZE=$(stat -c%s "$AGE_FILE" 2>/dev/null || stat -f%z "$AGE_FILE")
			R2_SHA=$(awk '{print $1}' "$SHA_FILE")
			echo "[$(date -u +%FT%TZ)] rclone OK: $R2_REMOTE:$R2_BUCKET/$(basename "$AGE_FILE") (${AGE_SIZE} bytes), sha256(dump)=$R2_SHA"
			r2_rotate
		fi
	fi
fi

# 机器报告备份（docs/报告 不入 git；目录未建=跳过不带崩备份）
if [ -d "$(dirname "$0")/../docs/报告" ]; then
	mkdir -p "$BACKUP_DIR/reports" && rsync -a --delete "$(dirname "$0")/../docs/报告/" "$BACKUP_DIR/reports/"
fi

exit "$STATUS"
