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
	*"SELECT count(*)"*) echo 0 ;;
	*"SELECT l.ticket"*) printf 'T-0341|swe-2-medium\n' ;;
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
export O_WORKER_DISPATCH_SCRIPT="$TMP/bin/dispatch-worker"
export O_DECISION_DEVIN_BIN="$TMP/bin/decision-devin"

# With no adjudication pending, DM skips its own model call but still resumes
# a rate-limited worker ticket through the shared dispatch gate.
bash "$ROOT/scripts/meta/dm_tick.sh"
[[ ! -e "$DECISION_LOG" ]]
grep -q '^T-0341 swe-2-medium ' "$DISPATCH_LOG"

echo "dm tick worker resume test PASS"
