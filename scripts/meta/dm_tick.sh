#!/usr/bin/env bash
# dm_tick.sh —— 决策席待裁批件拍壳（T-0312 第六刀——照 verdict_tick.sh
# 逐字对形：ops_ticket_state 待裁集水印闸>0 才烧零空拍；每拍（含空拍/
# 败拍）ops_headless META-DM 心跳行；FIN_DSN 隧道径 SG 直读）。
set -u
# T-0323 踢径钦定：裁毕通报/升件踢 O-DM 交互席；派单直踢 O-Dev 施工 pane。
export REPORT_KICK=O-DM
export REPORT_DISPATCH=O-Dev
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')

hb() { # $1=state 心跳行（META-DM——meta 席位口径）
  [ -n "$FIN_DSN" ] || return 0
  psql "$FIN_DSN" -qAt -c "INSERT INTO ops_headless(ticket,seat,state,last_activity)
    VALUES('META-DM','决策席','$1',now())
    ON CONFLICT (ticket) DO UPDATE SET seat=EXCLUDED.seat,
      state=EXCLUDED.state,last_activity=now(),updated_at=now()" >/dev/null 2>&1 || true
}

# ① 待裁闸：ops_ticket_state 待裁事件>0 才烧——否则 skip（心跳照写）。
if [ -n "$FIN_DSN" ]; then
  # T-0315②：最新刻事件集判定（max(ts) 时刻集含终态词 裁决/完成/驳回
  # 即离集——同 ts 撞族 DISTINCT ON 无序漂移根修；ops_event 唯一真源
  # 不票面交叉）。
  N=$(psql "$FIN_DSN" -qAt -c "SELECT count(*) FROM (
      SELECT ticket, max(ts) mt FROM ops_event WHERE ticket<>'' GROUP BY ticket
    ) l WHERE EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
        AND e.ts=l.mt AND e.event IN ('收口','移交','收口待审'))
      AND NOT EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
        AND e.ts=l.mt AND e.event IN ('裁决','完成','驳回'))" 2>/dev/null || echo 0)
  if [ "$N" = "0" ]; then
    hb alive
    exit 0
  fi
fi

# ② devin 批拍（cwd=scratch——裁决协议 runbook 绝对路径指件）。
cd /var/tmp/o-decision || exit 1
/home/mluser/.local/bin/devin -p --model swe --respect-workspace-trust false \
  --permission-mode dangerous -- \
  "读 /opt/o/skills/决策席/SKILL.md 裁决协议节 拍待裁件裁决"
RC=$?
# ③ 每拍心跳全件（成败同写——败拍=stalled 显形非隐）。
[ $RC -eq 0 ] && hb alive || hb stalled
exit 0
