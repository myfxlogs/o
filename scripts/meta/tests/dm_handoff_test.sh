#!/usr/bin/env bash
# dm_handoff_test.sh —— T-0459 交接对账哨兵桩测：psql 全桩（扫描窗谓词钉+
# 场景驱动；真 escalate.sh 配桩 psql 写径落日志）。六面（票面④钦定）：
# ①超阈报（note 携 #交行id+龄/阈双数）②接行落零报（接行谓词钉）③同交行
# 复跑零重（evergreen——桩按查询内 LIKE 字面对写径日志判重，壳写词形与
# SQL 查词形不自洽即炸）④终态票零报（终态杀票谓词钉）⑤词表命中零报
# ⑥龄未超阈零报（壳侧双检）。外围：扫描败=本体败 exit1（T-0372）/配对表
# 缺=exit1/FIN_DSN 缺=静默退/混合多交行同拍各一报/全表 7 对各查一拍。
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin"

cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
query="${*: -1}"
case "$query" in
  *"NOT EXISTS (SELECT 1 FROM ops_event r"*)
    # 交接哨兵扫描窗——谓词钉（CTO 裁定判据 SQL 齐位，缺一即桩败 exit 9）：
    pin() { grep -Fq "$1" <<<"$query" || { echo "stub: 谓词钉缺失: $1" >&2; exit 9; }; }
    [ "${PSQL_SCAN_RC:-0}" -eq 0 ] || exit "$PSQL_SCAN_RC" # 本体败场景（T-0372：败≠空集）
    pin "r.ts>e.ts OR (r.ts=e.ts AND r.id>e.id)"              # ②接行：同刻 id 大者序后
    pin "LIKE '断链:"                                         # ③evergreen 去重
    pin "e.id::text"                                          # ③evergreen：交行 id 锚（关联式）
    pin "IN ('完成','驳回','作废','处置')"                     # ④终态杀票
    pin "!~ '(作废|测试票|人工径)'"                            # ⑤免报词表
    pin "interval '604800 seconds'"                           # 7d 扫窗
    pin "e.ticket<>''"                                        # 零票不配对
    pin "l.event IN ("                                        # ④生命周期域（表驱动）
    ev=$(sed -nE "s/.*e\.event='([^']*)'.*/\1/p" <<<"$query")
    [ -n "$ev" ] || { echo "stub: 扫描窗无交event" >&2; exit 9; }
    printf '%s\n' "$ev" >> "$SCAN_EV_LOG"
    printf '%s\n' "$query" >> "$SCAN_SQL_LOG"
    [ -n "${HANDOFF_ROWS:-}" ] || exit 0
    printf '%b\n' "$HANDOFF_ROWS" | while IFS= read -r row; do
      [ -n "$row" ] || continue
      IFS=$'\x1f' read -r rev rid tid seat note age <<<"$row"
      [ "$rev" = "$ev" ] || continue
      # ③evergreen 桩判重：查询内 LIKE 字面（断链:<ev>:）×行（tid,#rid）对
      # 写径日志判——壳写词形与 SQL 查词形不自洽时复跑重报、本面即炸。
      if grep -Fq "断链:${ev}:${tid}——交行#${rid}" "$ESC_SQL_LOG" 2>/dev/null; then continue; fi
      printf '%s\x1f%s\x1f%s\x1f%s\x1f%s\n' "$rid" "$tid" "$seat" "$note" "$age"
    done
    ;;
  *"INSERT INTO ops_event"*)
    printf '%s\n' "$query" >> "$ESC_SQL_LOG" # escalate.sh 升件写径（seat=CTO 无类）
    ;;
esac
EOF
chmod +x "$TMP/bin/psql"

printf 'FIN_PG_DSN=fake\n' > "$TMP/env"
: > "$TMP/env-empty"
export PATH="$TMP/bin:$PATH"
export DM_ENV_FILE="$TMP/env"
export ESCALATE_ENV_FILE="$TMP/env"
export O_HANDOFF_ESCALATE_SCRIPT="$ROOT/scripts/meta/escalate.sh"
export O_HANDOFF_PAIRS_TSV="$ROOT/scripts/meta/handoff_pairs.tsv"
export SCAN_EV_LOG="$TMP/scan_ev.log"
export SCAN_SQL_LOG="$TMP/scan.sql.log"
export ESC_SQL_LOG="$TMP/esc.sql.log"

DM_SH="$ROOT/scripts/meta/dm_handoff.sh"
reset_logs() { : > "$SCAN_EV_LOG"; : > "$SCAN_SQL_LOG"; : > "$ESC_SQL_LOG"; }
esc_count() { grep -c "INSERT INTO ops_event" "$ESC_SQL_LOG" || true; } # tolerate: grep 零行=计 0 合法径

# ① 超阈报：派单行龄 4000s>阈 1800s——note 携 #交行id + 龄/阈双数（66m/30m）
reset_logs
HANDOFF_ROWS='派单\x1f77\x1fT-9101\x1f决策席\x1f派单 T-9101 读票面施工\x1f4000' bash "$DM_SH"
[[ $(esc_count) -eq 1 ]]
grep -Fq "VALUES('CTO','升件','T-9101','断链:派单:T-9101——交行#77 决策席 「派单 T-9101 读票面施工」龄66m>阈30m'" "$ESC_SQL_LOG"

# ② 接行落零报：接行谓词已钉（同刻 id 大者序后）——空集场景零升件；
#    派单行接行集按表落 SQL（接行集字面钉）
reset_logs
HANDOFF_ROWS='' bash "$DM_SH"
[[ $(esc_count) -eq 0 ]]
grep -Fq "r.event IN ('领单','拉起','候水','驳回','挂起','作废','完成','处置')" "$SCAN_SQL_LOG" # T-0463：候水入派单接行集

# ③ 同交行复跑零重（evergreen）：收口行 9000s>阈 7200s 首跑一报；
#    复跑同交行——桩按 SQL 查词形对已落账词形判重=零再报
reset_logs
R88='收口\x1f88\x1fT-9102\x1f施工席\x1f收口 T-9102 全链绿\x1f9000'
HANDOFF_ROWS="$R88" bash "$DM_SH"
[[ $(esc_count) -eq 1 ]]
grep -Fq "断链:收口:T-9102——交行#88 施工席 「收口 T-9102 全链绿」龄150m>阈120m" "$ESC_SQL_LOG"
HANDOFF_ROWS="$R88" bash "$DM_SH"
[[ $(esc_count) -eq 1 ]]

# ④ 终态票零报：终态杀票谓词钉+生命周期域含表侧交∪接（外派/已并码实证在册）
reset_logs
HANDOFF_ROWS='' bash "$DM_SH"
[[ $(esc_count) -eq 0 ]]
grep -Fq "IN ('完成','驳回','作废','处置')" "$SCAN_SQL_LOG"
grep -Fq "'外派'" "$SCAN_SQL_LOG"
grep -Fq "'已并码'" "$SCAN_SQL_LOG"

# ⑤ 词表命中零报：免报词表谓词钉（作废|测试票|人工径）——空集场景零升件
reset_logs
HANDOFF_ROWS='' bash "$DM_SH"
[[ $(esc_count) -eq 0 ]]
grep -Fq "!~ '(作废|测试票|人工径)'" "$SCAN_SQL_LOG"

# ⑥ 龄未超阈零报：桩径直回幼龄行（100s<阈 3600s——模拟扫描/取数间时钟漂）
#    ——壳侧双检必拦
reset_logs
HANDOFF_ROWS='呈批\x1f99\x1fT-9103\x1fCOS\x1f呈批 T-9103\x1f100' bash "$DM_SH"
[[ $(esc_count) -eq 0 ]]

# ⑦ 扫描败=本体败 exit 1（T-0372：败≠空集——拍壳升件「交接哨兵本体败」）
reset_logs
RC=0; PSQL_SCAN_RC=9 HANDOFF_ROWS='' bash "$DM_SH" || RC=$?
[[ $RC -eq 1 ]]

# ⑧ 配对表缺=exit 1（配置败 fail-loud 禁静默跳=盲区）
RC=0; O_HANDOFF_PAIRS_TSV=/nonexistent.tsv bash "$DM_SH" || RC=$?
[[ $RC -eq 1 ]]

# ⑨ FIN_DSN 缺=静默退（拍壳同律）——零查询零升件
reset_logs
RC=0; DM_ENV_FILE="$TMP/env-empty" bash "$DM_SH" || RC=$?
[[ $RC -eq 0 ]]
[[ ! -s "$SCAN_EV_LOG" && ! -s "$ESC_SQL_LOG" ]]

# ⑩ 混合多交行同拍各一报（不同交event 各自断链互不吞）
reset_logs
HANDOFF_ROWS='派单\x1f71\x1fT-9104\x1f决策席\x1f派单 T-9104\x1f4000\n移交\x1f72\x1fT-9105\x1fCTO\x1f移交 T-9105\x1f4000' bash "$DM_SH"
[[ $(esc_count) -eq 2 ]]
grep -Fq "'升件','T-9104'" "$ESC_SQL_LOG"
grep -Fq "'升件','T-9105'" "$ESC_SQL_LOG"

# ⑪ 全表 7 对各查一拍（配对表 CTO 裁定行数钉——表变本面随改）
reset_logs
HANDOFF_ROWS='' bash "$DM_SH"
[[ $(wc -l < "$SCAN_EV_LOG") -eq 7 ]]
for ev in 派单 移交 收口 收口待审 呈批 已批 外派; do grep -Fqx "$ev" "$SCAN_EV_LOG"; done

echo "dm handoff sentinel scenarios PASS"
