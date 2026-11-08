#!/usr/bin/env bash
# researcher_run.sh <id,id,...> —— 研究员事件触发会话本体（T-0504
# 方案A）：researcher_tick 瞬态单元（o-researcher-evt-*）拉起。
# 单实例互斥（flock——tick 消费面已锁，此为会话面双保险）+
# ops_event 按 id 直查取 payload（不读 pending 滤队——消行可能已落）
# +prompt 装配（契约指针+事件块）+env 卫生剥 ANTHROPIC_*+timeout
# 墙钟 2700s（契约 30min 帽+宽限）+claude -p --settings 钉死
# DeepSeek（只读引用 meta-archive 凭据件——零复制零入 git/日志）。
# DB_DSN=${RESEARCHER_DB_DSN:-} 缺省空=数据面缺席——会话按契约如实
# 回执不假装（T-0295 归正前卡写径死是已知挂起缺口，非本件修面）。
set -u
IDS=${1:-}
[[ "$IDS" =~ ^[0-9]+(,[0-9]+)*$ ]] || { echo "用法: researcher_run.sh <id,id,...>" >&2; exit 2; }

ENVF="${RESEARCHER_RUN_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
PROMPT_FILE=${O_RESEARCHER_EVENT_PROMPT:-"$SCRIPT_DIR/researcher_event_prompt.md"}
SETTINGS=${O_RESEARCHER_SETTINGS:-/opt/meta-archive/bin/researcher-deepseek.settings.json}
CLAUDE_BIN=${O_RESEARCHER_CLAUDE_BIN:-claude}
ST=${O_RESEARCHER_TICK_ST:-/var/tmp/o-researcher-tick}
WORK="$ST/session-$(date +%s)-$$"; mkdir -p "$WORK" 2>/dev/null || WORK=/tmp
LOG="$WORK/session.log"
WALL=${RESEARCHER_WALL:-2700}
LOCK=/tmp/researcher-event.lock

# 单实例互斥（并发触发=副本让位 rc=0——非故障，同 spawn 径语义）
exec 8>"$LOCK"
flock -n 8 || { echo "[$(date '+%F %T')] 让位：研究员会话在飞 tag=evt-$IDS" >> "$LOG" 2>/dev/null || true; exit 0; } # tolerate: yield 留痕尽力而为—锁主日志在案

[ -r "$SETTINGS" ] || { echo "[$(date '+%F %T')] FAIL settings 不可读 $SETTINGS" >> "$LOG"; exit 1; }
[ -r "$PROMPT_FILE" ] || { echo "[$(date '+%F %T')] FAIL prompt 不可读 $PROMPT_FILE" >> "$LOG"; exit 1; }

# payload 取证：ops_event 按 id 直查（消行已落亦可读——按 id 不按 pending）
EVENTS=""
if [ -n "$FIN_DSN" ]; then
  EVENTS=$(psql "$FIN_DSN" -qAt -F'|' -c \
    "SELECT id,seat,event,ticket,note,ts FROM ops_event WHERE id = ANY(string_to_array('$IDS',',')::bigint[]) ORDER BY id" \
    2>/dev/null || true) # tolerate: payload 读败=空块——prompt 内如实记缺证会话仍起（事件 id 在参）
fi
[ -n "$EVENTS" ] || EVENTS="(payload 读败——事件 id=$IDS，ops_event 直查无回；按 id 缺证会话起跑，产出如实记缺证)"

# prompt 装配：模板 __RESEARCH_EVENTS__ 占位换事件块（事件走文件——
# awk -v 会吃反斜杠转义，note 含 \ 实体须原样透传）
printf '%s\n' "$EVENTS" > "$WORK/events.txt"
PROMPT=$(awk 'FNR==NR{ev=ev $0 ORS; next} {if(index($0,"__RESEARCH_EVENTS__")) printf "%s", ev; else print}' \
  "$WORK/events.txt" "$PROMPT_FILE")

export PATH="$HOME/.local/bin:$HOME/.npm-global/bin:/usr/local/go/bin:$PATH" # systemd 极简 PATH（坑13 同 spawn 径）
export DB_DSN="${RESEARCHER_DB_DSN:-}"
export O_SEAT=Researcher
DSN_STATE=unset; [ -n "$DB_DSN" ] && DSN_STATE=set
echo "[$(date '+%F %T')] researcher event session ids=$IDS wall=${WALL}s dsn=$DSN_STATE" >> "$LOG"
cd "$WORK" || exit 1
setsid env -u ANTHROPIC_MODEL -u CLAUDE_MODEL -u ANTHROPIC_SMALL_FAST_MODEL \
  -u ANTHROPIC_DEFAULT_SONNET_MODEL -u ANTHROPIC_DEFAULT_OPUS_MODEL \
  -u ANTHROPIC_DEFAULT_HAIKU_MODEL \
  timeout --signal=TERM --kill-after=30s "$WALL" \
  "$CLAUDE_BIN" -p --dangerously-skip-permissions --settings "$SETTINGS" \
  "$PROMPT" </dev/null >> "$LOG" 2>&1 8>&-
RC=$?
echo "[$(date '+%F %T')] researcher event 会话退出 rc=$RC ids=$IDS log=$LOG" >> "$LOG"
exit "$RC"
