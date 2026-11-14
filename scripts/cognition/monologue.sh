#!/usr/bin/env bash
# monologue.sh —— 独白器官拍（o-cognition-monologue.service 载件;棒2「常驻
# 内心独白」): 内在线程=常驻节奏非信号驱动——定时醒,续独白链写一行念头,
# 不干活(干活归唤知会话;想=线程不是任务)。
# 单飞三闸: ①本机 systemd 面 o-cognition-*.service 在飞(剔自身件)即退——想不抢
#   饭碗,工作会话优先; ②跨机 PG 锁=让位判据收窄为「真在飞」:唤起行
#   5min 内未配对(剔自 trigger=monologue——只盖 spawn→心跳间隙)
#   ∨ ops_headless COG|MONO 心跳鲜活(运行中工作+前独白尸);
#   ③胜方插唤起行(trigger=monologue)→run.sh 释锁径(唤起果/败)接管。
# 静默退出=认知面真在飞(非常态——判据收窄后饿死缝消)。
set -u
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$DSN" ] || { echo "[monologue] FIN_PG_DSN 读不出($ENVF)——fail-closed不猜" >&2; exit 1; }

# 单飞闸①: 本机 o-cognition-*.service 在飞即退(剔自身件——服务态本件已在
# glob 下;*.service 收窄=防 o-cognition-*.timer 恒 active 混入自锁,2026-11-14②)。
if systemctl --user list-units 'o-cognition-*.service' --state=active,activating --no-legend --plain 2>/dev/null \
	| grep -vE 'o-cognition-monologue\.service' | grep -q .; then
	exit 0
fi

# 单飞闸②+锁: 跨机在飞判据同 dispatch(PG advisory lock 7969 序列化临界区;
# unit 定名非时间戳——唤起/果败配对键恒定=独白线程单轨)。
UNIT="o-cognition-monologue"
WON=$(psql "$DSN" -qAt <<SQL
SELECT pg_advisory_lock(7969);
INSERT INTO ops_event(seat,event,ticket,note)
SELECT '唤知机','唤起','','unit=$UNIT host=$(hostname) trigger=monologue'
WHERE NOT EXISTS (
	-- 独白槽位令(2026-11-14):唤起窗 30min→5min——只拦刚 spawn 未心跳
	-- 的工作会话(唤起→心跳间隙=秒级,5min 覆盖 spawn 败迹);已心跳
	-- 的由下行 ops_headless 判。原 30min 窗+唤起流每~16min=独白永让位
	-- 零产饿死(26h 实证)。剔 trigger=monologue 行=陈独白尸不拦己。
	SELECT 1 FROM ops_event e
	WHERE e.seat='唤知机' AND e.event='唤起' AND e.ts>now()-interval '5 minutes'
	  AND e.note NOT LIKE '%trigger=monologue%'
	  AND NOT EXISTS (SELECT 1 FROM ops_event x
	    WHERE x.ts>e.ts AND (
	      (x.seat='唤知机' AND x.event IN ('唤起败','唤起果')
	        AND x.note LIKE '%'||substring(e.note from 'unit=[^ ]+')||'%')
	      OR x.event='会话收口')))
AND NOT EXISTS (
	SELECT 1 FROM ops_headless h
	WHERE h.ticket ~ '^(COG|MONO)-' AND h.state='alive'
	  AND h.last_activity > now() - interval '100 seconds')
RETURNING id;
SELECT pg_advisory_unlock(7969);
SQL
)
WON=$(printf '%s\n' "$WON" | grep -oE '^[0-9]+$' | head -1)
[ -n "$WON" ] || exit 0 # 唤起在案=认知面在飞——本拍让位

# 独白档会话——role/runbook/model 本件钉死(临时 unit env 不继承 drop-in
# 实坑先例:依赖注入 env=换机哑火;env 覆写径仍留给测试/调优)。
export COG_ROLE=monologue
export COG_RUNBOOK="${COG_MONO_RUNBOOK:-$SCRIPT_DIR/monologue.md}"
export COG_RUN_MODEL="${COG_MONO_MODEL:-swe-2-flash-high}"
export COG_UNIT="$UNIT"
exec "$SCRIPT_DIR/run.sh"
