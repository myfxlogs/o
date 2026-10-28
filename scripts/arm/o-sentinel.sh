#!/usr/bin/env bash
# o-sentinel —— ARM 哨兵（T-0173 业主令：监控哨兵落 ARM——拨测全域存活，
# 红灯落 /var/log/o-sentinel/alerts.log）。轻量纯 bash，1G 内存无压力。
# 探针：①公网域名链（CF→SG o-core）②本地 PG-md（行情域）③隧道 PG-fin
# 15433 ④readcred 8095 ⑤本地 docker 守护。
set -u
LOG=/var/log/o-sentinel/alerts.log
mkdir -p /var/log/o-sentinel && chmod 755 /var/log/o-sentinel
now() { date -u +%FT%TZ; }
fail() { echo "[$(now)] FAIL $1" >> "$LOG"; }

# ① 公网域名链（DNS+CF+SG o-core 全链）
curl -so /dev/null --max-time 15 https://o.myfxlogs.org/ || fail "o.myfxlogs.org 不可达（CF/SG 链断）"
# ② 本地 PG-md（行情域库）
docker exec o-postgres-md pg_isready -h 127.0.0.1 -p 5432 -q -U o -d o || fail "PG-md 未就绪（容器异常）"
# ③ 隧道 PG-fin 15433
pg_isready -h 127.0.0.1 -p 15433 -q -U o -d o 2>/dev/null || fail "PG-fin 隧道 15433 断（o-sg-tunnel 掉线）"
# ④ readcred 8095（SG 凭据专口）
curl -so /dev/null --max-time 10 http://127.0.0.1:8095/healthz 2>/dev/null || curl -so /dev/null --max-time 10 http://127.0.0.1:8095/ 2>/dev/null || fail "readcred 8095 不可达（SG 侧凭据口断）"
# ⑤ docker 守护
docker info >/dev/null 2>&1 || fail "docker 守护异常"
