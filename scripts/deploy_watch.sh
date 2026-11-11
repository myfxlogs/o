#!/usr/bin/env bash
# deploy_watch.sh —— 部署管道自醒守望（业主令 2026-11-10「收口与命脱钩」：
# merge→push→deploy 最后一步不再靠会话寿命——master HEAD 与产线镜像 SHA
# 滞差即自动部署；同步失败/部署败→升件 deploy_lag/deploy_blocked 转信号，
# 唤知会话清障。被动件有 merge 才醒的钦定形态，15min 一拍幂等）。
set -uo pipefail
cd "$(git rev-parse --show-toplevel)" || exit 1
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN="${FIN_PG_DSN:-$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')}"
say() { echo "[deploy_watch] $*"; }
esc() { # 升件 dedup=未销才抑（同 escEdgeU 语义——销而未修重火）
	local scope="$1" note="$2"
	[ -n "$DSN" ] || return 0
	psql "$DSN" -qAt -c "
		SELECT count(*) FROM ops_event e WHERE e.event='升件' AND e.esc_scope='$scope'
		AND e.ts>now()-interval '24 hours'
		AND NOT EXISTS (SELECT 1 FROM ops_event x2 WHERE x2.ts>e.ts
			AND x2.note ~ ('升件#'||e.id::text||'([^0-9]|$)'))" 2>/dev/null | grep -qx 0 \
		&& psql "$DSN" -c "INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
			VALUES('探测席','升件','','$note','部署','$scope')" >/dev/null 2>&1 # tolerate: 升件写败只剩下拍重试，无二级径
}
git fetch origin master -q 2>/dev/null
LOCAL=$(git rev-parse master 2>/dev/null); REMOTE=$(git rev-parse origin/master 2>/dev/null)
[ -n "$LOCAL" ] && [ -n "$REMOTE" ] || { say "ref 缺——退出"; exit 0; }
BASE=$(git merge-base master origin/master)
if [ "$LOCAL" != "$REMOTE" ]; then
	if [ "$BASE" = "$LOCAL" ]; then
		git merge --ff-only origin/master -q 2>/dev/null && say "ff 至 origin/master $(git rev-parse --short master)"
		LOCAL=$(git rev-parse master)
	elif [ "$BASE" = "$REMOTE" ]; then
		# 尖未盖戳=合流在途（merge→check_all→converge 序内）——让位盖戳流
		# 不自推不升件（盖戳后本拍或会话径自然上岸；盲推撞 pre-push 闸=
		# 自伤型升件源，2026-11-10 产线实证 deploy_push_blocked 三连发）。
		if [ ! -f ".git/check-green-$LOCAL" ]; then
			say "尖未盖戳——让位盖戳流（不推不升）"
			exit 0 # 让位=真让位（不落部署段——未推尖部署必遭出处闸拒=自伤升件）
		elif ! git push origin master 2>/dev/null; then
			esc "deploy_push_blocked" "部署/deploy_push_blocked: 本地 master 超前未推成（pre-push 闸或网阻）——merge 在仓未上岸,归唤知会话查闸因"
			say "push 败——升件已落"
			exit 0 # 推败=尖未上岸——继续走部署=出处闸拒必升件（噪音类根修,stuck_esc#10657 实证）
		fi
	else
		esc "deploy_diverged" "部署/deploy_diverged: master 与 origin/master 分叉（本地=$LOCAL 远端=$REMOTE）——不自裁,归唤知会话"
		exit 0
	fi
fi
HEAD=$(git rev-parse master)
DEPLOYED=$([ -n "$DSN" ] && psql "$DSN" -qAt -c "SELECT (regexp_match(note,'o-core:([0-9a-f]+)'))[1] FROM ops_event WHERE seat='DEPLOY' AND event='部署' AND note LIKE '%rc=0%' ORDER BY id DESC LIMIT 1" 2>/dev/null)
SHORT=${HEAD:0:12}
if [ -z "$DEPLOYED" ] || [ "${HEAD:0:${#DEPLOYED}}" != "$DEPLOYED" ]; then
	# 部署锁持=另径部署在途（会话 deploy_at_sha/本拍并发）——让位不撞锁
	# （撞锁 rc=1+败事件+deploy_blocked 升件=自伤型信号源，2026-11-10
	# 产线实证每次部署首试 rc=1→重试 rc=0 三连序列同型）。
	if ssh "${O_SG_HOST:-root@192.229.85.35}" "test -d /opt/o/.deploy-lock" 2>/dev/null; then
		say "滞差但部署锁持（另径在途）——让位本拍"
		exit 0
	fi
	# 同 SHA 近窗败避让：DEPLOY 账内本 sha 近 15min 已有 rc!=0 试——
	# 让位本拍不落新行（重试风暴闸：唤源任意重复唤皆安全——
	# 2026-11-11 产线实证「HEAD 未推窗+事件唤链」rc=1 逐唤刷屏；
	# 窗期后真滞差照试不误发，升件侧 deploy_blocked 未销已抑复式）。
	if [ -n "$DSN" ]; then
		LASTFAIL=$(psql "$DSN" -qAt -c "SELECT id FROM ops_event WHERE seat='DEPLOY' AND event='部署' AND note LIKE '%o-core:${SHORT}%' AND note NOT LIKE '%rc=0%' AND ts>now()-interval '15 minutes' ORDER BY id DESC LIMIT 1" 2>/dev/null)
		[ -n "$LASTFAIL" ] && { say "同 sha 近窗部署败在案(#$LASTFAIL)——让位本拍"; exit 0; }
	fi
	say "滞差: 部署=$DEPLOYED vs master=$SHORT——起部署"
	out=$(bash scripts/deploy_image.sh 2>&1); rc=$?
	echo "$out" | tail -3 | while read -r l; do say "$l"; done
	[ $rc -ne 0 ] && esc "deploy_blocked" "部署/deploy_blocked: deploy_image rc=$rc（master=$SHORT 待部署）尾=$(echo "$out" | tail -2 | tr '\n' ' ' | cut -c1-120)——归唤知会话清障（脏树/锁/网）"
else
	say "已同步（部署=$DEPLOYED = master）"
fi
