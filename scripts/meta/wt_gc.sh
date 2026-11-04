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
rm_tree=0; rm_branch=0; keep=0; skip=0

# ①工树扫：净树+(枝已合 master|mtime>AGE_D 天) 且 unit 非活→摘
git -C /opt/o worktree list --porcelain | awk '/^worktree/{w=$2}/^branch/{gsub("refs/heads/","",$2); print w"|"$2}' |
while IFS='|' read -r w b; do
  [ "$w" = "/opt/o" ] && continue
  [ -d "$w" ] || continue
  tid=$(basename "$w" | tr 'A-Z' 'a-z')
  # 双闸 A：unit 活=跳（o-worker-<tid> 在跑禁碰）
  if systemctl --user is-active "o-worker-$tid.service" 2>/dev/null | grep -qx 'active\|activating'; then
    echo "SKIP 活run: $w"; continue
  fi
  # 脏树=跳
  if [ -n "$(git -C "$w" status --porcelain 2>/dev/null)" ]; then
    echo "SKIP 脏树: $w"; continue
  fi
  mt=$(stat -c %Y "$w" 2>/dev/null || echo 0)
  age=$(( (now-mt)/86400 ))
  merged=0; git -C /opt/o merge-base --is-ancestor "${b:-HEAD}" master 2>/dev/null && merged=1
  if [ "$merged" = 1 ] || [ "$age" -ge "$AGE_D" ]; then
    echo "摘树: $w (merged=$merged age=${age}d)"
    [ $DRY = 0 ] && git -C /opt/o worktree remove "$w" --force 2>/dev/null
  else
    echo "KEEP: $w (age=${age}d 未合)"
  fi
done

# ②枝扫：已合 master→git branch -d（-d 天然拒删未合）
for b in $(git -C /opt/o branch --list 't/*' --format='%(refname:short)'); do
  if git -C /opt/o merge-base --is-ancestor "$b" master 2>/dev/null; then
    echo "删枝: $b（已合）"
    [ $DRY = 0 ] && git -C /opt/o branch -d "$b" 2>/dev/null
  fi
done

[ "$DRY" = 1 ] && echo "DRY-RUN 完" || {
  [ -n "$FIN_DSN" ] && psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note) VALUES('META-GC','GC','T-0331','wt_gc 清场拍')" >/dev/null 2>&1 || true
}
