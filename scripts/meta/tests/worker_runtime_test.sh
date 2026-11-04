#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT

export O_WORKER_RUNTIME_DIR="$TMP/state"
source "$ROOT/scripts/meta/worker_runtime.sh"

mkdir -p "$TMP/bin" "$TMP/tasks"
touch "$TMP/tasks/T-9001-resume-test.md" "$TMP/tasks/T-9002-resume-test.md" \
	"$TMP/tasks/T-9003-resume-test.md"

cat > "$TMP/bin/devin" <<'EOF'
#!/usr/bin/env bash
if [[ ${1:-} == list ]]; then
	if [[ ${MOCK_LIST_FAIL:-0} == 1 ]]; then
		exit 9
	fi
	if [[ -f "$PWD/.mock-session" ]]; then
		printf '[{"session_id":"mock"}]\n'
	else
		printf '[]\n'
	fi
	exit 0
fi
printf '%s\n' "$*" >> "$DEVIN_CALLS"
export_file=
while (($#)); do
	if [[ $1 == --export ]]; then
		shift
		export_file=${1:-}
		break
	fi
	shift
done
if [[ -n "$export_file" ]]; then
	python3 - "$export_file" <<'PY'
import json,sys
with open(sys.argv[1],'w') as f:
 json.dump({'session_id':'mock-session','agent':{'model_name':'SWE-2 Medium'},
  'final_metrics':{'total_prompt_tokens':120,'total_completion_tokens':7,'total_cached_tokens':80}},f)
PY
fi
touch "$PWD/.mock-session"
if [[ ${MOCK_RATE_LIMIT:-0} == 1 ]]; then
	echo "Reached free model rate limit. Your limit will reset in 2 minutes." >&2
	exit 1
fi
echo "mock session completed"
EOF
cat > "$TMP/bin/report" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$REPORT_CALLS"
EOF
cat > "$TMP/bin/usage" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$USAGE_CALLS"
[[ ${MOCK_USAGE_FAIL:-0} != 1 ]]
EOF
chmod +x "$TMP/bin/devin" "$TMP/bin/report" "$TMP/bin/usage"

DEVIN_CALLS="$TMP/devin-calls"
REPORT_CALLS="$TMP/report-calls"
USAGE_CALLS="$TMP/usage-calls"
export DEVIN_CALLS REPORT_CALLS USAGE_CALLS
export O_WORKER_DEVIN_BIN="$TMP/bin/devin"
export O_WORKER_REPORT_BIN="$TMP/bin/report"
export O_WORKER_USAGE_BIN="$TMP/bin/usage"
export REPORT_ENV_FILE="$TMP/no-env"
export MOCK_RATE_LIMIT=1
export MOCK_USAGE_FAIL=1

bash "$ROOT/scripts/meta/dev_run.sh" T-9001 "start ticket" > "$TMP/first-run.log" 2>&1
grep -q 'token usage collection failed; continuing ticket lifecycle' "$TMP/first-run.log"
[[ -s "$O_WORKER_RUNTIME_DIR/quota-reset-epoch" ]]
grep -q '^挂起 T-9001 rate-limit:auto ' "$REPORT_CALLS"
grep -q '^T-9001 run 施工席 .*\.devin-usage\..*\.json ' "$USAGE_CALLS"
[[ $(find "$O_WORKER_RUNTIME_DIR/sessions/T-9001" -maxdepth 1 -name '.devin-usage.*.json' | wc -l) -eq 1 ]]
first_reset=$(cat "$O_WORKER_RUNTIME_DIR/quota-reset-epoch")
(( first_reset > $(date +%s) + 150 ))
printf 'temporary network failure\n' > "$TMP/network.log"
if o_worker_is_rate_limited "$TMP/network.log"; then
	echo "generic failures must not trigger quota cooldown" >&2
	exit 1
fi

# A concurrent shorter reset must not shorten the established cooldown.
longer_reset=$(( $(date +%s) + 600 ))
printf '%s\n' "$longer_reset" > "$O_WORKER_RUNTIME_DIR/quota-reset-epoch"
printf 'Your limit will reset in 2 minutes.\n' > "$TMP/limit.log"
o_worker_record_quota_reset "$TMP/limit.log" >/dev/null
[[ $(cat "$O_WORKER_RUNTIME_DIR/quota-reset-epoch") -eq $longer_reset ]]

# Simulate the cooldown passing; the next transient run must resume the same
# ticket's Devin session instead of creating a fresh one.
printf '0\n' > "$O_WORKER_RUNTIME_DIR/quota-reset-epoch"
export MOCK_RATE_LIMIT=0
export MOCK_USAGE_FAIL=0
bash "$ROOT/scripts/meta/dev_run.sh" T-9001 "resume ticket"
grep -q '^--continue ' "$DEVIN_CALLS"
[[ $(wc -l < "$USAGE_CALLS") -eq 2 ]]
[[ $(find "$O_WORKER_RUNTIME_DIR/sessions/T-9001" -maxdepth 1 -name '.devin-usage.*.json' | wc -l) -eq 1 ]]

# Another ticket gets a separate session directory and cold start.
bash "$ROOT/scripts/meta/dev_run.sh" T-9002 "other ticket"
[[ -d "$O_WORKER_RUNTIME_DIR/sessions/T-9001" ]]
[[ -d "$O_WORKER_RUNTIME_DIR/sessions/T-9002" ]]
grep -q 'other ticket' "$DEVIN_CALLS"

# A failed local session lookup fails closed and never starts a cold session.
export MOCK_LIST_FAIL=1
calls_before=$(wc -l < "$DEVIN_CALLS")
if bash "$ROOT/scripts/meta/dev_run.sh" T-9003 "must not cold start"; then
	echo "expected session discovery failure" >&2
	exit 1
fi
[[ $(wc -l < "$DEVIN_CALLS") -eq $calls_before ]]
grep -q 'devin-session-discovery-failed' "$REPORT_CALLS"
export MOCK_LIST_FAIL=0

cat > "$TMP/bin/systemctl" <<'EOF'
#!/usr/bin/env bash
if [[ ${2:-} == is-active ]]; then
	if grep -Fxq "${3:-}" "$ACTIVE_UNITS"; then
		echo active
		exit 0
	fi
	echo inactive
	exit 3
fi
if [[ ${2:-} == list-units ]]; then
	cat "$ACTIVE_UNITS"
	exit 0
fi
exit 2
EOF
cat > "$TMP/bin/systemd-run" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$LAUNCHES"
EOF
chmod +x "$TMP/bin/systemctl" "$TMP/bin/systemd-run"
touch "$TMP/active-units"
export ACTIVE_UNITS="$TMP/active-units"
export LAUNCHES="$TMP/launches"
export O_WORKER_SYSTEMCTL="$TMP/bin/systemctl"
export O_WORKER_SYSTEMD_RUN="$TMP/bin/systemd-run"
export O_WORKER_DEV_RUN_SCRIPT="$ROOT/scripts/meta/dev_run.sh"
export O_WORKER_TASK_DIR="$TMP/tasks"
export O_WORKER_PARALLEL_LIMIT=1

# One active worker fills the free-tier default; the dispatch stays queued.
printf 'o-worker-T-9000.service\n' > "$ACTIVE_UNITS"
if bash "$ROOT/scripts/meta/dispatch_worker.sh" T-9001 swe-2-medium "queued"; then
	echo "expected full worker limit to queue" >&2
	exit 1
else
	[[ $? -eq 75 ]]
fi
[[ ! -s "$LAUNCHES" ]]

# After the active worker exits, exactly one launch is accepted.
: > "$ACTIVE_UNITS"
bash "$ROOT/scripts/meta/dispatch_worker.sh" T-9001 swe-2-medium "queued"
[[ $(wc -l < "$LAUNCHES") -eq 1 ]]
grep -q -- '--setenv=DEV_RUN_MODEL=swe-2-medium' "$LAUNCHES"

# An already-active ticket is idempotent even at the one-worker limit.
printf 'o-worker-T-9001.service\n' > "$ACTIVE_UNITS"
bash "$ROOT/scripts/meta/dispatch_worker.sh" T-9001 swe-2-medium "duplicate"
[[ $(wc -l < "$LAUNCHES") -eq 1 ]]

# A deliberate override permits two concurrent workers, never more.
printf 'o-worker-T-9000.service\n' > "$ACTIVE_UNITS"
export O_WORKER_PARALLEL_LIMIT=2
bash "$ROOT/scripts/meta/dispatch_worker.sh" T-9002 swe-2-high "second slot"
[[ $(wc -l < "$LAUNCHES") -eq 2 ]]
printf 'o-worker-T-9000.service\no-worker-T-9001.service\n' > "$ACTIVE_UNITS"
if bash "$ROOT/scripts/meta/dispatch_worker.sh" T-9003 swe-2-high "third slot"; then
	echo "expected hard limit of two workers" >&2
	exit 1
else
	[[ $? -eq 75 ]]
fi
[[ $(wc -l < "$LAUNCHES") -eq 2 ]]

# A quota cooldown blocks dispatch before systemd-run.
printf '%s\n' "$(( $(date +%s) + 300 ))" > "$O_WORKER_RUNTIME_DIR/quota-reset-epoch"
export O_WORKER_PARALLEL_LIMIT=1
if bash "$ROOT/scripts/meta/dispatch_worker.sh" T-9002 swe-2-high "cooldown"; then
	echo "expected active quota cooldown to queue" >&2
	exit 1
else
	[[ $? -eq 75 ]]
fi
[[ $(wc -l < "$LAUNCHES") -eq 2 ]]

# A corrupt cooldown state fails closed rather than allowing more model calls.
printf 'not-an-epoch\n' > "$O_WORKER_RUNTIME_DIR/quota-reset-epoch"
if bash "$ROOT/scripts/meta/dispatch_worker.sh" T-9002 swe-2-high "bad state"; then
	echo "expected unreadable quota state to fail closed" >&2
	exit 1
else
	[[ $? -eq 1 ]]
fi
[[ $(wc -l < "$LAUNCHES") -eq 2 ]]

echo "worker runtime tests PASS"
