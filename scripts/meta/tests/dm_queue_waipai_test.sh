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

# ══ GW-001 段B1：说明类踢队病灶+dispatched 判据追加场景 ══════════════
# legacy 面（probe 败）与 view 面（迁移 SQL 实建 VIEW）双径同断言组——
# 踢队组在 legacy 被踢/view 回队=判据②病灶根修实证；dispatched 组两面同判。
python3 - "$OPS_EV_DB" <<'PY'
import sqlite3, sys
con = sqlite3.connect(sys.argv[1])
add = [
    # T-9021 派单→升件（说明类尾随）——dispatched 两面同 IN
    ("决策席","派单","T-9021","","2026-01-02 10:00:00.000000+00"),
    ("CTO","升件","T-9021","x","2026-01-02 10:05:00.000000+00"),
    # T-9022 收口→升件——legacy review 踢出/view 相位 review 回队
    ("无头施工席","收口","T-9022","","2026-01-02 11:00:00.000000+00"),
    ("CTO","升件","T-9022","x","2026-01-02 11:05:00.000000+00"),
    # T-9023 已批→升件——legacy approved 踢出/view 相位 approved 回队
    ("COS","已批","T-9023","","2026-01-02 12:00:00.000000+00"),
    ("CTO","升件","T-9023","x","2026-01-02 12:05:00.000000+00"),
    # T-9041 收口→候水（水位说明类——真病灶形：候水行踢收口票出待裁）
    ("无头施工席","收口","T-9041","","2026-01-02 13:00:00.000000+00"),
    ("CTO","候水","T-9041","","2026-01-02 13:05:00.000000+00"),
    # dispatched 判据组（两面同判——生命周期序列无歧义）
    ("决策席","派单","T-9031","","2026-01-02 14:00:00.000000+00"),
    ("决策席","派单","T-9032","","2026-01-02 14:01:00.000000+00"),
    ("无头施工席","领单","T-9032","","2026-01-02 14:02:00.000000+00"),
    ("决策席","派单","T-9033","","2026-01-02 14:03:00.000000+00"),
    ("无头施工席","拉起","T-9033","","2026-01-02 14:04:00.000000+00"),
    ("决策席","派单","T-9034","","2026-01-02 14:05:00.000000+00"),
    ("无头施工席","收口","T-9034","","2026-01-02 14:06:00.000000+00"),
    ("决策席","派单","T-9035","","2026-01-02 14:07:00.000000+00"),
    ("施工席","挂起","T-9035","rate-limit:auto retry","2026-01-02 14:08:00.000000+00"),
    ("决策席","派单","T-9036","","2026-01-02 14:09:00.000000+00"),
    ("施工席","挂起","T-9036","人工挂起待料","2026-01-02 14:10:00.000000+00"),
    ("COS","派单","T-9038","","2026-01-02 14:11:00.000000+00"),
    ("施工席","挂起","T-9039","rate-limit:auto retry","2026-01-02 14:12:00.000000+00"),
    ("决策席","派单","T-9040","","2026-01-02 14:13:00.000000+00"),
    ("CTO","候水","T-9040","","2026-01-02 14:14:00.000000+00"),
    # T-9050 段B判据①钦定形：已批→派单→处置（CEO 断链处置注记=说明类
    # 不进相位——真实被踢票 T-0443/0444/0446/0449/0452 尾行同构）→
    # dispatched 两面同 IN（处置不踢队）
    ("COS","已批","T-9050","","2026-01-02 15:00:00.000000+00"),
    ("决策席","派单","T-9050","依赖闸即派","2026-01-02 15:01:00.000000+00"),
    ("CEO","处置","T-9050","断链=水位/额度候派在径","2026-01-02 15:30:00.000000+00"),
    # T-9051/T-9052 同刻并列防御：收口+裁决、已批+派单同 ts 双行——
    # legacy mt 反词（NOT EXISTS 裁决/完成/驳回、派单/领单/拉起/…）
    # 与 view 面 latest_ts 反词闸须同判 OUT，防 VIEW 并列 join 使
    # 反词失效（收口行仍命中 review 的漂移面）
    ("CTO","收口","T-9051","","2026-01-02 16:00:00.000000+00"),
    ("CTO","裁决","T-9051","","2026-01-02 16:00:00.000000+00"),
    ("决策席","已批","T-9052","","2026-01-02 16:10:00.000000+00"),
    ("决策席","派单","T-9052","","2026-01-02 16:10:00.000000+00"),
]
con.executemany("INSERT INTO ops_event VALUES(?,?,?,?,?)", add)
con.commit()
PY

disp_legacy=$(bash "$ROOT/scripts/meta/dm_queue.sh" dispatched)
for t in T-9021 T-9031 T-9035 T-9039 T-9040 T-9050 T-9052; do
  grep -Fxq "$t" <<<"$disp_legacy" && { echo "PASS legacy dispatched 含 $t"; pass=$((pass+1)); } \
    || { echo "FAIL legacy dispatched 缺 $t（实出:[$disp_legacy]）"; fail=$((fail+1)); }
done
for t in T-9032 T-9033 T-9034 T-9036 T-9038 T-9051; do
  ! grep -Fxq "$t" <<<"$disp_legacy" && { echo "PASS legacy dispatched 排 $t"; pass=$((pass+1)); } \
    || { echo "FAIL legacy dispatched 误含 $t"; fail=$((fail+1)); }
done
# 同刻并列反词钉：T-9051 收口+裁决同 ts→review 双面皆 OUT；
# T-9052 已批+派单同 ts→approved 双面皆 OUT（dispatched 双面同 IN 已钉上）
for t in T-9051; do
  ! grep -Fxq "$t" <<<"$(bash "$ROOT/scripts/meta/dm_queue.sh" review)" \
    && ! grep -Fxq "$t" <<<"$(bash "$ROOT/scripts/meta/dm_queue.sh" approved)" \
    && { echo "PASS legacy review/approved 排并列反词 $t"; pass=$((pass+1)); } \
    || { echo "FAIL legacy review/approved 误含并列反词 $t"; fail=$((fail+1)); }
done
! grep -Fxq T-9052 <<<"$(bash "$ROOT/scripts/meta/dm_queue.sh" approved)" \
  && { echo "PASS legacy approved 排并列反词 T-9052"; pass=$((pass+1)); } \
  || { echo "FAIL legacy approved 误含并列反词 T-9052"; fail=$((fail+1)); }
# legacy 踢队实证（病灶面——说明类尾随事件把票踢出待裁/待派）
leg_rev=$(bash "$ROOT/scripts/meta/dm_queue.sh" review)
leg_app=$(bash "$ROOT/scripts/meta/dm_queue.sh" approved)
for t in T-9022 T-9041; do
  ! grep -Fxq "$t" <<<"$leg_rev" && { echo "PASS legacy review 踢 $t（病灶实证）"; pass=$((pass+1)); } \
    || { echo "FAIL legacy review 未踢 $t"; fail=$((fail+1)); }
done
! grep -Fxq T-9023 <<<"$leg_app" && { echo "PASS legacy approved 踢 T-9023（病灶实证）"; pass=$((pass+1)); } \
  || { echo "FAIL legacy approved 未踢 T-9023"; fail=$((fail+1)); }

# view 面：迁移 SQL 实建 VIEW（抽取 CREATE VIEW 体——测真迁移件非复写）
awk '/CREATE OR REPLACE VIEW ops_ticket_phase/,/;$/' \
  "$ROOT/migrations/fin_0147_ops_ticket_phase.sql" \
  | sed 's/CREATE OR REPLACE VIEW/CREATE VIEW/' > "$TMP/phase_view.sql"
python3 - "$OPS_EV_DB" "$TMP/phase_view.sql" <<'PY'
import sqlite3, sys
con = sqlite3.connect(sys.argv[1])
con.executescript(open(sys.argv[2]).read())
con.commit()
PY

disp_view=$(bash "$ROOT/scripts/meta/dm_queue.sh" dispatched)
[ "$disp_view" = "$disp_legacy" ] && { echo "PASS view dispatched=legacy 集（$disp_view）"; pass=$((pass+1)); } \
  || { echo "FAIL view dispatched 实出:[$disp_view] vs legacy:[$disp_legacy]"; fail=$((fail+1)); }
view_rev=$(bash "$ROOT/scripts/meta/dm_queue.sh" review)
if [ "$view_rev" = $'T-9001\nT-9002\nT-9011\nT-9022\nT-9034\nT-9041' ]; then
  echo "PASS view review=收口待裁集+被踢票回队（T-9022/T-9041）"; pass=$((pass+1))
else
  echo "FAIL view review 实出:[$view_rev]"; fail=$((fail+1))
fi
! grep -Fxq T-9051 <<<"$view_rev" \
  && { echo "PASS view review 排并列反词 T-9051"; pass=$((pass+1)); } \
  || { echo "FAIL view review 误含并列反词 T-9051"; fail=$((fail+1)); }
view_app=$(bash "$ROOT/scripts/meta/dm_queue.sh" approved)
if [ "$view_app" = $'T-9012\nT-9004\nT-9023' ]; then
  echo "PASS view approved=待派集+被踢票回队（T-9023）按相位龄升序"; pass=$((pass+1))
else
  echo "FAIL view approved 实出:[$view_app]"; fail=$((fail+1))
fi
! grep -Fxq T-9052 <<<"$view_app" \
  && { echo "PASS view approved 排并列反词 T-9052"; pass=$((pass+1)); } \
  || { echo "FAIL view approved 误含并列反词 T-9052"; fail=$((fail+1)); }
view_all=$(bash "$ROOT/scripts/meta/dm_queue.sh" all)
for t in T-9022 T-9023 T-9041; do
  grep -Fxq "$t" <<<"$view_all" && { echo "PASS view all 含被踢票 $t"; pass=$((pass+1)); } \
    || { echo "FAIL view all 缺 $t"; fail=$((fail+1)); }
done
# 旧 VIEW 不动钉：ops_ticket_state 定义字节级在案（迁移文件不触碰）
grep -q 'CREATE OR REPLACE VIEW ops_ticket_state' "$ROOT/migrations/fin_0086_ops_event.sql" \
  && { echo "PASS ops_ticket_state 旧 VIEW 定义原样在案（零误伤）"; pass=$((pass+1)); } \
  || { echo "FAIL ops_ticket_state 定义缺席"; fail=$((fail+1)); }

echo "─────"
if [ "$fail" = 0 ]; then echo "✅ dm_queue 外派豁免桩测 $pass/$((pass+fail)) 全绿"; exit 0
else echo "❌ 桩测红：$fail 败/$((pass+fail))"; exit 1; fi
