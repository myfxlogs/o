#!/usr/bin/env bash
# mech_tick.sh —— 机械轨拍壳（GW-001 段A：dm_tick 零 LLM 尾段整体承接
# +部署收敛窗——机械动作唯一执行者；dm_tick 退化纯判断会话壳，禁双轨）。
#
# 承接面（子件不改，原 dm_tick.sh 尾段逐字搬位）：
#   dm_depgate.sh 依赖闸即派 → 派单补拉窗（dispatch_worker.sh 共用闸）
#   → dm_suspend.sh 挂起看门狗 → dm_handoff.sh 交接哨兵
#   → ticketdraft 物料化 → dm_autoapprove.sh 机批闸 → dm_ledger.sh 票翻
#   → verdict_deliver.sh 判词送达。
# 追加收敛窗（唯一新逻辑）：deploy_window_due.sh state=应发→faces= 执行
#   image→deploy_image.sh / oci→deploy_oci.sh；幂等=ops_event 部署行
#   'deploy <sha12> rc=0'/'deploy-oci <sha12> rc=0' 查重；面败=ops_esc
#   DEPLOY-FIRE-FAIL:<sha12>:<face>+本面冻结 24h（窗内同面不重试）。
#   判据全在 deploy_window_due（裁「该不该发」），本窗只读 state 执行。
#
# 单例：自有 PG advisory lock（O_MECH_LOCK_KEY——与 CTO 会话锁
# 42540354 分离，锁分离=机械轨不被判断会话挡）；拿不到=退。
# 心跳：每跑拍 ops_headless seat='MECH'（gauges mech_hb 量表锚）。
set -u
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
# systemd user PATH 无 /usr/local/go/bin——go run/go test 子件（ticketdraft/
# dm_suspend 绿窗复跑）在单元内需补径；尾追=PATH 桩序不遮（原 dm_tick
# 同 PATH 同款潜伏病灶——升件 24h 去重掩盖未显形，随轨迁移在此根修在案）。
case ":$PATH:" in *:/usr/local/go/bin:*) ;; *) PATH="$PATH:/usr/local/go/bin";; esac
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO=${O_MECH_REPO:-/opt/o}
OPS_ESC_SH=${O_OPS_ESC_SH:-"$SCRIPT_DIR/ops_esc.sh"}
DM_DEPGATE_SH=${O_DM_DEPGATE_SH:-"$SCRIPT_DIR/dm_depgate.sh"}
DM_SUSPEND_SH=${O_DM_SUSPEND_SH:-"$SCRIPT_DIR/dm_suspend.sh"}
DM_HANDOFF_SH=${O_DM_HANDOFF_SH:-"$SCRIPT_DIR/dm_handoff.sh"}
DM_LEDGER_SH=${O_DM_LEDGER_SH:-"$SCRIPT_DIR/dm_ledger.sh"}
VERDICT_DELIVER_SH=${O_VERDICT_DELIVER_SH:-"$SCRIPT_DIR/verdict_deliver.sh"}
AUTOAPPROVE_SH=${O_AUTOAPPROVE_SH:-"$SCRIPT_DIR/dm_autoapprove.sh"}

hb() { # $1=state 心跳行（seat=MECH——机械轨活性锚，mech_hb 量表读面）
  [ -n "$FIN_DSN" ] || return 0
  psql "$FIN_DSN" -qAt -c "INSERT INTO ops_headless(ticket,seat,state,last_activity)
    VALUES('MECH','MECH','$1',now())
    ON CONFLICT (ticket) DO UPDATE SET seat=EXCLUDED.seat,
      state=EXCLUDED.state,last_activity=now(),updated_at=now()" >/dev/null 2>&1 || true # tolerate: 心跳落账尽力而为—败不阻拍面
}

# 容错自报同 dm_tick 式：本轨各段败→ops_event 升件行自报（禁只打
# stderr）；同窗去重防事件暴由 ops_esc 前缀键承担。
mech_esc() {
  "$OPS_ESC_SH" CTO "" "" "" "$1" "$2" || true # tolerate: 自报末环——ops_esc 写径败静默退（结构性限度在案）
}

# —— 单例闸：PG advisory lock（dm_tick FIFO-holder 同式——锁随脚本
# 生死；新键与 CTO 会话锁 42540354 分离）。拿不到=退（在飞拍吸收
# 新触发；5min timer+事件帧双兜底）。FIN_DSN 缺=无从持锁退化直行
# （与 kick 哑同律）。
MECH_LOCK_KEY=${O_MECH_LOCK_KEY:-42540355}
MECH_LOCK_PID=
MECH_GATE_DIR=
mech_gate_release() {
  [ -n "$MECH_LOCK_PID" ] && kill "$MECH_LOCK_PID" 2>/dev/null
  wait "$MECH_LOCK_PID" 2>/dev/null
  [ -n "$MECH_GATE_DIR" ] && rm -rf -- "$MECH_GATE_DIR" 2>/dev/null
  return 0
}
if [ -n "$FIN_DSN" ]; then
  MECH_GATE_DIR=$(mktemp -d "${TMPDIR:-/tmp}/o-mech-gate.XXXXXX") || exit 1
  mkfifo "$MECH_GATE_DIR/pipe" || { rm -rf -- "$MECH_GATE_DIR"; exit 1; }
  exec 8<>"$MECH_GATE_DIR/pipe" # 自持写端——EOF 不到，锁随进程生死
  exec 9<"$MECH_GATE_DIR/pipe"  # psql stdin 只读面（fd8=O_RDWR 已在=writer 不阻塞）
  psql "$FIN_DSN" -qAt >"$MECH_GATE_DIR/out" 2>/dev/null <&9 &
  MECH_LOCK_PID=$!
  printf 'SELECT pg_try_advisory_lock(%s);\n' "$MECH_LOCK_KEY" >&8
  GOT=
  for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do
    GOT=$(cat "$MECH_GATE_DIR/out" 2>/dev/null || true) # tolerate: gate out未落=续轮询合法径
    [ -n "$GOT" ] && break
    sleep 0.5
  done
  if [ "$GOT" != "t" ]; then
    echo "[mech] 机械轨闸未得（拍在飞）——后到即退" >&2
    mech_gate_release
    exit 0
  fi
  trap mech_gate_release EXIT
fi
hb alive

# T-0413 派单依赖闸（零交集已批即派——原 dm_tick 锁前位搬入本轨锁内：
# 机械轨单例内序跑，depgate/补拉幂等键自带防重）。
if [ -n "$FIN_DSN" ]; then
  DEPGATE_RC=0
  "$DM_DEPGATE_SH" || DEPGATE_RC=$?
  [ "$DEPGATE_RC" -ne 0 ] && mech_esc "派单依赖闸败" "rc=$DEPGATE_RC 已批快派面不可知按常径排队"
fi

# T-0343：常径派单和 rate-limit 挂起续跑共用 dispatch_worker.sh。仅最新
# 决策席派单或施工席 rate-limit:auto 挂起可补拉；其他挂起/异常不自动重试。
# 模型档从原派单备注恢复。补拉成功写 `拉起` 行自标幂等；无票文件免拉。
if [ -n "$FIN_DSN" ]; then
  DISPATCH_WORKER=${O_WORKER_DISPATCH_SCRIPT:-/opt/o/scripts/meta/dispatch_worker.sh}
  DISPATCH_WATER_SH=${O_DISPATCH_WATER_SH:-"$SCRIPT_DIR/dispatch_water.sh"} # T-0463 rc75 候水行唯一 emit 口径
  DISPATCH_TASK_DIR=${O_WORKER_TASK_DIR:-/opt/o/docs/tasks}
  DRC=0
  DISPATCH_ROWS=$(psql "$FIN_DSN" -qAt -F '|' -c "SELECT l.ticket || '|' || COALESCE((
      SELECT CASE WHEN e.note ILIKE '%swe-2-flash-high%' THEN 'swe-2-flash-high' WHEN e.note ILIKE '%swe-2-medium%' THEN 'swe-2-medium'
                  ELSE 'swe-2-flash-high' END
      FROM ops_event e WHERE e.ticket=l.ticket AND e.event='派单'
      ORDER BY e.ts DESC,e.id DESC LIMIT 1
    ),'swe-2-flash-high') FROM (
      SELECT ticket,max(ts) mt FROM ops_event WHERE event IN
        ('派单','领单','拉起','收口','移交','收口待审','裁决','完成','驳回',
         '挂起','异常')
        AND ticket<>'' GROUP BY ticket) l
    WHERE EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
        AND e.ts=l.mt AND e.event='派单' AND e.seat IN ('决策席','CTO'))
       OR EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
        AND e.ts=l.mt AND e.event='挂起' AND e.seat='施工席'
        AND e.note LIKE 'rate-limit:auto%')" 2>/dev/null) || DRC=$?
  [ "$DRC" -ne 0 ] && mech_esc "派单查询败" "rc=$DRC 派单/续跑候选集不可知按空集滑行"
  printf '%s\n' "$DISPATCH_ROWS" | while IFS='|' read -r tid model; do
    compgen -G "$DISPATCH_TASK_DIR/${tid}-*.md" >/dev/null || continue # ③无票免拉
    # T-0438 件⑥ 人工径滤（坑条目 real-money-ticket-machine-dispatch-waste
    # 律②——补派件把它踢出候选语义同域：票面「执行=人工径」出生行=机器
    # 永不可为，补拉窗不续派（不派不挂起不计漏拉））。
    MANUAL_LANE_SH=${O_MANUAL_LANE_SH:-"$SCRIPT_DIR/manual_lane_check.sh"}
    if "$MANUAL_LANE_SH" "$tid" "$DISPATCH_TASK_DIR"; then continue; fi # rc0=人工径票滤除；rc1/2=照旧
    st=$(systemctl --user is-active "o-worker-$tid.service" 2>/dev/null || true) # tolerate: is-active查败=态未知视同可拉
    case "$st" in active|activating) continue;; esac # 精确比对误配恒死修
    prompt="读 /opt/o/skills/施工席/SKILL.md 领单 $tid:只做工单范围内的事——写实现码/测试/提交/回填;范围外发现的问题不自己动手,写进交付备注;同票已有 Devin 会话或 worktree 时续跑不新开"
    if "$DISPATCH_WORKER" "$tid" "$model" "$prompt" >/dev/null 2>&1; then
      psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
        VALUES('CTO','拉起','$tid','漏拉/限流补派——session 续跑')" >/dev/null 2>&1 || true # tolerate: 拉起行落账尽力而为—事件败不阻补派
    else
      rc=$?
      if [ "$rc" -eq 75 ]; then
        "$DISPATCH_WATER_SH" "$tid" # T-0463：候水标记行（自含尽力而为——写败静默退下拍 rc75 自然重落）
        break # 并行水位/额度冷却——候选留队下拍重试
      fi
      echo "[mech] worker dispatch failed tid=$tid rc=$rc" >&2
    fi
  done
fi

# T-0365 挂起黑洞看门狗消费窗（消费面≠状态机：挂起不迁相位，只多开一扇
# 扫描窗）。未消化挂起行机械二择：await=绿窗 档复跑测试已消→dispatch_worker
# 续跑（note「挂起续跑」）；未消且>4h→升件呈业主。rate-limit:auto 挂起归上段既有窗不重复消费。
if [ -n "$FIN_DSN" ]; then
  SRC=0
  "$DM_SUSPEND_SH" || SRC=$?
  [ "$SRC" -ne 0 ] && mech_esc "看门狗本体败" "rc=$SRC dm_suspend 消费窗未跑净——挂起行消化停摆"
fi

# T-0459 交接对账哨兵消费窗（业主令「事事有交接，事事有回应」：交行龄>阈
# ∧同票无接行=断链升件，配对表 handoff_pairs.tsv 单源）。
if [ -n "$FIN_DSN" ]; then
  HRC=0
  "$DM_HANDOFF_SH" || HRC=$?
  [ "$HRC" -ne 0 ] && mech_esc "交接哨兵本体败" "rc=$HRC dm_handoff 消费窗未跑净——断链扫描停摆"
fi

# T-0393 候选票草案物料化窗（fin_ticket_draft pending→docs/tasks 候选票件
# +起草行；写径只落候选态零自派，批准权=COS/DM）。
if [ -n "$FIN_DSN" ]; then
  TRC=0
  (cd "$REPO" && FIN_PG_DSN="$FIN_DSN" go run ./tools/ticketdraft -repo "$REPO" >/dev/null 2>&1) || TRC=$?
  [ "$TRC" -ne 0 ] && mech_esc "物料化径败" "rc=$TRC ticketdraft 未跑净——pending 草案滞留（升件面工具内自报，本行=拍壳兜底）"
fi

# T-0455 机批类册窗（物料化落件的候选票同拍过机批闸：册内签名→CEO seat
# 已批行+范围钦定；册外/人工径→零批滞留 24h 升件呈 DM；驳回行=账面撤销）。
if [ -n "$FIN_DSN" ]; then
  AA_RC=0
  REPO="$REPO" FIN_DSN="$FIN_DSN" "$AUTOAPPROVE_SH" || AA_RC=$?
  [ "$AA_RC" -ne 0 ] && mech_esc "机批闸径败" "rc=$AA_RC dm_autoapprove 未跑净——候选批准面滞留"
fi

# T-0414 机械轨票翻窗（票翻/台账补登壳轨件：dm_ledger.sh——ops_event 新行
# 即办机械活；会话判词刚落的票在同拍尾窗即归档补登，零会话等候）。
if [ -n "$FIN_DSN" ]; then
  LRC=0
  "$DM_LEDGER_SH" || LRC=$?
  [ "$LRC" -ne 0 ] && mech_esc "机械轨本体败" "rc=$LRC dm_ledger 未跑净——票翻/台账补登滑行"
fi

# T-0438 判词送达窗（verdict_deliver.sh：需业主动作判词同拍转
# fin_condition_gap 请求箱+feishu 推送——「裁了≠送到了」机内化）。
if [ -n "$FIN_DSN" ]; then
  VDRC=0
  "$VERDICT_DELIVER_SH" || VDRC=$?
  [ "$VDRC" -ne 0 ] && mech_esc "判词送达闸败" "rc=$VDRC 需业主动作判词滞留未闭环（未闭环件下拍重试）"
fi

# ── 收敛窗（GW-001 段A 唯一新逻辑≈40 行）：deploy_window_due 判据→
# faces 执行。判据全在 deploy_window_due（裁「该不该发」）；本窗只读
# state=应发 执行 faces= 面集（image→deploy_image.sh / oci→
# deploy_oci.sh）。部署 sha=HEAD（两 deploy 件出处闸同径——HEAD 须在
# origin/master；本窗加钉 HEAD==origin/master 才发，未收敛=跳过待下拍，
# 非败不冻）。幂等=ops_event 部署行 sha12 查重（同 sha 重触部署行唯一）；
# 面败=ops_esc DEPLOY-FIRE-FAIL:<sha12>:<face>+本面冻结 24h（窗内同面
# 升件行在=不重试——败即升件+面冻结+不自愈重试，GW-001 §二原则）。
if [ -n "$FIN_DSN" ]; then
  DEPLOY_WINDOW_SH=${O_DEPLOY_WINDOW_SH:-"$SCRIPT_DIR/deploy_window_due.sh"}
  DEPLOY_IMAGE_SH=${O_DEPLOY_IMAGE_SH:-"$REPO/scripts/deploy_image.sh"}
  DEPLOY_OCI_SH=${O_DEPLOY_OCI_SH:-"$REPO/scripts/deploy_oci.sh"}
  WIN_OUT=$(bash "$DEPLOY_WINDOW_SH" 2>/dev/null); WRC=$?
  WSTATE=$(printf '%s\n' "$WIN_OUT" | sed -n 's/^state=//p' | head -1)
  WREASON=$(printf '%s\n' "$WIN_OUT" | sed -n 's/^reason=//p' | head -1)
  WFACES=$(printf '%s\n' "$WIN_OUT" | sed -n 's/^faces=//p' | head -1)
  echo "[mech] deploy_window state=${WSTATE:-不可读} faces=${WFACES:-空} rc=$WRC ${WREASON:-}" >&2
  if [ "$WSTATE" = "应发" ] && [ -n "$WFACES" ]; then
    DEP_SHA12=$(git -C "$REPO" rev-parse --short=12 HEAD 2>/dev/null || true) # tolerate: sha 不可解=空→下方等值判跳过
    OM_SHA=$(git -C "$REPO" rev-parse origin/master 2>/dev/null || true) # tolerate: 远端基线不可解=空→与 HEAD 等值判恒不等=未收敛跳过（fail-closed 不发）
    HEAD_SHA=$(git -C "$REPO" rev-parse HEAD 2>/dev/null || true) # tolerate: HEAD 不可解=空→同上恒不等=跳过（sha 空判在前行已兜）
    if [ -z "$DEP_SHA12" ] || [ "$HEAD_SHA" != "$OM_SHA" ]; then
      echo "[mech] HEAD≠origin/master（head=${HEAD_SHA:-不可解} om=${OM_SHA:-不可解}）——部署面未收敛跳过待下拍" >&2
    else
      fire_deploy_face() { # $1=face(image|oci)——面级幂等+冻结+发
        local face=$1 keypat doneq frz frc
        case "$face" in image|oci) ;; *) return 0 ;; esac
        keypat="DEPLOY-FIRE-FAIL:[0-9a-f]+:$face([^a-z]|$)"
        frz=$(psql "$FIN_DSN" -qAt -c "SELECT 1 FROM ops_event WHERE event='升件'
          AND note ~ '$keypat' AND ts>now()-interval '24 hours' LIMIT 1" 2>/dev/null); frc=$?
        if [ "$frc" -ne 0 ]; then
          mech_esc "DEPLOY-FREEZE-CHECK-FAIL:$face" "冻结态查询败 rc=$frc——按不可判不发（fail-closed）"
          return 0
        fi
        [ "$frz" = "1" ] && { echo "[mech] deploy 面冻结中 face=$face——24h 窗内不重试" >&2; return 0; }
        if [ "$face" = image ]; then
          doneq="SELECT 1 FROM ops_event WHERE event='部署'
            AND substring(note from 'deploy ([0-9a-f]+)')='$DEP_SHA12'
            AND note LIKE '% rc=0 %' LIMIT 1"
        else
          doneq="SELECT 1 FROM ops_event WHERE event='部署'
            AND substring(note from 'deploy-oci ([0-9a-f]+)')='$DEP_SHA12'
            AND note LIKE '% rc=0 %' LIMIT 1"
        fi
        done=$(psql "$FIN_DSN" -qAt -c "$doneq" 2>/dev/null); frc=$?
        if [ "$frc" -ne 0 ]; then
          mech_esc "DEPLOY-IDEM-CHECK-FAIL:$face" "部署行幂等查询败 rc=$frc——按不可判不发（fail-closed）"
          return 0
        fi
        [ "$done" = "1" ] && { echo "[mech] deploy $face sha=$DEP_SHA12 已有 rc=0 行——幂等跳过" >&2; return 0; }
        case "$face" in
          image) bash "$DEPLOY_IMAGE_SH" ;;
          oci)   bash "$DEPLOY_OCI_SH" ;;
        esac
        frc=$?
        if [ "$frc" -ne 0 ]; then
          mech_esc "DEPLOY-FIRE-FAIL:$DEP_SHA12:$face" "deploy_${face} rc=$frc——本面冻结24h不重试（判据在 deploy_window_due）"
        fi
        return 0
      }
      IFS=',' read -ra _FACES <<< "$WFACES"
      for f in "${_FACES[@]}"; do
        f=$(printf '%s' "$f" | tr -d '[:space:]')
        fire_deploy_face "$f"
      done
    fi
  fi
fi
exit 0
