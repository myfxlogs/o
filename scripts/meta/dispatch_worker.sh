#!/usr/bin/env bash
# Common quota/concurrency gate for all ticket worker launches.
set -u

TID=${1:-}
MODEL=${2:-swe-2-high}
PROMPT=${3:-"读 /opt/o/skills/施工席/SKILL.md 施工协议 领 $TID 施工"}
[[ "$TID" =~ ^T-[0-9]{4}$ ]] || { echo "TID must be T-NNNN" >&2; exit 2; }
case "$MODEL" in swe-2-high|swe-2-medium) ;; *) echo "unsupported worker model: $MODEL" >&2; exit 2 ;; esac

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
source "$SCRIPT_DIR/worker_runtime.sh"
o_worker_ensure_runtime_dir || exit 1

if o_worker_quota_until >/dev/null; then
	echo "worker queue held: free-model quota cooldown is active" >&2
	exit 75
else
	rc=$?
	[ "$rc" -eq 1 ] || { echo "quota state unreadable; refusing worker launch" >&2; exit 1; }
fi

LOCK="$(o_worker_runtime_dir)/dispatch.lock"
exec 9>"$LOCK"
flock -x 9

UNIT="o-worker-$TID.service"
SYSTEMCTL=${O_WORKER_SYSTEMCTL:-systemctl}
SYSTEMD_RUN=${O_WORKER_SYSTEMD_RUN:-systemd-run}
DEV_RUN_SCRIPT=${O_WORKER_DEV_RUN_SCRIPT:-/opt/o/scripts/meta/dev_run.sh}
TASK_DIR=${O_WORKER_TASK_DIR:-/opt/o/docs/tasks}
compgen -G "$TASK_DIR/$TID-*.md" >/dev/null || { echo "ticket file missing: $TID" >&2; exit 2; }
STATE=$("$SYSTEMCTL" --user is-active "$UNIT" 2>/dev/null || true)
case "$STATE" in active|activating) exit 0 ;; esac

if o_worker_quota_until >/dev/null; then
	echo "worker queue held: free-model quota cooldown became active" >&2
	exit 75
else
	rc=$?
	[ "$rc" -eq 1 ] || { echo "quota state unreadable; refusing worker launch" >&2; exit 1; }
fi

LIMIT=$(o_worker_parallel_limit) || exit $?
if ! UNITS=$("$SYSTEMCTL" --user list-units 'o-worker-*' --state=active,activating \
	--no-legend --plain 2>/dev/null); then
	echo "cannot read active worker units; refusing worker launch" >&2
	exit 1
fi
ACTIVE=$(printf '%s\n' "$UNITS" | awk 'NF { n++ } END { print n+0 }')
if (( ACTIVE >= LIMIT )); then
	echo "worker queue held: active=$ACTIVE limit=$LIMIT" >&2
	exit 75
fi

if "$SYSTEMD_RUN" --user --unit="o-worker-$TID" --collect \
	--setenv="DEV_RUN_MODEL=$MODEL" \
	"$DEV_RUN_SCRIPT" "$TID" "$PROMPT"; then
	flock -u 9
	exec 9>&-
	exit 0
else
	rc=$?
	flock -u 9
	exec 9>&-
	exit "$rc"
fi
