#!/usr/bin/env bash
# gateway/route.sh —— o-gateway 路由账子件（T-0469）：谁做什么的机械面——
# ①派单行无拉→补拉（dm_queue dispatched 单源推导+dispatch_worker 共用闸，
#   与 mech_tick 补拉窗逐字同判据——双轨共存期幂等键自带防重）；
# ②交接对账（dm_handoff 钦定件原样调用——断链升件自含）。
# 边界：派单序/人工径滤/模型档恢复/水位闸全在原件内，本件零判据。
# 环境覆写：同 common.sh + O_WORKER_* / O_DISPATCH_* 桩面（mech_tick 同款）。
set -u
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/common.sh"

DM_QUEUE_SH=${O_GW_DM_QUEUE_SH:-"$GW_META_DIR/dm_queue.sh"}
DISPATCH_WORKER=${O_WORKER_DISPATCH_SCRIPT:-/opt/o/scripts/meta/dispatch_worker.sh}
DISPATCH_WATER_SH=${O_DISPATCH_WATER_SH:-"$GW_META_DIR/dispatch_water.sh"}
DISPATCH_ORDER_SH=${O_DISPATCH_ORDER_SH:-"$GW_META_DIR/dispatch_order.sh"}
DM_HANDOFF_SH=${O_GW_HANDOFF_SH:-"$GW_META_DIR/dm_handoff.sh"}
MANUAL_LANE_SH=${O_MANUAL_LANE_SH:-"$GW_META_DIR/manual_lane_check.sh"}
DISPATCH_TASK_DIR=${O_WORKER_TASK_DIR:-/opt/o/docs/tasks}
DSN=$(gw_dsn); [ -n "$DSN" ] || exit 0
export O_DISPATCH_ORDER_TASK_DIR="$DISPATCH_TASK_DIR"

acted=0 # 本拍动作计数——零动作拍不落汇总行（账本去噪：空拍不刷屏）

# ── ① 派单行无拉→补拉（mech_tick.sh:102-141 同式收编——判据逐字同源，
# 拉起行 seat=gateway 记账真（相位投词不查 seat——pulled 出 dispatched
# 集自愈；派单行 seat 约束只管派单词本身不受影响）。
DRC=0
DISPATCH_TIDS=$("$DM_QUEUE_SH" dispatched 2>/dev/null) || DRC=$?
[ "$DRC" -ne 0 ] && gw_esc "" "GW-ROUTE-DISPATCHED-QUERY" "dm_queue dispatched rc=$DRC——候选集不可知按空集滑行"
if [ "$DRC" -eq 0 ]; then
	DISPATCH_TIDS=$(printf '%s\n' "$DISPATCH_TIDS" | "$DISPATCH_ORDER_SH" 2>/dev/null) || true # tolerate: 序重排败=原序滑行（fail-open 非链环）
	while IFS= read -r tid; do
		[ -n "$tid" ] || continue
		compgen -G "$DISPATCH_TASK_DIR/${tid}-*.md" >/dev/null || continue # 无票免拉
		"$MANUAL_LANE_SH" "$tid" "$DISPATCH_TASK_DIR" && continue # rc0=人工径滤除
		st=$(systemctl --user is-active "o-worker-$tid.service" 2>/dev/null || true) # tolerate: is-active查败=态未知视同可拉
		case "$st" in active|activating) continue;; esac
		model=$(psql "$DSN" -qAt -c "SELECT CASE WHEN note ILIKE '%swe-2-flash-high%' THEN 'swe-2-flash-high' WHEN note ILIKE '%swe-2-medium%' THEN 'swe-2-medium' ELSE 'swe-2-flash-high' END FROM ops_event WHERE ticket='$(printf '%s' "$tid" | sed "s/'/''/g")' AND event='派单' ORDER BY ts DESC,id DESC LIMIT 1" 2>/dev/null) # tolerate: 模型档查败=空串→默认档兜底
		[ -n "$model" ] || model=swe-2-flash-high
		prompt="读 /opt/o/skills/施工席/SKILL.md 领单 $tid:只做工单范围内的事——写实现码/测试/提交/回填;范围外发现的问题不自己动手,写进交付备注;同票已有 Devin 会话或 worktree 时续跑不新开"
		if "$DISPATCH_WORKER" "$tid" "$model" "$prompt" >/dev/null 2>&1; then
			psql "$DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
			  VALUES('gateway','拉起','$tid','网关补拉——派单未拉自愈（route.sh）')" >/dev/null 2>&1 || true # tolerate: 拉起行落账尽力而为
			acted=$((acted+1))
		else
			rc=$?
			if [ "$rc" -eq 75 ]; then "$DISPATCH_WATER_SH" "$tid"; acted=$((acted+1)); fi # 候水标记行（自含尽力而为）
		fi
	done <<< "$DISPATCH_TIDS"
	[ "$acted" -gt 0 ] && gw_ledger 路由 GATEWAY "dispatched 补拉窗动作=$acted（拉起/候水行各有账）"
fi

# ── ② 交接对账：dm_handoff 原样调用（断链升件判据全内件——evergreen
# 去重/终态杀票/免报词表/自激防零复制）。败才落账（rc0 空拍不刷屏）。
HRC=0
bash "$DM_HANDOFF_SH" >/dev/null 2>&1 || HRC=$?
[ "$HRC" -ne 0 ] && { gw_esc "" "GW-ROUTE-HANDOFF" "dm_handoff rc=$HRC——断链扫描停摆（席位班扫兜底不瘫）"
	gw_ledger 路由 GATEWAY "handoff rc=$HRC"; }
exit 0
