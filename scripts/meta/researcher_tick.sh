#!/usr/bin/env bash
# researcher_tick.sh —— 研究员触发消费拍（T-0504 方案A CTO 钦定）：
# ops_event seat='venue哨兵' event IN ('新品','达阈') 未终态行
# （NOT EXISTS 后至 event='消' 且 note 命中 触发#<id>——ceo_queue
# 议题#<id> 同构引证）→ paused 闸（research.researcher.paused=true
# 事件留 pending 不消费，翻键排空积压——钦定②）→ systemd-run 瞬态
# 单元 o-researcher-evt-* 拉起 researcher_run.sh → rc=0 逐事件落
# 消行（seat='研究员哨兵',event='消',note='触发#<id>…'——钦定③）。
# 拉起败=不落消行下拍重试+hb fail:launch 显形（venue_watch 败显形
# 同构——不静默绿）。空拍=hb alive 零烧。
# 单例：PG advisory lock（dm_tick FIFO-holder 同式——锁随脚本生死，
# 键 42540356 专用）。--list=只读滤队面（不占锁不消费——运维/桩测
# 直读 pending 集）。
set -u
ENVF="${RESEARCHER_TICK_ENV_FILE:-/opt/o-research/env}"
FIN_DSN="${FIN_DSN:-$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')}"
[ -n "$FIN_DSN" ] || exit 0 # tolerate: 无 DSN=哑拍退——与 venue_watch 同律（env 未就位期静默）
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
RUN_SH=${O_RESEARCHER_RUN_SH:-"$SCRIPT_DIR/researcher_run.sh"}
SYSTEMD_RUN=${O_RESEARCHER_SYSTEMD_RUN:-systemd-run}
ST=${O_RESEARCHER_TICK_ST:-/var/tmp/o-researcher-tick}; mkdir -p "$ST" 2>/dev/null || true # tolerate: ST 建败=hb/日志降级——消费判据不依赖 ST
LOCK_KEY=${O_RESEARCHER_LOCK_KEY:-42540356}

hb() { # 心跳——ops_headless 无 note 列，state=唯一自由槽（venue_watch 同构）
  [ -n "$FIN_DSN" ] || return 0
  psql "$FIN_DSN" -qAt -c \
    "INSERT INTO ops_headless(ticket,seat,state,last_activity) VALUES('META-RESEARCHER','研究员哨兵','$1',now()) ON CONFLICT (ticket) DO UPDATE SET seat=EXCLUDED.seat,state=EXCLUDED.state,last_activity=now(),updated_at=now()" \
    >/dev/null 2>&1 || true # tolerate: 心跳落账尽力而为—败不阻消费判据
}

# 滤队 SQL 单源——--list 与消费面共用（两面同判据防漂移）
PENDING_WHERE="e.seat='venue哨兵' AND e.event IN ('新品','达阈')
  AND NOT EXISTS (SELECT 1 FROM ops_event c WHERE c.id>e.id AND c.event='消'
    AND c.note ~ ('触发#' || e.id::text || '([^0-9]|$)'))"

if [ "${1:-}" = "--list" ]; then
  psql "$FIN_DSN" -qAt -F'|' -c \
    "SELECT e.id,e.seat,e.event,e.ticket,e.note,e.ts FROM ops_event e WHERE $PENDING_WHERE ORDER BY e.id" \
    2>/dev/null || true # tolerate: 只读面败=空输出——探视面非闸件
  exit 0
fi

# —— 单例闸：PG advisory lock（dm_tick FIFO-holder 同式——锁随脚本
# 生死；kick/兜底拍撞车后到即退）——
LOCK_PID=
GATE_DIR=
gate_release() {
  [ -n "$LOCK_PID" ] && kill "$LOCK_PID" 2>/dev/null
  wait "$LOCK_PID" 2>/dev/null
  [ -n "$GATE_DIR" ] && rm -rf -- "$GATE_DIR" 2>/dev/null
  return 0
}
GATE_DIR=$(mktemp -d "${TMPDIR:-/tmp}/o-rect-gate.XXXXXX") || exit 1
mkfifo "$GATE_DIR/pipe" || { rm -rf -- "$GATE_DIR"; exit 1; }
exec 8<>"$GATE_DIR/pipe" # 自持写端——EOF 不到，锁随进程生死
exec 9<"$GATE_DIR/pipe"  # psql stdin 只读面
psql "$FIN_DSN" -qAt >"$GATE_DIR/out" 2>/dev/null <&9 &
LOCK_PID=$!
printf 'SELECT pg_try_advisory_lock(%s);\n' "$LOCK_KEY" >&8
GOT=
for _ in $(seq 20); do
  GOT=$(cat "$GATE_DIR/out" 2>/dev/null || true) # tolerate: gate out未落=续轮询合法径
  [ -n "$GOT" ] && break
  sleep 0.5
done
if [ "$GOT" != "t" ]; then
  echo "[researcher-tick] 单例闸未得（拍在飞）——后到即退" >&2
  gate_release
  exit 0
fi
trap gate_release EXIT

# —— paused 闸（钦定②：paused=true 事件留 pending 不消费；键缺=
# 非 paused 照 T-0299 fail-open 语义；探测败=fail-closed 不烧会话）——
PAUSED=$(psql "$FIN_DSN" -qAt -c \
  "SELECT value::text FROM app_config WHERE key='research.researcher.paused'" 2>/dev/null) \
  || { hb "fail:paused-probe"; gate_release; exit 0; }
if [ "$PAUSED" = '"true"' ] || [ "$PAUSED" = "true" ]; then
  hb "paused"
  gate_release
  exit 0
fi

# —— 滤队（消费面同 --list 判据）——
PENDING=$(psql "$FIN_DSN" -qAt -F'|' -c \
  "SELECT e.id,e.ticket FROM ops_event e WHERE $PENDING_WHERE ORDER BY e.id" 2>/dev/null) \
  || { hb "fail:pending-probe"; gate_release; exit 0; }
N=$(printf '%s\n' "$PENDING" | awk 'NF' | wc -l)
if [ "$N" -eq 0 ]; then
  hb "alive:empty"
  gate_release
  exit 0
fi

[ -f "$RUN_SH" ] || { hb "fail:runsh-missing"; gate_release; exit 0; }

# —— 瞬态拉起（rc=0 才落消行——拉起败事件不吞下拍重试）——
IDS=$(printf '%s\n' "$PENDING" | awk -F'|' 'NF{printf "%s%s", (n++?",":""), $1}')
UNIT="o-researcher-evt-$(date +%s)-$$"
if ! "$SYSTEMD_RUN" --user --unit="$UNIT" --collect \
    --setenv="RESEARCHER_DB_DSN=${RESEARCHER_DB_DSN:-}" \
    "$RUN_SH" "$IDS" >/dev/null 2>&1; then
  hb "fail:launch"
  gate_release
  exit 0
fi

# —— 消费落账：逐事件消行（落账败=该事件仍 pending 下拍自愈重试）——
QC(){ printf '%s' "${1:-}" | sed "s/'/''/g"; }
CONSUMED=0
while IFS='|' read -r eid eticket; do
  [ -n "$eid" ] || continue
  esc_t=$(QC "$eticket")
  psql "$FIN_DSN" -qAt -c \
    "INSERT INTO ops_event(seat,event,ticket,note) VALUES('研究员哨兵','消','$esc_t','触发#$eid 已拉起 unit=$UNIT')" \
    >/dev/null 2>&1 && CONSUMED=$((CONSUMED+1)) # tolerate: 单行落账败=下拍重试该事件（消行缺席=仍 pending 自愈）
done <<< "$PENDING"
hb "launched:$CONSUMED/$N"
gate_release
exit 0
