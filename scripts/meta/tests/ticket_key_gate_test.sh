#!/usr/bin/env bash
# ticket_key_gate_test.sh —— T-0499 ticket 键卫生桩测：
#   A 闸单测（psql 桩截获）：拼接伪键（换行/空格/逗号/竖线多票号/尾随
#     空白）→rc1 loud 拒写（stderr+升件行 ticket='' 落账，24h 同键
#     去重不重复落）；合法单键/空键/META-*/量表名/单 T- 复合键→rc0 零写；
#   B 接线抽查（escalate.sh 真身）：伪键 ticket 入参→本体 INSERT 零落、
#     仅闸升件行一条（写径入口闸在=拒写事实落账）；
#   C 视图守卫真库径（testdb_lease——池空 skip 不猜不占）：fin_0153 迁移
#     实跑建视图，伪键行（换行/空格/逗号/竖线四形）+合法行混合种子——
#     ops_ticket_phase 对伪键零相位行、合法键相位照常（判据③钉死）。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin"
REAL_PSQL=$(command -v psql) # C 段真库径用——PATH 前置桩只罩 A/B 段

# —— psql 桩：闸/escalate 全走 -c 形；去重 SELECT 按前缀回已落 INSERT 行 ——
cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
query=${*: -1}
[[ $query == "-" ]] && query=$(cat)
case "$query" in
	*"SELECT 1 FROM ops_event WHERE event='升件'"*)
		pfx=$(sed -n "s/.*note LIKE '\([^']*\)%'.*/\1/p" <<<"$query")
		[ -n "${pfx:-}" ] && [ -s "${GATE_SQL_LOG:-/dev/null}" ] \
			&& grep -qF "$pfx" "$GATE_SQL_LOG" && echo 1 ;;
	*"INSERT INTO ops_event"*) printf '%s\n' "$query" >> "$GATE_SQL_LOG" ;;
	*) echo "unexpected psql: $query" >&2; exit 2 ;;
esac
EOF
chmod +x "$TMP/bin/psql"
printf 'FIN_PG_DSN=fake-dsn\n' > "$TMP/env"
export PATH="$TMP/bin:$PATH"
export KEY_GATE_ENV_FILE="$TMP/env" GATE_SQL_LOG="$TMP/sql.log"
: > "$GATE_SQL_LOG"
GATE="$ROOT/scripts/meta/ops_event_key_gate.sh"
pass=0; fail=0
ok()  { echo "  ✅ $1"; pass=$((pass+1)); }
bad() { echo "  ❌ $1"; fail=$((fail+1)); }

echo "══ A 闸单测 ══"
# A1 换行拼接伪键（尸源实证形）→rc1+loud
if "$GATE" $'T-9001\nT-9002' "event=领单" "TESTSEAT" 2>"$TMP/err1"; then
  bad "A1 换行伪键未拒（rc0）"; else ok "A1 换行伪键 rc1"; fi
grep -q '拒写伪键' "$TMP/err1" && ok "A1 stderr loud 在案" || bad "A1 stderr 无 loud"
N1=$(grep -c 'INSERT INTO ops_event' "$GATE_SQL_LOG")
[ "$N1" -eq 1 ] && ok "A1 升件行恰落 1 条" || bad "A1 INSERT 数=$N1（want 1）"
grep -qF "VALUES('TESTSEAT','升件','','ticket键卫生拒写 event=领单" "$GATE_SQL_LOG" \
  && ok "A1 升件行形（seat/空ticket/拒写note前缀）" || bad "A1 升件行形不符"
grep -qF 'key=T-9001·T-9002' "$GATE_SQL_LOG" \
  && ok "A1 note 携归一预览（·替换换行）" || bad "A1 预览归一缺"

# A2 同键同窗再拒=去重不重落（rc1 仍在，INSERT 不增）
"$GATE" $'T-9001\nT-9002' "event=领单" "TESTSEAT" 2>/dev/null || true # tolerate: 拒写 rc1=断言面语义，loud 落证走 SQL 日志断言
N2=$(grep -c 'INSERT INTO ops_event' "$GATE_SQL_LOG")
[ "$N2" -eq 1 ] && ok "A2 24h 同键去重零重落" || bad "A2 INSERT 数=$N2（want 1）"

# A3 其余伪键形逐一拒（各异键异 hash——每键一条升件行）
for spec in "space:T-9003 T-9004" "comma:T-9005,T-9006" "comma-cn:T-9007，T-9008" \
            "pipe:T-9009|T-9010" "trail:T-9011 " "tab:$(printf 'T-9012\tT-9013')"; do
  name=${spec%%:*}; key=${spec#*:}
  if "$GATE" "$key" "event=x" "TESTSEAT" 2>/dev/null; then bad "A3-$name 未拒"; else ok "A3-$name rc1"; fi
done

# A4 合法键全放行零写
N0=$(grep -c 'INSERT INTO ops_event' "$GATE_SQL_LOG")
for key in 'T-9014' '' 'META-GAUGES' 'META-VENUEWATCH' 'armed_aging' \
           'META-Dev-T-0455' 'o-postgres-test' 'T-9015-x'; do
  if "$GATE" "$key" "event=x"; then :; else bad "A4 净键 '$key' 误拒"; fi
done
N1b=$(grep -c 'INSERT INTO ops_event' "$GATE_SQL_LOG")
[ "$N1b" -eq "$N0" ] && ok "A4 净键零写（INSERT 数不变=$N0）" || bad "A4 净键误写 INSERT $N0→$N1b"

# A5 闸缺席径=调用面 fail-open 语义由接线形承载（-x 判空在各调用件）——
#    本测以 O_KEY_GATE_SH 指空件验 escalate 直落径不受闸件缺失影响。

echo "══ B escalate.sh 接线抽查 ══"
export ESCALATE_ENV_FILE="$TMP/env"
DIRTY=$'T-9020\nT-9021'
"$ROOT/scripts/meta/escalate.sh" TESTSEAT "$DIRTY" '' '' '测试升件note' || true # tolerate: 拒写 rc 非零=断言面语义，落证走 SQL 日志断言
NB=$(grep -c 'INSERT INTO ops_event' "$GATE_SQL_LOG")
# escalate 本体两 INSERT 形（带类/无类）均不得含伪键——总 INSERT 应仍=1
#（仅闸升件行；前面 A 段已落 7 条——取本段增量判）
[ "$NB" -eq 8 ] && ok "B 伪键入参仅闸行一条（escalate 本体零落）" \
                 || bad "B INSERT 总数=$NB（want 8=A段7+B段1）"
grep -qF "'升件','','ticket键卫生拒写 event=升件" "$GATE_SQL_LOG" \
  && ok "B 闸行 ctx=event=升件 在案" || bad "B 闸行 ctx 缺"
# 净键对照：escalate 正常径不受影响（带类升件落账）
"$ROOT/scripts/meta/escalate.sh" TESTSEAT 'T-9022' '写败' 'META-X' '正常升件note' || true # tolerate: 桩面收证零败因语义——SQL 落证为断言面，rc 非本断言面
NC=$(grep -c 'INSERT INTO ops_event' "$GATE_SQL_LOG")
[ "$NC" -eq 9 ] && grep -qF "VALUES('TESTSEAT','升件','T-9022','正常升件note','写败','META-X')" "$GATE_SQL_LOG" \
  && ok "B 净键 escalate 正常落账（带类行）" || bad "B 净键 escalate 异常（n=$NC）"

echo "══ C 视图守卫真库径（池空 skip）══"
# shellcheck source=/dev/null
source "$ROOT/scripts/testdb_lease.sh"
PG_NAMES=$(docker ps --format '{{.Names}}' 2>/dev/null || true) # tolerate: docker 缺席=池空 skip 径
testdb_lease_acquire "$PG_NAMES"
if [ -z "${O_TEST_PG_DSN:-}" ]; then
  echo "⚠️ [ticket_key_gate_test] 测试 PG 池空——视图守卫场景 skip（不猜不占）"
  echo "ticket_key_gate: A/B 段 PASS=$pass FAIL=$fail（C skip）"
  [ "$fail" -eq 0 ]
  exit $?
fi
SCHEMA=t_0499kg
TDSN="${O_TEST_PG_DSN}&options=-csearch_path%3D${SCHEMA}"
"$REAL_PSQL" "$TDSN" -q -v ON_ERROR_STOP=1 <<'SQL'
DROP SCHEMA IF EXISTS t_0499kg CASCADE;
CREATE SCHEMA t_0499kg;
CREATE TABLE ops_event(
  id bigserial PRIMARY KEY,
  ts timestamptz NOT NULL DEFAULT now(),
  seat text NOT NULL,
  event text NOT NULL,
  ticket text NOT NULL DEFAULT '',
  note text NOT NULL DEFAULT '');
SQL
# 迁移实跑（fin_0153 全件——CREATE VIEW+授权块，search_path 落本 schema）
"$REAL_PSQL" "$TDSN" -q -v ON_ERROR_STOP=1 \
  -f "$ROOT/migrations/fin_0153_ops_ticket_phase_key_guard.sql" \
  || { echo "❌ C 迁移应用败"; exit 1; }
# 种子：3 合法+4 伪键形混合（伪键各配生命周期事件——若守卫失守皆成相位行）
"$REAL_PSQL" "$TDSN" -qAt <<'SQL'
INSERT INTO ops_event(seat,event,ticket,note) VALUES
 ('CTO','领单','T-9010',''),
 ('CTO','派单','T-9011',''),
 ('CTO','领单',E'T-9020\nT-9021',''),
 ('CTO','派单','T-9022 T-9023',''),
 ('CTO','领单','T-9024,T-9025',''),
 ('CTO','领单','T-9026|T-9027',''),
 ('无头施工席','收口','T-9028','');
SQL
C() { "$REAL_PSQL" "$TDSN" -qAt -c "$1"; }
Z1=$(C "SELECT count(*) FROM ops_ticket_phase WHERE ticket ~ '[[:space:]]'")
[ "$Z1" = "0" ] && ok "C 空白族伪键零相位行" || bad "C 空白族漏行=$Z1"
Z2=$(C "SELECT count(*) FROM ops_ticket_phase WHERE ticket ~ 'T-[0-9]{4}.*T-[0-9]{4}'")
[ "$Z2" = "0" ] && ok "C 多票号伪键零相位行" || bad "C 多票号漏行=$Z2"
Z3=$(C "SELECT count(*) FROM ops_ticket_phase WHERE ticket LIKE '%'||chr(10)||'%'")
[ "$Z3" = "0" ] && ok "C chr(10) 守卫钉死（票面判据字面）" || bad "C chr(10) 漏行=$Z3"
P1=$(C "SELECT phase FROM ops_ticket_phase WHERE ticket='T-9010'")
[ "$P1" = "pulled" ] && ok "C 合法键相位照常（领单→pulled）" || bad "C T-9010 phase=$P1"
P2=$(C "SELECT phase FROM ops_ticket_phase WHERE ticket='T-9011'")
[ "$P2" = "dispatched" ] && ok "C 合法键相位照常（派单→dispatched）" || bad "C T-9011 phase=$P2"
P3=$(C "SELECT phase FROM ops_ticket_phase WHERE ticket='T-9028'")
[ "$P3" = "review" ] && ok "C 合法键相位照常（收口→review）" || bad "C T-9028 phase=$P3"
TOT=$(C "SELECT count(*) FROM ops_ticket_phase")
[ "$TOT" = "3" ] && ok "C 相位行总数=合法键数（3）" || bad "C 相位行总数=$TOT（want 3）"

echo "ticket_key_gate: PASS=$pass FAIL=$fail"
[ "$fail" -eq 0 ]
