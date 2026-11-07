#!/usr/bin/env bash
# testdb_gc_test.sh —— T-0435 件①测试库卫生清扫集成桩测（票面验收②④）。
# 真库集成（O_TEST_PG_DSN 缺=skip——testutil 惯例；跑在租约库上=独占面）：
#   G0 生产守卫：GC_DSN 出族（端口 5999/db 名出白名单）→rc2 拒连零触碰
#      （守卫先于一切 psql；默认 FAMILY_PORTS 静态无 5433 生产口断言；
#      故意不拿 5433 实测——守卫破时测试本身不得成为生产触碰源）；
#   G5 dry-run：垃圾在位跑 --dry-run→零副作用（件全存）；
#   G1 陈旧事务注入（idle in tx 持 advisory lock——T-0426 病灶重放）→
#      --apply 回收断言（锁消失）；
#   G2 孤儿 schema 三类：带超龄 _gc_born→收；带新鲜标→豁免（在租保护
#      实证）；无标（T-0186 历史语义）→收；
#   G3 死元组注入（GC_DEAD_TUP_MIN 压阈）→VACUUM 动作断言；
#   G4 遗留库注入：零活连接→DROP；有活连接→保留（宁可漏）。
# 垃圾件命名 t_gcj_*$$ / o_test_gcjunk_$$（进程号隔离）+trap 清尾。
# 注：G1 持锁连接用 fifo 喂形——writer 子壳 pid 可 kill，无尾随挂起。
set -uo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
GC="$ROOT/scripts/gc_test_schemas.sh"
FAIL=0
ok()  { echo "  ✅ $1"; }
bad() { echo "  ❌ $1"; FAIL=1; }

if [ -z "${O_TEST_PG_DSN:-}" ]; then
  echo "⏭️  testdb_gc_test：O_TEST_PG_DSN 未设——skip（testutil 惯例，不打未租实例）"
  exit 0
fi
DSN="$O_TEST_PG_DSN"
PORT=""; [[ "$DSN" =~ :([0-9]+)/ ]] && PORT="${BASH_REMATCH[1]}"
SFX=$$
TMPD=$(mktemp -d)
Q() { psql "$DSN" -Atqc "$1"; }

cleanup() {
  psql "$DSN" -qc "DROP SCHEMA IF EXISTS t_gcj_mark_$SFX CASCADE"   >/dev/null 2>&1
  psql "$DSN" -qc "DROP SCHEMA IF EXISTS t_gcj_fresh_$SFX CASCADE"  >/dev/null 2>&1
  psql "$DSN" -qc "DROP SCHEMA IF EXISTS t_gcj_legacy_$SFX CASCADE" >/dev/null 2>&1
  psql "$DSN" -qc "DROP TABLE IF EXISTS t_gcj_dead_$SFX"            >/dev/null 2>&1
  psql "$DSN" -qc "DROP DATABASE IF EXISTS o_test_gcjunk_$SFX"      >/dev/null 2>&1
  psql "$DSN" -qc "DROP DATABASE IF EXISTS o_test_gcconn_$SFX"      >/dev/null 2>&1
  rm -rf "$TMPD"
}
trap cleanup EXIT

# ══ 注入垃圾件（schema 面+死元组面）══
Q "CREATE SCHEMA t_gcj_mark_$SFX"                                  >/dev/null
Q "CREATE TABLE t_gcj_mark_$SFX._gc_born(ts timestamptz NOT NULL)" >/dev/null
Q "INSERT INTO t_gcj_mark_$SFX._gc_born VALUES (now() - interval '25 hours')" >/dev/null
Q "CREATE SCHEMA t_gcj_fresh_$SFX"                                 >/dev/null
Q "CREATE TABLE t_gcj_fresh_$SFX._gc_born(ts timestamptz NOT NULL)" >/dev/null
Q "INSERT INTO t_gcj_fresh_$SFX._gc_born VALUES (now())"           >/dev/null
Q "CREATE SCHEMA t_gcj_legacy_$SFX"                                >/dev/null
Q "CREATE TABLE t_gcj_dead_$SFX(id int)"                           >/dev/null
Q "INSERT INTO t_gcj_dead_$SFX SELECT generate_series(1,300)"      >/dev/null
Q "DELETE FROM t_gcj_dead_$SFX"                                    >/dev/null

# ══ G0 生产守卫 ══
r0a=$(GC_DSN="postgres://o:o@localhost:5999/o_test?sslmode=disable" bash "$GC" 2>&1); rc0a=$?
r0b=$(GC_DSN="postgres://o:o@localhost:$PORT/o?sslmode=disable" bash "$GC" 2>&1); rc0b=$?
grep "^FAMILY_PORTS=" "$GC" | grep -q "5433" && G0C=1 || G0C=0
if [ "$rc0a" = 2 ] && grep -q "守卫拒" <<< "$r0a" \
   && [ "$rc0b" = 2 ] && grep -q "守卫拒" <<< "$r0b" && [ "$G0C" = 0 ]; then
  ok "G0 出族 DSN rc2 拒连×2（端口/db 双形）+默认族口无 5433 生产口"
else
  bad "G0 守卫形异常：rc0a=$rc0a rc0b=$rc0b 族口含5433=$G0C"
fi

# ══ G5 dry-run 零副作用 ══
r5=$(GC_DSN="$DSN" bash "$GC" --dry-run 2>&1)
n5=$(grep -c "\[dry\]" <<< "$r5")
f5=0
Q "SELECT 1 FROM pg_namespace WHERE nspname='t_gcj_mark_$SFX'"   | grep -q 1 || f5=1
Q "SELECT 1 FROM pg_namespace WHERE nspname='t_gcj_legacy_$SFX'" | grep -q 1 || f5=1
if [ "$f5" = 0 ] && [ "$n5" -ge 2 ]; then
  ok "G5 dry-run：注入件全存（零副作用）+动作行全带 [dry]（$n5 行）"
else
  bad "G5 形异常：存续残缺=$f5 dry行数=$n5"
fi

# ══ G1 陈旧事务（T-0426 病灶重放：idle in tx 持 advisory lock）══
mkfifo "$TMPD/f"
( exec 3>"$TMPD/f"; echo "BEGIN; SELECT pg_advisory_lock(869001);" >&3; sleep 20 ) &
WR=$!
psql "$DSN" -q -o /dev/null < "$TMPD/f" &
PS=$!
g1ready=0
for _ in $(seq 20); do
  [ "$(Q "SELECT count(*) FROM pg_locks WHERE locktype='advisory' AND objid=869001")" = "1" ] && { g1ready=1; break; }
  sleep 0.3
done
r1=$(GC_DSN="$DSN" GC_STALE_TX_MIN=0 bash "$GC" 2>&1); rc1=$?
g1gone=0
for _ in $(seq 10); do
  [ "$(Q "SELECT count(*) FROM pg_locks WHERE locktype='advisory' AND objid=869001")" = "0" ] && { g1gone=1; break; }
  sleep 0.3
done
kill "$WR" "$PS" 2>/dev/null; wait "$WR" "$PS" 2>/dev/null
if [ "$g1ready" = 1 ] && [ "$g1gone" = 1 ] && grep -q "陈旧事务回收" <<< "$r1"; then
  ok "G1 idle-in-tx 持 advisory lock 注入→terminate 回收（锁消失+留痕行）"
else
  bad "G1 形异常：注入就位=$g1ready 锁消失=$g1gone rc=$rc1"
fi

# ══ G2 孤儿 schema 三类（同 r1 apply 跑面）══
mark_gone=1; fresh_kept=0; legacy_gone=1
Q "SELECT 1 FROM pg_namespace WHERE nspname='t_gcj_mark_$SFX'"   | grep -q 1 && mark_gone=0
Q "SELECT 1 FROM pg_namespace WHERE nspname='t_gcj_fresh_$SFX'"  | grep -q 1 && fresh_kept=1
Q "SELECT 1 FROM pg_namespace WHERE nspname='t_gcj_legacy_$SFX'" | grep -q 1 && legacy_gone=0
grep -q "孤儿 schema 回收 t_gcj_mark_$SFX"   <<< "$r1" && mg=1 || mg=0
grep -q "无标 schema 回收 t_gcj_legacy_$SFX" <<< "$r1" && lg=1 || lg=0
if [ "$mark_gone" = 1 ] && [ "$fresh_kept" = 1 ] && [ "$legacy_gone" = 1 ] && [ "$mg" = 1 ] && [ "$lg" = 1 ]; then
  ok "G2 超龄标收+新鲜标豁免（在租保护）+无标收（T-0186 语义）+留痕行齐"
else
  bad "G2 形异常：mark收=$mark_gone fresh存=$fresh_kept legacy收=$legacy_gone 留痕m=$mg l=$lg"
fi

# ══ G3 死元组 VACUUM（fresh schema 仍存——G3 跑面同验豁免持续）══
dead_ready=0
for _ in $(seq 10); do
  [ "$(( $(Q "SELECT n_dead_tup FROM pg_stat_user_tables WHERE relname='t_gcj_dead_$SFX'") ))" -gt 0 ] && { dead_ready=1; break; }
  sleep 0.5
done
r3=$(GC_DSN="$DSN" GC_DEAD_TUP_MIN=1 bash "$GC" 2>&1)
if [ "$dead_ready" = 1 ] && grep -q "VACUUM (ANALYZE) 起" <<< "$r3" && grep -q "VACUUM 完成" <<< "$r3"; then
  ok "G3 死元组超阈→VACUUM (ANALYZE) 动作+前后留痕"
else
  bad "G3 形异常：dead就位=$dead_ready VACUUM行数=$(grep -c VACUUM <<< "$r3")"
fi

# ══ G4 遗留库：零连接收 / 有连接留 ══
Q "CREATE DATABASE o_test_gcjunk_$SFX" >/dev/null
Q "CREATE DATABASE o_test_gcconn_$SFX" >/dev/null
psql "postgres://o:o@localhost:$PORT/o_test_gcconn_$SFX?sslmode=disable" -qc "SELECT pg_sleep(8)" >/dev/null 2>&1 &
CONNPID=$!
sleep 0.5
r4=$(GC_DSN="$DSN" bash "$GC" 2>&1); rc4=$?
junk_gone=1; conn_kept=0
Q "SELECT 1 FROM pg_database WHERE datname='o_test_gcjunk_$SFX'" | grep -q 1 && junk_gone=0
Q "SELECT 1 FROM pg_database WHERE datname='o_test_gcconn_$SFX'" | grep -q 1 && conn_kept=1
wait "$CONNPID" 2>/dev/null
if [ "$junk_gone" = 1 ] && [ "$conn_kept" = 1 ] \
   && grep -q "遗留库回收 o_test_gcjunk_$SFX" <<< "$r4" \
   && grep -qE "o_test_gcconn_$SFX 有 [0-9]+ 活连接——跳过" <<< "$r4"; then
  ok "G4 零活连接遗留库 DROP+有活连接保留（宁可漏）+留痕行齐"
else
  bad "G4 形异常：junk收=$junk_gone conn存=$conn_kept rc=$rc4"
fi

[ "$FAIL" = 0 ] && echo "✅ testdb_gc_test 全场景过" || { echo "❌ testdb_gc_test 有红"; exit 1; }
