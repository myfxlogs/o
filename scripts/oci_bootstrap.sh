#!/usr/bin/env bash
# oci_bootstrap.sh —— OCI 盒级一键复立册（T-0234 件②——盒死重立预案）。
# 跑位=新盒本体（OCI Ubuntu 24.04 ARM A1 · SG West · Ubuntu 24.04，root）。
#
# 零秘密原则：本脚本不含任何凭据。env 值面由业主解 escrow 后供给——
#   SG:/root/oci-secrets/escrow/o-md-escrow-*.tar.age（age 私钥业主离线
#   持有，系统内零副本）→ 解包得 o-collecto.env / o-projector.env / env
#   (o-research) / o-backup-md.env / o-backup-fin.env / sing-box config 等
#   → 置新盒 /root/bootstrap-env/（文件名同盒上原位基名）。
# 钥对现场重生：隧道键/推送键本脚本 keygen 并打印 SG 侧 authorized_keys
# 收口行（贴上即通）；Reality 钥对默认重生（dev mihomo 须同步换参——尾拍
# 打印三参；若业主 escrow 原配置在位则零重生零换参）。
#
# 前置（盒外，序不可倒）：
#   ① OCI 控制台开盒（ARM A1·SG West·Ubuntu 24.04）→ cloud-init/Console
#      贴 dev 公钥 → NSG 双开 22/443
#   ② dev: scp scripts/oci_bootstrap.sh scripts/backup_pg.sh
#      scripts/backup_push_sg.sh scripts/arm/o-backup-md.sh
#      scripts/arm/o-backup-fin.sh scripts/arm/o-sentinel.sh root@新盒:/root/
#   ③ SG→dev→新盒：拉末份 o-md dump（/opt/o-backups/oci-inbox 或 R2 oci/
#      前缀——解密须业主 age 私钥）→ 新盒 /root/restore/
#   ④ 业主: 解 escrow → 新盒 /root/bootstrap-env/（见上）
# 用法：
#   DRY_RUN=1 bash /root/oci_bootstrap.sh        # 干跑——只打印动作不执行
#   bash /root/oci_bootstrap.sh [段...]           # 缺省全序
# 段：sys pgmd scripts units tunnel singbox restore verify
set -euo pipefail

DRY="${DRY_RUN:-0}"
doit() { echo "+ $*"; [ "$DRY" = "1" ] || eval "$@"; }
wfile() { # wfile <dest> —— stdin 写入（dry-run 只打印不落盘）
	local dest="$1"; echo "+ write $dest"
	[ "$DRY" = "1" ] || cat > "$dest"
}
need_env() { # 值面件在位检查（缺=停，指路 escrow）
	local f="/root/bootstrap-env/$1"
	[ -f "$f" ] || { echo "缺 $f——escrow 解包步骤见文件头④"; exit 1; }
}

stage_sys() {
	echo "── sys：apt 面+docker+age+防火墙 ──"
	doit "apt-get update -qq"
	doit "DEBIAN_FRONTEND=noninteractive apt-get install -y -qq docker.io age iptables-persistent"
	doit "usermod -aG docker ubuntu"
	doit "loginctl enable-linger ubuntu"
	doit "iptables -I INPUT -p tcp --dport 443 -m state --state NEW -j ACCEPT"
	doit "netfilter-persistent save"
	echo "  （OCI NSG 侧 22/443 双开须控制台另核——本脚本管不到云面）"
}

stage_pgmd() {
	echo "── pgmd：o-postgres-md 容器（同产线参——卷/端口/restart 亲证 2026-10-29）──"
	doit "docker volume create o_o-pgdata-md"
	doit "docker run -d --name o-postgres-md --restart unless-stopped -v o_o-pgdata-md:/var/lib/postgresql -p 127.0.0.1:5435:5432 -e POSTGRES_USER=o -e POSTGRES_PASSWORD=o -e POSTGRES_DB=o timescale/timescaledb:2.30.2-pg18"
	doit "until docker exec o-postgres-md pg_isready -U o -q; do sleep 2; done"
}

stage_scripts() {
	echo "── scripts：备份链三件+fin/sentinel 件安置 ──"
	local f
	for f in oci_bootstrap.sh backup_pg.sh backup_push_sg.sh o-backup-md.sh o-backup-fin.sh o-sentinel.sh; do
		case "$f" in oci_bootstrap.sh) continue;; esac
		[ -f "/root/$f" ] || { echo "缺 /root/$f——dev 侧 scp 清单见文件头②"; exit 1; }
		doit "install -m 755 /root/$f /usr/local/bin/$f"
	done
	# 推送键（box→SG scp 专用——SG 侧收口行尾拍打印）
	if [ "$DRY" != "1" ] && [ ! -f /root/.ssh/id_ed25519_backup_push ]; then
		ssh-keygen -t ed25519 -N '' -C o-md-backup-push -f /root/.ssh/id_ed25519_backup_push -q
	fi
	doit "chmod 700 /root/.ssh && chmod 600 /root/.ssh/id_ed25519_backup_push"
}

stage_units() {
	echo "── units：md 备份链 unit+env+用户级三 unit ──"
	# o-backup-md.env：escrow 值面优先，模板位兜底
	if [ -f /root/bootstrap-env/o-backup-md.env ]; then
		doit "install -m 600 /root/bootstrap-env/o-backup-md.env /etc/o-backup-md.env"
	else
		wfile /etc/o-backup-md.env <<'EOF'
# 模板位——值面从 escrow 补（T-0234：AGE_PUBKEY=公钥非密件；PUSH_KEY=推送键）
AGE_PUBKEY=<escrow: SG backup-r2.env 同值>
PUSH_KEY=/root/.ssh/id_ed25519_backup_push
EOF
		doit "chmod 600 /etc/o-backup-md.env"
	fi
	# fin env（PGPASSWORD——escrow 值面）
	if [ -f /root/bootstrap-env/o-backup-fin.env ]; then
		doit "install -m 600 /root/bootstrap-env/o-backup-fin.env /etc/o-backup-fin.env"
	fi
	wfile /etc/systemd/system/o-backup-md.service <<'EOF'
[Unit]
Description=o PG-md backup chain (dump+push SG+escrow, daily)
[Service]
Type=oneshot
ExecStart=/usr/local/bin/o-backup-md.sh
EOF
	wfile /etc/systemd/system/o-backup-md.timer <<'EOF'
[Unit]
Description=o PG-md backup daily 02:40 UTC
[Timer]
OnCalendar=*-* 02:40:00 UTC
Persistent=true
[Install]
WantedBy=timers.target
EOF
	wfile /etc/systemd/system/o-sg-tunnel.service <<'EOF'
[Unit]
Description=o SG tunnel (PG-fin 15433 + readcred 8095)
After=network-online.target

[Service]
ExecStart=/usr/bin/ssh -N -T -o BatchMode=yes -o ExitOnForwardFailure=yes -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -i /home/ubuntu/.ssh/id_ed25519 -L 127.0.0.1:15433:127.0.0.1:15433 -L 127.0.0.1:8095:127.0.0.1:8095 root@192.229.85.35
Restart=always
RestartSec=10
User=ubuntu

[Install]
WantedBy=multi-user.target
EOF
	wfile /etc/systemd/system/o-sentinel.service <<'EOF'
[Unit]
Description=o sentinel probe (5min)
[Service]
Type=oneshot
ExecStart=/usr/local/bin/o-sentinel.sh
EOF
	wfile /etc/systemd/system/o-sentinel.timer <<'EOF'
[Unit]
Description=o sentinel every 5min
[Timer]
OnCalendar=*:0/5
Persistent=true
[Install]
WantedBy=timers.target
EOF
	# 用户级三 unit（linger 开启后无会话不死）
	doit "mkdir -p /home/ubuntu/.config/systemd/user /home/ubuntu/o /home/ubuntu/o-research"
	for u in o-collecto o-projector o-research; do
		if [ -f "/root/bootstrap-env/$u.service" ]; then
			doit "install -m 644 /root/bootstrap-env/$u.service /home/ubuntu/.config/systemd/user/$u.service"
		fi
	done
	# env 值面（DSN 口令——escrow 拆件名对齐原位基名；缺件=模板位提示不断链）
	if [ -f /root/bootstrap-env/o-collecto.env ]; then doit "install -m 600 /root/bootstrap-env/o-collecto.env /home/ubuntu/o-collecto.env"; else echo "  缺 o-collecto.env 值面——escrow 解包补入后手装（模板位见脚注）"; fi
	if [ -f /root/bootstrap-env/o-projector.env ]; then doit "install -m 600 /root/bootstrap-env/o-projector.env /home/ubuntu/o-projector.env"; else echo "  缺 o-projector.env 值面——同上"; fi
	if [ -f /root/bootstrap-env/o-sweeper.env ]; then doit "install -m 600 /root/bootstrap-env/o-sweeper.env /home/ubuntu/o-sweeper.env"; else echo "  缺 o-sweeper.env 值面——同上（T-0232 一节点一 env 文件归位件）"; fi
	if [ -f /root/bootstrap-env/env ]; then doit "install -m 600 /root/bootstrap-env/env /home/ubuntu/o-research/env"; else echo "  缺 o-research/env 值面——同上"; fi
	doit "chown -R ubuntu:ubuntu /home/ubuntu/.config /home/ubuntu/o /home/ubuntu/o-research /home/ubuntu/o-*.env"
	doit "systemctl daemon-reload"
	doit "systemctl enable --now o-backup-md.timer o-sentinel.timer"
	doit "systemctl enable o-sg-tunnel.service"
	# 臂体二进制位校验（值面/体面分离——二进制 dev 交叉编供给）
	local b miss=0
	for b in /home/ubuntu/o/collector /home/ubuntu/o/projector /home/ubuntu/o-research/o-research; do
		[ -f "$b" ] || { echo "  臂体缺 $b——dev: GOARCH=arm64 GOOS=linux go build -o <名> ./cmd/<入口>"; miss=1; }
	done
	[ "$miss" = "0" ] || echo "  （臂体补齐后: systemctl --machine=ubuntu@ --user enable --now o-collecto o-projector o-research）"
}

stage_tunnel() {
	echo "── tunnel：隧道键重生+unit 起+SG 收口行 ──"
	if [ "$DRY" != "1" ] && [ ! -f /home/ubuntu/.ssh/id_ed25519 ]; then
		sudo -u ubuntu ssh-keygen -t ed25519 -N '' -C oci-tunnel -f /home/ubuntu/.ssh/id_ed25519 -q
	fi
	doit "chmod 700 /home/ubuntu/.ssh"
	doit "systemctl restart o-sg-tunnel.service || true"
	local pub
	if [ "$DRY" != "1" ]; then pub="$(cat /home/ubuntu/.ssh/id_ed25519.pub)"; else pub="<隧道公钥 dry-run>"; fi
	cat <<EOF

═══ SG 侧收口 1/2（root@SG authorized_keys 追加——隧道无壳键）═══
restrict,command="/bin/false",port-forwarding,permitopen="127.0.0.1:15433",permitopen="127.0.0.1:8095" $pub

═══ SG 侧收口 2/2（root@SG authorized_keys 追加——推送 sftp 键）═══
restrict,command="internal-sftp" $( [ "$DRY" != "1" ] && cat /root/.ssh/id_ed25519_backup_push.pub 2>/dev/null || echo "<推送公钥 dry-run>" )
EOF
}

stage_singbox() {
	echo "── singbox：Reality 出口（escrow 配置优先，缺则重生）──"
	if [ -f /root/bootstrap-env/config.json ]; then
		doit "mkdir -p /etc/sing-box"
		doit "install -m 600 /root/bootstrap-env/config.json /etc/sing-box/config.json"
		echo "  escrow 原配置在位——Reality 三参不换，dev mihomo 零改"
	else
		echo "  无 escrow 配置——钥对重生（dev mihomo oci-reality 须同步换参，尾拍打印）"
		[ -f /usr/local/bin/sing-box ] || { echo "缺 /usr/local/bin/sing-box（产线锁 1.12.4 arm64）——operator scp 或 release 拉"; exit 1; }
		wfile /etc/sing-box/config.json <<'EOF'
{
  "log": { "level": "warn", "timestamp": true },
  "inbounds": [{
    "type": "vless", "tag": "vless-in", "listen": "::", "listen_port": 443,
    "users": [{ "uuid": "__UUID__", "flow": "xtls-rprx-vision" }],
    "tls": { "enabled": true, "server_name": "www.oracle.com",
      "reality": { "enabled": true,
        "handshake": { "server": "www.oracle.com", "server_port": 443 },
        "private_key": "__PRIVKEY__", "short_id": ["__SHORTID__"] } }
  }],
  "outbounds": [{ "type": "direct", "tag": "direct" }]
}
EOF
		if [ "$DRY" != "1" ]; then
			local kp uuid sid pub
			kp="$(/usr/local/bin/sing-box generate reality-keypair)"
			uuid="$(/usr/local/bin/sing-box generate uuid)"
			sid="$(openssl rand -hex 8)"
			pub="$(echo "$kp" | sed -n 's/^PublicKey: //p')"
			local pk; pk="$(echo "$kp" | sed -n 's/^PrivateKey: //p')"
			sed -i "s/__UUID__/$uuid/; s/__PRIVKEY__/$pk/; s/__SHORTID__/$sid/" /etc/sing-box/config.json
			chmod 600 /etc/sing-box/config.json
			echo "═══ dev mihomo oci-reality 换参（三参）═══"
			echo "public-key=$pub uuid=$uuid short_id=$sid"
		fi
	fi
	wfile /etc/systemd/system/sing-box.service <<'EOF'
[Unit]
Description=sing-box vless-reality exit
After=network-online.target
[Service]
ExecStart=/usr/local/bin/sing-box run -c /etc/sing-box/config.json
Restart=always
RestartSec=5
[Install]
WantedBy=multi-user.target
EOF
	doit "systemctl daemon-reload && systemctl enable --now sing-box.service"
}

stage_restore() {
	echo "── restore：末份 o-md dump 回灌（无件=空库+migrate 径亦合法）──"
	local dump
	dump="$(find /root/restore -maxdepth 1 -name 'o-md-*.sql.gz' 2>/dev/null | sort -r | head -1 || true)"
	if [ -n "$dump" ]; then
		echo "+ sha256 校验 $(basename "$dump")（.sha256 件随行；值比对径——哈希件记绝对路径不通用，2026-10-29 实证）"
		if [ "$DRY" != "1" ]; then
			local want got
			want="$(awk '{print $1}' "$dump.sha256")"
			got="$(sha256sum "$dump" | awk '{print $1}')"
			[ "$want" = "$got" ] || { echo "sha256 不符——件面不干净，止"; exit 1; }
			echo "  sha256 OK $got"
		fi
		# 新盒容器为空库——直接灌 o（dev 演练用隔离库系因旧库在位，场景不同）
		doit "gunzip -c '$dump' | docker exec -i o-postgres-md psql -U o -d o -v ON_ERROR_STOP=1"
		doit "docker exec o-postgres-md psql -U o -d o -tAc 'select count(*) from information_schema.tables where table_schema=\"public\"'"
	else
		echo "  /root/restore/ 无 dump——空库+migrate 径："
		echo "    dev: GOARCH=arm64 GOOS=linux go build -o migrate ./cmd/migrate && scp 上盒"
		echo "    盒:  MD_PG_DSN=postgres://o:o@127.0.0.1:5435/o?sslmode=disable ./migrate -domain=md"
	fi
}

stage_verify() {
	echo "── verify：尾拍 ──"
	doit "systemctl list-timers | grep -E 'o-backup|sentinel'"
	doit "docker ps --format '{{.Names}} {{.Status}}' | grep md"
	doit "ss -tlnp | grep -E ':(443|5435|15433|8095)'"
	cat <<'EOF'

═══ 复立尾查清单（盒外人工面）═══
1. SG authorized_keys 两收口行已贴（上文）→ 盒: systemctl restart o-sg-tunnel && 复跑 o-backup-md.service 一次
2. OCI 控制台 NSG 22/443 双开核验
3. dev mihomo oci-reality 三参（仅重生径——escrow 径零改）
4. app_config 灌键（okx.symbols=["*"] 等——源=dev 或 SG app_config 表）
5. SG: mkdir -p /opt/o-backups/oci-inbox /root/oci-secrets/escrow（700）
6. 首份 dump 链验: SG oci-inbox 落件 → backup_pg.sh 手跑 → R2 oci/ 前缀列见
EOF
}

STAGES=(sys pgmd scripts units tunnel singbox restore verify)
if [ $# -gt 0 ]; then STAGES=("$@"); fi
for s in "${STAGES[@]}"; do
	"stage_$s"
done
echo "oci_bootstrap: 完成（DRY_RUN=$DRY）"
