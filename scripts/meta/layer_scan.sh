#!/usr/bin/env bash
# layer_scan.sh —— T-0507 CEO自运感知票A 层扫窗：五片不变量机械扫
# （提案-CEO自运感知三缺①——感知面只报不判；mech_tick.sh 新窗承载，
# 零新守护件零新 unit；本件只被 mech_tick 尾段调用）。
#
# 五片（票面钦定枚举，类目级定性不诊断——「矛盾存在」即报，「为什么」
# 归票B诊断席）：
#   a 队列一致：dispatched 龄>30min 无在飞 unit（相位=dispatched 本身
#     已含「派单后零拉起/领单」语义——fin_0147/0153 相位投影钉）；
#     pulled 龄>30min 无存活 unit（限机器拉起证在案者——领单无 unit
#     对人工/交互径合法，META-Dev 心跳行∪拉起事件=spawn 实证集）；
#     review/approval_pending 老化>2h；draft 滞留>1h。
#   b 相位↔实体：非终相位（done/rejected/void 以外）T-票无票件=幽灵行；
#     done/ 件在而相位非 done 且件龄>30min（dm_ledger 每拍补登宽限）；
#     phase=done 而无收口族事件链（7d 扫窗——dm_handoff 同式首跑不洪史）。
#   c 容量水位：候选集（dm_queue dispatched 单源）非空∧拍间拉起/领单
#     事件=0 之 streak——2 拍=拉起-空转早期水位、3 拍=spawn-连败定性
#     （票面「超2拍/连续3拍」按同一 streak 阶梯落两档）；候水标记票
#     （rc75 合法候队）剔出候选集不计败拍；worker 退出（机器拉起证
#     在案∧无在飞 unit）后无收口事件链>30min。
#   d 账本真值（抽查非全量）：最近 5 张 done 相位票——收口 commit 在
#     master 可解（git log --grep 实解）+票面回填存在（文件在且状态
#     行含 完成|待验）。
#   e 外事聚合：origin/master 落后龄>2h（本地 master 未推件最老龄——
#     deploy 窗 HEAD==origin/master 收敛前提被推径卡死的聚合信号）；
#     deploy 批龄>12h（deploy_window_due.sh 同源探测只读复用——已有
#     探针不重复造，本窗只做聚合判断，OPS_ESC_SH=/bin/true 压其升件）。
#   f 用户端点健康（票面修订补面——2026-11-09 驾驶舱 502 实证业主令）：
#     用户同路径探 dashboard SPA 根+/o.dashboard.v1.CoreService/Ping
#     期望码 200 对拍；部署窗豁免——ops_event event='部署' 行龄≤5min
#     内端点红=image 重启瞬断不报，仍红>5min 或无部署行而红=升件
#     「端点失健」。端点清单初期=dashboard 一处（票面钦定，扩面另立）。
#
# 升件行：seat=MECH event=升件 note=「层扫定性<类目>:<对象>——<详情>」
#   ——走 ops_esc.sh 24h 前缀去重（同对象同类目 24h 一行不刷屏=票面
#   钦定 dedup；类目/对象稳定字面供 T-0508 signature=「层扫:<类目>:<对象>」
#   消重对拍）。seat=MECH=T-0506 语义闸钦定写席（verdict_reflow 同例）。
#   票域件 ticket=<tid>；系统面件（容量/外事）ticket='' 对象=稳定键。
#   机械升件留 CTO 壳轨不串味（CEO席职卡钦定）——MECH+升件不入 CEO 队。
# digest 行：每窗落 seat=MECH event='层扫' ticket='层扫'（伪键——ceo_queue
#   DISTINCT 坍缩单槽=最新体检行；终态行 ticket='层扫' 出队后新 digest
#   再入队）。note=相位分布计数+五面异常计数+面可判性。
#
# 阈值常量带出处注（票面钦定/初值——禁魔术数）；枚举只读
# ops_event/ops_ticket_phase/票件/systemd --user/git——零写生产面，
# 唯一写径=ops_event 升件+层扫两行（经 ops_esc/直 INSERT 账面）。
# FIN_DSN 缺=静默退（拍壳同律）；扫描败=exit 1（T-0372 败≠空集——
#   mech_tick 升件「层扫本体败」）；systemctl 败=unit 面不可判降级
#   （不误报全场无 unit——digest 载 unit=unk）。
set -u
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$FIN_DSN" ] || exit 0
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO=${O_LAYER_SCAN_REPO:-/opt/o}
TASK_DIR=${O_LAYER_SCAN_TASK_DIR:-$REPO/docs/tasks}
DONE_DIR=${O_LAYER_SCAN_DONE_DIR:-$TASK_DIR/done}
STATE_DIR=${O_LAYER_SCAN_STATE_DIR:-/var/tmp/o-layer-scan}
SYSTEMCTL=${O_LAYER_SCAN_SYSTEMCTL:-systemctl}
DM_QUEUE_SH=${O_DM_QUEUE_SH:-"$SCRIPT_DIR/dm_queue.sh"}
OPS_ESC_SH=${O_LAYER_SCAN_OPS_ESC_SH:-"$SCRIPT_DIR/ops_esc.sh"}
MANUAL_LANE_SH=${O_MANUAL_LANE_SH:-"$SCRIPT_DIR/manual_lane_check.sh"}
DEPLOY_SH=${O_LAYER_SCAN_DEPLOY_SH:-"$SCRIPT_DIR/deploy_window_due.sh"}
export OPS_ESC_ENV_FILE="$ENVF" # ops_esc 自走 env 径与本件同 DSN 源

# ── 阈值常量（出处注——票面钦定值/初值标初值）────────────────────────
DISPATCH_STALE_S=1800      # 票面a①：dispatched 龄>30min（钦定初值）
PULLED_UNIT_STALE_S=1800   # 票面a②未单列龄阈——取 dispatched 同族 30min
REVIEW_STALE_S=7200        # 票面a③：review/approval 老化>2h（钦定初值）
DRAFT_STALE_S=3600         # 票面a④：draft 滞留>1h（钦定）
WORKER_NOCLOSE_S=1800      # 票面c③：worker 退出无收口链>30min（钦定）
DONE_FLIP_GRACE_S=1800     # b②宽限 30min——dm_ledger 每拍补登，6 拍未翻=真滞留
DONE_CHAIN_WINDOW_S=604800 # b③扫窗 7d——dm_handoff SCAN_WINDOW 同式（首跑不洪史）
SPOT_N=5                   # 票面d：账本真值抽查面——最近完成票抽 5 张
ORIGIN_LAG_S=7200          # 票面e①：origin/master 落后龄>2h（钦定）
DEPLOY_BATCH_AGE_S=43200   # 票面e②：deploy 批龄>12h（钦定——deploy_window_due 批首件龄同源）
EMPTY_PULL_BEATS=2         # 票面c②：候选集非空拉起=0 升件拍数（「超2拍」阶梯下档）
FAIL_BEATS=3               # 票面c①：连续 spawn 败拍数（「连续3拍」阶梯上档）
ENDPOINT_DEPLOY_GRACE_S=300 # 票面f：「重启后仍红>5min」阈——部署行后 5min 内端点红=重启瞬断豁免
TERMINAL_PHASES="done rejected void" # 终相位集——b①非终判据（error=异常非终在列）

QC() { printf '%s' "${1:-}" | sed "s/'/''/g"; }
mkdir -p "$STATE_DIR" 2>/dev/null || true # tolerate: 态目录建败=容量面下拍重建（/var/tmp 世界写——建败罕见不炸扫）

# 异常计数器（digest 五面分项——面界=票面五片）
N_QUEUE=0; N_ENTITY=0; N_CAPACITY=0; N_LEDGER=0; N_EXTERNAL=0
report_find() { # $1=类目 $2=对象(稳定dedup键) $3=ticket字段 $4=详情
  case "$1" in dispatched-无拉|pulled-无unit|review-老化|approval-老化|draft-滞留) N_QUEUE=$((N_QUEUE+1));;
    幽灵票件|done-翻漏|done-无收口链) N_ENTITY=$((N_ENTITY+1));;
    spawn-连败|拉起-空转|worker-无收口) N_CAPACITY=$((N_CAPACITY+1));;
    账本-commit不可解|账本-回填缺席) N_LEDGER=$((N_LEDGER+1));;
    origin-落后|deploy-批龄|端点失健) N_EXTERNAL=$((N_EXTERNAL+1));; esac
  "$OPS_ESC_SH" MECH "$3" "" "" "层扫定性$1:$2" "$4" || true # tolerate: 升件末环败静默退（结构性限度在案——下拍重报自愈）
}

# ── 相位投影全图（五片共用单源——ops_ticket_phase fin_0153 视图；
#    同刻并列相位分歧票标 __tie__ 相位条件检全跳不误报）─────────────
PHASE_MAP=$(psql "$FIN_DSN" -qAt -F $'\x1f' -c "/*ls:phasemap*/
SELECT ticket,phase,latest_event,extract(epoch from now()-latest_ts)::bigint
FROM ops_ticket_phase" 2>/dev/null) || { echo "[layer-scan] 相位投影查询败——扫描面不可判" >&2; exit 1; }
declare -A PHASE_OF=() AGE_OF=()
N_TIE=0
while IFS=$'\x1f' read -r t ph ev age; do
  [ -n "$t" ] || continue
  phn=${ph:-null} # 空相位=审计中族（0153 词表外事件）——null 桶不入相位判据
  if [ -n "${PHASE_OF[$t]+x}" ] && [ "${PHASE_OF[$t]}" != "$phn" ]; then
    PHASE_OF[$t]="__tie__"; AGE_OF[$t]=0; N_TIE=$((N_TIE+1)); continue
  fi
  [ "${PHASE_OF[$t]:-}" = "__tie__" ] && continue
  PHASE_OF[$t]=$phn; AGE_OF[$t]=${age:-0}
done <<< "$PHASE_MAP"

# 相位直方图（digest 计数面——__tie__ 单列不入相位桶）
HIST=""; for p in draft approval_pending approved dispatched pulled working external review verdicted done rejected suspended void error null __tie__; do
  c=0; for t in "${!PHASE_OF[@]}"; do [ "${PHASE_OF[$t]}" = "$p" ] && c=$((c+1)); done
  HIST="$HIST $p=$c"
done

# ── 在飞 unit 集（systemctl --user 一次取——a/c 面共用；败=不可判降级）─
UNIT_OK=1
UNIT_LIST=$("$SYSTEMCTL" --user list-units 'o-worker-*' --state=active,activating \
  --no-legend --plain 2>/dev/null) || UNIT_OK=0
UNIT_TIDS=$(printf '%s\n' "$UNIT_LIST" | awk '{print $1}' \
  | sed -nE 's/^o-worker-(T-[0-9]{4})\.service$/\1/p')
unit_alive() { printf '%s\n' "$UNIT_TIDS" | grep -qxF "$1"; }

manual_lane() { # rc0=人工径票（manual_lane_check 同判据——rc2 缺件按可派不滤）
  "$MANUAL_LANE_SH" "$1" "$TASK_DIR" >/dev/null 2>&1
}

# ── a 队列一致 + b①幽灵行：相位条件检（unit/人工径/机器证滤在壳侧）──
# 机器拉起证集（pulled/working 无 unit 判据前置——拉起事件∪META-Dev
# 心跳行；交互/人工径领单无 unit 合法不入此集）。
MACHEV_TIDS=""
NEED_MACHEV=""
for t in "${!PHASE_OF[@]}"; do
  ph=${PHASE_OF[$t]}
  [ "$ph" = "__tie__" ] && continue
  case "$t" in T-[0-9][0-9][0-9][0-9]) ;; *) continue ;; esac # 伪键/DIAG- 族无 spawn 语义
  case "$ph" in pulled|working)
    [ "${AGE_OF[$t]:-0}" -gt "$PULLED_UNIT_STALE_S" ] && NEED_MACHEV="$NEED_MACHEV $t" ;;
  esac
done
if [ -n "${NEED_MACHEV# }" ]; then
  INLIST=$(printf '%s\n' $NEED_MACHEV | sort -u | sed "s/.*/'&'/" | paste -sd, -)
  MACHEV_TIDS=$(psql "$FIN_DSN" -qAt -c "/*ls:machev*/
SELECT ticket FROM ops_event WHERE event='拉起' AND ticket IN ($INLIST)
UNION
SELECT substring(ticket from 10) FROM ops_headless
WHERE ticket LIKE 'META-Dev-T-%' AND substring(ticket from 10) IN ($INLIST)" 2>/dev/null) || MACHEV_TIDS="" # tolerate: 机器证查询败=空集——pulled/working 无 unit 检全跳不误报（fail-closed 向静默）
fi
machev() { printf '%s\n' "$MACHEV_TIDS" | grep -qxF "$1"; }

for t in "${!PHASE_OF[@]}"; do
  ph=${PHASE_OF[$t]}; age=${AGE_OF[$t]:-0}
  [ "$ph" = "__tie__" ] && continue
  case "$t" in T-[0-9][0-9][0-9][0-9]) TID_OK=1;; *) TID_OK=0;; esac
  # a① dispatched-无拉：相位已含「派单后零拉起」；unit 在飞=spawn 实了跳过；
  # 人工径票机器永不可为不算断链（depgate/mech 补拉同滤）。
  if [ "$ph" = "dispatched" ] && [ "$age" -gt "$DISPATCH_STALE_S" ] && [ "$UNIT_OK" = 1 ]; then
    if ! unit_alive "$t" && ! manual_lane "$t"; then
      report_find "dispatched-无拉" "$t" "$t" "相位dispatched 龄$((age/60))m>阈$((DISPATCH_STALE_S/60))m 且无在飞o-worker unit"
    fi
  fi
  # a② pulled-无unit：机器拉起证在案（拉起行∪META-Dev 心跳）才判——
  # 交互/人工径领单无 unit 合法（领单 seat=施工席 双源不可分）。
  if [ "$ph" = "pulled" ] && [ "$age" -gt "$PULLED_UNIT_STALE_S" ] && [ "$UNIT_OK" = 1 ] \
     && machev "$t" && ! unit_alive "$t" && ! manual_lane "$t"; then
    report_find "pulled-无unit" "$t" "$t" "相位pulled 龄$((age/60))m 且机器拉起证在案而无存活o-worker unit"
  fi
  # c③ worker-无收口：机器拉起证在案∧unit 已退∧相位滞留 pulled/working
  # >30min=工人退出未落收口链（与 a② 同源不同面——队列一致 vs 容量水位；
  # 人工径同滤——机器生命周期不变量不罩人工票）。
  if { [ "$ph" = "pulled" ] || [ "$ph" = "working" ]; } && [ "$age" -gt "$WORKER_NOCLOSE_S" ] \
     && [ "$UNIT_OK" = 1 ] && machev "$t" && ! unit_alive "$t" && ! manual_lane "$t"; then
    report_find "worker-无收口" "$t" "$t" "相位$ph 龄$((age/60))m>阈$((WORKER_NOCLOSE_S/60))m 机器拉起证在案 unit 已退且无收口事件链"
  fi
  # a③ review/approval 老化（钦定初值 2h——呈批=approval_pending 相位）
  if [ "$ph" = "review" ] && [ "$age" -gt "$REVIEW_STALE_S" ]; then
    report_find "review-老化" "$t" "$t" "相位review 龄$((age/3600))h>阈$((REVIEW_STALE_S/3600))h 待裁积压"
  fi
  if [ "$ph" = "approval_pending" ] && [ "$age" -gt "$REVIEW_STALE_S" ]; then
    report_find "approval-老化" "$t" "$t" "相位approval_pending 龄$((age/3600))h>阈$((REVIEW_STALE_S/3600))h 呈批积压"
  fi
  # a④ draft 滞留（钦定 1h——物料化/批准链未动）
  if [ "$ph" = "draft" ] && [ "$age" -gt "$DRAFT_STALE_S" ]; then
    report_find "draft-滞留" "$t" "$t" "相位draft 龄$((age/3600))h>阈$((DRAFT_STALE_S/3600))h 草案未物料化/未批"
  fi
  # b① 幽灵行：非终相位 T-票而票件两处皆无（done/ 件在=非幽灵归 b②）
  if [ "$TID_OK" = 1 ] && [ "$age" -gt 0 ]; then
    case " $TERMINAL_PHASES " in *" $ph "*) ;; *)
      if ! compgen -G "$TASK_DIR/$t-*.md" >/dev/null && ! compgen -G "$DONE_DIR/$t-*.md" >/dev/null; then
        report_find "幽灵票件" "$t" "$t" "相位$ph 在案而 docs/tasks[/done] 票件两处皆无"
      fi ;;
    esac
  fi
done

# ── b② done/ 翻漏：done 件在∧相位非 done∧件龄>宽限（相位缺席=无生命
#    周期事件史——T9999 类销号件出相位域不判）────────────────────────
for f in "$DONE_DIR"/T-*.md; do
  [ -f "$f" ] || continue
  t=$(basename "$f" | sed -nE 's/^(T-[0-9]{4})-.*/\1/p')
  [ -n "$t" ] || continue
  ph=${PHASE_OF[$t]:-__none__}
  case "$ph" in done|__tie__|__none__) continue ;; esac # 终相位/并列分歧/无周期史皆不判
  mt=$(stat -c %Y "$f" 2>/dev/null || echo 0) # tolerate: stat 败=0 龄→宽限内不报
  now_s=$(date +%s)
  [ $((now_s - mt)) -gt "$DONE_FLIP_GRACE_S" ] || continue
  report_find "done-翻漏" "$t" "$t" "票件已入 done/ 而相位=$ph（件龄$(( (now_s-mt)/60 ))m>宽限$((DONE_FLIP_GRACE_S/60))m）"
done

# ── b③ done 无收口链：phase=done∧7d 窗内∧无收口族事件（审计中同族——
#    fin_0147 侧态收口链一员）───────────────────────────────────────
DONECHAIN=$(psql "$FIN_DSN" -qAt -c "/*ls:donechain*/
SELECT p.ticket FROM ops_ticket_phase p
WHERE p.phase='done' AND p.latest_ts > now()-interval '$((DONE_CHAIN_WINDOW_S/86400)) days'
  AND NOT EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=p.ticket
    AND e.event IN ('收口','移交','收口待审','审计中'))" 2>/dev/null) || DONECHAIN="" # tolerate: 查询败=空集跳过本面（相位图主面仍在——digest 不载本面细账）
while IFS= read -r t; do
  [ -n "$t" ] || continue
  case "$t" in *[!A-Za-z0-9_-]*) continue ;; esac
  report_find "done-无收口链" "$t" "$t" "相位done 在 7d 窗内而票史无收口/移交/收口待审/审计中事件"
done <<< "$DONECHAIN"

# ── d 账本真值（抽查非全量）：最近 SPOT_N 张 done 相位票双查 ────────
DONE5=$(psql "$FIN_DSN" -qAt -c "/*ls:done5*/
SELECT ticket FROM ops_ticket_phase WHERE phase='done'
ORDER BY latest_ts DESC, ticket LIMIT $SPOT_N" 2>/dev/null) || DONE5="" # tolerate: 抽查面查询败=跳过（主扫面不受累）
LEDGER_DEGRADED=0
if git -C "$REPO" rev-parse --verify --quiet master >/dev/null 2>&1; then
  while IFS= read -r t; do
    [ -n "$t" ] || continue
    case "$t" in T-[0-9][0-9][0-9][0-9]) ;; *) continue ;; esac
    # ①收口 commit 在 master 可解（merge/支件同祖先面——--grep 字面票号）
    CMT=$(git -C "$REPO" log --format=%H -1 --grep="$t" master 2>/dev/null) # tolerate: git 败/无命中=空→不可解判（REPO 实仓常态非空）
    if [ -z "$CMT" ]; then
      report_find "账本-commit不可解" "$t" "$t" "最近done票 master 上无含票号 commit 可解（收口落账存疑）"
      continue
    fi
    # ②票面回填存在：文件在（tasks 或 done）且状态行含 完成|待验
    fbfile=$(compgen -G "$TASK_DIR/$t-*.md" | head -1)
    [ -n "$fbfile" ] || fbfile=$(compgen -G "$DONE_DIR/$t-*.md" | head -1)
    if [ -n "$fbfile" ] && grep -qE '状态：.*(完成|待验)' "$fbfile"; then
      continue # 回填在案
    fi
    report_find "账本-回填缺席" "$t" "$t" "票件=${fbfile:-缺席} 状态行无 完成|待验 回填"
  done <<< "$DONE5"
else
  LEDGER_DEGRADED=1 # master 不可解=账本面整面不可判（digest 载——不逐票误报）
fi

# ── e 外事聚合（已有探针不重复——origin 落后=本窗自查 git 只读；
#    deploy 批龄=deploy_window_due 同源探测复用压升件副作用）─────────
# e① origin/master 落后：本地 master 领先远端的未推件最老龄>2h——
#    deploy 窗 HEAD==origin/master 收敛前提被推径卡死的聚合信号。
git -C "$REPO" fetch origin master --quiet 2>/dev/null || true # tolerate: fetch 败=按本地 origin/master 判（deploy_window_due 同式滑行）
if git -C "$REPO" rev-parse --verify --quiet origin/master >/dev/null 2>&1 \
   && git -C "$REPO" rev-parse --verify --quiet master >/dev/null 2>&1; then
  N_UNPUSHED=$(git -C "$REPO" rev-list --count origin/master..master 2>/dev/null || echo 0) # tolerate: rev-list 败=0 不报
  if [ "${N_UNPUSHED:-0}" -gt 0 ]; then
    OLDEST_CT=$(git -C "$REPO" log --format=%ct --reverse origin/master..master 2>/dev/null | head -1) # tolerate: 空=龄不可判不报
    case "$OLDEST_CT" in ''|*[!0-9]*) OLDEST_CT=0;; esac
    now_s=$(date +%s)
    if [ "$OLDEST_CT" -gt 0 ] && [ $((now_s - OLDEST_CT)) -gt "$ORIGIN_LAG_S" ]; then
      report_find "origin-落后" "origin/master" "" "本地master领先origin/master ${N_UNPUSHED}件 最老未推龄$(( (now_s-OLDEST_CT)/3600 ))h>阈$((ORIGIN_LAG_S/3600))h——推径疑滞"
    fi
  fi
fi
# e② deploy 批龄：deploy_window_due 同源探测（只读——OPS_ESC_SH=/bin/true
# 压其升件副作用；批首件龄>12h=窗内批积超阈的聚合判断，24h OVERRUN
# 升件径仍归该探针本体不重复）。
if [ -f "$DEPLOY_SH" ]; then
  DOUT=$(timeout 90 env OPS_ESC_SH=/bin/true DM_ENV_FILE="$ENVF" O_DEPLOY_WINDOW_REPO="$REPO" \
    bash "$DEPLOY_SH" 2>/dev/null) || DOUT="" # tolerate: 探测超时/败=本面跳过（deploy 窗主径每拍自跑——聚合面缺席不误报）
  DEP_REASON=$(printf '%s\n' "$DOUT" | grep '批非空 commits=' | tail -1)
  case "$DEP_REASON" in
    *"批首件龄="*)
      DEP_AGE_S=$(printf '%s' "$DEP_REASON" | sed -n 's/.*批首件龄=\([0-9][0-9]*\)s.*/\1/p')
      if [ -n "$DEP_AGE_S" ] && [ "$DEP_AGE_S" -gt "$DEPLOY_BATCH_AGE_S" ]; then
        report_find "deploy-批龄" "deploy面" "" "部署批积龄${DEP_AGE_S}s>阈${DEPLOY_BATCH_AGE_S}s（deploy_window_due 同源——$(printf '%s' "$DEP_REASON" | grep -oE 'commits=[0-9]+|tickets=[^ ]*' | paste -sd' ' -)）"
      fi ;;
  esac
fi

# ── f 用户端点健康（补面——2026-11-09 驾驶舱 502 实证业主令）：用户同
#    路径探 SPA 根+/o.dashboard.v1.CoreService/Ping 期望码 200 对拍；
#    部署窗豁免——最新部署行龄≤5min 内端点红=image 重启瞬断不报，
#    仍红>5min 或无部署行而红=升件「端点失健:dashboard」。清单初期
#    =dashboard 一处（票面钦定，扩面另立）。─────────────────────────
ENDPOINT_BASE=${O_LAYER_SCAN_ENDPOINT:-https://o.myfxlogs.org}
http_code() { curl -s -o /dev/null -w '%{http_code}' --max-time 10 "$@" 2>/dev/null || true; } # tolerate: curl 败=-w 已吐 000——rc 不用于判定
SPA_CODE=$(http_code "$ENDPOINT_BASE/")
PING_CODE=$(http_code -X POST -H 'Content-Type: application/json' -d '{}' \
  "$ENDPOINT_BASE/o.dashboard.v1.CoreService/Ping")
if [ "$SPA_CODE" != 200 ] || [ "$PING_CODE" != 200 ]; then
  DEP_TS=$(psql "$FIN_DSN" -qAt -c "/*ls:endpoint*/
SELECT extract(epoch from max(ts))::bigint FROM ops_event
WHERE event='部署' AND ts > now() - interval '24 hours'" 2>/dev/null) || DEP_TS="" # tolerate: 部署行查询败=空→按无部署行判（红仍报——fail-closed 向报不向隐）
  case "$DEP_TS" in ''|*[!0-9]*) DEP_TS=0;; esac
  now_s=$(date +%s)
  if [ "$DEP_TS" -eq 0 ] || [ $((now_s - DEP_TS)) -gt "$ENDPOINT_DEPLOY_GRACE_S" ]; then
    report_find "端点失健" "dashboard" "" "SPA=$SPA_CODE Ping=$PING_CODE（期望200）部署行龄=$([ "$DEP_TS" -eq 0 ] && echo 无 || echo $(( (now_s-DEP_TS)/60 ))m)——用户同路径红超豁免窗"
  fi
fi

# ── c 容量水位：候选集非空∧拍间拉起/领单=0 之 streak（候水标记票剔出
#    ——rc75 合法候队非败；streak 存 STATE_DIR 跨拍；首拍建基线不评）──
CAP_SKIP=""
CANDS=$("$DM_QUEUE_SH" dispatched 2>/dev/null)
if [ $? -ne 0 ]; then CAP_SKIP="dm_queue败"; CANDS=""; fi
STATE_FILE="$STATE_DIR/layer_scan.state"
LAST_SCAN=0; STREAK=0
if [ -f "$STATE_FILE" ]; then
  . "$STATE_FILE" 2>/dev/null || true # tolerate: 态件畸形/读败=变量缺席按基线（态件自写两枚数字变量——注入面限己写）
  LAST_SCAN=${last_scan:-0}; STREAK=${streak:-0} # 态件键=小写（写侧同形）
fi
case "$LAST_SCAN" in ''|*[!0-9]*) LAST_SCAN=0;; esac
case "$STREAK" in ''|*[!0-9]*) STREAK=0;; esac
NOW_S=$(date +%s)
if [ -n "$CAP_SKIP" ]; then
  : # 候选集不可判——原态件保留不重写（未知≠空集：last_scan 不前移，
    # 下拍拉起窗仍自上个有效拍起算不漏计）
else
  # 候水剔出：票内最新事件='候水'（rc75 合法候队——dispatch_water 钦定账态）
  CAND_TIDS=$(printf '%s\n' "$CANDS" | awk 'NF' | sort -u)
  EFF_CANDS=""
  if [ -n "$CAND_TIDS" ]; then
    INLIST=$(printf '%s\n' "$CAND_TIDS" | sed "s/.*/'&'/" | paste -sd, -)
    WATER_TIDS=$(psql "$FIN_DSN" -qAt -c "/*ls:water*/
SELECT ticket FROM (
  SELECT DISTINCT ON (ticket) ticket,event FROM ops_event
  WHERE ticket<>'' AND ticket IN ($INLIST)
  ORDER BY ticket,ts DESC,id DESC) l WHERE event='候水'" 2>/dev/null) || WATER_TIDS="" # tolerate: 候水查询败=空集不剔（保守全计候选——多报向不漏报）
    EFF_CANDS=$(printf '%s\n' "$CAND_TIDS" | grep -vxF -f <(printf '%s\n' "$WATER_TIDS") || true) # tolerate: grep 空集=全数有效合法径
  fi
  PULLS=0
  if [ "$LAST_SCAN" -gt 0 ] && [ "$LAST_SCAN" -lt "$NOW_S" ]; then
    PULLS=$(psql "$FIN_DSN" -qAt -c "/*ls:pulls*/
SELECT count(*) FROM ops_event WHERE event IN ('拉起','领单')
  AND ts > to_timestamp($LAST_SCAN)" 2>/dev/null) || PULLS="" # tolerate: 计数败=空→按 0 计（保守向报不向隐）
    PULLS=${PULLS:-0}
  fi
  WINDOW_OK=0
  if [ "$LAST_SCAN" -le 0 ]; then
    STREAK=0; WINDOW_OK=1 # 首拍建基线——无拍间窗可判不评（写态立基线）
  elif [ "$LAST_SCAN" -gt "$NOW_S" ]; then
    : # 时钟回拨——拍间窗不可判：streak 保留态件不前移（未知≠空集同律）
  else
    WINDOW_OK=1 # 有效拍间窗（同秒零宽窗=合法一拍——拉起窗内零事件即 0）
    if [ -n "$EFF_CANDS" ] && [ "$PULLS" -eq 0 ]; then STREAK=$((STREAK+1)); else STREAK=0; fi
  fi
  if [ "$WINDOW_OK" = 1 ]; then
    # 阶梯两档：2 拍=早期水位信号、3 拍=连败定性（票面两阈按同一 streak 落）
    if [ "$STREAK" -ge "$EMPTY_PULL_BEATS" ]; then
      report_find "拉起-空转" "候选集" "" "连续${STREAK}拍候选集非空（$(printf '%s\n' "$EFF_CANDS" | wc -l)票）而拍间拉起/领单=0"
    fi
    if [ "$STREAK" -ge "$FAIL_BEATS" ]; then
      report_find "spawn-连败" "候选集" "" "连续${STREAK}拍候选集非空而spawn面零拉起——派单落账无拉起实证（dispatch-spawn-silent-failure 同族面）"
    fi
    printf 'last_scan=%s\nstreak=%s\n' "$NOW_S" "$STREAK" > "$STATE_FILE" 2>/dev/null || true # tolerate: 态件写败=下拍按旧基线重建（streak 续计不错乱）
  fi
fi

# ── digest 行：每窗必落（静默≠缺席——全绿态零升件但 digest 照落）─────
DIGEST="层扫$(printf '%s' "$HIST" | tr -s ' ')｜异常 queue=$N_QUEUE entity=$N_ENTITY capacity=$N_CAPACITY ledger=$N_LEDGER external=$N_EXTERNAL｜unit=$([ "$UNIT_OK" = 1 ] && echo ok || echo unk) cap=$([ -n "$CAP_SKIP" ] && echo skip || echo ok) ledger_face=$([ "$LEDGER_DEGRADED" = 1 ] && echo unk || echo ok) tie=$N_TIE"
QC_DIGEST=$(QC "$DIGEST")
psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
  VALUES('MECH','层扫','层扫','$QC_DIGEST')" >/dev/null 2>&1 || true # tolerate: digest 落账尽力而为——写败下拍重评（行缺席不误报）
exit 0
