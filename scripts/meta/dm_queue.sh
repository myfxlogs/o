#!/usr/bin/env bash
# dm_queue.sh —— DM 班次队列唯一推导（T-0354 件②：拍壳闸与会话班内 rescan
# 同源同式——两处各写一份 SQL 必漂，故单源出此）。输出=待裁/待派票号
# （一行一个，去重升序）；判据=T-0315②/T-0347 原文照搬：最新刻收口类
# （收口/移交/收口待审）无终态裁决（裁决/完成/驳回），或最新刻已批未进
# 派单链（派单/领单/拉起/裁决/完成/驳回/挂起）。FIN_DSN 缺=exit 78
# （拍壳退化径自理）。
set -u
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$FIN_DSN" ] || exit 78
psql "$FIN_DSN" -qAt -c "SELECT l.ticket FROM (
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
  ) ORDER BY l.ticket" 2>/dev/null | awk 'NF && !seen[$0]++'
