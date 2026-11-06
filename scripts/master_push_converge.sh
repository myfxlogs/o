#!/usr/bin/env bash
# master_push_converge.sh —— T-0403 件②：部署树共享 master 双席并推竞态
# 自动合流径（0397 f4c050a4 dangling 实证：远端先行时一席 reset --hard
# 甩落他席未推 commit——竞态面零防护）。
#
# 调点：scripts/hooks/pre-push 内 remote_ref=refs/heads/master 且
# repo_root=/opt/o 时喂参——被推 sha 不含 origin/master 即非快进：
# 就地 git rebase origin/master 自动合流（本地 commit 全数前移零丢失
# ——rebase 语义天然保双方件，非 reset 甩件径）。合流成本推作罢：绿戳
# 绑死 sha，rebase 后 sha 必变须 check_all.sh 重跑盖新戳再推。rebase
# 冲突/脏树即 --abort 回原状+红报人工合流（禁 git reset --hard
# origin/master 甩远端件——0397 案在案）。
#
# 用法：master_push_converge.sh <local_sha>
# rc：0=快进放行（远端已含于被推链/远端不可达让位原闸）
#     3=已合流须重戳再推 | 1=人工（冲突/脏树/当前支非 master）
set -uo pipefail
sha=${1:?用法: master_push_converge.sh <local_sha>}

git fetch origin master --quiet 2>/dev/null || {
  echo "WARN(pre-push-converge): git fetch 败——合流判不可为，让位原闸（非快进远端自拒）" >&2
  exit 0
}
git merge-base --is-ancestor origin/master "$sha" 2>/dev/null && exit 0

echo "ℹ [pre-push-converge] master 非快进（远端先行——双席并推竞态）——自动合流 git rebase origin/master" >&2
cur=$(git branch --show-current 2>/dev/null || true) # tolerate: detached=空 cur——下落非 master 支判显报人工非静默吞
[ "$cur" = "master" ] || {
  echo "❌ [pre-push-converge] 当前支非 master（${cur:-detached}）——不代合流，人工处置" >&2
  exit 1
}
# O_ALLOW_NONFF=1：rebase 重放 commit 天然非 ff 移 ref（旧 sha 不入新链
# 但语义零丢失）——reference-transaction non-FF 闸防的是甩件回退，此处
# 钦定哨兵放行合法合流位移（闸本体/O_ALLOW_* 面不动——T-0403 红线）。
if O_ALLOW_NONFF=1 git rebase origin/master >&2; then
  echo "❌ [pre-push-converge] master 已合流 origin/master（本地 commit 全数前移零丢失）——本推作罢：绿戳绑死 sha，scripts/check_all.sh 重跑盖新戳后再推" >&2
  exit 3
fi
git rebase --abort >/dev/null 2>&1 || true # tolerate: abort 败=本不在 rebase（脏树前置拒未起链）——rc1 红报人工兜底
echo "❌ [pre-push-converge] rebase 冲突/脏树——已回原状。人工合流：git pull --rebase origin master → 解冲 → check_all.sh 重戳再推（禁 git reset --hard origin/master——甩远端 commit=0397 dangling 案）" >&2
exit 1
