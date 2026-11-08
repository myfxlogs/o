#!/usr/bin/env bash
# attribution_why_chain_test.sh —— T-0464 桩测：问链五段断点归段+立案写径
# +24h 同断点去重（生产谱形 fixture——psql 桩按 /*wc:段*/ 标签应答；
# 四值机判序归因面=Go 侧 TestAttributePnlFourValues/
# TestPostmortemPnlAttributionWriteback 钉死，本件不重复）。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin"

# 桩 psql：/*wc:<段>*/ 标签分派段取数（env 供谱形 fixture）；INSERT/心跳
# 行记 PSQL_LOG 供立案写径断言；未料查询=显式炸（防漏桩静默绿）。
cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "psql $*" >> "$PSQL_LOG"
query="${*: -1}"
case "$query" in
	*"/*wc:armed*/"*)   echo "${ARMED_LINE:-0|0|0|}" ;;
	*"/*wc:open*/"*)    echo "${OPEN_LINE:-0||0}" ;;
	*"/*wc:fill*/"*)    echo "${FILL_LINE:-0|0|}" ;;
	*"/*wc:settle*/"*)  echo "${SETTLE_LINE:-0|0|}" ;;
	*"/*wc:pnl*/"*)     echo "${PNL_LINE:-0|0|0/0}" ;;
	*"/*wc:esc*/"*)     echo "${ESC_N:-0}" ;;
	*"/*wc:events*/"*)  printf '%s\n' "${EVENTS_TXT:-}" ;;
	*"/*wc:openage*/"*) echo "${OPEN_AGEH:-0}" ;;
	*"/*wc:dup*/"*)     echo "${DUP_N:-0}" ;;
	*"INSERT INTO fin_econ_lesson"*) : ;;
	*"INSERT INTO ops_headless"*) : ;;
	*) echo "unexpected psql: $query" >&2; exit 2 ;;
esac
EOF
chmod +x "$TMP/bin/psql"
printf 'FIN_PG_DSN=fake-dsn\nFIN_PG_RO_DSN=ro-dsn\n' > "$TMP/env"
export PSQL_LOG="$TMP/psql.log" PATH="$TMP/bin:$PATH"
: > "$PSQL_LOG"
SCAN="$ROOT/scripts/meta/why_chain_scan.sh"
VT="$ROOT/scripts/meta/verdict_tick.sh"

# 全绿 fixture（五段全通+零背离——静默空跑）。
GREEN_ENV=(
	ARMED_LINE="10|0|5|2026-11-08 00:00+00"
	OPEN_LINE="3|2026-11-08 10:00+00|3"
	OPEN_AGEH="2"
	FILL_LINE="40|0|"
	SETTLE_LINE="0|0|2026-11-08 09:00+00"
	PNL_LINE="5.5|12.5|3/8"
	ESC_N="0" EVENTS_TXT="" DUP_N="0"
)
run_scan() { # env 覆写经环境注入
	env "$@" WHYCHAIN_ENV_FILE="$TMP/env" O_WHYCHAIN_FILE_SH="$VT" \
		VERDICT_ENV_FILE="$TMP/env" "$SCAN" >/dev/null 2>&1 || true # tolerate: 桩测壳层 rc 不碍断言
}
files() { grep -c 'INSERT INTO fin_econ_lesson' "$PSQL_LOG" || true; }
reset_log() { : > "$PSQL_LOG"; }

# ① 全绿=零立案（兜底拍零烧——无 INSERT）。
reset_log
run_scan "${GREEN_ENV[@]}"
[ "$(files)" -eq 0 ]

# ② armed 滞留>20=链首断点=armed（fill 同步坏亦归 armed——序钉死）。
reset_log
run_scan "${GREEN_ENV[@]}" ARMED_LINE="25|2|30|2026-11-07 00:00+00" \
	FILL_LINE="40|3|failed"
[ "$(files)" -eq 1 ]
grep -q 'ref=why_chain#armed#' "$PSQL_LOG"
grep -q '"break_segment":"armed"' "$PSQL_LOG"
grep -q "'rootcause_diag'" "$PSQL_LOG"
# 读写分面断言：段取数走 RO 册（ro-dsn），去重预检走写册（fake-dsn）。
grep 'wc:armed' "$PSQL_LOG" | grep -q 'ro-dsn'
grep 'wc:dup' "$PSQL_LOG" | grep -q 'fake-dsn'

# ③ armed 通+open 零开仓窗>24h=断点=open（fill 坏不越序）。
reset_log
run_scan "${GREEN_ENV[@]}" OPEN_AGEH="30" OPEN_LINE="0||0" \
	FILL_LINE="40|2|ambiguous"
[ "$(files)" -eq 1 ]
grep -q 'ref=why_chain#open#' "$PSQL_LOG"
grep -q '"break_segment":"open"' "$PSQL_LOG"

# ④ fill 错因行非空（armed/open/settle 通）=断点=fill。
reset_log
run_scan "${GREEN_ENV[@]}" FILL_LINE="40|2|failed,ambiguous"
[ "$(files)" -eq 1 ]
grep -q 'ref=why_chain#fill#' "$PSQL_LOG"
grep -q '"break_segment":"fill"' "$PSQL_LOG"

# ⑤ settle 滞留平仓未结>1h=断点=settle。
reset_log
run_scan "${GREEN_ENV[@]}" SETTLE_LINE="2|0|2026-11-08 08:00+00"
[ "$(files)" -eq 1 ]
grep -q 'ref=why_chain#settle#' "$PSQL_LOG"
grep -q '"break_segment":"settle"' "$PSQL_LOG"

# ⑥ pnl 7d 净额<0=断点=pnl（符号锚——钱面收口段）。
reset_log
run_scan "${GREEN_ENV[@]}" PNL_LINE="-8.2|-21.4|2/9"
[ "$(files)" -eq 1 ]
grep -q 'ref=why_chain#pnl#' "$PSQL_LOG"
grep -q '"break_segment":"pnl"' "$PSQL_LOG"

# ⑦ 24h 同断点去重——dup 预检=1 →零立案（同向背离不重复案卷）。
reset_log
run_scan "${GREEN_ENV[@]}" ARMED_LINE="25|2|30|2026-11-07 00:00+00" DUP_N="1"
[ "$(files)" -eq 0 ]

# ⑧ 事件驱面：T-0465 议题行在案+链全通=断点=chain_clear 立案
#    （断点不在钱链五段如实记——议题 metric 名随案卷留痕）。
reset_log
run_scan "${GREEN_ENV[@]}" \
	EVENTS_TXT=$'北极星背离:armed滞留=25 vs 基线20——问链引T-0464'
[ "$(files)" -eq 1 ]
grep -q 'ref=why_chain#chain_clear#' "$PSQL_LOG"
grep -q '"break_segment":"chain_clear"' "$PSQL_LOG"
grep -q '北极星背离:armed滞留' "$PSQL_LOG"

# ⑨ 断链未领>0=ops 背离触发面（链全通→chain_clear 立案）。
reset_log
run_scan "${GREEN_ENV[@]}" ESC_N="3"
[ "$(files)" -eq 1 ]
grep -q 'ref=why_chain#chain_clear#' "$PSQL_LOG"
grep -q '"unclaimed_escalations":3' "$PSQL_LOG"

# ⑩ verdict_tick.sh file 子命令直测——kind='rootcause_diag' 写径词形
#    +缺参 fail-closed。
reset_log
env WHYCHAIN_ENV_FILE="$TMP/env" VERDICT_ENV_FILE="$TMP/env" \
	"$VT" file "why_chain#armed#20990101" '{"case":{"break_segment":"armed"}}' \
	>/dev/null 2>&1 || true # tolerate: 桩测壳层
grep -q "INSERT INTO fin_econ_lesson" "$PSQL_LOG"
grep -q "rootcause_diag" "$PSQL_LOG"
grep -q "ref=why_chain#armed#20990101" "$PSQL_LOG"
if env VERDICT_ENV_FILE="$TMP/env" "$VT" file "" '{}' >/dev/null 2>&1; then
	echo "file 子命令缺参未 fail-closed" >&2; exit 1
fi

echo "attribution_why_chain PASS"
