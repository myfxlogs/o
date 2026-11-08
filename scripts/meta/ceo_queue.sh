#!/usr/bin/env bash
# ceo_queue.sh —— CEO 席无头待裁队列推导（T-0456 业主令双无头建席
# 配套）：席位上行件归 CEO 方向层；探针/壳轨机械升件留 CTO 壳轨。
# 判据: event ∈ {升件,呈批} 且 seat ∈ 席位集(COS上行/COS/O-Dev交互席/
#   决策席/META-DM/裁定席/施工席/外审席——席上行的件非探针自产)
#   且其后零终态(裁决/完成/驳回/消/处置/挂起)——输出 ticket 一行一个。
# 用法: ceo_queue.sh；FIN_DSN 隧道径与 dm_tick 同源。
set -euo pipefail
FIN_DSN="${FIN_DSN:-$(grep '^FIN_PG_DSN=' "${DM_ENV_FILE:-/opt/o-research/env}" 2>/dev/null | cut -d= -f2- | tr -d '"')}"
[ -n "$FIN_DSN" ] || exit 0
psql "$FIN_DSN" -qAt <<'SQL'
SELECT DISTINCT e.ticket FROM ops_event e
WHERE e.event IN ('升件','呈批')
  AND e.seat IN ('COS上行','COS','O-Dev交互席','决策席','META-DM','CTO','CEO',
                 '裁定席','施工席','外审席','CEO上行','管理席')
  AND NOT EXISTS (
    SELECT 1 FROM ops_event t
    WHERE t.ticket=e.ticket AND t.ts>e.ts
      AND t.event IN ('裁决','完成','驳回','消','处置','挂起','领单'))
ORDER BY e.ticket;
SQL
