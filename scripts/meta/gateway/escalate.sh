#!/usr/bin/env bash
# gateway/escalate.sh —— o-gateway 升件账子件（T-0469）：未认领超龄升件
# 的机械路由——归口→CTO→CEO→业主四级阶梯，只路由不裁定性。
# 认领判据=escalation_unclaimed 量表同式（seat='gateway' 自路由行不计
# 领——机械派单≠席领，防自激）；阶梯推进=每级一次 '升件路由' 落账行
# （event 新词不入相位投影/不入升件本体——原升件行仍是唯一裁断对象）。
# 动作面全机械：席唤起=systemctl --user start --no-block 对应席单元
# （timer 同径 ExecStart——敲门非代裁）；业主级=COS上行行+feishu 尽力
# 推送。幂等键=升件#<id>+级——重评零重复路由；级冷却防连 ringing。
# 边界：升件定性/裁断永归归口席，本件只保证「没人接会叫到对的人」。
# 环境覆写：同 common.sh + O_GW_ESC_* 桩面。
set -u
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/common.sh"

UNCLAIMED_SEC=${O_GW_ESC_UNCLAIMED_SEC:-1800}   # 未认领龄阈（30min=量表同阈）
LEVEL_COOLDOWN=${O_GW_ESC_LEVEL_COOLDOWN:-7200} # 级间冷却（2h）
case "$UNCLAIMED_SEC" in ''|*[!0-9]*) UNCLAIMED_SEC=1800 ;; esac
case "$LEVEL_COOLDOWN" in ''|*[!0-9]*) LEVEL_COOLDOWN=7200 ;; esac
MAX_LEVEL=4                                   # 归口→CTO→CEO→业主
FEISHU_SH=${O_GW_FEISHU_SH:-/opt/meta/scripts/feishu-notify.sh}
DSN=$(gw_dsn); [ -n "$DSN" ] || exit 0

# ── ⓪ 分类归领前段：机械可证类自动销（T-0517 业主令「gateway 分类
#   处理派发席位，需 LLM 的报 CEO」——机械可证账面差本件自销零龄阈
#   直接销；裁断/LLM 判断类零动作让梯顺流到 CEO/业主级=派发语义）。
#   销件行 seat=处置席/CTO/COS——认领判据 seat<>'gateway' 防自激钉死
#   （完成/作废同票生命周期词归各行本体席成例;处置=名域中立销词）。
#   层扫类目签名=「层扫定性<类目>:<对象>——…」机读；其余签名让梯。
res_ins() { # $1=seat $2=event $3=ticket $4=note——销件行写径（gw_ledger seat 钉死 gateway 不可用）
	#   写败不吞——stderr 留样落 write_fail.log（KB heredoc-sql-unquoted-
	#   silent-swallow 教训②：哑写死径不可再静默；容错口径=不留证即虚写）
	local out
	out=$(psql "$DSN" -qAt 2>&1 <<SQL
INSERT INTO ops_event(seat,event,ticket,note)
VALUES('$(gw_qc "$1")','$(gw_qc "$2")','$(gw_qc "$3")','$(gw_qc "$4")');
SQL
	) || printf '%s res_ins写败 %s/%s/%s — %s\n' "$(date -u +%FT%TZ)" "$1" "$2" "$3" \
		"$out" >> "${GW_STATE_DIR:-/var/tmp/o-gateway}/write_fail.log" 2>/dev/null || true # tolerate: 失败上报径自身兜底——写日志再败不可炸归领循环,降级为静默(该行即失败证据面)
	return 0
}
phase_void() { # $1=ticket——无籍行/墓碑化翻相（ops_ticket_phase=视图,翻相唯一径=作废事件行→void;相位行=层扫签名源,翻void=签名永死）
	res_ins 'COS' '作废' "$1" "无籍行墓碑化——相位行翻void(归领处置签名源断)"
}

UNCLAIMED_SQL="SELECT e.id, COALESCE(e.ticket,''),
    COALESCE(replace(substring(e.note from 1 for 120),'|','/'),'')
  FROM ops_event e WHERE e.event='升件' AND e.note NOT LIKE '升件老化%'
    AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts AND x.seat<>'gateway' AND
      ((e.ticket<>'' AND x.ticket=e.ticket AND x.event IN ('裁决','完成','驳回','挂起'))
       OR x.note ~ ('升件#' || e.id::text || '([^0-9]|$)')))
  ORDER BY e.id"
mapfile -t CLS_ROWS < <(gw_sql "$UNCLAIMED_SQL")
resolved=0
for row in "${CLS_ROWS[@]}"; do
	IFS='|' read -r eid etid enote <<< "$row"
	[ -n "$eid" ] || continue
	case "$enote" in
		*层扫定性done-翻漏:*)
			# 票件 done/ 实证=账面差可机证——'完成'同票行=相位补翻+销件同拍
			if [ -n "$etid" ] && compgen -G "$GW_REPO/docs/tasks/done/${etid}-*.md" >/dev/null; then
				res_ins 'CTO' '完成' "$etid" "层扫归位——done-翻漏机械补翻（票件done/在案实证;销升件#$eid；dm_ledger补登同语义）"
				resolved=$((resolved+1))
			fi ;;
		*层扫定性幽灵票件:*)
			# T-NNNN 相位在案而票件双无=真幽灵——作废归 void（T-9001式）+销件
			case "$etid" in
				T-[0-9][0-9][0-9][0-9]|T-[0-9][0-9][0-9][0-9]R)
					if ! compgen -G "$GW_REPO/docs/tasks/${etid}-*.md" >/dev/null \
					  && ! compgen -G "$GW_REPO/docs/tasks/done/${etid}-*.md" >/dev/null; then
						res_ins 'COS' '作废' "$etid" "层扫幽灵票件作废——相位行在而票件双无实证（T-9001式;销升件#$eid）"
						res_ins '处置席' '处置' '' "升件#$eid 销——$etid 幽灵作废在案"
						resolved=$((resolved+1))
					fi ;;
			esac ;;
		*层扫定性无籍行:*)
			case "$etid" in
				T-9[0-9][0-9][0-9]|SELFTEST-*)
					phase_void "$etid"
					res_ins '处置席' '处置' '' "升件#$eid 销——无籍行测试/夹具名域（$etid；相位翻void墓碑化签名永死）"
					resolved=$((resolved+1)) ;;
				T-[0-9][0-9][0-9][0-9]|T-[0-9][0-9][0-9][0-9]R)
					# 终相无件=历史债记档——翻void墓碑化签名永死（done+无件
					# 亦翻:无件之 done=墓碑非实证成件,churn 环断此）
					phase_void "$etid"
					res_ins '处置席' '处置' '' "升件#$eid 销——$etid 无籍行（终相/伪号票件双无实证记档+相位翻void墓碑化）"
					resolved=$((resolved+1)) ;;
				'') : ;; # 无 ticket 的无籍行=系统件——让梯归席
				*)  # 非T伪键=工作流/提案/名域标签（GW-001/DBP*/提案-*等）——翻void墓碑化（伪键相位行=签名源）
					phase_void "$etid"
					res_ins '处置席' '处置' '' "升件#$eid 销——无籍行非票名域($etid;相位翻void墓碑化签名永死)"
					resolved=$((resolved+1)) ;;
			esac ;;
		*层扫定性done-无收口链:*)
			# 终相票史链缺=历史债不可伪造——记档即销（补链造史禁）
			res_ins '处置席' '处置' '' "升件#$eid 销——done-无收口链史债记档（终相票缺收口族链;不可伪造记档即销）"
			resolved=$((resolved+1)) ;;
		*层扫定性升件失领:*)
			# 二阶件:母升件全销则自销——母件还有未销=信号活着让梯
			if [ -n "$etid" ]; then
				popen=$(gw_sql "SELECT count(*) FROM ops_event e WHERE e.event='升件'
				  AND e.note NOT LIKE '升件老化%' AND e.note NOT LIKE '层扫定性升件失领%'
				  AND e.id<>$eid
				  AND (e.ticket='$(gw_qc "$etid")' OR e.esc_scope='$(gw_qc "$etid")')
				  AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts AND x.seat<>'gateway'
				    AND ((e.ticket<>'' AND x.ticket=e.ticket AND x.event IN ('裁决','完成','驳回','挂起'))
				      OR x.note ~ ('升件#' || e.id::text || '([^0-9]|$)')))")
				if [ "${popen:-1}" = "0" ]; then
					res_ins '处置席' '处置' '' "升件#$eid 销——升件失领母件已全销（二阶件自销）"
					resolved=$((resolved+1))
				fi
			fi ;;
		*) : ;; # 裁断/LLM 判断类零动作——让梯归席定性（本件不碰）
	esac
done
[ "${resolved:-0}" -gt 0 ] && gw_ledger 升件路由 GATEWAY "分类归领自销=$resolved（机械可证类——done翻漏/幽灵/无籍/史链/二阶失领;裁断类让梯不变）"

# 席→唤起单元映射（钉死有无头形态者；无单元席=只落账靠量表连鸣）
seat_unit() {
	case "$1" in
		CTO|决策席) printf 'o-cto-seat.service\n' ;;
		CEO|管理席) printf 'o-ceo-seat.service\n' ;;
		MECH)       printf 'o-mech.service\n' ;;
		*)          printf '\n' ;;
	esac
}
# 级→目标席（1=归口默认CTO——esc_scope 命中席名则用之；2=CTO；3=CEO；4=业主）
level_target() { # $1=level $2=esc_scope $3=ticket
	case "$1" in
		1) case "$2" in CTO|CEO|决策席|管理席) printf '%s\n' "$2" ;; *) printf 'CTO\n' ;; esac ;;
		2) printf 'CTO\n' ;;
		3) printf 'CEO\n' ;;
		*) printf 'OWNER\n' ;;
	esac
}

# ── 未认领超龄升件扫描（认领判据=量表同式+gateway 自行排除）─────
mapfile -t ESC_ROWS < <(gw_sql "SELECT e.id, COALESCE(e.ticket,''), COALESCE(e.esc_scope,''),
    COALESCE(replace(substring(e.note from 1 for 60),'|','/'),''),
    COALESCE(extract(epoch from e.ts)::bigint,0)::text
  FROM ops_event e WHERE e.event='升件' AND e.note NOT LIKE '升件老化%'
    AND e.ts < now() - interval '$UNCLAIMED_SEC seconds'
    AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts AND x.seat<>'gateway' AND
      ((e.ticket<>'' AND x.ticket=e.ticket AND x.event IN ('裁决','完成','驳回','挂起'))
       OR x.note ~ ('升件#' || e.id::text || '([^0-9]|$)')))
  ORDER BY e.id")
[ "${#ESC_ROWS[@]}" -gt 0 ] || exit 0

for row in "${ESC_ROWS[@]}"; do
	IFS='|' read -r esc_id tid esc_scope note_head esc_ts <<< "$row"
	[ -n "$esc_id" ] || continue
	# 已路由级数+末次路由时刻（幂等键=升件#id——evergreen 查不重不窗）
	rstat=$(gw_sql "SELECT count(*)::text || '|' || COALESCE(extract(epoch from max(ts))::bigint,0)::text
	  FROM ops_event WHERE event='升件路由' AND note LIKE '%升件#${esc_id} %'")
	rcnt=${rstat%%|*}; rlast=${rstat##*|}
	case "$rcnt" in ''|*[!0-9]*) rcnt=0 ;; esac
	case "$rlast" in ''|*[!0-9]*) rlast=0 ;; esac
	[ "$rcnt" -ge "$MAX_LEVEL" ] && continue # 已到业主级——静默归量表连鸣
	now=$(gw_now)
	if [ "$rcnt" -eq 0 ]; then
		# 一级=未认领龄阈已满足（扫描谓词已含）——即路由
		:
	elif [ $(( now - ${rlast:-0} )) -lt "$LEVEL_COOLDOWN" ]; then
		continue # 级间冷却内不重复 ringing
	fi
	lvl=$(( rcnt + 1 ))
	target=$(level_target "$lvl" "$esc_scope" "$tid")
	lvl_name=$(case "$lvl" in 1) printf '归口';; 2) printf 'CTO';; 3) printf 'CEO';; *) printf '业主';; esac)

	# 账本先行：升件路由行（新词不入相位/不入升件本体——原升件行唯一裁断对象）
	gw_ledger 升件路由 "${tid:-GATEWAY}" "升件#${esc_id} L${lvl}→${target}（${lvl_name}级——未认领超龄机械路由，定性归席）"

	case "$target" in
		OWNER)
			gw_ledger COS上行 "${tid:-GATEWAY}" "升件#${esc_id} 路由业主——L${lvl} 终级（${note_head}）"
			[ -x "$FEISHU_SH" ] && "$FEISHU_SH" "[升件上行] 升件#${esc_id} ${tid:-无票}——${note_head}（L${lvl} 终级路由）" >/dev/null 2>&1 || true # tolerate: feishu 尽力推送——写径败不阻账
			;;
		*)
			unit=$(seat_unit "$target")
			[ -n "$unit" ] && systemctl --user start --no-block "$unit" >/dev/null 2>&1 || true # tolerate: 唤起败=量表连鸣兜底不阻账
			;;
	esac
done
exit 0
