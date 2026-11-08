#!/usr/bin/env bash
# dispatch_order_test.sh —— GW-001 段C3 派单序单源桩测：真实 dispatch_order.sh
# 原样（psql 桩喂已批ts）。面：①族机械映射四档序（postmortem/rootcause >
# 写败/账链断 > 观测盲 > 其余）②同 rank 已批ts 升序（先批先派）③票面
# 优先级 P0 覆写（观测盲跳队）④P2=默认档不覆写（族映射定 rank）⑤已批ts
# 查败 fail-open ⑥非法词形垫尾透传 ⑦空入空出。
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin" "$TMP/repo/docs/tasks"

cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
[ "${PSQL_FAIL:-0}" = 1 ] && exit 1
query="${*: -1}"
case "$query" in
  *"event='已批'"*) printf '%b\n' "${APPROVED_TS:-}" ;;
  *) : ;;
esac
EOF
chmod +x "$TMP/bin/psql"

# 桩票件：签名行/优先级行可控
mk_ticket() { # $1=tid $2=签名(可空) $3=优先级行(可空)
	local f="$TMP/repo/docs/tasks/$1-order.md"
	{
		echo "# $1 派单序桩票"
		echo ""
		if [ -n "$3" ]; then echo "- 优先级：$3（桩）"; fi
		if [ -n "$2" ]; then echo "病灶签名：\`$2\`（桩）"; fi
	} > "$f"
}
mk_ticket T-9001 "postmortem:fast_close" ""
mk_ticket T-9002 "写败:LIFECYCLE-HUNGER" ""
mk_ticket T-9003 "观测盲:face:table:x" ""
mk_ticket T-9004 "rootcause:unit_mismatch" ""
mk_ticket T-9005 "" ""
mk_ticket T-9006 "账链断:ops_event" ""
mk_ticket T-9007 "观测盲:face:y" "P0"     # P0 覆写——观测盲跳到最前
mk_ticket T-9008 "postmortem:settled_over" "P2" # P2 默认档=不覆写（族映射 rank0）

printf 'FIN_PG_DSN=fake\n' > "$TMP/env"
export PATH="$TMP/bin:$PATH"
export DM_ENV_FILE="$TMP/env"
export O_DISPATCH_ORDER_REPO="$TMP/repo"
ORDER="$ROOT/scripts/meta/dispatch_order.sh"

# ── ① 族机械映射四档序（输入乱序→输出 rank 升序，同 rank 票号序） ──
out=$(printf 'T-9003\nT-9005\nT-9001\nT-9002\nT-9006\nT-9008\nT-9004\n' | bash "$ORDER")
[ "$out" = $'T-9001\nT-9004\nT-9008\nT-9002\nT-9006\nT-9003\nT-9005' ] \
  || { echo "FAIL ①族映射序实出:[$out]"; exit 1; }
echo "ok ①族机械映射 postmortem/rootcause>写败/账链断>观测盲>其余"

# ── ② 同 rank 已批ts 升序（先批先派） ──
export APPROVED_TS=$'T-9001|2000\nT-9004|1000'
out=$(printf 'T-9001\nT-9004\n' | bash "$ORDER")
unset APPROVED_TS
[ "$out" = $'T-9004\nT-9001' ] || { echo "FAIL ②已批ts序实出:[$out]"; exit 1; }
echo "ok ②同 rank 已批ts 升序先批先派"

# ── ③ 票面优先级 P0 覆写：观测盲（rank2）跳到未覆写观测盲前 ──
out=$(printf 'T-9003\nT-9007\n' | bash "$ORDER")
[ "$out" = $'T-9007\nT-9003' ] || { echo "FAIL ③P0覆写实出:[$out]"; exit 1; }
echo "ok ③CTO 优先级 P0 覆写跳队（同族内显式改判先行）"

# ── ④ P2=起草默认档不覆写：postmortem 仍 rank0 ──
out=$(printf 'T-9003\nT-9008\n' | bash "$ORDER")
[ "$out" = $'T-9008\nT-9003' ] || { echo "FAIL ④P2默认档实出:[$out]"; exit 1; }
echo "ok ④P2 默认档不覆写（族映射定 rank）"

# ── ⑤ 已批ts 查败 fail-open：仍按 rank+票号排 ──
export PSQL_FAIL=1
out=$(printf 'T-9003\nT-9001\nT-9005\n' | bash "$ORDER")
unset PSQL_FAIL
[ "$out" = $'T-9001\nT-9003\nT-9005' ] || { echo "FAIL ⑤fail-open实出:[$out]"; exit 1; }
echo "ok ⑤psql 败 fail-open 按 rank 排不炸"

# ── ⑥ 非法词形垫尾透传 ──
out=$(printf 'T-9001\nGW-001\n' | bash "$ORDER")
[ "$out" = $'T-9001\nGW-001' ] || { echo "FAIL ⑥非法词形实出:[$out]"; exit 1; }
echo "ok ⑥非 T-NNNN 词形垫尾透传"

# ── ⑦ 空入空出 ──
out=$(printf '' | bash "$ORDER")
[ -z "$out" ] || { echo "FAIL ⑦空入非空出:[$out]"; exit 1; }
echo "ok ⑦空入空出"

echo "dispatch order tests PASS"
