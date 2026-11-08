#!/usr/bin/env bash
# wt_gc.sh —— T-0331 worktree/t 分支 GC 器（钦定双闸：unit active+树龄
# 双查禁碰活 run 工树；删枝只用 -d 拒删未合枝天然闸；--dry-run 干跑开关
# 钦定；清场数 ops_event 'GC' 行录）。
# T-0375 件④ 退役口径：audit-*/dec-t-* 系支=审计/裁决过程支——结论档在库
# （docs/decisions+docs/ops 档案），支体非审计载体；「无 worktree 挂+
# 末 commit>RETIRE_D(14d)」即退役（-D 硬删，未合亦删——口径钦定；
# 带 wt 支与 RETIRE_D 内支不动）。GC_REPO 覆写=桩测径（默认 /opt/o）。
set -u
REPO="${GC_REPO:-/opt/o}"
DRY=0; DISK=0
for a in "$@"; do case "$a" in --dry-run) DRY=1;; --disk) DISK=1;; esac; done
ENVF="${GC_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
AGE_D=7
RETIRE_D=14
now=$(date +%s)
rm_tree=0; rm_branch=0; retire=0; retire_keep=0; keep=0; skip_active=0; skip_dirty=0; orphan=0

# 在飞 TID 集：o-worker-* unit(active/activating) 剥 o-worker-/.service → T-NNNN
LIVE=$(systemctl --user list-units 'o-worker-*' --state=active,activating \
  --plain --no-legend 2>/dev/null | awk '{print $1}' |
  sed -n 's/^o-worker-\(.*\)\.service$/\1/p' | tr 'a-z' 'A-Z')

# 工树名方言归一→T-NNNN（t0330/audit-t-0001/t-0226/t0079-v2 取首个数字段）
tid_norm() {
  local d
  d=$(basename "$1" | tr 'A-Z' 'a-z' | grep -oE '[0-9]+' | head -1)
  [ -n "$d" ] && printf 'T-%04d' "$((10#$d))"
}

# ── T-0466 保洁面（裁域①③⑥——deploy 构建产物出径+孤儿锁+GOCACHE 界）──
# 三件：⑤孤儿锁扫杀（/tmp/o-check_all-*.lock 持fd者 ppid=1=父死孤儿
#   →杀进程树+ops_event 报行——实证：kill wrapper 后 go test 树孤儿
#   持 fd9 锁 15min+ 新跑傻等）；①builder prune 双触发（周拍 marker
#   >7d 常规 -f／根分区>80% 高水位 -af——141.5G 实证 -af 才放得动）；
# ③go clean -cache 周拍+GOCACHE 分家目录清（/tmp/gocache-<topkey>
#   三闸：wt 已删∧锁未持∧龄>24h——锁文件绝不删，删了双 inode 锁
#   互斥破）。桩测面：GC_LOCK_GLOB/GC_GOCACHE_GLOB/GC_ROOTFS_PCT/
#   GC_ROOTFS_HIGH_PCT/GC_GOCACHE_AGE_S/GC_OPS_SINK/GC_PRUNE_MARK/
#   GC_GOCLEAN_MARK 覆写；GC_HYGIENE=0 整段关（他测实跑 wt_gc 防
#   真 prune/杀真进程侧效）。
disk_hygiene() {
  local orphank=0 prune_hit=0 gclean=0 gck=0 pct week_hit
  LOCK_GLOB="${GC_LOCK_GLOB:-/tmp/o-check_all-*.lock}"
  GOCACHE_GLOB="${GC_GOCACHE_GLOB:-/tmp/gocache-*}"
  PRUNE_MARK="${GC_PRUNE_MARK:-/var/tmp/o-wt-gc.last_prune}"
  GCLEAN_MARK="${GC_GOCLEAN_MARK:-/var/tmp/o-wt-gc.last_goclean}"
  GC_AGE_S="${GC_GOCACHE_AGE_S:-86400}"

  ops_evt() { # $1=note——GC 域 ops_event 行（GC_OPS_SINK 桩测面优先）
    if [ -n "${GC_OPS_SINK:-}" ]; then printf '%s\n' "$1" >> "$GC_OPS_SINK"; return 0; fi
    [ -n "$FIN_DSN" ] && psql "$FIN_DSN" -qAt -c \
      "INSERT INTO ops_event(seat,event,ticket,note) VALUES('META-GC','GC','T-0466','$1')" \
      >/dev/null 2>&1 || true # tolerate: 保洁报行写败不阻断——审计件非硬闸
  }
  rootfs_pct() {
    if [ -n "${GC_ROOTFS_PCT:-}" ]; then printf '%s' "$GC_ROOTFS_PCT"; return; fi
    df -P / 2>/dev/null | awk 'NR==2{gsub(/%/,"",$5);print $5}'
  }
  descendants() { local p=$1 c; for c in $(pgrep -P "$p" 2>/dev/null); do echo "$c"; descendants "$c"; done; }

  # ⑤孤儿锁扫杀：fuser 持fd集逐 pid 判 ppid=1（父死孤儿）→杀子孙+本体
  for lock in $LOCK_GLOB; do
    [ -e "$lock" ] || continue
    for pid in $(fuser "$lock" 2>/dev/null | tr -s ' ' '\n'); do
      [ -n "$pid" ] && [ "$pid" != "$$" ] || continue
      [ "$(ps -o ppid= -p "$pid" 2>/dev/null | tr -d ' ')" = "1" ] || continue
      kd=0
      if [ $DRY = 0 ]; then
        for c in $(descendants "$pid"); do kill "$c" 2>/dev/null && kd=$((kd+1)); done
        kill "$pid" 2>/dev/null && kd=$((kd+1))
      fi
      orphank=$((orphank+kd))
      echo "孤儿锁清扫$( [ "$DRY" = 1 ] && echo '[DRY]' ): $(basename "$lock") holder_pid=$pid（ppid=1 父死）killed=$kd"
      [ $DRY = 0 ] && ops_evt "孤儿锁清扫 lock=$(basename "$lock") holder_pid=$pid killed=$kd"
    done
  done

  # ①builder prune 双触发：周拍（mark>7d 或缺席——缺席按到期引导周律，
  # 不缺席永不触发=死闸）常规 -f；高水位（>80%）激进 -af 优先（双触同窗
  # 仍走 -af——高水位才是病灶档）。
  pct=$(rootfs_pct); pct=${pct:-0}
  week_hit=0
  [ $(( now - $(stat -c %Y "$PRUNE_MARK" 2>/dev/null || echo 0) )) -ge 604800 ] && week_hit=1
  if [ "$week_hit" = 1 ] || [ "$pct" -ge "${GC_ROOTFS_HIGH_PCT:-80}" ]; then
    prune_hit=1
    echo "builder prune 触发（week=$week_hit rootfs=${pct}% mark=$(basename "$PRUNE_MARK")）"
    if [ $DRY = 0 ]; then
      if [ "$pct" -ge "${GC_ROOTFS_HIGH_PCT:-80}" ]; then
        docker builder prune -af >/dev/null 2>&1 && echo "builder prune -af（高水位档）done" \
          || echo "⚠️ builder prune -af 败" >&2 # tolerate: prune 败=缓存残留下轮再收——保洁尽力件
      else
        docker builder prune -f >/dev/null 2>&1 && echo "builder prune -f（周拍档）done" \
          || echo "⚠️ builder prune -f 败" >&2 # tolerate: 同上
      fi
      touch "$PRUNE_MARK" 2>/dev/null || true # tolerate: mark 写败=下拍重触——幂等无害
    fi
  fi

  # ③go clean -cache 周拍（GCLEAN_MARK>7d 或缺席首跑；分家目录走下段三闸清）
  gclean_age=$(( now - $(stat -c %Y "$GCLEAN_MARK" 2>/dev/null || echo 0) ))
  if [ "$week_hit" = 1 ] || [ "$gclean_age" -ge 604800 ]; then
    gclean=1
    echo "go clean -cache 触发（周拍）"
    if [ $DRY = 0 ]; then
      go clean -cache >/dev/null 2>&1 || echo "⚠️ go clean -cache 败" >&2 # tolerate: clean 败=缓存续长下轮再清——保洁尽力件
      touch "$GCLEAN_MARK" 2>/dev/null || true # tolerate: mark 写败=下拍重触——幂等无害
    fi
  fi

  # ⑦GOCACHE 分家目录清（每拍）：活 checkout topkey 集外+对应锁未持+龄>GC_AGE_S
  LIVE_KEYS=$( { git -C "$REPO" worktree list --porcelain 2>/dev/null | awk '/^worktree/{print $2}'; printf '%s\n' "$REPO"; } \
    | while IFS= read -r w; do printf '%s\n' "$(realpath "$w" 2>/dev/null)" | sha256sum | cut -c1-16; done | sort -u )
  for d in $GOCACHE_GLOB; do
    [ -d "$d" ] || continue
    key=${d##*gocache-}
    printf '%s' "$key" | grep -qE '^[0-9a-f]{16}$' || continue # tolerate: 非键形目录不碰（gocache-m 类他物）
    printf '%s\n' "$LIVE_KEYS" | grep -qx "$key" && continue
    lk="/tmp/o-check_all-${key}.lock"
    [ -f "$lk" ] && [ -n "$(fuser "$lk" 2>/dev/null)" ] && continue
    [ $(( now - $(stat -c %Y "$d" 2>/dev/null || echo 0) )) -ge "$GC_AGE_S" ] || continue
    sz=$(du -sm "$d" 2>/dev/null | cut -f1)
    echo "GOCACHE 孤儿目录清: $d (${sz:-?}MB key=$key)"
    if [ $DRY = 0 ]; then rm -rf "$d" || continue; fi
    gck=$((gck+1))
  done
  HYNOTE="保洁拍 orphank=$orphank prune=$prune_hit goclean=$gclean gocache_rm=$gck rootfs=${pct}%"
  if [ "${1:-row}" = "fold" ]; then HYNOTE_FOLD="$HYNOTE"; else
    [ "$DRY" = 1 ] && echo "DRY-RUN 完 $HYNOTE" || ops_evt "$HYNOTE"
  fi
}

# --disk=只跑保洁面（deploy/o-disk-hygiene 30min 拍专用——工树/枝 GC
# 仍归日拍默认径）；默认径=工树 GC+保洁面 fold 进同一 GC 行。
if [ "$DISK" = 1 ]; then [ "${GC_HYGIENE:-1}" = 1 ] && disk_hygiene row; exit 0; fi

# ①工树扫：净树+(枝已合 master|mtime>AGE_D 天) 且 unit 非活→摘
while IFS='|' read -r w b; do
  [ "$w" = "/opt/o" ] || [ "$w" = "$REPO" ] && continue
  if [ ! -d "$w" ]; then echo "孤儿树: $w"; orphan=$((orphan+1)); continue; fi
  t=$(tid_norm "$w")
  # 双闸 A：unit 活=跳（o-worker-<TID> 在跑禁碰——TID 归一后比对在飞集）
  if [ -n "$t" ] && printf '%s\n' "$LIVE" | grep -Fxq "$t"; then
    echo "SKIP 活run: $w (unit o-worker-$t)"
    skip_active=$((skip_active+1)); continue
  fi
  # 脏树=跳
  if [ -n "$(git -C "$w" status --porcelain 2>/dev/null)" ]; then
    echo "SKIP 脏树: $w"; skip_dirty=$((skip_dirty+1)); continue
  fi
  mt=$(stat -c %Y "$w" 2>/dev/null || echo 0)
  age=$(( (now-mt)/86400 ))
  merged=0; git -C "$REPO" merge-base --is-ancestor "${b:-HEAD}" master 2>/dev/null && merged=1
  if [ "$merged" = 1 ] || [ "$age" -ge "$AGE_D" ]; then
    echo "摘树: $w (merged=$merged age=${age}d)"
    if [ $DRY = 0 ]; then
      git -C "$REPO" worktree remove "$w" --force 2>/dev/null \
        && rm_tree=$((rm_tree+1)) || echo "摘树败: $w"
    else rm_tree=$((rm_tree+1)); fi
  else
    echo "KEEP: $w (age=${age}d 未合)"; keep=$((keep+1))
  fi
done < <(git -C "$REPO" worktree list --porcelain |
         awk '/^worktree/{w=$2}/^branch/{gsub("refs/heads/","",$2); print w"|"$2}')

[ "$DRY" = 0 ] && [ "$orphan" -gt 0 ] && git -C "$REPO" worktree prune

# ②枝扫：已合 master→git branch -d（-d 天然拒删未合）
for b in $(git -C "$REPO" branch --list 't/*' --format='%(refname:short)'); do
  if git -C "$REPO" merge-base --is-ancestor "$b" master 2>/dev/null; then
    echo "删枝: $b（已合）"
    if [ $DRY = 0 ]; then
      git -C "$REPO" branch -d "$b" 2>/dev/null \
        && rm_branch=$((rm_branch+1)) || echo "删枝败: $b"
    else rm_branch=$((rm_branch+1)); fi
  fi
done

# ③退役支扫（T-0375 件④）：audit-*/dec-t-* 系——无 wt 挂+末 commit>RETIRE_D
# 即退役（-D 硬删——结论档在库,支体非审计载体）；带 wt/RETIRE_D 内不动
ATTACHED=$(git -C "$REPO" worktree list --porcelain |
  awk '/^branch/{gsub("refs/heads/","",$2); print $2}')
for b in $(git -C "$REPO" branch --list 'audit-*' 'dec-t-*' --format='%(refname:short)'); do
  if printf '%s\n' "$ATTACHED" | grep -Fxq "$b"; then
    echo "RETIRE-SKIP 带wt: $b"; retire_keep=$((retire_keep+1)); continue
  fi
  lc=$(git -C "$REPO" log -1 --format=%ct "$b" 2>/dev/null || echo "$now")
  age=$(( (now-lc)/86400 ))
  if [ "$age" -ge "$RETIRE_D" ]; then
    echo "退役枝: $b（末commit ${age}d 前——结论档在库,口径退役）"
    if [ $DRY = 0 ]; then
      git -C "$REPO" branch -D "$b" 2>/dev/null \
        && retire=$((retire+1)) || echo "退役败: $b"
    else retire=$((retire+1)); fi
  else
    echo "RETIRE-KEEP: $b（${age}d<${RETIRE_D}d）"; retire_keep=$((retire_keep+1))
  fi
done

[ "${GC_HYGIENE:-1}" = 1 ] && disk_hygiene fold   # T-0466：日拍默认径同挂保洁面（计数 fold 进本行）
NOTE="wt_gc 清场拍 trees=$rm_tree branches=$rm_branch retire=$retire orphan=$orphan skip_active=$skip_active skip_dirty=$skip_dirty keep=$keep retire_keep=$retire_keep ${HYNOTE_FOLD:-}"
[ "$DRY" = 1 ] && echo "DRY-RUN 完 $NOTE" || {
  [ -n "$FIN_DSN" ] && psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note) VALUES('META-GC','GC','T-0331','$NOTE')" >/dev/null 2>&1 || true
}
