#!/usr/bin/env bash
# app_config 值型别普查（T-0046 ④b——canon config-json-type-fragility）：
# 读器已双型兼容（GetFloat=number|string、GetBool=bool|string），本普查揪
# 结构性异类（object/array/null——任何读器都不吃）+ 给数字消费键的人核清单。
# 用法：PHASE_TEST_PG_DSN=… 或 DB_DSN=…（生产普查由决策席跑）
# 出口：结构性异类存在=非零退出（异常行零=PASS）。
set -euo pipefail
DSN="${PHASE_TEST_PG_DSN:-${DB_DSN:-}}"
if [ -z "$DSN" ]; then echo "需要 PHASE_TEST_PG_DSN 或 DB_DSN" >&2; exit 2; fi

echo "== 全键型别分布 =="
psql "$DSN" -Atc "SELECT t||'  '||n FROM (SELECT jsonb_typeof(value) t, count(*) n FROM app_config GROUP BY 1) x ORDER BY 1;"

echo "== 结构性异类（object/null 任何读器不吃；array 仅 *.symbols 清单键合法） =="
ANOM=$(psql "$DSN" -Atc "SELECT key FROM app_config WHERE jsonb_typeof(value) IN ('object','null') OR (jsonb_typeof(value)='array' AND key NOT LIKE '%symbols') ORDER BY key;")
if [ -n "$ANOM" ]; then echo "$ANOM"; echo "异常行存在——修值后重跑"; exit 1; fi
echo "（零异常行）"

echo "== 数字消费键非数字形清单（GetFloat 消费面人核——字符串形合法但须知情） =="
psql "$DSN" -Atc "SELECT key FROM app_config WHERE jsonb_typeof(value)='string' AND key ~ '(bps|hours|seconds|usd|pct|tol)' ORDER BY key;" || true
echo "普查完：零异常行=PASS"
