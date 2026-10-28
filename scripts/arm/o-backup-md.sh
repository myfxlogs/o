#!/usr/bin/env bash
# o-backup-md —— OCI PG-md 日备份链总闸（T-0234 业主令：OCI 撤退战备——
# PG-md 零备份根治）。链序：
#   ① backup_pg.sh——容器内 dump+gzip+sha256+滚动 14 份（BACKUP_CONTAINER=
#     o-postgres-md；R2_DISABLED=1 纯本地段——R2 凭据主权梯 SG 独占不破）
#   ② backup_push_sg.sh——scp 推 SG /opt/o-backups/oci-inbox/（o-md- 前缀
#     钦定不变；SG 03:00 UTC cron 收罗统推 R2 oci/ 前缀）
#   ③ 配置 escrow——env/单元/sing-box/ssh 件 tar+age 同键加密+scp 落 SG
#     /root/oci-secrets/escrow/（滚动 14 份由 SG backup_pg.sh 裁尾；小件
#     零成本，盒死即配置面全在 SG 信托锚）。
# 凭据面：零新凭据——本机只持 AGE_PUBKEY（公钥非密件，加密径同键）；推送
# 专用键 /root/.ssh/id_ed25519_backup_push（SG authorized_keys restrict+
# internal-sftp 收口）；escrow 私钥面除外（钥对现场重生，径见 oci_bootstrap.sh）。
# 调度：systemd o-backup-md.timer 02:40 UTC（SG 收罗 03:00 UTC 前位——inbox
# 陈化 ~20min，26h 阈余量拉满）；env=/etc/o-backup-md.env（600 非仓库件）。
set -euo pipefail
[ -f /etc/o-backup-md.env ] && . /etc/o-backup-md.env
BACKUP_DIR="${BACKUP_DIR:-/opt/o-backups}"
export BACKUP_DIR
export BACKUP_CONTAINER="${BACKUP_CONTAINER:-o-postgres-md}"
export R2_DISABLED=1
export PUSH_KEY="${PUSH_KEY:-/root/.ssh/id_ed25519_backup_push}"
SG_HOST="${SG_HOST:-root@192.229.85.35}"
ESCROW_SG_DIR="${ESCROW_SG_DIR:-/root/oci-secrets/escrow}"

mkdir -p "$BACKUP_DIR" && chmod 700 "$BACKUP_DIR"

# 本地 escrow 残件裁尾（上次推败遗留——案 14 份；find 径=空匹配 exit 0，
# ls 空匹配 exit 2 × pipefail=静默炸链——2026-10-29 首跑实证）
find "$BACKUP_DIR" -maxdepth 1 -name 'o-md-escrow-*.tar.age' | sort -r | tail -n +15 | xargs -r rm -f
find "$BACKUP_DIR" -maxdepth 1 -name 'o-md-escrow-*.tar.age.sha256' | sort -r | tail -n +15 | xargs -r rm -f

# ① 本地 dump 段（与 SG 库守同件同构）
/usr/local/bin/backup_pg.sh

# ② 推 SG 段
/usr/local/bin/backup_push_sg.sh

# ③ 配置 escrow（--ignore-failed-read=件缺不炸链；私钥永不入 tar）
: "${AGE_PUBKEY:?AGE_PUBKEY 缺——/etc/o-backup-md.env}"
ts="$(date -u +%Y%m%dT%H%M%SZ)"
escrow="$BACKUP_DIR/o-md-escrow-${ts}.tar.age"
tar -C / --ignore-failed-read -cf - \
	etc/o-backup-md.env \
	etc/o-backup-fin.env \
	etc/systemd/system/o-backup-fin.service \
	etc/systemd/system/o-backup-fin.timer \
	etc/systemd/system/o-backup-md.service \
	etc/systemd/system/o-backup-md.timer \
	etc/systemd/system/o-sg-tunnel.service \
	etc/systemd/system/o-sentinel.service \
	etc/systemd/system/o-sentinel.timer \
	etc/sing-box/config.json \
	root/.ssh/authorized_keys \
	home/ubuntu/.ssh/authorized_keys \
	home/ubuntu/.ssh/known_hosts \
	home/ubuntu/o-collecto.env \
	home/ubuntu/o-projector.env \
	home/ubuntu/o-research/env \
	home/ubuntu/.config/systemd/user/o-collecto.service \
	home/ubuntu/.config/systemd/user/o-projector.service \
	home/ubuntu/.config/systemd/user/o-research.service \
	2>/dev/null | age -r "$AGE_PUBKEY" -o "$escrow"
sha256sum "$escrow" > "$escrow.sha256"
KEYOPT=()
if [ -n "$PUSH_KEY" ]; then KEYOPT=(-i "$PUSH_KEY"); fi
scp -q -o BatchMode=yes -o ConnectTimeout=10 "${KEYOPT[@]}" \
	"$escrow" "$escrow.sha256" "${SG_HOST}:${ESCROW_SG_DIR}/"
rm -f "$escrow" "$escrow.sha256"
echo "[$(date -u +%FT%TZ)] escrow OK $(basename "$escrow") → ${SG_HOST}:${ESCROW_SG_DIR}/"
