#!/usr/bin/env bash
# dm_tick.sh —— 决策席待裁批件拍壳（T-0312 第六刀——照 verdict_tick.sh
# 逐字对形：ops_ticket_state 待裁集水印闸>0 才烧零空拍；每拍（含空拍/
# 败拍）ops_headless META-DM 心跳行；FIN_DSN 隧道径 SG 直读）。
#              T-0354 班次制三件：单例闸（PG advisory lock——kick 连发/
#              兜底拍撞车后到即退）+班次 drain（会话读 SKILL 班次节
#              scan→裁→rescan→队列空才退）+收班判（班后队列空=
#              state=exit 正常终态）；队列推导单源 dm_queue.sh（闸与
#              班内 rescan 同式防漂）。
set -u
DECISION_DIR=${O_DECISION_WORKDIR:-/var/tmp/o-decision}
# T-0323 踢径钦定单目标：一切自报（含派单通报/升件）踢 O-DM 交互席；
# 派单不踢 O-Dev——常径施工拉起走 systemd-run o-worker-<tid>（T-0324）。
export REPORT_KICK=O-DM
export O_SEAT=决策席
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
DM_QUEUE_SH=${O_DM_QUEUE_SH:-"$SCRIPT_DIR/dm_queue.sh"}
DM_DEPGATE_SH=${O_DM_DEPGATE_SH:-"$SCRIPT_DIR/dm_depgate.sh"}
OPS_ESC_SH=${O_OPS_ESC_SH:-"$SCRIPT_DIR/ops_esc.sh"}

hb() { # $1=state 心跳行（META-DM——meta 席位口径）
  [ -n "$FIN_DSN" ] || return 0
  psql "$FIN_DSN" -qAt -c "INSERT INTO ops_headless(ticket,seat,state,last_activity)
    VALUES('META-DM','决策席','$1',now())
    ON CONFLICT (ticket) DO UPDATE SET seat=EXCLUDED.seat,
      state=EXCLUDED.state,last_activity=now(),updated_at=now()" >/dev/null 2>&1 || true
}

# T-0369 容错必报首用（立法后第一改造面）；T-0372 usage_esc 同式提
# 共享件 ops_esc.sh（dm_tick/dev_run/deploy_sync 同消费面复用，禁新
# 通知栈）——usage 采集面败→ops_event 升件行自报（禁只打 stderr）。
# 同窗去重防事件暴——note 前缀为键 24h 内至多一行；dedup 查询自身败
# =空答→照常升（报侧 fail-open 不吞讯）。$1=归属票 $2=败情一句话。
usage_esc() {
  "$OPS_ESC_SH" META-DM "$1" 写败 META-DM-USAGE "usage 采集败" "$2" || true # tolerate: 自报末环——ops_esc 写径败静默退（结构性限度在案）
}
# T-0372 静默组同式接报：dm 内部件败（队列推导/派单查询/看门狗本体）
# →无类升件行（永不入判例面照常上行）——容错语义零动，败了仍滑行只加报。
dm_esc() {
  "$OPS_ESC_SH" META-DM "" "" "" "$1" "$2" || true # tolerate: 自报末环——ops_esc 写径败静默退（结构性限度在案）
}

# T-0413 派单依赖闸必须在班次锁前跑：kick 撞在飞班次时，零交集已批件
# 仍可直派；交集/不可判件不进派单链，锁拒后照常留班内排队等裁决。
if [ -n "$FIN_DSN" ]; then
  DEPGATE_RC=0
  "$DM_DEPGATE_SH" || DEPGATE_RC=$?
  [ "$DEPGATE_RC" -ne 0 ] && dm_esc "派单依赖闸败" "rc=$DEPGATE_RC 已批快派面不可知按常径排队"
fi

# —— T-0354 件③ 单例闸：PG advisory lock（int8 专键）。holder=psql 连接
# 挂 FIFO：fd8=O_RDWR 自持写端永不 EOF（锁随脚本生死）；psql stdin 走
# fd9=O_RDONLY 只读面——写端只归 fd8，脚本被杀（EXIT trap 不及）即
# EOF→psql 退→PG 会话断→锁自放，防孤儿 psql 永持锁闷死全席。
# kick 连发与兜底拍撞车=后到锁拒即退（在飞班 drain 吸收新事件）。
# FIN_DSN 缺=无从持锁退化现状直行（与 kick 哑同律）；闸答非 t（含连接
# 败超时）=退——kick 是纯加速件，下班承接不丢票。
DM_LOCK_KEY=${O_DM_LOCK_KEY:-42540354}
DM_LOCK_PID=
DM_GATE_DIR=
dm_gate_release() {
  [ -n "$DM_LOCK_PID" ] && kill "$DM_LOCK_PID" 2>/dev/null
  wait "$DM_LOCK_PID" 2>/dev/null
  [ -n "$DM_GATE_DIR" ] && rm -rf -- "$DM_GATE_DIR" 2>/dev/null
  return 0
}
if [ -n "$FIN_DSN" ]; then
  DM_GATE_DIR=$(mktemp -d "${TMPDIR:-/tmp}/o-dm-gate.XXXXXX") || exit 1
  mkfifo "$DM_GATE_DIR/pipe" || { rm -rf -- "$DM_GATE_DIR"; exit 1; }
  exec 8<>"$DM_GATE_DIR/pipe" # 自持写端——EOF 不到，锁随进程生死
  exec 9<"$DM_GATE_DIR/pipe"  # psql stdin 只读面（fd8=O_RDWR 已在=writer 不阻塞）
  psql "$FIN_DSN" -qAt >"$DM_GATE_DIR/out" 2>/dev/null <&9 &
  DM_LOCK_PID=$!
  printf 'SELECT pg_try_advisory_lock(%s);\n' "$DM_LOCK_KEY" >&8
  GOT=
  for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do
    GOT=$(cat "$DM_GATE_DIR/out" 2>/dev/null || true)
    [ -n "$GOT" ] && break
    sleep 0.5
  done
  if [ "$GOT" != "t" ]; then
    echo "[dm] 班次闸未得（班次在飞/kick 连发）——后到即退" >&2
    dm_gate_release
    exit 0
  fi
  trap dm_gate_release EXIT
fi

# ① 待裁闸：无待裁时跳过 LLM，但仍运行下方零 LLM 的 worker 派发/续跑闸。
# T-0414 会话面收窄：开班判据=review 面（判断件——收口类待裁），已批待派面
# 全归壳轨（dm_depgate 零交集即派/补拉窗/dm_ledger 票翻补登）不烧会话。
SKIP_DECISION=0
PENDING_TICKETS=
if [ -n "$FIN_DSN" ]; then
  # T-0315②/T-0347 判据原文单源 dm_queue.sh——拍壳闸与会话班内 rescan
  # 同式（判据细则见该件头注：收口类最新刻无终态裁决，或已批未进派单链）。
  QRC=0
  PENDING_TICKETS=$("$DM_QUEUE_SH" review 2>/dev/null) || QRC=$?
  [ "$QRC" -ne 0 ] && dm_esc "队列推导败" "rc=$QRC 待裁集不可知按空集滑行"
  N=$(printf '%s\n' "$PENDING_TICKETS" | awk 'NF {n++} END {print n+0}')
  if [ "$N" -eq 0 ]; then
    hb alive
    SKIP_DECISION=1
  fi
fi

if [ "$SKIP_DECISION" -eq 0 ]; then
  # ② 固定本拍票集供 token 行归属（dm_queue.sh 已去重——多票一批记
  # META-DM 聚合行，不伪分摊）。
  PENDING_COUNT=$(printf '%s\n' "$PENDING_TICKETS" | awk 'NF {n++} END {print n+0}')
  USAGE_TICKET=META-DM
  if [ "$PENDING_COUNT" -eq 1 ]; then
    USAGE_TICKET=$(printf '%s\n' "$PENDING_TICKETS" | head -1)
  fi
  BATCH_NOTE="tickets=$(printf '%s\n' "$PENDING_TICKETS" | paste -sd, -)"
  [ "$PENDING_COUNT" -ne 0 ] || BATCH_NOTE="tickets=unresolved"

  # ③ Devin 批拍（cwd=scratch——裁决协议 runbook 绝对路径指件）。
  mkdir -p "$DECISION_DIR" || exit 1
  USAGE_DIR=$(mktemp -d "$DECISION_DIR/usage.XXXXXX")
  chmod 700 "$USAGE_DIR"
  USAGE_TRANSCRIPT="$USAGE_DIR/transcript.json"
  cd "$DECISION_DIR" || exit 1
  DECISION_DEVIN=${O_DECISION_DEVIN_BIN:-/home/mluser/.local/bin/devin}
  "$DECISION_DEVIN" --export "$USAGE_TRANSCRIPT" -p --model swe --respect-workspace-trust false \
    --permission-mode dangerous -- \
    "读 /opt/o/skills/决策席/SKILL.md 裁决协议节+班次节 班次开工——scan 待裁/待派集→逐件裁→rescan→队列空才退；每票动手前查重裁幂等"
  RC=$?
  # ④ 用量采集旁路；失败只告警，不阻断裁决和事件链。
  [ ! -e "$USAGE_TRANSCRIPT" ] || chmod 600 "$USAGE_TRANSCRIPT" 2>/dev/null || true
  USAGE_BIN=${O_WORKER_USAGE_BIN:-/opt/o/scripts/meta/ticket_usage.sh}
  if [ -s "$USAGE_TRANSCRIPT" ]; then
    if "$USAGE_BIN" "$USAGE_TICKET" audit 决策席 "$USAGE_TRANSCRIPT" "$FIN_DSN" "$BATCH_NOTE"; then
      rm -rf -- "$USAGE_DIR"
    else
      echo "[dm] token usage collection failed; continuing decision lifecycle; transcript kept=$USAGE_DIR" >&2
      usage_esc "$USAGE_TICKET" "写径败 transcript kept=$USAGE_DIR"
    fi
  else
    echo "[dm] token transcript missing/empty; continuing decision lifecycle" >&2
    usage_esc "$USAGE_TICKET" "transcript 缺/空——export 未产"
    rm -rf -- "$USAGE_DIR"
  fi
  # ⑤ 每拍心跳全件（成败同写——败拍=stalled 显形非隐）。
  #    T-0354 收班判：RC=0 且班后队列空=班次干净自退→state=exit（fin_0093
  #    六态词表正常终态）；队列仍非空=本班未尽，留 alive 待下班（kick/
  #    兜底拍续班）。
  QRC2=0; QRECHECK=
  if [ $RC -eq 0 ]; then QRECHECK=$("$DM_QUEUE_SH" review 2>/dev/null) || QRC2=$?; fi
  [ "$QRC2" -ne 0 ] && dm_esc "队列推导败" "rc=$QRC2 drain复核败——班后队列不可知按空集判退（容错语义不动）"
  if [ $RC -eq 0 ] && [ -z "$QRECHECK" ]; then
    hb exit
  else
    [ $RC -eq 0 ] && hb alive || hb stalled
  fi
fi

# T-0343：常径派单和 rate-limit 挂起续跑共用 dispatch_worker.sh。仅最新
# 决策席派单或施工席 rate-limit:auto 挂起可补拉；其他挂起/异常不自动重试。
# 模型档从原派单备注恢复。补拉成功写 `拉起` 行自标幂等；无票文件免拉。
if [ -n "$FIN_DSN" ]; then
  DISPATCH_WORKER=${O_WORKER_DISPATCH_SCRIPT:-/opt/o/scripts/meta/dispatch_worker.sh}
  DISPATCH_TASK_DIR=${O_WORKER_TASK_DIR:-/opt/o/docs/tasks}
  DRC=0
  DISPATCH_ROWS=$(psql "$FIN_DSN" -qAt -F '|' -c "SELECT l.ticket || '|' || COALESCE((
      SELECT CASE WHEN e.note ILIKE '%swe-2-medium%' THEN 'swe-2-medium'
                  ELSE 'swe-2-high' END
      FROM ops_event e WHERE e.ticket=l.ticket AND e.event='派单'
      ORDER BY e.ts DESC,e.id DESC LIMIT 1
    ),'swe-2-high') FROM (
      SELECT ticket,max(ts) mt FROM ops_event WHERE event IN
        ('派单','领单','拉起','收口','移交','收口待审','裁决','完成','驳回',
         '挂起','异常')
        AND ticket<>'' GROUP BY ticket) l
    WHERE EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
        AND e.ts=l.mt AND e.event='派单' AND e.seat='决策席')
       OR EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
        AND e.ts=l.mt AND e.event='挂起' AND e.seat='施工席'
        AND e.note LIKE 'rate-limit:auto%')" 2>/dev/null) || DRC=$?
  [ "$DRC" -ne 0 ] && dm_esc "派单查询败" "rc=$DRC 派单/续跑候选集不可知按空集滑行"
  printf '%s\n' "$DISPATCH_ROWS" | while IFS='|' read -r tid model; do
    compgen -G "$DISPATCH_TASK_DIR/${tid}-*.md" >/dev/null || continue # ③无票免拉
    # T-0438 件⑥ 人工径滤（坑条目 real-money-ticket-machine-dispatch-waste
    # 律②——补派件把它踢出候选语义同域：票面「执行=人工径」出生行=机器
    # 永不可为，补拉窗不续派（不派不挂起不计漏拉））。
    MANUAL_LANE_SH=${O_MANUAL_LANE_SH:-"$SCRIPT_DIR/manual_lane_check.sh"}
    if "$MANUAL_LANE_SH" "$tid" "$DISPATCH_TASK_DIR"; then continue; fi # rc0=人工径票滤除；rc1/2=照旧
    st=$(systemctl --user is-active "o-worker-$tid.service" 2>/dev/null || true)
    case "$st" in active|activating) continue;; esac # 精确比对误配恒死修
    prompt="读 /opt/o/skills/施工席/SKILL.md 施工协议领单/续领 $tid；按票面施工，同票已有 Devin 会话/worktree 时续跑"
    if "$DISPATCH_WORKER" "$tid" "$model" "$prompt" >/dev/null 2>&1; then
      psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
        VALUES('META-DM','拉起','$tid','漏拉/限流补派——session 续跑')" >/dev/null 2>&1 || true
    else
      rc=$?
      [ "$rc" -eq 75 ] && break # 并行水位/额度冷却——队列留待下拍
      echo "[dm] worker dispatch failed tid=$tid rc=$rc" >&2
    fi
  done
fi

# T-0365 挂起黑洞看门狗消费窗（零 LLM 段尾——消费面≠状态机：挂起不迁
# 相位，只多开一扇扫描窗）。未消化挂起行机械二择：await=绿窗 档复跑
# 测试已消→dispatch_worker 续跑（note「挂起续跑」）；未消且>4h→升件
# 呈业主。rate-limit:auto 挂起归上段既有窗不重复消费。
if [ -n "$FIN_DSN" ]; then
  DM_SUSPEND_SH=${O_DM_SUSPEND_SH:-"$SCRIPT_DIR/dm_suspend.sh"}
  SRC=0
  "$DM_SUSPEND_SH" || SRC=$?
  [ "$SRC" -ne 0 ] && dm_esc "看门狗本体败" "rc=$SRC dm_suspend 消费窗未跑净——挂起行消化停摆"
fi

# T-0393 候选票草案物料化窗（零 LLM 段尾——fin_ticket_draft pending→
# docs/tasks 候选票件+起草行；写径只落候选态零自派，批准权=COS/DM）。
# 败=升件自报（写败/META-DRAFT——类册面不入起草拍签名域防自起草回环）。
if [ -n "$FIN_DSN" ]; then
  TRC=0
  (cd /opt/o && FIN_PG_DSN="$FIN_DSN" go run ./tools/ticketdraft -repo /opt/o >/dev/null 2>&1) || TRC=$?
  [ "$TRC" -ne 0 ] && dm_esc "物料化径败" "rc=$TRC ticketdraft 未跑净——pending 草案滞留（升件面工具内自报，本行=拍壳兜底）"
fi

# T-0414 机械轨尾窗（零 LLM 段尾——票翻/台账补登壳轨件：dm_ledger.sh）。
# 位置义：kick 直驱不走会话——ops_event 新行即办机械活；会话判词刚落的
# 票在同拍尾窗即归档补登（零会话等候）。败=升件自报不静默。
if [ -n "$FIN_DSN" ]; then
  DM_LEDGER_SH=${O_DM_LEDGER_SH:-"$SCRIPT_DIR/dm_ledger.sh"}
  LRC=0
  "$DM_LEDGER_SH" || LRC=$?
  [ "$LRC" -ne 0 ] && dm_esc "机械轨本体败" "rc=$LRC dm_ledger 未跑净——票翻/台账补登滑行"
fi

# T-0438 判词送达尾窗（零 LLM 段尾——verdict_deliver.sh：需业主动作判词
# 同拍转 fin_condition_gap 请求箱+feishu 推送；坑条目 real-money-ticket-
# machine-dispatch-waste 律④「裁了≠送到了」机内化——会话判词刚落的票
# 同拍尾窗即送达零等候）。败=升件自报不静默（rc3=有未闭环件下拍重试）。
if [ -n "$FIN_DSN" ]; then
  VERDICT_DELIVER_SH=${O_VERDICT_DELIVER_SH:-"$SCRIPT_DIR/verdict_deliver.sh"}
  VDRC=0
  "$VERDICT_DELIVER_SH" || VDRC=$?
  [ "$VDRC" -ne 0 ] && dm_esc "判词送达闸败" "rc=$VDRC 需业主动作判词滞留未闭环（未闭环件下拍重试）"
fi
exit 0
