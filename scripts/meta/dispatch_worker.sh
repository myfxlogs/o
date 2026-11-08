#!/usr/bin/env bash
# Common quota/concurrency gate for all ticket worker launches.
set -u

TID=${1:-}
MODEL=${2:-swe-2-flash-high}
PROMPT=${3:-"读 /opt/o/skills/施工席/SKILL.md 施工协议 领 $TID 施工"}
[[ "$TID" =~ ^T-[0-9]{4}$ ]] || { echo "TID must be T-NNNN" >&2; exit 2; }
case "$MODEL" in swe-2-high|swe-2-medium|swe-2-flash-high) ;; *) echo "unsupported worker model: $MODEL" >&2; exit 2 ;; esac

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
TICKET_FILE=$(compgen -G "$TASK_DIR/$TID-*.md" | head -n1)
[ -n "$TICKET_FILE" ] || { echo "ticket file missing: $TID" >&2; exit 2; }
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

# 提案-KB向量检索-经验库召回面 首阶③：票面→kb_recall top-K 摘要注入提示词
# 「相关经验」段——mech_tick/dm_depgate/dm_suspend 一切派单径同经本装配面
# 同径得召回；物料缺席/零命中=跳过该段照发（fail-open 增强非闸件，
# 与 dispatch_order 同律）。召回行=读物指针非施工细节（单行指针律豁免面）。
KB_HITS=$("$SCRIPT_DIR/kb_recall.sh" "$TICKET_FILE" 2>/dev/null || true) # tolerate: 召回件缺席/败=零段 fail-open
if [ -n "$KB_HITS" ]; then
	PROMPT="$PROMPT"$'\n\n'"【相关经验·经验库召回】施工前对号读条目正文（docs/*/经验库/条目/，读物非执行件）："$'\n'"$KB_HITS"
fi

# GW-001 段C1：worker 退场即唤机械轨（ExecStopPost 单元出生即附——拉起
# 链闭环：本票收工→o-mech 起拍→补拉窗秒级续派下张）。半成品窗：属性只罩
# 本改动后新拉起的单元，改码瞬间在飞 worker 退场无此钩——由 o-mech.timer
# 5min 兜底拍吸收（候队留 dispatched 账不留断链，最多迟一拍不丢派）。
# `-` 前缀=唤轨败不炸本单元（worker 成败语义不被尾钩污染）；命令行字面
# 实证 systemd 255 --property=Exec* 整串拆 argv（o-c1probe 探针在案）。
if "$SYSTEMD_RUN" --user --unit="o-worker-$TID" --collect \
	--property="ExecStopPost=-/usr/bin/systemctl --user start --no-block o-mech.service" \
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
