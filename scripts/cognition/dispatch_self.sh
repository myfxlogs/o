#!/usr/bin/env bash
# dispatch_self.sh —— 自我席(CEO)唤起拍（组织表件·业主令 2026-11-15）。
# 公司唯一「我」:6h 慢拍恒唤——无条件触发(CEO 对钱与承诺负责,
# 非事件驱动件)。单飞: o-self-* 在飞即退(慢判断不堆叠);跨机:
# PG advisory lock 7971+自我机 seat 唤起行 30min 在飞判(与唤知7969/
# 施工7970 分域——三锁三账)。
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$DSN" ] || { echo "[dispatch-self] FIN_PG_DSN 读不出($ENVF)——fail-closed不猜" >&2; exit 1; }

# 单飞闸: o-self-*.service 有 active/activating 即退(自我道串行——
# 一拍一我,不叠判断);必剔 o-self-dispatch.service 自身——派发器
# 名字同 glob,不剔=永远自锁唤不起(施工闸同坑已修,抄全)。
if systemctl --user list-units 'o-self-*.service' --state=active,activating \
	--no-legend --plain 2>/dev/null \
	| grep -vE 'o-self-dispatch\.service' | grep -q .; then
	echo "[dispatch-self] o-self 在飞——自我道串行退出"; exit 0
fi
if ! systemctl --user is-enabled o-self.timer >/dev/null 2>&1; then
	exit 0 # timer 未立=本机非唤起机(部署分工判据同 dispatch.sh)
fi

UNIT="o-self-$(date +%s)"

# 唤醒判据(30min快拍): 保底拍=距上次唤起≥6h必醒;早唤=要事触发——
# ①P1新票立案 ②P1票被挂起还票(裁定被退=死锁形) ③新仓活水
# (fin_cashflow新position_ref) ④窗内现金净动>$250(钱面突变)。
# roster_rw界:fin_position/fin_ledger_snapshot皆越权不可读(实证
# 判词④同坑——读面全走ops_event+fin_cashflow)。
# 无事=退零成本;查询败=不唤fail-closed。
LAST_WAKE=$(psql "$DSN" -qAt -c "SELECT COALESCE(max(ts),'epoch'::timestamptz)
	FROM ops_event WHERE seat='自我机' AND event='唤起'" 2>/dev/null)
[ -n "$LAST_WAKE" ] || exit 0
TRIGGER=$(psql "$DSN" -qAt <<SQL
SELECT CASE
  WHEN now() - '$LAST_WAKE'::timestamptz >= interval '6 hours' THEN 'baseline'
  WHEN EXISTS (SELECT 1 FROM ops_event WHERE ts>'$LAST_WAKE'::timestamptz
    AND event='工单' AND note ~ '级=P1') THEN 'early:p1_ticket'
  WHEN EXISTS (SELECT 1 FROM ops_event WHERE ts>'$LAST_WAKE'::timestamptz
    AND event='处置' AND note LIKE '%挂起条件=%' AND note ~ '级=P1')
    THEN 'early:p1_rebound'
  WHEN EXISTS (SELECT 1 FROM fin_cashflow WHERE ts>'$LAST_WAKE'::timestamptz
    AND position_ref IS NOT NULL) THEN 'early:new_position'
  WHEN (SELECT abs(COALESCE(sum(amount),0)) FROM fin_cashflow
    WHERE ts>'$LAST_WAKE'::timestamptz) > 250 THEN 'early:cashflow_burst'
  ELSE 'quiet' END;
SQL
)
[ "$TRIGGER" = "quiet" ] && exit 0
[ -z "$TRIGGER" ] && exit 0 # 查询败=不唤,fail-closed

# 唤起单写径: 7969 唤知/7970 施工/7971 自我——三锁三账。
# 条件插入免 DELETE(append-only 铁律):「30min 唤起在案且无唤起败/果/
# 收口收尾∨SELF- 心跳鲜活」=跨机在飞判据——seat='自我机' 分账。
WON=$(psql "$DSN" -qAt <<SQL
SELECT pg_advisory_lock(7971);
INSERT INTO ops_event(seat,event,ticket,note)
SELECT '自我机','唤起','','unit=$UNIT host=$(hostname) trigger=$TRIGGER'
WHERE NOT EXISTS (
	SELECT 1 FROM ops_event e
	WHERE e.seat='自我机' AND e.event='唤起' AND e.ts>now()-interval '30 minutes'
	  AND NOT EXISTS (SELECT 1 FROM ops_event x
	    WHERE x.ts>e.ts AND (
	      (x.seat='自我机' AND x.event IN ('唤起败','唤起果')
	        AND x.note LIKE '%'||substring(e.note from 'unit=[^ ]+')||'%')
	      OR x.event='会话收口' AND x.note ~ 'o-self')))
AND NOT EXISTS (
	SELECT 1 FROM ops_headless h
	WHERE h.ticket ~ '^SELF-' AND h.state='alive'
	  AND h.last_activity > now() - interval '100 seconds')
RETURNING id;
SELECT pg_advisory_unlock(7971);
SQL
)
WON=$(printf '%s\n' "$WON" | grep -oE '^[0-9]+$' | head -1)
[ -n "$WON" ] || exit 0 # 30min 内跨机自我唤起在案=在飞会话非本机辖——退

systemd-run --user --unit="$UNIT" --collect \
	--property=Description="自我席会话(CEO慢拍)" \
	--property=RuntimeMaxSec=7200 \
	--setenv=COG_UNIT="$UNIT" \
	--setenv=COG_ROLE=self \
	--setenv=COG_RUN_MODEL=swe-2-max \
	--setenv=COG_RUNBOOK="$SCRIPT_DIR/self_runbook.md" \
	"$SCRIPT_DIR/run.sh" || {
		psql "$DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('自我机','唤起败','','unit=$UNIT host=$(hostname) spawn落败')" >/dev/null 2>&1
		echo "[dispatch-self] systemd-run 唤起落败(败迹已清锁)" >&2; exit 1; }
echo "[dispatch-self] 唤起 unit=$UNIT"
