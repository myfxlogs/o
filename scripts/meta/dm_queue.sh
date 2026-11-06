#!/usr/bin/env bash
# dm_queue.sh —— DM 班次队列唯一推导（T-0354 件②：拍壳闸与会话班内 rescan
# 同源同式——两处各写一份 SQL 必漂，故单源出此）。输出=待裁/待派票号
# （一行一个）；可选 mode=review|approved 分面复用同判据（T-0413 依赖闸），
# 默认 all 保原序；approved 面按最近已批 ts 升序供最旧候选先派。判据=
# （收口/移交/收口待审）无终态裁决（裁决/完成/驳回），或最新刻已批未进
# 派单链（派单/领单/拉起/裁决/完成/驳回/挂起）。FIN_DSN 缺=exit 78
# （拍壳退化径自理）；psql 败=exit 1（T-0372：原管道 awk rc 恒 0——
# 查询败=空集静默位根修，败≠空集由消费面 dm_tick 升件显报）。
# T-0375 件① 外派豁免（返修1——86efad3b 业主令新规）：外派件收口回 DM
# 常径裁（收口面零排除，自入待裁）；豁免只盖「已批」分支。
# T-0380 判据事件化（T-0379 实证——业主令撤外派改内循环新已批落行,
# 仍因票史有外派行被永排出待派集）：排除由「票史存在外派行」改查
# 「外派行后无再批」——外派 ts 后有已批新行=外派被撤票回常径待派
# （外派→已批=入）；外派后接领单 latest=领单,已批分支首条件自败=
# 不入；同刻 已批+外派 批写仍按外派向排除。非外派件推导零变。
set -u
MODE=${1:-all}
case "$MODE" in all|review|approved) ;; *)
  echo "usage: dm_queue.sh [all|review|approved]" >&2
  exit 2
esac
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$FIN_DSN" ] || exit 78
REVIEW_PRED="EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
        AND e.ts=l.mt AND e.event IN ('收口','移交','收口待审'))
     AND NOT EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
        AND e.ts=l.mt AND e.event IN ('裁决','完成','驳回'))"
APPROVED_PRED="EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
        AND e.ts=l.mt AND e.event='已批')
     AND NOT EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
        AND e.ts=l.mt AND e.event IN
          ('派单','领单','拉起','裁决','完成','驳回','挂起'))
     AND NOT EXISTS (SELECT 1 FROM ops_event w WHERE w.ticket=l.ticket
        AND w.event='外派' AND NOT EXISTS
          (SELECT 1 FROM ops_event a WHERE a.ticket=l.ticket
            AND a.event='已批' AND a.ts>w.ts))"
case "$MODE" in
  review)   PRED="($REVIEW_PRED)"; ORDER="l.ticket" ;;
  approved) PRED="($APPROVED_PRED)"; ORDER="l.mt,l.ticket" ;;
  *)        PRED="(($REVIEW_PRED) OR ($APPROVED_PRED))"; ORDER="l.ticket" ;;
esac
RAW=$(psql "$FIN_DSN" -qAt -c "SELECT l.ticket FROM (
    SELECT ticket, max(ts) mt FROM ops_event WHERE ticket<>'' GROUP BY ticket
  ) l WHERE $PRED
  ORDER BY $ORDER" 2>/dev/null) || exit 1
printf '%s\n' "$RAW" | awk 'NF && !seen[$0]++'
