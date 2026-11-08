#!/usr/bin/env bash
# oci_ci_push.sh —— 输送径：施工分支推 OCI scratch → runner 自动 check_all scoped
# → 绿回报回本盒核签落戳；缺席/超时/不可核=fail-open 本盒现径（提案判据④）。
#
# 用法：scripts/oci_ci_push.sh [branch]   （缺省=当前分支；master 不走 OCI——终盖归本盒）
#
# 半成品窗纪律：push 只对 oci-ci scratch（--no-verify 跳过 origin 专用 pre-push
# 形闸——scratch 不是 origin，refs 永不带出）；戳经 oci_ci_accept.sh 落
# git-common-dir，与本地戳同域同格式——mirror_push/pre-push 消费面零改动。
#
# 环境：OCI_CI_SSH / OCI_CI_ROOT / OCI_CI_PUSH_REMOTE / OCI_CI_TIMEOUT /
#       OCI_CI_POLL / OCI_CI_FORCE_LOCAL=1（演练 fail-open）/
#       OCI_CI_LOCAL_FALLBACK（测试面替换本盒现径命令）。
set -uo pipefail

ROOT=$(git rev-parse --show-toplevel 2>/dev/null) || { echo "❌ [oci-ci] 非 git 树" >&2; exit 2; }
SELF=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
GITDIR=$(git rev-parse --git-common-dir 2>/dev/null) || exit 2
GITDIR=$(cd "$GITDIR" 2>/dev/null && pwd -P) || exit 2

REMOTE=${OCI_CI_PUSH_REMOTE:-oci-ci}
SSH_TARGET=${OCI_CI_SSH:-ubuntu@149.118.58.98}
CI_ROOT=${OCI_CI_ROOT:-/home/ubuntu/o-ci}
WAIT=${OCI_CI_TIMEOUT:-3600}
POLL=${OCI_CI_POLL:-5}
BRANCH=${1:-$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)} # tolerate: 分离 HEAD=空——下面 [ -n ] 拦用法错

local_check() { # $1=交回本盒理由
  echo "⚠️ [oci-ci] $1——fail-open 本盒现径（scripts/check_all.sh）" >&2
  if [ -n "${OCI_CI_LOCAL_FALLBACK:-}" ]; then
    bash -c "$OCI_CI_LOCAL_FALLBACK"
  else
    bash "$ROOT/scripts/check_all.sh"
  fi
  exit $?
}

[ -n "$BRANCH" ] || { echo "❌ [oci-ci] 分离 HEAD/无分支——用法: oci_ci_push.sh <branch>" >&2; exit 2; }
case "$BRANCH" in *[!A-Za-z0-9._/-]*) echo "❌ [oci-ci] 分支名字符异常: $BRANCH" >&2; exit 2 ;; esac
SHA=$(git rev-parse --verify "$BRANCH^{commit}" 2>/dev/null) \
  || { echo "❌ [oci-ci] 分支不可解: $BRANCH" >&2; exit 2; }
[ "$BRANCH" != master ] || local_check "master 不走 OCI（档 A 终盖归本盒）"
[ "${OCI_CI_FORCE_LOCAL:-}" = 1 ] && local_check "OCI_CI_FORCE_LOCAL=1"
git remote get-url "$REMOTE" >/dev/null 2>&1 \
  || local_check "remote $REMOTE 未配置（先跑 scripts/oci_ci_provision.sh）"
git rev-parse --verify -q master >/dev/null 2>&1 \
  || local_check "本仓无 master 基线（scope 对拍缺）"

echo "▶ [oci-ci] push ${SHA:0:12} ($BRANCH) + master 基线 → $REMOTE" >&2
if ! timeout "${OCI_CI_PUSH_TIMEOUT:-600}" git push --no-verify "$REMOTE" \
     "master:refs/heads/master" "$SHA:refs/heads/$BRANCH" >&2; then
  local_check "push $REMOTE 败"
fi

echo "⏳ [oci-ci] 等 OCI runner 回报（sha=${SHA:0:12}，上界 ${WAIT}s）…" >&2
N=$(( WAIT / POLL )); [ "$N" -ge 1 ] || N=1
if ! ssh -o BatchMode=yes -o ConnectTimeout=10 -o ServerAliveInterval=30 "$SSH_TARGET" \
  "export XDG_RUNTIME_DIR=/run/user/\$(id -u); for i in \$(seq 1 $N); do [ -f '$CI_ROOT/reports/$SHA.report' ] && exit 0; sleep $POLL; done; exit 66" 2>/dev/null; then
  local_check "report 超时/SSH 缺席（${WAIT}s）"
fi

TMP=$(mktemp -d) || local_check "mktemp 败"
trap 'rm -rf "$TMP"' EXIT
ssh -o BatchMode=yes -o ConnectTimeout=10 "$SSH_TARGET" "cat '$CI_ROOT/reports/$SHA.report'" > "$TMP/report" 2>/dev/null \
  || local_check "report 拉取败"
ssh -o BatchMode=yes -o ConnectTimeout=10 "$SSH_TARGET" "cat '$CI_ROOT/reports/$SHA.log'" > "$TMP/run.log" 2>/dev/null \
  || local_check "日志拉取败"

OCI_CI_REPORT_FILE="$TMP/report" OCI_CI_LOG_FILE="$TMP/run.log" \
  bash "$SELF/oci_ci_accept.sh" "$SHA"
RC=$?
case "$RC" in
  0)  echo "✅ [oci-ci] 绿戳已落（后续推 origin 照常 scripts/git_push.sh）" >&2; exit 0 ;;
  1)  echo "❌ [oci-ci] OCI 红——不落戳；修后重推（无本盒重跑=失败迭代离岸）" >&2; exit 1 ;;
  *)  local_check "报告不可核/非定向谱（accept rc=$RC）" ;;
esac
