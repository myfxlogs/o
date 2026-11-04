#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin"

cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
query="${*: -1}"
case "$query" in
	*"SELECT count(*)"*)
		printf '%s\n' "$query" >> "$PENDING_SQL_LOG"
		grep -Fq "e.event IN ('收口','移交','收口待审')" <<<"$query"
		grep -Fq "e.event='已批'" <<<"$query"
		grep -Fq "('派单','领单','拉起','裁决','完成','驳回','挂起')" <<<"$query"
		case "$DM_SCENARIO" in
			empty) echo 0 ;;
			review|approved) echo 1 ;;
			*) echo "unknown scenario: $DM_SCENARIO" >&2; exit 2 ;;
		esac
		;;
	*"SELECT l.ticket"*)
		case "$DM_SCENARIO" in
			empty|approved) printf 'T-0341|swe-2-medium\n' ;;
			review) ;;
		esac
		;;
esac
EOF
cat > "$TMP/bin/systemctl" <<'EOF'
#!/usr/bin/env bash
if [[ ${2:-} == is-active ]]; then
	echo inactive
	exit 3
fi
if [[ ${2:-} == list-units ]]; then
	exit 0
fi
exit 2
EOF
cat > "$TMP/bin/dispatch-worker" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$DISPATCH_LOG"
EOF
cat > "$TMP/bin/decision-devin" <<'EOF'
#!/usr/bin/env bash
echo called >> "$DECISION_LOG"
EOF
chmod +x "$TMP/bin/"*

printf 'FIN_PG_DSN=fake\n' > "$TMP/env"
export PATH="$TMP/bin:$PATH"
export DM_ENV_FILE="$TMP/env"
export DISPATCH_LOG="$TMP/dispatch.log"
export DECISION_LOG="$TMP/decision.log"
export PENDING_SQL_LOG="$TMP/pending.sql.log"
export O_WORKER_DISPATCH_SCRIPT="$TMP/bin/dispatch-worker"
export O_DECISION_DEVIN_BIN="$TMP/bin/decision-devin"

# Pure-empty queue stays zero-LLM but still resumes a rate-limited worker.
export DM_SCENARIO=empty
bash "$ROOT/scripts/meta/dm_tick.sh"
[[ ! -e "$DECISION_LOG" ]]
grep -q '^T-0341 swe-2-medium ' "$DISPATCH_LOG"

# Review-close pending and approved-dispatch pending both enter the decision
# model gate. The stub checks the SQL carries both predicates and the complete
# approved-ticket terminal set before returning each synthetic row count.
export DM_SCENARIO=review
bash "$ROOT/scripts/meta/dm_tick.sh"
export DM_SCENARIO=approved
bash "$ROOT/scripts/meta/dm_tick.sh"
[[ $(wc -l < "$DECISION_LOG") -eq 2 ]]
[[ $(grep -c '^SELECT count(\*)' "$PENDING_SQL_LOG") -eq 3 ]]
[[ $(grep -c '^T-0341 ' "$DISPATCH_LOG") -eq 2 ]]

echo "dm tick pending gate three-state dry run PASS"
