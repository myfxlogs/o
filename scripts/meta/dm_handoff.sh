#!/usr/bin/env bash
# dm_handoff.sh —— 交接对账哨兵消费窗（T-0459 业主令「事事有交接，事事有
# 回应」——每「交」行必有「接」行配对；交行龄>阈∧同票无接行∧未报过=断链
# 升件。与 T-0458 活卡互补：卡管「看得见」，本件管「没人接会叫」）。
# 配对表 handoff_pairs.tsv 单源（交event|接行集|阈秒|说明——gauges.tsv 同
# 形）；在案不双哨（表不列=不扫）：升件→escalation_unclaimed 量表（T-0437
# 30min 连鸣）、挂起→dm_suspend 看门狗（T-0365）。判据（CTO 裁定 2026-11-08）：
#   终态杀票——票内最新生命周期事件∈{完成,驳回,作废,处置}→该票交行全免报
#     （生命周期域=本表交∪接∪终态——表变判据随变，单源不漂）；
#   免报词表——交行 note 命中 作废|测试票|人工径 免报（DBP3 词表缺口同型防）；
#   evergreen 去重——已存升件行 note 含「断链:<交event>:<tid>——交行#<id>」
#     字面→零再报（不用 24h 窗，断链不因窗过期复鸣；持续压归
#     escalation_unclaimed 30min 连鸣——消音=处置落行携 升件#<断链行id> 引证，
#     量表既有一式，不新造鸣径）；
#   扫窗——交行 ts>now()-7d 且龄>阈；首跑捕现存断链=预期捕账非误报
#     （终态/词表滤后余者=真链断），每交行捕一次后永不再报；
#   自激防——接行集不含「升件」（表侧钦定），本哨自鸣行永不充当接行。
# 升件=escalate.sh 无类（断链裁断上下文各异不入判例面，dm_suspend 同款）；
# 词形：断链:<交event>:<tid>——交行#<id> <交seat> 「<note截40字>」龄<X>m>阈<Y>m
# ——携交行 id 引证。SQL 参数全出自库内受控配对表（token 字面硬界校验）。
# FIN_DSN 缺=静默退（拍壳同律）；配对表缺/畸形/扫描败=exit 1（T-0372：败≠
# 空集——拍壳升件「交接哨兵本体败」）。本件只被 dm_tick.sh 尾段零 LLM 段调用。
set -u
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$FIN_DSN" ] || exit 0
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
PAIRS_TSV=${O_HANDOFF_PAIRS_TSV:-"$SCRIPT_DIR/handoff_pairs.tsv"}
ESCALATE_SH=${O_HANDOFF_ESCALATE_SCRIPT:-"$SCRIPT_DIR/escalate.sh"}
SCAN_WINDOW=${O_HANDOFF_SCAN_WINDOW:-604800} # 扫窗 7d（秒）——首跑捕现存断链属预期捕账
case "$SCAN_WINDOW" in ''|*[!0-9]*) echo "[dm-handoff] 扫窗非数: $SCAN_WINDOW" >&2; exit 1;; esac

# 事件/接行 token 字面硬界——禁空/引号/反斜杠/通配/空白/竖线（SQL 等值与
# LIKE 字面域；畸形=配置败 fail-loud exit 1，禁静默跳=盲区）。
tok_ok() {
  case "$1" in
    ''|*"'"*|*'"'*|*"%"*|*"_"*|*"\\"*|*"*"*|*"|"*) return 1 ;;
  esac
  case "$1" in *[[:space:]]*) return 1 ;; esac
  return 0
}

# 配对表加载：#头注/空行跳；交event|接行集(逗号)|阈秒|说明。
LC='完成 驳回 作废 处置' # 生命周期域种子=终态集（交∪接运行时并入）
P_EV=(); P_RECV=(); P_THR=()
N=0
while IFS= read -r line; do
  N=$((N+1))
  line=${line%$'\r'}
  [ -n "$line" ] || continue
  case "$line" in '#'*) continue ;; esac
  IFS='|' read -r ev recv thr _ <<<"$line"
  tok_ok "$ev"   || { echo "[dm-handoff] $PAIRS_TSV:$N 交event token 畸形: $ev" >&2; exit 1; }
  tok_ok "$thr"  || { echo "[dm-handoff] $PAIRS_TSV:$N 阈秒 token 畸形: $thr" >&2; exit 1; }
  [ -n "$recv" ] || { echo "[dm-handoff] $PAIRS_TSV:$N 接行集空" >&2; exit 1; }
  RSQL=
  IFS=',' read -ra RL <<<"$recv"
  for r in "${RL[@]}"; do
    tok_ok "$r" || { echo "[dm-handoff] $PAIRS_TSV:$N 接行 token 畸形: $r" >&2; exit 1; }
    RSQL+="'$r'," # 接行集不含「升件」=自激防（表侧钦定——升件行永不充当接行）
    case " $LC " in *" $r "*) ;; *) LC="$LC $r" ;; esac
  done
  RSQL=${RSQL%,}
  case " $LC " in *" $ev "*) ;; *) LC="$LC $ev" ;; esac
  P_EV+=("$ev"); P_RECV+=("$RSQL"); P_THR+=("$thr")
done < "$PAIRS_TSV" || { echo "[dm-handoff] 配对表读败: $PAIRS_TSV" >&2; exit 1; }
[ "${#P_EV[@]}" -gt 0 ] || { echo "[dm-handoff] 配对表零行: $PAIRS_TSV" >&2; exit 1; }
LC_SQL=$(printf '%s\n' $LC | sort -u | sed "s/.*/'&'/" | paste -sd, -) # 生命周期域=交∪接∪终态

for i in "${!P_EV[@]}"; do
  ev=${P_EV[$i]} recv=${P_RECV[$i]} thr=${P_THR[$i]}
  # 按表逐行反连（单查自含四滤——接行/evergreen/终态杀票/词表，谓词钉见桩测）。
  ROWS=$(psql "$FIN_DSN" -qAt -F $'\x1f' -c "
SELECT e.id,e.ticket,e.seat,translate(left(e.note,40),E'\r\n','  '),extract(epoch from now()-e.ts)::bigint
FROM ops_event e
WHERE e.event='$ev' AND e.ticket<>''
  AND e.ts>now()-interval '${SCAN_WINDOW} seconds'
  AND e.ts<now()-interval '$thr seconds'
  AND e.note !~ '(作废|测试票|人工径)'
  AND NOT EXISTS (SELECT 1 FROM ops_event r WHERE r.ticket=e.ticket
    AND r.event IN ($recv) AND (r.ts>e.ts OR (r.ts=e.ts AND r.id>e.id)))
  AND NOT EXISTS (SELECT 1 FROM ops_event z WHERE z.event='升件'
    AND z.note LIKE '断链:$ev:'||e.ticket||'——交行#' || e.id::text || ' %')
  AND NOT EXISTS (SELECT 1 FROM ops_event t WHERE t.ticket=e.ticket
    AND t.event IN ('完成','驳回','作废','处置')
    AND NOT EXISTS (SELECT 1 FROM ops_event l WHERE l.ticket=e.ticket
      AND l.event IN ($LC_SQL) AND (l.ts>t.ts OR (l.ts=t.ts AND l.id>t.id))))
ORDER BY e.id") || exit 1 # T-0372：扫描败≠空集——非零由消费面 dm_tick 升件显报
  [ -n "$ROWS" ] || continue
  printf '%s\n' "$ROWS" | while IFS=$'\x1f' read -r rid tid seat note age; do
    [ -n "$tid" ] || continue
    case "$tid" in *[!A-Za-z0-9_-]*) continue ;; esac # dm_suspend 同款——伪票/畸形票不投递
    case "$rid" in ''|*[!0-9]*) continue ;; esac
    case "$age" in ''|*[!0-9]*) continue ;; esac
    [ "$age" -gt "$thr" ] || continue # 龄壳侧双检（SQL 已滤——防御钉，防扫描/取数间时钟漂）
    aget=$((age/60)); thrt=$((thr/60))
    "$ESCALATE_SH" "CTO" "$tid" "" "" \
      "断链:$ev:$tid——交行#$rid $seat 「$note」龄${aget}m>阈${thrt}m" \
      || true # tolerate: 升件尽力而为—败不阻哨兵巡（evergreen 未落=下拍重报自愈）
  done
done
exit 0
