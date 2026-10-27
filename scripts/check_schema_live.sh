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
	SG_PORT="$(timeout 10 docker port o-postgres 5432 2>/dev/null | head -n1 | awk -F: '{print $NF}' || true)"
	SG_DSN="postgres://o:o@localhost:${SG_PORT:-15433}/o?sslmode=disable"
fi
if [ -z "$MD_DSN" ]; then
	MD_PORT="$(timeout 10 docker port o-postgres-md 5432 2>/dev/null | head -n1 | awk -F: '{print $NF}' || true)"
	MD_DSN="postgres://o:o@localhost:${MD_PORT:-5435}/o?sslmode=disable"
fi
fail=0
for dom in fin md; do
	reg=$(grep -oE "\{[0-9]+, \"${dom}_[0-9]+" internal/store/pgstore/migrate_table.go \
		| grep -oE '^\{[0-9]+' | tr -d '{' | sort -n | tail -1)
	dsn=$([ "$dom" = fin ] && echo "$SG_DSN" || echo "$MD_DSN")
	live=$(timeout 15 psql "$dsn" -tc "SELECT COALESCE(max(version),0) FROM schema_migrations" 2>/dev/null | tr -d ' ' || true)
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

# —— 盘漂哨兵（T-0181 追加面并案：SG 58G 盘曾被 nox/phase 死卷悄积 18G
# 无人量——docker system df 卷/镜像/缓存三项超阈即告警入 rejects；
# 输出连容器名一起报防按名误删——alphaforge-postgres 挂卷名
# ant_postgres_data=错名绑定实证教训）。
VOL_WARN_GB="${VOL_WARN_GB:-40}"      # SG 默认卷警线（可 config 化）
BC_WARN_GB="${BC_WARN_GB:-100}"       # dev 默认 buildcache 警线
to_gb() { # docker size 字符串→GB 粗值（kB/MB/GB/TB 四档）
	local v=$1 u=$2
	case $u in
		TB) echo "$v * 1024" | bc | cut -d. -f1 ;;
		GB) echo "$v" | cut -d. -f1 ;;
		MB) echo "$v / 1024" | bc | cut -d. -f1 ;;
		*)  echo 0 ;;
	esac
}
docker system df --format '{{.Type}}	{{.TotalCount}}	{{.Size}}' 2>/dev/null | while read -r typ cnt sz; do
	v=${sz%% *}; u=${sz##* }
	g=$(to_gb "$v" "$u")
	case $typ in
		Volumes) th=$VOL_WARN_GB ;;
		"Build Cache") th=$BC_WARN_GB ;;
		Images) th=$BC_WARN_GB ;;
		*) th=999999 ;;
	esac
	if [ "${g:-0}" -gt "$th" ]; then
		echo "CRITICAL: 盘漂——$typ $sz 超阈 ${th}GB（容器面见下防按名误删）" >&2
		timeout 8 docker ps --format '  {{.Names}} {{.Image}}' | head -8 >&2
		fail=1
	else
		echo "OK: $typ $sz ≤ 阈 ${th}GB"
	fi
done
exit $fail
