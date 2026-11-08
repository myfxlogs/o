#!/usr/bin/env bash
# mirror_push.sh —— T-0467 件②镜面推：部署树取「最近绿戳且可快进」
# sha 推 origin master——推已绿 sha 非 HEAD（跑步机根修：绿戳绑
# 被测 sha 非 HEAD，check_all 跑期 HEAD 移动不废戳；跑期新落
# commit 归下趟镜面，不冻不废——业主令「推已绿 sha」钦定面①）。
#
# 选择口径（裁填钉死）：戳面 mtime 新→旧遍历，首个四断言全过者
#   ①戳龄 ≤21600s（与 pre-push stamp_ok 同 6h 窗）；
#   ②sha 未被 origin/master 含（已合入=无新件跳过）；
#   ③origin/master 是 sha 祖先（ff 可推——远端已前进/分叉=跳过，
#     非快进仍归 pre-push 合流闸管不在此抢）；
#   ④GW-001 A3 HEAD 包含闸：sha 是本树 HEAD 祖先——分支戳（绿验
#     在非部署线 commit 上）不推 master，防 worktree/支线绿戳
#     越线污染镜面。
# 全灭=镜面空窗合法 exit 0 零动作（不为推而推）。
#
# 并存关系（裁填钉死=包非替）：git_push.sh 重试包装复用；
# master_push_converge.sh pre-push 合流闸原位不动——镜面推的
# local_sha 恒为 origin/master 后代，C0 合流闸判定天然直通。
#
# 环境覆写（桩测/对拍面）：O_MIRROR_REPO（默认 /opt/o）、
# O_MIRROR_PUSH_SH（默认 $REPO/scripts/git_push.sh）、
# O_MIRROR_NOW（秒戳——戳龄可重放）。
set -uo pipefail
REPO=${O_MIRROR_REPO:-/opt/o}
PUSH_SH=${O_MIRROR_PUSH_SH:-"$REPO/scripts/git_push.sh"}
NOW=${O_MIRROR_NOW:-$(date +%s)}
cd "$REPO" 2>/dev/null || { echo "❌ [mirror_push] repo 不可达 $REPO" >&2; exit 1; }
GITDIR=$(git rev-parse --git-common-dir) || { echo "❌ [mirror_push] 非 git 树 $REPO" >&2; exit 1; }

# 远端快照对齐（败=按本地 origin/master 判——推时非快进远端自拒兜底，
# 与 deploy_image.sh fetch 败降级同式）。
git fetch origin master --quiet 2>/dev/null \
  || echo "⚠️ [mirror_push] git fetch 败——按本地 origin/master 快照判（推时远端自拒兜底）" >&2
git rev-parse --verify -q origin/master >/dev/null 2>&1 || {
  echo "ℹ [mirror_push] 无 origin/master 基线——镜面空窗"
  exit 0
}

pushed=0
while IFS= read -r stamp; do
  [ -f "$stamp" ] || continue
  case "$stamp" in *.mode) continue ;; esac
  sha=${stamp##*check-green-}
  git cat-file -e "$sha^{commit}" 2>/dev/null || continue
  age=$(( NOW - $(date -r "$stamp" +%s) ))
  [ "$age" -le 21600 ] || continue
  git merge-base --is-ancestor "$sha" origin/master 2>/dev/null && continue
  git merge-base --is-ancestor origin/master "$sha" 2>/dev/null || continue
  # GW-001 A3 ④HEAD 包含闸：绿戳 sha 须在本树 HEAD 祖先链——支线/
  # worktree 戳（sha 不在部署线历史）即便新鲜也不推 master。
  git merge-base --is-ancestor "$sha" HEAD 2>/dev/null || continue
  echo "▶ [mirror_push] 绿戳候选 sha=${sha:0:12}（戳龄 ${age}s）→ origin ${sha}:master"
  if "$PUSH_SH" origin "$sha:master"; then
    pushed=1
    break
  fi
  echo "⚠️ [mirror_push] ${sha:0:12} 推败——落下一候选" >&2
done < <(ls -t "$GITDIR"/check-green-* 2>/dev/null)

if [ "$pushed" = 1 ]; then
  echo "✅ [mirror_push] 已绿 sha 推送毕（HEAD 无关——跑期新 commit 归下趟）"
else
  echo "ℹ [mirror_push] 无可推绿戳（三断言全灭=镜面空窗合法）"
fi
exit 0
