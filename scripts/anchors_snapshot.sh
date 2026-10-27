#!/usr/bin/env bash
# anchors_snapshot.sh —— T-0169 ④ 反证锚快照（72h 实拍）：SG PG CPU/隧道载荷/
# lifecycle 扫拍时长三锚+暴力计数复核（口径=journalctl -u ssh——fail2ban systemd
# backend 同源，T-0167 基线 2,840/24h 即此口径钉源）。
# 用法：scripts/anchors_snapshot.sh [窗口秒，缺省 60]
set -euo pipefail
WIN="${1:-60}"
SG="${SG:-sg}"   # ssh host 别名（~/.ssh/config）

echo "== 锚① SG o-postgres CPU%（docker stats 单拍+容器累计）=="
ssh "$SG" "docker stats --no-stream --format '{{.Name}} CPU={{.CPUPerc}} MEM={{.MemUsage}}' o-postgres o-core"

echo "== 锚② 隧道载荷（5544 连接 60s 字节增量——dev 侧 ss 采样）=="
b1=$(ss -tin '( dport = :5544 or sport = :5544 )' 2>/dev/null | grep -oE 'bytes_sent:[0-9]+|bytes_received:[0-9]+' | awk -F: '{s+=$2} END {print s+0}')
sleep "$WIN"
b2=$(ss -tin '( dport = :5544 or sport = :5544 )' 2>/dev/null | grep -oE 'bytes_sent:[0-9]+|bytes_received:[0-9]+' | awk -F: '{s+=$2} END {print s+0}')
echo "tunnel_bytes_per_sec=$(( (b2 - b1) / WIN ))（窗口 ${WIN}s 总差 $((b2 - b1))）"
echo "行/分代理：SG md_* 近 60s 收行（tunnel 源+SG 本地共写合计——真分源比以 client_addr 对拍期回填）"
ssh "$SG" "docker exec o-postgres psql -U o -d o -tA -c \"
	SELECT 'md_ticks', count(*) FROM md_ticks WHERE recv_ts>now()-interval '60 seconds'
	UNION ALL SELECT 'md_mark_index', count(*) FROM md_mark_index WHERE recv_ts>now()-interval '60 seconds'
	UNION ALL SELECT 'md_funding', count(*) FROM md_funding WHERE recv_ts>now()-interval '60 seconds'
	UNION ALL SELECT 'md_signal', count(*) FROM md_signal WHERE ts>now()-interval '60 seconds';\""

echo "== 锚③ lifecycle 扫拍时长（journal 近 1h 拍耗时线）=="
ssh "$SG" "docker logs o-core --since 1h 2>&1 | grep -iE 'lifecycle|扫拍|openLive' | tail -8 || true"

echo "== 锚④ SG 暴力认证 24h 计数（口径钉源=journalctl -u ssh；T-0167 基线 2840）=="
ssh "$SG" "journalctl -u ssh --since '24 hours ago' | grep -cE 'Failed|Invalid user|Connection closed by authenticating' || true"
echo "== 快照毕 @ $(date -Is) =="
