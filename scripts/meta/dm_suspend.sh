#!/usr/bin/env bash
# dm_suspend.sh —— 挂起黑洞看门狗消费窗（T-0365：消费面≠状态机——
# 挂起不迁相位，只给 DM 常拍多开一扇扫描窗）。未消化判据=票内最新
# 生命周期事件仍='挂起'（处置事件即认消化，同领单消化径）；
# rate-limit:auto 挂起归 dm_tick 既有续跑窗不重复消费。消解机械
# 二择：note 携 await=绿窗:<[pkg:]Test> 档→复跑对应测试即判——
# 绿→dispatch_worker 续跑派工（拉起行 note 标「挂起续跑」）；未消
# （绿窗复跑红/非绿窗档不可机械判/无 await）且挂起龄>4h→
# escalate.sh 升件呈业主。不发明第三种处置。FIN_DSN 缺=静默退
# （拍壳同律）；本件只被 dm_tick.sh 尾段零 LLM 段调用。
set -u
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$FIN_DSN" ] || exit 0
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
DISPATCH_WORKER=${O_WORKER_DISPATCH_SCRIPT:-/opt/o/scripts/meta/dispatch_worker.sh}
KEY_GATE=${O_KEY_GATE_SH:-"$SCRIPT_DIR/ops_event_key_gate.sh"} # T-0499 键卫生闸（缺席 fail-open）
ESCALATE_SH=${O_SUSPEND_ESCALATE_SCRIPT:-"$SCRIPT_DIR/escalate.sh"}
REPO_DIR=${O_SUSPEND_REPO_DIR:-/opt/o}
AGE_LIMIT=${O_SUSPEND_AGE_LIMIT:-14400}
GO_TIMEOUT=${O_SUSPEND_GO_TIMEOUT:-900}

QC(){ printf '%s' "${1:-}" | sed "s/'/''/g"; }

# 票内最新生命周期事件='挂起' 且非 rate-limit:auto（既有窗消费）——
# '升件'/'开工' 入集：处置/重启事件同认消化防重扫（本窗升件落行后
# 下拍自消化，不重升）。
ROWS=$(psql "$FIN_DSN" -qAt -F $'\x1f' -c "SELECT ticket,note,seat,extract(epoch from now()-ts)::bigint
FROM (
  SELECT DISTINCT ON (ticket) ticket,event,note,seat,ts
  FROM ops_event
  WHERE ticket<>'' AND event IN
    ('派单','领单','开工','拉起','收口','移交','收口待审','裁决','完成','驳回','挂起','异常','升件')
  ORDER BY ticket,ts DESC,id DESC
) l
WHERE event='挂起' AND NOT (seat='施工席' AND note LIKE 'rate-limit:auto%')" 2>/dev/null) || exit 1 # T-0372：扫描败≠空集——非零由消费面 dm_tick 升件显报
[ -n "$ROWS" ] || exit 0

printf '%s\n' "$ROWS" | while IFS=$'\x1f' read -r tid note seat age; do
  [ -n "$tid" ] || continue
  case "$tid" in *[!A-Za-z0-9_-]*) continue;; esac
  case "$age" in ''|*[!0-9]*) age=0;; esac

  # await=<档>:<参> 结构化阻塞条件（鼓励写法——无/不识=不可机械判，
  # 走超时升件径）。全角冒号归一防写面漂移。
  spec=
  case "$note" in *await=*) spec=${note#*await=};; esac
  spec=${spec%%[[:space:]]*}
  spec=${spec//：/:}
  gate=${spec%%:*}
  param=
  [ "$spec" != "$gate" ] && param=${spec#*:}

  resolved=0
  case "$gate" in
    绿窗|green|green-window|green_window)
      pkg= arg=
      case "$param" in
        *:*) pkg=${param%%:*}; arg=${param#*:} ;;
        *)   arg=$param ;;
      esac
      if [ -n "$arg" ]; then
        gargs=(-count=1 -run "^${arg}$")
        [ -n "$pkg" ] && gargs+=("$pkg") || gargs+=(./...)
        if ( cd "$REPO_DIR" && timeout "$GO_TIMEOUT" go test "${gargs[@]}" ) >/dev/null 2>&1; then
          resolved=1
        fi
      fi
      ;;
  esac

  if [ "$resolved" -eq 1 ]; then
    # 模型档从原派单备注恢复（与 dm_tick 尾段同式）。
    model=$(psql "$FIN_DSN" -qAt -c "SELECT CASE WHEN e.note ILIKE '%swe-2-flash-high%' THEN 'swe-2-flash-high' WHEN e.note ILIKE '%swe-2-medium%' THEN 'swe-2-medium' ELSE 'swe-2-flash-high' END FROM ops_event e WHERE e.ticket='$(QC "$tid")' AND e.event='派单' ORDER BY e.ts DESC,e.id DESC LIMIT 1" 2>/dev/null)
    [ -n "$model" ] || model=swe-2-flash-high
    prompt="读 /opt/o/skills/施工席/SKILL.md 施工协议领单/续领 $tid；按票面施工，同票已有 Devin 会话/worktree 时续跑；本次=挂起续跑（await=${spec:-无} 阻塞已消）"
    if "$DISPATCH_WORKER" "$tid" "$model" "$prompt" >/dev/null 2>&1; then
      NMSG="挂起续跑——await=${spec} 阻塞消解，dispatch_worker 续跑派工"
      # T-0499 键卫生闸：拉起行 ticket 落库前拒伪键（loud 在闸内；
      # 字符集校验下本闸恒过=双保险，闸缺席 fail-open）。
      [ ! -x "$KEY_GATE" ] || "$KEY_GATE" "$tid" "event=拉起" || continue
      psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note) VALUES('CTO','拉起','$(QC "$tid")','$(QC "$NMSG")')" >/dev/null 2>&1 || true # tolerate: 拉起行落账尽力而为—事件败不阻哨兵巡
    else
      rc=$?
      [ "$rc" -eq 75 ] && break # 并行水位/额度冷却——挂起行仍在，下拍续判
      echo "[dm-suspend] resume dispatch failed tid=$tid rc=$rc" >&2
    fi
    continue
  fi

  # 未消：挂起龄超阈→升件呈业主（升件行入消化词表——下拍不重扫）。
  if [ "$age" -gt "$AGE_LIMIT" ]; then
    "$ESCALATE_SH" "CTO" "$tid" "" "" \
      "挂起超时未消（龄>${AGE_LIMIT}s）——await=${spec:-无} 原挂起note=${note:0:160}——呈业主裁" || true # tolerate: 升件尽力而为—败不阻挂起超时哨兵巡
  fi
done
exit 0
