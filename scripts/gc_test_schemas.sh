#!/usr/bin/env bash
# gc_test_schemas.sh —— T-0186 乙案兜底（日拍 cron）：o-postgres-test
# t_* schema 清面（测试内 GC 漏网行补清；DROP 范围只许 t_* 前缀——
# public/fin_*/md_* 白名单永不进 DROP 集）。
set -euo pipefail
DSN="${TEST_DSN:-postgres://o:o@localhost:5434/o_test?sslmode=disable}"
psql "$DSN" -Atc "SELECT 'DROP SCHEMA IF EXISTS '||quote_ident(nspname)||' CASCADE'
FROM pg_namespace WHERE nspname LIKE 't\_%'" | psql "$DSN" -q
echo "gc_test_schemas: done"
