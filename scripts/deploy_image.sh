#!/usr/bin/env bash
# deploy_image.sh — 本机构建镜像船运 SG（业主令 2026-10-24：SG 5Mbps 构建链慢，
# 镜像传输取代远端编译）。流：出处闸 → SG 部署锁 → 本机 build → save|gzip →
# SG load → compose up（无 --build）→ 产地尾拍 + 健康闸。
# T-0217/D-0035：部署唯一合法径（直 docker compose 改产=绕锁违例）——
# ① 出处闸：脏树拒 / HEAD 不在 origin/master 拒（O_DEPLOY_ALLOW_UNPUSHED=1 放行
#   回滚/热补），堵陈码静默盖新码；
# ② SG 部署锁：ssh mkdir /opt/o/.deploy-lock 原子互斥（dev→SG 实证 rc=1 撞拒），
#   持锁写 holder=席位@主机+sha+ISO ts，陈锁 30min 或 O_DEPLOY_FORCE=1 可抢
#   （抢前 WARN），EXIT trap 释锁；
# ③ SHA tag 留痕：o-core:<sha12> 主 tag + o-core:latest 同层别名双 ref 齐运，
#   compose image: ${O_CORE_IMAGE:-o-core:latest} 环境可覆写，容器
#   {{.Config.Image}} 自报所跑源码 commit。
set -euo pipefail
SG="${O_SG_HOST:-root@192.229.85.35}"
LOCK_DIR=/opt/o/.deploy-lock
cd "$(dirname "$0")/.."
die() { echo "❌ [deploy_image] $1" >&2; exit 1; }

# ── ① 出处闸 ────────────────────────────────────────────────
[ -z "$(git status --porcelain)" ] \
  || die "工作树脏（git status 非空）——commit/stash 后再部署"
git fetch origin master --quiet 2>/dev/null \
  || echo "⚠️  [deploy_image] git fetch 失败——按本地 origin/master 快照判"
if ! git merge-base --is-ancestor HEAD origin/master; then
  if [ "${O_DEPLOY_ALLOW_UNPUSHED:-0}" != 1 ]; then
    echo "❌ [deploy_image] HEAD 不在 origin/master——未并码禁部署。待推 commits:" >&2
    git log --oneline origin/master..HEAD | head -10 >&2
    die "回滚/热补显式放行：O_DEPLOY_ALLOW_UNPUSHED=1"
  fi
  echo "⚠️  [deploy_image] O_DEPLOY_ALLOW_UNPUSHED=1——放行未并 HEAD（回滚/热补径）"
fi

SHA=$(git rev-parse --short=12 HEAD)
TAG="o-core:$SHA"
SEAT="${O_DEPLOY_SEAT:-$(whoami)}@$(hostname -s 2>/dev/null || hostname)"
TOKEN="$SEAT-$$-$(date +%s)"

# ── ② SG 部署锁 ─────────────────────────────────────────────
LOCK_HELD=0
release_lock() {
  [ "$LOCK_HELD" = 1 ] || return 0
  # 只释自己锁——holder token 对不上（抢锁后原主 trap 等情形）不动
  ssh "$SG" "grep -q '^token=$TOKEN\$' $LOCK_DIR/holder 2>/dev/null && rm -rf $LOCK_DIR" \
    >/dev/null 2>&1 || true
}
trap release_lock EXIT

if ssh "$SG" "mkdir $LOCK_DIR" 2>/dev/null; then
  LOCK_HELD=1
else
  holder=$(ssh "$SG" "cat $LOCK_DIR/holder 2>/dev/null" || true)
  age=$(ssh "$SG" "echo \$(( \$(date +%s) - \$(stat -c %Y $LOCK_DIR 2>/dev/null || echo 0) ))")
  if [ "${O_DEPLOY_FORCE:-0}" = 1 ] || [ "${age:-0}" -gt 1800 ]; then
    echo "⚠️  [deploy_image] WARN 抢部署锁（force=${O_DEPLOY_FORCE:-0} age=${age}s/1800s）holder: ${holder:-<空>}" >&2
    ssh "$SG" "rm -rf $LOCK_DIR && mkdir $LOCK_DIR" \
      || die "抢锁失败——他席恰持锁，下轮再试"
    LOCK_HELD=1
  else
    die "部署锁占用（age=${age}s/1800s 未陈）holder: ${holder:-<空>} ——待释或 O_DEPLOY_FORCE=1 抢"
  fi
fi
ssh "$SG" "printf 'seat=%s\nsha=%s\nts=%s\ntoken=%s\n' '$SEAT' '$SHA' \"\$(date -Is)\" '$TOKEN' > $LOCK_DIR/holder"
echo "[deploy_image] 锁持 $LOCK_DIR holder=$SEAT sha=$SHA"

# ── ③ SHA tag 留痕 ──────────────────────────────────────────
echo "[deploy_image] 本机构建 $TAG（+latest 同层别名）"
docker build -q -f Dockerfile.core -t "$TAG" -t o-core:latest .
echo "[deploy_image] 船运→SG（gzip 流，双 ref 齐运）"
docker save "$TAG" o-core:latest | gzip -1 | ssh "$SG" 'gunzip | docker load'
ssh "$SG" "cd /opt/o && O_CORE_IMAGE=$TAG docker compose -f docker-compose.yml -f docker-compose.sg.yml up -d o-core"
echo "[deploy_image] 产地尾拍——容器自报镜像："
ssh "$SG" "docker inspect o-core --format '{{.Config.Image}}'"

# 健康闸：等 compose healthcheck 转 healthy（验收锚——部署后 o-core 健康，
# 不绿禁收工；none=无 healthcheck 定义按过）。
for i in $(seq 1 18); do
  st=$(ssh "$SG" "docker inspect o-core --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}'")
  if [ "$st" = healthy ] || [ "$st" = none ]; then break; fi
  [ "$i" = 18 ] && die "o-core 部署后 90s 未转 healthy（现态=$st）——docker logs o-core 查因"
  sleep 5
done
echo "[deploy_image] o-core 健康态=$st"

# T-0153 船运后镜像自剪——留 72h 回滚余量（SG 盘 76% 实证病灶面治）。
ssh "$SG" 'docker image prune -f --filter "until=72h" && echo "[deploy_image] prune≤72h done"'
echo "[deploy_image] done→$SG（部署锁 trap 已释）"
