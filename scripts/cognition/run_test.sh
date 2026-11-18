#!/usr/bin/env bash
# run_test.sh —— run.sh 唤起判果桩测（工单#22868 验收）：
# 桩 devin（--export 异步迟到写盘+list 空会话）+桩 psql（PATH 注入按词面
# 应答全量落 PSQL_LOG）+REPORT_ENV_FILE 桩 env（FIN_PG_DSN 非空令账径
# 走桩）——验两案：①rc=0+延迟 export=唤起果（旧径=唤起败败账失真）；
# ②rc=0+永不 export=唤起败（静默败判据不变）。fail-open 全程无真库。
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
RUN="$HERE/run.sh"
T=$(mktemp -d /tmp/run22868-XXXXXX)
trap 'rm -rf "$T"' EXIT
PASS=0; FAILN=0
ok()  { PASS=$((PASS+1)); echo "  ok   $1"; }
bad() { FAILN=$((FAILN+1)); echo "  FAIL $1"; }

mkdir -p "$T/bin" "$T/rt"
cat > "$T/bin/psql" <<'STUB'
#!/usr/bin/env bash
# psql 桩：SQL 全量落 PSQL_LOG，INSERT 应答 0 1，其余吐空
sql=""; prev=""
for a in "$@"; do [ "$prev" = "-c" ] && sql="$a"; prev="$a"; done
printf '%s\n' "$sql" >> "${PSQL_LOG:-/dev/null}"
case "$sql" in INSERT*) echo "INSERT 0 1";; esac
exit 0
STUB
chmod +x "$T/bin/psql"

# 桩 devin：list 空会话（session_state=new）；主跑 exit 0 后 spawn
# 后台写手按 STUB_DEFER 秒延迟落 export（真 devin --export 异步迟滞同构）；
# STUB_DEFER=never 案不写=静默败。
cat > "$T/bin/devin" <<'STUB'
#!/usr/bin/env bash
[ "${1:-}" = "list" ] && { echo '[]'; exit 0; }
export_path=""; prev=""
for a in "$@"; do [ "$prev" = "--export" ] && export_path="$a"; prev="$a"; done
echo "[stub-devin] session ran"
case "${STUB_DEFER:-8}" in
	never) ;;
	*) ( sleep "$STUB_DEFER"; printf '{"stub":1}\n' > "$export_path" ) & ;;
esac
exit 0
STUB
chmod +x "$T/bin/devin"

printf 'FIN_PG_DSN=postgres://stub.invalid/fin\n' > "$T/env"
printf '# stub runbook\n' > "$T/runbook.md"

run_case() { # $1=tag $2=defer $3=wait_s
	local tag=$1 defer=$2 wait=$3
	rm -rf "$T/rt"; mkdir -p "$T/rt"
	rm -f "$T/psql.log"
	env -i PATH="$T/bin:/usr/bin:/bin" HOME="$T" \
		O_WORKER_RUNTIME_DIR="$T/rt" O_WORKER_DEVIN_BIN="$T/bin/devin" \
		REPORT_ENV_FILE="$T/env" COG_RUNBOOK="$T/runbook.md" \
		COG_UNIT="test-$tag" COG_EXPORT_WAIT_S="$wait" \
		STUB_DEFER="$defer" PSQL_LOG="$T/psql.log" \
		bash "$RUN" >/dev/null 2>&1
	return $?
}

# 案① rc=0+8s 迟 export+窗 30s → 唤起果
t0=$(date +%s)
run_case ok 8 30
el=$(( $(date +%s) - t0 ))
if grep -q '唤起果' "$T/psql.log" && ! grep -q '唤起败' "$T/psql.log"; then
	ok "rc=0+延迟export(${el}s)→唤起果"
else
	bad "rc=0+延迟export→应唤起果实得:$(grep -c '唤起' "$T/psql.log" 2>/dev/null)行唤起"
fi
[ "$el" -ge 8 ] && ok "等待窗实等${el}s≥延迟8s" || bad "未等延迟(${el}s<8)"

# 案② rc=0+永不 export+窗 12s → 唤起败
t0=$(date +%s)
run_case fail never 12
el=$(( $(date +%s) - t0 ))
if grep -q '唤起败' "$T/psql.log" && ! grep -q '唤起果' "$T/psql.log"; then
	ok "rc=0+永不export(${el}s)→唤起败"
else
	bad "rc=0+永不export→应唤起败实得:$(grep -c '唤起' "$T/psql.log" 2>/dev/null)行唤起"
fi
[ "$el" -ge 10 ] && ok "静默败等全窗${el}s" || bad "静默败未等全窗(${el}s<10)"

echo "run_test: PASS=$PASS FAIL=$FAILN"
[ "$FAILN" -eq 0 ]
