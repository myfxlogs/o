#!/usr/bin/env bash
# dispatch_research.sh —— 研究席触发拍（工单#24412：四触发钦定唤研究席
# 会话——research-intake-spec 立环而环内无派发件,升件到唤知手里=触发
# 链断在派发层,11-11 源普查后 7d 零登记实证）。
# 阈面（研究员 SKILL.md 钦定四触发,仅事件唤起禁定期节拍）:
#   ① ops_esc_pending intake_dry∨breed_dry 升件未决——进水口干涸/
#     进化环停摆=源普查循环唤起(esc#<id> 记触发源,销后重火=新id新唤);
#   ② ops_ticket_desk OPEN 票 note ~ '^(求知[:：]|研究席[:：])'——求知票外拓
#     调研/业主手注研究席事件(venue新品等机检缺席面的人工注入口,
#     SKILL「决策席/CEO可手动代发同类事件」钦定)(tk#<id>记源);
#   ③ fin_venue_candidate state='recommended'——T-0300 达阈=注册
#     清单起草(venue=<名>记源,30d重发闸后新请求=新唤)。
#   去重=唤起行载源键(esc#/tk#/venue=)——同键已唤不重唤,败重唤
#     (NOT EXISTS 只认 '唤起' 行,唤起败不吞源键);
#   空扫簇退避=2h唤起≥3拍而研究席产出零行(同工人道同构——连唤
#     皆空扫=烧钱,真触发下一拍再验);
#   单飞闸=o-research-* 有 active/activating 即退(研究道串行);
#   跨机单飞=PG advisory lock(7973)+唤起在案行判据(同唤知/施工径,
#   seat='研究机' 分账——7973 与 7969唤知/7971自我/7972施工 不撞锁面)。
# 用法: systemd timer 周期驱(o-research.timer 建议 15min)或手跑;
#   RES_DRY=1=干跑(探针照跑唤起前退出打触发判定,不落行不 spawn)。
set -u
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$DSN" ] || { echo "[dispatch-research] FIN_PG_DSN 读不出($ENVF)——fail-closed不猜" >&2; exit 1; }

# 主机钉位（deploy/seat-hosts.tsv——业主令 2026-11-16 职能固定：错机唤起
# 物理拒唤+唤起败落账；表缺/行缺=放行向后兼容——钉位真源在表）。
PIN_ALLOW=$(awk -v u=o-research 'BEGIN{FS="[ \t]+"} $1==u{print $2; exit}' \
	"$SCRIPT_DIR/../../deploy/seat-hosts.tsv" 2>/dev/null || true) # tolerate: 钉位表读败=放行向后兼容（表缺不锁座——钉位语义明文行缺/表缺放行）
if [ -n "$PIN_ALLOW" ]; then
	PIN_HOST=$(hostname 2>/dev/null || echo unknown)
	case ",$PIN_ALLOW," in
		*",$PIN_HOST,"*) ;;
		*) psql "$DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('研究机','唤起败','','host=$PIN_HOST 主机钉位拒唤——pin=$PIN_ALLOW(deploy/seat-hosts.tsv)')" \
			>/dev/null 2>&1 || true # tolerate: 拒唤报账尽力——DSN 写败仅本机 stderr 留迹
		   echo "[dispatch-research] 主机钉位拒唤 host=$PIN_HOST pin=$PIN_ALLOW" >&2
		   exit 1 ;;
	esac
fi

# 触发扫（源键去重——唤起行载 esc#/tk#/venue= 键,同键已唤不重唤;
# NOT EXISTS 只认 '唤起' 行=唤起败不吞键可重唤;源键集拼 trigger note）。
TRIGGERS=$(psql "$DSN" -qAt <<'SQL'
SELECT 'esc#'||e.id||':'||e.esc_scope FROM ops_esc_pending e
WHERE e.esc_scope IN ('intake_dry','breed_dry')
AND NOT EXISTS (SELECT 1 FROM ops_event x
	WHERE x.seat='研究机' AND x.event='唤起'
	AND x.note ~ ('esc#'||e.id::text||'([^0-9]|$)'))
UNION ALL
SELECT 'tk#'||t.id||':'||left(t.note,40) FROM ops_ticket_desk t
WHERE t.closed IS NOT TRUE AND t.note ~ '^(求知[:：]|研究席[:：])'
AND NOT EXISTS (SELECT 1 FROM ops_event x
	WHERE x.seat='研究机' AND x.event='唤起'
	AND x.note ~ ('tk#'||t.id::text||'([^0-9]|$)'))
UNION ALL
SELECT 'venue='||c.venue||':recommended' FROM fin_venue_candidate c
WHERE c.state='recommended'
AND NOT EXISTS (SELECT 1 FROM ops_event x
	WHERE x.seat='研究机' AND x.event='唤起'
	AND x.ts > c.last_request_at AND x.note ~ ('venue='||c.venue||'([^a-zA-Z0-9._-]|$)'))
SQL
)
TRIGGERS=$(printf '%s\n' "$TRIGGERS" | grep -v '^$' | paste -sd' ' -)
[ -n "$TRIGGERS" ] || { [ "${RES_DRY:-}" = 1 ] && echo "[dispatch-research] dry: trigger=none"; exit 0; }
[ "${RES_DRY:-}" = 1 ] && { echo "[dispatch-research] dry: trigger=$TRIGGERS"; exit 0; }

# 空扫簇退避（同施工道工单#23931②同构——2h唤起≥3拍而研究席产出
# 零行=连唤皆空扫,本拍退避不烧钱;产出面=seat 研究席/研究员 行
# 含交付/登记/收口）。退避非压制——真触发下一拍再验。
SPAWN_2H=$(psql "$DSN" -qAt -c "SELECT count(*) FROM ops_event
WHERE seat='研究机' AND event='唤起' AND ts>now()-interval '2 hours'" 2>/dev/null)
PROD_2H=$(psql "$DSN" -qAt -c "SELECT count(*) FROM ops_event
WHERE seat IN ('研究席','研究员') AND ts>now()-interval '2 hours'" 2>/dev/null)
if [ "${SPAWN_2H:-0}" -ge 3 ] && [ "${PROD_2H:-0}" -eq 0 ]; then
	echo "[dispatch-research] 空扫簇退避: 2h唤起${SPAWN_2H}拍零产出——本拍退" >&2
	exit 0
fi

# 单飞闸: o-research-*.service 有 active/activating 即退(研究道串行——
# 双会话抢同一题=双起草双账,串行=天然去重;剔本dispatch unit防自锁)。
if systemctl --user list-units 'o-research-*.service' --state=active,activating --no-legend --plain 2>/dev/null \
	| grep -vE 'o-research-dispatch\.service' | grep -q .; then
	exit 0
fi

UNIT="o-research-$(date +%s)"

# 跨机单飞闸: advisory lock(7973) 序列化临界区,「30min 唤起在案且无同
# unit 唤起败/果∧无后随收口」=跨机在飞判据——seat='研究机' 分账面
# 与唤知/施工互不见(各道各自有锁);RES- 心跳键同不入别道在飞面。
WON=$(psql "$DSN" -qAt <<SQL
SELECT pg_advisory_lock(7973);
INSERT INTO ops_event(seat,event,ticket,note)
SELECT '研究机','唤起','','unit=$UNIT host=$(hostname) trigger=$TRIGGERS'
WHERE NOT EXISTS (
	SELECT 1 FROM ops_event e
	WHERE e.seat='研究机' AND e.event='唤起' AND e.ts>now()-interval '30 minutes'
	  AND NOT EXISTS (SELECT 1 FROM ops_event x
	    WHERE x.ts>e.ts AND (
	      (x.seat='研究机' AND x.event IN ('唤起败','唤起果')
	        AND x.note LIKE '%'||substring(e.note from 'unit=[^ ]+')||'%')
	      OR x.event='会话收口' AND x.note ~ 'o-research')))
AND NOT EXISTS (
	SELECT 1 FROM ops_headless h
	WHERE h.ticket ~ '^RES-' AND h.state='alive'
	  AND h.last_activity > now() - interval '100 seconds')
RETURNING id;
SELECT pg_advisory_unlock(7973);
SQL
)
WON=$(printf '%s\n' "$WON" | grep -oE '^[0-9]+$' | head -1)
[ -n "$WON" ] || exit 0 # 30min 内跨机研究唤起在案=在飞会话非本机辖——退

systemd-run --user --unit="$UNIT" --collect \
	--property=Description="研究席会话(trigger=$TRIGGERS)" \
	--property=RuntimeMaxSec=7200 \
	--setenv=COG_UNIT="$UNIT" \
	--setenv=COG_ROLE=research \
	--setenv=COG_RUNBOOK="$SCRIPT_DIR/research_runbook.md" \
	"$SCRIPT_DIR/run.sh" || {
		psql "$DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('研究机','唤起败','','unit=$UNIT host=$(hostname) trigger=$TRIGGERS spawn落败')" >/dev/null 2>&1
		echo "[dispatch-research] systemd-run 唤起落败(败迹已清锁)" >&2; exit 1; }
echo "[dispatch-research] 唤起 $UNIT trigger=$TRIGGERS"
