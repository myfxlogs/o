#!/usr/bin/env bash
# check_migrations_rehearsal_test.sh —— 断言⑩ 破坏性 DDL 存量违例 lint 桩测
# （工单#25799 形①）：夹具=真仓 migrations/+domain.tsv
# 拷入临时 git 仓（①-⑨ 基线面恒绿），逐案加水位以上新迁移+登记行跑
# scripts/check_migrations.sh，按 FAIL 行断言——绿例须全谱 OK，红例
# 逐条钉判词。水位 fin≤215/md≤1100 豁免由真仓存量 41 件违例迁移（含并线隙 fin_0310/md_1100 两件）
# 全绿实证（本夹具同拷=同豁免面）。
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
SRC=$(cd "$HERE/.." && pwd)
T=$(mktemp -d /tmp/cmreh-test-XXXXXX)
trap 'rm -rf "$T"' EXIT
PASS=0; FAILN=0
ok()  { PASS=$((PASS+1)); echo "  ok   $1"; }
bad() { FAILN=$((FAILN+1)); echo "  FAIL $1"; }

# ── 夹具基线 ──
FX="$T/fx"
mkdir -p "$FX/scripts"
git -C "$FX" init -q
cp -r "$SRC/migrations" "$FX/migrations"
cp "$HERE/check_migrations.sh" "$HERE/mig_dep_lib.sh" "$FX/scripts/"

add_mig() { # $1=版号(注记用) $2=文件名 $3=SQL全文——O-41367 落文件即登记
	printf '%s\n' "$3" > "$FX/migrations/$2"
}
run_cm() { (cd "$FX" && bash scripts/check_migrations.sh) 2>&1; }
expect_fail() { # $1=标签 $2=grep式——输出须含判词行
	if printf '%s\n' "$OUT" | grep -q "$2"; then ok "$1"; else bad "$1（输出无判词 /$2/）"; printf '%s\n' "$OUT" | tail -5 | sed 's/^/    /'; fi
}

echo "== ⑩绿例：两段式+豁免行 → 十断言全过 =="
add_mig 310 fin_0310_reh_ok.sql \
'-- reh_ok 两段式正例
ALTER TABLE fin_cashflow
	ADD CONSTRAINT fin_cashflow_dir_chk CHECK (direction IN ('"'"'long'"'"','"'"'short'"'"')) NOT VALID;
ALTER TABLE fin_cashflow VALIDATE CONSTRAINT fin_cashflow_dir_chk;
-- reh_ok2 豁免行正例（新表 UNIQUE 索引）
CREATE UNIQUE INDEX IF NOT EXISTS fin_cashflow_uq ON fin_cashflow (k);
-- mig-rehearsal-ok: fin_cashflow 本迁同建空表'
OUT=$(run_cm); CRC=$?
if [ "$CRC" -eq 0 ] && printf '%s\n' "$OUT" | grep -q 'OK: check_migrations（十断言'; then
	ok "两段式+豁免全谱绿"
else
	bad "两段式+豁免未全绿"; printf '%s\n' "$OUT" | tail -8 | sed 's/^/    /'
fi

echo "== ⑩红例逐条 =="
add_mig 311 fin_0311_reh_nnv.sql \
'-- rehnnv 缺 NOT VALID
ALTER TABLE fin_cashflow ADD CONSTRAINT fin_reh_nnv_chk CHECK (a > 0);'
OUT=$(run_cm)
expect_fail "ADD CONSTRAINT CHECK 缺 NOT VALID=红" 'fin_0311_reh_nnv.sql.*缺 NOT VALID'

add_mig 312 fin_0312_reh_noval.sql \
'-- rehnoval NOT VALID 不验
ALTER TABLE fin_cashflow ADD CONSTRAINT fin_reh_noval_chk CHECK (a > 0) NOT VALID;'
OUT=$(run_cm)
expect_fail "NOT VALID 无 VALIDATE=红" 'fin_0312_reh_noval.sql.*无 VALIDATE'

add_mig 313 fin_0313_reh_snn.sql \
'-- rehsnn SET NOT NULL 无豁免
ALTER TABLE fin_cashflow ALTER COLUMN a SET NOT NULL;'
OUT=$(run_cm)
expect_fail "SET NOT NULL 无豁免=红" 'fin_0313_reh_snn.sql.*SET NOT NULL'

add_mig 314 fin_0314_reh_colnn.sql \
'-- rehcolnn ADD COLUMN NOT NULL 无 DEFAULT 无豁免
ALTER TABLE fin_cashflow ADD COLUMN reh_c int NOT NULL;'
OUT=$(run_cm)
expect_fail "ADD COLUMN NOT NULL 无 DEFAULT=红" 'fin_0314_reh_colnn.sql.*ADD COLUMN NOT NULL'

add_mig 315 fin_0315_reh_uq.sql \
'-- rehuq UNIQUE 约束无豁免
ALTER TABLE fin_cashflow ADD CONSTRAINT fin_reh_uq_uq UNIQUE (a, b);'
OUT=$(run_cm)
expect_fail "ADD CONSTRAINT UNIQUE 无豁免=红" 'fin_0315_reh_uq.sql.*UNIQUE'

add_mig 316 fin_0316_reh_cui.sql \
'-- rehcui 唯一索引无豁免
CREATE UNIQUE INDEX fin_reh_cui_idx ON fin_cashflow (a);'
OUT=$(run_cm)
expect_fail "CREATE UNIQUE INDEX 无豁免=红" 'fin_0316_reh_cui.sql.*CREATE UNIQUE INDEX'

echo "== ⑩边界：豁免行覆盖+NOT VALID 双约束分段 =="
add_mig 317 fin_0317_reh_edge.sql \
'-- rehedge 边界例
-- mig-rehearsal-ok: 存量已人工清洗
ALTER TABLE fin_cashflow ADD CONSTRAINT fin_reh_e1_chk CHECK (a > 0);
ALTER TABLE fin_cashflow ADD CONSTRAINT fin_reh_e2_chk CHECK (b > 0) NOT VALID;
ALTER TABLE fin_cashflow VALIDATE CONSTRAINT fin_reh_e2_chk;'
OUT=$(run_cm)
expect_fail "豁免行不覆盖 CHECK 缺 NOT VALID（A 形无豁免径）" 'fin_0317_reh_edge.sql.*fin_reh_e1_chk.*缺 NOT VALID'
if printf '%s\n' "$OUT" | grep -q 'fin_reh_e2_chk.*缺 NOT VALID'; then
	bad "NOT VALID+VALIDATE 之 e2 竟误红"
else
	ok "两段式 e2 不误红"
fi

echo "== 汇总: PASS=$PASS FAIL=$FAILN =="
[ "$FAILN" -eq 0 ]
