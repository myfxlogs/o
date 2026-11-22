#!/usr/bin/env bash
# sessionpulse_test.sh —— 会话在修龄阈监控桩测（工单O-32268②）。
# 桩 psql 按 SQL 词面分派（全量落 PSQL_LOG；argv 全量落 PSQL_ARGS——
# escalate.sh scope/prefix/detail 走 -v 参不在 SQL 文面，session_stall
# 断言须查 args 面）：Q1 候选查询回 STUB_CAND（id|epoch|seat|unit
# TSV——龄过滤在 SQL 面桩不可达，龄>90min 判据由 STUB_CAND 非空=
# 已过龄语义承载）；Q2 配对计数回 STUB_PAIR；ops_esc stdin 径回行 id。
# 案件（spec 三案+边界）：
#   ①龄超阈无配对→session_stall 升件（scope/prefix 按 unit 分账）
#   ②配对行在（收口/果/败任一）→不升件
#   ③候选空（全配对/全鲜）→不升件
#   ④unit token 缺席行→跳不升（旧形唤起行不判）
#   ⑤dispatch.sh 拍前预扫接线在账（grep 锚）
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
SP="$HERE/sessionpulse.sh"
T=$(mktemp -d /tmp/sessionpulseXXXXXX)
trap 'rm -rf "$T"' EXIT
PASS=0; FAILN=0
ok()  { PASS=$((PASS+1)); echo "  ok   $1"; }
bad() { FAILN=$((FAILN+1)); echo "  FAIL $1"; }

mkdir -p "$T/bin"
printf 'FIN_PG_DSN=stub://sessionpulse\n' > "$T/env"

cat > "$T/bin/psql" <<'STUB'
#!/usr/bin/env bash
# psql 桩：-c 词面优先，否则读 stdin（escalate.sh stdin 喂串径）；
# SQL 全量落 PSQL_LOG，argv 全量落 PSQL_ARGS（-v 参面断言用）。
# Q1=含 event='唤起' 的 SELECT；Q2=含 '会话收口','唤起果','唤起败'
# 的 count；ops_esc=SELECT ops_esc(。
sql=""; prev=""; stdin_sql=""
for a in "$@"; do [ "$prev" = "-c" ] && sql="$a"; prev="$a"; done
printf '%s\n' "$*" >> "${PSQL_ARGS:-/dev/null}" # tolerate: args 面写败不翻测判
if [ -z "$sql" ]; then stdin_sql=$(cat 2>/dev/null); sql="$stdin_sql"; fi
printf '%s\n' "$sql" >> "${PSQL_LOG:-/dev/null}"
case "$sql" in
	*"event='唤起'"*) printf '%s\n' "${STUB_CAND:-}" ;;
	*会话收口*唤起果*唤起败*|*唤起果*唤起败*会话收口*) echo "${STUB_PAIR:-0}" ;;
	*ops_esc*) echo "${STUB_ESC_ID:-7}" ;;
esac
exit 0
STUB
chmod +x "$T/bin/psql"

run_sp() { # 余参=桩应答覆写（env 前缀式）
	env PATH="$T/bin:$PATH" REPORT_ENV_FILE="$T/env" \
		PSQL_LOG="$T/psql.log" PSQL_ARGS="$T/psql.args" \
		STUB_CAND="${STUB_CAND:-}" STUB_PAIR="${STUB_PAIR:-0}" STUB_ESC_ID="${STUB_ESC_ID:-7}" \
		bash "$SP" >/dev/null 2>&1
}

# 案① 龄超阈无配对→升件
: > "$T/psql.log"; : > "$T/psql.args"
STUB_CAND=$'101|1700000000|唤知机|o-cognition-111' run_sp
if grep -q 'session_stall_o-cognition-111' "$T/psql.args"; then ok "龄超阈无配对→session_stall升件"; else bad "龄超阈无配对未升件"; fi
if grep -q 'ops_esc' "$T/psql.log"; then ok "升件走ops_esc函件径"; else bad "升件未走函件径"; fi

# 案② 配对行在→不升件
: > "$T/psql.log"; : > "$T/psql.args"
STUB_CAND=$'102|1700000000|唤知机|o-cognition-222' STUB_PAIR=1 run_sp
if grep -q 'session_stall' "$T/psql.args"; then bad "配对行在竟升件"; else ok "配对行在不升件"; fi

# 案③ 候选空→不升件
: > "$T/psql.log"; : > "$T/psql.args"
STUB_CAND='' run_sp
if grep -q 'session_stall' "$T/psql.args"; then bad "候选空竟升件"; else ok "候选空不升件"; fi

# 案④ unit token 缺席→跳
: > "$T/psql.log"; : > "$T/psql.args"
STUB_CAND=$'103|1700000000|唤知机|' STUB_PAIR=0 run_sp
if grep -q 'session_stall' "$T/psql.args"; then bad "无unit行竟升件"; else ok "无unit行跳过不升件"; fi

# 案⑤ 多未配对候选→逐支升件（PAIR=0 全未配对面；混合配对面由
# 案①②方向合证——STUB_PAIR 单应答桩不可分行，分行判据在 Q2 SQL）
: > "$T/psql.log"; : > "$T/psql.args"
STUB_CAND=$'104|1700000000|唤知机|o-cognition-333\n105|1700000001|审计机|o-audit-444' STUB_PAIR=0 run_sp
n=$(grep -c 'ops_esc' "$T/psql.log")
[ "$n" -eq 2 ] && ok "多未配对候选逐支升件($n)" || bad "多候选升件数=$n 应=2"

# 案⑥ 接线锚：dispatch.sh 拍前预扫在账
grep -q 'sessionpulse.sh' "$HERE/dispatch.sh" && ok "dispatch.sh接线sessionpulse" || bad "dispatch.sh未接sessionpulse"

echo "sessionpulse_test: PASS=$PASS FAIL=$FAILN"
[ "$FAILN" -eq 0 ]
