#!/usr/bin/env bash
# check_deploy_commit_seat.sh —— T-0467 件③部署树 commit 时闸判件
# （§9「施工禁在部署树检出/编辑/提交」前移：commit-msg 阶段才见
# subject 词形，pre-commit 判别面不存在——本件=commit-msg 内嵌
# 调用的纯判定函数，进不了历史=闸语义等价达成）。
#
# 判法（钦定面③+裁填）：root realpath=部署树 ∧ branch=master ∧
# 非 merge 在途（MERGE_HEAD 无——合支正当径不动）→ subject 首词
# 位 [角色:施工] 即拒。判别域=施工前缀负向列——T-0370 语义一字
# 不动（当年闸只禁施工前缀）；无标 commit=DM/决策票立流在案
# （origin/master first-parent 实证 T-0440 系无标立票件常态）
# 不收；[角色:决策]/[角色:COS]/[角色:外审]/[角色:管理] 等他席标
# 直落=既有合法径不收窄（判别域只窄不宽——施工外扩需另票实证）。
#
# 用法：check_deploy_commit_seat.sh <root> <branch> <subject>
#   [expected_root]——expected_root 仅供桩测覆写（缺省 /opt/o；
#   hook 调用不传=恒钉部署树，env/参面无绕过径）。
set -uo pipefail
ROOT=${1:?usage: check_deploy_commit_seat.sh <root> <branch> <subject> [expected_root]}
BR=${2:-}
SUBJ=${3:-}
EXPECTED=${4:-/opt/o}

[ "$(realpath "$ROOT" 2>/dev/null)" = "$EXPECTED" ] || exit 0
[ "$BR" = "master" ] || exit 0
git -C "$ROOT" rev-parse -q --verify MERGE_HEAD >/dev/null 2>&1 && exit 0

case "$SUBJ" in
  \[角色:施工\]*)
    echo "FAIL(T-0467): 部署树 /opt/o master 直落 [角色:施工] commit——" >&2
    echo "  §9 前移闸：施工禁在部署树提交（判例=T-0367 abe3fd1a 直落实证）。" >&2
    echo "  正当径：票支 worktree 施工→check_all 绿→合支随 merge 入 master。" >&2
    echo "  拒因 subj=[$SUBJ]" >&2
    exit 1 ;;
esac
exit 0
