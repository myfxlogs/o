#!/usr/bin/env bash
# check_seat_writes.sh —— 席位写域机械闸（T-0145 越权实证立法）：
# 无头决策席会话产出对 diff 白名单——自写自合实现码硬拒。
# 判据:
#   直 commit（标 [角色:决策] 或 author=claude-glm@local 无头签名）:
#     diff 件集 ⊆ {docs/, skills/, *.md} ——判词/返修件/卡面=决策席白名单
#   merge commit: 支侧 commit 若全为决策席自产且触非白名单=自写自合硬拒;
#     支侧含他席痕（施工/外审/COS 任一）=放行
# 扫描域: origin/master..HEAD（pre-push 面）;CHECK_SEAT_RANGE 覆盖可测
# 豁免: O_SEAT_WRITE_ALLOW=1（外审裁过的人工放行——同 O_ALLOW_NONFF 先例,
#   WARN 留痕不静默;在档违例件经外审裁定后推径）
set -euo pipefail
if [ "${O_SEAT_WRITE_ALLOW:-0}" = "1" ]; then
  echo "WARN: check_seat_writes 人工豁免放行（O_SEAT_WRITE_ALLOW=1——外审裁后推径，违例标仍留档）" >&2
  exit 0
fi
cd "$(git rev-parse --show-toplevel)"
RANGE=${CHECK_SEAT_RANGE:-origin/master..HEAD}
WL='^(docs/|skills/|[^/]*\.md$)'
viol=0
while read -r sha; do
  subj=$(git log -1 --format=%s "$sha")
  author=$(git log -1 --format=%ae "$sha")
  case "$subj|$author" in *"角色:决策"*|*"claude-glm@local"*) ;; *) continue;; esac
  bad=$(git diff-tree --no-commit-id --name-only -r "$sha" | grep -vE "$WL" || true)
  [ -z "$bad" ] && continue
  echo "FAIL: 决策席直commit越白名单 $sha ${subj:0:40}:"; echo "$bad" | sed 's/^/      /'
  viol=1
done < <(git rev-list "$RANGE" --no-merges 2>/dev/null)
while read -r sha; do
  # 支侧 commit 全谱: 若每件都是决策席自产→查其触件;含他席即放行
  side=$(git rev-list "$sha" --not "$sha^1" 2>/dev/null || true)
  [ -z "$side" ] && continue
  allmine=1; nonwl=
  for s in $side; do
    subj=$(git log -1 --format=%s "$s"); author=$(git log -1 --format=%ae "$s")
    case "$subj|$author" in *"角色:决策"*|*"claude-glm@local"*)
      nonwl="$nonwl$(git diff-tree --no-commit-id --name-only -r "$s" | grep -vE "$WL" || true)" ;;
    *) allmine=0;; esac
  done
  [ "$allmine" = 1 ] && [ -n "$nonwl" ] && {
    echo "FAIL: 决策席自写自合越权 merge $sha:"; echo "$nonwl" | sort -u | sed 's/^/      /'; viol=1; }
done < <(git rev-list "$RANGE" --merges 2>/dev/null)
[ "$viol" = 0 ] && echo "OK: check_seat_writes（决策席写域白名单——零越权件）" || { echo "FAIL: check_seat_writes——决策席越权实现在档"; exit 1; }
