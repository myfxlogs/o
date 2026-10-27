#!/usr/bin/env bash
# ledger_hash.sh —— 账链面零触动指纹（T-0169 验收锚：切换前后 fin_cashflow/
# fin_position 行哈希一致）。行变量 ::text=全列文本聚合 md5——任一列变即指纹变。
# 切换前跑一份留档、切换后重跑 diff 本文件——一致=零触动实证。
# 用法：SG_DSN=postgres://... scripts/ledger_hash.sh [out文件]
set -euo pipefail
SG_DSN="${SG_DSN:?SG_DSN 必填}"
OUT="${1:-/tmp/ledger_hash_$(date +%s).txt}"
PSQL="docker run --rm --network host timescale/timescaledb:2.30.2-pg18 psql"
{
	echo "# 账链指纹 @ $(date -Is)（SG=$SG_DSN）"
	for t in fin_cashflow fin_position fin_execution fin_strategy; do
		$PSQL "$SG_DSN" -tA -v ON_ERROR_STOP=1 \
			-c "SELECT '$t', count(*), COALESCE(sum(id),0), md5(COALESCE(string_agg(t::text, ';' ORDER BY id),'∅')) FROM $t t"
	done
} >> "$OUT"
cat "$OUT"
echo "指纹已落 $OUT——切换后重跑 diff 本文件即验收锚证据"
