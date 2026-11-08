#!/usr/bin/env bash
# dm_queue.sh —— DM 班次队列唯一推导（T-0354 件②：拍壳闸与会话班内 rescan
# 同源同式——两处各写一份 SQL 必漂，故单源出此）。输出=待裁/待派票号
# （一行一个）；可选 mode=review|approved|dispatched 分面（T-0413 依赖闸
# /GW-001 段B1 补拉窗同源），默认 all=review∪approved 保原序；approved/
# dispatched 面按最新相位 ts 升序供最旧候选先派。FIN_DSN 缺=exit 78
# （拍壳退化径自理）；psql 败=exit 1（T-0372：原管道 awk rc 恒 0——
# 查询败=空集静默位根修，败≠空集由消费面 dm_tick 升件显报）。
#
# GW-001 段B1 相位投影：判据底座=ops_ticket_phase VIEW（fin_0147——
# 生命周期词集过滤：正词 起草→已批→派单→拉起/领单→收口/移交/收口待审
# →裁决→完成，侧态 呈批/开工/外派/审计中，负词 驳回/挂起/作废/异常；
# 升件/候水/处置/送达/退班/等锁/先例适用/部署等说明类事件一律不进相位
# ——处置=CEO 断链处置注记非票相位（被踢票实证+判据①钦定形），旧
# ops_ticket_state 全事件 DISTINCT ON 被尾随说明类事件踢出队列的病灶
# 根修面；旧 VIEW 一字不动，存量消费者零误伤）。dispatched=最新相位
# 派单（seat 决策席/CTO）或挂起（seat 施工席∧note rate-limit:auto%）——
# 派单未拉起+限流续跑候选（原 mech_tick 补拉窗内联 SQL 收编单源，模型档
# 恢复仍归消费面票级子查询——非队列推导）。
# 半成品窗桥：VIEW 未迁移落地（探针 LIMIT 0 败）→legacy 面=旧推导 SQL
# 逐字保留+dispatched 同义移植——迁移落地自动切 view 面（过渡桥非双轨：
# 两面同判据族，旧面随迁移确认退役另票；在飞会话判据语义迁移前后连续，
# 唯一差=说明类事件不再踢队=被踢票回队——预期方向）。
# T-0375 件① 外派豁免（返修1——86efad3b 业主令新规）：外派件收口回 DM
# 常径裁（收口面零排除，自入待裁）；豁免只盖「已批」分支。
# T-0380 判据事件化（T-0379 实证——业主令撤外派改内循环新已批落行,
# 仍因票史有外派行被永排出待派集）：排除由「票史存在外派行」改查
# 「外派行后无再批」——外派 ts 后有已批新行=外派被撤票回常径待派
# （外派→已批=入）；外派后接领单 latest=领单,已批分支首条件自败=
# 不入；同刻 已批+外派 批写仍按外派向排除。非外派件推导零变。
set -u
MODE=${1:-all}
case "$MODE" in all|review|approved|dispatched) ;; *)
  echo "usage: dm_queue.sh [all|review|approved|dispatched]" >&2
  exit 2
esac
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$FIN_DSN" ] || exit 78

# view 面判据（ops_ticket_phase 列=latest_event/latest_ts/latest_seat/
# latest_note/phase——票史级外派豁免 VIEW 只载最新相位载不下，回
# ops_event 对拍票史，T-0375/T-0380 语义逐字不动）。同刻并列防御：
# legacy 判据在 mt 同 ts 上带 NOT EXISTS 反词（收口族×裁决/完成/驳回、
# 已批×派单/领单/拉起/裁决/完成/驳回/挂起）——VIEW 同刻并列 join 出
# 多行会让反词失效（收口+裁决并列→收口行仍命中 review），故 view 面
# 同式补 latest_ts 反词闸，tie 语义与 legacy 逐字对齐。
WAIPAI_HIST="NOT EXISTS (SELECT 1 FROM ops_event w
    WHERE w.ticket=ops_ticket_phase.ticket AND w.event='外派'
      AND NOT EXISTS (SELECT 1 FROM ops_event a
        WHERE a.ticket=ops_ticket_phase.ticket AND a.event='已批'
          AND a.ts>w.ts))"
REVIEW_ANTI="NOT EXISTS (SELECT 1 FROM ops_event x
    WHERE x.ticket=ops_ticket_phase.ticket
      AND x.ts=ops_ticket_phase.latest_ts
      AND x.event IN ('裁决','完成','驳回'))"
APPROVED_ANTI="NOT EXISTS (SELECT 1 FROM ops_event x
    WHERE x.ticket=ops_ticket_phase.ticket
      AND x.ts=ops_ticket_phase.latest_ts
      AND x.event IN ('派单','领单','拉起','裁决','完成','驳回','挂起'))"
if psql "$FIN_DSN" -qAt -c "SELECT 1 FROM ops_ticket_phase LIMIT 0" >/dev/null 2>&1; then
  case "$MODE" in
    review)     VSQL="SELECT ticket FROM ops_ticket_phase
      WHERE latest_event IN ('收口','移交','收口待审') AND $REVIEW_ANTI
      ORDER BY ticket" ;;
    approved)   VSQL="SELECT ticket FROM ops_ticket_phase
      WHERE latest_event='已批' AND $APPROVED_ANTI AND $WAIPAI_HIST
      ORDER BY latest_ts,ticket" ;;
    dispatched) VSQL="SELECT ticket FROM ops_ticket_phase
      WHERE (latest_event='派单' AND latest_seat IN ('决策席','CTO'))
         OR (latest_event='挂起' AND latest_seat='施工席'
             AND latest_note LIKE 'rate-limit:auto%')
      ORDER BY latest_ts,ticket" ;;
    *)          VSQL="SELECT ticket FROM ops_ticket_phase
      WHERE (latest_event IN ('收口','移交','收口待审') AND $REVIEW_ANTI)
         OR (latest_event='已批' AND $APPROVED_ANTI AND $WAIPAI_HIST)
      ORDER BY ticket" ;;
  esac
  RAW=$(psql "$FIN_DSN" -qAt -c "$VSQL" 2>/dev/null) || exit 1
else
  # legacy 面（VIEW 未迁移落地过渡桥——判据与段B1 前逐字同，
  # dispatched=原 mech_tick 补拉窗内联判据同义收编）
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
  DISPATCH_PRED="EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
          AND e.ts=l.mt AND e.event='派单' AND e.seat IN ('决策席','CTO'))
       OR EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
          AND e.ts=l.mt AND e.event='挂起' AND e.seat='施工席'
          AND e.note LIKE 'rate-limit:auto%')"
  case "$MODE" in
    review)     PRED="($REVIEW_PRED)"; ORDER="l.ticket"
                LSQL="SELECT ticket,max(ts) mt FROM ops_event WHERE ticket<>'' GROUP BY ticket" ;;
    approved)   PRED="($APPROVED_PRED)"; ORDER="l.mt,l.ticket"
                LSQL="SELECT ticket,max(ts) mt FROM ops_event WHERE ticket<>'' GROUP BY ticket" ;;
    dispatched) PRED="($DISPATCH_PRED)"; ORDER="l.mt,l.ticket"
                LSQL="SELECT ticket,max(ts) mt FROM ops_event WHERE event IN
      ('派单','领单','拉起','收口','移交','收口待审','裁决','完成','驳回',
       '挂起','异常') AND ticket<>'' GROUP BY ticket" ;;
    *)          PRED="(($REVIEW_PRED) OR ($APPROVED_PRED))"; ORDER="l.ticket"
                LSQL="SELECT ticket,max(ts) mt FROM ops_event WHERE ticket<>'' GROUP BY ticket" ;;
  esac
  RAW=$(psql "$FIN_DSN" -qAt -c "SELECT l.ticket FROM (
      $LSQL) l WHERE $PRED
      ORDER BY $ORDER" 2>/dev/null) || exit 1
fi
printf '%s\n' "$RAW" | awk 'NF && !seen[$0]++'
