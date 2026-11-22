#!/usr/bin/env bash
# dispatch_self.sh —— 自我席(CEO)唤起拍（组织表件·业主令 2026-11-15）。
# 公司唯一「我」:6h 慢拍恒唤——无条件触发(CEO 对钱与承诺负责,
# 非事件驱动件)。单飞: o-self-* 在飞即退(慢判断不堆叠);跨机:
# PG advisory lock 7971+自我机 seat 唤起行 30min 在飞判(与唤知7969/
# 审计7970/施工7972 分域——四锁四账)。
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$DSN" ] || { echo "[dispatch-self] FIN_PG_DSN 读不出($ENVF)——fail-closed不猜" >&2; exit 1; }

# 主机钉位（deploy/seat-hosts.tsv——业主令 2026-11-16 职能固定：错机唤起
# 物理拒唤+唤起败落账；表缺/行缺=放行向后兼容——钉位真源在表）。
PIN_ALLOW=$(awk -v u=o-self 'BEGIN{FS="[ \t]+"} $1==u{print $2; exit}' \
	"$SCRIPT_DIR/../../deploy/seat-hosts.tsv" 2>/dev/null || true) # tolerate: 钉位表读败=放行向后兼容（表缺不锁座——钉位语义明文行缺/表缺放行）
if [ -n "$PIN_ALLOW" ]; then
	PIN_HOST=$(hostname 2>/dev/null || echo unknown)
	case ",$PIN_ALLOW," in
		*",$PIN_HOST,"*) ;;
		*) psql "$DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('自我机','唤起拒','','host=$PIN_HOST 主机钉位拒唤——pin=$PIN_ALLOW(deploy/seat-hosts.tsv)')" \
			>/dev/null 2>&1 || true # tolerate: 拒唤报账尽力——DSN 写败仅本机 stderr 留迹
		   echo "[dispatch-self] 主机钉位拒唤 host=$PIN_HOST pin=$PIN_ALLOW" >&2
		   exit 0 ;;
	esac
fi

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
# ⓪活敞口跟进钟(2026-11-17 董事会判——仓#86裸空4h30m,自我席21:08
# 知情后2h33m未醒=无跟进):fin_guard_close_status 有 state='closing'
# 且 attempts>=2 在册行=仓该平而平不掉(守卫连试连败)→每拍必唤直
# 至实平。精确谓词在册(工单#22248②——roster_rw 窄视图可读;旧代
# 理=守卫环升件近窗,实平后仍尾唤一窗+非卡态升件误唤双偏皆消)。
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
  WHEN EXISTS (SELECT 1 FROM fin_guard_close_status
    WHERE state='closing' AND attempts >= 2)
    THEN 'early:live_incident'
  WHEN now() - '$LAST_WAKE'::timestamptz >= interval '6 hours' THEN 'baseline'
  WHEN EXISTS (SELECT 1 FROM ops_event WHERE ts>'$LAST_WAKE'::timestamptz
    AND event='工单' AND note ~ '级=P1') THEN 'early:p1_ticket'
  WHEN EXISTS (SELECT 1 FROM ops_event WHERE ts>'$LAST_WAKE'::timestamptz
    AND event='处置' AND note LIKE '%挂起条件=%' AND note ~ '级=P1')
    THEN 'early:p1_rebound'
  WHEN (SELECT count(*) FROM ops_ticket_desk WHERE rulable) >= ${ADJ_DEBT_N:-4}
    OR (SELECT max(now()-susp_since) FROM ops_ticket_desk WHERE rulable)
       > interval '${ADJ_DEBT_AGE:-48} hours'
    THEN 'early:adjudication_debt'
  WHEN EXISTS (SELECT 1 FROM fin_cashflow WHERE ts>'$LAST_WAKE'::timestamptz
    AND position_ref IS NOT NULL) THEN 'early:new_position'
  WHEN (SELECT abs(COALESCE(sum(amount),0)) FROM fin_cashflow
    WHERE ts>'$LAST_WAKE'::timestamptz) > 250 THEN 'early:cashflow_burst'
  ELSE 'quiet' END;
SQL
)
[ "$TRIGGER" = "quiet" ] && exit 0
[ -z "$TRIGGER" ] && exit 0 # 查询败=不唤,fail-closed

# 唤起单写径: 7969 唤知/7970 审计/7971 自我/7972 施工——四锁四账。
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
	        AND x.note ~ (substring(e.note from 'unit=[^ ]+')||'([^0-9A-Za-z_-]|\$)'))
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
