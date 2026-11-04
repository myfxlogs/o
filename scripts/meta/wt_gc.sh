#!/usr/bin/env bash
# wt_gc.sh —— T-0331 worktree/t 分支 GC 器（钦定双闸：unit active+树龄
# 双查禁碰活 run 工树；删枝只用 -d 拒删未合枝天然闸；--dry-run 干跑开关
# 钦定；清场数 ops_event 'GC' 行录）。
set -u
DRY=0; [ "${1:-}" = "--dry-run" ] && DRY=1
ENVF="${GC_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
AGE_D=7
now=$(date +%s)
rm_tree=0; rm_branch=0; keep=0; skip_active=0; skip_dirty=0; orphan=0

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

# ①工树扫：净树+(枝已合 master|mtime>AGE_D 天) 且 unit 非活→摘
while IFS='|' read -r w b; do
  [ "$w" = "/opt/o" ] && continue
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
  merged=0; git -C /opt/o merge-base --is-ancestor "${b:-HEAD}" master 2>/dev/null && merged=1
  if [ "$merged" = 1 ] || [ "$age" -ge "$AGE_D" ]; then
    echo "摘树: $w (merged=$merged age=${age}d)"
    if [ $DRY = 0 ]; then
      git -C /opt/o worktree remove "$w" --force 2>/dev/null \
        && rm_tree=$((rm_tree+1)) || echo "摘树败: $w"
    else rm_tree=$((rm_tree+1)); fi
  else
    echo "KEEP: $w (age=${age}d 未合)"; keep=$((keep+1))
  fi
done < <(git -C /opt/o worktree list --porcelain |
         awk '/^worktree/{w=$2}/^branch/{gsub("refs/heads/","",$2); print w"|"$2}')

[ "$DRY" = 0 ] && [ "$orphan" -gt 0 ] && git -C /opt/o worktree prune

# ②枝扫：已合 master→git branch -d（-d 天然拒删未合）
for b in $(git -C /opt/o branch --list 't/*' --format='%(refname:short)'); do
  if git -C /opt/o merge-base --is-ancestor "$b" master 2>/dev/null; then
    echo "删枝: $b（已合）"
    if [ $DRY = 0 ]; then
      git -C /opt/o branch -d "$b" 2>/dev/null \
        && rm_branch=$((rm_branch+1)) || echo "删枝败: $b"
    else rm_branch=$((rm_branch+1)); fi
  fi
done

NOTE="wt_gc 清场拍 trees=$rm_tree branches=$rm_branch orphan=$orphan skip_active=$skip_active skip_dirty=$skip_dirty keep=$keep"
[ "$DRY" = 1 ] && echo "DRY-RUN 完 $NOTE" || {
  [ -n "$FIN_DSN" ] && psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note) VALUES('META-GC','GC','T-0331','$NOTE')" >/dev/null 2>&1 || true
}
