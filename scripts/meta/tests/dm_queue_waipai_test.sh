#!/usr/bin/env bash
# dm_queue_waipai_test.sh —— T-0375 件① 外派豁免桩测：真实 dm_queue.sh
# 原样出 SQL（psql 桩截获）× sqlite3 引擎评（查询面标准 SQL，直跑）。
# 场景矩阵（ISO 文本 ts 同构排——词法序=时序）：
#   A 外派+收口（无终态裁决）→零行（锚①钦定桩）
#   B 外派全轨（已批→外派→领单→收口/移交）→零行（真实手动外派序列）
#   C 外派+收口+裁决→零行（终态后仍零）
#   D 外派→已批（重批外派件）→零行（整轨 COS,DM 不接）
#   E 纯收口无裁决（非外派件）→有行（对照——豁免不误伤）
#   F 纯已批（非外派件）→有行（对照——常径不变形）
#   G 收口+裁决（非外派件终态收敛）→零行（对照——收敛律照旧）
#   H 已批+派单（非外派件已派）→零行（对照——排除集照旧）
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin"

# psql 桩：仅支持 dm_queue.sh 调形（psql <DSN> -qAt -c <SQL>）——SQL 原样
# 交 sqlite3 评（fixture 库 OPS_EV_DB；ts=ISO 文本，词法序=时序）
cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
SQL="" ; prev=""
for a in "$@"; do
  if [ "$prev" = "-c" ]; then SQL=$a; break; fi
  prev=$a
done
[ -n "$SQL" ] || { echo "psql-stub: 无 -c 实参" >&2; exit 1; }
python3 - "$OPS_EV_DB" "$SQL" <<'PY'
import sqlite3, sys
con = sqlite3.connect(sys.argv[1])
for row in con.execute(sys.argv[2]):
    print(row[0])
PY
EOF
chmod +x "$TMP/bin/psql"

printf 'FIN_PG_DSN=stub\n' > "$TMP/env"
export PATH="$TMP/bin:$PATH"
export DM_ENV_FILE="$TMP/env"
export OPS_EV_DB="$TMP/ops_event.db"

# fixture 建表+场景行灌入（ts 依序微秒递进,同刻事件同 ts——真实 ops_event 同构）
python3 - "$OPS_EV_DB" <<'PY'
import sqlite3, sys
con = sqlite3.connect(sys.argv[1])
con.execute("CREATE TABLE ops_event(seat text, event text, ticket text, note text, ts text)")
rows = [
    # A 外派+收口（锚①）
    ("COS","外派","T-9001","","2026-01-01 10:00:00.000000+00"),
    ("无头施工席","收口","T-9001","","2026-01-01 11:00:00.000000+00"),
    # B 外派全轨（真实手动外派序列）
    ("COS","已批","T-9002","","2026-01-01 10:00:00.000000+00"),
    ("COS","外派","T-9002","","2026-01-01 10:00:05.000000+00"),
    ("无头施工席","领单","T-9002","","2026-01-01 10:30:00.000000+00"),
    ("无头施工席","收口","T-9002","","2026-01-01 12:00:00.000000+00"),
    ("无头施工席","移交","T-9002","","2026-01-01 12:00:02.000000+00"),
    # C 外派+收口+裁决
    ("COS","外派","T-9003","","2026-01-01 10:00:00.000000+00"),
    ("无头施工席","收口","T-9003","","2026-01-01 11:00:00.000000+00"),
    ("COS","裁决","T-9003","","2026-01-01 13:00:00.000000+00"),
    # D 外派→已批（重批外派件）
    ("COS","外派","T-9004","","2026-01-01 10:00:00.000000+00"),
    ("COS","已批","T-9004","","2026-01-01 10:05:00.000000+00"),
    # E 纯收口无裁决（非外派件——豁免不误伤）
    ("无头施工席","收口","T-9011","","2026-01-01 11:00:00.000000+00"),
    # F 纯已批（非外派件——常径不变形）
    ("COS","已批","T-9012","","2026-01-01 10:00:00.000000+00"),
    # G 收口+裁决（非外派件收敛）
    ("无头施工席","收口","T-9013","","2026-01-01 11:00:00.000000+00"),
    ("决策席","裁决","T-9013","","2026-01-01 13:00:00.000000+00"),
    # H 已批+派单（非外派件已派）
    ("COS","已批","T-9014","","2026-01-01 10:00:00.000000+00"),
    ("决策席","派单","T-9014","","2026-01-01 10:01:00.000000+00"),
]
con.executemany("INSERT INTO ops_event VALUES(?,?,?,?,?)", rows)
con.commit()
PY

out=$(bash "$ROOT/scripts/meta/dm_queue.sh")
pass=0; fail=0
has(){ grep -Fxq "$1" <<<"$out" && { echo "PASS $2（$1 在队）"; pass=$((pass+1)); } \
       || { echo "FAIL $2（$1 应在队,实出:[$out]）"; fail=$((fail+1)); }; }
lacks(){ ! grep -Fxq "$1" <<<"$out" && { echo "PASS $2（$1 不在队）"; pass=$((pass+1)); } \
       || { echo "FAIL $2（$1 应排除,实出:[$out]）"; fail=$((fail+1)); }; }

lacks T-9001 "A 外派+收口→零行（锚①）"
lacks T-9002 "B 外派全轨→零行"
lacks T-9003 "C 外派+收口+裁决→零行"
lacks T-9004 "D 外派→已批→零行"
has   T-9011 "E 纯收口无裁决→有行（对照）"
has   T-9012 "F 纯已批→有行（对照）"
lacks T-9013 "G 收口+裁决→零行（对照收敛）"
lacks T-9014 "H 已批+派单→零行（对照排除）"

echo "─────"
if [ "$fail" = 0 ]; then echo "✅ dm_queue 外派豁免桩测 $pass/$((pass+fail)) 全绿"; exit 0
else echo "❌ 桩测红：$fail 败/$((pass+fail))"; exit 1; fi
