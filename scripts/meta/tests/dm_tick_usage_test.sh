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
case "$query" in
	*"SELECT count(*)"*) echo "$DM_PENDING_COUNT" ;;
	*"SELECT l.ticket FROM ("*"COALESCE"*) ;;
	*"SELECT l.ticket FROM ("*) printf '%s\n' "$DM_PENDING_TICKETS" ;;
	*"WITH lock AS MATERIALIZED"*)
		grep -Fq "source='devin-cli'" <<<"$query"
		[[ ${DM_USAGE_FAIL:-0} != 1 ]] || exit 1
		printf '%s\n' "$*" >> "$USAGE_INSERTS"
		echo inserted
		;;
	*"INSERT INTO ops_headless"*) ;;
	*) echo "unexpected psql query: $query" >&2; exit 2 ;;
esac
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
chmod +x "$TMP/bin/psql" "$TMP/bin/devin"
printf 'FIN_PG_DSN=fake-dsn\n' > "$TMP/env"
export PATH="$TMP/bin:$PATH"
export DM_ENV_FILE="$TMP/env"
export O_DECISION_WORKDIR="$TMP/decision"
mkdir -p "$O_DECISION_WORKDIR"
export O_DECISION_DEVIN_BIN="$TMP/bin/devin"
export O_WORKER_USAGE_BIN="$ROOT/scripts/meta/ticket_usage.sh"
export DECISION_CALLS="$TMP/decision-calls"
export USAGE_INSERTS="$TMP/usage-inserts"

export DM_PENDING_COUNT=0 DM_PENDING_TICKETS=
bash "$ROOT/scripts/meta/dm_tick.sh" >/dev/null
[[ ! -e "$DECISION_CALLS" && ! -e "$USAGE_INSERTS" ]]

# One ticket keeps direct ticket attribution; a multi-ticket model batch is
# recorded once under META-DM with its explicit ticket set, never duplicated.
export DM_PENDING_COUNT=1 DM_PENDING_TICKETS=T-9001
bash "$ROOT/scripts/meta/dm_tick.sh" >/dev/null
grep -q -- "-v ticket=T-9001 -v kind=audit -v model=SWE-2 Max" "$USAGE_INSERTS"
grep -q 'note=session=decision-session;seat=决策席;wm_prompt=90;wm_completion=8;wm_cached=30;batch=tickets=T-9001' "$USAGE_INSERTS"

export DM_PENDING_COUNT=2 DM_PENDING_TICKETS=$'T-9001\nT-9002'
bash "$ROOT/scripts/meta/dm_tick.sh" >/dev/null
grep -q -- "-v ticket=META-DM -v kind=audit" "$USAGE_INSERTS"
grep -q 'batch=tickets=T-9001,T-9002' "$USAGE_INSERTS"
[[ $(wc -l < "$DECISION_CALLS") -eq 2 ]]
[[ $(grep -c 'WITH lock AS MATERIALIZED' "$USAGE_INSERTS") -eq 2 ]]
[[ $(find "$O_DECISION_WORKDIR" -mindepth 1 -maxdepth 1 -type d -name 'usage.*' | wc -l) -eq 0 ]]

export DM_PENDING_COUNT=1 DM_PENDING_TICKETS=T-9002 DM_USAGE_FAIL=1
bash "$ROOT/scripts/meta/dm_tick.sh" > "$TMP/usage-failure.log" 2>&1
grep -q 'token usage collection failed; continuing decision lifecycle' "$TMP/usage-failure.log"
[[ $(wc -l < "$DECISION_CALLS") -eq 3 ]]
[[ $(find "$O_DECISION_WORKDIR" -mindepth 1 -maxdepth 1 -type d -name 'usage.*' | wc -l) -eq 1 ]]

echo "decision ticket usage one/multi/empty batch dry run PASS"
