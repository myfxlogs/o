#!/usr/bin/env bash
# testdb_shard_test.sh —— T-0468 族内分库桩测（判据：分域映射/锁等待观测/
# 裸 DSN 配对面/GC 卫生面/真 PG advisory 锁命名空间实证）。
#   G1 testutil.go 分库三件套在（O_TEST_SHARDS/shardForDir/ensureShardDB
#      +失败落基库注释——grep 钉死接线不漂移）；
#   G2 testdb_lease.sh acquire 后 O_TEST_SHARDS 默认 16 导出+外部预置
#      值尊重（${VAR:-16} 语义——不覆写人工档）；
#   G3 裸 DSN 配对面：writefail/pipeline_face 钉
#      cfg.ConnConfig.Database=testutil.ShardDatabase()（schema 在分库
#      不在基库=断言必败根修点）；pgguard 无 schema 语义裸池不动；
#   G4 gc_test_schemas.sh：库内枚举正则+守卫放宽 o_test_s<N>+扫④
#      分库豁免判名+扫④ 只在基库跑一次；
#   G5 真 PG 实证（可达才跑——断言不猜）：advisory 锁 860223 在
#      不同库互不阻塞=分库消 herd 的机制根；同库互斥不破；
#   G6 端到端：O_TEST_SHARDS=8 跑一包真测试→o_test_s<N> 库实建
#      （ensureShardDB 自确保径）+该包 schema 落分库非基库。
set -uo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
FAIL=0
ok()  { echo "  ✅ $1"; }
bad() { echo "  ❌ $1"; FAIL=1; }
TU="$ROOT/internal/platform/testutil/testutil.go"
LEASE="$ROOT/scripts/testdb_lease.sh"
GC="$ROOT/scripts/gc_test_schemas.sh"

echo "══ G1 testutil 分库接线钉死 ══"
for pat in 'O_TEST_SHARDS' 'shardForDir' 'ensureShardDB' 'ShardDatabase' 'o_test_s' '落基库'; do
  grep -q "$pat" "$TU" && ok "G1 testutil 含 $pat" || bad "G1 testutil 缺 $pat"
done

echo "══ G2 租约导出 O_TEST_SHARDS ══"
OUT=$(bash -c '
  TESTDB_LEASE_LOCKDIR="/tmp"; TESTDB_LEASE_WAIT=1
  source "'"$LEASE"'"
  testdb_lease_acquire "" >/dev/null 2>&1
  echo "def=${O_TEST_SHARDS:-NONE}"
' )
[ "$OUT" = "def=16" ] && ok "G2 缺省导出 O_TEST_SHARDS=16" || bad "G2 缺省档=$OUT（want def=16）"
OUT2=$(bash -c '
  TESTDB_LEASE_LOCKDIR="/tmp"; TESTDB_LEASE_WAIT=1; O_TEST_SHARDS=4
  source "'"$LEASE"'"
  testdb_lease_acquire "" >/dev/null 2>&1
  echo "pre=${O_TEST_SHARDS:-NONE}"
')
[ "$OUT2" = "pre=4" ] && ok "G2 外部预置 O_TEST_SHARDS=4 尊重不覆写" || bad "G2 预置档=$OUT2（want pre=4）"

echo "══ G3 裸 DSN 配对面 ══"
grep -q 'ConnConfig.Database = testutil.ShardDatabase()' "$ROOT/cmd/core/pipeline_face_test.go" \
  && ok "G3 pipeline_face 第二池钉分库" || bad "G3 pipeline_face 未钉 ShardDatabase"
grep -q 'ConnConfig.Database = testutil.ShardDatabase()' "$ROOT/internal/venue/onboard/writefail_test.go" \
  && ok "G3 writefail deny 池钉分库" || bad "G3 writefail 未钉 ShardDatabase"
grep -q 'testutil' "$ROOT/internal/venue/onboard/writefail_test.go" \
  && ok "G3 writefail testutil import 在" || bad "G3 writefail 缺 testutil import"

echo "══ G4 GC 卫生面 ══"
grep -q "o_test(_s\[0-9\]+)" "$GC" && ok "G4 GC 库内枚举 o_test 族正则在" || bad "G4 GC 枚举正则缺"
grep -q 'o_test_s\[0-9\]' "$GC" && grep -q '守卫拒' "$GC" \
  && ok "G4 GC 守卫放宽+拒文在" || bad "G4 GC 守卫面缺"
grep -q 'o_test_s\[0-9\]+\$.*continue' "$GC" \
  && ok "G4 GC 扫④分库豁免判名在" || bad "G4 GC 扫④豁免缺"
N4=$(grep -c 'if \[ "\$db" = "o_test" \]' "$GC")
[ "$N4" -ge 1 ] && ok "G4 GC 扫④基库单跑闸在" || bad "G4 GC 扫④单跑闸缺"

echo "══ G5 真 PG advisory 锁命名空间实证 ══"
# 用双探针库 o_test_s97/s98（命名域合法、用后即删）——不碰 o_test：
# 在途 check_all herd 持 860223@o_test 不干扰断言归属。
PORT="${G5_PORT:-5434}"
BASE="postgres://o:o@localhost:$PORT"
P1="o_test_s97"; P2="o_test_s98"
if psql "$BASE/postgres?sslmode=disable" -Atqc 'SELECT 1' >/dev/null 2>&1 \
   && psql "$BASE/o_test?sslmode=disable" -Atqc 'SELECT 1' >/dev/null 2>&1; then
  psql "$BASE/postgres?sslmode=disable" -qc "DROP DATABASE IF EXISTS $P1" >/dev/null 2>&1
  psql "$BASE/postgres?sslmode=disable" -qc "DROP DATABASE IF EXISTS $P2" >/dev/null 2>&1
  psql "$BASE/postgres?sslmode=disable" -qc "CREATE DATABASE $P1" >/dev/null 2>&1 \
    && psql "$BASE/postgres?sslmode=disable" -qc "CREATE DATABASE $P2" >/dev/null 2>&1 \
    && ok "G5 探针双库 $P1/$P2 已建" || bad "G5 探针库建败"
  # 跨库腿：A 持 P1 的 860223；B 取 P2 同键——应立即得（库隔离=herd 消机制根）
  psql "$BASE/$P1?sslmode=disable" -qc 'SELECT pg_advisory_lock(860223); SELECT pg_sleep(6);' >/dev/null 2>&1 &
  APID=$!; sleep 1
  T0=$(date +%s%N)
  psql "$BASE/$P2?sslmode=disable" -Atqc 'SELECT pg_advisory_lock(860223)' >/dev/null 2>&1
  T1=$(date +%s%N)
  EL=$(( (T1 - T0) / 1000000 ))
  wait $APID 2>/dev/null
  if [ "$EL" -lt 3000 ]; then
    ok "G5 跨库同键不互阻（${EL}ms<3000——advisory 锁按库隔离实证=herd 消机制根）"
  else
    bad "G5 跨库同键阻塞 ${EL}ms——锁域未按库隔离（机制根不成立）"
  fi
  # 同库腿：A 持 P1 的 860223；C 同库取——应阻塞至 lock_timeout（锁语义无损）
  psql "$BASE/$P1?sslmode=disable" -qc 'SELECT pg_advisory_lock(860223); SELECT pg_sleep(6);' >/dev/null 2>&1 &
  APID=$!; sleep 1
  T0=$(date +%s%N)
  psql "$BASE/$P1?sslmode=disable" -Atqc 'SET lock_timeout=2000; SELECT pg_advisory_lock(860223)' >/dev/null 2>&1
  RC=$?
  T1=$(date +%s%N); EL=$(( (T1 - T0) / 1000000 ))
  wait $APID 2>/dev/null
  if [ "$RC" -ne 0 ] && [ "$EL" -ge 1500 ]; then
    ok "G5 同库同键互斥不破（${EL}ms 阻塞至 lock_timeout——锁语义无损）"
  else
    bad "G5 同库同键未互斥（rc=$RC ${EL}ms）——锁语义破"
  fi
  psql "$BASE/postgres?sslmode=disable" -qc "DROP DATABASE IF EXISTS $P1" >/dev/null 2>&1
  psql "$BASE/postgres?sslmode=disable" -qc "DROP DATABASE IF EXISTS $P2" >/dev/null 2>&1 \
    && ok "G5 探针双库已回收" || bad "G5 探针库回收败（$P1/$P2 残留——手工清）"
else
  echo "  ⏭️  G5 PG $PORT 不可达——跳过（不猜不红；check_all 环境自带真跑）"
fi

echo "══ G6 端到端分库实证 ══"
DSN="${O_TEST_PG_DSN:-}"
if [ -n "$DSN" ] && [[ "$DSN" == *://* ]]; then
  MAINT="${DSN%/*}/postgres?sslmode=disable"
  # 期望分库=pgstore 包目录 fnv1a32 %8 +1（与 testutil.shardForDir 同式）
  EXPECT=$(python3 - "$ROOT/internal/store/pgstore" <<'PY'
import sys
h = 0x811c9dc5
for b in sys.argv[1].encode():
    h = ((h ^ b) * 0x01000193) & 0xFFFFFFFF
print(f"o_test_s{h % 8 + 1}")
PY
)
  (cd "$ROOT" && O_TEST_PG_DSN="$DSN" O_TEST_SHARDS=8 \
     go test -v ./internal/store/pgstore/ -run 'TestTrxStoreRoundtrip' -count=1 -timeout=300s) >"$TMP/g6.log" 2>&1
  RC=$?
  if [ "$RC" -eq 0 ] && ! grep -q '落基库' "$TMP/g6.log"; then
    ok "G6 pgstore TestTrxStoreRoundtrip O_TEST_SHARDS=8 绿且零落基库回退"
  else
    bad "G6 pgstore 异常 rc=$RC 落基库=$(grep -c '落基库' "$TMP/g6.log")——$(tail -3 "$TMP/g6.log" | tr '\n' ' ')"
  fi
  HAS=$(psql "$MAINT" -Atqc "SELECT 1 FROM pg_database WHERE datname='$EXPECT'" 2>/dev/null)
  [ "$HAS" = "1" ] && ok "G6 期望分库 $EXPECT 实建在案（ensureShardDB 自确保径通）" \
    || bad "G6 期望分库 $EXPECT 不存在——ensureShardDB 未触发"
else
  echo "  ⏭️  G6 O_TEST_PG_DSN 缺/非 URL 形——跳过（check_all 环境自带真跑）"
fi

echo "────────────────────────────"
if [ "$FAIL" = 0 ]; then echo "✅ testdb_shard_test 全场景过"; else echo "❌ testdb_shard_test 有红"; fi
exit "$FAIL"
