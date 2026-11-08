#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin"

cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
query=${*: -1}
[[ $query == "-" ]] && query=$(cat)
case "$query" in
	"-qAt") echo t ;; # T-0354 advisory-lock holder 桩（无 -c=锁会话形，答 t 放行）
	*"ops_ticket_phase"*) exit 1 ;; # GW-001 段B1：VIEW 未迁移=legacy 桥面
	*"SELECT count(*)"*) echo "$DM_PENDING_COUNT" ;;
	*"SELECT l.ticket FROM ("*"COALESCE"*) ;;
	*"SELECT l.ticket FROM ("*) printf '%s\n' "$DM_PENDING_TICKETS" ;;
	*"WITH lock AS MATERIALIZED"*)
		grep -Fq "source='devin-cli'" <<<"$query"
		[[ ${DM_USAGE_FAIL:-0} != 1 ]] || exit 1
		printf '%s\n%s\n' "$*" "$query" >> "$USAGE_INSERTS"
		echo inserted
		;;
	*"SELECT 1 FROM ops_event WHERE event='升件'"*)
		# T-0369 同窗去重——升件落账即去重凭据（账行在=窗内已报）
		[ -s "$ESC_INSERTS" ] && echo 1 ;;
	*"DISTINCT ON (ticket)"*) ;; # T-0372：dm_suspend 扫描桩——无挂起行空集（看门狗不误报）
	*"l.ticket ||"*) ;; # T-0372：派单查询桩——空集（旧桩落 *) exit2=静默败被管道吞的活标本）
	*"INSERT INTO ops_event"*) printf '%s\n' "$query" >> "$ESC_INSERTS" ;;
	*"INSERT INTO ops_headless"*) ;;
	*) echo "unexpected psql query: $query" >&2; exit 2 ;;
esac
EOF
cat > "$TMP/bin/depgate" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
cat > "$TMP/bin/go" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
cat > "$TMP/bin/devin" <<'EOF'
#!/usr/bin/env bash
export_file=
while (($#)); do
	if [[ $1 == --export ]]; then shift; export_file=${1:-}; break; fi
	shift
done
[[ -n "$export_file" ]]
python3 - "$export_file" <<'PY'
import json,sys
with open(sys.argv[1],'w') as f:
 json.dump({'session_id':'decision-session','agent':{'model_name':'SWE-2 Max'},
  'final_metrics':{'total_prompt_tokens':90,'total_completion_tokens':8,'total_cached_tokens':30}},f)
PY
echo decision-call >> "$DECISION_CALLS"
EOF
chmod +x "$TMP/bin/psql" "$TMP/bin/devin" "$TMP/bin/depgate" "$TMP/bin/go"
printf 'FIN_PG_DSN=fake-dsn\n' > "$TMP/env"
export PATH="$TMP/bin:$PATH"
export DM_ENV_FILE="$TMP/env"
export O_CTO_WORKDIR="$TMP/decision" # dm_tick 班次目=O_CTO_WORKDIR（原测 O_DECISION_WORKDIR 系失配死变量——usage.* 断言恒空）
mkdir -p "$O_CTO_WORKDIR"
export O_MECH_SYSTEMCTL=/bin/true # GW-001 A2：尾段唤 o-mech 桩掉——零真 systemctl 调用
export O_DECISION_DEVIN_BIN="$TMP/bin/devin"
export O_DM_DEPGATE_SH="$TMP/bin/depgate"
export O_WORKER_USAGE_BIN="$ROOT/scripts/meta/ticket_usage.sh"
export O_DM_LEDGER_SH=/bin/true # T-0414：机械轨件单测——tick 接线面桩掉防真 git/psql
export DECISION_CALLS="$TMP/decision-calls"
export USAGE_INSERTS="$TMP/usage-inserts"
export ESC_INSERTS="$TMP/esc-inserts"
export ESCALATE_ENV_FILE="$TMP/env"

export DM_PENDING_COUNT=0 DM_PENDING_TICKETS=
bash "$ROOT/scripts/meta/dm_tick.sh" >/dev/null
[[ ! -e "$DECISION_CALLS" && ! -e "$USAGE_INSERTS" ]]

# One ticket keeps direct ticket attribution; a multi-ticket model batch is
# recorded once under META-DM with its explicit ticket set, never duplicated.
export DM_PENDING_COUNT=1 DM_PENDING_TICKETS=T-9001
bash "$ROOT/scripts/meta/dm_tick.sh" >/dev/null
grep -q -- "-v ticket=T-9001 -v kind=audit -v model=SWE-2 Max" "$USAGE_INSERTS"
grep -q 'note=session=decision-session;seat=CTO;wm_prompt=90;wm_completion=8;wm_cached=30;batch=tickets=T-9001' "$USAGE_INSERTS" # seat=CTO——O_SEAT 现值（原测「决策席」字面系席位更名前陈账）

export DM_PENDING_COUNT=2 DM_PENDING_TICKETS=$'T-9001\nT-9002'
bash "$ROOT/scripts/meta/dm_tick.sh" >/dev/null
grep -q -- "-v ticket=CTO -v kind=audit" "$USAGE_INSERTS" # 聚合键=CTO（原测 META-DM 系席位更名前陈账——dm_tick USAGE_TICKET 现值）
grep -q 'batch=tickets=T-9001,T-9002' "$USAGE_INSERTS"
[[ $(wc -l < "$DECISION_CALLS") -eq 2 ]]
[[ $(grep -c 'WITH lock AS MATERIALIZED' "$USAGE_INSERTS") -eq 2 ]]
[[ $(find "$O_CTO_WORKDIR" -mindepth 1 -maxdepth 1 -type d -name 'usage.*' | wc -l) -eq 0 ]]

export DM_PENDING_COUNT=1 DM_PENDING_TICKETS=T-9002 DM_USAGE_FAIL=1
bash "$ROOT/scripts/meta/dm_tick.sh" > "$TMP/usage-failure.log" 2>&1
grep -q 'token usage collection failed; continuing decision lifecycle' "$TMP/usage-failure.log"
# T-0369 容错必报：写径败→带类升件行落账（写败/CTO-USAGE，note 前缀键——
# 原测 META-DM/META-DM-USAGE 系席位更名前陈账，usage_esc 现传 seat=CTO scope=CTO-USAGE）
grep -q "INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)" "$ESC_INSERTS"
grep -q "VALUES('CTO','升件','T-9002','usage 采集败——写径败" "$ESC_INSERTS"
grep -q "'写败','CTO-USAGE'" "$ESC_INSERTS"
[[ $(wc -l < "$DECISION_CALLS") -eq 3 ]]
[[ $(find "$O_CTO_WORKDIR" -mindepth 1 -maxdepth 1 -type d -name 'usage.*' | wc -l) -eq 1 ]]

# 同窗去重：再拍同败→dedup 见账行不再重报（防事件暴）
bash "$ROOT/scripts/meta/dm_tick.sh" > "$TMP/usage-failure2.log" 2>&1
[[ $(grep -c 'INSERT INTO ops_event' "$ESC_INSERTS") -eq 1 ]]
[[ $(wc -l < "$DECISION_CALLS") -eq 4 ]]

echo "decision ticket usage one/multi/empty batch dry run PASS"
