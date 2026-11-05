#!/usr/bin/env bash
# arch_daily.sh —— T-0230 arch 日拍三件（OCI 盒 /usr/local/bin 落 cron 02:0x）：
# ①活跃集行情日快照（md_ticks/mark_index 每键末帧→md_daily_snapshot，
#   archive-only 显式标注——非业务刷新面）；
# ②arch_* 保留期清（arch_keep_days 配置键读 app_config，缺=180d 底）；
# ③arch 族 dump+gzip+sha256 滚动 14 份（本地件——T-0229/T-0234 OCI→SG→R2
#   链落成后由 inbox 收罗，本件=零凭据保底径）。
# 用法：scripts/arch_daily.sh（sudo docker 径——OCI ubuntu 免密 sudo 实证）。
set -euo pipefail
CT="${ARCH_CT:-o-postgres-md}"
BACKUP_DIR="${BACKUP_DIR:-/opt/o-backups-md}"
q() { sudo docker exec "$CT" psql -U o -d o -Atc "$1"; }
# T-0372 容错自报（OCI 单件部署面无 scripts/meta 树——内联 ops_esc.sh
# 同式：FIN_PG_DSN 隧道径→ops_event 升件+note 前缀 24h 去重；写径末环
# 败=静默退结构性限度在案）。
esc() { # $1=prefix $2=detail
	local dsn="" ef
	for ef in "${OPS_ESC_ENV_FILE:-}" /home/ubuntu/o-research/env /opt/o-research/env; do
		[ -f "$ef" ] || continue
		dsn=$(grep '^FIN_PG_DSN=' "$ef" 2>/dev/null | cut -d= -f2- | tr -d '"' || true) # tolerate: env 探测——件缺/键缺=跳过本候选非吞败（自报径自身降级面）
		[ -n "$dsn" ] && break
	done
	[ -n "$dsn" ] || return 0
	psql "$dsn" -qAt -c "SELECT 1 FROM ops_event WHERE event='升件' AND note LIKE '$1%' AND ts>now()-interval '24 hours' LIMIT 1" 2>/dev/null | grep -qx 1 && return 0
	psql "$dsn" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note) VALUES('OCI-CRON','升件','','$1——$2')" >/dev/null 2>&1 || true # tolerate: 自报末环——INSERT 败静默退（结构性限度在案）
}
ts="$(date -u +%Y%m%dT%H%M%SZ)"

# ① 日快照（UNIQUE(snap_date,inst_id,src) 幂等——同日重跑覆盖不叠）
sudo docker exec "$CT" psql -U o -d o -c "
INSERT INTO md_daily_snapshot(snap_date,inst_id,last,src)
SELECT CURRENT_DATE,inst_id,last,'tick' FROM (SELECT DISTINCT ON (inst_id) inst_id,last FROM md_ticks ORDER BY inst_id,ts DESC) t
ON CONFLICT (snap_date,inst_id,src) DO UPDATE SET last=EXCLUDED.last,captured_at=now()" >/dev/null
sudo docker exec "$CT" psql -U o -d o -c "
INSERT INTO md_daily_snapshot(snap_date,inst_id,last,src)
SELECT CURRENT_DATE,inst_id,px,'mark_index' FROM (SELECT DISTINCT ON (inst_id) inst_id,px FROM md_mark_index WHERE channel='mark-price' ORDER BY inst_id,ts DESC) t
ON CONFLICT (snap_date,inst_id,src) DO UPDATE SET last=EXCLUDED.last,captured_at=now()" >/dev/null
echo "[$ts] snapshot OK"

# ② 保留期清（arch_keep_days 缺=180）
keep=$(q "SELECT COALESCE((SELECT value::float FROM app_config WHERE key='arch_keep_days'),180)" | tr -d ' ')
keep=${keep%.*}; [ -n "$keep" ] || keep=180
for t in arch_eval arch_signal; do
	q "DELETE FROM $t WHERE arch_ts<now()-make_interval(days=>$keep)" >/dev/null || esc "arch 保留期清扫败" "表=$t keep=${keep}d——arch_* 淤积增长无人知"
done
echo "[$ts] retention OK keep=${keep}d"

# ③ arch dump+滚动 14
mkdir -p "$BACKUP_DIR" && chmod 700 "$BACKUP_DIR" 2>/dev/null || true
f="$BACKUP_DIR/arch-${ts}.sql.gz"
sudo docker exec "$CT" pg_dump -U o -d o -t 'arch_*' -t md_daily_snapshot | gzip -9 > "$f"
sha256sum "$f" > "$f.sha256"
ls -1t "$BACKUP_DIR"/arch-*.sql.gz 2>/dev/null | tail -n +15 | xargs -r rm -f
ls -1t "$BACKUP_DIR"/arch-*.sql.gz.sha256 2>/dev/null | tail -n +15 | xargs -r rm -f
echo "[$ts] arch dump OK $f ($(du -h "$f"|cut -f1))"
