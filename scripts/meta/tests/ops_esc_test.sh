#!/usr/bin/env bash
# ops_esc_test.sh —— T-0372 容错自报共享件单测：note 前缀 24h 同窗去重/
# 带类 escalate 径/无 escalate 直落径（OCI 单件部署面同式）/无 DSN fail-open。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin"

cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
query=${*: -1}
[[ $query == "-" ]] && query=$(cat)
case "$query" in
	*"SELECT 1 FROM ops_event WHERE event='升件'"*)
		[ "${DEDUP_FAIL:-0}" = 1 ] && exit 1 # dedup 查询败=空答→照常升（fail-open）
		# 前缀感知去重——抽 LIKE 'X%' 键回已落账 INSERT 行（同窗已报=1）
		pfx=$(sed -n "s/.*note LIKE '\([^']*\)%'.*/\1/p" <<<"$query")
		[ -n "${pfx:-}" ] && [ -s "${ESC_INSERTS:-/dev/null}" ] && grep -qF "$pfx" "$ESC_INSERTS" && echo 1 ;;
	*"INSERT INTO ops_event"*) printf '%s\n' "$query" >> "$ESC_INSERTS" ;;
	*) echo "unexpected psql: $query" >&2; exit 2 ;;
esac
EOF
chmod +x "$TMP/bin/psql"
printf 'FIN_PG_DSN=fake-dsn\n' > "$TMP/env"
export PATH="$TMP/bin:$PATH" OPS_ESC_ENV_FILE="$TMP/env" ESC_INSERTS="$TMP/inserts"
OPS_ESC="$ROOT/scripts/meta/ops_esc.sh"

# ① 首报落账——带类径（escalate.sh 在：esc_class/esc_scope 携值入库）
"$OPS_ESC" META-DM T-9002 写败 META-DM-USAGE "usage 采集败" "写径败 x"
[[ $(grep -c "INSERT INTO ops_event" "$ESC_INSERTS") -eq 1 ]]
grep -qF "VALUES('META-DM','升件','T-9002','usage 采集败——写径败 x','写败','META-DM-USAGE')" "$ESC_INSERTS"

# ② 同前缀 24h 窗内再报=去重不落第二行（防事件暴）
"$OPS_ESC" META-DM T-9002 写败 META-DM-USAGE "usage 采集败" "再败 y"
[[ $(grep -c "INSERT INTO ops_event" "$ESC_INSERTS") -eq 1 ]]

# ③ 异前缀=异窗——无类升件照落（class/scope 空→NULL 永不入判例面）
"$OPS_ESC" META-DM "" "" "" "队列推导败" "rc=1 空集滑行"
[[ $(grep -c "INSERT INTO ops_event" "$ESC_INSERTS") -eq 2 ]]
grep -qF "VALUES('META-DM','升件','','队列推导败——rc=1 空集滑行',NULL,NULL)" "$ESC_INSERTS"

# ④ 无 escalate.sh 径（OCI 单件部署面同式）——直落同构 INSERT
mkdir -p "$TMP/lonely"
cp "$OPS_ESC" "$TMP/lonely/ops_esc.sh"
"$TMP/lonely/ops_esc.sh" OCI-CRON "" "" "" "arch 保留期清扫败" "表=arch_eval"
grep -qF "VALUES('OCI-CRON','升件','','arch 保留期清扫败——表=arch_eval'" "$ESC_INSERTS"

# ⑤ dedup 查询自身败=空答→照常升（报侧 fail-open 不吞讯——T-0369 同律）
DEDUP_FAIL=1 "$OPS_ESC" META-DM "" "" "" "dedup败照升" "x"
[[ $(grep -c "INSERT INTO ops_event" "$ESC_INSERTS") -eq 4 ]]

echo "ops_esc dedup/class/fallback/failopen PASS"
