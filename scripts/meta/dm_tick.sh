#!/usr/bin/env bash
# dm_tick.sh —— 决策席待裁批件拍壳（T-0312 第六刀——照 verdict_tick.sh
# 逐字对形：ops_ticket_state 待裁集水印闸>0 才烧零空拍；每拍（含空拍/
# 败拍）ops_headless META-DM 心跳行；FIN_DSN 隧道径 SG 直读）。
set -u
DECISION_DIR=${O_DECISION_WORKDIR:-/var/tmp/o-decision}
# T-0323 踢径钦定单目标：一切自报（含派单通报/升件）踢 O-DM 交互席；
# 派单不踢 O-Dev——常径施工拉起走 systemd-run o-worker-<tid>（T-0324）。
export REPORT_KICK=O-DM
export O_SEAT=决策席
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')

hb() { # $1=state 心跳行（META-DM——meta 席位口径）
  [ -n "$FIN_DSN" ] || return 0
  psql "$FIN_DSN" -qAt -c "INSERT INTO ops_headless(ticket,seat,state,last_activity)
    VALUES('META-DM','决策席','$1',now())
    ON CONFLICT (ticket) DO UPDATE SET seat=EXCLUDED.seat,
      state=EXCLUDED.state,last_activity=now(),updated_at=now()" >/dev/null 2>&1 || true
}

# ① 待裁闸：无待裁时跳过 LLM，但仍运行下方零 LLM 的 worker 派发/续跑闸。
SKIP_DECISION=0
if [ -n "$FIN_DSN" ]; then
  # T-0315②/T-0347：最新刻事件集判定。裁决候选仍按收口类+裁决终态；
  # 已批派单待件独立纳入，且派单/领单/拉起/裁决/完成/驳回/挂起均离集。
  # max(ts) 同时刻保留整组事件，避免 DISTINCT ON 无序漂移；ops_event 唯一真源。
  N=$(psql "$FIN_DSN" -qAt -c "SELECT count(*) FROM (
      SELECT ticket, max(ts) mt FROM ops_event WHERE ticket<>'' GROUP BY ticket
    ) l WHERE (
      (EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
          AND e.ts=l.mt AND e.event IN ('收口','移交','收口待审'))
       AND NOT EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
          AND e.ts=l.mt AND e.event IN ('裁决','完成','驳回')))
      OR
      (EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
          AND e.ts=l.mt AND e.event='已批')
       AND NOT EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
          AND e.ts=l.mt AND e.event IN
            ('派单','领单','拉起','裁决','完成','驳回','挂起')))
    )" 2>/dev/null || echo 0)
  if [ "$N" = "0" ]; then
    hb alive
    SKIP_DECISION=1
  fi
fi

if [ "$SKIP_DECISION" -eq 0 ]; then
  # ② 固定本拍票集供 token 行归属；多票一批记 META-DM 聚合行，不伪分摊。
  PENDING_TICKETS=$(psql "$FIN_DSN" -qAt -c "SELECT l.ticket FROM (
      SELECT ticket, max(ts) mt FROM ops_event WHERE ticket<>'' GROUP BY ticket
    ) l WHERE (
      (EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
          AND e.ts=l.mt AND e.event IN ('收口','移交','收口待审'))
       AND NOT EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
          AND e.ts=l.mt AND e.event IN ('裁决','完成','驳回')))
      OR
      (EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
          AND e.ts=l.mt AND e.event='已批')
       AND NOT EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
          AND e.ts=l.mt AND e.event IN
            ('派单','领单','拉起','裁决','完成','驳回','挂起')))
    ) ORDER BY l.ticket" 2>/dev/null || true)
  PENDING_TICKETS=$(printf '%s\n' "$PENDING_TICKETS" | awk 'NF && !seen[$0]++')
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
    "读 /opt/o/skills/决策席/SKILL.md 裁决协议节 拍待裁件裁决"
  RC=$?
  # ④ 用量采集旁路；失败只告警，不阻断裁决和事件链。
  [ ! -e "$USAGE_TRANSCRIPT" ] || chmod 600 "$USAGE_TRANSCRIPT" 2>/dev/null || true
  USAGE_BIN=${O_WORKER_USAGE_BIN:-/opt/o/scripts/meta/ticket_usage.sh}
  if [ -s "$USAGE_TRANSCRIPT" ]; then
    if "$USAGE_BIN" "$USAGE_TICKET" audit 决策席 "$USAGE_TRANSCRIPT" "$FIN_DSN" "$BATCH_NOTE"; then
      rm -rf -- "$USAGE_DIR"
    else
      echo "[dm] token usage collection failed; continuing decision lifecycle; transcript kept=$USAGE_DIR" >&2
    fi
  else
    echo "[dm] token transcript missing/empty; continuing decision lifecycle" >&2
    rm -rf -- "$USAGE_DIR"
  fi
  # ⑤ 每拍心跳全件（成败同写——败拍=stalled 显形非隐）。
  [ $RC -eq 0 ] && hb alive || hb stalled
fi

# T-0343：常径派单和 rate-limit 挂起续跑共用 dispatch_worker.sh。仅最新
# 决策席派单或施工席 rate-limit:auto 挂起可补拉；其他挂起/异常不自动重试。
# 模型档从原派单备注恢复。补拉成功写 `拉起` 行自标幂等；无票文件免拉。
if [ -n "$FIN_DSN" ]; then
  DISPATCH_WORKER=${O_WORKER_DISPATCH_SCRIPT:-/opt/o/scripts/meta/dispatch_worker.sh}
  psql "$FIN_DSN" -qAt -F '|' -c "SELECT l.ticket || '|' || COALESCE((
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
        AND e.note LIKE 'rate-limit:auto%')" 2>/dev/null | while IFS='|' read -r tid model; do
    compgen -G "/opt/o/docs/tasks/${tid}-*.md" >/dev/null || continue # ③无票免拉
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
exit 0
