#!/usr/bin/env bash
# dm_shiftend_test.sh —— T-0437 件③ 班退闸桩测（伪件退班流程重演——
# 退班理由行必现断言）：
# ①红桩——已批未领非空∧部署窗有挡→理由行必落（event=退班/seat=决策席/
#   note 三字段 谁挡=/等什么=/下一拍谁接= 齐携）+shiftend=reasoned
# ②全净（approved 空∧批空）→shiftend=clean 零行（静默退唯一合法态）
# ③队列推导败=不可知→fail-closed 照落行（不可知也必须发声）
# ④窗壳本体败（无 state 行）→fail-closed 照落行
# ⑤理由行写径败→exit 1 显报（禁静默退）
# 桩面：queue/window 脚本桩+psql 捕获桩——不碰真库真窗壳。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d); trap 'rm -rf -- "$TMP"' EXIT

INSERTS="$TMP/inserts"; : > "$INSERTS"
cat > "$TMP/psql" <<EOF
#!/usr/bin/env bash
set -euo pipefail
query=\${*: -1}
if [[ "\$query" == *"INSERT INTO ops_event"* ]]; then
	[[ "\${PSQL_FAIL:-0}" = 1 ]] && exit 1
	printf '%s\n' "\$query" >> "$INSERTS"
	exit 0
fi
echo "unexpected psql: \$query" >&2
exit 2
EOF
chmod +x "$TMP/psql"
printf 'FIN_PG_DSN=fake-dsn\n' > "$TMP/env"

# queue 桩：行为由控制文件定（empty/rc1/list）
QCTL="$TMP/qctl"
cat > "$TMP/q.sh" <<EOF
#!/usr/bin/env bash
case "\$(cat "$QCTL")" in
	rc1) exit 1 ;;
	list) printf 'T-0424\nT-0423\n' ;;
	*) exit 0 ;;
esac
EOF
chmod +x "$TMP/q.sh"
# window 桩：有挡/批空/本体败三态
WCTL="$TMP/wctl"
cat > "$TMP/w.sh" <<EOF
#!/usr/bin/env bash
case "\$(cat "$WCTL")" in
	blocked) printf 'state=有挡\nfaces=image,oci\nreason=挡件未清: 在飞:T-0432(img)——应发面集(image,oci)全被挡(冻结面=image)\n'; exit 11 ;;
	batchempty) printf 'state=批空\nfaces=\nreason=diff 双基线俱空\n'; exit 10 ;;
	*) echo "boom" >&2; exit 3 ;;
esac
EOF
chmod +x "$TMP/w.sh"

run_gate() {
	( env DM_ENV_FILE="$TMP/env" O_SHIFTEND_QUEUE_SH="$TMP/q.sh" \
		O_SHIFTEND_WINDOW_SH="$TMP/w.sh" O_SHIFTEND_PSQL="$TMP/psql" \
		bash "$ROOT/scripts/meta/dm_shiftend.sh" )
}
fail() { echo "❌ $*" >&2; exit 1; }

# ①红桩——双源非空：理由行必现（三字段齐）
printf 'list' > "$QCTL"; printf 'blocked' > "$WCTL"
out=$(run_gate) || fail "①闸异常退: $out"
grep -q '^shiftend=reasoned' <<<"$out" || fail "①输出非 reasoned: $out"
[ "$(grep -c "INSERT INTO ops_event" "$INSERTS")" -eq 1 ] || fail "①理由行未落"
ROW=$(cat "$INSERTS")
grep -qF "VALUES('决策席','退班','','退班理由：" <<<"$ROW" || fail "①行头非决策席退班: $ROW"
grep -qF "谁挡=已批未领=T-0424,T-0423；部署窗有挡" <<<"$ROW" || fail "①谁挡字段缺/清单错: $ROW"
grep -qF "等什么=等派单" <<<"$ROW" || fail "①等什么缺派单条件: $ROW"
grep -qF "等挡件清窗——挡件未清: 在飞:T-0432(img)" <<<"$ROW" || fail "①等什么缺清窗条件: $ROW"
grep -qF "下一拍谁接=dm_tick 30min 兜底拍" <<<"$ROW" || fail "①下一拍字段缺: $ROW"

# ②全净——clean 零行
: > "$INSERTS"; printf 'empty' > "$QCTL"; printf 'batchempty' > "$WCTL"
out=$(run_gate) || fail "②闸异常退: $out"
grep -q '^shiftend=clean' <<<"$out" || fail "②输出非 clean: $out"
[ ! -s "$INSERTS" ] || fail "②全净竟落行: $(cat "$INSERTS")"

# ③队列推导败——fail-closed 落行（不可知发声）
: > "$INSERTS"; printf 'rc1' > "$QCTL"
out=$(run_gate) || fail "③闸异常退: $out"
grep -q '^shiftend=reasoned' <<<"$out" || fail "③输出非 reasoned: $out"
grep -qF "谁挡=已批未领面不可知(rc=1)" "$INSERTS" || fail "③不可知未发声: $(cat "$INSERTS")"
grep -qF "fail-closed——队列推导败不可知也必须发声" "$INSERTS" || fail "③不可知等因缺: $(cat "$INSERTS")"

# ④窗壳本体败（rc=3 无 state 行）——fail-closed 落行
: > "$INSERTS"; printf 'empty' > "$QCTL"; printf 'die' > "$WCTL"
out=$(run_gate) || fail "④闸异常退: $out"
grep -qF "部署窗有挡项面不可知(rc=3)" "$INSERTS" || fail "④窗壳败未发声: $(cat "$INSERTS")"

# ⑤写径败——exit 1 显报禁静默退
printf 'list' > "$QCTL"; printf 'blocked' > "$WCTL"
rc=0; out=$(PSQL_FAIL=1 run_gate 2>&1) || rc=$?
[ "$rc" -eq 1 ] || fail "⑤写径败应 exit1 实得 rc=$rc: $out"
grep -q "写径败" <<<"$out" || fail "⑤stderr 缺显报: $out"

echo "dm_shiftend_test 全绿"
