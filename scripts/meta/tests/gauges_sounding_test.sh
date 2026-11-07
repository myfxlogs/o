#!/usr/bin/env bash
# gauges_sounding_test.sh —— T-0437 升件老化探针桩测（⑧mode=连鸣 律面）：
# ①红桩：升件行挂起>阈（count≥1）→首拍即鸣（跳滞回——不需连续2拍）
# ②同日次拍续鸣（跳日频上限1——冻结必须持续发声）
# ③阈内静默（count=0 零行）→阈上连鸣恢复
# ④{d} 明细携老化行 id 清单（处置可对行）
# ⑤默认律回归（无⑧列行：滞回2拍+日频1 原语义零漂移）
# 桩面：TMP TSV+psql 桩（RO 取数按查询指纹答数/INSERT 捕获）+
# O_GAUGES_ST 状态目录覆写——不碰产径 /var/tmp/o-gauges、不碰真库。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d); trap 'rm -rf -- "$TMP"' EXIT
ST="$TMP/state"; mkdir -p "$ST"

# ── 桩 TSV：连鸣行（升件老化判据同形查询指纹）+默认行（旧律对照）──
TSV="$TMP/gauges.tsv"
cat > "$TSV" <<EOF
g_test_sounding	缺席	sql_ro	SELECT count(*) FROM ops_event e WHERE e.note NOT LIKE '升件老化%'	1	升件老化 count={v}（{d}）	SELECT string_agg(e.id::text,',' ORDER BY e.id) FROM ops_event e WHERE e.note NOT LIKE '升件老化%'	连鸣
g_test_default	淤积	sql_ro	SELECT count(*) FROM fin_proposal WHERE state='fin_armed'	1	default 淤积={v}
EOF

# ── psql 桩：RO 取数按查询指纹答数；INSERT 捕获不落真库 ──────────
INSERTS="$TMP/inserts"; : > "$INSERTS"
cat > "$TMP/psql" <<EOF
#!/usr/bin/env bash
set -euo pipefail
query=\${*: -1}
case "\$query" in
  *"count(*) FROM ops_event"*) cat "$TMP/count_ops" ;;
  *"string_agg(e.id"*) cat "$TMP/ids" ;;
  *"count(*) FROM fin_proposal"*) cat "$TMP/count_prop" ;;
  *"INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope) VALUES('巡检闸'"*|*"VALUES('巡检闸','升件'"*)
    printf '%s\n' "\$query" >> "$INSERTS" ;;
  *"INSERT INTO ops_headless"*) : ;;
  *) echo "unexpected psql: \$query" >&2; exit 2 ;;
esac
EOF
chmod +x "$TMP/psql"
printf 'FIN_PG_DSN=fake-fin\nFIN_PG_RO_DSN=fake-ro\n' > "$TMP/env"

run_tick() { # 一拍（状态目录/env/tsv 全覆写——零产径接触）
  ( env PATH="$TMP:$PATH" GAUGES_ENV_FILE="$TMP/env" \
      GAUGES_TSV="$TSV" O_GAUGES_ST="$ST" \
      bash "$ROOT/scripts/meta/gauges_tick.sh" )
}
fail() { echo "❌ $*" >&2; exit 1; }

# ①② 连鸣律——count=1 恒答（冻结态）：首拍即鸣+次拍续鸣（同日破日频1）
printf '1\n' > "$TMP/count_ops"; printf '17,18\n' > "$TMP/ids"
printf '0\n' > "$TMP/count_prop"
run_tick; run_tick
N=$(awk '/INSERT INTO ops_event/ {n++} END {print n+0}' "$INSERTS") # awk 计数零 rc 面（grep -c 零匹配退1需 tolerate 位——测试非容错面不占册）
[ "$N" -eq 2 ] || fail "①②连鸣两拍应 2 行实得 $N"
grep -qF "'g_test_sounding','升件老化 count=1" "$INSERTS" || fail "①鸣行 note 缺计数"
grep -qF "count=1（17,18）" "$INSERTS" || fail "①{d} 明细缺老化行 id 清单"
grep -qF "'缺席','g_test_sounding'" "$INSERTS" || fail "①带类面缺（esc_class/esc_scope）"

# ③ 阈内静默：count=0 零新增行——阈上恢复连鸣
: > "$INSERTS"; printf '0\n' > "$TMP/count_ops"
run_tick
N=$(awk '/INSERT INTO ops_event/ {n++} END {print n+0}' "$INSERTS") # awk 计数零 rc 面（grep -c 零匹配退1需 tolerate 位——测试非容错面不占册）
[ "$N" -eq 0 ] || fail "③阈内应静默实得 $N 行"
printf '1\n' > "$TMP/count_ops"
run_tick
N=$(awk '/INSERT INTO ops_event/ {n++} END {print n+0}' "$INSERTS") # awk 计数零 rc 面（grep -c 零匹配退1需 tolerate 位——测试非容错面不占册）
[ "$N" -eq 1 ] || fail "③阈上恢复应 1 行实得 $N"

# ⑤ 默认律回归（无⑧列）——滞回2拍+日频1 原语义：首拍静默/造昨日痕→次拍鸣/同日再拍静默
: > "$INSERTS"; printf '0\n' > "$TMP/count_ops" # 连鸣行归零——⑤专判默认行旧律
run_tick # 首拍：count_prop=0 不触——置触值再拍仍静默（无昨日痕）
printf '5\n' > "$TMP/count_prop"
run_tick
N=$(awk '/INSERT INTO ops_event/ {n++} END {print n+0}' "$INSERTS") # awk 计数零 rc 面（grep -c 零匹配退1需 tolerate 位——测试非容错面不占册）
[ "$N" -eq 0 ] || fail "⑤滞回首拍应静默实得 $N 行"
Y=$(date -d yesterday +%F)
echo "$Y" > "$ST/g_test_default.last_hit_date"
run_tick
N=$(awk '/INSERT INTO ops_event/ {n++} END {print n+0}' "$INSERTS") # awk 计数零 rc 面（grep -c 零匹配退1需 tolerate 位——测试非容错面不占册）
[ "$N" -eq 1 ] || fail "⑤滞回第二拍应鸣实得 $N 行"
grep -q "default 淤积=5 阈=1" "$INSERTS" || fail "⑤默认行 note 缺计数"
run_tick
N=$(awk '/INSERT INTO ops_event/ {n++} END {print n+0}' "$INSERTS") # awk 计数零 rc 面（grep -c 零匹配退1需 tolerate 位——测试非容错面不占册）
[ "$N" -eq 1 ] || fail "⑤日频上限应压第三拍实得 $N 行"

echo "gauges_sounding_test 全绿"
