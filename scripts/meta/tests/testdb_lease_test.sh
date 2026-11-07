#!/usr/bin/env bash
# testdb_lease_test.sh —— T-0435 租约隔离+假红标识桩测（票面验收①③）。
# 租约面=纯 flock 文件锁语义（不触真库）：
#   L1 两跑同刻争池→各得独占库（端口互异——断言不再互踩同 DSN）；
#   L2 池单库被占→排队等（holder 释后到手，非重指他库=禁租A注B）；
#   L3 池空→DSN 不注入+warn（fail-closed 不猜不占，skip 径）；
#   L4 外部 DSN→锁随 DSN 端口走（DSN 原样不变=租A注A）；
#   L5 外部 DSN 端口被占→排队后同端口到手（独占语义对人工径同成立）；
#   L6 外部 DSN 端口出族→不锁不改动 DSN（warn 留痕）；
#   L7 锁面坏→落共享库 5434+warn（票面④判不准 fail-closed 可观测）。
# 假红标面=check_all.sh 拷贝集成（go 桩三形，scope 桩测同法）：
#   E1 超时败+pgx 栈→.mode 尾行 env-contention+stderr 显式行+exit 1；
#   E2 断言红（无 panic）→不附标；
#   E3 超时败但栈非 pgx（真超时）→不附标（真超时断言不禁）。
set -uo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
FAIL=0
ok()  { echo "  ✅ $1"; }
bad() { echo "  ❌ $1"; FAIL=1; }
LEASE_SRC="$ROOT/scripts/testdb_lease.sh"
LOCKDIR="$TMP/locks"; mkdir -p "$LOCKDIR"

# _acq <names 逗号分隔→内转 newline（对齐 docker ps 形）> [topkey] [dsn] [wait]
# —— 独立 bash 进程跑一次 acquire，输出「port|warn行数|rc」；port=导出 DSN 端口或 NONE
_acq() {
  local names="${1//,/$'\n'}" topkey="${2:-}" dsn="${3:-}" wait="${4:-10}"
  bash -c '
    TESTDB_LEASE_LOCKDIR="'"$LOCKDIR"'"; TESTDB_LEASE_WAIT="'"$wait"'"
    TOPKEY="'"$topkey"'"; O_TEST_PG_DSN="'"$dsn"'"
    source "'"$LEASE_SRC"'"
    testdb_lease_acquire "$1" 2>"'"$TMP"'/acq.err"; rc=$?
    port=NONE
    [[ "${O_TEST_PG_DSN:-}" =~ :([0-9]+)/ ]] && port="${BASH_REMATCH[1]}"
    warns=$(grep -c "⚠️\|❌" "'"$TMP"'/acq.err" 2>/dev/null)
    echo "${port:-NONE}|${warns:-0}|$rc"
  ' _ "$names"
}

# ══ L1 两跑同刻争池→各得独占库 ══
rm -f "$TMP/l1a" "$TMP/l1b"
_acq "o-postgres-test,o-postgres-test-b" 00010000 "" 10 > "$TMP/l1a" &
_a=$!
_acq "o-postgres-test,o-postgres-test-b" 00010000 "" 10 > "$TMP/l1b" &
_b=$!
wait "$_a" "$_b"
PA=$(cut -d'|' -f1 "$TMP/l1a"); PB=$(cut -d'|' -f1 "$TMP/l1b")
if [ -n "$PA" ] && [ -n "$PB" ] && [ "$PA" != "$PB" ] && [ "$PA" != NONE ] && [ "$PB" != NONE ]; then
  ok "L1 两跑各得独占库（$PA≠$PB——零互踩同 DSN）"
else
  bad "L1 争池结果异常：PA=$PA PB=$PB"
fi

# ══ L2 池单库被占→排队等 ══
rm -f "$TMP/l2"
( bash -c '
    TESTDB_LEASE_LOCKDIR="'"$LOCKDIR"'"; TESTDB_LEASE_WAIT="10"
    source "'"$LEASE_SRC"'"
    testdb_lease_acquire "o-postgres-test" 2>/dev/null
    sleep 1.2 # 持库跑——模拟在途 check_all
  ' ) &
_h=$!
sleep 0.4 # 让 holder 先到手
T0=$(date +%s%N)
_acq "o-postgres-test" "" "" 10 > "$TMP/l2"
T1=$(date +%s%N)
wait "$_h" 2>/dev/null
P=$(cut -d'|' -f1 "$TMP/l2"); DMS=$(((T1 - T0) / 1000000))
if [ "$P" = "5434" ] && [ "$DMS" -ge 800 ]; then
  ok "L2 全忙排队：等 ${DMS}ms 后同端口 5434 到手（排队非重指非互踩）"
else
  bad "L2 排队形异常：port=$P 耗时=${DMS}ms"
fi

# ══ L3 池空→DSN 不注入+warn ══
r3=$(_acq "" "" "" 10)
P3=$(cut -d'|' -f1 <<< "$r3"); W3=$(cut -d'|' -f2 <<< "$r3")
if [ "$P3" = "NONE" ] && [ "$W3" -ge 1 ]; then
  ok "L3 池空：DSN 未注入（skip 径）+warn 留痕"
else
  bad "L3 池空形异常：port=$P3 warns=$W3"
fi

# ══ L4 外部 DSN→锁随端口（DSN 原样）══
rm -f "$TMP/l4"
_acq "o-postgres-test" 00010000 "postgres://o:o@localhost:5435/o_test?sslmode=disable" 10 > "$TMP/l4"
P4=$(cut -d'|' -f1 "$TMP/l4"); RC4=$(cut -d'|' -f3 "$TMP/l4")
if [ "$P4" = "5435" ] && [ "$RC4" = "0" ] && [ -f "$LOCKDIR/o-testdb-lease-5435.lock" ]; then
  ok "L4 外部 DSN：租约锁 5435 与 DSN 同源（租A注A——锁文件在案）"
else
  bad "L4 外部 DSN 形异常：port=$P4 rc=$RC4"
fi

# ══ L5 外部 DSN 端口被占→排队后同端口 ══
( bash -c '
    TESTDB_LEASE_LOCKDIR="'"$LOCKDIR"'"; TESTDB_LEASE_WAIT="10"
    source "'"$LEASE_SRC"'"
    testdb_lease_acquire "o-postgres-test-b" 2>/dev/null
    sleep 1.2
  ' ) &
_h5=$!
sleep 0.4
rm -f "$TMP/l5"
_acq "" "" "postgres://o:o@localhost:5435/o_test?sslmode=disable" 10 > "$TMP/l5"
wait "$_h5" 2>/dev/null
P5=$(cut -d'|' -f1 "$TMP/l5")
if [ "$P5" = "5435" ]; then
  ok "L5 外部 DSN 被占：排队后仍 5435（DSN 端口不重指=禁租A注B）"
else
  bad "L5 形异常：port=$P5"
fi

# ══ L6 外部 DSN 端口出族→不锁不改动 ══
rm -f "$TMP/l6"
_acq "" "" "postgres://o:o@localhost:9999/x?sslmode=disable" 10 > "$TMP/l6"
P6=$(cut -d'|' -f1 "$TMP/l6"); W6=$(cut -d'|' -f2 "$TMP/l6"); RC6=$(cut -d'|' -f3 "$TMP/l6")
ls "$LOCKDIR" | grep -q "9999" && BAD6=1 || BAD6=0
if [ "$RC6" = "0" ] && [ "$W6" -ge 1 ] && [ "$BAD6" = 0 ]; then
  ok "L6 出族 DSN：零锁零改动+warn 留痕（人工径自负）"
else
  bad "L6 形异常：rc=$RC6 warns=$W6 锁泄漏=$BAD6"
fi

# ══ L7 锁面坏→落共享库 5434+warn（票面④）══
r7=$(TESTDB_LEASE_LOCKDIR="/nonexistent-t0435-probe" bash -c '
    TESTDB_LEASE_WAIT="2"; TOPKEY=""
    source "'"$LEASE_SRC"'"
    testdb_lease_acquire "o-postgres-test" 2>"'"$TMP"'/l7.err"; rc=$?
    port=NONE
    [[ "${O_TEST_PG_DSN:-}" =~ :([0-9]+)/ ]] && port="${BASH_REMATCH[1]}"
    echo "$port|$rc"
  ')
P7=$(cut -d'|' -f1 <<< "$r7"); RC7=$(cut -d'|' -f2 <<< "$r7")
if [ "$P7" = "5434" ] && [ "$RC7" = "0" ]; then
  ok "L7 锁面坏：判不准→落共享库 5434（fail-closed 可观测，rc0 续跑）"
else
  bad "L7 形异常：port=$P7 rc=$RC7"
fi

# ══ 假红标集成面（check_all 拷贝+go 桩三形）══
REPO="$TMP/repo"; BIN="$TMP/bin"; mkdir -p "$REPO" "$BIN"
cd "$REPO" && git init -q && git config user.email t@t && git config user.name t
mkdir -p scripts web
printf '#!/usr/bin/env bash\nexit 0\n' > "$BIN/docker"
printf '#!/usr/bin/env bash\nexit 0\n' > "$BIN/gofmt"
for g in check_import_direction check-env-reads check_rejects_keys \
         check_config_write_domain check_migrations check_bridge_registry \
         check_tolerate_report check_ops_chain; do
  printf '#!/usr/bin/env bash\nexit 0\n' > "$REPO/scripts/$g.sh"
done
cp "$ROOT/scripts/check_all.sh" "$REPO/scripts/check_all.sh"
cp "$ROOT/scripts/testdb_lease.sh" "$REPO/scripts/testdb_lease.sh"
mkdir -p web && printf 'body{}\n' > web/app.css
git add -A && git commit -qm base
git update-ref refs/remotes/origin/master "$(git rev-parse HEAD)"
git checkout -qb work
printf 'body{color:red}\n' > web/app.css # 清单外面→full 档
git add -A && git commit -qm red
SHA=$(git rev-parse HEAD)

# go 桩：$GO_MODE=timeout-pgx | assert-fail | timeout-cpu
# 实参形硬化：test 桩 $2 必形如包路径（./...或 o/…）——run_go_test 漏
# shift 时 fail-msg 混入实参表（真 go 报 malformed import path，桩同红）
cat > "$BIN/go" <<'EOF'
#!/usr/bin/env bash
case "$1" in
  build|vet) exit 0 ;;
  test)
    case "${2:-}" in
      ./*|o/*) : ;;
      *) printf 'malformed import path "%s"\nFAIL\n' "${2:-<empty>}"; exit 1 ;;
    esac
    case "${GO_MODE:-}" in
      timeout-pgx)
        printf 'panic: test timed out after 30m0s\n\trunning tests:\n\t\tTestMigrateHang\n\ngoroutine 1 [select]:\ngithub.com/jackc/pgx/v5.(*Conn).Exec(0x0, 0x0)\nFAIL\n'
        exit 1 ;;
      timeout-cpu)
        printf 'panic: test timed out after 30m0s\n\ngoroutine 1 [running]:\no/internal/foo.burnCPU(0x1)\nFAIL\n'
        exit 1 ;;
      *) printf -- '--- FAIL: TestX (0.00s)\nFAIL\n'; exit 1 ;;
    esac ;;
  *) exit 0 ;;
esac
EOF
chmod +x "$BIN"/*

run_ca() { # $1=GO_MODE → rc 到 RC_CA，stderr 到 $TMP/ca.err
  ( cd "$REPO" && PATH="$BIN:$PATH" GO_MODE="$1" bash scripts/check_all.sh ) \
    >"$TMP/ca.out" 2>"$TMP/ca.err"
  RC_CA=$?
}
modefile() { echo "$REPO/.git/check-green-$SHA.mode"; }

run_ca timeout-pgx
if [ "$RC_CA" != 0 ] && [ "$(tail -1 "$(modefile)")" = "env-contention" ] \
  && grep -q "env-contention" "$TMP/ca.err"; then
  ok "E1 超时+pgx 栈：exit 1+.mode 附 env-contention+stderr 显式行（假红可区分）"
else
  bad "E1 形异常：rc=$RC_CA mode尾=$(tail -1 "$(modefile)" 2>/dev/null)"
fi

run_ca assert-fail
if [ "$RC_CA" != 0 ] && [ "$(tail -1 "$(modefile)")" = "full" ]; then
  ok "E2 断言红：不附标（码红保持裸形）"
else
  bad "E2 形异常：rc=$RC_CA mode尾=$(tail -1 "$(modefile)" 2>/dev/null)"
fi

run_ca timeout-cpu
if [ "$RC_CA" != 0 ] && [ "$(tail -1 "$(modefile)")" = "full" ]; then
  ok "E3 真超时（非 pgx 栈）：不附标（真超时断言不禁）"
else
  bad "E3 形异常：rc=$RC_CA mode尾=$(tail -1 "$(modefile)" 2>/dev/null)"
fi

[ "$FAIL" = 0 ] && echo "✅ testdb_lease_test 全场景过" || { echo "❌ testdb_lease_test 有红"; exit 1; }
