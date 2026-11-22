#!/usr/bin/env bash
# closeoutcert_test.sh —— 收口「调用径实证」闸桩测（工单O-32269）。
# 桩 psql 按 SQL 词面分派（argv 全量落 PSQL_ARGS——escalate.sh
# scope/prefix/detail 走 -v 参不在 SQL 文面）：候选查询回 STUB_CAND
# （id|note TSV——ts/生效锚过滤在 SQL 面桩不可达，由 STUB_CAND 非空
# =已过锚语义承载）；SEEN 查询（esc_scope=closeout_callpath）回
# STUB_SEEN 行集；ops_esc stdin 径回行 id 并落 PSQL_LOG。
# 桩 git 按子命令分派：cat-file 恒在案（STUB_BADSHA 除外）/fetch 恒
# 成/merge-base 回 STUB_MB/diff --name-status 回 STUB_NS/diff 回
# STUB_PATCH——diff 内容面全由桩应答承载，谓词三分支在本测覆盖。
# 案件（spec 三案+边界）：
#   ①触发类(.sh新改)+实证段在→不升件
#   ②触发类+实证段缺→审计/closeout_callpath 升件（载回执#id）
#   ③非触发类(纯Go)→豁免不升件
#   ④CREATE FUNCTION 增行→触发②升件
#   ⑤psql 调用形增行→触发③升件
#   ⑥已报回执#id（SEEN 在案）→不重报
#   ⑦--check 单件判三态（过/要件缺/豁免/不可解）
#   ⑧dispatch.sh 拍前预扫接线在账
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
SP="$HERE/closeoutcert.sh"
T=$(mktemp -d /tmp/closeoutcertXXXXXX)
trap 'rm -rf "$T"' EXIT
PASS=0; FAILN=0
ok()  { PASS=$((PASS+1)); echo "  ok   $1"; }
bad() { FAILN=$((FAILN+1)); echo "  FAIL $1"; }

SHA_A=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
SHA_B=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
mkdir -p "$T/bin"
printf 'FIN_PG_DSN=stub://closeoutcert\n' > "$T/env"

cat > "$T/bin/psql" <<'STUB'
#!/usr/bin/env bash
# psql 桩：-c 词面优先，否则读 stdin（escalate.sh stdin 喂串径）；
# argv 全量落 PSQL_ARGS（-v 参面断言用），SQL 落 PSQL_LOG。
sql=""; prev=""
for a in "$@"; do [ "$prev" = "-c" ] && sql="$a"; prev="$a"; done
printf '%s\n' "$*" >> "${PSQL_ARGS:-/dev/null}" # tolerate: args 面写败不翻测判
if [ -z "$sql" ]; then sql=$(cat 2>/dev/null); fi
printf '%s\n' "$sql" >> "${PSQL_LOG:-/dev/null}"
case "$sql" in
	*ops_esc*) echo "${STUB_ESC_ID:-7}" ;;
	*closeout_callpath*event*升件*|*event*升件*closeout_callpath*) printf '%s\n' "${STUB_SEEN:-}" ;;
	*完工回执*|*施工收口*) printf '%s\n' "${STUB_CAND:-}" ;;
esac
exit 0
STUB
chmod +x "$T/bin/psql"

cat > "$T/bin/git" <<'STUB'
#!/usr/bin/env bash
# git 桩：子命令分派——cat-file/fetch 恒应；merge-base 回 STUB_MB；
# diff --name-status 回 STUB_NS；diff 回 STUB_PATCH；余参静默 rc0。
sub=""; skip=0
for a in "$@"; do
	if [ "$skip" = 1 ]; then skip=0; continue; fi
	case "$a" in -C) skip=1; continue;; -*) continue;; *) sub="$a"; break;; esac
done
case "$sub" in
	cat-file) for a in "$@"; do case "$a" in *^{commit}) [ "$a" = "${STUB_BADSHA:-___none___}^{commit}" ] && exit 1; exit 0;; esac; done; exit 0 ;;
	fetch) exit 0 ;;
	merge-base) echo "${STUB_MB:-mb000000000000000000000000000000000000}"; exit 0 ;;
	diff)
		for a in "$@"; do [ "$a" = "--name-status" ] && { printf '%s\n' "${STUB_NS:-}"; exit 0; }; done
		printf '%s\n' "${STUB_PATCH:-}"; exit 0 ;;
esac
exit 0
STUB
chmod +x "$T/bin/git"

run_cc() { # 余参=env 前缀覆写
	env PATH="$T/bin:$PATH" REPORT_ENV_FILE="$T/env" O_REPO_DIR="$T/repo" \
		PSQL_LOG="$T/psql.log" PSQL_ARGS="$T/psql.args" \
		STUB_CAND="${STUB_CAND:-}" STUB_SEEN="${STUB_SEEN:-}" \
		STUB_MB="${STUB_MB:-mb000000000000000000000000000000000000}" \
		STUB_NS="${STUB_NS:-}" STUB_PATCH="${STUB_PATCH:-}" \
		STUB_BADSHA="${STUB_BADSHA:-}" STUB_ESC_ID="${STUB_ESC_ID:-7}" \
		bash "$SP" >/dev/null 2>&1
}
reset_logs() { : > "$T/psql.log"; : > "$T/psql.args"; }

NOTE_NOEVID="完工回执——O-9001 feat/x 落成 feat/x@$SHA_A 已推origin。裁案取舍=候选a。"
NOTE_EVID="完工回执——O-9001 feat/x 落成 feat/x@$SHA_A 已推origin。调用径实证: 实跑 psql -c \"SELECT ops_esc()\" → INSERT 0 1 id=55 rc=0"

# 案① 触发类(.sh新改)+实证段在→不升件
reset_logs
STUB_CAND=$'9001|'"$NOTE_EVID" STUB_NS=$'M\tscripts/foo.sh' run_cc
if grep -q 'ops_esc' "$T/psql.log"; then bad "触发+实证在竟升件"; else ok "触发类+实证段在不升件"; fi

# 案② 触发类+实证段缺→升件（载回执#id+scope）
reset_logs
STUB_CAND=$'9002|'"$NOTE_NOEVID" STUB_NS=$'M\tscripts/foo.sh' run_cc
if grep -q 'ops_esc' "$T/psql.log" && grep -q '回执#9002' "$T/psql.args"; then ok "触发类缺实证→升件载回执#id"; else bad "触发类缺实证未升件/缺回执id"; fi

# 案③ 非触发类(纯Go+.md)→豁免不升件（无实证段亦不追）
reset_logs
STUB_CAND=$'9003|'"$NOTE_NOEVID" STUB_NS=$'M\tcmd/core/foo.go\nM\tdocs/x.md' \
	STUB_PATCH=$'@@ -1 +1 @@\n-old\n+new go line\n+doc line' run_cc
if grep -q 'ops_esc' "$T/psql.log"; then bad "非触发类竟升件"; else ok "非触发类豁免不升件"; fi

# 案④ CREATE FUNCTION 增行→触发②升件
reset_logs
STUB_CAND=$'9004|'"$NOTE_NOEVID" STUB_NS=$'M\tmigrations/fin_9999_x.sql' \
	STUB_PATCH=$'@@ -0,0 +1 @@\n+CREATE OR REPLACE FUNCTION ops_esc() RETURNS int' run_cc
if grep -q 'ops_esc' "$T/psql.log" && grep -q '回执#9004' "$T/psql.args"; then ok "CREATE FUNCTION增行→触发升件"; else bad "CREATE FUNCTION增行未触发"; fi

# 案⑤ psql 调用形增行→触发③升件（.txt 非.sh 非迁移面）
reset_logs
STUB_CAND=$'9005|'"$NOTE_NOEVID" STUB_NS=$'M\tscripts/notes.txt' \
	STUB_PATCH=$'@@ -0,0 +1 @@\n+result=$(psql "$DSN" -c "SELECT 1")' run_cc
if grep -q 'ops_esc' "$T/psql.log" && grep -q '回执#9005' "$T/psql.args"; then ok "psql调用形增行→触发升件"; else bad "psql调用形增行未触发"; fi

# 案⑥ 已报回执#id（SEEN 在案）→不重报
reset_logs
STUB_CAND=$'9006|'"$NOTE_NOEVID" STUB_NS=$'M\tscripts/foo.sh' \
	STUB_SEEN=$'升件旧note 回执#9006@abc123456 已报' run_cc
if grep -q 'ops_esc' "$T/psql.log"; then bad "已报回执竟重报"; else ok "已报回执#id不重报"; fi

# 案⑦ --check 单件判四态（git 桩 env 须显式传——非导出壳变量不入子进程）
cc_check() { env PATH="$T/bin:$PATH" O_REPO_DIR="$T/repo" \
	STUB_NS="${STUB_NS:-}" STUB_PATCH="${STUB_PATCH:-}" \
	STUB_BADSHA="${STUB_BADSHA:-}" STUB_MB="${STUB_MB:-mb000000000000000000000000000000000000}" \
	bash "$SP" --check "$@" >/dev/null 2>&1; }
STUB_NS=$'M\tscripts/foo.sh' STUB_PATCH=''
if cc_check "$SHA_A" "$NOTE_EVID"; then ok "--check 触发+实证→rc0"; else bad "--check 触发+实证非rc0"; fi
if cc_check "$SHA_A" "$NOTE_NOEVID"; then bad "--check 触发缺实证竟rc0"; else ok "--check 触发缺实证→rc1"; fi
STUB_NS=$'M\tcmd/core/foo.go' STUB_PATCH=$'+go line only'
if cc_check "$SHA_A" ''; then ok "--check 非触发→rc0豁免"; else bad "--check 非触发非rc0"; fi
STUB_BADSHA="$SHA_B"
if cc_check "$SHA_B" "$NOTE_NOEVID"; then bad "--check 坏sha竟rc0"; else ok "--check 坏sha→rc2不可解"; fi

# 案⑧ 接线锚：dispatch.sh 拍前预扫在账
grep -q 'closeoutcert.sh' "$HERE/dispatch.sh" && ok "dispatch.sh接线closeoutcert" || bad "dispatch.sh未接closeoutcert"

echo "closeoutcert_test: PASS=$PASS FAIL=$FAILN"
[ "$FAILN" -eq 0 ]
