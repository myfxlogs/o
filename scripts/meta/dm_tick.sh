#!/usr/bin/env bash
# dm_tick.sh —— 决策席待裁批件拍壳（T-0312 第六刀——照 verdict_tick.sh
# 逐字对形：ops_ticket_state 待裁集水印闸>0 才烧零空拍；每拍（含空拍/
# 败拍）ops_headless CTO 心跳行；FIN_DSN 隧道径 SG 直读）。
#              T-0354 班次制三件：单例闸（PG advisory lock——kick 连发/
#              兜底拍撞车后到即退）+班次 drain（会话读 SKILL 班次节
#              scan→裁→rescan→队列空才退）+收班判（班后队列空=
#              state=exit 正常终态）；队列推导单源 dm_queue.sh（闸与
#              班内 rescan 同式防漂）。
#              GW-001 段A 瘦身：本壳=纯判断会话面（锁闸+队闸+devin+hb）；
#              零 LLM 机械尾段（depgate/补拉/suspend/handoff/ticketdraft/
#              autoapprove/ledger/verdict_deliver/部署收敛）整体迁
#              mech_tick.sh——机械动作唯一执行者=o-mech，禁双轨；
#              尾段一行唤 o-mech.service（锁分离——本锁只管会话班次，
#              机械轨自有 advisory 锁不被本锁挡）。
set -u
CTO_DIR=${O_CTO_WORKDIR:-/var/tmp/o-cto}
# T-0323 踢径钦定单目标：一切自报（含派单通报/升件）踢 O-DM 交互席；
# 派单不踢 O-Dev——常径施工拉起走 systemd-run o-worker-<tid>（T-0324）。
export REPORT_KICK=O-DM
export O_SEAT=CTO
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
DM_QUEUE_SH=${O_DM_QUEUE_SH:-"$SCRIPT_DIR/dm_queue.sh"}
OPS_ESC_SH=${O_OPS_ESC_SH:-"$SCRIPT_DIR/ops_esc.sh"}

hb() { # $1=state 心跳行（META-DM——meta 席位口径）
  [ -n "$FIN_DSN" ] || return 0
  psql "$FIN_DSN" -qAt -c "INSERT INTO ops_headless(ticket,seat,state,last_activity)
    VALUES('CTO','CTO','$1',now())
    ON CONFLICT (ticket) DO UPDATE SET seat=EXCLUDED.seat,
      state=EXCLUDED.state,last_activity=now(),updated_at=now()" >/dev/null 2>&1 || true # tolerate: 心跳落账尽力而为—败不阻拍面
}

# T-0369 容错必报首用（立法后第一改造面）；T-0372 usage_esc 同式提
# 共享件 ops_esc.sh（dm_tick/dev_run/deploy_sync 同消费面复用，禁新
# 通知栈）——usage 采集面败→ops_event 升件行自报（禁只打 stderr）。
# 同窗去重防事件暴——note 前缀为键 24h 内至多一行；dedup 查询自身败
# =空答→照常升（报侧 fail-open 不吞讯）。$1=归属票 $2=败情一句话。
usage_esc() {
  "$OPS_ESC_SH" CTO "$1" 写败 CTO-USAGE "usage 采集败" "$2" || true # tolerate: 自报末环——ops_esc 写径败静默退（结构性限度在案）
}
# T-0372 静默组同式接报：dm 内部件败（队列推导/派单查询/看门狗本体）
# →无类升件行（永不入判例面照常上行）——容错语义零动，败了仍滑行只加报。
dm_esc() {
  "$OPS_ESC_SH" CTO "" "" "" "$1" "$2" || true # tolerate: 自报末环——ops_esc 写径败静默退（结构性限度在案）
}

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
    GOT=$(cat "$DM_GATE_DIR/out" 2>/dev/null || true) # tolerate: gate out未落=续轮询合法径
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

# ① 待裁闸：无待裁时跳过 LLM。零 LLM 机械面（派发/续跑/票翻/收敛窗）
# GW-001 段A 起整体归 mech_tick 轨——本壳仅在有待裁时烧会话。
# T-0414 会话面收窄：开班判据=review 面（判断件——收口类待裁），已批待派面
# 全归机械轨（mech_tick 内 depgate 零交集即派/补拉窗/dm_ledger 票翻补登）不烧会话。
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
  # CTO 聚合行，不伪分摊）。
  PENDING_COUNT=$(printf '%s\n' "$PENDING_TICKETS" | awk 'NF {n++} END {print n+0}')
  USAGE_TICKET=CTO
  if [ "$PENDING_COUNT" -eq 1 ]; then
    USAGE_TICKET=$(printf '%s\n' "$PENDING_TICKETS" | head -1)
  fi
  BATCH_NOTE="tickets=$(printf '%s\n' "$PENDING_TICKETS" | paste -sd, -)"
  [ "$PENDING_COUNT" -ne 0 ] || BATCH_NOTE="tickets=unresolved"
  # 开班即响（纯通知腿不落账——report.sh会写ops_event领单行污染
  # 队列态;直调feishu只响铃）。移交方闭环=移交铃+开班铃两响。
  "/opt/meta/scripts/feishu-notify.sh" "[调度环] CTO 开班 队列${PENDING_COUNT}件:${PENDING_TICKETS//$'\n'/,}——已领待裁" >/dev/null 2>&1 || true # tolerate: 开班铃尽力而为—铃败不阻开班

  # ③ Devin 批拍（cwd=scratch——裁决协议 runbook 绝对路径指件）。
  mkdir -p "$CTO_DIR" || exit 1
  USAGE_DIR=$(mktemp -d "$CTO_DIR/usage.XXXXXX")
  chmod 700 "$USAGE_DIR"
  USAGE_TRANSCRIPT="$USAGE_DIR/transcript.json"
  cd "$CTO_DIR" || exit 1
  DECISION_DEVIN=${O_DECISION_DEVIN_BIN:-/home/mluser/.local/bin/devin}
  # T-0456 续跑式常驻：transcript 在案则 -c 续同 session——跨班
  # 上下文续命省冷启重读（压实丢精度险=ops_event 单源账对冲）。
  RESUME_FLAG=()
  last=$(ls -t "$CTO_DIR"/usage.*/transcript.json 2>/dev/null \
    | grep -v "$USAGE_TRANSCRIPT" | head -1 || true) # tolerate: 无旧transcript=首班合法径
  [ -n "$last" ] && [ -s "$last" ] && RESUME_FLAG=(-c)
  "$DECISION_DEVIN" --export "$USAGE_TRANSCRIPT" "${RESUME_FLAG[@]}" \
  -p --model swe --respect-workspace-trust false \
    --permission-mode dangerous -- \
    "读 /opt/o/skills/CTO席/SKILL.md 裁决协议节与班次节——你是CTO无头席,本班开始:扫描待裁决/待派工队列,逐件处理;你的活是裁决与派工,实现代码一律派给施工席,绝不自己写(含merge票内实现件);每办结一件重新扫队列,队列空才退出;动手前查该票是否已有裁决,防重复裁"
  RC=$?
  # ④ 用量采集旁路；失败只告警，不阻断裁决和事件链。
  [ ! -e "$USAGE_TRANSCRIPT" ] || chmod 600 "$USAGE_TRANSCRIPT" 2>/dev/null || true # tolerate: chmod尽力而为—不阻usage旁路
  USAGE_BIN=${O_WORKER_USAGE_BIN:-/opt/o/scripts/meta/ticket_usage.sh}
  if [ -s "$USAGE_TRANSCRIPT" ]; then
    if "$USAGE_BIN" "$USAGE_TICKET" audit CTO "$USAGE_TRANSCRIPT" "$FIN_DSN" "$BATCH_NOTE"; then
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

# GW-001 段A：零 LLM 机械尾段整体迁 mech_tick.sh（单轨——spawn/补拉/
# 票翻/推/发唯一执行者=o-mech，本壳不再具备任何机械执行能力，禁双轨）。
# 尾段一行唤机械轨：o-mech 自有 advisory 锁与本班次锁分离——本锁只管
# 判断会话，机械轨不被会话 drain 挡；唤醒失败静默（o-mech.timer 5min
# 兜底拍+事件帧双径自愈，非静默病灶）。
${O_MECH_SYSTEMCTL:-systemctl} --user start --no-block o-mech.service >/dev/null 2>&1 || true # tolerate: 唤醒尽力而为——o-mech.timer 5min 兜底拍+事件帧双径自愈，唤败不阻本壳（非静默病灶）
exit 0
