#!/usr/bin/env bash
# deploy_window_due.sh —— 部署窗触发律机读壳（T-0398，业主令「走1吧」：
# 攒批有标注无触发律，批积 11 件不发窗=触发律缺位实证）。DM 每班查一次
# 不靠感觉。
#
# 三态输出（首行 state= 机读，余行 reason=）+exit 分态：
#   0  应发  ——判据①批非空 ∧ 判据②无挡件；或兜底节拍压过（批积首件≥24h
#             未发→无论条件②发窗并升件报，ops_event 升件行
#             DEPLOY-WINDOW-OVERRUN 前缀同窗去重）
#   10 批空  ——origin/master↔生产部署 SHA 间镜像面 diff 空
#   11 有挡  ——批非空但挡件未清（在飞 o-worker 码件>0 或 待裁/收口待审面
#             含码件/迁移件>0；待裁面不可知（dsn缺/查询败）fail-closed
#             按有挡计——窗误压不滥发）
#   78 环境缺——部署 SHA 不可得（ssh inspect 与 ops_event 部署行双源皆败）
#   1  查询败——git 面推导失败
#
# 判据原文（票面钦定语义，禁读票面状态行当输入——git diff+ops_event/
# systemd 实态为准）：
#   ①批非空: origin/master 与生产部署 SHA 间 diff 含镜像面路径非空。
#     镜像面钦定清单=票面正列（cmd/internal/proto/web/migrations/
#     Dockerfile*）+入镜像实据（go.mod/go.sum/buf*.yaml/buf*.yml/
#     docker-compose*/.dockerignore——compose 由 SG /opt/o 检出随部署窗
#     生效）；非镜像面=docs/**,skills/**,*.md,scripts/**（meta/部署壳
#     检出即生效不入镜像——collab §11 同口径）,.github/**,.devin/**,
#     tools/**（宿主 go run 不进 o-core 二进制）,deploy/**（systemd/env
#     件宿主生效）。未列名路径→按镜像面计（宁滥发窗勿漏批）。
#   ②无挡件: 在飞 o-worker 码件=0 ∧ 待裁/收口待审面含码件/迁移件=0
#     （docs/研究件不挡窗）。码件判定=票支 diff∪wt 脏面碰镜像面；
#     支树皆不可定→保守按码件计；已批未派件无支=未动工不挡。
#   兜底节拍: 批积首件（最老镜像面 commit）起 ≥24h 未发→无论条件②
#     发窗并升件报。
#
# 环境覆写（测试/对拍面）：O_DEPLOY_WINDOW_REPO（仓树，默认 /opt/o）、
# O_MASTER_REF（默认 origin/master）、O_DEPLOYED_SHA（跳双源直取）、
# O_SG_HOST（默认 root@192.229.85.35）、DM_ENV_FILE、O_DM_QUEUE_SH、
# O_DEPLOY_WINDOW_NOW（秒戳——兜底节拍可重放）。
set -u
REPO=${O_DEPLOY_WINDOW_REPO:-/opt/o}
MASTER_REF=${O_MASTER_REF:-origin/master}
SG=${O_SG_HOST:-root@192.229.85.35}
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
DM_QUEUE_SH=${O_DM_QUEUE_SH:-"$SCRIPT_DIR/dm_queue.sh"}
NOW=${O_DEPLOY_WINDOW_NOW:-$(date +%s)}

emit() { printf 'state=%s\n' "$1"; }
reason() { printf 'reason=%s\n' "$1"; }

# ── 镜像面判定：非镜像白名单补集（未列名=镜像面，漏批方向保守）───
is_image_path() {
  case "$1" in
    docs/*|skills/*|*.md|scripts/*|.github/*|.devin/*|tools/*|deploy/*) return 1 ;;
    *) return 0 ;;
  esac
}
# stdin=path 集 → 任一碰镜像面 rc=0
touches_image() {
  local p
  while IFS= read -r p; do
    [ -n "$p" ] || continue
    p=${p##* -> } # rename 形取新名
    is_image_path "$p" && return 0
  done
  return 1
}
commit_touches_image() { # $1=commit（merge 件 -m 逐亲 diff）
  git -C "$REPO" diff-tree -m --no-commit-id -r --name-only "$1" 2>/dev/null \
    | touches_image
}
# 票支全枚举（本地→远端，同票多支并查——head-1 挑净支漏判案）
ticket_refs() { # $1=T-NNNN → 支名集
  local num=${1#T-}
  git -C "$REPO" branch --list "t/$num" "t/$num-*" --format='%(refname:short)'
  git -C "$REPO" branch -r --list "origin/t/$num" "origin/t/$num-*" \
    --format='%(refname:short)'
}
# 票号→码件判定：支 diff∪wt 脏面碰镜像=码件；支树皆不可定=保守码件
ticket_is_code() { # $1=T-NNNN → rc=0 码件（或不可定保守计）
  local num=${1#T-} ref saw=0
  while read -r ref; do
    [ -n "$ref" ] || continue
    saw=1
    git -C "$REPO" diff --name-only "$MASTER_REF...$ref" 2>/dev/null | touches_image \
      && return 0
  done < <(ticket_refs "$1")
  local wtd=/opt/o-wt/t$num
  if [ -d "$wtd" ] && [ -e "$wtd/.git" ]; then
    saw=1
    # wt 未提交脏面+wt 内未推 commit 面并查
    { git -C "$wtd" status --porcelain 2>/dev/null | sed 's/^...//'
      git -C "$wtd" diff --name-only "$MASTER_REF...HEAD" 2>/dev/null
    } | touches_image && return 0
  fi
  [ "$saw" -eq 1 ] && return 1 # 支/树可定且镜像面零碰=非码件
  return 0 # 支树皆不可定→保守码件
}

# ── 部署 SHA 双源：SG docker inspect（权威实拍）→ ops_event 末次
# rc=0 部署行（ssh 断退化径）──────────────────────────────────
DEPLOYED=${O_DEPLOYED_SHA:-}
DEPLOYED_SRC=env
if [ -z "$DEPLOYED" ]; then
  img=$(ssh -o BatchMode=yes -o ConnectTimeout=10 "$SG" \
    "docker inspect o-core --format '{{.Config.Image}}'" 2>/dev/null || true) # tolerate: ssh 断退化 ops_event 源——双源皆败下判 exit 78 环境缺显报非静默
  case "$img" in
    o-core:*) DEPLOYED=${img#o-core:}; DEPLOYED_SRC=sg-inspect ;;
  esac
fi
if [ -z "$DEPLOYED" ] && [ -n "$FIN_DSN" ]; then
  DEPLOYED=$(psql "$FIN_DSN" -qAt -c "SELECT substring(note from 'deploy ([0-9a-f]+)')
    FROM ops_event WHERE event='部署' AND note LIKE 'deploy % rc=0 %'
    ORDER BY ts DESC LIMIT 1" 2>/dev/null || true) # tolerate: 退化源查询败=DEPLOYED 空→exit 78 环境缺显报非静默
  [ -n "$DEPLOYED" ] && DEPLOYED_SRC=ops-event
fi
if [ -z "$DEPLOYED" ]; then
  emit 环境缺; reason "部署 SHA 不可得（ssh inspect+ops_event 部署行双源皆败）——批界不可定"
  exit 78
fi
if ! git -C "$REPO" rev-parse --verify --quiet "$DEPLOYED^{commit}" >/dev/null; then
  emit 环境缺; reason "部署 SHA=$DEPLOYED(src=$DEPLOYED_SRC) 本地不可解——fetch 欠账或镜像 tag 非本仓 commit"
  exit 78
fi
git -C "$REPO" fetch origin master --quiet 2>/dev/null || true # tolerate: 快照陈化滑行（deploy_image 同式——fetch 败按本地 origin/master 判，批界偏差由 SG SHA 实锚兜）

# ── 判据①批非空：deployed..master diff 镜像面集 ───────────────
RANGE_FILES=$(git -C "$REPO" diff --name-only "$DEPLOYED..$MASTER_REF" 2>/dev/null)
GRC=$?
[ "$GRC" -eq 0 ] || { emit 查询败; reason "git diff $DEPLOYED..$MASTER_REF 败 rc=$GRC"; exit 1; }
IMG_PATHS=$(printf '%s\n' "$RANGE_FILES" | while IFS= read -r p; do
  [ -n "$p" ] && is_image_path "$p" && printf '%s\n' "$p"
done)
if [ -z "$IMG_PATHS" ]; then
  emit 批空
  reason "diff $DEPLOYED(src=$DEPLOYED_SRC)..$MASTER_REF 镜像面空——docs-only/零差批不压窗"
  exit 10
fi

# 批件集=range 内镜像面 commit 子集（票号集自 subject 提 T-\d{4}；
# 兜底节拍锚=批积首件=最老镜像面 commit ts）
IMG_COMMITS=$(git -C "$REPO" log --format='%H %ct %s' "$DEPLOYED..$MASTER_REF" \
  | while read -r c ct cs; do
      commit_touches_image "$c" && printf '%s %s %s\n' "$c" "$ct" "$cs"
    done)
NBATCH=$(printf '%s\n' "$IMG_COMMITS" | awk 'NF {n++} END {print n+0}')
EARLIEST=$(printf '%s\n' "$IMG_COMMITS" | awk 'NF && (min=="" || $2<min) {min=$2} END {print min+0}')
TICKETS=$(printf '%s\n' "$IMG_COMMITS" | grep -oE 'T-[0-9]{4}' | sort -u | paste -sd, -)
AGE=$(( NOW - EARLIEST ))
reason "批非空 commits=$NBATCH tickets=${TICKETS:-无号} deployed=$DEPLOYED(src=$DEPLOYED_SRC) 批首件龄=${AGE}s"

# ── 判据②挡件集 ──────────────────────────────────────────────
BLOCKERS=""
# ②a 在飞 o-worker 码件（systemd 实态）
while read -r unit _rest; do
  case "$unit" in o-worker-T-*.service) ;; *) continue ;; esac
  tid=${unit#o-worker-}; tid=${tid%.service}
  ticket_is_code "$tid" && BLOCKERS="$BLOCKERS 在飞:$tid"
done < <(systemctl --user list-units 'o-worker-*' --state=active,activating \
  --no-legend 2>/dev/null)
# ②b 待裁/收口待审面含码件（dm_queue.sh 同源推导；dsn缺/查询败=
# 面不可知 fail-closed 有挡）。已批未派件无支=未动工不挡（挡它
# =永锁窗），有支再判 diff。
if [ -n "$FIN_DSN" ]; then
  QRC=0; PENDING=$("$DM_QUEUE_SH" 2>/dev/null) || QRC=$?
  if [ "$QRC" -ne 0 ]; then
    BLOCKERS="$BLOCKERS 待裁面查询败rc=$QRC"
  else
    while read -r tid; do
      [ -n "$tid" ] || continue
      code=0; saw=0
      while read -r ref; do
        [ -n "$ref" ] || continue
        saw=1
        git -C "$REPO" diff --name-only "$MASTER_REF...$ref" 2>/dev/null | touches_image \
          && { code=1; break; }
      done < <(ticket_refs "$tid")
      # 已批未派件无支=未动工不挡（挡它=永锁窗），有支再判 diff
      [ "$saw" -eq 1 ] && [ "$code" -eq 1 ] && BLOCKERS="$BLOCKERS 待裁:$tid"
    done <<< "$PENDING"
  fi
else
  BLOCKERS="$BLOCKERS 待裁面不可知(dsn缺)"
fi

# ── 兜底节拍：批首件龄 ≥24h → 无论条件②发窗并升件报 ────────────
BL=${BLOCKERS# }; BL=${BL:-无}
if [ "$AGE" -ge 86400 ]; then
  emit 应发
  reason "兜底节拍压过——批积首件龄 ${AGE}s≥86400s，无论条件②发窗（挡件=$BL）"
  reason "batch=${TICKETS:-无号} deployed=$DEPLOYED"
  "${OPS_ESC_SH:-$SCRIPT_DIR/ops_esc.sh}" META-DM "" "" "" \
    "DEPLOY-WINDOW-OVERRUN" \
    "批积首件龄${AGE}s≥24h——无论挡件发窗 tickets=${TICKETS:-无号} blockers=$BL" \
    || true # tolerate: 升件末环败静默退（发窗判据不受影响）
  exit 0
fi

if [ "$BL" != "无" ]; then
  emit 有挡
  reason "挡件未清: $BL——批待清窗压（docs/研究件不挡）"
  exit 11
fi

emit 应发
reason "判据两条件齐——批非空(tickets=${TICKETS:-无号})∧挡件零→当班发窗，deploy_image.sh 单径"
exit 0
