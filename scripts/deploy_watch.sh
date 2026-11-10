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
		if ! git push origin master 2>/dev/null; then
			esc "deploy_push_blocked" "部署/deploy_push_blocked: 本地 master 超前未推成（pre-push 闸或网阻）——merge 在仓未上岸,归唤知会话查闸因"
			say "push 败——升件已落"
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
	say "滞差: 部署=$DEPLOYED vs master=$SHORT——起部署"
	out=$(bash scripts/deploy_image.sh 2>&1); rc=$?
	echo "$out" | tail -3 | while read -r l; do say "$l"; done
	[ $rc -ne 0 ] && esc "deploy_blocked" "部署/deploy_blocked: deploy_image rc=$rc（master=$SHORT 待部署）尾=$(echo "$out" | tail -2 | tr '\n' ' ' | cut -c1-120)——归唤知会话清障（脏树/锁/网）"
else
	say "已同步（部署=$DEPLOYED = master）"
fi
