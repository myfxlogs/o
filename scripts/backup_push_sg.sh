#!/usr/bin/env bash
# backup_push_sg.sh —— md 备份推 SG 件（T-0176 立构 dev→SG；T-0234 方向反转
# OCI→SG：scp 最新 o-md 件→SG:/opt/o-backups/oci-inbox/，由 SG 单加密径统推
# R2 oci/ 前缀——前缀=数据产地，dev/ 存量历史键冻结）。零新凭据——R2/age
# 主权梯 SG 独占不破；推送 SSH 键经 PUSH_KEY 注入（SG 侧 restrict+
# internal-sftp 收口，无壳无转发；dev 期免键径随前缀归位退役）。
# 用法：dump 后链调；BACKUP_DIR=本机备份面，PUSH_KEY=推送键路径。
set -euo pipefail
BACKUP_DIR="${BACKUP_DIR:-/opt/o-backups}"
SG_HOST="${SG_HOST:-root@192.229.85.35}"
INBOX="${SG_INBOX:-/opt/o-backups/oci-inbox}"
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
# latest 侦取用 (ls||true) 径——ls 空匹配 exit 2 × pipefail 会绕过下方
# 「备份件缺」守卫静默退（守卫语义保真：空件仍走 FAIL 分支）
latest=$({ ls -1t "$BACKUP_DIR"/o-*.sql.gz 2>/dev/null || true; } | head -1)
[ -n "$latest" ] || { echo "[$(date -u +%FT%TZ)] push_sg FAIL: 本机备份件缺" >&2; exit 1; }
# 重命名 o-md- 前缀（R2 oci/ 前缀隔离——sg/o-* 与 oci/o-md-* 键面钦定不变）
base="o-md-$(basename "$latest" | sed 's/^o-//')"
KEYOPT=()
if [ -n "${PUSH_KEY:-}" ]; then KEYOPT=(-i "$PUSH_KEY"); fi
scp -q -o BatchMode=yes -o ConnectTimeout=10 "${KEYOPT[@]}" "$latest" "${SG_HOST}:${INBOX}/${base}"
scp -q -o BatchMode=yes -o ConnectTimeout=10 "${KEYOPT[@]}" "$latest.sha256" "${SG_HOST}:${INBOX}/${base}.sha256" 2>/dev/null || esc "sha256 校验件推败" "base=$base——SG 侧校验面残件（dump 本体已达 inbox）"
echo "[$(date -u +%FT%TZ)] push_sg OK $base → $INBOX"
