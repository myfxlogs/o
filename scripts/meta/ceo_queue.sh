#!/usr/bin/env bash
# ceo_queue.sh —— CEO 席无头待裁队列推导（T-0456 业主令双无头建席
# 配套）：席位上行件归 CEO 方向层；探针/壳轨机械升件留 CTO 壳轨。
# 判据: event ∈ {升件,呈批} 且 seat ∈ 席位集(COS上行/COS/O-Dev交互席/
#   决策席/META-DM/裁定席/施工席/外审席/研究员/MECH——T-0509 白名单
#   呈批作者实写；探针/壳轨机械升件留 CTO 壳轨 seat=CTO 同列)
#   且 ticket<>'' 且其后零终态(裁决/完成/驳回/消/处置/挂起/领单)
#   ——输出 ticket 一行一个。
# 票外呈批行（T-0509：seat=MECH event=呈批 ticket='' 为票外件不入
#   相位投影的钉死形）——以「呈批#<ops_event.id>」合成行入队
#   （议题#id 同构），出队=其后终态事件 note 引证「呈批#id」。
# 用法: ceo_queue.sh；FIN_DSN 隧道径与 dm_tick 同源。
# 议题行读面（T-0465 钦定③）：ceo_northstar 北极星背离自生成件=班内件源
# ——seat=CEO event='议题' 未结行入队（票空行以「议题#<ops_event.id>」出列，
# 处置落行 note 引证「议题#id」即出队——升件#id 引证同例）。
set -euo pipefail
FIN_DSN="${FIN_DSN:-$(grep '^FIN_PG_DSN=' "${DM_ENV_FILE:-/opt/o-research/env}" 2>/dev/null | cut -d= -f2- | tr -d '"')}"
[ -n "$FIN_DSN" ] || exit 0
psql "$FIN_DSN" -qAt <<'SQL'
SELECT * FROM (
  SELECT DISTINCT e.ticket AS ticket FROM ops_event e
  WHERE e.event IN ('升件','呈批')
    AND e.ticket <> ''
    AND e.seat IN ('COS上行','COS','O-Dev交互席','决策席','META-DM','CTO','CEO',
                   '裁定席','施工席','外审席','CEO上行','管理席','研究员','MECH')
    AND NOT EXISTS (
      SELECT 1 FROM ops_event t
      WHERE t.ticket=e.ticket AND t.ts>e.ts
        AND t.event IN ('裁决','完成','驳回','消','处置','挂起','领单'))
  UNION ALL
  SELECT '议题#'||e.id::text FROM ops_event e
  WHERE e.event='议题' AND e.seat='CEO'
    AND NOT EXISTS (
      SELECT 1 FROM ops_event c
      WHERE c.ts>e.ts
        AND c.event IN ('裁决','完成','驳回','消','处置','挂起')
        AND c.note ~ ('议题#' || e.id::text || '([^0-9]|$)'))
  UNION ALL
  SELECT '呈批#'||e.id::text FROM ops_event e
  WHERE e.event='呈批' AND e.ticket='' AND e.seat='MECH'
    AND NOT EXISTS (
      SELECT 1 FROM ops_event c
      WHERE c.ts>e.ts
        AND c.event IN ('裁决','完成','驳回','消','处置','挂起')
        AND c.note ~ ('呈批#' || e.id::text || '([^0-9]|$)'))
) u
ORDER BY CASE WHEN u.ticket LIKE '议题#%' THEN 0 ELSE 1 END, u.ticket;
SQL
