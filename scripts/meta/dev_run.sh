#!/usr/bin/env bash
# dev_run.sh —— 无头施工席瞬态 run 壳（T-0324 拉起式钦定——systemd-run
# o-worker-<tid> unit 载本壳；每票稳定 session cwd，worktree 职卡领单步自建）。
set -u
TID=${1:?usage: dev_run.sh <T-编号> [提示文]}
PROMPT=${2:-"读 /opt/o/skills/施工席/SKILL.md 施工协议 领 $TID 施工"}
[[ "$TID" =~ ^T-[0-9]{4}$ ]] || { echo "TID must be T-NNNN" >&2; exit 2; }
# 施工席模=swe-2-flash-high（业主令2026-11-08）；白名单留 high/medium 兼容历史派单行。
MODEL=${DEV_RUN_MODEL:-swe-2-flash-high}
case "$MODEL" in swe-2-high|swe-2-medium|swe-2-flash-high) ;; *) MODEL=swe-2-flash-high;; esac

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
source "${O_WORKER_RUNTIME_LIB:-$SCRIPT_DIR/worker_runtime.sh}"
o_worker_ensure_runtime_dir || exit 1
DEVIN_BIN=${O_WORKER_DEVIN_BIN:-/home/mluser/.local/bin/devin}

export O_SEAT=施工席
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
REPORT_BIN=${O_WORKER_REPORT_BIN:-$HOME/bin/report.sh}
OPS_ESC_SH=${O_WORKER_OPS_ESC_SH:-$SCRIPT_DIR/ops_esc.sh}

hb() { # $1=state——按票分行心跳（META-Dev-<tid> 并行施工互不盖）
  [ -n "$FIN_DSN" ] || return 0
  psql "$FIN_DSN" -qAt -c "INSERT INTO ops_headless(ticket,seat,state,last_activity)
    VALUES('META-Dev-$TID','施工席','$1',now())
    ON CONFLICT (ticket) DO UPDATE SET seat=EXCLUDED.seat,
      state=EXCLUDED.state,last_activity=now(),updated_at=now()" >/dev/null 2>&1 || true
}

SESSION_DIR="$(o_worker_runtime_dir)/sessions/$TID"
mkdir -p "$SESSION_DIR"
chmod 700 "$SESSION_DIR"
cd "$SESSION_DIR" || exit 1

if quota_reset=$(o_worker_quota_until); then
	echo "[worker] quota cooldown active until epoch $quota_reset"
	"$REPORT_BIN" 挂起 "$TID" "rate-limit:auto until=$quota_reset model=$MODEL" >/dev/null 2>&1 || true
	hb stalled
	exit 0
else
	quota_rc=$?
	if [ "$quota_rc" -ne 1 ]; then
		echo "[worker] quota state unreadable; refusing model invocation" >&2
		"$REPORT_BIN" 挂起 "$TID" "quota-state-unreadable" >/dev/null 2>&1 || true
		hb stalled
		exit 1
	fi
fi

# Each ticket keeps its own CLI session. A failed session lookup stops here
# instead of silently paying the cold-start context cost again.
SESSION_STATE=$(o_worker_session_state "$DEVIN_BIN")
SESSION_RC=$?
if [ "$SESSION_RC" -ne 0 ]; then
	echo "[worker] Devin session discovery failed; refusing cold restart" >&2
	"$REPORT_BIN" 挂起 "$TID" "devin-session-discovery-failed" >/dev/null 2>&1 || true
	hb stalled
	exit 1
fi
hb alive

LOG_FILE=$(mktemp "$(o_worker_runtime_dir)/${TID}.XXXXXX")
chmod 600 "$LOG_FILE"
trap 'rm -f "$LOG_FILE"' EXIT
DEVIN_ARGS=()
case "$SESSION_STATE" in
	resume) DEVIN_ARGS+=(--continue) ;;
	new) ;;
	*) echo "Devin session discovery failed; refusing cold restart" >&2; hb stalled; exit 1 ;;
esac

TRANSCRIPT_FILE=$(mktemp "$SESSION_DIR/.devin-usage.XXXXXX.json")
chmod 600 "$TRANSCRIPT_FILE"
set +e

"$DEVIN_BIN" "${DEVIN_ARGS[@]}" --export "$TRANSCRIPT_FILE" -p --model "$MODEL" \
	--respect-workspace-trust false --permission-mode dangerous -- "$PROMPT" 2>&1 | tee "$LOG_FILE"
RC=${PIPESTATUS[0]}
set -e

if o_worker_is_rate_limited "$LOG_FILE"; then
	quota_reset=$(o_worker_record_quota_reset "$LOG_FILE")
	echo "[worker] free-model rate limit; queued until epoch $quota_reset"
	"$REPORT_BIN" 挂起 "$TID" "rate-limit:auto until=$quota_reset model=$MODEL" >/dev/null 2>&1 || true
	RC=1
fi

USAGE_BIN=${O_WORKER_USAGE_BIN:-$SCRIPT_DIR/ticket_usage.sh}
if [ -e "$TRANSCRIPT_FILE" ]; then
	chmod 600 "$TRANSCRIPT_FILE" 2>/dev/null || true
fi
if [ -s "$TRANSCRIPT_FILE" ]; then
	if "$USAGE_BIN" "$TID" run 施工席 "$TRANSCRIPT_FILE" "$FIN_DSN"; then
		rm -f "$TRANSCRIPT_FILE"
	else
		echo "[worker] token usage collection failed; continuing ticket lifecycle; transcript kept=$TRANSCRIPT_FILE" >&2
		OPS_ESC_ENV_FILE="$ENVF" "$OPS_ESC_SH" 施工席 "$TID" 写败 META-DEV-USAGE \
			"usage 采集败" "写径败 transcript kept=$TRANSCRIPT_FILE" || true # tolerate: 自报末环——ops_esc 写径败静默退（结构性限度在案）
	fi
else
	echo "[worker] token transcript missing/empty; continuing ticket lifecycle" >&2
	OPS_ESC_ENV_FILE="$ENVF" "$OPS_ESC_SH" 施工席 "$TID" 写败 META-DEV-USAGE \
		"usage 采集败" "transcript 缺/空——export 未产" || true # tolerate: 自报末环——ops_esc 写径败静默退（结构性限度在案）
	rm -f "$TRANSCRIPT_FILE"
fi
[ $RC -eq 0 ] && hb alive || hb stalled
exit 0
