#!/usr/bin/env bash
# dispatch_worker.sh —— 施工席触发拍（双车道件：工单道与信号道分离——
# 业主裁面 2026-11-15 钦定:唤知会话=信号/升件道+工单验收销号;
# 施工席=工单专办道,OPEN 桌认领→干→写回执,永不碰信号永不自销)。
# 阈面: dispatchable 工单在案(fin_0194 WIP窗——workable中按级/期限排前N
#   张=服务窗,窗内才可拣;窗外backlog在账不可拣,由唤知统管)即唤,
#   不等龄——票面即优先级;
#   单飞闸=o-worker-* 有 active/activating 即退(自有道glob——
#   不撞 o-cognition-* 唤知道锁,双道物理隔离);
#   跨机单飞=PG advisory lock(7970) 序列化+唤起在案行判据(同唤知径,
#   seat='施工机' 分账不混唤知锁面)。
# 用法: systemd timer 周期驱(o-worker.timer 建议 15min)或手跑。
set -u
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$DSN" ] || { echo "[dispatch-worker] FIN_PG_DSN 读不出($ENVF)——fail-closed不猜" >&2; exit 1; }

# OPEN 票在案=唤(票态单点=ops_ticket_desk 视图,fin_0190 语义层根修——
# 销/挂起/接续/认领/级解析只活在那一份,此处不再复制正则面)。
TICKET_N=$(psql "$DSN" -qAt -c "
SELECT count(*) FROM ops_ticket_desk WHERE dispatchable AND tkind='工单'" 2>/dev/null)
TICKET_N=${TICKET_N:-0}
# backlog=窗外在账不可拣(fin_0194 WIP窗——施工道只见窗内,窗外淤积由
# queue_depth聚合卡报唤知统管;backlog只进触发note供账页观测)
BACKLOG_N=$(psql "$DSN" -qAt -c "
SELECT count(*) FROM ops_ticket_desk
WHERE workable AND NOT dispatchable AND tkind='工单'" 2>/dev/null)
BACKLOG_N=${BACKLOG_N:-0}
[ "$TICKET_N" -ge 1 ] || exit 0 # 零 OPEN 票=静默退(常态)

# 单飞闸: o-worker-*.service 有 active/activating 即退(施工道串行——
# 双会话抢同一工桌面=双认领双干,串行=天然去重;剔本dispatch unit防自锁)。
if systemctl --user list-units 'o-worker-*.service' --state=active,activating --no-legend --plain 2>/dev/null \
	| grep -vE 'o-worker-dispatch\.service' | grep -q .; then
	exit 0
fi

UNIT="o-worker-$(date +%s)"

# 跨机单飞闸: advisory lock(7970) 序列化临界区,「30min 唤起在案且无同
# unit 唤起败/果∧无后随收口」=跨机在飞判据——seat='施工机' 分账面
# 与唤知机互不见(两道各自有锁);WRK- 心跳键同不入唤知在飞面。
WON=$(psql "$DSN" -qAt <<SQL
SELECT pg_advisory_lock(7970);
INSERT INTO ops_event(seat,event,ticket,note)
SELECT '施工机','唤起','','unit=$UNIT host=$(hostname) trigger=ticket_n$TICKET_N backlog=$BACKLOG_N'
WHERE NOT EXISTS (
	SELECT 1 FROM ops_event e
	WHERE e.seat='施工机' AND e.event='唤起' AND e.ts>now()-interval '30 minutes'
	  AND NOT EXISTS (SELECT 1 FROM ops_event x
	    WHERE x.ts>e.ts AND (
	      (x.seat='施工机' AND x.event IN ('唤起败','唤起果')
	        AND x.note LIKE '%'||substring(e.note from 'unit=[^ ]+')||'%')
	      OR x.event='会话收口' AND x.note ~ 'o-worker')))
AND NOT EXISTS (
	SELECT 1 FROM ops_headless h
	WHERE h.ticket ~ '^WRK-' AND h.state='alive'
	  AND h.last_activity > now() - interval '100 seconds')
RETURNING id;
SELECT pg_advisory_unlock(7970);
SQL
)
WON=$(printf '%s\n' "$WON" | grep -oE '^[0-9]+$' | head -1)
[ -n "$WON" ] || exit 0 # 30min 内跨机施工唤起在案=在飞会话非本机辖——退

systemd-run --user --unit="$UNIT" --collect \
	--property=Description="施工席会话(ticket_n=$TICKET_N backlog=$BACKLOG_N)" \
	--property=RuntimeMaxSec=7200 \
	--setenv=COG_UNIT="$UNIT" \
	--setenv=COG_ROLE=worker \
	--setenv=COG_RUN_MODEL=swe-2-flash-high \
	--setenv=COG_RUNBOOK="$SCRIPT_DIR/worker_runbook.md" \
	"$SCRIPT_DIR/run.sh" || {
		psql "$DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('施工机','唤起败','','unit=$UNIT host=$(hostname) trigger=ticket_n$TICKET_N backlog=$BACKLOG_N spawn落败')" >/dev/null 2>&1
		echo "[dispatch-worker] systemd-run 唤起落败(败迹已清锁)" >&2; exit 1; }
echo "[dispatch-worker] 唤起 $UNIT ticket_n=$TICKET_N backlog=$BACKLOG_N"
