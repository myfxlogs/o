#!/usr/bin/env bash
# ticket_seq_test.sh —— 票号原子发号桩测（提案-票号原子发号）：
# 真 PG scratch 库（test-pg 池/外部 DSN 径）+真 git 仓 fixture 自造。
# 场景：
#   ① 序列径优先——预置 last_value 高于 repo_max → next_ticket 出序列号
#     （扫号径本会出 repo_max+1，异值=序列径实证）；
#   ② 地板自校正——last_value 低于 repo_max → setval 地板抬升后发号
#     ≥repo_max+1（种子漂移自愈，不回跳不烧旧号）；
#   ③ 并发两席取号→异号（nextval 原子性实证）；
#   ④ DSN 不可达→fail-open 回扫号径行为不变；
#   ⑤ --seq 对账：last_value<repo_max 红 / ≥绿 / 序列缺席=半成品窗跳过。
set -u
ROOT=$(cd "$(dirname "$0")/../../.." && pwd)
TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
pass=0; fail=0
ok(){ echo "PASS $1"; pass=$((pass+1)); }
bad(){ echo "FAIL $1"; fail=$((fail+1)); }

# ── ① scratch PG（test-pg 池——o_test_s* 分片同池，独立库名隔离）──
BASE_DSN="${O_TEST_PG_DSN:-postgres://o:o@localhost:5438/o_test?sslmode=disable}"
ADMIN_DSN=$(printf '%s' "$BASE_DSN" | sed 's|/o_test?|/postgres?|;s|/o_test$|/postgres|')
DBN="o_tseq_t$$"
psql "$ADMIN_DSN" -qAtc "DROP DATABASE IF EXISTS $DBN WITH (FORCE)" >/dev/null 2>&1
psql "$ADMIN_DSN" -qAtc "CREATE DATABASE $DBN" >/dev/null 2>&1 \
  || { echo "SKIP: test-pg 不可达（$BASE_DSN）——票号序列桩测跳过"; exit 0; }
trap 'timeout 60 psql "'"$ADMIN_DSN"'" -qAtc "DROP DATABASE IF EXISTS '"$DBN"' WITH (FORCE)" >/dev/null 2>&1; rm -rf "$TMP"' EXIT
DSN=$(printf '%s' "$BASE_DSN" | sed "s|/o_test?|/$DBN?|")
psql "$DSN" -qAt <<'SQL' >/dev/null
CREATE TABLE ops_event(id bigserial PRIMARY KEY, ts timestamptz NOT NULL DEFAULT now(),
  seat text NOT NULL, event text NOT NULL, ticket text NOT NULL DEFAULT '', note text NOT NULL DEFAULT '');
CREATE SEQUENCE fin_ticket_seq START WITH 100;
SQL
sqlq(){ psql "$DSN" -qAtc "$1"; }

# ── ② 临时 git 仓 fixture：docs/tasks 两面票件 max=0499 ──────────
GITR="$TMP/repo"; mkdir -p "$GITR/docs/tasks/done"; cd "$GITR" || exit 1
git init -qb master >/dev/null; git config user.email t@t; git config user.name t
echo x > docs/tasks/T-0499-alpha.md; echo x > docs/tasks/done/T-0480-beta.md
git add -A; git commit -qm 'fixture tickets'
NT="$ROOT/scripts/next_ticket.sh"; CTN="$ROOT/scripts/check_ticket_numbers.sh"
printf 'FIN_PG_DSN=%s\n' "$DSN" > "$TMP/env"
export DM_ENV_FILE="$TMP/env"

# ── ③ 序列径优先：last_value=100<repo_max=499→地板抬至499→发500 ──
OUT=$(cd "$GITR" && bash "$NT" 测试席 2>/dev/null)
[ "$OUT" = "T-0500" ] && ok "① 序列径+地板自校正→T-0500（repo_max+1）" || bad "① 出=$OUT 期=T-0500"
# 序列抬高后：setval 700→nextval=701（扫号径本会出 500——异值=序列径实证）
sqlq "SELECT setval('fin_ticket_seq',700,true)" >/dev/null
OUT=$(cd "$GITR" && bash "$NT" 测试席 2>/dev/null)
[ "$OUT" = "T-0701" ] && ok "② 序列高位径→T-0701（扫号径盲区实证）" || bad "② 出=$OUT 期=T-0701"
# ── ④ 并发两席取号→异号（后台子 shell 变量不回传——落文件再收）──
(cd "$GITR" && bash "$NT" 席A 2>/dev/null > "$TMP/oa") &
(cd "$GITR" && bash "$NT" 席B 2>/dev/null > "$TMP/ob") &
wait
OA=$(cat "$TMP/oa"); OB=$(cat "$TMP/ob")
[ -n "$OA" ] && [ -n "$OB" ] && [ "$OA" != "$OB" ] \
  && ok "③ 并发取号异号（$OA vs $OB）" || bad "③ 并发同号/空（A=$OA B=$OB）"
# ── ⑤ DSN 不可达→fail-open 回扫号径 ──
printf 'FIN_PG_DSN=postgres://bad:bad@127.0.0.1:1/x?sslmode=disable\n' > "$TMP/envbad"
DM_ENV_FILE="$TMP/envbad" bash -c "cd '$GITR' && bash '$NT'" 2>/dev/null > "$TMP/outbad"
OUT=$(cat "$TMP/outbad")
[ "$OUT" = "T-0500" ] && ok "④ DSN 败→扫号径回退 T-0500 行为不变" || bad "④ 出=$OUT 期=T-0500"
# ── ⑥ --seq 对账：落后红/达标绿/缺席跳过 ──
sqlq "SELECT setval('fin_ticket_seq',100,true)" >/dev/null
cd "$GITR" && DM_ENV_FILE="$TMP/env" bash "$CTN" --seq >/dev/null 2>&1; rc=$?
[ "$rc" -eq 1 ] && ok "⑤a last_value=100<max499→对账红" || bad "⑤a rc=$rc 期=1"
sqlq "SELECT setval('fin_ticket_seq',600,true)" >/dev/null
cd "$GITR" && DM_ENV_FILE="$TMP/env" bash "$CTN" --seq >/dev/null 2>&1; rc=$?
[ "$rc" -eq 0 ] && ok "⑤b last_value=600≥499→对账绿" || bad "⑤b rc=$rc 期=0"
sqlq "DROP SEQUENCE fin_ticket_seq" >/dev/null
cd "$GITR" && DM_ENV_FILE="$TMP/env" bash "$CTN" --seq >/dev/null 2>&1; rc=$?
[ "$rc" -eq 0 ] && ok "⑤c 序列缺席（迁移未应用）→半成品窗跳过" || bad "⑤c rc=$rc 期=0"
# ── ⑦ 领号台账行随票落账 ──
sqlq "CREATE SEQUENCE fin_ticket_seq START WITH 800" >/dev/null
cd "$GITR" && DM_ENV_FILE="$TMP/env" bash "$NT" 台账席 >/dev/null 2>&1
sqlq "SELECT ticket FROM ops_event WHERE event='领号' ORDER BY id DESC LIMIT 1" | grep -q '^T-0801$' \
  && ok "⑥ 领号行 T-0801 落账" || bad "⑥ 领号行缺席/异值"

echo "─────"
if [ "$fail" = 0 ]; then echo "✅ ticket_seq 桩测 $pass/$((pass+fail)) 全绿"; exit 0
else echo "❌ 桩测红：$fail 败/$((pass+fail))"; exit 1; fi
