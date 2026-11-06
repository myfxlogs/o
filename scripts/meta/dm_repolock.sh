#!/usr/bin/env bash
# dm_repolock.sh —— /opt/o 部署树 master 同台写径互斥锁（T-0414 件④——
# 部署锁型复用：mkdir 原子+holder 件+陈锁抢，形制照 deploy_oci
# .oci-deploy-lock）。护的是 git 写序：壳轨票翻（dm_ledger.sh）与会话
# merge/push 同一写径互斥，无双写竞态；读径（fetch/log/status）不闸。
#   用法:
#     dm_repolock.sh acquire <holder词> [wait秒] → 0=得（stdout 一行 token）
#       3=忙（stderr 持锁者行）；6=抢陈锁得（stdout token，stderr STOLEN 行——
#       调用面宜升件显报）；1=异常
#     dm_repolock.sh release <token>          → 0=放/锁目已失；4=token 不符
#     dm_repolock.sh status                   → holder 件文（无锁=空出 rc0）
#   锁目=<git-common-dir>/dm-repo.lock（worktree 共 .git 同锁——主树 wt 同径）。
# env: O_REPO_LOCK_REPO=/opt/o O_REPO_LOCK_STALE=1800 O_REPO_LOCK_DIR=覆盖锁目（测试桩）
set -u
CMD=${1:-}
REPO=${O_REPO_LOCK_REPO:-/opt/o}
STALE=${O_REPO_LOCK_STALE:-1800}
if [ -n "${O_REPO_LOCK_DIR:-}" ]; then
	LOCK=$O_REPO_LOCK_DIR
else
	GD=$(git -C "$REPO" rev-parse --git-common-dir 2>/dev/null) || { echo "git-common-dir 不可解 repo=$REPO" >&2; exit 1; }
	case "$GD" in /*) ;; *) GD="$REPO/$GD";; esac
	LOCK="$GD/dm-repo.lock"
fi

holder_line() { cat "$LOCK/holder" 2>/dev/null || true; } # tolerate: holder 件缺席=残壳锁目——空出按无锁/陈锁判
lock_age() { # 锁目 mtime 龄秒；不可读=0
	local mt now
	mt=$(stat -c %Y "$LOCK" 2>/dev/null) || { echo 0; return; }
	now=$(date +%s)
	echo $(( now > mt ? now - mt : 0 ))
}

case "$CMD" in
acquire)
	HOLDER=${2:?用法: dm_repolock.sh acquire <holder> [wait秒]}
	WAIT=${3:-0}
	DEAD=$(( $(date +%s) + WAIT ))
	TOKEN="$HOLDER-$$-$(date +%s%N)"
	while :; do
		if mkdir "$LOCK" 2>/dev/null; then
			printf 'token=%s\nholder=%s\npid=%s\nts=%s\n' "$TOKEN" "$HOLDER" "$$" "$(date -Is)" >"$LOCK/holder" 2>/dev/null || true # tolerate: holder 件写败=锁仍持——身份行失不影响互斥语义
			printf '%s\n' "$TOKEN"
			exit 0
		fi
		age=$(lock_age)
		if [ "$age" -gt "$STALE" ]; then
			prev=$(holder_line)
			if rm -rf -- "$LOCK" 2>/dev/null && mkdir "$LOCK" 2>/dev/null; then
				printf 'token=%s\nholder=%s\npid=%s\nts=%s\nstolen=%s\n' "$TOKEN" "$HOLDER" "$$" "$(date -Is)" "${prev:-<空>}" >"$LOCK/holder" 2>/dev/null || true # tolerate: 同上
				printf '%s\n' "$TOKEN"
				printf 'STOLEN age=%ss>%ss prev=%s\n' "$age" "$STALE" "${prev:-<空>}" >&2
				exit 6
			fi
		fi
		[ "$(date +%s)" -ge "$DEAD" ] && break
		sleep 2
	done
	echo "[dm-repolock] busy holder: $(holder_line)" >&2
	exit 3
	;;
release)
	TOKEN=${2:?用法: dm_repolock.sh release <token>}
	[ -d "$LOCK" ] || exit 0 # tolerate: 锁目已失=释意已达成——幂等放行
	cur=$(sed -n 's/^token=//p' "$LOCK/holder" 2>/dev/null | head -1)
	if [ "$cur" = "$TOKEN" ]; then
		rm -rf -- "$LOCK" 2>/dev/null || true # tolerate: rm 败=残壳由陈锁抢兜底——结构性限度
		exit 0
	fi
	echo "[dm-repolock] release 拒——token 不符（持锁者非本拍）" >&2
	exit 4
	;;
status)
	holder_line
	exit 0
	;;
*)
	echo "用法: dm_repolock.sh acquire <holder> [wait秒] | release <token> | status" >&2
	exit 2
	;;
esac
