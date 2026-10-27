#!/usr/bin/env bash
# check_schema_live.sh —— T-0181 漂移哨兵：SG fin/dev md 各查
# max(schema_migrations.version) 对照登记表最大号，落后即红——「已合并≠
# 已执行」灰色态机械报（哨兵只读对照不自动补迁）。用法：
# scripts/check_schema_live.sh [SG_PG_DSN] [MD_PG_DSN]（缺=容器探测）。
set -euo pipefail
# 登记表最大号（单源——migrate_table 登记链是合法迁移清单；按链名取号）
max_of() { grep -oE '\{[0-9]+, "[a-z]+_[0-9]+' internal/store/pgstore/migrate_table.go 2>/dev/null || true; }
SG_DSN="${1:-}"
MD_DSN="${2:-}"
if [ -z "$SG_DSN" ]; then
	SG_PORT="$(docker port o-postgres 5432 2>/dev/null | head -n1 | awk -F: '{print $NF}')"
	SG_DSN="postgres://o:o@localhost:${SG_PORT:-15433}/o?sslmode=disable"
fi
if [ -z "$MD_DSN" ]; then
	MD_PORT="$(docker port o-postgres-md 5432 2>/dev/null | head -n1 | awk -F: '{print $NF}')"
	MD_DSN="postgres://o:o@localhost:${MD_PORT:-5435}/o?sslmode=disable"
fi
fail=0
for dom in fin md; do
	reg=$(grep -oE "\{[0-9]+, \"${dom}_[0-9]+" internal/store/pgstore/migrate_table.go \
		| grep -oE '^\{[0-9]+' | tr -d '{' | sort -n | tail -1)
	dsn=$([ "$dom" = fin ] && echo "$SG_DSN" || echo "$MD_DSN")
	live=$(psql "$dsn" -tc "SELECT COALESCE(max(version),0) FROM schema_migrations" 2>/dev/null | tr -d ' ')
	if [ -z "$live" ]; then
		echo "check_schema_live: $dom 链 schema_migrations 读败——fail-closed 红" >&2
		exit 1
	fi
	if [ "$live" -lt "$reg" ]; then
		echo "FAIL: $dom 链落后——live=$live 登记表=$reg（已合并≠已执行灰色态）" >&2
		fail=1
	else
		echo "OK: $dom 链 live=$live ≥ 登记表=$reg"
	fi
done
exit $fail
