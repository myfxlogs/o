#!/usr/bin/env bash
# ceo_queue_test.sh —— T-0502 CEO 滤队白名单补席桩测：真 PG scratch
# 库（test-pg 池端口）+自造 ops_event 最小表——全真 SQL 语义非桩面：
#   ① seat=研究员 升件入队（白名单补席生效）；非白名单席升件仍拒；
#   ② 裁决行落库（ts 更晚）→ 终态消队语义不破；
#   ③ T-0509：seat=MECH 票外呈批 ticket='' 以「呈批#<id>」入队；
#     引证裁决消队；非 MECH 票外呈批不入队。
#   ⑥ T-0507（返修2 和解后）：MECH+层扫 digest 入队（事件级白名单
#     收窄——MECH 席集仅呈批）；MECH+升件 仍拒（机械升件留 CTO 壳轨）；
#     处置 ticket='层扫' 终态消队；新 digest 再入队。
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

# ── 场景③：T-0509 白名单呈批——seat=MECH 呈批 ticket='' 票外件以
#    「呈批#<id>」合成行入队（席位集补席+票外呈批可见性生效）──
RID=$(sqlq "INSERT INTO ops_event(seat,event,ticket,note,ts) VALUES
  ('MECH','呈批','','表外动作呈批：规则#1:T-0500——白名单表外触发呈裁','2026-01-01 00:02:00') RETURNING id")
Q3=$(runq)
grep -qxF "呈批#$RID" <<< "$Q3" \
  && ok "seat=MECH 票外呈批以呈批#id 入 CEO 队" || bad "MECH 呈批未入队（席位集/票外呈批分支漏?）: $(printf '%s' "$Q3" | tr '\n' ',')"

# ── 场景④：裁决行 note 引证「呈批#id」→ 终态消队 ──────────────────
sqlq "INSERT INTO ops_event(seat,event,ticket,note,ts) VALUES
  ('CEO','裁决','','呈批#$RID 批准——规则#1 翻生效走表更径','2026-01-01 00:03:00')" >/dev/null
Q4=$(runq)
grep -qxF "呈批#$RID" <<< "$Q4" \
  && bad "引证裁决落库后呈批仍滞留（呈批#id 消队破?）" || ok "呈批#id 引证裁决消队"

# ── 场景⑤：非 MECH 席票外呈批不入队（合成分支席域钉死）────────────
sqlq "INSERT INTO ops_event(seat,event,ticket,note,ts) VALUES
  ('CEO','呈批','','业主呈批——非白名单呈批对照','2026-01-01 00:04:00')" >/dev/null
Q5=$(runq)
[ -z "$(printf '%s' "$Q5" | tr -d '[:space:]')" ] \
  && ok "非 MECH 票外呈批不入队（席域未放开）" || bad "非 MECH 呈批误入队: $(printf '%s' "$Q5" | tr '\n' ',')"

# ── 场景⑥（T-0507）：MECH+层扫 digest 入队；MECH+升件 仍拒 ─────────
# （返修2 和解后 MECH 席集收窄=仅呈批——层扫例外支叠加共存）
sqlq "INSERT INTO ops_event(seat,event,ticket,note,ts) VALUES
  ('MECH','层扫','层扫','层扫 draft=0 approved=0 dispatched=0 pulled=0 working=0 review=0 verdicted=0 done=0 rejected=0 suspended=0 external=0 error=0 __tie__=0｜异常 queue=0 entity=0 capacity=0 ledger=0 external=0｜unit=ok cap=ok ledger_face=ok tie=0','2026-01-01 00:05:00'),
  ('MECH','升件','T-9999','层扫定性dispatched-无拉:T-9999——相位dispatched 龄66m','2026-01-01 00:05:00')" >/dev/null
Q6=$(runq)
grep -qxF '层扫' <<< "$Q6" \
  && ok "MECH+层扫 digest 入 CEO 队" || bad "层扫 digest 未入队（T-0507 白名单支破?）: $(printf '%s' "$Q6" | tr '\n' ',')"
grep -qxF 'T-9999' <<< "$Q6" \
  && bad "MECH+升件误入 CEO 队（和解收窄后事件级白名单放开过度?）" || ok "MECH+升件仍留 CTO 壳轨"

# ── 场景⑦：处置 ticket='层扫' → digest 终态消队；新 digest 再入队 ──
sqlq "INSERT INTO ops_event(seat,event,ticket,note,ts) VALUES
  ('CEO','处置','层扫','体检读毕——全绿','2026-01-01 00:06:00')" >/dev/null
Q7=$(runq)
grep -qxF '层扫' <<< "$Q7" \
  && bad "处置落库后层扫 digest 仍滞留（终态消队破?）" || ok "处置落库 digest 终态消队"
sqlq "INSERT INTO ops_event(seat,event,ticket,note,ts) VALUES
  ('MECH','层扫','层扫','层扫 draft=0 approved=0 dispatched=0 pulled=0 working=0 review=0 verdicted=0 done=0 rejected=0 suspended=0 external=0 error=0 __tie__=0｜异常 queue=1 entity=0 capacity=0 ledger=0 external=0｜unit=ok cap=ok ledger_face=ok tie=0','2026-01-01 00:07:00')" >/dev/null
Q8=$(runq)
grep -qxF '层扫' <<< "$Q8" \
  && ok "新 digest 再入队（单槽最新体检行语义）" || bad "新 digest 未再入队: $(printf '%s' "$Q8" | tr '\n' ',')"

echo "ceo_queue scenarios: $pass pass / $fail fail"
[ "$fail" = 0 ]
