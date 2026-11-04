#!/usr/bin/env bash
# Persist cumulative Devin transcript metrics as append-only per-session deltas.
set -euo pipefail

TICKET=${1:-}
KIND=${2:-}
SEAT=${3:-}
TRANSCRIPT=${4:-}
DSN=${5:-}
BATCH=${6:-}

[[ "$TICKET" =~ ^(T-[0-9]{4}|META-DM)$ ]] || { echo "usage: $0 T-NNNN|META-DM run|audit seat transcript dsn [batch]" >&2; exit 2; }
[[ "$KIND" == run || "$KIND" == audit ]] || { echo "invalid usage kind: $KIND" >&2; exit 2; }
[[ "$SEAT" == 施工席 || "$SEAT" == 决策席 ]] || { echo "invalid usage seat: $SEAT" >&2; exit 2; }
[[ -r "$TRANSCRIPT" ]] || { echo "[usage] transcript unavailable: $TRANSCRIPT" >&2; exit 1; }
[[ -n "$DSN" ]] || { echo "[usage] FIN_PG_DSN unavailable" >&2; exit 1; }

METRICS=$(python3 - "$TRANSCRIPT" <<'PY'
import json
import re
import sys

try:
    with open(sys.argv[1], encoding="utf-8") as f:
        transcript = json.load(f)
    session = transcript.get("session_id")
    if not isinstance(session, str) or not re.fullmatch(r"[A-Za-z0-9_-]+", session):
        raise ValueError("missing or invalid session_id")
    final = transcript.get("final_metrics") or {}
    agent = transcript.get("agent") or {}
    model = agent.get("model_name") or ""
    if not model:
        for step in reversed(transcript.get("steps") or []):
            model = step.get("model_name") or (step.get("extra") or {}).get("generation_model") or ""
            if model:
                break
    if not model:
        raise ValueError("model is missing")
    counts = [final.get(k) for k in (
        "total_prompt_tokens", "total_completion_tokens", "total_cached_tokens"
    )]
    if any(isinstance(v, bool) or not isinstance(v, int) or v < 0 for v in counts):
        raise ValueError("final token metrics are incomplete")
    model = str(model).replace("\t", " ").replace("\n", " ").strip()
    print("\t".join([session, model, *(str(v) for v in counts)]))
except Exception as exc:
    print(f"[usage] transcript parse failed: {exc}", file=sys.stderr)
    raise SystemExit(1)
PY
) || exit 1

IFS=$'\t' read -r SESSION MODEL PROMPT COMPLETION CACHED <<<"$METRICS"
[[ -n "$SESSION" && "$PROMPT" =~ ^[0-9]+$ && "$COMPLETION" =~ ^[0-9]+$ && "$CACHED" =~ ^[0-9]+$ ]] || {
	echo "[usage] invalid parsed transcript metrics" >&2
	exit 1
}
NOTE="session=$SESSION;seat=$SEAT;wm_prompt=$PROMPT;wm_completion=$COMPLETION;wm_cached=$CACHED"
[[ -z "$BATCH" ]] || NOTE+=";batch=$BATCH"

SQL=$(cat <<'SQL'
WITH lock AS MATERIALIZED (
    SELECT pg_advisory_xact_lock(hashtext(:'ticket' || ':' || :'session'))
), previous AS (
    SELECT id,
           COALESCE((regexp_match(note, '(^|;)wm_prompt=([0-9]+)(;|$)'))[2]::bigint, 0) AS prompt,
           COALESCE((regexp_match(note, '(^|;)wm_completion=([0-9]+)(;|$)'))[2]::bigint, 0) AS completion,
           COALESCE((regexp_match(note, '(^|;)wm_cached=([0-9]+)(;|$)'))[2]::bigint, 0) AS cached
      FROM ops_ticket_usage, lock
     WHERE ticket=:'ticket' AND source='devin-cli'
       AND (regexp_match(note, '^session=([^;]+);'))[1] = :'session'
     ORDER BY id DESC
     LIMIT 1
), current_metrics AS (
    SELECT :'prompt'::bigint AS prompt,
           :'completion'::bigint AS completion,
           :'cached'::bigint AS cached
), delta AS (
    SELECT c.prompt, c.completion, c.cached,
           COALESCE(p.prompt, 0) AS old_prompt,
           COALESCE(p.completion, 0) AS old_completion,
           COALESCE(p.cached, 0) AS old_cached,
           p.id IS NOT NULL AS seen
      FROM current_metrics c LEFT JOIN previous p ON TRUE
), inserted AS (
    INSERT INTO ops_ticket_usage
        (ticket, kind, model, prompt_tokens, completion_tokens, cached_tokens, source, note)
    SELECT :'ticket', :'kind', :'model',
           d.prompt-d.old_prompt, d.completion-d.old_completion, d.cached-d.old_cached,
           'devin-cli', :'note'
      FROM delta d
     WHERE d.prompt >= d.old_prompt
       AND d.completion >= d.old_completion
       AND d.cached >= d.old_cached
       AND (NOT d.seen OR d.prompt>d.old_prompt OR d.completion>d.old_completion OR d.cached>d.old_cached)
    RETURNING id
)
SELECT CASE
         WHEN EXISTS (SELECT 1 FROM inserted) THEN 'inserted'
         WHEN EXISTS (SELECT 1 FROM delta WHERE prompt>=old_prompt AND completion>=old_completion AND cached>=old_cached)
           THEN 'no-delta'
         ELSE 'watermark-regression'
       END;
SQL
)

if ! RESULT=$(psql "$DSN" -XqAt -v ON_ERROR_STOP=1 \
	-v "ticket=$TICKET" -v "kind=$KIND" -v "model=$MODEL" \
	-v "session=$SESSION" -v "prompt=$PROMPT" -v "completion=$COMPLETION" \
	-v "cached=$CACHED" -v "note=$NOTE" -c "$SQL"); then
	echo "[usage] ops_ticket_usage write failed for $TICKET/$SESSION" >&2
	exit 1
fi
case "$RESULT" in
	inserted) echo "[usage] recorded $TICKET $SEAT $MODEL cumulative_prompt=$PROMPT cumulative_completion=$COMPLETION cumulative_cached=$CACHED" ;;
	no-delta) echo "[usage] no new tokens for $TICKET/$SESSION" ;;
	*) echo "[usage] cumulative transcript counters regressed for $TICKET/$SESSION" >&2; exit 1 ;;
esac
