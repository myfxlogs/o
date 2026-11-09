#!/usr/bin/env bash
# whitelist_actions.sh —— T-0509 CEO自运感知·票C 白名单动作面执行器：
# 表内自决/表外呈批。mech_tick 窗消费（机械轨唯一执行者，同轨不双开）。
#
# 三层架构（判据全机械——查询结果非 LLM 读面）：
#   检测层=规则#1~5 五类触发器（对应票面表初版）——只读
#     ops_ticket_phase/ops_event/systemd 状态/票件存在性，零写；
#   判定层=whitelist_actions.tsv status 列——「生效」→执行动作，
#     待裁/停用/缺席→表外呈批行（event='呈批' seat='MECH' ticket=''
#     ——呈批是相位词，挂真票会翻相位，对象身份落 note 前缀引证；
#     前缀「表外动作呈批：规则#N:<对象>」24h 同键去重）；
#   执行层=动作本体——成功落引证行（seat=MECH 或钦定 seat，note=
#     「规则#N命中」——自决≠无账）；失败落升件行（ops_esc 前缀
#     「白名单动作败：规则#N:<对象>」）且同（规则,对象）升件在案且其后
#     无相位/处置族事件=等裁期零重试（附条件三败径升件不自愈——
#     lift 事件集=生命周期+裁决注记族，对象被他径处置过=重新具备
#     评估资格，再败再抑自限每裁决周期一次）。
#
# 规则#1 pulled尸重派：phase=pulled∧龄>MIN_AGE_S∧票件在∧键净∧非人工径
#   ∧o-worker-<tid> 非活（active/activating/deactivating 除外）→
#   dispatch_worker+拉起行（seat=CTO 同补拉窗钦定口径）；同票 24h 内
#   已有关#1 拉起引证行=防循环跳过（票面钦定界）。rc75=候水账态
#   （dispatch_water）非败不升件。
# 规则#2 error作废：phase=error∧（伪键∨无票件）双查可判→作废行
#   （seat=MECH）；^T-NNNN 真票有件/DIAG-* 诊断键=不可判→升件不猜
#   （「白名单不可判：规则#2:<tid>」前缀 24h 去重——非常态不动作
#   只上报，status 不闸此径）。
# 规则#3 醒票：phase=suspended∧非 rate-limit:auto∧await=绿窗档复跑
#   绿→续跑（判据同 dm_suspend——本窗在其后序跑，已续者相位已翻
#   pulled 本检结构性不重；「拉起」晚于「挂起」=已消解跳过）；
#   锚未至/不可机械判=零动作零呈批（>4h 升件归 dm_suspend 既有径）。
# 规则#4 席位kick：param units=unit:stall_s 表内席——unit 非活∧
#   InactiveEnterTimestampUSec 龄>stall_s（空/0=未跑过=满龄）∧队列
#   探针非空→systemctl --user start --no-block（同事件 kick 语义，
#   oneshot 单例闸现成）；cooldown_s 内已有关#4 自决行=跳过；
#   无队列探针的 unit=不可判跳过不猜。
# 规则#5 界内微调：param key=/domain=——候水 24h≥3∧现值=域上限→步降
#   域下限；候水 24h=0∧现值=域下限→回域上限；目标∉domain=呈批
#   （表列防御）；覆写档=$(o_worker_runtime_dir)/parallel-limit-
#   override 原子写（worker_runtime.sh 采信面——env>档>缺省回落）；
#   参数调整行（seat=MECH ticket=''）引证；24h 内已有关#5 参数调整
#   行=防抖跳过。
#
# 红线：资金扩投/新平台/席位立换永不入表（票面明文重申——本件不
#   识别也不执行任何表外动作，只呈批）。
# env 覆写（桩测面）：O_WL_TABLE/O_WL_TASK_DIR/O_WL_REPO_DIR/
#   O_WL_DISPATCH/O_WL_DISPATCH_WATER/O_WL_DM_QUEUE/O_WL_CEO_QUEUE/
#   O_WL_OPS_ESC/O_WL_KEY_GATE/O_WL_SYSTEMCTL/O_WL_GO/O_WL_GO_TIMEOUT/
#   O_WL_MANUAL_LANE/O_WL_MIN_AGE_S/O_WORKER_RUNTIME_LIB。
set -u
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$FIN_DSN" ] || exit 0 # 同拍壳律——无 DSN 静默退（mech_tick 已按 FIN_DSN 门控，本件双保险）

TABLE=${O_WL_TABLE:-$SCRIPT_DIR/whitelist_actions.tsv}
TASK_DIR=${O_WL_TASK_DIR:-${O_WORKER_TASK_DIR:-/opt/o/docs/tasks}}
REPO_DIR=${O_WL_REPO_DIR:-/opt/o}
DISPATCH_WORKER=${O_WL_DISPATCH:-/opt/o/scripts/meta/dispatch_worker.sh}
DISPATCH_WATER_SH=${O_WL_DISPATCH_WATER:-$SCRIPT_DIR/dispatch_water.sh}
DM_QUEUE_SH=${O_WL_DM_QUEUE:-$SCRIPT_DIR/dm_queue.sh}
CEO_QUEUE_SH=${O_WL_CEO_QUEUE:-$SCRIPT_DIR/ceo_queue.sh}
OPS_ESC_SH=${O_WL_OPS_ESC:-$SCRIPT_DIR/ops_esc.sh}
KEY_GATE=${O_WL_KEY_GATE:-$SCRIPT_DIR/ops_event_key_gate.sh}
MANUAL_LANE_SH=${O_WL_MANUAL_LANE:-$SCRIPT_DIR/manual_lane_check.sh}
SYSTEMCTL=${O_WL_SYSTEMCTL:-systemctl}
GO_BIN=${O_WL_GO:-go}
GO_TIMEOUT=${O_WL_GO_TIMEOUT:-900}
MIN_AGE_S=${O_WL_MIN_AGE_S:-600}
[[ "$MIN_AGE_S" =~ ^[0-9]+$ ]] || MIN_AGE_S=600
[[ "$GO_TIMEOUT" =~ ^[0-9]+$ ]] || GO_TIMEOUT=900
[[ -x "$DISPATCH_WORKER" ]] || DISPATCH_WORKER=/opt/o/scripts/meta/dispatch_worker.sh
case ":$PATH:" in *:/usr/local/go/bin:*) ;; *) PATH="$PATH:/usr/local/go/bin" ;; esac # 同 mech_tick——systemd PATH 无 go

[ -r "$TABLE" ] || { echo "[wl] 白名单表缺/不可读 $TABLE" >&2; exit 1; }

# TSV → per-rule status/param/bound（列：rule_id action param bound status note）
declare -A WL_STATUS=() WL_PARAM=() WL_BOUND=()
while IFS=$'\t' read -r rid act param bound status note; do
	case "${rid:-}" in ''|'#'*) continue ;; esac
	WL_STATUS[$rid]=$status; WL_PARAM[$rid]=$param; WL_BOUND[$rid]=$bound
done < "$TABLE"

QC(){ printf '%s' "${1:-}" | sed "s/'/''/g"; }
q(){ psql "$FIN_DSN" -qAt -c "$1" 2>/dev/null; }

# 升件（执行失败/不可判）——ops_esc 共享径（prefix=去重键兼 note 头，
# 24h 同键去重在件内）；seat=CTO 同 mech_esc/dm_suspend 机械升件钦定
# 口径（MECH 非 ceo_queue 席位集——升件落 CTO 壳轨上行径）；ticket=
# 对象（真票=TID；席位/键级=伪键单 token——键卫生闸放行，非相位词
# 不产生相位行）。
act_fail() { # $1=rid $2=obj $3=detail
	"$OPS_ESC_SH" CTO "$2" "" "" "白名单动作败：规则#$1:$2" "$3" || true # tolerate: 升件尽力而为——写败下拍重判（结构性限度在案）
}
unjudgeable() { # $1=rid $2=obj $3=why
	"$OPS_ESC_SH" CTO "$2" "" "" "白名单不可判：规则#$1:$2" "$3——双查不可判不猜（T-0509 表内界）" || true # tolerate: 升件尽力而为——写败下拍重判
}

# 呈批（表外动作触发）——event='呈批' seat='MECH' ticket=''（票外件不入
# 相位投影——ticket<>'' 视图过滤天然隔离；对象身份落 note 前缀引证，
# 24h 同键去重；ceo_queue 以「呈批#<ops_event.id>」合成行入 CEO 队
# ——议题#id 同构，终态事件 note 引证「呈批#id」出队，席域钉 MECH）。
baoming() { # $1=rid $2=obj $3=desc
	local rid=$1 obj=$2 desc=$3 prefix note dup
	prefix="表外动作呈批：规则#${rid}:${obj}"
	dup=$(q "SELECT 1 FROM ops_event WHERE event='呈批' AND note LIKE '$(QC "$prefix")%' AND ts>now()-interval '24 hours' LIMIT 1") || return 0 # 查询败=本拍不呈（fail-closed——宁可下拍重判不冒重复呈批）
	[ "$dup" = "1" ] && return 0
	note="$prefix——$desc（白名单表外触发呈裁，T-0509 附条件一）"
	q "INSERT INTO ops_event(seat,event,ticket,note) VALUES('MECH','呈批','','$(QC "$note")')" >/dev/null || true # tolerate: 呈批落账尽力而为——写败下拍重判
}

# 失败抑制闸：同（规则,对象）升件在案且其后无 lift 事件=等裁期零重试。
# lift 事件集=生命周期+裁决注记族——对象被他径处置过=重新具备评估资格。
# 查询败=按抑制处理（fail-closed——附条件三零重试不因读败放行）。
LIFT_EVENTS="'处置','消','裁决','领单','拉起','作废','已批','派单','收口','移交','收口待审','完成','驳回','挂起','开工','外派','审计中','呈批','起草','归流','先例适用','管理令','业主令','业主批','已立','已派','并票','转办','返修','启用','议题','等锁','部署','部署中','待部署','GC','开仓挂起','开仓解冻','冲销迹','裸袋','饥饿','饥饿响应','面清单新面','依赖对清单','依赖对失明','覆盖失明','族沉默','feed断更','新品','异常词','漂移','漂移消','教训矿工','已并码','earn_auto_purchase','quote_watchlist_set','price_alert_build','price_alert_state','自决','参数调整'"
suppressed() { # $1=rid $2=obj → rc0=抑制中（含查询败按抑制）
	local rid=$1 obj=$2 prefix v
	prefix="白名单动作败：规则#${rid}:${obj}"
	v=$(q "SELECT 1 FROM ops_event f WHERE f.event='升件' AND f.note LIKE '$(QC "$prefix")%'
	  AND f.ts > COALESCE((SELECT max(l.ts) FROM ops_event l WHERE l.ticket='$(QC "$obj")'
	    AND l.event IN ($LIFT_EVENTS)),'-infinity'::timestamptz) LIMIT 1") || return 0
	[ "$v" = "1" ]
}

# 24h 防循环闸：同票 24h 内已有关#N <event> 引证行=跳过（规则#1 钦定界
# 同式复用——note 前缀「规则#N命中」为引证锚）。查询败=按已动过处理
# 本拍不动作（fail-closed——防循环界不因读败放行）。
acted_recent() { # $1=rid $2=ticket $3=event词
	local v
	v=$(q "SELECT 1 FROM ops_event WHERE ticket='$(QC "$2")' AND event='$3' AND note LIKE '规则#$1命中%' AND ts>now()-interval '24 hours' LIMIT 1") || return 0
	[ "$v" = "1" ]
}

model_for() { # $1=tid → 派单备注模型档恢复（mech_tick/dm_suspend 同式）
	local m
	m=$(q "SELECT CASE WHEN note ILIKE '%swe-2-flash-high%' THEN 'swe-2-flash-high' WHEN note ILIKE '%swe-2-medium%' THEN 'swe-2-medium' ELSE 'swe-2-flash-high' END FROM ops_event WHERE ticket='$(QC "$1")' AND event='派单' ORDER BY ts DESC,id DESC LIMIT 1")
	printf '%s' "${m:-swe-2-flash-high}"
}

is_active() { [ "${WL_STATUS[$1]:-}" = "生效" ]; }

# ── 规则#1：pulled 尸重派 ──
R1ROWS=$(q "SELECT ticket, extract(epoch from now()-latest_ts)::bigint FROM ops_ticket_phase WHERE phase='pulled' AND latest_ts < now() - interval '${MIN_AGE_S} seconds' ORDER BY latest_ts") || { echo "[wl] 规则#1扫描败" >&2; exit 1; }
printf '%s\n' "$R1ROWS" | while IFS=$'\t' read -r tid age; do
	[ -n "$tid" ] || continue
	[[ "$tid" =~ ^T-[0-9]{4}$ ]] || continue
	compgen -G "$TASK_DIR/$tid-*.md" >/dev/null || continue
	if [ -x "$MANUAL_LANE_SH" ]; then
		"$MANUAL_LANE_SH" "$tid" "$TASK_DIR"; mlc=$?
		[ "$mlc" -eq 1 ] || continue # 0=人工径滤除/2票件缺/3用法错=不可判一律跳过不猜
	fi
	ust=$($SYSTEMCTL --user is-active "o-worker-$tid.service" 2>/dev/null)
	case "$ust" in active|activating|deactivating) continue ;; esac
	acted_recent 1 "$tid" "拉起" && continue
	suppressed 1 "$tid" && continue
	if is_active 1; then
		[ ! -x "$KEY_GATE" ] || "$KEY_GATE" "$tid" "event=拉起" "MECH" || continue
		model=$(model_for "$tid")
		prompt="读 /opt/o/skills/施工席/SKILL.md 领单 $tid：只做工单范围内的事——写实现码/测试/提交/回填；范围外发现的问题不自己动手，写进交付备注；同票已有 Devin 会话或 worktree 时续跑不新开（白名单规则#1自决重派）"
		"$DISPATCH_WORKER" "$tid" "$model" "$prompt" >/dev/null 2>&1
		rc=$?
		case "$rc" in
			0) q "INSERT INTO ops_event(seat,event,ticket,note) VALUES('CTO','拉起','$(QC "$tid")','规则#1命中——pulled尸（无存活unit龄>${MIN_AGE_S}s）白名单自决重派（T-0509）')" >/dev/null || act_fail 1 "$tid" "拉起行落账败" ;;
			75) "$DISPATCH_WATER_SH" "$tid" || true ;; # tolerate: 候水标记尽力而为——下拍自然重落
			*)  act_fail 1 "$tid" "dispatch_worker rc=$rc" ;;
		esac
	else
		baoming 1 "$tid" "pulled尸（龄${age:-?}s无存活unit）"
	fi
done

# ── 规则#2：error 相位作废/不可判 ──
R2ROWS=$(q "SELECT ticket FROM ops_ticket_phase WHERE phase='error' ORDER BY latest_ts") || { echo "[wl] 规则#2扫描败" >&2; exit 1; }
printf '%s\n' "$R2ROWS" | while IFS= read -r tid; do
	[ -n "$tid" ] || continue
	vctx=; vjudge=
	if [[ "$tid" =~ ^T-[0-9]{4}$ ]]; then
		if compgen -G "$TASK_DIR/$tid-*.md" >/dev/null; then
			vjudge=unjudgeable; vctx="真票有件error相位——票件级判断非白名单域"
		else
			vjudge=voidable; vctx="无票件"
		fi
	elif [[ "$tid" =~ ^DIAG-[0-9]+$ ]]; then
		vjudge=unjudgeable; vctx="DIAG伪键error——诊断域不可判"
	else
		vjudge=voidable; vctx="伪键"
	fi
	if [ "$vjudge" = unjudgeable ]; then
		unjudgeable 2 "$tid" "$vctx"
		continue
	fi
	suppressed 2 "$tid" && continue
	if is_active 2; then
		[ ! -x "$KEY_GATE" ] || "$KEY_GATE" "$tid" "event=作废" "MECH" || continue
		q "INSERT INTO ops_event(seat,event,ticket,note) VALUES('MECH','作废','$(QC "$tid")','规则#2命中——error相位作废归档（${vctx}，白名单T-0509自决）')" >/dev/null || act_fail 2 "$tid" "作废行落账败"
	else
		baoming 2 "$tid" "error相位${vctx}——规则#2未生效"
	fi
done

# ── 规则#3：suspended 醒票（await=绿窗锚消解）──
R3ROWS=$(q "SELECT ticket, latest_note, latest_seat FROM ops_ticket_phase WHERE phase='suspended' ORDER BY latest_ts") || { echo "[wl] 规则#3扫描败" >&2; exit 1; }
printf '%s\n' "$R3ROWS" | while IFS=$'\t' read -r tid note seat; do
	[ -n "$tid" ] || continue
	[[ "$tid" =~ ^T-[0-9]{4}$ ]] || continue
	[ "$seat" = "施工席" ] && case "$note" in rate-limit:auto*) continue ;; esac
	spec=; case "$note" in *await=*) spec=${note#*await=};; esac
	spec=${spec%%[[:space:]]*}; spec=${spec//：/:}
	gate=${spec%%:*}; param=; [ "$spec" != "$gate" ] && param=${spec#*:}
	case "$gate" in 绿窗|green|green-window|green_window) ;; *) continue ;; esac
	[ -n "$param" ] || continue
	pkg= arg=; case "$param" in *:*) pkg=${param%%:*}; arg=${param#*:} ;; *) arg=$param ;; esac
	[ -n "$arg" ] || continue
	gargs=(-count=1 -run "^${arg}$"); [ -n "$pkg" ] && gargs+=("$pkg") || gargs+=(./...)
	( cd "$REPO_DIR" && timeout "$GO_TIMEOUT" "$GO_BIN" test "${gargs[@]}" ) >/dev/null 2>&1 || continue
	already=$(q "SELECT 1 FROM ops_event WHERE ticket='$(QC "$tid")' AND event='拉起'
	  AND ts>(SELECT max(ts) FROM ops_event WHERE ticket='$(QC "$tid")' AND event='挂起') LIMIT 1") || continue # 查询败=不可判本拍不续跑
	[ "$already" = "1" ] && continue
	acted_recent 3 "$tid" "拉起" && continue # 24h 防自环——同票已关#3 续跑引证行在案跳过
	suppressed 3 "$tid" && continue
	if is_active 3; then
		[ ! -x "$KEY_GATE" ] || "$KEY_GATE" "$tid" "event=拉起" "MECH" || continue
		model=$(model_for "$tid")
		prompt="读 /opt/o/skills/施工席/SKILL.md 施工协议领单/续领 $tid；按票面施工，同票已有 Devin 会话/worktree 时续跑；本次=白名单规则#3自决续跑（await=${spec} 锚已消解）"
		"$DISPATCH_WORKER" "$tid" "$model" "$prompt" >/dev/null 2>&1
		rc=$?
		case "$rc" in
			0) q "INSERT INTO ops_event(seat,event,ticket,note) VALUES('CTO','拉起','$(QC "$tid")','规则#3命中——suspended await=${spec} 锚消解白名单自决续跑（T-0509）')" >/dev/null || act_fail 3 "$tid" "拉起行落账败" ;;
			75) "$DISPATCH_WATER_SH" "$tid" || true ;; # tolerate: 候水标记尽力而为——下拍自然重落
			*)  act_fail 3 "$tid" "dispatch_worker rc=$rc" ;;
		esac
	else
		baoming 3 "$tid" "suspended await=${spec} 锚已消解"
	fi
done

# ── 规则#4：oneshot 席 kick（stall+队列非空→start）──
r4_param_units() { # WL_PARAM[4] → "unit:stall_s:cooldown_s" 行
	local p=${WL_PARAM[4]:-} kv units cd spec u st
	cd=3600; units=
	local IFS=';'
	for kv in $p; do
		case "$kv" in
			units=*) units=${kv#units=} ;;
			cooldown_s=*) cd=${kv#cooldown_s=} ;;
		esac
	done
	[[ "$cd" =~ ^[0-9]+$ ]] || cd=3600
	local IFS=','
	for spec in $units; do
		u=${spec%%:*}; st=${spec#*:}
		[[ "$u" =~ ^[A-Za-z0-9@_.:-]+$ ]] || continue
		[[ "$st" =~ ^[0-9]+$ ]] || st=2400
		printf '%s\t%s\t%s\n' "$u" "$st" "$cd"
	done
}
queue_n() { # $1=unit → stdout=队列计数；rc2=无探针 rc1=取数败
	local out e eng
	case "$1" in
		*cto-seat*) out=$("$DM_QUEUE_SH" review 2>/dev/null) || return 1
			printf '%s\n' "$out" | awk 'NF{n++}END{print n+0}' ;;
		*ceo-seat*) out=$("$CEO_QUEUE_SH" 2>/dev/null) || return 1
			printf '%s\n' "$out" | awk 'NF{n++}END{print n+0}' ;;
		*econ-verdict*|*verdict*)
			e=$(q "SELECT count(*) FROM fin_econ_lesson WHERE status='pending'") || return 1
			eng=$(q "SELECT count(*) FROM fin_eng_lesson WHERE status='pending'"); eng=${eng:-0}
			[[ "$e" =~ ^[0-9]+$ && "$eng" =~ ^[0-9]+$ ]] || return 1
			printf '%s\n' $((e+eng)) ;;
		*) return 2 ;;
	esac
}
r4_param_units | while IFS=$'\t' read -r unit stall_s cooldown_s; do
	[ -n "$unit" ] || continue
	cdq=$(q "SELECT 1 FROM ops_event WHERE event='自决' AND ticket='$(QC "$unit")' AND note LIKE '规则#4命中%' AND ts>now()-interval '${cooldown_s} seconds' LIMIT 1") || continue # 查询败=不可判本拍不kick
	[ "$cdq" = "1" ] && continue
	suppressed 4 "$unit" && continue
	ust=$($SYSTEMCTL --user is-active "$unit" 2>/dev/null)
	case "$ust" in active|activating|deactivating) continue ;; esac
	uts=$($SYSTEMCTL --user show "$unit" -p InactiveEnterTimestampUSec --value 2>/dev/null); usrc=$?
	[ "$usrc" -eq 0 ] || continue # show 败=不可判跳过不猜
	uage=999999999
	if [[ "$uts" =~ ^[0-9]+$ ]] && [ "$uts" -gt 0 ]; then
		nowus=$(date +%s%6N 2>/dev/null); [[ "$nowus" =~ ^[0-9]+$ ]] || nowus=0
		uage=$(( (nowus - uts) / 1000000 ))
	fi
	[ "$uage" -gt "$stall_s" ] || continue
	qn=$(queue_n "$unit"); qrc=$?
	[ "$qrc" -eq 0 ] || continue # 无探针/取数败=不可判跳过
	[ "${qn:-0}" -gt 0 ] 2>/dev/null || continue
	if is_active 4; then
		"$SYSTEMCTL" --user start --no-block "$unit" >/dev/null 2>&1
		krc=$?
		if [ "$krc" -eq 0 ]; then
			q "INSERT INTO ops_event(seat,event,ticket,note) VALUES('MECH','自决','$(QC "$unit")','规则#4命中——席位kick $unit（stall龄${uage}s>界${stall_s}s+队列=$qn，白名单T-0509自决）')" >/dev/null || act_fail 4 "$unit" "自决行落账败"
		else
			act_fail 4 "$unit" "systemctl start --no-block rc=$krc"
		fi
	else
		baoming 4 "$unit" "席位stall（龄${uage}s>界${stall_s}s）+队列=$qn"
	fi
done

# ── 规则#5：界内微调（key=/domain=——候水驱动 step 调档）──
source "${O_WORKER_RUNTIME_LIB:-$SCRIPT_DIR/worker_runtime.sh}" 2>/dev/null || true # tolerate: 库件缺席=规则#5整体跳过（下方判空防）
if declare -F o_worker_parallel_limit >/dev/null && declare -F o_worker_runtime_dir >/dev/null; then
	R5P=${WL_PARAM[5]:-}; R5KEY=; R5DOM=
	IFS=';'
	for kv in $R5P; do
		case "$kv" in key=*) R5KEY=${kv#key=} ;; domain=*) R5DOM=${kv#domain=} ;; esac
	done
	IFS=' \t\n'
	if [ -n "$R5KEY" ] && [ -n "$R5DOM" ]; then
		R5CUR=$(o_worker_parallel_limit 2>/dev/null); r5rc=$?
		R5CNT=$(q "SELECT count(*) FROM ops_event WHERE event='候水' AND ts>now()-interval '24 hours'")
		# 域值全数值守卫——非数域=表列形非法不可判跳过不猜
		if [ "$r5rc" -eq 0 ] && [[ "$R5CNT" =~ ^[0-9]+$ ]] && [[ "$R5DOM" =~ ^[0-9]+(,[0-9]+)*$ ]]; then
			R5TGT=
			if [ "$R5CNT" -ge 3 ]; then
				# 拥塞→步降一档：域内 <cur 的最大值（cur≤域底=不可降零动作）
				R5TGT=$(printf '%s\n' ${R5DOM//,/ } | sort -n | awk -v c="$R5CUR" '$1<c{t=$1}END{print t}')
			elif [ "$R5CNT" -eq 0 ]; then
				# 空载→步升一档：域内 >cur 的最小值（cur≥域顶=不可升零动作）
				R5TGT=$(printf '%s\n' ${R5DOM//,/ } | sort -n | awk -v c="$R5CUR" '$1>c{print $1;exit}')
			fi
			if [ -n "$R5TGT" ]; then
				R5DEDUP=$(q "SELECT 1 FROM ops_event WHERE event='参数调整' AND note LIKE '规则#5命中%' AND ts>now()-interval '24 hours' LIMIT 1")
				suppressed 5 "$R5KEY" || [ "$R5DEDUP" = "1" ] || {
					if is_active 5; then
						# 目标∈domain 表列防御——域外=呈批不猜
						dom_hit=; for dv in ${R5DOM//,/ }; do [ "$dv" = "$R5TGT" ] && dom_hit=1; done
						if [ -z "$dom_hit" ]; then
							baoming 5 "$R5KEY" "目标值$R5TGT∉domain=$R5DOM——超界呈批"
						else
							R5DIR=$(o_worker_runtime_dir) || R5DIR=
							if [ -n "$R5DIR" ] && mkdir -p "$R5DIR" 2>/dev/null; then
								R5TMP=$(mktemp "$R5DIR/.parallel-limit.XXXXXX" 2>/dev/null) # tolerate: mktemp 败=空→下方判空走升件
								if [ -n "$R5TMP" ] && printf '%s\n' "$R5TGT" > "$R5TMP" && chmod 600 "$R5TMP" && mv -f "$R5TMP" "$R5DIR/parallel-limit-override" 2>/dev/null; then
									q "INSERT INTO ops_event(seat,event,ticket,note) VALUES('MECH','参数调整','','规则#5命中——$R5KEY $R5CUR→$R5TGT（界内域$R5DOM，候水24h=$R5CNT，白名单T-0509自决）')" >/dev/null || act_fail 5 "$R5KEY" "参数调整行落账败"
								else
									[ -n "$R5TMP" ] && rm -f "$R5TMP" 2>/dev/null # tolerate: 清理尽力而为
									act_fail 5 "$R5KEY" "覆写档写败"
								fi
							else
								act_fail 5 "$R5KEY" "runtime dir 不可用"
							fi
						fi
					else
						baoming 5 "$R5KEY" "界内微调触发（候水24h=$R5CNT 现值$R5CUR→$R5TGT）——规则#5未生效"
					fi
				}
			fi
		fi
	fi
fi
exit 0
