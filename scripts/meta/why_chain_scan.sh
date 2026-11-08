#!/usr/bin/env bash
# why_chain_scan.sh —— T-0464 系统级为什么链扫描（业主令「反问为什么会这样」）。
#
# 链=armed→open→fill→settle→PnL 五段实测取证（每段=24h 行数窗+最末 ts+
# 错因行）；断点=自链首起第一段产出异常（率<阈或错因非空）即归该段立案。
# 触发面（T-0464②钦定）：背离事件驱为主——扫 ops_event 未结 '议题' 行
# note 前缀「北极星背离」（T-0465 账页产出词形，seat=CEO ticket=''）；
# 周期兜底——本件由 ceo_tick 开班段挂点每拍跑，背离判据=T-0465 账页指标集
# 同源硬阈（armed滞留/零开仓/断链未领/批龄等——不另造指标源；T-0465
# ceo_northstar.sh 落地后账→议题→本扫事件径为主，兜底径同判据幂等）。
# 立案写径=verdict_tick.sh file（fin_econ_lesson kind='rootcause_diag'
# 复用 T-0431 裁定席诊断草案受理道——问链产出=草案，裁定席裁；
# roster_rw 列级 INSERT 授权面=fin_0146）。去重=同断点 24h 一案
# （entity_ref=why_chain#<段>#<yyyymmdd>+ts 窗预检双闸）。
# 纯诊断件——零 venue 写零钱径写零判据改（问链只产假设不动交易逻辑）。
set -u
ENVF="${WHYCHAIN_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
RO_DSN=$(grep '^FIN_PG_RO_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
FILE_SH=${O_WHYCHAIN_FILE_SH:-"$SCRIPT_DIR/verdict_tick.sh"}

# 段取数读面=FIN_PG_RO_DSN（o_fin_ro 窄读册——fin_proposal/fin_position/
# fin_execution/fin_order_leg(fin_0146 补授)/ops_event 白名单内）；
# RO 缺席回落 FIN_PG_DSN（roster_rw——fin_* 读权生产面实态待核，
# 兜底径如实尝试不静默改判据）。fin_econ_lesson 去重预检走 FIN_DSN
# （roster_rw SELECT,UPDATE 在册 fin_0103——RO 册无此表授权）。
[ -n "$FIN_DSN" ] || [ -n "$RO_DSN" ] || exit 0 # 双 dsn 俱缺=环境未备静默退
RDSN="${RO_DSN:-$FIN_DSN}"

q() { psql "$RDSN" -qAt -c "$1" 2>/dev/null; } # tolerate: 取数败=空串——段判据按零锚不炸
qw() { psql "$FIN_DSN" -qAt -c "$1" 2>/dev/null; } # tolerate: 同上（写册读面）
num() { case "$1" in ''|*[!0-9.eE+-]*) echo 0;; *) echo "$1";; esac; } # SQL 败件/非数→0 不炸 jq

IFS='|' read -r ARMED_N ARMED_AGED ARMED_AGEH ARMED_TS <<< "$(q "/*wc:armed*/
SELECT (SELECT count(*) FROM fin_proposal WHERE state IN ('fin_proposed','fin_armed')),
 (SELECT count(*) FROM fin_proposal WHERE state='fin_armed' AND created_at<now()-interval '4 hours'),
 COALESCE((SELECT max(extract(epoch from now()-created_at))/3600 FROM fin_proposal
   WHERE state IN ('fin_proposed','fin_armed')),0),
 COALESCE((SELECT max(created_at)::text FROM fin_proposal
   WHERE state IN ('fin_proposed','fin_armed')),'')")"
IFS='|' read -r OPEN_N OPEN_TS OPEN_EXEC <<< "$(q "/*wc:open*/
SELECT (SELECT count(*) FROM fin_position WHERE opened_at>now()-interval '24 hours'),
 COALESCE((SELECT max(opened_at)::text FROM fin_position),''),
 (SELECT count(*) FROM fin_execution WHERE created_at>now()-interval '24 hours')")"
IFS='|' read -r FILL_N FILL_ERR FILL_STATES <<< "$(q "/*wc:fill*/
SELECT (SELECT count(*) FROM fin_order_leg WHERE ts>now()-interval '24 hours'),
 (SELECT count(*) FROM fin_order_leg WHERE ts>now()-interval '24 hours'
   AND (state IN ('failed','ambiguous','rolled_back')
     OR venue_err_code IS NOT NULL OR venue_err_msg IS NOT NULL)),
 COALESCE((SELECT string_agg(DISTINCT state,',') FROM fin_order_leg
   WHERE ts>now()-interval '24 hours'
     AND (state IN ('failed','ambiguous','rolled_back')
       OR venue_err_code IS NOT NULL OR venue_err_msg IS NOT NULL)),''))")"
IFS='|' read -r SETTLE_LINGER SETTLE_STUCK SETTLE_TS <<< "$(q "/*wc:settle*/
SELECT (SELECT count(*) FROM fin_position WHERE closed_at IS NOT NULL
    AND settled_at IS NULL AND closed_at<now()-interval '1 hour'),
 (SELECT count(*) FROM fin_position WHERE closed_at IS NULL
    AND opened_at<now()-interval '7 days'),
 COALESCE((SELECT max(settled_at)::text FROM fin_position WHERE settled_at IS NOT NULL),'')")"
IFS='|' read -r PNL_24 PNL_7D PNL_WR <<< "$(q "/*wc:pnl*/
SELECT COALESCE((SELECT sum(settled_amount) FROM fin_position
    WHERE closed_at>now()-interval '24 hours'),0),
 COALESCE((SELECT sum(settled_amount) FROM fin_position
    WHERE closed_at>now()-interval '7 days'),0),
 COALESCE((SELECT count(*) FROM fin_position WHERE closed_at>now()-interval '7 days'
    AND settled_amount>0),0)||'/'||
 COALESCE((SELECT count(*) FROM fin_position WHERE closed_at>now()-interval '7 days'),0)")"
ESC_UNCL=$(q "/*wc:esc*/
SELECT count(*) FROM ops_event e WHERE e.event='升件' AND e.note NOT LIKE '升件老化%'
 AND e.ts<now()-interval '30 minutes'
 AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
   AND ((e.ticket<>'' AND x.ticket=e.ticket AND x.event IN ('裁决','完成','驳回','挂起'))
     OR x.note ~ ('升件#' || e.id::text || '([^0-9]|$)')))")
EVENTS=$(q "/*wc:events*/
SELECT COALESCE(string_agg(note, E'\n' ORDER BY id),'') FROM ops_event
 WHERE event='议题' AND seat='CEO' AND note LIKE '北极星背离%' AND ts>now()-interval '24 hours'")

# 段异常判据（率<阈或错因非空——阈=T-0465 账页硬阈同源）：
#   armed=滞留>20 或最老龄>24h；open=末开仓龄>24h（零开仓窗）；
#   fill=错因行非空；settle=滞留平仓未结/超龄未平行非空；pnl=7d 净额<0。
OPEN_AGEH=$(q "/*wc:openage*/
SELECT CASE WHEN (SELECT max(opened_at) FROM fin_position) IS NULL THEN 99999
 ELSE extract(epoch from now()-(SELECT max(opened_at) FROM fin_position))/3600 END")
ARMED_BAD=0; [ "${ARMED_N:-0}" -gt 20 ] 2>/dev/null && ARMED_BAD=1
[ "${ARMED_AGEH%.*}" -gt 24 ] 2>/dev/null && ARMED_BAD=1
OPEN_BAD=0;  [ "${OPEN_AGEH%.*}" -gt 24 ] 2>/dev/null && OPEN_BAD=1
FILL_BAD=0;  [ "${FILL_ERR:-0}" -gt 0 ] 2>/dev/null && FILL_BAD=1
SETTLE_BAD=0; { [ "${SETTLE_LINGER:-0}" -gt 0 ] || [ "${SETTLE_STUCK:-0}" -gt 0 ]; } 2>/dev/null && SETTLE_BAD=1
PNL_BAD=0;   awk "BEGIN{exit !(${PNL_7D:-0} < 0)}" 2>/dev/null && PNL_BAD=1

# 断点=链首首异常段（armed→open→fill→settle→pnl 序钉死）。
BREAK=""
[ "$ARMED_BAD" = 1 ] && BREAK=armed
[ -z "$BREAK" ] && [ "$OPEN_BAD" = 1 ] && BREAK=open
[ -z "$BREAK" ] && [ "$FILL_BAD" = 1 ] && BREAK=fill
[ -z "$BREAK" ] && [ "$SETTLE_BAD" = 1 ] && BREAK=settle
[ -z "$BREAK" ] && [ "$PNL_BAD" = 1 ] && BREAK=pnl

# 事件驱触发面：T-0465 议题行在案=背离已立——链全通亦立案（断点=链外，
# 归因线索=议题行 metric 名）；断链未领>0=ops 背离同径。
TRIG=""
[ -n "$EVENTS" ] && TRIG="northstar_topic"
[ "${ESC_UNCL:-0}" -gt 0 ] 2>/dev/null && TRIG="${TRIG:+$TRIG+}escalation_unclaimed"

if [ -z "$BREAK" ] && [ -z "$TRIG" ]; then
  exit 0 # 零背离零异常——静默空跑（兜底拍零烧）
fi
[ -z "$BREAK" ] && BREAK=chain_clear # 断点不在钱链五段——链外背离如实记

REF="why_chain#${BREAK}#$(date +%Y%m%d)"
# 24h 同断点去重预检（entity_ref 日桶+ts 窗双闸——同向背离不重复立案）。
DUP=$(qw "/*wc:dup*/
SELECT count(*) FROM fin_econ_lesson WHERE kind='rootcause_diag'
 AND entity_ref LIKE 'why_chain#${BREAK}#%' AND ts>now()-interval '24 hours'")
[ "${DUP:-0}" -gt 0 ] 2>/dev/null && exit 0

NUMBERS=$(jq -nc \
  --arg break "$BREAK" --arg trig "$TRIG" \
  --arg armed_ts "$ARMED_TS" --arg open_ts "$OPEN_TS" --arg fill_states "$FILL_STATES" \
  --arg settle_ts "$SETTLE_TS" --arg pnl_wr "$PNL_WR" \
  --arg events "$EVENTS" \
  --argjson armed_n "$(num "$ARMED_N")" --argjson armed_aged "$(num "$ARMED_AGED")" \
  --argjson armed_ageh "$(num "$ARMED_AGEH")" \
  --argjson open_n "$(num "$OPEN_N")" --argjson open_exec "$(num "$OPEN_EXEC")" \
  --argjson open_ageh "$(num "$OPEN_AGEH")" \
  --argjson fill_n "$(num "$FILL_N")" --argjson fill_err "$(num "$FILL_ERR")" \
  --argjson settle_linger "$(num "$SETTLE_LINGER")" --argjson settle_stuck "$(num "$SETTLE_STUCK")" \
  --argjson pnl24 "$(num "$PNL_24")" --argjson pnl7d "$(num "$PNL_7D")" \
  --argjson esc "$(num "$ESC_UNCL")" \
  '{case:{chain:"armed→open→fill→settle→PnL",break_segment:$break,
     trigger:$trig,
     armed:{open_rows:$armed_n,aged_4h:$armed_aged,
       oldest_age_h:$armed_ageh,latest_ts:$armed_ts},
     open:{new_24h:$open_n,exec_24h:$open_exec,
       last_open_age_h:$open_ageh,latest_ts:$open_ts},
     fill:{legs_24h:$fill_n,err_rows:$fill_err,
       err_states:$fill_states},
     settle:{closed_unsettled_1h:$settle_linger,
       open_over_7d:$settle_stuck,latest_settled_ts:$settle_ts},
     pnl:{net_24h:$pnl24,net_7d:$pnl7d,win_total_7d:$pnl_wr}},
   sources:["系统级为什么链（T-0464——北极星背离→五段取证断点归段）"],
   seeds:[],
   evidence_queries:[
     "SELECT state,count(*),min(created_at),max(created_at) FROM fin_proposal WHERE state IN ('"'"'fin_proposed'"'"','"'"'fin_armed'"'"') GROUP BY state",
     "SELECT state,count(*) FROM fin_order_leg WHERE ts>now()-interval '"'"'24 hours'"'"' GROUP BY state",
     "SELECT count(*) FILTER (WHERE settled_at IS NULL),count(*) FROM fin_position WHERE closed_at IS NOT NULL",
     "SELECT sum(settled_amount),count(*) FROM fin_position WHERE closed_at>now()-interval '"'"'7 days'"'"'"],
   northstar_topics:$events,
   unclaimed_escalations:$esc,
   target_output:"最可能断点根因+证据链（五段锚列）+建议修法面（草案——假设非判词，禁自批；族标=chain:<段> 或 novel:<一句定性>）"}')

"$FILE_SH" file "$REF" "$NUMBERS" || true # tolerate: 立案写径败不炸扫拍——下拍重试（24h 去重防重）
