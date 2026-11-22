#!/usr/bin/env bash
# regimedigest_test.sh —— regimedigest.sh 体制读数机械层桩测（工单O-32603）。
# 桩 psql 按 SQL 词面分派应答（全量落 PSQL_LOG 可查）：
#   regime_score=对拍 INSERT / md5(=到期判据 SELECT / regime_due=标件 INSERT。
# 案件（spec 桩测四案+接线/契约锚）：
#   ①到期 flag=1→标件 INSERT 发出且回显 regime读数到期
#   ②末读数新鲜 flag=0→零标件写
#   ③标件已在未消费（DB 幂等闸返空）→stdout 零产
#   ④对拍段：horizon 满读数→regime_score INSERT 发出（hit 机算在 SQL 内）
#   ⑤无凭据环境（env 无 FIN_PG_DSN）→不报错静默退
#   ⑥dispatch.sh 接线+runbook 指引+模板契约锚（闭集/NOT EXISTS/24h 窗）
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
DS="$HERE/regimedigest.sh"
T=$(mktemp -d /tmp/regimedigestXXXXXX)
trap 'rm -rf "$T"' EXIT
PASS=0; FAILN=0
ok()  { PASS=$((PASS+1)); echo "  ok   $1"; }
bad() { FAILN=$((FAILN+1)); echo "  FAIL $1"; }

mkdir -p "$T/bin"
printf 'FIN_PG_DSN=stub://regimedigest\n' > "$T/env"
printf '' > "$T/env.nodsn"

cat > "$T/bin/psql" <<'STUB'
#!/usr/bin/env bash
sql=""; prev=""; varfp=""
for a in "$@"; do
	[ "$prev" = "-c" ] && sql="$a"
	[ "$prev" = "-v" ] && varfp="$a"
	prev="$a"
done
[ -z "$sql" ] && sql="$(cat)"
printf '%s\n' "$sql" >> "${PSQL_LOG:-/dev/null}"
case "$sql" in
	*regime_score*) printf '%s\n' "${STUB_SCORE:-}" ;;
	*md5*) printf '%s\n' "${STUB_DUE:-0|deadbeef00}" ;;
	*regime_due*) printf '%s\n' "${STUB_MARKER:-}" ;;
esac
exit 0
STUB
chmod +x "$T/bin/psql"
export PATH="$T/bin:$PATH" PSQL_LOG="$T/psql.log" REPORT_ENV_FILE="$T/env"
export STUB_SCORE STUB_DUE STUB_MARKER

run() { : > "$PSQL_LOG"; OUT=$("$DS" 2>/dev/null); }
has() { printf '%s' "$OUT" | grep -q "$1"; }

echo "== 案① 到期→标件落账 =="
STUB_DUE='1|a1b2c3d4e5f60718'
STUB_MARKER='regime读数到期 fp=a1b2c3d4e5f60718——唤知会话按 scripts/cognition/regime_prompt.md 契约产 regime_digest 行(evid 须载 fp:a1b2c3d)'
STUB_SCORE=''
run
has 'regime读数到期' && has 'fp=a1b2c3d4' \
	&& ok "① 标件回显含 fp 与模板指针" || bad "① OUT=[$OUT]"
grep -q 'regime_due' "$PSQL_LOG" \
	&& ok "①b 标件 INSERT 发出（ops_event 量表/regime_due）" \
	|| bad "①b PSQL_LOG=$(cat "$PSQL_LOG")"

echo "== 案② 末读数新鲜→零标件 =="
STUB_DUE='0|a1b2c3d4e5f60718'; STUB_MARKER=''
run
[ -z "$OUT" ] && ! grep -q 'regime_due' "$PSQL_LOG" \
	&& ok "② flag=0：零标件写零产（对拍段 INSERT 属常态面不计）" \
	|| bad "② OUT=[$OUT] LOG=$(cat "$PSQL_LOG")"

echo "== 案③ 标件未消费幂等（DB 返空）=="
STUB_DUE='1|a1b2c3d4e5f60718'; STUB_MARKER=''
run
[ -z "$OUT" ] && grep -q 'regime_due' "$PSQL_LOG" \
	&& ok "③ INSERT 发出但 RETURNING 空=未消费标件不重复（DB 侧 NOT EXISTS 闸）" \
	|| bad "③ OUT=[$OUT]"

echo "== 案④ 对拍段评分行 =="
STUB_SCORE='regime_digest#99 hit=1 ret=2.50% label=liq_ease+risk_on conf=mid horizon=24h——对拍md_klines BTC-USDT-SWAP 1D确认棒 risk轴(薄=|ret|≤3%);hit_rate30d=55.0%'
STUB_DUE='0|x'
run
has 'regime_digest#99' && has 'hit=1' \
	&& ok "④ score 行回显（hit/ret/label/hit_rate30d 全帧）" || bad "④ OUT=[$OUT]"
grep -q 'regime_score' "$PSQL_LOG" \
	&& ok "④b 对拍 INSERT 发出（ticket=regime_score）" || bad "④b LOG=$(cat "$PSQL_LOG")"
! grep -qiE 'UPDATE|DELETE' "$PSQL_LOG" \
	&& ok "④c 全程零 UPDATE/DELETE（append-only）" || bad "④c LOG=$(cat "$PSQL_LOG")"

echo "== 案⑤ 无凭据环境静默退 =="
REPORT_ENV_FILE="$T/env.nodsn" OUT=$("$DS" 2>/dev/null); rc=$?
[ $rc -eq 0 ] && [ -z "$OUT" ] \
	&& ok "⑤ env 无 FIN_PG_DSN：rc=0 零输出零报错" || bad "⑤ rc=$rc OUT=[$OUT]"
REPORT_ENV_FILE="$T/env"

echo "== 案⑥ 接线+契约锚 =="
grep -q 'regimedigest.sh' "$HERE/dispatch.sh" \
	&& ok "⑥a dispatch.sh 拍前预扫调 regimedigest.sh" || bad "⑥a"
grep -q 'regime:due' "$HERE/dispatch.sh" \
	&& ok "⑥b dispatch.sh trigger 携 regime:due（随行+独立双径）" || bad "⑥b"
grep -q 'regime_prompt.md' "$HERE/runbook.md" \
	&& ok "⑥c runbook 指引 regime:due→模板" || bad "⑥c"
TPL_OK=1
for tok in 'regime=<liq_*+risk_*>' 'conf=<low|mid|high>' 'horizon=<24h|7d>' 'evid=fp:' 'NOT EXISTS' '24 hours' 'liq_ease' 'risk_thin'; do
	grep -qF "$tok" "$HERE/regime_prompt.md" || { TPL_OK=0; bad "⑥d 模板缺契约锚: $tok"; }
done
[ "$TPL_OK" = 1 ] && ok "⑥d 模板契约锚齐（闭集/去重窗/evid/龄界）"

echo "== 结果 PASS=$PASS FAIL=$FAILN =="
[ "$FAILN" = 0 ]
