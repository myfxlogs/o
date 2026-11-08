#!/usr/bin/env bash
# ceo_tick.sh —— CEO 席无头班拍（T-0456 业主令双无头建席）：
# 照 dm_tick.sh 同构三件——PG advisory 单例闸（kick 连发/兜底拍撞车
# 后到即退）+队列闸（ceo_queue.sh 空集=心跳即退零烧）+班次 drain
# （会话班内 rescan→队空才退）。续跑式常驻：devin -c 钉同 session
# 跨班续上下文（压实丢精度险=ops_event 单源账对冲）。
# 唤径: metabus dm 帧双唤（与决策席同帧各自滤队）+30min 兜底拍。
set -u
CEO_DIR=${O_CEO_WORKDIR:-/var/tmp/o-ceo}
export REPORT_KICK=O-DM
export O_SEAT=CEO
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
CEO_QUEUE_SH=${O_CEO_QUEUE_SH:-"$SCRIPT_DIR/ceo_queue.sh"}
OPS_ESC_SH="$SCRIPT_DIR/ops_esc.sh"
USAGE_BIN=${O_WORKER_USAGE_BIN:-/opt/o/scripts/meta/ticket_usage.sh}

hb() { # 心跳行——ops_headless seat=CEO 拍迹
  [ -n "$FIN_DSN" ] || return 0
  psql "$FIN_DSN" -qAt -c \
    "INSERT INTO ops_headless(seat,state,note) VALUES('CEO','$1','$2')" \
    >/dev/null 2>&1 || true
}

# 单例闸（PG advisory——kick 连发撞车后到即退）
LOCK_KEY=884201
LOCKRC=0
LOCKED=$(psql "$FIN_DSN" -qAt -c \
  "SELECT pg_try_advisory_lock($LOCK_KEY)" 2>/dev/null) || LOCKRC=$?
[ "$LOCKRC" -ne 0 ] && { hb err "锁探败 rc=$LOCKRC 按有主退"; exit 0; }
[ "$LOCKED" != "t" ] && exit 0  # 有班在飞——后到即退
trap 'psql "$FIN_DSN" -qAt -c "SELECT pg_advisory_unlock($LOCK_KEY)" >/dev/null 2>&1 || true' EXIT

# 队列闸——空集=心跳即退零烧会话
PENDING=$(FIN_DSN="$FIN_DSN" "$CEO_QUEUE_SH" 2>/dev/null || true)
N=$(printf '%s\n' "$PENDING" | awk 'NF {n++} END {print n+0}')
if [ "$N" -eq 0 ]; then
  hb alive "空拍——CEO 队列空"
  exit 0
fi

# 开班——续跑式：transcript 在案则 -c 续同 session（新席首班自然无档全启）
mkdir -p "$CEO_DIR" || exit 1
USAGE_DIR=$(mktemp -d "$CEO_DIR/usage.XXXXXX"); chmod 700 "$USAGE_DIR"
USAGE_TRANSCRIPT="$USAGE_DIR/transcript.json"
cd "$CEO_DIR" || exit 1
DEVIN_BIN=${O_DECISION_DEVIN_BIN:-/home/mluser/.local/bin/devin}
RESUME_FLAG=()
last=$(ls -t "$CEO_DIR"/usage.*/transcript.json 2>/dev/null | grep -v "$USAGE_TRANSCRIPT" | head -1 || true)
[ -n "$last" ] && [ -s "$last" ] && RESUME_FLAG=(-c)
BATCH="tickets=$(printf '%s\n' "$PENDING" | paste -sd, -)"
"$DEVIN_BIN" --export "$USAGE_TRANSCRIPT" "${RESUME_FLAG[@]}" \
  -p --model swe --respect-workspace-trust false \
  --permission-mode dangerous -- \
  "读 /opt/o/skills/CEO席/SKILL.md——CEO 无头班次开工:scan CEO 队列($BATCH)→逐件裁或呈业主→rescan→队空才退;机械/执行面件移交 CTO 不代办;方向级件裁毕落 ops_event seat=CEO 行;拿不准=呈批升级不裁" \
  >/dev/null 2>&1 || true

# usage 落账（dm_tick 同径）
if [ -s "$USAGE_TRANSCRIPT" ] && [ -x "$USAGE_BIN" ]; then
  UT=META-CEO; [ "$N" -eq 1 ] && UT=$(printf '%s\n' "$PENDING" | head -1)
  "$USAGE_BIN" "$UT" audit "$USAGE_TRANSCRIPT" "seat=CEO;$BATCH" >/dev/null 2>&1 || true
fi
hb alive "收班——本班集 $BATCH"
