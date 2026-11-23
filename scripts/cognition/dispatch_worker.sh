#!/usr/bin/env bash
# dispatch_worker.sh —— 施工席触发拍（双车道件：工单道与信号道分离——
# 业主裁面 2026-11-15 钦定:唤知会话=信号/升件道+工单验收销号;
# 施工席=工单专办道,OPEN 桌认领→干→写回执,永不碰信号永不自销)。
# 阈面: dispatchable 工单在案(fin_0194 WIP窗——workable中按级/期限排前N
#   张=服务窗,窗内才可拣;窗外backlog在账不可拣,由唤知统管)即唤,
#   不等龄——票面即优先级;
#   单飞闸=o-worker-* 有 active/activating 即退(自有道glob——
#   不撞 o-cognition-* 唤知道锁,双道物理隔离);
#   跨机单飞=PG advisory lock(7972) 序列化+唤起在案行判据(同唤知径,
#   seat='施工机' 分账不混唤知锁面)。
# 用法: systemd timer 周期驱(o-worker.timer 建议 15min)或手跑。
set -u
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$DSN" ] || { echo "[dispatch-worker] FIN_PG_DSN 读不出($ENVF)——fail-closed不猜" >&2; exit 1; }

# 主机钉位（deploy/seat-hosts.tsv——业主令 2026-11-16 职能固定：错机唤起
# 物理拒唤+唤起败落账；表缺/行缺=放行向后兼容——钉位真源在表）。
PIN_ALLOW=$(awk -v u=o-worker 'BEGIN{FS="[ \t]+"} $1==u{print $2; exit}' \
	"$SCRIPT_DIR/../../deploy/seat-hosts.tsv" 2>/dev/null || true) # tolerate: 钉位表读败=放行向后兼容（表缺不锁座——钉位语义明文行缺/表缺放行）
if [ -n "$PIN_ALLOW" ]; then
	PIN_HOST=$(hostname 2>/dev/null || echo unknown)
	case ",$PIN_ALLOW," in
		*",$PIN_HOST,"*) ;;
		*) psql "$DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('施工机','唤起拒','','host=$PIN_HOST 主机钉位拒唤——pin=$PIN_ALLOW(deploy/seat-hosts.tsv)')" \
			>/dev/null 2>&1 || true # tolerate: 拒唤报账尽力——DSN 写败仅本机 stderr 留迹
		   # 工单#21291：钉位拒唤落「唤起拒」非「唤起败」（唤知dispatch.sh同
		   # 契）——败行虚增喂spawn_fail探阈值，源头词面归位优于下游过滤
		   echo "[dispatch-worker] 主机钉位拒唤 host=$PIN_HOST pin=$PIN_ALLOW" >&2
		   exit 1 ;;
	esac
fi

# 工单#26306③：亡 unit 收口壳兜底——每拍扫 8d 唤起面,果/败在案∨超
# RuntimeMaxSec(7200s+余量=3h)仍无同 unit「会话收口」行=亡前未写,
# 派工壳补簿记行(run.sh EXIT trap 第一保,本拍=第二保;seat 仍记
# 施工席——行=会话收口契约簿记非派工自产)。写败尽力不阻主径。
psql "$DSN" -qAt <<'SQL' >/dev/null 2>&1 || true # tolerate: 兜底扫=簿记尽力件——DSN/写败仅 stderr 留迹,不阻派工主径
INSERT INTO ops_event(seat,event,ticket,note)
SELECT '施工席','会话收口','',
  'unit='||u.unit||'——派工壳兜底收口:unit亡前会话未自写收口行——判词归唤起果败行+transcript'
FROM (SELECT substring(e.note from 'unit=(o-worker-[0-9]+)') AS unit, e.ts
      FROM ops_event e
      WHERE e.seat='施工机' AND e.event='唤起'
        AND e.ts>now()-interval '8 days') u
WHERE u.unit IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='会话收口'
    AND x.note ~ ('unit='||u.unit||'([^0-9A-Za-z_-]|$)'))
  AND (EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>u.ts
         AND x.seat='施工机' AND x.event IN ('唤起果','唤起败')
         AND x.note ~ ('unit='||u.unit||'([^0-9A-Za-z_-]|$)'))
    OR u.ts < now()-interval '3 hours');
SQL

# OPEN 票在案=唤(票态单点=ops_ticket_desk 视图,fin_0190 语义层根修——
# 销/挂起/接续/认领/级解析只活在那一份,此处不再复制正则面)。
# 工单#22578③/D-0048④:桌面窗只排「钱安全/钱数真伪/找钱」三类——机器看自己
# (自管类 slug)剔出施工道;剔除集=自观域词缀钦定单(钱/采集/部署域票不受影响,
# venue/mt5/cmc 接入属采集钱径不剔)。
TICKET_N=$(psql "$DSN" -qAt -c "
SELECT count(*) FROM ops_ticket_desk WHERE dispatchable AND tkind='工单'
AND note !~ '^工单/(dispatch|spawn|runbook|selfwatch|self-|stall|stuck|session|seat|pin-refuse|landq|queue|host-consol|close-finality|meta-|digest-|canon|kb-|org-|frame|probe|watch|beat|ticket-|worker-|cognition-|audit-|review-|lease|monologue|alert|feed-)'" 2>/dev/null)
TICKET_N=${TICKET_N:-0}
# backlog=窗外在账不可拣(fin_0194 WIP窗——施工道只见窗内,窗外淤积计数入每日一行（housekeeping_daily）;backlog只进触发note供账页观测)
BACKLOG_N=$(psql "$DSN" -qAt -c "
SELECT count(*) FROM ops_ticket_desk
WHERE workable AND NOT dispatchable AND tkind='工单'" 2>/dev/null)
BACKLOG_N=${BACKLOG_N:-0}
[ "$TICKET_N" -ge 1 ] || exit 0 # 零 OPEN 票=静默退(常态)

# 空扫簇退避（工单#23931②——dispatchable集≠可拣集错配出血收）：
# 2h唤起≥3拍而施工act零产(claim/receipt/suspend/resume/close皆算产)
# =连唤皆空扫,本拍退避不烧钱——真可拾票下一拍再验(退避非压制,
# 同簇连败阈同构)。
SPAWN_2H=$(psql "$DSN" -qAt -c "SELECT count(*) FROM ops_event
WHERE seat='施工机' AND event='唤起' AND ts>now()-interval '2 hours'" 2>/dev/null)
PROD_2H=$(psql "$DSN" -qAt -c "SELECT count(*) FROM ops_event_act
WHERE seat='施工机' AND act IN ('claim','receipt','suspend','resume','close')
  AND ts>now()-interval '2 hours'" 2>/dev/null)
if [ "${SPAWN_2H:-0}" -ge 3 ] && [ "${PROD_2H:-0}" -eq 0 ]; then
	echo "[dispatch-worker] 空扫簇退避: 2h唤起${SPAWN_2H}拍零产出——本拍退" >&2
	exit 0
fi

# 单飞闸: o-worker-*.service 有 active/activating 即退(施工道串行——
# 双会话抢同一工桌面=双认领双干,串行=天然去重;剔本dispatch unit防自锁)。
if systemctl --user list-units 'o-worker-*.service' --state=active,activating --no-legend --plain 2>/dev/null \
	| grep -vE 'o-worker-dispatch\.service' | grep -q .; then
	exit 0
fi

UNIT="o-worker-$(date +%s)"

# batchable=（工单O-34419观测段）：拾桌 top10 中 REVIEW 道可批数——
# 判据单点在 batchdesk.sh（会话拾桌同件复用，本件只计数不选票，
# 派发零选票权不动）；读败回落 0 不阻唤起。
BATCHABLE_N=$(bash "$SCRIPT_DIR/batchdesk.sh" --count 2>/dev/null)
BATCHABLE_N=${BATCHABLE_N:-0}

# 跨机单飞闸: advisory lock(7972) 序列化临界区,「30min 唤起在案且无同
# unit 唤起败/果∧无后随收口」=跨机在飞判据——seat='施工机' 分账面
# 与唤知机互不见(两道各自有锁);WRK- 心跳键同不入唤知在飞面。
WON=$(psql "$DSN" -qAt <<SQL
SELECT pg_advisory_lock(7972);
INSERT INTO ops_event(seat,event,ticket,note)
SELECT '施工机','唤起','','unit=$UNIT host=$(hostname) trigger=ticket_n$TICKET_N backlog=$BACKLOG_N batchable=$BATCHABLE_N'
WHERE NOT EXISTS (
	SELECT 1 FROM ops_event e
	WHERE e.seat='施工机' AND e.event='唤起' AND e.ts>now()-interval '30 minutes'
	  AND NOT EXISTS (SELECT 1 FROM ops_event x
	    WHERE x.ts>e.ts AND (
	      (x.seat='施工机' AND x.event IN ('唤起败','唤起果')
	        AND x.note ~ (substring(e.note from 'unit=[^ ]+')||'([^0-9A-Za-z_-]|\$)'))
	      OR x.event='会话收口' AND x.note ~ 'o-worker')))
AND NOT EXISTS (
	SELECT 1 FROM ops_headless h
	WHERE h.ticket ~ '^WRK-' AND h.state='alive'
	  AND h.last_activity > now() - interval '100 seconds')
RETURNING id;
SELECT pg_advisory_unlock(7972);
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
			VALUES('施工机','唤起败','','unit=$UNIT host=$(hostname) trigger=ticket_n$TICKET_N backlog=$BACKLOG_N batchable=$BATCHABLE_N spawn落败')" >/dev/null 2>&1
		echo "[dispatch-worker] systemd-run 唤起落败(败迹已清锁)" >&2; exit 1; }
echo "[dispatch-worker] 唤起 $UNIT ticket_n=$TICKET_N backlog=$BACKLOG_N batchable=$BATCHABLE_N"
