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
# devin 自寻路（主机无关）——写死 /home/mluser 实坑 2026-11-10：SG 第二
# dispatcher 临时 unit 不继承 drop-in env,硬路径缺席→discovery 败→瞬死
# 占跨机锁饿死好机×4。env 覆写在,PATH/$HOME 兜底在,猜径=fail-closed。
DEVIN_BIN=${O_WORKER_DEVIN_BIN:-$(command -v devin 2>/dev/null || echo "$HOME/.local/bin/devin")}
RUNBOOK=${COG_RUNBOOK:-$SCRIPT_DIR/runbook.md}
[ -s "$RUNBOOK" ] || { echo "[cognition] runbook 缺: $RUNBOOK" >&2; exit 2; }

# COG_ROLE 角色化（cognition 默认|audit 审计臂——同壳两席：会话目录
# 按角色隔离即隔离 devin list 续会话面,审计绝不 --continue 唤知会话;
# 心跳键前缀/败迹 seat 随角色分账）。
COG_ROLE=${COG_ROLE:-cognition}
case "$COG_ROLE" in
	cognition) SEAT=唤知席; SEAT_MACHINE=唤知机; HB_PREFIX=COG- ;;
	audit)     SEAT=审计席; SEAT_MACHINE=审计机; HB_PREFIX=AUD- ;;
	*) echo "[cognition] COG_ROLE 非法: $COG_ROLE（cognition|audit）" >&2; exit 2 ;;
esac

export O_SEAT=$SEAT
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')

# 唤起果/唤起败释锁 trap（跨机锁=ops_event 唤起行在案即占,早退不补果败迹=
# 占满 30min TTL 饿死他机——2026-11-10 四连败实证）。OK=会话完整跑完
# (RC=0∧transcript 非空——devin 瞬退零消息但 exit 0 的静默败也算败)。
# 唤起行必配果/败行（坑处置: 唤起断尾幻影根修——会话判词不落账=唤起
# 无产出=假死象,排查须跨机翻 transcript;壳兜底行=唤起存在性对账,
# 会话自写会话收口=判词语义层,两层不替）。残余黑洞=SIGKILL/宕机写不了行,
# 由锁 TTL 兜底（30min 一次非无限）。
COG_UNIT=${COG_UNIT:-manual}
HOST_TAG=$(hostname 2>/dev/null || echo '?')
T0=$(date +%s)
OK=0; RC=0; REASON=session_incomplete
release_lock() {
	[ -n "${FIN_DSN:-}" ] || return 0
	local dur=$(( $(date +%s) - T0 )) ev=唤起果
	[ "$OK" = 1 ] || ev=唤起败
	psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('$SEAT_MACHINE','$ev','','unit=$COG_UNIT host=$HOST_TAG rc=$RC reason=$REASON dur=${dur}s')" >/dev/null 2>&1 || true # tolerate: 果败写径本身是败迹——写败无二级上报径,锁由 30min TTL 兜底,不阻主径退出
}
trap 'rm -f ${LOG_FILE:+"$LOG_FILE"}; release_lock' EXIT

hb() { # $1=state——心跳按日键（会话逐日新面,互不盖；前缀随角色）
  [ -n "$FIN_DSN" ] || return 0
  psql "$FIN_DSN" -qAt -c "INSERT INTO ops_headless(ticket,seat,state,last_activity)
    VALUES('${HB_PREFIX}$(date +%Y%m%d)','$SEAT','$1',now())
    ON CONFLICT (ticket) DO UPDATE SET seat=EXCLUDED.seat,
      state=EXCLUDED.state,last_activity=now(),updated_at=now()" >/dev/null 2>&1 || true # tolerate: 心跳=尽力径——hb写败不阻会话主径（dev_run.sh 先例同构）
}

SESSION_DIR="$(o_worker_runtime_dir)/sessions/$COG_ROLE"
mkdir -p "$SESSION_DIR" && chmod 700 "$SESSION_DIR" && cd "$SESSION_DIR" || exit 1

if quota_reset=$(o_worker_quota_until); then
	echo "[cognition] quota cooldown active until epoch $quota_reset"
	REASON=quota_cooldown; hb stalled; exit 0
else
	quota_rc=$?
	[ "$quota_rc" -ne 1 ] && { echo "[cognition] quota state unreadable; refusing" >&2; REASON=quota_unreadable; hb stalled; exit 1; }
fi

SESSION_STATE=$(o_worker_session_state "$DEVIN_BIN")
[ $? -ne 0 ] && { echo "[cognition] session discovery failed; refusing cold restart" >&2; REASON=session_discovery; hb stalled; exit 1; }
hb alive

LOG_FILE=$(mktemp "$(o_worker_runtime_dir)/cognition.XXXXXX")
chmod 600 "$LOG_FILE"
DEVIN_ARGS=()
case "$SESSION_STATE" in
	resume) DEVIN_ARGS+=(--continue) ;;
	new) ;;
	*) echo "[cognition] session discovery failed; refusing" >&2; REASON=session_state_invalid; hb stalled; exit 1 ;;
esac

# KB 召回（T-0521：经验库只写不读根修）——recall.sh 按未决升件词面命中
# 条目挂 prompt 尾：每场会话起手头已带命中案卷。败=空串静默（fail-open
# 不阻主径）——会话见块照口径读件,无块=本拍零命中非库坏。
if [ "$COG_ROLE" = audit ] && [ -n "${COG_RECALL_WORDS:-}" ]; then
	RECALL=$("$SCRIPT_DIR/recall.sh" -w "$COG_RECALL_WORDS" 2>/dev/null || true) # tolerate: 同上
else
	RECALL=$("$SCRIPT_DIR/recall.sh" 2>/dev/null || true) # tolerate: 召回败=知识面缺非闸面——空串静默不阻会话主径（runbook ①½ 生成面兜底）
fi
PROMPT=""
# COG_SCOPE 审计域注入（dispatch 塞的域牌——非空即冠 prompt 首）。
[ -n "${COG_SCOPE:-}" ] && PROMPT="## 本场审计域（dispatch 注入）
scope=$COG_SCOPE

"
PROMPT="$PROMPT$(cat "$RUNBOOK")"
[ -n "$RECALL" ] && PROMPT="$PROMPT

## KB 召回案卷（自动注入——按本拍未决升件词面命中 TOP8）
先读命中条目再判案——案卷已按 scope/class/域词命中排在先；
条目是判例 canon,命中即照口径,不符再查库。件径=部署树正本。
$RECALL"

TRANSCRIPT_FILE=$(mktemp "$SESSION_DIR/.devin-usage.XXXXXX.json")
chmod 600 "$TRANSCRIPT_FILE"
# 活性心跳续命（坑处置: 跨机锁30min TTL<RuntimeMaxSec 2h——长会话唤起行
# 过期即失在飞保护,双机双会话实证 2026-11-12。60s 续期保
# last_activity 鲜活=dispatch 跨机在飞判据;devin 主径阻塞期后台跑,
# kill 于主径返回后——会话死透心跳即停,陈旧心跳≈死证自清）。
( while :; do sleep 60; hb alive; done ) & HB_LOOP_PID=$!
set +e
"$DEVIN_BIN" "${DEVIN_ARGS[@]}" --export "$TRANSCRIPT_FILE" -p --model "$MODEL" \
	--respect-workspace-trust false --permission-mode dangerous -- "$PROMPT" 2>&1 | tee "$LOG_FILE"
RC=${PIPESTATUS[0]}
kill "$HB_LOOP_PID" 2>/dev/null || true # tolerate: 心跳环随主径收——环死/已死不阻收口
set -e

if o_worker_is_rate_limited "$LOG_FILE"; then
	quota_reset=$(o_worker_record_quota_reset "$LOG_FILE")
	echo "[cognition] rate limit; cooldown until epoch $quota_reset"
	REASON=rate_limited; RC=1
fi
# OK=完整跑完——devin exit 0 但 transcript 空=零消息静默败（占锁同害算败）。
[ $RC -eq 0 ] && [ -s "$TRANSCRIPT_FILE" ] && OK=1 || REASON=${REASON/session_incomplete/devin_rc_$RC}
[ -s "$TRANSCRIPT_FILE" ] || rm -f "$TRANSCRIPT_FILE"
[ "$OK" = 1 ] && hb done || hb dead
exit 0
