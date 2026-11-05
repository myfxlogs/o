#!/usr/bin/env bash
# check_master_direct.sh —— T-0370 件①：master 直推缝闸（部署树红线
# §9「施工禁在部署树检出/编辑/提交」的推面机检补格）。
# 实证病灶：T-0367 无头施工 run 把票面回填 commit 直落 /opt/o 部署树
# master 并推 origin 成功——本地 check_all 绿戳过 pre-push 形闸，
# 红线被事实绕过（机制上施工 run 持危险权限可直推 master）。
# 判法：git log --first-parent <remote_sha>..<local_sha>——施工 commit
# 经合支（merge 第二亲链）不入 first-parent=正当径放行；直落 master
# 线（含 ff 合支形）=拒。豁免裁定=零豁免：施工 commit 到 master 唯一
# 正当径=票支合入，台账/docs 白名单件不豁免（§9 禁的是部署树提交
# 本身非内容类；回填落己支随合入到 master）。
# 用法：check_master_direct.sh <local_sha> <remote_sha>（pre-push 喂参；
# 删 ref/新 ref 全零=无对账对象跳过——master 禁删/禁新建形由形闸兜）。
# fail-closed：对账区间不可解（remote_sha 本地缺件）=拒推——master 推
# 面不可解=不放行，先 fetch 对齐再推。
set -uo pipefail
LOCAL=${1:?usage: check_master_direct.sh <local_sha> <remote_sha>}
REMOTE=${2:?usage: check_master_direct.sh <local_sha> <remote_sha>}
ZERO=0000000000000000000000000000000000000000
[ "$LOCAL" = "$ZERO" ] && exit 0
[ "$REMOTE" = "$ZERO" ] && exit 0

RANGE_OUT=$(git log --first-parent --format='%h %s' "$REMOTE..$LOCAL" 2>/dev/null)
RC=$?
if [ "$RC" -ne 0 ]; then
	echo "❌ [master-direct] 对账区间不可解（${REMOTE:0:12}..${LOCAL:0:12}）——fail-closed 拒推，先 git fetch 对齐 remote_sha" >&2
	exit 1
fi
OFFENDERS=$(printf '%s\n' "$RANGE_OUT" | grep -E '^[0-9a-f]+ \[角色:施工\]' || true) # tolerate: grep 空集=零违规合法径
[ -z "$OFFENDERS" ] && exit 0
echo "❌ [master-direct] master first-parent 链检出 [角色:施工] 前缀 commit——施工禁直写部署树（§9；合支径施工 commit 居 merge 第二亲不在此列）:" >&2
echo "$OFFENDERS" >&2
exit 1
