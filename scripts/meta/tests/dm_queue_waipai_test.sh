#!/usr/bin/env bash
# dm_queue_waipai_test.sh —— T-0375 件① 外派豁免桩测：真实 dm_queue.sh
# 原样出 SQL（psql 桩截获）× sqlite3 引擎评（查询面标准 SQL，直跑）。
# 场景矩阵（ISO 文本 ts 同构排——词法序=时序）：
# 返修1 新规断言（86efad3b 业主令——外派件收口回 DM 常径裁，豁免只盖已批面）：
#   A 外派+收口（无终态裁决）→有行（收口面零排除，回 DM 待裁）
#   B 外派全轨（已批→外派→领单→收口/移交）→有行（收口面回 DM 常径）
#   C 外派+收口+裁决→零行（终态收敛照旧）
#   E 纯收口无裁决（非外派件）→有行（对照——豁免不误伤）
#   G 收口+裁决（非外派件终态收敛）→零行（对照——收敛律照旧）
#   H 已批+派单（非外派件已派）→零行（对照——排除集照旧）
# T-0380 三形断言（latest 事件语义——判据由「票史有外派行」改「外派
# 后无再批」；D 原「零行」断言随判据事件化翻转=T-0379 实况复现）：
#   D 外派→已批（撤外派改内循环重批）→有行（外派被再批覆盖回待派）
#   I 外派→领单（外部领走无再批）→零行（外派向照旧不入 DM 机派）
#   F 纯已批（非外派件）→有行（对照——常径不变形）
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
    # D 外派→已批（撤外派改内循环重批——T-0379 实况复现）
    ("COS","外派","T-9004","","2026-01-01 10:00:00.000000+00"),
    ("COS","已批","T-9004","","2026-01-01 10:05:00.000000+00"),
    # I 外派→领单（外部领走无再批——外派向照旧不入 DM 机派）
    ("COS","外派","T-9005","","2026-01-01 10:00:00.000000+00"),
    ("无头施工席","领单","T-9005","","2026-01-01 10:30:00.000000+00"),
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

has   T-9001 "A 外派+收口→有行（收口回 DM 待裁）"
has   T-9002 "B 外派全轨→收口→有行（收口面回 DM 常径）"
lacks T-9003 "C 外派+收口+裁决→零行"
has   T-9004 "D 外派→已批→有行（latest 语义——撤外派回待派,T-0379 复现）"
lacks T-9005 "I 外派→领单→零行（外派向照旧不入机派）"
has   T-9011 "E 纯收口无裁决→有行（对照）"
has   T-9012 "F 纯已批→有行（对照）"
lacks T-9013 "G 收口+裁决→零行（对照收敛）"
lacks T-9014 "H 已批+派单→零行（对照排除）"

review_out=$(bash "$ROOT/scripts/meta/dm_queue.sh" review)
approved_out=$(bash "$ROOT/scripts/meta/dm_queue.sh" approved)
if [ "$review_out" = $'T-9001\nT-9002\nT-9011' ]; then
  echo "PASS review 分面=收口待裁集"; pass=$((pass+1))
else
  echo "FAIL review 分面实出:[$review_out]"; fail=$((fail+1))
fi
# approved 面按最近已批 ts 升序：T-9012(10:00) 先于 T-9004(10:05)。
if [ "$approved_out" = $'T-9012\nT-9004' ]; then
  echo "PASS approved 分面=待派集按已批龄升序"; pass=$((pass+1))
else
  echo "FAIL approved 分面实出:[$approved_out]"; fail=$((fail+1))
fi

echo "─────"
if [ "$fail" = 0 ]; then echo "✅ dm_queue 外派豁免桩测 $pass/$((pass+fail)) 全绿"; exit 0
else echo "❌ 桩测红：$fail 败/$((pass+fail))"; exit 1; fi
