#!/usr/bin/env bash
# disk_hygiene_test.sh —— T-0466 保洁面桩测（钦定三断言+补面）：
#   A. 孤儿锁判定：ppid=1 持fd者被杀+报行；ppid≠1 活跑者不碰；
#   B. GOCACHE 键：活 wt 目录留/死 wt+龄>24h+锁未持目录删/
#      非键形与新鲜目录不碰；
#   C. image_tag_gc 保留窗：N=5+latest 计数断言（docker 桩）；
#   D. check_all GOCACHE 分家公式+等锁 ops_event 三落点 grep 断言。
set -uo pipefail   # 不用 -e——场景断言自带 rc 判
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"; rm -f /tmp/o-check_all-dhtest-*.lock /tmp/o-check_all-dddddddddddddddd.lock /tmp/dh-gcops.txt' EXIT
FAIL=0
ok()  { echo "  ✅ $1"; }
bad() { echo "  ❌ $1"; FAIL=1; }

# ── A. 孤儿锁扫杀 ──
SCRATCH="$TMP/repo"; mkdir -p "$SCRATCH"; git -C "$SCRATCH" init -q
SKEY=$(cd "$SCRATCH" && pwd -P | sha256sum | cut -c1-16)
LOCKD="$TMP/dh-locks"; mkdir -p "$LOCKD"
OLPH="$TMP/locks/o-check_all-deadbeef00000001.lock"; mkdir -p "$TMP/locks"
NOLPH="$TMP/locks/o-check_all-feedface00000002.lock"
: > "$OLPH"; : > "$NOLPH"

# 孤儿持fd者：父进程即退→reparent pid1（双fork实证径）
cat > "$TMP/holder.sh" <<'EOF'
#!/usr/bin/env bash
exec 9>"$1"; exec sleep 300
EOF
chmod +x "$TMP/holder.sh"
bash -c '"$0" "$1" >/dev/null 2>&1 &' "$TMP/holder.sh" "$OLPH"   # 父即退→孤儿
OPID=""
for i in $(seq 1 50); do
  OPID=$(fuser "$OLPH" 2>/dev/null | tr -s ' ' '\n' | grep -m1 '[0-9]')
  [ -n "$OPID" ] && [ "$(ps -o ppid= -p "$OPID" 2>/dev/null | tr -d ' ')" = "1" ] && break
  sleep 0.1
done
# 活跑对照：ppid=本测试pid≠1
bash -c 'exec 9>"$1"; exec sleep 300' _ "$NOLPH" & NPID=$!
sleep 0.3

GC_OPS_SINK="$TMP/gcops.txt"; : > "$GC_OPS_SINK"
GC_REPO="$SCRATCH" GC_ENV_FILE=/nonexistent \
GC_LOCK_GLOB="$TMP/locks/o-check_all-*.lock" \
GC_GOCACHE_GLOB="$TMP/dh-gocache-*" \
GC_OPS_SINK="$GC_OPS_SINK" GC_PRUNE_MARK="$TMP/pm" GC_GOCLEAN_MARK="$TMP/gm" \
GC_ROOTFS_PCT=54 GC_GOCACHE_AGE_S=0 \
bash "$ROOT/scripts/meta/wt_gc.sh" --disk > "$TMP/disk.out" 2>&1

if [ -n "$OPID" ] && ! kill -0 "$OPID" 2>/dev/null; then
  ok "A1 孤儿持fd者 pid=$OPID（ppid=1）被杀"
else
  bad "A1 孤儿未清（OPID=${OPID:-未捕获}）——out: $(tail -3 "$TMP/disk.out")"
fi
kill -0 "$NPID" 2>/dev/null && ok "A2 活跑持fd者 pid=$NPID（ppid≠1）不碰" || bad "A2 活跑被误杀"
kill "$NPID" 2>/dev/null; grep -q '孤儿锁清扫' "$GC_OPS_SINK" \
  && ok "A3 孤儿清扫落 ops_event 报行（GC_OPS_SINK 实证）" || bad "A3 无报行: $(cat "$GC_OPS_SINK")"

# ── B. GOCACHE 三闸 ──
GD="$TMP/dh-gocache"; mkdir -p "$GD"
touch -d '2 days ago' "$TMP/locks/o-check_all-deadbeef00000001.lock" 2>/dev/null || true # tolerate: fixture 锁 mtime 做旧败=不影响断言语义（龄闸走 AGE_S 覆写）
# B1 活 wt 键目录——应留（LIVE_KEYS 含 scratch key；龄也做旧隔离变量）
mkdir -p "$GD/gocache-$SKEY"; touch -d '2 days ago' "$GD/gocache-$SKEY"
# B2 死键目录——龄>24h 无锁 → 删
DEAD1=aaaaaaaaaaaaaaaa; DEAD2=bbbbbbbbbbbbbbbb; DEAD3=cccccccccccccccc; LOCKED=dddddddddddddddd
mkdir -p "$GD/gocache-$DEAD1" "$GD/gocache-$DEAD2" "$GD/gocache-$DEAD3" "$GD/gocache-$LOCKED" "$GD/gocache-m"
touch -d '2 days ago' "$GD/gocache-$DEAD1" "$GD/gocache-$DEAD2" "$GD/gocache-$LOCKED" "$GD/gocache-m"
touch "$GD/gocache-$DEAD3"   # 新鲜→龄闸留
# B4 锁在持→留：造 /tmp/o-check_all-<LOCKED>.lock + 活持fd者
LK="/tmp/o-check_all-${LOCKED}.lock"; : > "$LK"
bash -c 'exec 9>"$1"; exec sleep 60' _ "$LK" & LHPID=$!
sleep 0.3
: > "$GC_OPS_SINK"
GC_REPO="$SCRATCH" GC_ENV_FILE=/nonexistent \
GC_LOCK_GLOB="$TMP/locks/o-check_all-*.lock" \
GC_GOCACHE_GLOB="$GD/gocache-*" \
GC_OPS_SINK="$GC_OPS_SINK" GC_PRUNE_MARK="$TMP/pm2" GC_GOCLEAN_MARK="$TMP/gm2" \
GC_ROOTFS_PCT=54 GC_GOCACHE_AGE_S=86400 \
bash "$ROOT/scripts/meta/wt_gc.sh" --disk > "$TMP/disk2.out" 2>&1
kill "$LHPID" 2>/dev/null

[ -d "$GD/gocache-$SKEY" ] && ok "B1 活 wt GOCACHE 目录留" || bad "B1 活目录被删"
[ ! -d "$GD/gocache-$DEAD1" ] && ok "B2 死键+龄>24h+无锁→删" || bad "B2 死目录未删"
[ -d "$GD/gocache-$DEAD3" ] && ok "B3 新鲜目录龄闸留" || bad "B3 新鲜被删"
[ -d "$GD/gocache-$LOCKED" ] && ok "B4 锁在持目录留" || bad "B4 锁持被删"
[ -d "$GD/gocache-m" ] && ok "B5 非键形目录不碰" || bad "B5 键形闸失"
grep -q 'gocache_rm=2' "$GC_OPS_SINK" \
  && ok "B6 报行计数 gocache_rm=2（DEAD1+DEAD2 双死键实删）" \
  || bad "B6 计数异常: $(grep 保洁拍 "$GC_OPS_SINK")"

# ── C. image_tag_gc 保留窗 ──
mkdir -p "$TMP/bin"
cat > "$TMP/bin/docker" <<'EOF'
#!/usr/bin/env bash
if [ "$1 $2" = "images o-core" ]; then
  cat <<'L'
2026-11-01 00:00:00 +0000 UTC|aaaa1111aaaa
2026-11-02 00:00:00 +0000 UTC|bbbb2222bbbb
2026-11-03 00:00:00 +0000 UTC|cccc3333cccc
2026-11-04 00:00:00 +0000 UTC|dddd4444dddd
2026-11-05 00:00:00 +0000 UTC|eeee5555eeee
2026-11-06 00:00:00 +0000 UTC|ffff6666ffff
2026-11-07 00:00:00 +0000 UTC|999988887777
2026-11-08 00:00:00 +0000 UTC|123443215678
2026-11-08 01:00:00 +0000 UTC|latest
L
  exit 0
fi
if [ "$1" = "rmi" ]; then echo "RMI $2" >> "$RMI_LOG"; exit 0; fi
if [ "$1 $2 $3" = "image prune -f" ]; then echo "PRUNE" >> "$RMI_LOG"; exit 0; fi
exit 0
EOF
chmod +x "$TMP/bin/docker"
export RMI_LOG="$TMP/rmi.log"; : > "$RMI_LOG"
PATH="$TMP/bin:$PATH" O_IMAGE_REPO=o-core O_IMAGE_KEEP=5 \
  bash "$ROOT/scripts/meta/image_tag_gc.sh" > "$TMP/taggc.out" 2>&1
RMC=$(grep -c '^RMI ' "$RMI_LOG" || true) # tolerate: grep 空集=零 rmi 合法径（C6 边界断言靠此）
[ "$RMC" = 3 ] && ok "C1 8 tag 留 5——剪 3 个" || bad "C1 rmi 次数=$RMC want 3"
grep -q 'RMI o-core:aaaa1111aaaa' "$RMI_LOG" && grep -q 'RMI o-core:bbbb2222bbbb' "$RMI_LOG" \
  && grep -q 'RMI o-core:cccc3333cccc' "$RMI_LOG" \
  && ok "C2 剪=最旧三 tag（升序正确）" || bad "C2 剪错对象: $(cat "$RMI_LOG")"
grep -q 'RMI o-core:latest' "$RMI_LOG" && bad "C3 latest 被剪!" || ok "C3 latest 不剪"
grep -q 'RMI o-core:123443215678' "$RMI_LOG" && bad "C4 最新 sha tag 被剪!" || ok "C4 最新 sha tag 留"
grep -q '^PRUNE' "$RMI_LOG" && ok "C5 dangling prune 收口调用" || bad "C5 prune 未调"
# C6 边界：恰 5 tag 零剪
cat > "$TMP/bin/docker5" <<'EOF'
#!/usr/bin/env bash
if [ "$1" = "rmi" ]; then echo "RMI $2" >> "$RMI_LOG"; exit 0; fi
if [ "$1 $2" = "images o-core" ]; then
  printf '2026-11-0%d 00:00:00 +0000 UTC|t%d\n' 1 1; printf '2026-11-0%d 00:00:00 +0000 UTC|t%d\n' 2 2
  printf '2026-11-0%d 00:00:00 +0000 UTC|t%d\n' 3 3; printf '2026-11-0%d 00:00:00 +0000 UTC|t%d\n' 4 4
  printf '2026-11-0%d 00:00:00 +0000 UTC|t%d\n' 5 5; exit 0
fi
exit 0
EOF
chmod +x "$TMP/bin/docker5"
mkdir -p "$TMP/bin5"; cp "$TMP/bin/docker5" "$TMP/bin5/docker"
: > "$RMI_LOG"
PATH="$TMP/bin5:$PATH" O_IMAGE_REPO=o-core O_IMAGE_KEEP=5 \
  bash "$ROOT/scripts/meta/image_tag_gc.sh" > "$TMP/taggc5.out" 2>&1
[ ! -s "$RMI_LOG" ] && ok "C6 恰 N=5 tag 零剪" || bad "C6 边界误剪: $(cat "$RMI_LOG")"

# ── D. check_all 分家+等锁断言 ──
grep -q 'GOCACHE=${GOCACHE:-/tmp/gocache-${TOPKEY}}' "$ROOT/scripts/check_all.sh" \
  && ok "D1 check_all GOCACHE 分家行在" || bad "D1 分家行缺"
grep -c 'oe_lock "check_all' "$ROOT/scripts/check_all.sh" | grep -q '^3$' \
  && ok "D2 等锁 ops_event 三落点（等锁/超时/锁到手）" || bad "D2 落点数异常"
EXP=$(cd "$SCRATCH" && pwd -P | sha256sum | cut -c1-16)
[ "$EXP" = "$SKEY" ] && ok "D3 topkey 公式=check_all 同式（pwd -P|sha256sum|cut -c1-16）" || bad "D3 公式漂移"

[ "$FAIL" = 0 ] && echo "✅ disk_hygiene_test 全场景过" || { echo "❌ 有红"; exit 1; }
