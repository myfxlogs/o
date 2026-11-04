#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin"

cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
ticket= kind= model= session= prompt= completion= cached= note= sql=
while (($#)); do
	case "$1" in
		-v)
			shift; key=${1%%=*}; value=${1#*=}
			case "$key" in
				ticket) ticket=$value ;; kind) kind=$value ;; model) model=$value ;;
				session) session=$value ;; prompt) prompt=$value ;;
				completion) completion=$value ;; cached) cached=$value ;; note) note=$value ;;
			esac
			;;
		-c) shift; sql=$1 ;;
	esac
	shift
done
grep -Fq 'pg_advisory_xact_lock' <<<"$sql"
grep -Fq "source='devin-cli'" <<<"$sql"
grep -Fq "(regexp_match(note, '^session=([^;]+);'))[1] = :'session'" <<<"$sql"
grep -Fq 'd.prompt-d.old_prompt' <<<"$sql"
grep -Fq 'd.completion-d.old_completion' <<<"$sql"
grep -Fq 'd.cached-d.old_cached' <<<"$sql"
printf '%s\n' "$sql" >> "$USAGE_SQL_LOG"

old=$(awk -F '\t' -v t="$ticket" -v s="$session" '$1==t && $2==s {line=$0} END {print line}' "$USAGE_STATE" 2>/dev/null || true)
if [[ -n "$old" ]]; then
	IFS=$'\t' read -r _ _ old_prompt old_completion old_cached <<<"$old"
else
	old_prompt=0 old_completion=0 old_cached=0
fi
if (( prompt < old_prompt || completion < old_completion || cached < old_cached )); then
	echo watermark-regression
	exit 0
fi
if [[ -n "$old" ]] && (( prompt == old_prompt && completion == old_completion && cached == old_cached )); then
	echo no-delta
	exit 0
fi
printf '%s\t%s\t%s\t%s\t%s\n' "$ticket" "$session" "$prompt" "$completion" "$cached" >> "$USAGE_STATE"
printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$ticket" "$kind" "$model" \
	"$((prompt-old_prompt))" "$((completion-old_completion))" \
	"$((cached-old_cached))" "$note" >> "$USAGE_ROWS"
echo inserted
EOF
chmod +x "$TMP/bin/psql"

export PATH="$TMP/bin:$PATH"
export USAGE_STATE="$TMP/state.tsv"
export USAGE_ROWS="$TMP/rows.tsv"
export USAGE_SQL_LOG="$TMP/sql.log"

write_transcript() {
	python3 - "$1" "$2" "$3" "$4" "$5" <<'PY'
import json,sys
path,session,prompt,completion,cached=sys.argv[1:]
with open(path,'w') as f:
 json.dump({'session_id':session,'agent':{'model_name':'SWE-2 High'},'final_metrics':{
  'total_prompt_tokens':int(prompt),'total_completion_tokens':int(completion),
  'total_cached_tokens':int(cached)}},f)
PY
}

write_transcript "$TMP/worker.json" worker-session 100 10 40
"$ROOT/scripts/meta/ticket_usage.sh" T-9001 run 施工席 "$TMP/worker.json" fake-dsn >/dev/null
write_transcript "$TMP/worker.json" worker-session 150 17 55
"$ROOT/scripts/meta/ticket_usage.sh" T-9001 run 施工席 "$TMP/worker.json" fake-dsn >/dev/null
"$ROOT/scripts/meta/ticket_usage.sh" T-9001 run 施工席 "$TMP/worker.json" fake-dsn | grep -q 'no new tokens'
[[ $(wc -l < "$USAGE_ROWS") -eq 2 ]]
awk -F '\t' 'NR==1 {if ($4!=100 || $5!=10 || $6!=40) exit 1} NR==2 {if ($4!=50 || $5!=7 || $6!=15) exit 1}' "$USAGE_ROWS"
grep -q 'session=worker-session;seat=施工席;wm_prompt=150;wm_completion=17;wm_cached=55' "$USAGE_ROWS"

write_transcript "$TMP/decision.json" decision-session 90 8 30
"$ROOT/scripts/meta/ticket_usage.sh" META-DM audit 决策席 "$TMP/decision.json" fake-dsn 'tickets=T-9001,T-9002' >/dev/null
grep -q '^META-DM.*audit.*SWE-2 High.*90.*8.*30.*seat=决策席' "$USAGE_ROWS"

write_transcript "$TMP/worker.json" worker-session 140 17 55
if "$ROOT/scripts/meta/ticket_usage.sh" T-9001 run 施工席 "$TMP/worker.json" fake-dsn >/dev/null 2>&1; then
	echo "watermark regression must fail closed" >&2
	exit 1
fi
if "$ROOT/scripts/meta/ticket_usage.sh" T-9001 run 施工席 "$TMP/missing.json" fake-dsn >/dev/null 2>&1; then
	echo "missing transcript must be visible to caller" >&2
	exit 1
fi

echo "ticket usage transcript, watermark, seat, and failure dry run PASS"
