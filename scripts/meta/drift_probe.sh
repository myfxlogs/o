#!/usr/bin/env bash
# drift_probe.sh —— GW-001 段F2 漂移只读探针（GitOps 期望态/实态对账）。
#
# 期望态=deploy/manifest.tsv（repo 件唯一真源）；本件只读实态：
#   unit/timer: systemctl --user cat 内容哈希 vs repo 件哈希；
#               timer 另读 UnitFileState(enabled)+ActiveState(active)；
#   binary:     go version -m 取 vcs.revision vs HEAD 构建源（deps 变更判
#               落后；无嵌入=sha 不可考；源未并 HEAD=sha-unmerged；deps 含
#               钱径包=钱径域只升件不碰）；
#   migration:  live schema_migrations max vs migrate_table 登记链最大号；
#   repo:       HEAD↔origin/master 落差（master 落差=非安全类）。
#
# 零写径：只读 live 实态；写面仅三处——观测态快照 $ST/drift.tsv+last.ts、
# ops_event 说明类观测行（event=漂移/漂移消，不进票相位——段B1 先例）、
# 不触任何 live unit/binary/DB 数据。机修执行器=drift_converge.sh（后启，
# 半成品窗：探针先行零写径，执行器须显式开关）。
#
# 行格式（drift.tsv，TAB 八列）：
#   key kind name spec subtype detail fixable streak
#   fixable=1 安全类可机修｜0 非安全类只升件｜- 观测盲只报不升
# streak=连续拍数（同 key 上次快照龄≤O_DRIFT_STREAK_WINDOW 才累加；
#   机修双拍确认由 converge 读 streak≥2 判）。
#
# 用法：drift_probe.sh [--count|--keys]（缺省=采一拍）
#   --count  输出当前快照行数（gauges drift_open 消费面；零再采集）
#   --keys   输出当前快照 key 逗号清单（gauges {d} 明细面）
# 退出：0=采集完成（个别实态不可读=行内观测盲，不炸拍）；1=manifest/
#   状态目录不可用；2=用法错。DM_ENV_FILE 缺/FIN_DSN 缺=只落本地快照
#   不写 ops_event（探针面仍全跑）。
set -u
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO=${O_DRIFT_REPO:-/opt/o}
MANIFEST=${O_DRIFT_MANIFEST:-"$REPO/deploy/manifest.tsv"}
ST=${O_DRIFT_STATE_DIR:-/var/tmp/o-drift}
SNAP="$ST/drift.tsv"
TSF="$ST/last.ts"
SYSTEMCTL=${O_DRIFT_SYSTEMCTL:-systemctl}
GO_BIN=${O_DRIFT_GO:-go}
GIT_BIN=${O_DRIFT_GIT:-git}
NOW=${O_DRIFT_NOW:-$(date +%s)}
WINDOW=${O_DRIFT_STREAK_WINDOW:-1800}
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=${O_DRIFT_FIN_DSN:-$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')}
SG_HOST=${O_DRIFT_SG_HOST:-root@192.229.85.35}
MD_HOST=${O_DRIFT_MD_HOST:-ubuntu@149.118.58.98}
MONEY_RE='^o/internal/(exec|trading|venue|ledger)(/|$)'

case "${1:-}" in
  --count)
    if [ -f "$SNAP" ]; then wc -l < "$SNAP" | tr -d ' '; else echo 0; fi
    exit 0 ;;
  --keys)
    [ -f "$SNAP" ] || { echo ""; exit 0; }
    awk -F'\t' '{printf "%s%s", sep, $1; sep=","}' "$SNAP"; echo
    exit 0 ;;
  ''|--quiet) ;;
  *) echo "usage: drift_probe.sh [--count|--keys]" >&2; exit 2 ;;
esac

case "$NOW" in ''|*[!0-9]*) echo "[drift] O_DRIFT_NOW 非数: $NOW" >&2; exit 2 ;; esac
[ -r "$MANIFEST" ] || { echo "[drift] manifest 缺席: $MANIFEST" >&2; exit 1; }
mkdir -p "$ST" 2>/dev/null || { echo "[drift] 状态目录不可建: $ST" >&2; exit 1; }
CUR=$(mktemp "$ST/.drift.XXXXXX") || exit 1
trap 'rm -f -- "$CUR"' EXIT
RC=0
sq() { printf '%s' "${1:-}" | sed "s/'/''/g"; }
clean() { printf '%s' "${1:-}" | tr '\t\n' '  '; }

# ── 上一拍快照（龄>窗=不续 streak，防久隔两拍算连续） ─────────────
declare -A PS
PS_N=0
PREV_TS=0
if [ -f "$SNAP" ] && [ -f "$TSF" ]; then
  PREV_TS=$(cat "$TSF" 2>/dev/null) # tolerate: 时间戳读败=按首拍处理（streak 归1 保守）
  case "$PREV_TS" in ''|*[!0-9]*) PREV_TS=0 ;; esac
  if [ "$PREV_TS" -gt 0 ] && [ "$NOW" -ge "$PREV_TS" ] \
     && [ $((NOW - PREV_TS)) -le "$WINDOW" ]; then
    while IFS=$'\t' read -r k _ _ _ _ _ _ streak; do
      [ -n "$k" ] || continue
      PS["$k"]=$streak
      PS_N=$((PS_N + 1))
    done < "$SNAP"
  fi
fi
CUR_KEYS=" "

row_add() { # $1=key $2=kind $3=name $4=spec $5=subtype $6=detail $7=fixable
  local key=$1 streak=1
  if [ -n "${PS[$key]+x}" ]; then
    case "${PS[$key]}" in
      ''|*[!0-9]*) : ;;
      *) streak=$(( PS[$key] + 1 )) ;;
    esac
  fi
  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
    "$key" "$2" "$3" "$4" "$5" "$(clean "$6")" "$7" "$streak" >> "$CUR"
  CUR_KEYS="$CUR_KEYS$key "
}

# ── 实态读面（全只读） ─────────────────────────────────────────
file_hash() { sha256sum "$1" 2>/dev/null | cut -d' ' -f1; }

unit_hash() { # $1=unit 名——systemctl --user cat 去 systemd 头行后哈希；败=rc1
  local out
  out=$("$SYSTEMCTL" --user cat "$1" 2>/dev/null) || return 1
  [ -n "$out" ] || return 1
  printf '%s\n' "$out" \
    | awk 'BEGIN{h=1} h && /^# \// {next} {h=0; print}' \
    | sha256sum | cut -d' ' -f1
}

bin_sha() { # $1=live 二进制——go version -m 的 vcs.revision（无=空）
  "$GO_BIN" version -m "$1" 2>/dev/null \
    | awk '$1=="build" && $2 ~ /^vcs\.revision=/ {sub(/^vcs\.revision=/,"",$2); print $2; exit}'
}

# deps 扫描：DEPS_DIRS=repo 相对目录串（git diff 路径面）；MONEY_DEP=钱径包
# （命中即非安全类）；go list 败=不可证→dep-unknown 不进机修面（fail-safe）。
DEPS_DIRS=""
MONEY_DEP=""
deps_scan() { # $1=cmd 构建路径
  local out dir imp
  DEPS_DIRS=""; MONEY_DEP=""
  out=$(cd "$REPO" && "$GO_BIN" list -deps -f '{{.Dir}}|{{.ImportPath}}' "$1" 2>/dev/null) || return 1
  [ -n "$out" ] || return 1
  while IFS='|' read -r dir imp; do
    [ -n "$dir" ] || continue
    case "$dir" in "$REPO"/*) DEPS_DIRS="$DEPS_DIRS ${dir#"$REPO"/}" ;; esac
    if printf '%s' "$imp" | grep -qE "$MONEY_RE"; then MONEY_DEP=$imp; fi
  done <<< "$out"
  [ -n "$DEPS_DIRS" ] || return 1
}

reg_max() { # $1=域 $2=登记表相对路径——登记链最大号
  grep -oE '\{[0-9]+, "'"$1"'_[0-9]+_[^"]*\.sql"' "$REPO/$2" 2>/dev/null \
    | sed -n 's/^{\([0-9]*\),.*/\1/p' | sort -n | tail -1
}

schema_live() { # $1=域——live schema_migrations max；读败=rc1（观测盲非零假设）
  local dom=$1 out rc
  if [ -n "${O_DRIFT_SCHEMA_CMD:-}" ]; then
    out=$("$O_DRIFT_SCHEMA_CMD" "$dom" 2>/dev/null); rc=$?
  elif [ "$dom" = fin ]; then
    out=$(timeout 8 ssh -o BatchMode=yes -o ConnectTimeout=5 "$SG_HOST" \
      "docker exec o-postgres psql -U o -d o -tc 'SELECT COALESCE(max(version),0) FROM schema_migrations'" 2>/dev/null); rc=$?
  else
    out=$(timeout 10 ssh -o BatchMode=yes -o ConnectTimeout=5 "$MD_HOST" \
      "sudo docker exec o-postgres-md psql -U o -d o -tc 'SELECT COALESCE(max(version),0) FROM schema_migrations'" 2>/dev/null); rc=$?
  fi
  [ "$rc" -eq 0 ] || return 1
  out=$(printf '%s' "$out" | tr -d '[:space:]')
  case "$out" in ''|*[!0-9]*) return 1 ;; esac
  printf '%s\n' "$out"
}

# ── manifest 遍历采集 ─────────────────────────────────────────
HEAD=$("$GIT_BIN" -C "$REPO" rev-parse HEAD 2>/dev/null) # tolerate: sha 不可解=空，binary 落后判按空保守（sha-unrecoverable 面）
HEAD=${HEAD:-}
while IFS=$'\t' read -r kind name repo expect; do
  case "${kind:-}" in ''|\#*) continue ;; esac
  [ "$kind" = kind ] && continue # 表头行（kind<TAB>name<TAB>repo<TAB>expect）
  name=${name%$'\r'}; repo=${repo%$'\r'}; expect=${expect%$'\r'}
  case "$expect" in
    sync|enabled|green|max) ;;
    *) echo "[drift] manifest expect 非法: $kind/$name/$expect" >&2; RC=1; continue ;;
  esac
  case "$kind" in
    unit)
      case "$name" in */*|..*) echo "[drift] manifest unit 名非法: $name" >&2; RC=1; continue ;; esac
      repo_abs="$REPO/$repo"
      if [ ! -f "$repo_abs" ]; then
        row_add "unit:$name" unit "$name" "$repo" missing "repo 件缺席: $repo" 0
        continue
      fi
      rh=$(file_hash "$repo_abs")
      if ! lh=$(unit_hash "$name"); then
        row_add "unit:$name" unit "$name" "$repo" missing "live unit 未装/不可读" 1
        continue
      fi
      if [ "$lh" != "$rh" ]; then
        row_add "unit:$name" unit "$name" "$repo" content \
          "live=${lh:0:8} repo=${rh:0:8}" 1
      fi
      ;;
    timer)
      case "$name" in */*|..*) echo "[drift] manifest timer 名非法: $name" >&2; RC=1; continue ;; esac
      repo_abs="$REPO/$repo"
      if [ ! -f "$repo_abs" ]; then
        row_add "timer:$name" timer "$name" "$repo" missing "repo 件缺席: $repo" 0
        continue
      fi
      rh=$(file_hash "$repo_abs")
      subtype=""
      subdetail=""
      if ! lh=$(unit_hash "$name"); then
        subtype="missing"; subdetail="live timer 未装/不可读"
      else
        if [ "$lh" != "$rh" ]; then
          subtype="content"; subdetail="live=${lh:0:8} repo=${rh:0:8}"
        fi
        en=$("$SYSTEMCTL" --user is-enabled "$name" 2>/dev/null) # tolerate: is-enabled 读败=空按未 enabled 保守报
        case "$en" in
          enabled|enabled-runtime) ;;
          *) subtype=${subtype:+$subtype+}disabled; subdetail="${subdetail:+$subdetail }enabled=${en:-不可读}" ;;
        esac
        ac=$("$SYSTEMCTL" --user is-active "$name" 2>/dev/null) # tolerate: is-active 读败=空按非 active 保守报
        [ "$ac" = active ] || { subtype=${subtype:+$subtype+}inactive; subdetail="${subdetail:+$subdetail }active=${ac:-不可读}"; }
      fi
      [ -n "$subtype" ] && row_add "timer:$name" timer "$name" "$repo" "$subtype" "$subdetail" 1
      ;;
    binary)
      live=${name/#\~/$HOME}
      if [ ! -f "$live" ]; then
        row_add "binary:$live" binary "$live" "$repo" missing "live 二进制缺席" 1
        continue
      fi
      bsha=$(bin_sha "$live")
      if [ -z "$bsha" ]; then
        row_add "binary:$live" binary "$live" "$repo" sha-unrecoverable \
          "go version -m 无 vcs.revision（构建源不可考——只升件不碰）" 0
        continue
      fi
      if ! "$GIT_BIN" -C "$REPO" cat-file -e "$bsha^{commit}" 2>/dev/null; then
        row_add "binary:$live" binary "$live" "$repo" sha-unrecoverable \
          "vcs.revision=${bsha:0:12} 仓内不可解（构建源不可考——只升件不碰）" 0
        continue
      fi
      [ "$bsha" = "$HEAD" ] && continue
      if ! "$GIT_BIN" -C "$REPO" merge-base --is-ancestor "$bsha" HEAD 2>/dev/null; then
        row_add "binary:$live" binary "$live" "$repo" sha-unmerged \
          "vcs.revision=${bsha:0:12} 非 HEAD 祖先（未并码不机修——只升件）" 0
        continue
      fi
      if ! deps_scan "$repo"; then
        row_add "binary:$live" binary "$live" "$repo" dep-unknown \
          "go list -deps 败（构建依赖不可证——只升件不碰）" 0
        continue
      fi
      if [ -n "$MONEY_DEP" ]; then
        row_add "binary:$live" binary "$live" "$repo" money-path \
          "deps 含钱径包 $MONEY_DEP（只升件不碰）vcs=${bsha:0:12}" 0
        continue
      fi
      # shellcheck disable=SC2086 # DEPS_DIRS 为空格分隔路径串（repo 路径无空格，构造成立）
      "$GIT_BIN" -C "$REPO" diff --quiet "$bsha"..HEAD -- $DEPS_DIRS go.mod go.sum 2>/dev/null
      drc=$?
      case "$drc" in
        0) ;;
        1) row_add "binary:$live" binary "$live" "$repo" stale \
             "vcs=${bsha:0:12} head=${HEAD:0:12} deps 已前进（按已并 HEAD 绿戳 sha 重建）" 1 ;;
        *) row_add "binary:$live" binary "$live" "$repo" dep-unknown \
             "git diff deps 败 rc=$drc（构建源不可证——只升件不碰）" 0 ;;
      esac
      ;;
    migration)
      case "$name" in fin|md) ;; *) echo "[drift] manifest migration 域非法: $name" >&2; RC=1; continue ;; esac
      reg=$(reg_max "$name" "$repo")
      case "$reg" in ''|*[!0-9]*) echo "[drift] 登记链 $name 最大号不可解（$repo）" >&2; RC=1; continue ;; esac
      if ! live=$(schema_live "$name"); then
        row_add "migration:$name" migration "$name" "$repo" unreadable \
          "schema_migrations 读败（观测盲——只报不升）reg=$reg" -
        continue
      fi
      if [ "$live" -lt "$reg" ]; then
        row_add "migration:$name" migration "$name" "$repo" debt \
          "live=$live 登记=$reg（迁移欠账——只升件不碰）" 0
      elif [ "$live" -gt "$reg" ]; then
        row_add "migration:$name" migration "$name" "$repo" ahead \
          "live=$live 登记=$reg（live 超登记——未入册迁移在库，只升件不碰）" 0
      fi
      ;;
    *)
      echo "[drift] manifest kind 非法: $kind" >&2; RC=1 ;;
  esac
done < "$MANIFEST"

# ── repo 落差面（无 manifest 行——实态自取证；非安全类只升件） ──
OM=$("$GIT_BIN" -C "$REPO" rev-parse origin/master 2>/dev/null) # tolerate: origin/master 不可解=空走下方不可解行
OM=${OM:-}
if [ -n "$HEAD" ] && [ -n "$OM" ]; then
  ahead=$("$GIT_BIN" -C "$REPO" rev-list --count "$OM".."$HEAD" 2>/dev/null) # tolerate: rev-list 读败=空按不可解下行
  behind=$("$GIT_BIN" -C "$REPO" rev-list --count "$HEAD".."$OM" 2>/dev/null) # tolerate: rev-list 读败=空按不可解下行
  case "${ahead:-x}${behind:-x}" in
    *[!0-9]*) row_add "repo:master-gap" repo master "$REPO" unreadable "HEAD↔origin/master 落差不可解" - ;;
    *)
      if [ "$ahead" -gt 0 ] || [ "$behind" -gt 0 ]; then
        row_add "repo:master-gap" repo master "$REPO" master-gap \
          "ahead=$ahead behind=$behind head=${HEAD:0:12} origin=${OM:0:12}（落差非安全类只升件不碰）" 0
      fi
      ;;
  esac
else
  row_add "repo:master-gap" repo master "$REPO" unreadable "HEAD/origin/master 不可解" -
fi

# ── 快照落盘（先落新快照再发观测行——观测行败不丢对账面） ──────
mv -f -- "$CUR" "$SNAP" 2>/dev/null || { echo "[drift] 快照落盘败: $SNAP" >&2; exit 1; }
printf '%s\n' "$NOW" > "$TSF" 2>/dev/null # tolerate: 时间戳写败=下拍按首拍（streak 归1 保守）

# ── ops_event 说明类观测行（event=漂移/漂移消——不进票相位） ────
oe() { # $1=event $2=note $3=dedupe-prefix（空=不去重）
  [ -n "$FIN_DSN" ] || return 0
  local ev=$1 note=$2 prefix=${3:-} hit
  local qn qp
  qn=$(sq "$note")
  if [ -n "$prefix" ]; then
    qp=$(sq "$prefix")
    hit=$(psql "$FIN_DSN" -qAt -c "SELECT 1 FROM ops_event WHERE event='$ev'
      AND note LIKE '$qp%' AND ts>now()-interval '24 hours' LIMIT 1" 2>/dev/null) # tolerate: 去重查询败=按未报处理（宁可重报不吞漂移）
    [ "$hit" = 1 ] && return 0
  fi
  psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
    VALUES('MECH','$ev','','$qn')" >/dev/null 2>&1 # tolerate: 观测行末环败静默退（结构性限度在案，快照仍在）
  return 0
}
while IFS=$'\t' read -r k kind nm sp st dt fx streak; do
  [ -n "$k" ] || continue
  if [ -n "${PS[$k]+x}" ]; then continue; fi
  oe 漂移 "漂移 $k [$st] $dt fixable=$fx streak=$streak" "漂移 $k "
done < "$SNAP"
# 消行：上一拍键集不在本拍（PS 在 mv 前读入；空集合法径）
if [ "$PREV_TS" -gt 0 ] && [ "$PS_N" -gt 0 ]; then
  for k in "${!PS[@]}"; do
    case "$CUR_KEYS" in
      *" $k "*) ;;
      *) oe 漂移消 "漂移消 $k 回探转绿/已消" "" ;;
    esac
  done
fi

exit "$RC"
