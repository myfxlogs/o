#!/usr/bin/env bash
# check_surface_lifecycle_test.sh —— 積面出生闸桩测（工单O-39400②B）：
# ①未登记 CREATE TABLE=闸红 ②同迁移 fin_surface_lifecycle 登记=绿
# ③豁免册命中=绿 ④登一表不免同文件他表=闸红 ⑤零迁移面=绿。
set -uo pipefail
cd "$(dirname "$0")/.." || exit 1
GATE=scripts/check_surface_lifecycle.sh
T=$(mktemp -d); trap 'rm -rf "$T"' EXIT
mkdir -p "$T/mig"
REG="$T/reg.txt"; printf 'exempt_table\nfin_surface_lifecycle\n' > "$REG"

# ①未登记积面=红
printf 'CREATE TABLE new_accum (id int);\n' > "$T/mig/a.sql"
if SURFACE_LIFECYCLE_MIG_DIR="$T/mig" SURFACE_LIFECYCLE_REG="$REG" bash "$GATE" >/dev/null 2>&1; then
	echo "FAIL: 未登记未拦"; exit 1
fi

# ②同迁移登记=绿（登一表不免他表——④同件覆盖）
printf 'CREATE TABLE new_accum (id int);\nINSERT INTO fin_surface_lifecycle(surface,accumulates,consumer_ref,terminal_path) VALUES('"'"'new_accum'"'"',true,'"'"'c'"'"','"'"'t'"'"');\n' > "$T/mig/a.sql"
SURFACE_LIFECYCLE_MIG_DIR="$T/mig" SURFACE_LIFECYCLE_REG="$REG" SURFACE_LIFECYCLE_FILES="a.sql" bash "$GATE" >/dev/null 2>&1 \
	|| { echo "FAIL: 已登记误拦"; exit 1; }

# ③豁免册=绿
printf 'CREATE TABLE exempt_table (id int);\n' > "$T/mig/a.sql"
SURFACE_LIFECYCLE_MIG_DIR="$T/mig" SURFACE_LIFECYCLE_REG="$REG" SURFACE_LIFECYCLE_FILES="a.sql" bash "$GATE" >/dev/null 2>&1 \
	|| { echo "FAIL: 豁免册误拦"; exit 1; }

# ④登一表不免同文件他表=红
printf 'CREATE TABLE new_accum (id int);\nCREATE TABLE sneaky_accum (id int);\nINSERT INTO fin_surface_lifecycle(surface,accumulates,consumer_ref,terminal_path) VALUES('"'"'new_accum'"'"',true,'"'"'c'"'"','"'"'t'"'"');\n' > "$T/mig/a.sql"
if SURFACE_LIFECYCLE_MIG_DIR="$T/mig" SURFACE_LIFECYCLE_REG="$REG" SURFACE_LIFECYCLE_FILES="a.sql" bash "$GATE" >/dev/null 2>&1; then
	echo "FAIL: 同文件未登记表漏拦"; exit 1
fi

# ⑤零面=绿
rm -f "$T/mig/"*.sql
SURFACE_LIFECYCLE_MIG_DIR="$T/mig" SURFACE_LIFECYCLE_REG="$REG" bash "$GATE" >/dev/null 2>&1 \
	|| { echo "FAIL: 零迁移面误拦"; exit 1; }

echo "OK: check_surface_lifecycle 桩测五锚全绿"
