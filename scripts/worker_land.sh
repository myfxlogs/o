#!/usr/bin/env bash
# worker_land.sh —— 施工席落地道（业主裁面 2026-11-16 钦定分流）：
#   完工支→land_scope 分流：FREE 面→戳在案+ff-merge 落 master
#   （mirror_push→deploy_watch 自动上船）；REVIEW 面→推支候审不自合。
#
# 用法：worker_land.sh <branch>   （在仓内跑；支可为本地名或 origin/ 名）
#   rc0=已合 master；rc2=钱径面已推支候审；rc1=戳缺/戳陈/非ff/取数败。
#
# 戳规：check-green-<支尖> 须存在且 ≤24h——陈戳=闸过期，重跑 check_all。
# merge 域=/opt/o 部署树——只许 --ff-only 对齐（戒律面：只 pull/deploy，
# 本地合点禁留，merge 簿记归 land_queue 独树，工单#22949）——脏树/非ff=拒。
# 本件在免疫册——改本闸=触免疫面。
set -u
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO="${O_LAND_REPO:-/opt/o}"
br="${1:?用法: worker_land.sh <branch>}"
cd "$REPO" || exit 1
git fetch -q origin master 2>/dev/null || { echo "[worker_land] fetch 败" >&2; exit 1; }

tip=$(git rev-parse "$br" 2>/dev/null || git rev-parse "origin/$br" 2>/dev/null) \
	|| { echo "[worker_land] 支名解不出 $br（本地无且 origin/ 无）" >&2; exit 1; }

bash "$SCRIPT_DIR/land_scope.sh" "$tip" origin/master
rc=$?
if [ "$rc" -eq 2 ]; then
	case "$br" in # 支名含/才推支（传 sha 者无名可推——尖已留 origin 支面在案）
		*/*) git push -q origin "$tip:refs/heads/${br#origin/}" 2>/dev/null \
			|| "$SCRIPT_DIR/cognition/escalate.sh" DEPLOY "" "" "" "worker_land 推支败" \
				"$br 尖 $tip 未推上 origin——候审面缺支（本地仓仍可取）" >/dev/null 2>&1 \
				|| true # tolerate: 推支败走升件自报径——escalate 末环限度在案 ;;
	esac
	echo "[worker_land] 钱径面——支尖 $tip 已留 origin 候审（merge 归唤知/外席）"
	exit 2
fi
[ "$rc" -eq 0 ] || exit 1

stamp="$(git rev-parse --git-common-dir)/check-green-$tip"
[ -f "$stamp" ] || { echo "[worker_land] 无绿戳 check-green-$tip——先过 check_all" >&2; exit 1; }
age=$(( $(date +%s) - $(stat -c %Y "$stamp") ))
[ "$age" -le 86400 ] || { echo "[worker_land] 绿戳 ${age}s 陈——重跑 check_all" >&2; exit 1; }

git checkout -q master || { echo "[worker_land] 部署树不在 master 面且切不回——拒动" >&2; exit 1; }
git pull -q --ff-only origin master || { echo "[worker_land] master 对齐败" >&2; exit 1; }
git merge --ff-only "$tip" || { echo "[worker_land] 非快进——master 已前进，支须重基重测" >&2; exit 1; }
echo "[worker_land] 已合 master=$tip（绿戳 ${age}s）——mirror_push/deploy_watch 接力上船"
