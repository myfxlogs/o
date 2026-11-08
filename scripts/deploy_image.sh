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
# ④ 完工投件：EXIT trap 统一发完工回执（feishu 直推——mbus 邮箱总线
#   2026-11-05 前后清档退役：/opt/meta/mbus.sh→archive+箱面尽撤，悬空径
#   每窗「回执缺」实证 devin-relay#8618+；T-0375 件③修通改活通知面）。
#   成功/失败/被闸三路同径；败静默不影响部署 rc；业主令 2026-10-31
#   「事事有应答」。
# ⑤ OCI PG-md 迁移随窗（T-0336）：arm64 migrate scp 上盒跑 -domain=md+
#   水位断言——PG-md 欠账不再靠手补（O_MD_MIGRATE_SKIP=1 跳步哨兵照红）。
set -euo pipefail
SG="${O_SG_HOST:-root@192.229.85.35}"
LOCK_DIR=/opt/o/.deploy-lock
cd "$(dirname "$0")/.."
die() { echo "❌ [deploy_image] $1" >&2; exit 1; }

# ── ⓪ 退场义务（EXIT trap 统一径：释锁 + 完工投件）──────────────
# 投件设闸前——出处闸拒也须报（业主令「事事有应答」）；败静默：mbus 缺席/投败
# 仅落本地 relay log，部署 rc 不受影响（件是回执非命门）。SG 侧不投（mbus 是
# dev 件）；DEPLOY 不在 SEATS 枚举，同 GHEVT 裁量借 MANAGE via 标记。
release_lock() {
  [ "$LOCK_HELD" = 1 ] || return 0
  # 只释自己锁——holder token 对不上（抢锁后原主 trap 等情形）不动
  ssh "$SG" "grep -q '^token=$TOKEN\$' $LOCK_DIR/holder 2>/dev/null && rm -rf $LOCK_DIR" \
    >/dev/null 2>&1 || true
}
deploy_event() {
  local rc=$1 sha=${SHA:-$(git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)}
  # 回执径（T-0375 件③修通）：mbus 邮箱总线清档退役（脚本→archive、
  # /opt/meta/mail 箱面尽撤——复活即逆清档令），改 feishu-notify 直推
  # （同 report.sh 自报径的活通知面；实活验证 rc=0）。
  echo "[deploy_image] 完工回执 deploy $sha rc=$rc image=o-core:$sha" >>"$HOME/devin-relay.log" 2>/dev/null || true # tolerate: relay 留痕行 best-effort——回执本体 feishu 推送败已显报（下行回执缺 echo）
  timeout 15 /opt/meta/scripts/feishu-notify.sh \
    "[部署回执] deploy $sha rc=$rc image=o-core:$sha" >>"$HOME/devin-relay.log" 2>&1 \
    || echo "[deploy_image] 完工投件败（回执缺，不影响部署 rc=$rc）" >&2
  # 驾驶舱 v4 完结线（设计稿钉：完结=部署上线）：部署事件落 ops_event
  # （event=部署,ticket=''——管道按部署 ts≥票末事件退场判，fin_0092 触发器
  # 同拍 NOTIFY→前端行秒级退场）。凭据取径同 report.sh（FIN_PG_DSN 自
  # /opt/o-research/env）；败静默不影响部署 rc。
  local dsn
  dsn=$(grep '^FIN_PG_DSN=' /opt/o-research/env 2>/dev/null | cut -d= -f2- | tr -d '"')
  if [ -n "$dsn" ]; then
    timeout 15 psql "$dsn" -Atc \
      "INSERT INTO ops_event(seat,event,ticket,note) VALUES('DEPLOY','部署','','deploy $sha rc=$rc image=o-core:$sha')" \
      >/dev/null 2>&1 \
      || echo "[deploy_image] ops_event 部署事件落簿败（管道完结线缺本拍，不影响部署 rc=$rc）" >&2
  fi
}
LOCK_HELD=0
trap 'deploy_event $?; release_lock' EXIT

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

# ── ⑤ OCI PG-md 迁移随窗步（T-0336）──────────────────────────
# 病灶（T-0333 红②③在案）：本径原只船 SG o-core——PG-md 迁移从不随窗，
# projector boot 迁移是 OCI 旧件不随 master 进，md_1087+bridge-reverse
# 五表欠账无人补。此步随锁同窗：arm64 migrate 交叉建→scp 上盒→env 件取
# MD_PG_DSN 跑 -domain=md→水位断言 live≥登记表（「已合并≠已执行」机检，
# 同 check_schema_live 判据）。O_MD_MIGRATE_SKIP=1 跳步（OCI 失联/演习——
# 跳步即欠账，哨兵照样报红）。
if [ "${O_MD_MIGRATE_SKIP:-0}" = 1 ]; then
  echo "⚠️  [deploy_image] O_MD_MIGRATE_SKIP=1——PG-md 迁移步跳（欠账哨兵照红）" >&2
else
  MD_HOST="${O_MD_HOST:-ubuntu@149.118.58.98}"
  reg_md=$(grep -oE '\{[0-9]+, "md_[0-9]+' internal/store/pgstore/migrate_table.go \
    | grep -oE '^\{[0-9]+' | tr -d '{' | sort -n | tail -1)
  echo "[deploy_image] OCI PG-md 迁移随窗——host=$MD_HOST 登记表水位=$reg_md"
  GOARCH=arm64 GOOS=linux go build -o "/tmp/o-migrate-md.$$" ./cmd/migrate
  scp -q -o BatchMode=yes -o ConnectTimeout=10 "/tmp/o-migrate-md.$$" "$MD_HOST:/tmp/o-migrate-md"
  rm -f "/tmp/o-migrate-md.$$"
  ssh -o BatchMode=yes -o ConnectTimeout=10 "$MD_HOST" '
    set -e
    MD_PG_DSN=$(grep "^MD_PG_DSN=" ~/o-projector.env | cut -d= -f2- | tr -d "\"")
    [ -n "$MD_PG_DSN" ] || { echo "o-projector.env MD_PG_DSN 缺" >&2; exit 1; }
    PG_DSN="$MD_PG_DSN" /tmp/o-migrate-md -domain=md'
  live_md=$(ssh -o BatchMode=yes -o ConnectTimeout=10 "$MD_HOST" \
    "sudo docker exec o-postgres-md psql -U o -d o -tc 'SELECT COALESCE(max(version),0) FROM schema_migrations'" | tr -d ' ')
  [ "${live_md:-0}" -ge "$reg_md" ] \
    || die "PG-md 水位断言红——live=${live_md:-<空>} < 登记表=$reg_md"
  echo "[deploy_image] PG-md 水位 live=$live_md ≥ 登记表=$reg_md"
fi

# ── ⑥ 本机构建面 tag 保留窗（T-0466 件②）────────────────────
# 一 commit 一 tag 永不删=镜像堆单调增长病灶（66 tag≈5G 实证）；
# 留近 5 个 sha tag+latest 保回滚径。保洁件尽力——败不翻部署结果。
bash scripts/meta/image_tag_gc.sh || echo "⚠️ [deploy_image] image_tag_gc 败（不阻断部署收口）" >&2 # tolerate: 保洁尽力件——tag 残留下轮 deploy 再收，部署成败不绑保洁

echo "[deploy_image] done→$SG（部署锁 trap 已释）"
