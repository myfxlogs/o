#!/usr/bin/env bash
# deploy_window_due.sh —— 部署窗触发律机读壳（T-0398，业主令「走1吧」：
# 攒批有标注无触发律，批积 11 件不发窗=触发律缺位实证）。DM 每班查一次
# 不靠感觉。
#
# 三态输出（首行 state= 机读，次行 faces= 机读发窗面集——逗分隔
# image|oci 子集，批空=空串；余行 reason=）+exit 分态：
#   0  应发  ——判据①批非空 ∧ 判据②无挡件；或兜底节拍压过（批积首件≥24h
#             未发→无论条件②发窗并升件报，ops_event 升件行
#             DEPLOY-WINDOW-OVERRUN 前缀同窗去重）
#   10 批空  ——origin/master↔两面各自部署基线间 diff 俱空
#   11 有挡  ——批非空但挡件未清（在飞 o-worker 码件>0 或 待裁/收口待审面
#             含码件/迁移件>0；待裁面不可知（dsn缺/查询败）fail-closed
#             按有挡计——窗误压不滥发）
#   78 环境缺——镜像面部署 SHA 不可得（ssh inspect 与 ops_event 部署行
#             双源皆败）
#   1  查询败——git 面推导失败
#
# 判据原文（票面钦定语义，禁读票面状态行当输入——git diff+ops_event/
# systemd 实态为准）：
#   ①批非空（T-0425 两半部署制——批面二分，任一面非空=批非空）:
#     镜像面: origin/master 与生产部署 SHA 间 diff 含镜像面路径非空→
#       发 deploy_image.sh。镜像面钦定清单=票面正列（cmd/internal/
#       proto/web/migrations/Dockerfile*）+入镜像实据（go.mod/go.sum/
#       buf*.yaml/buf*.yml/docker-compose*/.dockerignore——compose 由
#       SG /opt/o 检出随部署窗生效）；非镜像面=docs/**,skills/**,*.md,
#       scripts/**（meta/部署壳检出即生效不入镜像——collab §11 同口径）,
#       .github/**,.devin/**,tools/**（宿主 go run 不进 o-core 二进制）,
#       deploy/**（systemd/env 件宿主生效）。未列名路径→按镜像面计
#       （宁滥发窗勿漏批）。
#     OCI二进制面: origin/master 与 OCI 盒上三臂部署 SHA 间 diff 碰
#       OCI 构建源→同窗发 deploy_oci.sh。面判式=镜像面∧¬镜像专属面
#       （镜像专属否列钉 deploy/deploy-faces.tsv image_only 类——
#       web/Docker 件/非 OCI cmd 主件；未列名=面不可判保守按双发，
#       件缺=否列空全面双发）。OCI 基线三源：env→盒上三臂 $ep.sha
#       最老件→ops_event deploy-oci 行；三源皆败/谱系岔=面不可判
#       →OCI 面按非空保守计（发 deploy_oci 自检径兜——不可判不静默）。
#   ②无挡件: 在飞 o-worker 码件=0 ∧ 待裁/收口待审面含码件/迁移件=0
#     （docs/研究件不挡窗）。码件判定=票支 diff∪wt 脏面碰镜像面；
#     支树皆不可定→保守按码件计；已批未派件无支=未动工不挡。
#   兜底节拍: 批积首件（最老部署面 commit——镜像∪OCI 批并集）起
#     ≥24h 未发→无论条件②发窗并升件报。
#   半截升件: 两基线谱系可序时陈面范围碰陈面=发了一半——
#     镜像面发了 OCI 面没发（或反之）→ops_event「半截部署」升件
#     禁静默半部署（24h 前缀去重）；谱系岔开不可序=面不可判同升。
#
# 环境覆写（测试/对拍面）：O_DEPLOY_WINDOW_REPO（仓树，默认 /opt/o）、
# O_MASTER_REF（默认 origin/master）、O_DEPLOYED_SHA（跳镜像双源直取）、
# O_OCI_DEPLOYED_SHA（跳 OCI 三源直取）、O_OCI_HOST（默认
# ubuntu@149.118.58.98）、O_DEPLOY_FACES_TSV（默认 $REPO/deploy/
# deploy-faces.tsv）、O_SG_HOST（默认 root@192.229.85.35）、DM_ENV_FILE、
# O_DM_QUEUE_SH、O_DEPLOY_WINDOW_NOW（秒戳——兜底节拍可重放）、
# OPS_ESC_SH（升件径覆写——桩测用）。
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
# ── OCI 面判定（T-0425 两半部署制）：OCI 面=镜像面∧¬镜像专属面——
# 未列名 deployable 路径=面不可判→保守按双发（钦定）。镜像专属
# 否列与 OCI 臂级谱系同钉 deploy/deploy-faces.tsv（单本同源——
# 件缺=否列空→全镜像面落 OCI 双发，滥发方向不盲发向）。
FACES_TSV=${O_DEPLOY_FACES_TSV:-$REPO/deploy/deploy-faces.tsv}
is_image_only_path() {
  local cls g
  [ -r "$FACES_TSV" ] || return 1 # 件缺=否列空→全镜像面落 OCI 双发（滥发方向不盲发向）
  while IFS=$'\t' read -r cls g; do
    { [ "$cls" = image_only ] && [ -n "$g" ]; } || continue
    case "$1" in $g) return 0;; esac
  done < "$FACES_TSV"
  return 1
}
is_oci_path() { is_image_path "$1" && ! is_image_only_path "$1"; }
# stdin=path 集 → 任一碰 OCI 面 rc=0
touches_oci() {
  local p
  while IFS= read -r p; do
    [ -n "$p" ] || continue
    p=${p##* -> } # rename 形取新名
    is_oci_path "$p" && return 0
  done
  return 1
}
commit_touches_oci() { # $1=commit（merge 件 -m 逐亲 diff）
  git -C "$REPO" diff-tree -m --no-commit-id -r --name-only "$1" 2>/dev/null \
    | touches_oci
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

# ── OCI 面部署基线三源：env→盒上三臂 $ep.sha 最老件（权威实拍——
# ExecStart 自 fragment 件取+%h/%u/%s 展，同 deploy_oci 环约径）
# →ops_event deploy-oci 行（退化径——deploy_oci 完工同拍落簿）。
# 三源皆败=面不可判→OCI 面按非空保守计（发 deploy_oci 自检径
# 兜——不可判不静默）。臂 SHA 取 committer ts 最老件=批界最广覆盖。──
OCI=${O_OCI_HOST:-ubuntu@149.118.58.98}
OCI_DEPLOYED=${O_OCI_DEPLOYED_SHA:-}; OCI_SRC=env
if [ -z "$OCI_DEPLOYED" ]; then
  oci_shas=$(ssh -o BatchMode=yes -o ConnectTimeout=10 "$OCI" '
    for u in o-collecto o-projector o-research; do
      f=$(systemctl --user show "$u" -p FragmentPath --value 2>/dev/null)
      [ -n "$f" ] || continue
      e=$(grep -m1 "^ExecStart=" "$f" 2>/dev/null | sed "s/^ExecStart=//" | cut -d" " -f1)
      [ -n "$e" ] || continue
      e=${e//%h/$HOME}; e=${e//%u/$(id -un)}; e=${e//%s/${u%.service}}; e=${e//%%/%}
      sed -n "s/^sha=//p" "$e.sha" 2>/dev/null | head -1
    done' 2>/dev/null || true) # tolerate: ssh 断/臂件缺=空集→ops_event 退化源再试
  obest=""; obt=0
  for s in $oci_shas; do
    ct=$(git -C "$REPO" show -s --format=%ct "$s" 2>/dev/null || echo 0)
    [ "${ct:-0}" -gt 0 ] || continue
    if [ "$obt" -eq 0 ] || [ "$ct" -lt "$obt" ]; then obest=$s; obt=$ct; fi
  done
  [ -n "$obest" ] && { OCI_DEPLOYED=$obest; OCI_SRC=box-sha; }
fi
if [ -z "$OCI_DEPLOYED" ] && [ -n "$FIN_DSN" ]; then
  OCI_DEPLOYED=$(psql "$FIN_DSN" -qAt -c "SELECT substring(note from 'deploy-oci ([0-9a-f]+)')
    FROM ops_event WHERE event='部署' AND note LIKE 'deploy-oci % rc=0 %'
    ORDER BY ts DESC LIMIT 1" 2>/dev/null || true) # tolerate: 退化源查询败=面不可判保守径兜非静默
  [ -n "$OCI_DEPLOYED" ] && OCI_SRC=ops-event
fi

# OCI 批基线=部署 SHA↔master merge-base（部署自未并支/热补件非
# master 祖先常态——mb 口径=「master 上该盒未含的码」）；部署 SHA
# 本地不可解/mb 败同归面不可判。
OCI_BASE=""
if [ -n "$OCI_DEPLOYED" ] \
   && git -C "$REPO" rev-parse --verify --quiet "$OCI_DEPLOYED^{commit}" >/dev/null; then
  OCI_BASE=$(git -C "$REPO" merge-base "$OCI_DEPLOYED" "$MASTER_REF" 2>/dev/null || true) # tolerate: mb 败=谱系岔→OCI_STAT=undecidable 按非空保守计非静默
fi
OCI_STAT=resolved; [ -n "$OCI_BASE" ] || OCI_STAT=undecidable

# ── 半截部署检出：两基线谱系可序时陈面范围碰陈面=发了一半——
# OCI陈(oci≺img): diff(oci..img)∩oci≠∅；镜像陈反之；谱系岔开不可
# 序=面不可判同升。「半截部署」升件禁静默半部署（24h 前缀去重）。
# OCI 基线不可判时无谱系可证→不升（发窗径已保守兜）。──
HALF=""
if [ -n "$OCI_DEPLOYED" ] \
   && git -C "$REPO" rev-parse --verify --quiet "$OCI_DEPLOYED^{commit}" >/dev/null \
   && [ "$OCI_DEPLOYED" != "$DEPLOYED" ]; then
  if git -C "$REPO" merge-base --is-ancestor "$OCI_DEPLOYED" "$DEPLOYED"; then
    printf '%s\n' "$(git -C "$REPO" diff --name-only "$OCI_DEPLOYED..$DEPLOYED" 2>/dev/null)" | touches_oci \
      && HALF="镜像面发了OCI面没发——oci=$OCI_DEPLOYED(src=$OCI_SRC)≺img=$DEPLOYED(src=$DEPLOYED_SRC) lag-oci-range碰OCI构建源"
  elif git -C "$REPO" merge-base --is-ancestor "$DEPLOYED" "$OCI_DEPLOYED"; then
    printf '%s\n' "$(git -C "$REPO" diff --name-only "$DEPLOYED..$OCI_DEPLOYED" 2>/dev/null)" | touches_image \
      && HALF="OCI面发了镜像面没发——img=$DEPLOYED(src=$DEPLOYED_SRC)≺oci=$OCI_DEPLOYED(src=$OCI_SRC) lag-image-range碰镜像面"
  else
    HALF="两部署基线谱系岔开不可序——面不可判按半部署疑升 img=$DEPLOYED oci=$OCI_DEPLOYED(src=$OCI_SRC)"
  fi
fi
if [ -n "$HALF" ]; then
  "${OPS_ESC_SH:-$SCRIPT_DIR/ops_esc.sh}" META-DM "" "" "" "半截部署" \
    "$HALF——两半部署制发窗补发同拍" \
    || true # tolerate: 升件末环败静默退（发窗判据不受影响）
  reason "半截部署升件——$HALF"
fi

# ── 判据①批非空：两面各对各自基线判（镜像 deployed..master；
# OCI oci_base..master）──────────────────────────────────
RANGE_FILES=$(git -C "$REPO" diff --name-only "$DEPLOYED..$MASTER_REF" 2>/dev/null)
GRC=$?
[ "$GRC" -eq 0 ] || { emit 查询败; reason "git diff $DEPLOYED..$MASTER_REF 败 rc=$GRC"; exit 1; }
IMG_PATHS=$(printf '%s\n' "$RANGE_FILES" | while IFS= read -r p; do
  [ -n "$p" ] && is_image_path "$p" && printf '%s\n' "$p"
done)
OCI_PATHS=""
if [ "$OCI_STAT" = resolved ]; then
  OCI_PATHS=$(git -C "$REPO" diff --name-only "$OCI_BASE..$MASTER_REF" 2>/dev/null \
    | while IFS= read -r p; do
        [ -n "$p" ] && is_oci_path "$p" && printf '%s\n' "$p"
      done)
fi

# 发窗面集（机读次行）：面不可判=OCI 保守计入——钦定按双发
FACES=""
[ -n "$IMG_PATHS" ] && FACES=image
if [ -n "$OCI_PATHS" ] || [ "$OCI_STAT" = undecidable ]; then
  FACES="${FACES:+$FACES,}oci"
fi
if [ -z "$FACES" ]; then
  emit 批空
  printf 'faces=\n'
  reason "diff 双基线镜像+OCI面俱空——docs-only/零差批不压窗（img=$DEPLOYED src=$DEPLOYED_SRC oci=$OCI_DEPLOYED src=$OCI_SRC）"
  exit 10
fi

# 批件集=两面批并集（票号集自 subject 提 T-\d{4}；兜底节拍锚=
# 批积首件=最老部署面 commit ts）
IMG_COMMITS=$(git -C "$REPO" log --format='%H %ct %s' "$DEPLOYED..$MASTER_REF" \
  | while read -r c ct cs; do
      commit_touches_image "$c" && printf '%s %s %s\n' "$c" "$ct" "$cs"
    done)
OCI_COMMITS=""
if [ "$OCI_STAT" = resolved ]; then
  OCI_COMMITS=$(git -C "$REPO" log --format='%H %ct %s' "$OCI_BASE..$MASTER_REF" \
    | while read -r c ct cs; do
        commit_touches_oci "$c" && printf '%s %s %s\n' "$c" "$ct" "$cs"
      done)
fi
BATCH_COMMITS=$(printf '%s\n%s\n' "$IMG_COMMITS" "$OCI_COMMITS" | awk 'NF' | sort -u)
NBATCH=$(printf '%s\n' "$BATCH_COMMITS" | awk 'NF {n++} END {print n+0}')
EARLIEST=$(printf '%s\n' "$BATCH_COMMITS" | awk 'NF && (min=="" || $2<min) {min=$2} END {print min+0}')
TICKETS=$(printf '%s\n' "$BATCH_COMMITS" | grep -oE 'T-[0-9]{4}' | sort -u | paste -sd, -)
AGE=$(( NOW - EARLIEST ))
reason "批非空 commits=$NBATCH tickets=${TICKETS:-无号} img=$DEPLOYED(src=$DEPLOYED_SRC) oci=${OCI_DEPLOYED:-不可判}(src=$OCI_SRC stat=$OCI_STAT) 批首件龄=${AGE}s"

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
# （EARLIEST=0=批件零——纯不可判批无首件可压，压过语义无锚不升）
BL=${BLOCKERS# }; BL=${BL:-无}
if [ "$EARLIEST" -gt 0 ] && [ "$AGE" -ge 86400 ]; then
  emit 应发
  printf 'faces=%s\n' "$FACES"
  reason "兜底节拍压过——批积首件龄 ${AGE}s≥86400s，无论条件②发窗（挡件=$BL）"
  reason "batch=${TICKETS:-无号} deployed=$DEPLOYED oci=${OCI_DEPLOYED:-不可判}"
  "${OPS_ESC_SH:-$SCRIPT_DIR/ops_esc.sh}" META-DM "" "" "" \
    "DEPLOY-WINDOW-OVERRUN" \
    "批积首件龄${AGE}s≥24h——无论挡件发窗 tickets=${TICKETS:-无号} blockers=$BL faces=$FACES" \
    || true # tolerate: 升件末环败静默退（发窗判据不受影响）
  exit 0
fi

if [ "$BL" != "无" ]; then
  emit 有挡
  printf 'faces=%s\n' "$FACES"
  reason "挡件未清: $BL——批待清窗压（docs/研究件不挡）"
  exit 11
fi

emit 应发
printf 'faces=%s\n' "$FACES"
reason "判据两条件齐——批非空(tickets=${TICKETS:-无号})∧挡件零→当班发窗，faces=$FACES 各面各径（image→deploy_image.sh oci→deploy_oci.sh）"
exit 0
