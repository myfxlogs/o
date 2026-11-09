#!/usr/bin/env bash
# run.sh —— 唤知会话壳（纪元3 D-0045：systemd-run o-cognition-<ts> unit
# 载本壳；脱自纪元2 dev_run.sh——票机件全剔,保留配额/心跳/usage三件）。
# 用法: run.sh ——runbook 固定取 scripts/cognition/runbook.md（版本化正本）。
set -u
MODEL=${COG_RUN_MODEL:-swe-2-max}
case "$MODEL" in swe-2-max|swe-2-high|swe-2-medium|swe-2-flash-high) ;; *) MODEL=swe-2-max;; esac

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
source "${O_WORKER_RUNTIME_LIB:-$SCRIPT_DIR/worker_runtime.sh}"
o_worker_ensure_runtime_dir || exit 1
DEVIN_BIN=${O_WORKER_DEVIN_BIN:-/home/mluser/.local/bin/devin}
RUNBOOK=${COG_RUNBOOK:-$SCRIPT_DIR/runbook.md}
[ -s "$RUNBOOK" ] || { echo "[cognition] runbook 缺: $RUNBOOK" >&2; exit 2; }

export O_SEAT=唤知席
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')

hb() { # $1=state——唤知席心跳按日键（会话逐日新面,互不盖）
  [ -n "$FIN_DSN" ] || return 0
  psql "$FIN_DSN" -qAt -c "INSERT INTO ops_headless(ticket,seat,state,last_activity)
    VALUES('COG-$(date +%Y%m%d)','唤知席','$1',now())
    ON CONFLICT (ticket) DO UPDATE SET seat=EXCLUDED.seat,
      state=EXCLUDED.state,last_activity=now(),updated_at=now()" >/dev/null 2>&1 || true # tolerate: 心跳=尽力径——hb写败不阻会话主径（dev_run.sh 先例同构）
}

SESSION_DIR="$(o_worker_runtime_dir)/sessions/cognition"
mkdir -p "$SESSION_DIR" && chmod 700 "$SESSION_DIR" && cd "$SESSION_DIR" || exit 1

if quota_reset=$(o_worker_quota_until); then
	echo "[cognition] quota cooldown active until epoch $quota_reset"
	hb stalled
	exit 0
else
	quota_rc=$?
	[ "$quota_rc" -ne 1 ] && { echo "[cognition] quota state unreadable; refusing" >&2; hb stalled; exit 1; }
fi

SESSION_STATE=$(o_worker_session_state "$DEVIN_BIN")
[ $? -ne 0 ] && { echo "[cognition] session discovery failed; refusing cold restart" >&2; hb stalled; exit 1; }
hb alive

LOG_FILE=$(mktemp "$(o_worker_runtime_dir)/cognition.XXXXXX")
chmod 600 "$LOG_FILE"
trap 'rm -f "$LOG_FILE"' EXIT
DEVIN_ARGS=()
case "$SESSION_STATE" in
	resume) DEVIN_ARGS+=(--continue) ;;
	new) ;;
	*) echo "[cognition] session discovery failed; refusing" >&2; hb stalled; exit 1 ;;
esac

TRANSCRIPT_FILE=$(mktemp "$SESSION_DIR/.devin-usage.XXXXXX.json")
chmod 600 "$TRANSCRIPT_FILE"
set +e
"$DEVIN_BIN" "${DEVIN_ARGS[@]}" --export "$TRANSCRIPT_FILE" -p --model "$MODEL" \
	--respect-workspace-trust false --permission-mode dangerous -- "$(cat "$RUNBOOK")" 2>&1 | tee "$LOG_FILE"
RC=${PIPESTATUS[0]}
set -e

if o_worker_is_rate_limited "$LOG_FILE"; then
	quota_reset=$(o_worker_record_quota_reset "$LOG_FILE")
	echo "[cognition] rate limit; cooldown until epoch $quota_reset"
	RC=1
fi
[ -s "$TRANSCRIPT_FILE" ] || rm -f "$TRANSCRIPT_FILE"
[ $RC -eq 0 ] && hb alive || hb stalled
exit 0
