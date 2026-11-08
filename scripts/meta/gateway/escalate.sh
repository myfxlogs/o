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
