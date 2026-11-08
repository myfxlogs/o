#!/usr/bin/env bash
# ceo_tick.sh —— CEO 席无头班拍（T-0456 业主令双无头建席）：
# 照 dm_tick.sh 同构三件——PG advisory 单例闸（kick 连发/兜底拍撞车
# 后到即退）+北极星账先扫（T-0465：ceo_northstar.sh 账→议题→队序，
# 背离自生成议题行入队）+队列闸（ceo_queue.sh 空集=心跳即退零烧）
# +班次 drain
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
    >/dev/null 2>&1 || true # tolerate: 心跳落账尽力而为—败不阻锁拍
}

# 单例闸（PG advisory——kick 连发撞车后到即退）
LOCK_KEY=884201
LOCKRC=0
LOCKED=$(psql "$FIN_DSN" -qAt -c \
  "SELECT pg_try_advisory_lock($LOCK_KEY)" 2>/dev/null) || LOCKRC=$?
[ "$LOCKRC" -ne 0 ] && { hb err "锁探败 rc=$LOCKRC 按有主退"; exit 0; }
[ "$LOCKED" != "t" ] && exit 0  # 有班在飞——后到即退
trap 'psql "$FIN_DSN" -qAt -c "SELECT pg_advisory_unlock($LOCK_KEY)" >/dev/null 2>&1 || true' EXIT # tolerate: trap解锁尽力而为—锁已释或会话断合法径

# 北极星账先扫（T-0465 钦定三步序：账→议题→队）——七项指标直查+硬阈
# 背离落议题行；议题行=ceo_queue 班内件源，背离自生成队件点火原空拍。
NORTHSTAR_SH=${O_CEO_NORTHSTAR_SH:-"$SCRIPT_DIR/ceo_northstar.sh"}
[ -f "$NORTHSTAR_SH" ] && "$NORTHSTAR_SH" || true # tolerate: 账页扫败不阻开班——队闸照旧判（扫件内建 rc 恒 0，败因载页/行）

# 队列闸——空集=心跳即退零烧会话
PENDING=$(FIN_DSN="$FIN_DSN" "$CEO_QUEUE_SH" 2>/dev/null || true) # tolerate: 队列脚本败=队空视同,开班照判
N=$(printf '%s\n' "$PENDING" | awk 'NF {n++} END {print n+0}')
if [ "$N" -eq 0 ]; then
  hb alive "空拍——CEO 队列空"
  exit 0
fi
# 开班即响——纯通知腿不落账（同 dm_tick 口径）。
"/opt/meta/scripts/feishu-notify.sh" "[调度环] CEO 开班 队列${N}件:${PENDING//$'\n'/,}——已领待裁" >/dev/null 2>&1 || true # tolerate: 开班铃尽力而为—铃败不阻开班

# 开班——续跑式：transcript 在案则 -c 续同 session（新席首班自然无档全启）
mkdir -p "$CEO_DIR" || exit 1
USAGE_DIR=$(mktemp -d "$CEO_DIR/usage.XXXXXX"); chmod 700 "$USAGE_DIR"
USAGE_TRANSCRIPT="$USAGE_DIR/transcript.json"
cd "$CEO_DIR" || exit 1
DEVIN_BIN=${O_DECISION_DEVIN_BIN:-/home/mluser/.local/bin/devin}
RESUME_FLAG=()
last=$(ls -t "$CEO_DIR"/usage.*/transcript.json 2>/dev/null | grep -v "$USAGE_TRANSCRIPT" | head -1 || true) # tolerate: 无旧transcript=首班合法径
[ -n "$last" ] && [ -s "$last" ] && RESUME_FLAG=(-c)
BATCH="tickets=$(printf '%s\n' "$PENDING" | paste -sd, -)"
"$DEVIN_BIN" --export "$USAGE_TRANSCRIPT" "${RESUME_FLAG[@]}" \
  -p --model swe --respect-workspace-trust false \
  --permission-mode dangerous -- \
  "读 /opt/o/skills/CEO席/SKILL.md——你是CEO无头席,本班开始:扫描CEO队列($BATCH)逐件处理;方向/批准类件直接裁决并落 ops_event 行(seat=CEO);机械/执行类一律写移交事件转CTO,不要代办;超出职权或红线类写呈批升级给业主,不要自行裁;队列中 议题#N 件=北极星账自生成背离件——读 /var/tmp/o-northstar/northstar.txt 对位指标,处置落 ops_event 行(event=处置,note 含「议题#N」引证+结论)即出队,归因诊断已由问链立案走裁定席不代诊;全部办完后重扫一次队列,空了才退出" \
  >/dev/null 2>&1 || true # tolerate: 会话rc不取—输出已落transcript

# usage 落账（dm_tick 同径）
if [ -s "$USAGE_TRANSCRIPT" ] && [ -x "$USAGE_BIN" ]; then
  UT=META-CEO; [ "$N" -eq 1 ] && UT=$(printf '%s\n' "$PENDING" | head -1)
  "$USAGE_BIN" "$UT" audit "$USAGE_TRANSCRIPT" "seat=CEO;$BATCH" >/dev/null 2>&1 || true # tolerate: usage审计旁路尽力而为—不阻裁决
fi
hb alive "收班——本班集 $BATCH"
