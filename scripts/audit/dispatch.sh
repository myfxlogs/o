#!/usr/bin/env bash
# dispatch.sh —— 审计机触发拍（审计臂唤器——脱自 cognition/dispatch.sh
# 同构：域轮换唤审计会话,不看未决升件面不跑矿工）。
# 语义:scopes.tsv 域册逐域查「唤过否」→NULL 优先（唤过域选最陈者）→
#   systemd-run 单飞审计会话（COG_ROLE=audit——同 run.sh 壳,角色分账）。
# 域择: 每域 max(ts) seat='审计机' event='唤起' note LIKE 'unit=% scope=<name> %'
#   ——NULL=从未唤过优先（文件序首个）,全非 NULL 选最陈者（轮转公平）。
# 单飞闸=o-audit-* 有 active/activating 即退（剔自身 dispatch.service——
#   闸面自锁死实坑同 cognition 首拍）；
#   跨机闸=PG advisory lock(7970)+「30min 唤起在案且无同 unit 唤起败」；
#   静默退出=单飞中/跨机在飞（域册缺席=fail-closed exit 2 非静默）。
# 用法: systemd timer 周期驱（o-audit.timer 03:30 日拍）或手跑。
set -u
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO=$(cd -- "$SCRIPT_DIR/../.." && pwd)
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$DSN" ] || { echo "[audit-dispatch] FIN_PG_DSN 读不出($ENVF)——fail-closed不猜" >&2; exit 1; }

SCOPES_FILE="${AUDIT_SCOPES_FILE:-$SCRIPT_DIR/scopes.tsv}"
[ -s "$SCOPES_FILE" ] || { echo "[audit-dispatch] 域册缺席: $SCOPES_FILE——fail-closed" >&2; exit 2; }
RUNBOOK="${AUDIT_RUNBOOK:-$SCRIPT_DIR/runbook.md}"
[ -s "$RUNBOOK" ] || { echo "[audit-dispatch] runbook 缺: $RUNBOOK" >&2; exit 2; }

# 域册读行（name<TAB>paths<TAB>recall_words；# 注/空行跳）。逐域查唤起史：
# NULL=未唤过→文件序首个即中；全非 NULL→max(ts) 最小者（最陈轮换）。
SCOPE=""; PATHS=""; WORDS=""
BEST_TS="" # 已唤域中最陈 max(ts)（epoch；空=未遇已唤域）
while IFS=$'\t' read -r name paths recall_words _rest; do
	case "$name" in ''|\#*) continue ;; esac
	last=$(psql "$DSN" -qAt -c "SELECT COALESCE(extract(epoch from max(ts))::bigint,0)
		FROM ops_event WHERE seat='审计机' AND event='唤起'
		AND note LIKE 'unit=% scope=$name %'" 2>/dev/null)
	last=${last:-0}
	if [ "$last" = "0" ]; then
		SCOPE=$name; PATHS=$paths; WORDS=$recall_words; break
	fi
	if [ -z "$BEST_TS" ] || [ "$last" -lt "$BEST_TS" ]; then
		BEST_TS=$last; SCOPE=$name; PATHS=$paths; WORDS=$recall_words
	fi
done < "$SCOPES_FILE"
[ -n "$SCOPE" ] || { echo "[audit-dispatch] 域册零有效行——fail-closed" >&2; exit 2; }

# 单飞闸: o-audit-<epoch> 会话件 active 即退（剔自身 dispatch unit——
# o-audit-* glob 罩住 dispatcher,不剔=自锁死）。
if systemctl --user list-units 'o-audit-*' --state=active,activating --no-legend --plain 2>/dev/null \
	| grep -vE 'o-audit-dispatch\.service' | grep -q .; then
	exit 0
fi

UNIT="o-audit-$(date +%s)"

# 跨机单飞闸（cognition 7969 同构——审计臂锁键 7970）：PG advisory lock
# 序列化临界区,「30min 唤起在案且无同 unit 唤起败」为跨机在飞判据。
# note 尾随一空格——域择 LIKE 'unit=% scope=<name> %' 靠尾空格钉名边界。
WON=$(psql "$DSN" -qAt <<SQL
SELECT pg_advisory_lock(7970);
INSERT INTO ops_event(seat,event,ticket,note)
SELECT '审计机','唤起','','unit=$UNIT host=$(hostname) scope=$SCOPE '
WHERE NOT EXISTS (
	SELECT 1 FROM ops_event e
	WHERE e.seat='审计机' AND e.event='唤起' AND e.ts>now()-interval '30 minutes'
	  AND NOT EXISTS (SELECT 1 FROM ops_event x
	    WHERE x.seat='审计机' AND x.event IN ('唤起败','唤起果') AND x.ts>e.ts
	      AND x.note LIKE '%'||substring(e.note from 'unit=[^ ]+')||'%'))
RETURNING id;
SELECT pg_advisory_unlock(7970);
SQL
)
WON=$(printf '%s\n' "$WON" | grep -oE '^[0-9]+$' | head -1)
[ -n "$WON" ] || exit 0 # 30min 内跨机唤起在案=在飞会话非本机辖——退
systemd-run --user --unit="$UNIT" --collect \
	--property=Description="审计会话(scope=$SCOPE)" \
	--property=RuntimeMaxSec=5400 \
	--setenv=COG_UNIT="$UNIT" \
	--setenv=COG_ROLE=audit \
	--setenv=COG_SCOPE="$SCOPE|$PATHS" \
	--setenv=COG_RECALL_WORDS="$WORDS" \
	--setenv=COG_RUNBOOK="$RUNBOOK" \
	"$REPO/scripts/cognition/run.sh" || {
		psql "$DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('审计机','唤起败','','unit=$UNIT host=$(hostname) scope=$SCOPE spawn落败')" >/dev/null 2>&1
		echo "[audit-dispatch] systemd-run 唤起落败(败迹已清锁)" >&2; exit 1; }
echo "[audit-dispatch] 唤起 $UNIT scope=$SCOPE (esc#$WON)"
