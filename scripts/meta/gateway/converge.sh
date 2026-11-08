#!/usr/bin/env bash
# gateway/converge.sh —— o-gateway 收敛账子件（T-0469）：推已绿 sha→
# 净发窗分面发射→验活落账的机械面。
# ①冻结断言：升件未决（24h 窗认领判据同 escalation_unclaimed 量表式，
#   '升件老化'自鸣前缀除外防自激）→收敛面全面不发（push/deploy 全冻，
#   V0 钦定「升件未决/事故期→全面不发」继承）；
# ②绿戳推：戳面签名变∪镜推冷却到→mirror_push.sh 原样调用（推已绿
#   sha 钦定件——三断言/ff/HEAD包含闸全内件）；
# ③净发窗：deploy_window_due.sh 原样调用只读 state/faces（判据全内件）
#   →HEAD==origin/master 收敛态→faces 分面发射 deploy_image/deploy_oci
#   （面级幂等=部署行 sha12 查重+DEPLOY-FIRE-FAIL 面冻 24h——与
#   mech_tick 同键共享冻结账，双轨共存期零重发）；
# ④验活：rc=0 后部署行须在案（deploy_* 完工同拍落簿）——缺席=
#   半截部署疑升件（DEPLOY-VERIFY-FAIL）。
# 边界：判「该不该发」永归 deploy_window_due/deploy_* 本体，本件零判据；
# 败即升件+面冻结+不自愈重试。环境覆写：同 common.sh + O_DEPLOY_* 桩面。
set -u
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/common.sh"

MIRROR_PUSH_SH=${O_GW_MIRROR_PUSH_SH:-"$GW_REPO/scripts/mirror_push.sh"}
DEPLOY_WINDOW_SH=${O_GW_DEPLOY_WINDOW_SH:-"$GW_META_DIR/deploy_window_due.sh"}
DEPLOY_IMAGE_SH=${O_DEPLOY_IMAGE_SH:-"$GW_REPO/scripts/deploy_image.sh"}
DEPLOY_OCI_SH=${O_DEPLOY_OCI_SH:-"$GW_REPO/scripts/deploy_oci.sh"}
FREEZE_WIN=${O_GW_FREEZE_WINDOW_SEC:-86400}
MIRROR_MIN=${O_GW_MIRROR_MIN_SEC:-60}
case "$FREEZE_WIN" in ''|*[!0-9]*) FREEZE_WIN=86400 ;; esac
case "$MIRROR_MIN" in ''|*[!0-9]*) MIRROR_MIN=60 ;; esac
DSN=$(gw_dsn); [ -n "$DSN" ] || exit 0

# ── ① 冻结断言：升件未决→全面不发 ─────────────────────────────
# 认领判据=escalation_unclaimed 量表同式（同票后随 裁决/完成/驳回/挂起
# 或任后行 note 引证 升件#<id>）——量表口径单源不新造。
fz=$(gw_sql "SELECT count(*) FROM ops_event e WHERE e.event='升件'
  AND e.note NOT LIKE '升件老化%' AND e.ts>now()-interval '$FREEZE_WIN seconds'
  AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts AND x.seat<>'gateway' AND
    ((e.ticket<>'' AND x.ticket=e.ticket AND x.event IN ('裁决','完成','驳回','挂起'))
     OR x.note ~ ('升件#' || e.id::text || '([^0-9]|$)')))")
fz=${fz:-0}
case "$fz" in ''|*[!0-9]*) fz=0 ;; esac
if [ "$fz" -gt 0 ]; then
	sig="fz$fz"; now=$(gw_now)
	lastsig=$(gw_state_get converge.freeze_sig); lastts=$(gw_state_get converge.freeze_ts); lastts=${lastts:-0}
	if [ "$sig" != "$lastsig" ] || [ $(( now - lastts )) -ge 3600 ]; then
		gw_ledger 收敛 GATEWAY "freeze——升件未决${fz}件（${FREEZE_WIN}s 窗）收敛面全面不发"
		gw_state_set converge.freeze_sig "$sig"; gw_state_set converge.freeze_ts "$now"
	fi
	gw_state_set converge.frozen 1
	exit 0
fi
gw_state_set converge.frozen 0

# ── ② 绿戳推：戳面签名变∪镜推冷却到→mirror_push 原样调用 ──────
GITDIR=$(git -C "$GW_REPO" rev-parse --git-common-dir 2>/dev/null || true) # tolerate: git 面不可解=空→戳签名空→冷却径仍跑 mirror_push 自裁
STAMP_SIG=""
[ -n "$GITDIR" ] && STAMP_SIG=$(ls -t "$GITDIR"/check-green-* 2>/dev/null | head -1 | xargs -r stat -c '%n|%Y' 2>/dev/null) # tolerate: 戳面缺=空签名合法径
last_sig=$(gw_state_get converge.stamp_sig); last_push=$(gw_state_get converge.mirror_ts); last_push=${last_push:-0}
now=$(gw_now)
if [ "$STAMP_SIG" != "$last_sig" ] || [ $(( now - last_push )) -ge "$MIRROR_MIN" ]; then
	mrc=0
	O_MIRROR_REPO="$GW_REPO" bash "$MIRROR_PUSH_SH" >/dev/null 2>&1 || mrc=$?
	gw_state_set converge.stamp_sig "$STAMP_SIG"; gw_state_set converge.mirror_ts "$now"
	[ "$mrc" -ne 0 ] && { gw_esc "" "GW-CONVERGE-MIRROR" "mirror_push rc=$mrc——绿戳推面败（席位班扫兜底不瘫）"
		gw_ledger 收敛 GATEWAY "mirror_push rc=$mrc"; }
fi

# ── ③ 净发窗：deploy_window_due 判据→faces 分面发射 ────────────
WIN_OUT=$(bash "$DEPLOY_WINDOW_SH" 2>/dev/null); WRC=$?
WSTATE=$(printf '%s\n' "$WIN_OUT" | sed -n 's/^state=//p' | head -1)
WFACES=$(printf '%s\n' "$WIN_OUT" | sed -n 's/^faces=//p' | head -1)
WREASON=$(printf '%s\n' "$WIN_OUT" | sed -n 's/^reason=//p' | head -1)
SHA12=$(git -C "$GW_REPO" rev-parse --short=12 HEAD 2>/dev/null || true) # tolerate: sha 不可解=空→等值判跳过
# PLAN 落账去重：签名=state|faces|sha12 变才写（账本先行——判定先于动作）
plan_sig="${WSTATE:-rc$WRC}|${WFACES:-}|${SHA12:-?}"
last_plan=$(gw_state_get converge.plan_sig)
if [ "$plan_sig" != "$last_plan" ]; then
	gw_ledger 收敛 GATEWAY "PLAN state=${WSTATE:-不可读} faces=${WFACES:-空} sha=${SHA12:-?} rc=$WRC ${WREASON:-}"
	gw_state_set converge.plan_sig "$plan_sig"
fi
[ "$WRC" -ne 0 ] && [ -z "$WSTATE" ] && { gw_esc "" "GW-CONVERGE-WINDOW" "deploy_window_due rc=$WRC 输出不可读——窗判据面不可知按空集滑行"; exit 0; }

if [ "$WSTATE" = "应发" ] && [ -n "$WFACES" ]; then
	OM_SHA=$(git -C "$GW_REPO" rev-parse origin/master 2>/dev/null || true) # tolerate: 远端基线不可解=空→与 HEAD 等值判恒不等=未收敛跳过（fail-closed 不发）
	HEAD_SHA=$(git -C "$GW_REPO" rev-parse HEAD 2>/dev/null || true) # tolerate: HEAD 不可解=空→同上恒不等=跳过
	if [ -z "$SHA12" ] || [ "$HEAD_SHA" != "$OM_SHA" ]; then
		: # 未收敛=跳过待下拍（非败不冻——mech_tick 同律）
	else
		fire_deploy_face() { # $1=face(image|oci)——面级幂等+冻结+发+验
			local face=$1 keypat doneq frz frc vd vpat
			case "$face" in image|oci) ;; *) return 0 ;; esac
			keypat="DEPLOY-FIRE-FAIL:[0-9a-f]+:$face([^a-z]|$)"
			frz=$(psql "$DSN" -qAt -c "SELECT 1 FROM ops_event WHERE event='升件'
			  AND note ~ '$keypat' AND ts>now()-interval '24 hours' LIMIT 1" 2>/dev/null); frc=$?
			if [ "$frc" -ne 0 ]; then
				gw_esc "" "DEPLOY-FREEZE-CHECK-FAIL:$face" "冻结态查询败 rc=$frc——按不可判不发（fail-closed）"
				return 0
			fi
			[ "$frz" = "1" ] && return 0 # 面冻结 24h 窗内不重试
			if [ "$face" = image ]; then
				doneq="SELECT 1 FROM ops_event WHERE event='部署'
				  AND substring(note from 'deploy ([0-9a-f]+)')='$SHA12'
				  AND note LIKE '% rc=0 %' LIMIT 1"
				vpat='deploy ([0-9a-f]+)'
			else
				doneq="SELECT 1 FROM ops_event WHERE event='部署'
				  AND substring(note from 'deploy-oci ([0-9a-f]+)')='$SHA12'
				  AND note LIKE '% rc=0 %' LIMIT 1"
				vpat='deploy-oci ([0-9a-f]+)'
			fi
			vd=$(psql "$DSN" -qAt -c "$doneq" 2>/dev/null); frc=$?
			if [ "$frc" -ne 0 ]; then
				gw_esc "" "DEPLOY-IDEM-CHECK-FAIL:$face" "部署行幂等查询败 rc=$frc——按不可判不发（fail-closed）"
				return 0
			fi
			[ "$vd" = "1" ] && return 0 # 幂等键 sha:face 已部署——零重复动作
			gw_ledger 收敛 GATEWAY "FIRE deploy_$face sha=$SHA12（幂等键 sha:face——账本先行）"
			case "$face" in
				image) bash "$DEPLOY_IMAGE_SH" ;;
				oci)   bash "$DEPLOY_OCI_SH" ;;
			esac
			frc=$?
			if [ "$frc" -ne 0 ]; then
				gw_esc "" "DEPLOY-FIRE-FAIL:$SHA12:$face" "deploy_$face rc=$frc——本面冻结24h不重试（判据在 deploy_window_due）"
				return 0
			fi
			# ④ 验活：部署行在案=完工落簿实证（缺席=半截部署疑显形）
			vd=$(psql "$DSN" -qAt -c "SELECT 1 FROM ops_event WHERE event='部署'
			  AND substring(note from '$vpat')='$SHA12'
			  AND note LIKE '% rc=0 %' AND ts>now()-interval '10 minutes' LIMIT 1" 2>/dev/null)
			if [ "$vd" = "1" ]; then
				gw_ledger 收敛 GATEWAY "VERIFY deploy_$face sha=$SHA12——部署行在案 rc=0"
			else
				gw_esc "" "DEPLOY-VERIFY-FAIL:$SHA12:$face" "deploy_$face rc=0 但部署行缺席（10min 窗）——半截部署疑验活不可知"
				gw_ledger 收敛 GATEWAY "VERIFY-FAIL deploy_$face sha=$SHA12——部署行缺席"
			fi
			return 0
		}
		IFS=',' read -ra _FACES <<< "$WFACES"
		for f in "${_FACES[@]}"; do
			f=$(printf '%s' "$f" | tr -d '[:space:]')
			fire_deploy_face "$f"
		done
	fi
fi
exit 0
