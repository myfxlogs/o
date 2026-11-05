#!/usr/bin/env bash
# dm_queue.sh —— DM 班次队列唯一推导（T-0354 件②：拍壳闸与会话班内 rescan
# 同源同式——两处各写一份 SQL 必漂，故单源出此）。输出=待裁/待派票号
# （一行一个，去重升序）；判据=T-0315②/T-0347 原文照搬：最新刻收口类
# （收口/移交/收口待审）无终态裁决（裁决/完成/驳回），或最新刻已批未进
# 派单链（派单/领单/拉起/裁决/完成/驳回/挂起）。FIN_DSN 缺=exit 78
# （拍壳退化径自理）；psql 败=exit 1（T-0372：原管道 awk rc 恒 0——
# 查询败=空集静默位根修，败≠空集由消费面 dm_tick 升件显报）。
# T-0375 件① 外派豁免（返修1——86efad3b 业主令新规）：外派件收口回 DM
# 常径裁（收口面零排除，自入待裁）；豁免只盖「已批」分支——外派件不被
# DM 机派（防「外派后重批 latest=已批」缘案机派双发），票级历史排除
# （外派恒早于领单/收口落行、不在最新刻——排除须查全史，@max 排除集
# 盖不住）。两分支判定结构与最新刻事件面比对不变形；非外派件推导零变。
set -u
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$FIN_DSN" ] || exit 78
RAW=$(psql "$FIN_DSN" -qAt -c "SELECT l.ticket FROM (
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
          ('派单','领单','拉起','裁决','完成','驳回','挂起'))
     AND NOT EXISTS (SELECT 1 FROM ops_event w WHERE w.ticket=l.ticket
        AND w.event='外派'))
  )
  ORDER BY l.ticket" 2>/dev/null) || exit 1
printf '%s\n' "$RAW" | awk 'NF && !seen[$0]++'
