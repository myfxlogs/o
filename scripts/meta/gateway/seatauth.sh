#!/usr/bin/env bash
# gateway/seatauth.sh —— o-gateway 席权账子件（T-0469）：谁起谁退的
# 机械面——已批未领即派（dm_depgate 钦定件原样调用）+死席唤醒
# （ops_headless 道龄超阈∧队有件→systemctl 拉起席单元，冷却防连唤）。
# 边界：只机械调度——派单判据/依赖交集/水位全在 depgate/dispatch_worker
# 原件内，本件零判据；席位判断永归席自身（唤起=敲门非代裁）。
# 幂等：depgate/dispatch_worker 内部幂等键自带；唤醒冷却=state 件。
# 环境覆写：同 common.sh（O_GATEWAY_* / O_*_SH 桩面）。
set -u
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/common.sh"

DM_DEPGATE_SH=${O_GW_DEPGATE_SH:-"$GW_META_DIR/dm_depgate.sh"}
DM_QUEUE_SH=${O_GW_DM_QUEUE_SH:-"$GW_META_DIR/dm_queue.sh"}
STALE_SEC=${O_GW_SEAT_STALE_SEC:-600}        # 席道龄阈（10min）
WAKE_COOLDOWN=${O_GW_SEAT_WAKE_COOLDOWN:-600} # 同席连唤冷却（10min）
DSN=$(gw_dsn); [ -n "$DSN" ] || exit 0

# ── ① 已批未领即派：dm_depgate 原样调用（依赖闸+派单序+水位闸全内件）。
# 空拍不落汇总行（账本去噪——动作账=派单/拉起/候水各原件行+seatwake 行）。
DEPGATE_RC=0
bash "$DM_DEPGATE_SH" >/dev/null 2>&1 || DEPGATE_RC=$?
[ "$DEPGATE_RC" -ne 0 ] && { gw_esc "" "GW-SEATAUTH-DEPGATE" "dm_depgate rc=$DEPGATE_RC——已批快派面不可知按常径排队（席位班扫兜底不瘫）"
	gw_ledger 席权 GATEWAY "depgate rc=$DEPGATE_RC"; }

# ── ② 死席唤醒：lane 道龄>STALE 且队有件→拉起席单元（冷却内不重复唤）。
# 席→单元映射钉死两无头席（CTO=决策席 dm_tick / CEO=管理席 ceo_tick）；
# 机械轨/巡检/分析等他域单元不归本账（自有 timer 兜底原样）。
seat_lane_stale() { # $1=ops_headless ticket 道名 → 道龄秒（无行=极大值）
	gw_sql "SELECT EXTRACT(EPOCH FROM now()-max(last_activity))::bigint
	  FROM ops_headless WHERE ticket='$1'" | head -1
}
seat_queue_nonempty() { # $1=CTO|CEO → 队面非空判据（机械只读不判）
	case "$1" in
		CTO) "$DM_QUEUE_SH" 2>/dev/null | grep -q . ;;
		CEO) # 呈批滞留=CEO 队件代理面（已批/裁决后随即出队）
			[ -n "$(gw_sql "SELECT 1 FROM ops_event e WHERE e.event='呈批'
			  AND e.ts>now()-interval '24 hours' AND NOT EXISTS
			  (SELECT 1 FROM ops_event x WHERE x.ticket=e.ticket
			   AND x.event IN ('已批','裁决','驳回','领单','处置','挂起')
			   AND x.ts>e.ts) LIMIT 1")" ] ;;
	esac
}
seat_wake() { # $1=seat名 $2=unit名 —— 冷却闸+拉起+落账
	local seat=$1 unit=$2 last now age
	last=$(gw_state_get "seatwake.$seat"); last=${last:-0}
	now=$(gw_now); age=$(seat_lane_stale "$seat"); age=${age:-999999}
	[ "$age" -lt "$STALE_SEC" ] && return 0
	[ $(( now - last )) -lt "$WAKE_COOLDOWN" ] && return 0
	seat_queue_nonempty "$seat" || return 0
	if systemctl --user start --no-block "$unit" >/dev/null 2>&1; then
		gw_state_set "seatwake.$seat" "$now"
		gw_ledger 席权 GATEWAY "seatwake $seat 道龄${age}s 队有件→start $unit"
	fi
	return 0
}
seat_wake CTO o-cto-seat.service
seat_wake CEO o-ceo-seat.service
exit 0
