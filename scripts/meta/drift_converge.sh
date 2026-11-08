#!/usr/bin/env bash
# drift_converge.sh —— GW-001 段F3 漂移自收敛执行器（分级机修）。
#
# 消费 drift_probe.sh 快照（drift.tsv），只动「安全类」且「漂移持续≥2拍」
# 的件（探针误报不得触发写动作——双拍确认硬闸在 streak>=2）：
#   安全类（fixable=1）：
#     unit 内容/缺件 → repo 件 sync（原子换件+daemon-reload；timer/path 另
#       restart 排程面；服务不重启在飞实例——配置文件下个自然 restart 生效）
#     timer disabled/inactive → systemctl --user enable --now（声明态）
#     binary stale/missing → 按已并入 HEAD 的绿戳 sha 重建（HEAD==origin/
#       master∧HEAD 有 check-green 戳∧树净∧deps 证不含钱径包），原子换件
#       后 restart 原活跃属主单元；败即回滚旧件
#   非安全类（fixable=0）：迁移欠账/超登记、master↔origin 落差、sha 不可考/
#     未并、钱径域（deps 含 internal/{exec,trading,venue,ledger}）、构建依赖
#     不可证——只升件不碰（ops_esc 24h 前缀去重；持续漂移每拍由 streak 判）。
#   观测盲（fixable=-）：只报不升（探针/DB 读面问题不是实态漂移）。
#
# 半成品窗：本执行器默认不由 o-mech 调用——探针先行跑满，COS 复核后
# touch $ST/fix_enable（或 o-mech 环境 O_DRIFT_FIX_ENABLE=1）才开写径。
# 失败冻结：机修/升件失败落 $ST/fix-fail-<hash>，24h 内不重试（禁自愈
# 重试风暴——GW-001 §二原则；下次漂移变化后仍按新快照判）。
# 单例：flock $ST/fix.lock（mech 单例外的直跑兜底；拿不到=退）。
#
# 用法：drift_converge.sh（零参数；读 $O_DRIFT_STATE_DIR/drift.tsv）
# 退出：0=跑到尾（单件败=升件不炸整拍）；1=快照/状态目录不可用。
set -u
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO=${O_DRIFT_REPO:-/opt/o}
ST=${O_DRIFT_STATE_DIR:-/var/tmp/o-drift}
SNAP="$ST/drift.tsv"
SYSTEMCTL=${O_DRIFT_SYSTEMCTL:-systemctl}
GO_BIN=${O_DRIFT_GO:-go}
GIT_BIN=${O_DRIFT_GIT:-git}
NOW=${O_DRIFT_NOW:-$(date +%s)}
UNIT_DIR=${O_DRIFT_UNIT_DIR:-$HOME/.config/systemd/user}
BIN_ROOTS=${O_DRIFT_BIN_ROOTS:-"/opt/o/bin:$HOME/bin:/opt/o-research/bin"}
FAIL_FREEZE=${O_DRIFT_FAIL_FREEZE:-86400}
OPS_ESC_SH=${O_DRIFT_OPS_ESC:-"$SCRIPT_DIR/ops_esc.sh"}
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=${O_DRIFT_FIN_DSN:-$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')}
MONEY_RE='^o/internal/(exec|trading|venue|ledger)(/|$)'

case "$NOW" in ''|*[!0-9]*) echo "[drift-fix] O_DRIFT_NOW 非数: $NOW" >&2; exit 2 ;; esac
[ -r "$SNAP" ] || exit 0
mkdir -p "$ST" 2>/dev/null || { echo "[drift-fix] 状态目录不可建: $ST" >&2; exit 1; }
exec 9>"$ST/fix.lock" || exit 1
flock -n 9 2>/dev/null || { echo "[drift-fix] 收敛闸在飞——后到即退" >&2; exit 0; }

sq() { printf '%s' "${1:-}" | sed "s/'/''/g"; }
clean() { printf '%s' "${1:-}" | tr '\t\n' '  '; }
file_hash() { sha256sum "$1" 2>/dev/null | cut -d' ' -f1; }
unit_hash() {
  local out
  out=$("$SYSTEMCTL" --user cat "$1" 2>/dev/null) || return 1
  [ -n "$out" ] || return 1
  printf '%s\n' "$out" | awk 'BEGIN{h=1} h && /^# \// {next} {h=0; print}' \
    | sha256sum | cut -d' ' -f1
}
bin_sha() {
  "$GO_BIN" version -m "$1" 2>/dev/null \
    | awk '$1=="build" && $2 ~ /^vcs\.revision=/ {sub(/^vcs\.revision=/,"",$2); print $2; exit}'
}
ledger() { # $1=event $2=note
  [ -n "$FIN_DSN" ] || return 0
  local qn; qn=$(sq "$(clean "$2")")
  psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
    VALUES('MECH','$1','','$qn')" >/dev/null 2>&1 # tolerate: 执行账末环败静默退（快照/升件面仍在）
  return 0
}
escalate() { # $1=key $2=kind $3=name $4=subtype $5=detail $6=streak $7=prefix
  local key=$1 kind=$2 name=$3 st=$4 dt=$5 streak=$6 prefix=$7
  "$OPS_ESC_SH" CTO '' '漂移' "$key" "$prefix" \
    "kind=$kind name=$name subtype=$st detail=$dt streak=$streak" >/dev/null 2>&1
  return 0
}
fail_path() { printf '%s/fix-fail-%s' "$ST" "$(printf '%s' "$1" | sha256sum | cut -c1-16)"; }
frozen() { # $1=key → rc0=失败冻结窗内
  local f age
  f=$(fail_path "$1")
  [ -f "$f" ] || return 1
  age=$(( NOW - $(stat -c %Y "$f" 2>/dev/null || echo 0) )) # tolerate: stat 败=按陈件（重试一次）保守
  [ "$age" -lt "$FAIL_FREEZE" ]
}

# ── 安全类 unit/timer：repo→live 原子 sync ─────────────────────
fail_reason=""
fix_unit() { # $1=name $2=spec $3=subtype $4=kind
  local name=$1 spec=$2 st=$3 kind=$4 live repo_abs tmp newh repoh
  fail_reason=""
  case "$spec" in deploy/*) ;; *) fail_reason="repo 路径非 deploy/*（$spec）"; return 1 ;; esac
  case "$name" in ''|*/*|..*) fail_reason="unit 名非法"; return 1 ;; esac
  repo_abs="$REPO/$spec"; live="$UNIT_DIR/$name"
  [ -f "$repo_abs" ] || { fail_reason="repo 件缺席 $spec"; return 1; }
  case "$st" in
    *content*|*missing*)
      tmp="$live.drift-new.$$"
      install -m 644 "$repo_abs" "$tmp" 2>/dev/null \
        || { fail_reason="写临时件败 $tmp"; return 1; }
      mv -f -- "$tmp" "$live" 2>/dev/null \
        || { rm -f -- "$tmp" 2>/dev/null; fail_reason="换件败 $live"; return 1; }
      "$SYSTEMCTL" --user daemon-reload >/dev/null 2>&1 \
        || { fail_reason="daemon-reload 败"; return 1; }
      case "$name" in
        *.timer|*.path)
          "$SYSTEMCTL" --user restart "$name" >/dev/null 2>&1 \
            || { fail_reason="排程单元 restart 败 $name"; return 1; } ;;
      esac
      ;;
  esac
  case "$st" in
    *disabled*|*inactive*)
      "$SYSTEMCTL" --user enable --now "$name" >/dev/null 2>&1 \
        || { fail_reason="enable --now 败 $name"; return 1; }
      ;;
  esac
  # 回探转绿：内容哈希（若有内容面）+ enabled/active（若有状态面）
  case "$st" in
    *content*|*missing*)
      newh=$(unit_hash "$name") || { fail_reason="回探 live 不可读"; return 1; }
      repoh=$(file_hash "$repo_abs")
      [ "$newh" = "$repoh" ] || { fail_reason="回探未绿 live=${newh:0:8} repo=${repoh:0:8}"; return 1; }
      ;;
  esac
  case "$st" in
    *disabled*)
      case "$("$SYSTEMCTL" --user is-enabled "$name" 2>/dev/null)" in
        enabled|enabled-runtime) ;;
        *) fail_reason="回探 enabled 未绿"; return 1 ;;
      esac
      ;;
  esac
  case "$st" in
    *inactive*)
      [ "$("$SYSTEMCTL" --user is-active "$name" 2>/dev/null)" = active ] \
        || { fail_reason="回探 active 未绿"; return 1; }
      ;;
  esac
  return 0
}

# ── 安全类 binary：按已并入 HEAD 的绿戳 sha 重建 ──────────────
DEPS_DIRS=""; MONEY_DEP=""
deps_scan() {
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
owners_of() { # $1=live 路径 → 属主 .service 名（ExecStart 字面）
  local live=$1 f
  for f in "$UNIT_DIR"/*.service; do
    [ -f "$f" ] || continue
    awk -v p="ExecStart=$live" 'index($0,p)==1 {n=substr($0,length(p)+1); if (n=="" || n ~ /^[[:space:]]/) {print FILENAME; exit}}' "$f" 2>/dev/null
  done | xargs -r -n1 basename 2>/dev/null # tolerate: xargs 空集=无属主合法径（feishu-notify 类按需工具）
}
fix_binary() { # $1=live $2=spec
  local live=$1 spec=$2 target om gitdir tmpbin tmplive newsha oldsha owners u pre post
  local dirok=1
  fail_reason=""
  case "$live" in /*) ;; *) fail_reason="live 非绝对路径"; return 1 ;; esac
  case ":$BIN_ROOTS:" in *":$(dirname "$live"):"*) ;; *) fail_reason="live 路径不在机修允许根"; return 1 ;; esac
  target=$("$GIT_BIN" -C "$REPO" rev-parse HEAD 2>/dev/null) \
    || { fail_reason="HEAD 不可解"; return 1; }
  om=$("$GIT_BIN" -C "$REPO" rev-parse origin/master 2>/dev/null) \
    || { fail_reason="origin/master 不可解"; return 1; }
  [ "$target" = "$om" ] || { fail_reason="HEAD≠origin/master（未收敛不重建）"; return 1; }
  [ -z "$("$GIT_BIN" -C "$REPO" status --porcelain 2>/dev/null)" ] \
    || { fail_reason="树脏（脏构建 sha 不可信）"; return 1; }
  gitdir=$("$GIT_BIN" -C "$REPO" rev-parse --git-common-dir 2>/dev/null) \
    || { fail_reason="git-common-dir 不可解"; return 1; }
  case "$gitdir" in /*) ;; *) gitdir="$REPO/$gitdir" ;; esac
  [ -f "$gitdir/check-green-$target" ] \
    || { fail_reason="HEAD 无绿戳 check-green-${target:0:12}（无戳不重建）"; return 1; }
  deps_scan "$spec" || { fail_reason="go list -deps 败（构建依赖不可证）"; return 1; }
  [ -z "$MONEY_DEP" ] || { fail_reason="deps 含钱径包 $MONEY_DEP（钱径域不碰）"; return 1; }
  tmpbin="$ST/build-$(basename "$live").$$"
  ( cd "$REPO" && "$GO_BIN" build -o "$tmpbin" "$spec" ) >/dev/null 2>&1 \
    || { rm -f -- "$tmpbin" 2>/dev/null; fail_reason="go build 败 $spec"; return 1; }
  newsha=$(bin_sha "$tmpbin")
  [ "$newsha" = "$target" ] \
    || { rm -f -- "$tmpbin" 2>/dev/null; fail_reason="构建嵌入 sha=${newsha:-无}≠target=${target:0:12}"; return 1; }
  owners=$(owners_of "$live")
  [ -f "$live" ] && cp -f -- "$live" "$live.drift-bak" 2>/dev/null # tolerate: 备份败=下方换件前旧件仍在，失败可回滚面退化
  tmplive="$live.drift-new.$$"
  cp -f -- "$tmpbin" "$tmplive" 2>/dev/null \
    || { rm -f -- "$tmpbin" "$tmplive" 2>/dev/null; fail_reason="写新件败 $tmplive"; return 1; }
  chmod 755 "$tmplive" 2>/dev/null # tolerate: chmod 败=cp 保留源 755；下方 mv 后核读面兜底
  mv -f -- "$tmplive" "$live" 2>/dev/null \
    || { rm -f -- "$tmpbin" "$tmplive" 2>/dev/null; fail_reason="原子换件败 $live"; return 1; }
  rm -f -- "$tmpbin" 2>/dev/null # tolerate: 临时构建件删败=下一拍同前缀覆盖（/var/tmp 保洁面兜底）
  # restart 原活跃属主（inactive 单元不拉起——不越期望态）；任一败=回滚旧件
  for u in $owners; do
    pre=$("$SYSTEMCTL" --user is-active "$u" 2>/dev/null) # tolerate: is-active 读败=空按非活跃（不重启）保守
    [ "$pre" = active ] || continue
    if ! "$SYSTEMCTL" --user restart "$u" >/dev/null 2>&1; then
      fail_reason="属主 restart 败 $u"; dirok=0; break
    fi
    post=$("$SYSTEMCTL" --user is-active "$u" 2>/dev/null) # tolerate: 回探读败=空按未绿（回滚）保守
    if [ "$post" != active ]; then
      fail_reason="属主回探未 active $u（$post）"; dirok=0; break
    fi
  done
  if [ "$dirok" = 0 ]; then
    [ -f "$live.drift-bak" ] && cp -f -- "$live.drift-bak" "$live" 2>/dev/null # tolerate: 回滚末环败=升件行已明示人工介入（不自愈重试）
    for u in $owners; do
      "$SYSTEMCTL" --user restart "$u" >/dev/null 2>&1 # tolerate: 回滚 restart 尽力——失败面由升件行显形
    done
    return 1
  fi
  # 回探转绿：live 嵌入 sha == target
  [ "$(bin_sha "$live")" = "$target" ] \
    || { fail_reason="回探 live sha≠target"; return 1; }
  rm -f -- "$live.drift-bak" 2>/dev/null # tolerate: 备份清败=留件不碍对账（非活件面）
  return 0
}

# ── 主循环：streak≥2 才动手（探针误报不得触发写） ─────────────
while IFS=$'\t' read -r key kind name spec subtype detail fixable streak; do
  [ -n "$key" ] || continue
  case "$streak" in ''|*[!0-9]*) continue ;; esac
  [ "$streak" -ge 2 ] || continue
  [ "$fixable" = 1 ] || [ "$fixable" = 0 ] || continue
  frozen "$key" && { echo "[drift-fix] $key 失败冻结窗内——跳过" >&2; continue; }
  if [ "$fixable" = 0 ]; then
    escalate "$key" "$kind" "$name" "$subtype" "$detail" "$streak" "DRIFT:$key"
    continue
  fi
  case "$kind" in
    unit|timer)
      if fix_unit "$name" "$spec" "$subtype" "$kind"; then
        ledger '机修' "机修 $key 同步 repo→live subtype=$subtype detail=$detail streak=$streak 回探=绿"
        rm -f -- "$(fail_path "$key")" 2>/dev/null # tolerate: 失败痕清败=下拍仍按无痕（成功已留账）
      else
        printf '%s\n' "$NOW" > "$(fail_path "$key")" 2>/dev/null # tolerate: 冻结痕写败=下拍重试一次（败面已升件）
        ledger '机修' "机修败 $key subtype=$subtype detail=$detail 因=${fail_reason:-未知}（24h 冻结不重试）"
        escalate "$key" "$kind" "$name" "$subtype" "${detail}；机修败：${fail_reason:-未知}" "$streak" "DRIFT-FIX-FAIL:$key"
      fi
      ;;
    binary)
      if fix_binary "$name" "$spec"; then
        ledger '机修' "机修 $key 重建 subtype=$subtype detail=$detail streak=$streak 回探=绿"
        rm -f -- "$(fail_path "$key")" 2>/dev/null # tolerate: 失败痕清败=下拍仍按无痕（成功已留账）
      else
        printf '%s\n' "$NOW" > "$(fail_path "$key")" 2>/dev/null # tolerate: 冻结痕写败=下拍重试一次（败面已升件）
        ledger '机修' "机修败 $key subtype=$subtype detail=$detail 因=${fail_reason:-未知}（24h 冻结不重试）"
        escalate "$key" "$kind" "$name" "$subtype" "${detail}；机修败：${fail_reason:-未知}" "$streak" "DRIFT-FIX-FAIL:$key"
      fi
      ;;
  esac
done < "$SNAP"
exit 0
