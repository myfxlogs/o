#!/usr/bin/env bash
# check_migrations_dep_test.sh —— 断言⑨ depends_on 声明合法性桩测（工单#25800）：
# 夹具=真仓 migrations/+migrate_table.go/domain.tsv 拷入临时 git 仓
# （①-⑧ 基线面恒绿），逐案加测试迁移+登记行跑 scripts/check_migrations.sh，
# 按 FAIL 行断言——绿例须全谱 OK，红例逐条钉判词。
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
SRC=$(cd "$HERE/.." && pwd)
T=$(mktemp -d /tmp/cmdep-test-XXXXXX)
trap 'rm -rf "$T"' EXIT
PASS=0; FAILN=0
ok()  { PASS=$((PASS+1)); echo "  ok   $1"; }
bad() { FAILN=$((FAILN+1)); echo "  FAIL $1"; }

# ── 夹具基线 ──
FX="$T/fx"
mkdir -p "$FX/scripts" "$FX/internal/store/pgstore"
git -C "$FX" init -q
cp -r "$SRC/migrations" "$FX/migrations"
cp "$SRC/internal/store/pgstore/migrate_table.go" "$FX/internal/store/pgstore/"
cp "$HERE/check_migrations.sh" "$HERE/mig_dep_lib.sh" "$FX/scripts/"

REGF="$FX/internal/store/pgstore/migrate_table.go"
add_mig() { # $1=登记版号 $2=文件名 $3=SQL全文——落文件+登记行（表尾 } 前）
	printf '%s\n' "$3" > "$FX/migrations/$2"
	{ head -n -1 "$REGF"; printf '\t{%s, "%s"},\n' "$1" "$2"; echo '}'; } > "$REGF.tmp"
	mv "$REGF.tmp" "$REGF"
}
run_cm() { (cd "$FX" && bash scripts/check_migrations.sh) 2>&1; }
expect_fail() { # $1=标签 $2=grep式——输出须含判词行
	if printf '%s\n' "$OUT" | grep -q "$2"; then ok "$1"; else bad "$1（输出无判词 /$2/）"; printf '%s\n' "$OUT" | tail -5 | sed 's/^/    /'; fi
}

echo "== ⑨绿例：同链在册小号依赖 → 十断言全过（P2 ⑩ 并存） =="
add_mig 215 fin_0215_okdep.sql \
'-- okdep 正例
-- depends_on: fin_0001_app_config.sql
SELECT 1;'
OUT=$(run_cm); CRC=$?
if [ "$CRC" -eq 0 ] && printf '%s\n' "$OUT" | grep -q 'OK: check_migrations（十断言'; then
	ok "合法 depends_on 全谱绿（含⑩零候选面）"
else
	bad "合法 depends_on 未全绿"; printf '%s\n' "$OUT" | tail -8 | sed 's/^/    /'
fi

echo "== ⑨红例逐条 =="
add_mig 216 fin_0216_inv_a.sql \
'-- inv_a 号序倒挂
-- depends_on: fin_0217_inv_b.sql
SELECT 1;'
add_mig 217 fin_0217_inv_b.sql \
'-- inv_b 干净迁移
SELECT 1;'
OUT=$(run_cm)
expect_fail "依赖号≥声明者=倒挂红" 'fin_0216_inv_a.sql.*倒挂'

add_mig 218 fin_0218_ghost.sql \
'-- ghost 未登记依赖
-- depends_on: fin_9999_ghost.sql
SELECT 1;'
OUT=$(run_cm)
expect_fail "依赖未登记=红" 'fin_0218_ghost.sql.*未登记'

add_mig 219 fin_0219_xchain.sql \
'-- xchain 跨链依赖
-- depends_on: md_1003_market_facts.sql
SELECT 1;'
OUT=$(run_cm)
expect_fail "跨链依赖=红" 'fin_0219_xchain.sql.*跨链'

add_mig 220 fin_0220_stray.sql \
'-- stray 错位声明
SELECT 1;
-- depends_on: fin_0001_app_config.sql'
OUT=$(run_cm)
expect_fail "头区外声明=错位红" 'fin_0220_stray.sql.*声明错位'

add_mig 221 fin_0221_badval.sql \
'-- badval 非法值
-- depends_on: bogus.txt
SELECT 1;'
OUT=$(run_cm)
expect_fail "depends_on 值非法=红" 'fin_0221_badval.sql.*值非法'

echo "== 汇总: PASS=$PASS FAIL=$FAILN =="
[ "$FAILN" -eq 0 ]
