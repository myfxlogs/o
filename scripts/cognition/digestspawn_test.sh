#!/usr/bin/env bash
# digestspawn_test.sh —— digestspawn.sh 发现词面→求知票自产闸桩测（工单O-31997）。
# 桩 psql 按 SQL 词面分派应答（全量落 PSQL_LOG 可查）——本件单 SQL
# （heredoc 无 -c），桩读 stdin 兜底捕获。STUB_SPAWN 注入 RETURNING 应答。
# 案件（spec 四案全覆盖+边界）：
#   ①digest含发现词→产求知票（stdout 回显票面含 求知:/digest#id/命中词）
#   ②无词/无候选（DB 返空）→不产（stdout 空）
#   ③重放幂等——桩态文件模拟「首产后 desk 已有引用票」二拍返空→零增
#   ④销件语义不变——全程零 UPDATE/DELETE/ops_esc_pending 写面
#   ⑤dispatch.sh 拍前预扫接线在账（grep 锚）
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
DS="$HERE/digestspawn.sh"
T=$(mktemp -d /tmp/digestspawnXXXXXX)
trap 'rm -rf "$T"' EXIT
PASS=0; FAILN=0
ok()  { PASS=$((PASS+1)); echo "  ok   $1"; }
bad() { FAILN=$((FAILN+1)); echo "  FAIL $1"; }

mkdir -p "$T/bin"
printf 'FIN_PG_DSN=stub://digestspawn\n' > "$T/env"

# 桩 psql：-c 词面优先，否则读 stdin（heredoc 喂 SQL）；spawn 判形=
# 同时含 regexp_matches∧ops_ticket_desk∧INSERT INTO ops_event。
# STUB_ONCE=1 时首应答后落标件，次拍返空=模拟幂等闸已在账。
cat > "$T/bin/psql" <<'STUB'
#!/usr/bin/env bash
sql=""; prev=""
for a in "$@"; do [ "$prev" = "-c" ] && sql="$a"; prev="$a"; done
[ -z "$sql" ] && sql="$(cat)"
printf '%s\n' "$sql" >> "${PSQL_LOG:-/dev/null}"
case "$sql" in
	*regexp_matches*ops_ticket_desk*|*ops_ticket_desk*regexp_matches*)
		if [ "${STUB_ONCE:-0}" = 1 ]; then
			if [ -f "${STUB_MARK:?}" ]; then exit 0; fi
			touch "${STUB_MARK}"
		fi
		printf '%s\n' "${STUB_SPAWN:-}" ;;
esac
exit 0
STUB
chmod +x "$T/bin/psql"
export PATH="$T/bin:$PATH" PSQL_LOG="$T/psql.log" REPORT_ENV_FILE="$T/env"
export STUB_SPAWN STUB_ONCE STUB_MARK="$T/spawn.mark"

run() { OUT=$("$DS" 2>/dev/null); }
has() { printf '%s' "$OUT" | grep -q "$1"; }

echo "== 案① digest含发现词→产求知票 =="
STUB_SPAWN='求知：O-27488——digest#31714发现词面派生：适配器提案/达标候选'
STUB_ONCE=0
run
has '求知：O-27488' && has 'digest#31714' && has '适配器提案/达标候选' \
	&& ok "① 产票回显：票面=求知:题——digest#id发现词面派生:命中词" \
	|| bad "① OUT=[$OUT]"
grep -q 'INSERT INTO ops_event' "$PSQL_LOG" \
	&& ok "①b 写径=ops_event INSERT（工单行→desk视图同源入板）" \
	|| bad "①b PSQL_LOG=$(cat "$PSQL_LOG")"

echo "== 案② 无候选→不产 =="
: > "$PSQL_LOG"; STUB_SPAWN=""
run
[ -z "$OUT" ] && ok "② DB返空：stdout零产" || bad "② OUT=[$OUT]"

echo "== 案③ 重放幂等零增 =="
rm -f "$STUB_MARK"; STUB_ONCE=1
STUB_SPAWN='求知：O-27488——digest#31714发现词面派生：适配器提案/达标候选'
run
has 'digest#31714' && ok "③a 首拍产票" || bad "③a OUT=[$OUT]"
run
[ -z "$OUT" ] && ok "③b 次拍（desk已有引用票）：零增" || bad "③b OUT=[$OUT]"

echo "== 案④ 销件语义不变 =="
! grep -qiE 'UPDATE|DELETE|ops_esc_pending|esc_class|valence' "$PSQL_LOG" \
	&& ok "④ 全程零 UPDATE/DELETE/esc 面写——digest销件链不触" \
	|| bad "④ PSQL_LOG=$(cat "$PSQL_LOG")"

echo "== 案⑤ dispatch.sh 拍前预扫接线 =="
grep -q 'digestspawn.sh' "$HERE/dispatch.sh" \
	&& ok "⑤ dispatch.sh 调 digestspawn.sh（mine.sh 后拍前预扫位）" \
	|| bad "⑤ dispatch.sh 无接线"

echo "== 结果 PASS=$PASS FAIL=$FAILN =="
[ "$FAILN" = 0 ]
