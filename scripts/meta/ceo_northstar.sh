#!/usr/bin/env bash
# ceo_northstar.sh —— T-0465 CEO 北极星账（业主令「CEO 开班先看结果再问
# 为什么」钦定面）：七项指标直查段（独立派生面非寄 gauges——经营账≠
# 巡检面）+7d 滚动基线（ops_event '北极星账' 快照行 kv 中位）+硬阈背离
# →议题行（seat=CEO event='议题' ticket='' note='北极星背离:<指标>=<值>
#  vs 基线<基线>——问链引T-0464'，24h 同指标去重——depblind 同例）+
# 为什么型背离调 T-0464 问链（why_chain_scan.sh 在案才调；未合缺席合
# 法——其 ceo_tick 兜底拍同判据幂等接）。
# 三步序位：ceo_tick 开班段先调本件——账→议题→队固定（议题行=ceo_queue
# 班内件源，背离自生成队件点火原空拍）。
# 取数面：FIN_PG_RO_DSN 窄读优先（o_fin_ro 白名单在册：fin_position/
# fin_proposal/ops_event）回落 FIN_PG_DSN；写面（快照/议题行）=FIN_PG_DSN。
# 部署批龄=deploy_window_due.sh 同源探测（OPS_ESC_SH=/bin/true 压其升件
# 副作用——探针只读不代报；批空/有挡/应发同出数，环境缺/查询败=NA）。
# 硬阈（T-0465②钦定）：净PnL7d<0 连2拍（prev 快照 24h 内且同负）/
# 末开仓龄>24h/armed滞留 n>20 或最老龄>24h/断链未领>0/部署批龄>24h/
# 候水票最老龄>4h（T-0463 量表同阈——持续候水容量信号非为什么型不问链）。
# 胜率/平仓笔数=展示指标无阈（②未列）。
# 用法: ceo_northstar.sh——页落 $O_NORTHSTAR_DIR/northstar.txt 并 stdout 同打；
# rc 恒 0（拍壳件——败因载页面/行内不炸开班）。
set -u
ENVF="${NORTHSTAR_ENV_FILE:-${DM_ENV_FILE:-/opt/o-research/env}}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
RO_DSN=$(grep '^FIN_PG_RO_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$FIN_DSN" ] || [ -n "$RO_DSN" ] || exit 0
RDSN="${RO_DSN:-$FIN_DSN}"
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
DEPLOY_SH=${O_NORTHSTAR_DEPLOY_SH:-"$SCRIPT_DIR/deploy_window_due.sh"}
WHY_SH=${O_NORTHSTAR_WHY_SH:-"$SCRIPT_DIR/why_chain_scan.sh"}
OUT_DIR=${O_NORTHSTAR_DIR:-/var/tmp/o-northstar}
PAGE="$OUT_DIR/northstar.txt"
mkdir -p "$OUT_DIR" 2>/dev/null || true # tolerate: 页目录建败=stdout 仍出，文件面缺席不炸拍

QC() { printf '%s' "${1:-}" | sed "s/'/''/g"; }
num() { case "$1" in ''|*[!0-9.eE+-]*) echo 0;; *) echo "$1";; esac; } # 空/非数→0 不炸 awk（ERR 面另行显式标）
snapn() { case "$1" in ERR|NA|''|*[!0-9.eE+-]*) echo NA;; *) awk -v v="$1" 'BEGIN{printf "%.6g",v+0}';; esac; } # 快照面：取数败/非数=NA 缺席中位不写真 0
snapf() { case "$1" in ERR|NA|''|*[!0-9.eE+-]*) echo NA;; *) awk -v v="$1" 'BEGIN{printf "%.2f",v+0}';; esac; }

# ── 指标取数（/*ns:key*/ 探针锚=桩测谱形钉）────────────────────────
PNL_RC=0; PNL=$(psql "$RDSN" -qAt -c "/*ns:pnl*/
SELECT COALESCE((SELECT sum(settled_amount) FROM fin_position WHERE closed_at>now()-interval '24 hours'),0),
       COALESCE((SELECT sum(settled_amount) FROM fin_position WHERE closed_at>now()-interval '7 days'),0),
       (SELECT count(*) FROM fin_position WHERE closed_at>now()-interval '7 days'),
       COALESCE((SELECT count(*) FROM fin_position WHERE closed_at>now()-interval '7 days' AND settled_amount>0),0)" 2>/dev/null) || PNL_RC=$?
IFS='|' read -r PNL24 PNL7D CLOSED7D WINS7D <<< "$PNL"
PNL24=${PNL24:-ERR}; PNL7D=${PNL7D:-ERR}; CLOSED7D=${CLOSED7D:-ERR}; WINS7D=${WINS7D:-ERR}
[ "$PNL_RC" -ne 0 ] && { PNL24=ERR; PNL7D=ERR; CLOSED7D=ERR; WINS7D=ERR; }

ARMED_RC=0; ARMED=$(psql "$RDSN" -qAt -c "/*ns:armed*/
SELECT count(*), COALESCE(max(extract(epoch from now()-created_at))/3600,0)
FROM fin_proposal WHERE state IN ('fin_proposed','fin_armed')" 2>/dev/null) || ARMED_RC=$?
IFS='|' read -r ARMED_N ARMED_AGEH <<< "$ARMED"
[ "$ARMED_RC" -ne 0 ] && { ARMED_N=ERR; ARMED_AGEH=ERR; }
ARMED_N=${ARMED_N:-0}; ARMED_AGEH=${ARMED_AGEH:-0}

OPEN_RC=0; OPEN=$(psql "$RDSN" -qAt -c "/*ns:open*/
SELECT (SELECT count(*) FROM fin_position WHERE opened_at>now()-interval '24 hours'),
       COALESCE(extract(epoch from now()-(SELECT max(opened_at) FROM fin_position))/3600, 99999)" 2>/dev/null) || OPEN_RC=$?
IFS='|' read -r OPEN24 OPEN_AGEH <<< "$OPEN"
[ "$OPEN_RC" -ne 0 ] && { OPEN24=ERR; OPEN_AGEH=ERR; }
OPEN24=${OPEN24:-0}; OPEN_AGEH=${OPEN_AGEH:-0}

ESC_RC=0; ESC_UNCL=$(psql "$RDSN" -qAt -c "/*ns:esc*/
SELECT count(*) FROM ops_event e WHERE e.event='升件' AND e.note NOT LIKE '升件老化%'
 AND e.ts<now()-interval '30 minutes'
 AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
   AND ((e.ticket<>'' AND x.ticket=e.ticket AND x.event IN ('裁决','完成','驳回','挂起'))
     OR x.note ~ ('升件#' || e.id::text || '([^0-9]|\$)')))" 2>/dev/null) || ESC_RC=$?
[ "$ESC_RC" -ne 0 ] && ESC_UNCL=ERR
ESC_UNCL=${ESC_UNCL:-0}

WATER_RC=0; WATER=$(psql "$RDSN" -qAt -c "/*ns:water*/
SELECT count(*), COALESCE(max(extract(epoch from now()-ts))/3600,0) FROM (
  SELECT DISTINCT ON (ticket) ticket,event,ts FROM ops_event WHERE ticket<>'' ORDER BY ticket,ts DESC,id DESC
) t WHERE t.event='候水'" 2>/dev/null) || WATER_RC=$?
IFS='|' read -r WATER_N WATER_AGEH <<< "$WATER"
[ "$WATER_RC" -ne 0 ] && { WATER_N=ERR; WATER_AGEH=ERR; }
WATER_N=${WATER_N:-0}; WATER_AGEH=${WATER_AGEH:-0}

# 上拍快照（24h 内——「连2拍」相邻拍语义：隔夜/停摆后首拍不连计）
PREV_NOTE=$(psql "$RDSN" -qAt -c "/*ns:prev*/
SELECT note FROM ops_event WHERE event='北极星账' AND seat='CEO'
 AND ts>now()-interval '24 hours' ORDER BY ts DESC,id DESC LIMIT 1" 2>/dev/null | head -1) # tolerate: 上拍查询败=空串——连2拍判据按无前拍不误鸣

# ── 部署批龄/积压（deploy_window_due 同源探测——升件副作用压成只读）──
DEP_AGEH=NA; DEP_COMMITS=NA; DEP_STATE=NA
if [ -f "$DEPLOY_SH" ]; then
  DOUT=$(timeout 90 env OPS_ESC_SH=/bin/true DM_ENV_FILE="$ENVF" bash "$DEPLOY_SH" 2>/dev/null) # tolerate: 部署窗探测超时/败=dep 面 NA 不阻账页（探针只读；env 同源传递防双 env 漂）
  DEP_STATE=$(printf '%s\n' "$DOUT" | awk -F= '/^state=/{print $2; exit}')
  DEP_REASON=$(printf '%s\n' "$DOUT" | grep '批非空 commits=' | tail -1) # 批面 reason 行非末行——挡件/半截等后行不冲批值
  case "$DEP_REASON" in
    *"批非空 commits="*)
      DEP_COMMITS=$(printf '%s' "$DEP_REASON" | sed -n 's/.*commits=\([0-9][0-9]*\).*/\1/p')
      DEP_AGE_S=$(printf '%s' "$DEP_REASON" | sed -n 's/.*批首件龄=\([0-9][0-9]*\)s.*/\1/p')
      [ -n "$DEP_AGE_S" ] && DEP_AGEH=$(awk -v s="$DEP_AGE_S" 'BEGIN{printf "%.1f", s/3600}')
      DEP_COMMITS=${DEP_COMMITS:-NA} ;;
    *)
      case "$DEP_STATE" in
        批空) DEP_COMMITS=0; DEP_AGEH=0 ;;
        *) DEP_STATE=${DEP_STATE:-探测败} ;;
      esac ;;
  esac
fi

# ── 基线/去重/议题件 ────────────────────────────────────────────────
med() { # $1=快照 kv 键（ASCII——与快照行字段同名）→ 7d 中位 %.4g（无快照/查询败=na）
  local v
  v=$(psql "$RDSN" -qAt -c "SELECT percentile_cont(0.5) WITHIN GROUP (ORDER BY kv[2]::float8)
    FROM ops_event e, regexp_matches(e.note,'([a-z_0-9]+)=(-?[0-9.]+)','g') AS mm(kv)
    WHERE e.event='北极星账' AND e.seat='CEO' AND e.ts>now()-interval '7 days'
      AND kv[1]='$1'" 2>/dev/null | head -1)
  case "$v" in ''|*[!0-9.eE+-]*) printf 'na\n';; *) awk -v x="$v" 'BEGIN{printf "%.4g\n",x+0}';; esac
}
TOPIC_NEW=0; WHY_ACTIVE=0
emit_topic() { # $1=议题指标键(=快照kv键——去重LIKE锚+基线键同名) $2=值面 $3=why型(1=问链/0=容量已知径)
  local key=$1 disp=$2 why=$3 base
  [ -n "$FIN_DSN" ] || return 0
  base=$(med "$key")
  local dup
  dup=$(psql "$RDSN" -qAt -c "SELECT 1 FROM ops_event WHERE event='议题' AND seat='CEO'
    AND starts_with(note, '北极星背离:$(QC "$key")=') AND ts>now()-interval '24 hours' LIMIT 1" 2>/dev/null) # tolerate: 去重查询败=空答→照常落行（fail-open 同 ops_esc 径；starts_with 防键内_被 LIKE 通配）
  [ "$dup" = "1" ] && return 0
  local note="北极星背离:$key=$disp vs 基线${base:-na}——问链引T-0464"
  psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note) VALUES('CEO','议题','','$(QC "$note")')" >/dev/null 2>&1 || true # tolerate: 议题行落账尽力而为——写败下拍重评（24h 去重窗内不重复落）
  TOPIC_NEW=$((TOPIC_NEW+1))
  [ "$why" = 1 ] && WHY_ACTIVE=1
}

# 硬阈判据（②钦定——ERR 面不评不误鸣：取数败指标跳过背离判）
case "$PNL7D" in ERR) ;; *)
  if awk "BEGIN{exit !(${PNL7D} < 0)}" 2>/dev/null; then
    PREV_PNL=$(printf '%s' "$PREV_NOTE" | grep -oE 'pnl7d=-?[0-9.]+' | head -1 | cut -d= -f2)
    awk "BEGIN{exit !(${PREV_PNL:-0} < 0)}" 2>/dev/null \
      && emit_topic 'pnl7d' "$(awk -v v="$PNL7D" 'BEGIN{printf "%.4f",v}')" 1
  fi ;; esac
case "$OPEN_AGEH" in ERR) ;; *)
  awk "BEGIN{exit !(${OPEN_AGEH} > 24)}" 2>/dev/null \
    && emit_topic 'open_age_h' "$(awk -v v="$OPEN_AGEH" 'BEGIN{printf "%.1f",v}')h" 1 ;; esac
case "$ARMED_N" in ERR) ARMED_BAD=0 ;; *)
  ARMED_BAD=0
  awk "BEGIN{exit !(${ARMED_N} > 20)}" 2>/dev/null && ARMED_BAD=1
  case "$ARMED_AGEH" in ERR) ;; *)
    awk "BEGIN{exit !(${ARMED_AGEH} > 24)}" 2>/dev/null && ARMED_BAD=1 ;; esac
  [ "$ARMED_BAD" = 1 ] && emit_topic 'armed_n' "n=$(num "$ARMED_N") age=$(awk -v v="$ARMED_AGEH" 'BEGIN{printf "%.1f",v}')h" 1 ;; esac
case "$ESC_UNCL" in ERR) ;; *)
  [ "${ESC_UNCL:-0}" -gt 0 ] 2>/dev/null && emit_topic 'esc_uncl' "$ESC_UNCL" 1 ;; esac
case "$DEP_AGEH" in NA|ERR) ;; *)
  awk "BEGIN{exit !(${DEP_AGEH} > 24)}" 2>/dev/null \
    && emit_topic 'dep_age_h' "$(awk -v v="$DEP_AGEH" 'BEGIN{printf "%.1f",v}')h commits=${DEP_COMMITS:-NA}" 0 ;; esac
case "$WATER_AGEH" in ERR) ;; *)
  awk "BEGIN{exit !(${WATER_AGEH} > 4)}" 2>/dev/null \
    && emit_topic 'water_age_h' "n=$(num "$WATER_N") age=$(awk -v v="$WATER_AGEH" 'BEGIN{printf "%.1f",v}')h" 0 ;; esac

# ── 快照行（7d 滚动基线原料——每拍一行 kv 面；dep NA=非数不入中位）──
if [ -n "$FIN_DSN" ]; then
  SNAP="北极星账 pnl24=$(snapn "$PNL24") pnl7d=$(snapn "$PNL7D") closed7d=$(snapn "$CLOSED7D") wins7d=$(snapn "$WINS7D") armed_n=$(snapn "$ARMED_N") armed_age_h=$(snapf "$ARMED_AGEH") open24h=$(snapn "$OPEN24") open_age_h=$(snapf "$OPEN_AGEH") esc_uncl=$(snapn "$ESC_UNCL") dep_age_h=$(snapn "$DEP_AGEH") dep_commits=$(snapn "$DEP_COMMITS") water_n=$(snapn "$WATER_N") water_age_h=$(snapf "$WATER_AGEH")"
  psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note) VALUES('CEO','北极星账','','$(QC "$SNAP")')" >/dev/null 2>&1 || true # tolerate: 快照行尽力而为——基线原料缺拍不阻账页/议题（下拍续记）
fi

# ── 账页渲染（stdout+页文件同打——「公司这拍在不在赚钱」一题面）───────
fmt2() { awk -v v="$1" 'BEGIN{if(v==""||v=="ERR")printf "%s",v; else printf "%.4f",v+0}'; }
fmt1() { awk -v v="$1" 'BEGIN{if(v==""||v=="ERR")printf "%s",v; else printf "%.1f",v+0}'; }
WINRATE=—; if [ "$CLOSED7D" != "ERR" ] && [ "$CLOSED7D" != "0" ] && [ "$WINS7D" != "ERR" ]; then
  WINRATE=$(awk -v w="$WINS7D" -v c="$CLOSED7D" 'BEGIN{printf "%.0f", w*100/c}')%
fi
DEP_SHOW="探测败(${DEP_STATE})"
if [ "$DEP_STATE" = 批空 ]; then DEP_SHOW="批空"
elif [ "$DEP_COMMITS" != "NA" ]; then DEP_SHOW="龄 $(fmt1 "$DEP_AGEH")h·积压 ${DEP_COMMITS} commits"; fi
PAGE_BODY=$(cat <<EOF
═══ 北极星账 $(date -Is) ═══
【钱面】净PnL 24h=$(fmt2 "$PNL24") 7d=$(fmt2 "$PNL7D") USDT｜平仓7d=${CLOSED7D} 笔·胜率 ${WINRATE}｜armed滞留=${ARMED_N} 件·最老 $(fmt1 "$ARMED_AGEH")h｜新开仓24h=${OPEN24}（末开仓龄 $(fmt1 "$OPEN_AGEH")h）
【运营面】断链未领升件=${ESC_UNCL}｜部署批=${DEP_SHOW}｜候水票=${WATER_N}（最老 $(fmt1 "$WATER_AGEH")h）
【基线】7d滚动中位（北极星账快照行驱动）：pnl7d=$(med pnl7d) armed_n=$(med armed_n) open_age_h=$(med open_age_h)
【背离】${TOPIC_NEW} 件新议题（24h 同指标去重窗内重评不重落）
EOF
)
printf '%s\n' "$PAGE_BODY"
printf '%s\n' "$PAGE_BODY" > "$PAGE" 2>/dev/null || true # tolerate: 页文件写败=stdout 已出——账页文件面缺席不炸拍

# ── 为什么型背离→T-0464 问链（在案才调——容量/已知在径不问链）───────
if [ "$WHY_ACTIVE" = 1 ] && [ -f "$WHY_SH" ]; then
  timeout 120 env WHYCHAIN_ENV_FILE="$ENVF" bash "$WHY_SH" >/dev/null 2>&1 || true # tolerate: 问链扫败不炸账拍——其 ceo_tick 兜底拍同判据幂等接（24h 案去重；env 同源传递）
fi
exit 0
