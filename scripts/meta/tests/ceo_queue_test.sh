#!/usr/bin/env bash
# ceo_queue_test.sh —— T-0502 CEO 滤队白名单补席桩测：真 PG scratch
# 库（test-pg 池端口）+自造 ops_event 最小表——全真 SQL 语义非桩面：
#   ① seat=研究员 升件入队（白名单补席生效）；非白名单席升件仍拒；
#   ② 裁决行落库（ts 更晚）→ 终态消队语义不破。
set -u
ROOT=$(cd "$(dirname "$0")/../../.." && pwd)
TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
pass=0; fail=0
ok(){ echo "PASS $1"; pass=$((pass+1)); }
bad(){ echo "FAIL $1"; fail=$((fail+1)); }

# ── 真 PG scratch 库（test-pg 池——独立库名隔离，case_writeback 同构）──
BASE_DSN="${O_TEST_PG_DSN:-postgres://o:o@localhost:5438/o_test?sslmode=disable}"
ADMIN_DSN=$(printf '%s' "$BASE_DSN" | sed 's|/o_test?|/postgres?|;s|/o_test$|/postgres|')
DBN="o_ceoq_t$$"
psql "$ADMIN_DSN" -qAtc "DROP DATABASE IF EXISTS $DBN WITH (FORCE)" >/dev/null 2>&1
psql "$ADMIN_DSN" -qAtc "CREATE DATABASE $DBN" >/dev/null 2>&1 \
  || { echo "SKIP: test-pg 不可达（$BASE_DSN）——ceo_queue 桩测跳过"; exit 0; }
trap 'timeout 60 psql "'"$ADMIN_DSN"'" -qAtc "DROP DATABASE IF EXISTS '"$DBN"' WITH (FORCE)" >/dev/null 2>&1; rm -rf "$TMP"' EXIT
DSN=$(printf '%s' "$BASE_DSN" | sed "s|/o_test?|/$DBN?|")
sqlq(){ psql "$DSN" -qAtc "$1"; }

psql "$DSN" -qAt <<'SQL' >/dev/null
CREATE TABLE ops_event(id bigserial PRIMARY KEY, ts timestamptz NOT NULL DEFAULT now(),
  seat text NOT NULL, event text NOT NULL, ticket text NOT NULL DEFAULT '', note text NOT NULL DEFAULT '');
SQL

runq(){ FIN_DSN="$DSN" bash "$ROOT/scripts/meta/ceo_queue.sh"; }

# ── 场景①：研究员升件入队 + 非白名单席对照仍拒 ──────────────────────
# ts 显式钉死——NOT EXISTS 终态判据走 t.ts>e.ts，序必须确定非 now() 竞态
sqlq "INSERT INTO ops_event(seat,event,ticket,note,ts) VALUES
  ('研究员','升件','VENUE-ALPHA','venue新族提案hypo#7','2026-01-01 00:00:00'),
  ('探针自检','升件','SHOULD-NOT-QUEUE','非白名单席对照','2026-01-01 00:00:00')" >/dev/null
Q1=$(runq)
grep -qxF 'VENUE-ALPHA' <<< "$Q1" \
  && ok "seat=研究员 升件入 CEO 队" || bad "研究员升件未入队（白名单未生效?）: $(printf '%s' "$Q1" | tr '\n' ',')"
grep -qxF 'SHOULD-NOT-QUEUE' <<< "$Q1" \
  && bad "非白名单席升件误入队（白名单放开过度?）" || ok "非白名单席升件仍拒"

# ── 场景②：裁决行落库 → 终态消队 ────────────────────────────────────
sqlq "INSERT INTO ops_event(seat,event,ticket,note,ts) VALUES
  ('管理席','裁决','VENUE-ALPHA','通过——族准入','2026-01-01 00:01:00')" >/dev/null
Q2=$(runq)
grep -qxF 'VENUE-ALPHA' <<< "$Q2" \
  && bad "裁决落库后仍滞留 CEO 队（终态消队破?）" || ok "裁决落库终态消队"

echo "ceo_queue scenarios: $pass pass / $fail fail"
[ "$fail" = 0 ]
