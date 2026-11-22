#!/usr/bin/env bash
# wakebus_test.sh —— 唤醒总线桩测（工单O-32300 验收四案+判据补件两案）：
# 桩 psql/git/env（dispatch_research/dispatch.sh 干跑面）+ WAKEBUS_ROWS_FILE
# 行集文件（Go 件 file 桩测面——matchRow 同文谓词预筛）+ WAKEBUS_KICK_CMD
# 踹件桩。
#   案① 工单求知行→踹 o-research-dispatch + dispatch_research RES_DRY=1
#       验 trigger 行在（踹不空转）；
#   案② 完工回执→踹 cognition / 复审过→踹 land_queue（候审链两跳）；
#   案③ 未知行类（量表/处置/记事无词面）→零踹；
#   案④ 同窗多行→同单元只踹一次（攒批去重）；
#   案⑤ dispatch.sh valmid 判据：非crit升件 valence≥60 且龄>600s→
#       COG_DRY trigger 含 valmid:（O-32300③普查必填项）；
#   案⑥ dispatch.sh await 判据：fresh 窗内完工待验票→trigger 含
#       await:N（O-32300②普查必填项）。
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
ROOT=$(cd "$HERE/../.." && pwd)
DISP="$HERE/dispatch.sh"; DRE="$HERE/dispatch_research.sh"
T=$(mktemp -d /tmp/wakebus-test-XXXXXX)
trap 'rm -rf "$T"' EXIT
PASS=0; FAILN=0
ok()  { PASS=$((PASS+1)); echo "  ok   $1"; }
bad() { FAILN=$((FAILN+1)); echo "  FAIL $1"; }

# ── 桩件铺装：Go 件一次构建 + psql/git 桩（dispatch 干跑面）──
mkdir -p "$T/bin"
export PATH="/usr/local/go/bin:$PATH"
if ! (cd "$ROOT" && go build -o "$T/bin/wakebus" ./tools/wakebus) 2>"$T/build.log"; then
	echo "  FAIL go build tools/wakebus（$(tail -3 "$T/build.log" | tr '\n' ' ')）"
	exit 1
fi
WB="$T/bin/wakebus"
cat > "$T/env" <<'EOF'
FIN_PG_DSN=postgres://stub/fin
EOF
cat > "$T/bin/psql" <<'EOF'
#!/usr/bin/env bash
# psql 桩：-c SQL 词面应答；heredoc stdin 非 tty 时补读（dispatch_research
# TRIGGERS 等喂法）——词面不识=空成功退出。
sql=""; prev=""
for a in "$@"; do [ "$prev" = "-c" ] && sql="$a"; prev="$a"; done
[ -z "$sql" ] && [ ! -t 0 ] && sql=$(cat)
printf '%s\n' "$sql" >> "${PSQL_LOG:-/dev/null}"
case "$sql" in
  *intake_dry*) cat "${RES_TRIGGERS:-/dev/null}" 2>/dev/null; exit 0;;
  *"act='receipt'"*) printf '%s|%s\n' "${T_AWAIT_N:-0}" "${T_AWAIT_FRESH:-0}"; exit 0;;
  *"FROM ops_esc_pending p"*) printf '%s|%s|%s\n' "${T_N:-0}" "${T_OLDEST:-0}" "${T_MAXV:-0}"; exit 0;;
  *"WHERE rulable"*) echo "${T_RULE_N:-0}"; exit 0;;
  *"WHERE workable OR rulable"*) echo "${T_TICKET_N:-0}"; exit 0;;
  *"esc_class IN"*对账*) echo "${T_CRIT_N:-0}"; exit 0;;
  *"seat='唤知机' AND event='唤起'"*) echo "${T_LAST_WOKE:-$(date +%s)}"; exit 0;;
  INSERT*) echo "INSERT 0 1";;
esac
exit 0
EOF
chmod +x "$T/bin/psql"
cat > "$T/bin/git" <<'EOF'
#!/usr/bin/env bash
# git 桩：dispatch.sh 仓探全应答成功+空集（不触真仓）。
case " $* " in
  *" rev-parse --git-dir "*|*" rev-parse --git-dir") echo .git;;
  *fetch*) exit 0;;
  *for-each-ref*) exit 0;;
  *merge-base*|*cat-file*) exit 1;;
  *rev-parse*verify*) exit 1;;
esac
exit 0
EOF
chmod +x "$T/bin/git"
cat > "$T/kick" <<'EOF'
#!/usr/bin/env bash
echo "$1" >> "${KICK_LOG:?}"
EOF
chmod +x "$T/kick"

run_wb() { # $1=行集文件(id|event|seat|note) → 单批次跑，踹日志 $T/kick.log
  : > "$T/kick.log"
  env WAKEBUS_ONCE=1 WAKEBUS_WM=0 WAKEBUS_ROWS_FILE="$1" \
    WAKEBUS_KICK_CMD="$T/kick" KICK_LOG="$T/kick.log" \
    "$WB" >/dev/null 2>&1
}
kicks() { sort "$T/kick.log" | uniq -c; }
rows() { printf '%s\n' "$@" > "$T/rows.txt"; }

# ── 案① 工单求知行→research 踹 + RES_DRY trigger 行 ──
rows '101|工单|外席|求知：测试外拓题——venue 面'
run_wb "$T/rows.txt"
if kicks | grep -q 'o-research-dispatch.service'; then ok "①a 求知工单→research踹"; else bad "①a 求知工单→research踹（实得:$(kicks|tr '\n' ' ')）"; fi
if kicks | grep -q 'o-worker-dispatch.service'; then bad "①b 求知工单误踹worker"; else ok "①b 求知工单不误踹worker"; fi
printf 'tk#777:求知:测试外拓题\n' > "$T/res_triggers.txt"
RES_OUT=$(env PATH="$T/bin:$PATH" REPORT_ENV_FILE="$T/env" PSQL_LOG="$T/psql.log" \
  RES_TRIGGERS="$T/res_triggers.txt" RES_DRY=1 bash "$DRE" 2>/dev/null)
if printf '%s' "$RES_OUT" | grep -q 'dry: trigger=.*tk#'; then ok "①c dispatch_research RES_DRY trigger行在"; else bad "①c dispatch_research RES_DRY trigger行（实得:$RES_OUT）"; fi

# ── 案② 完工回执→cognition / 复审过→landq ──
rows '102|完工回执|施工席|完工回执——O-1 支@sha 全量' \
     '103|复审过|唤知席|复审过 sha=0123456789abcdef0123456789abcdef01234567 fix/x——三维过' \
     '104|记事|外席|完工推支候审 fix/y sha=0123456789abcdef0123456789abcdef01234567——候审词面兜底'
run_wb "$T/rows.txt"
kicks | grep -q 'o-cognition-dispatch.service' && ok "②a 回执/候审→cognition踹" || bad "②a 回执→cognition踹（实得:$(kicks|tr '\n' ' ')）"
kicks | grep -q 'o-land-queue.service' && ok "②b 复审过→landq踹" || bad "②b 复审过→landq踹（实得:$(kicks|tr '\n' ' ')）"

# ── 案③ 未知行类→零踹 ──
rows '105|量表|唤知机|我在=v=x|0|1' \
     '106|处置|处置席|升件#1 销——已处置' \
     '107|记事|唤知|普通记事无路由词面' \
     '108|会话收口|唤知|unit=x 本轮收口'
run_wb "$T/rows.txt"
if [ ! -s "$T/kick.log" ]; then ok "③ 未知行类零踹"; else bad "③ 未知行类误踹:$(kicks|tr '\n' ' ')"; fi

# ── 案④ 同窗多行→同单元只踹一次 ──
rows '109|升件|探测席|部署/x: 测试升件' \
     '110|完工回执|施工席|完工回执——O-2' \
     '111|候审回执|外席|候审回执 fix/z sha=0123456789abcdef0123456789abcdef01234567'
run_wb "$T/rows.txt"
n=$(kicks | awk '$2=="o-cognition-dispatch.service"{print $1}')
[ "${n:-0}" = 1 ] && ok "④ 同窗三行→cognition只踹一次" || bad "④ 去重失效 n=$n（实得:$(kicks|tr '\n' ' ')）"

# ── 案⑤⑥ dispatch.sh 判据补件（COG_DRY 干跑）──
run_disp() { # 余参=桩应答覆写（env 操作数式）
  env PATH="$T/bin:$PATH" REPORT_ENV_FILE="$T/env" PSQL_LOG="$T/psql.log" \
    T_N="${T_N:-0}" T_OLDEST="${T_OLDEST:-0}" T_MAXV="${T_MAXV:-0}" \
    T_RULE_N="${T_RULE_N:-0}" T_AWAIT_N="${T_AWAIT_N:-0}" T_AWAIT_FRESH="${T_AWAIT_FRESH:-0}" \
    T_TICKET_N="${T_TICKET_N:-0}" T_CRIT_N="${T_CRIT_N:-0}" \
    T_LAST_WOKE="${T_LAST_WOKE:-$(date +%s)}" \
    COG_DRY=1 "$@" bash "$DISP" 2>/dev/null
}
D5=$(run_disp T_N=1 T_OLDEST=$(( $(date +%s) - 3600 )) T_MAXV=60)
printf '%s' "$D5" | grep -q 'valmid:60' && ok "⑤ valmid判据 trigger=valmid:60" || bad "⑤ valmid判据（实得:$D5）"
D5B=$(run_disp T_N=1 T_OLDEST=$(( $(date +%s) - 3600 )) T_MAXV=55)
printf '%s' "$D5B" | grep -q 'valmid' && bad "⑤b valence55误入valmid（实得:$D5B）" || ok "⑤b valence55<阈不入valmid"
D6=$(run_disp T_AWAIT_N=3 T_AWAIT_FRESH=1)
printf '%s' "$D6" | grep -q 'await:1' && ok "⑥ await判据 trigger含await:1" || bad "⑥ await判据（实得:$D6）"
D6B=$(run_disp T_AWAIT_N=3 T_AWAIT_FRESH=0)
printf '%s' "$D6B" | grep -q 'await:' && bad "⑥b 陈账await误触发（实得:$D6B）" || ok "⑥b fresh=0不触发await"

echo "wakebus_test: PASS=$PASS FAIL=$FAILN"
[ "$FAILN" = 0 ]
