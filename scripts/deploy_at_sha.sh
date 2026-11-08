#!/usr/bin/env bash
# deploy_at_sha.sh —— T-0467 件②部署半：按「已绿 sha」发镜像——
# 「镜面任务完成后统一部署」形制的部署侧：临时 worktree 检出目标
# sha→bootstrap 补 gen/dist→跑该 sha 自带 scripts/deploy_image.sh
# （出处闸/SG 部署锁/SHA tag/健康闸/PG-md 随窗步全继承原样——
# 部署物=该 sha 树逐字节，工具链亦取该 sha 版非部署树版）。
#
# 前置断言（fail-closed）：
#   ①sha 可解为 commit；②持绿戳（.git/check-green-<sha> 在案——
#     「已绿」语义钉：无戳=未验证物禁发；戳龄窗归推面管，部署面
#     只认戳在不在——发窗滞后>6h 不吊销已推事实）；
#   ③sha 是 origin/master 祖先（已并入远端——未推/未合件禁发，
#     与 deploy_image 出处闸「HEAD∈origin/master」同向互证）。
#
# 用法：scripts/deploy_at_sha.sh <sha> [deploy_image.sh 其余参数]
# 环境覆写（桩测）：O_DEPLOY_AT_REPO（默认 /opt/o——对象库/戳域
#   所在树）、O_DEPLOY_AT_SKIP_BOOTSTRAP=1（跳 bootstrap——桩测/
#   gen 已备场景）。
set -euo pipefail
SHA_ARG=${1:?用法: deploy_at_sha.sh <sha>}
shift || true # tolerate: 无参尾=合法——deploy_image.sh 无必收参数
REPO=${O_DEPLOY_AT_REPO:-/opt/o}
cd "$REPO" || { echo "❌ [deploy_at_sha] repo 不可达 $REPO" >&2; exit 1; }
# GW-001 A3 戳位归一：戳读 git-common-dir——与 check_all/mirror_push/
# pre-push 同域（O_DEPLOY_AT_REPO 指 linked worktree 时 --git-dir 私有
# 读不到公共戳面）。
GITDIR=$(git rev-parse --git-common-dir)

die() { echo "❌ [deploy_at_sha] $1" >&2; exit 1; }

SHA=$(git rev-parse --verify "$SHA_ARG^{commit}" 2>/dev/null) \
  || die "sha 不可解为 commit: $SHA_ARG"
git fetch origin master --quiet 2>/dev/null \
  || echo "⚠️ [deploy_at_sha] git fetch 败——按本地 origin/master 快照判" >&2
[ -f "$GITDIR/check-green-$SHA" ] \
  || die "$SHA 无绿戳——未验证物禁发（先 check_all 绿+mirror_push 推）"
git merge-base --is-ancestor "$SHA" origin/master 2>/dev/null \
  || die "$SHA 未并入 origin/master——先推后发（mirror_push 径）"

WT=$(mktemp -d)/deploy-wt
cleanup() { git worktree remove --force "$WT" >/dev/null 2>&1 || true; } # tolerate: EXIT 清场尽力件——临时 wt 残留=下次 add 自清/git worktree prune 兜底，清场败不翻部署结果
trap cleanup EXIT
git worktree add --detach "$WT" "$SHA" >/dev/null \
  || die "worktree add 败 $WT @ ${SHA:0:12}"
if [ "${O_DEPLOY_AT_SKIP_BOOTSTRAP:-0}" != 1 ]; then
  bash "$REPO/scripts/bootstrap_worktree.sh" "$WT" >&2 \
    || die "bootstrap_worktree 败——gen/dist 缺则构建必炸"
fi
echo "▶ [deploy_at_sha] 按 ${SHA:0:12}（绿戳在案∧已并 master）发镜像——临时树 $WT"
exec bash "$WT/scripts/deploy_image.sh" "$@"
