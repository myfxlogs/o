#!/usr/bin/env bash
# researcher_tick_test.sh —— T-0504 研究员触发消费拍桩测（方案A钦定）：
# 真 PG scratch 库（test-pg 池端口）+自造 ops_event/app_config/ops_headless
# 最小表——全真 SQL 语义非桩面；拉起面经 O_RESEARCHER_SYSTEMD_RUN 桩
# （记录调用+可控 rc）实证「拉起调用」本体：
#   ① 合成新品行→tick 滤队命中→拉起调用实证+逐事件消行落账；
#      复拍 pending 空→零拉起（消费落终态行后不再命中——票面判据④）；
#   ② paused=true→零拉起零消行，事件留 pending；翻键 false→排空积压；
#   ③ 拉起败（桩 rc=1）→不落消行下拍重试（败不吞事件）；
#   ④ 达阈事件同型命中（event IN 闭集两值）。
set -u
ROOT=$(cd "$(dirname "$0")/../../.." && pwd)
TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
pass=0; fail=0
ok(){ echo "PASS $1"; pass=$((pass+1)); }
bad(){ echo "FAIL $1"; fail=$((fail+1)); }

# ── 桩 systemd-run：记录调用到 $STUB_LOG，rc 由 $STUB_RC 控 ──────────
STUB_LOG="$TMP/systemd-run.calls"; : > "$STUB_LOG"
cat > "$TMP/systemd-run-stub" <<'EOF'
#!/usr/bin/env bash
echo "$*" >> "$STUB_LOG_PATH"
exit "${STUB_RC:-0}"
EOF
chmod +x "$TMP/systemd-run-stub"
export STUB_LOG_PATH="$STUB_LOG" STUB_RC=0

# ── 真 PG scratch 库（test-pg 池——独立库名隔离，ceo_queue_test 同构）──
BASE_DSN="${O_TEST_PG_DSN:-postgres://o:o@localhost:5438/o_test?sslmode=disable}"
ADMIN_DSN=$(printf '%s' "$BASE_DSN" | sed 's|/o_test?|/postgres?|;s|/o_test$|/postgres|')
DBN="o_rect_t$$"
psql "$ADMIN_DSN" -qAtc "DROP DATABASE IF EXISTS $DBN WITH (FORCE)" >/dev/null 2>&1
psql "$ADMIN_DSN" -qAtc "CREATE DATABASE $DBN" >/dev/null 2>&1 \
  || { echo "SKIP: test-pg 不可达（$BASE_DSN）——researcher_tick 桩测跳过"; exit 0; }
trap 'timeout 60 psql "'"$ADMIN_DSN"'" -qAtc "DROP DATABASE IF EXISTS '"$DBN"' WITH (FORCE)" >/dev/null 2>&1; rm -rf "$TMP"' EXIT
DSN=$(printf '%s' "$BASE_DSN" | sed "s|/o_test?|/$DBN?|")
sqlq(){ psql "$DSN" -qAtc "$1"; }

psql "$DSN" -qAt <<'SQL' >/dev/null
CREATE TABLE ops_event(id bigserial PRIMARY KEY, ts timestamptz NOT NULL DEFAULT now(),
  seat text NOT NULL, event text NOT NULL, ticket text NOT NULL DEFAULT '', note text NOT NULL DEFAULT '');
CREATE TABLE app_config(key text PRIMARY KEY, value jsonb);
CREATE TABLE ops_headless(ticket text PRIMARY KEY, seat text NOT NULL DEFAULT '',
  state text NOT NULL DEFAULT '', last_activity timestamptz, updated_at timestamptz NOT NULL DEFAULT now());
SQL

TICK="$ROOT/scripts/meta/researcher_tick.sh"
ST="$TMP/tick-st"; mkdir -p "$ST"
RUN_SH="$TMP/run-stub"; printf '#!/usr/bin/env bash\nexit 0\n' > "$RUN_SH"; chmod +x "$RUN_SH"

tick(){ # $1=追加参数；env 注入 scratch DSN+桩件
  FIN_DSN="$DSN" O_RESEARCHER_SYSTEMD_RUN="$TMP/systemd-run-stub" \
    O_RESEARCHER_RUN_SH="$RUN_SH" O_RESEARCHER_TICK_ST="$ST" \
    bash "$TICK" "$@" 2>/dev/null
}

# ── 场景①：合成新品+达阈行→滤队命中→拉起调用+消行落账→复拍不命中 ──────
sqlq "INSERT INTO ops_event(seat,event,ticket,note,ts) VALUES
  ('venue哨兵','新品','META-VENUEWATCH','okx:ann:【合成演练】test listing','2026-01-01 00:00:00'),
  ('venue哨兵','达阈','META-VENUEWATCH','venue=mexc family=carry net_apr_mid=12','2026-01-01 00:00:01'),
  ('探针自检','升件','SHOULD-NOT-FIRE','非触发族对照','2026-01-01 00:00:02')" >/dev/null
L1=$(tick --list)
N_L1=$(printf '%s\n' "$L1" | awk 'NF' | wc -l)
[ "$N_L1" = 2 ] && ok "滤队命中=2（新品+达阈；升件对照拒）" || bad "滤队命中数=$N_L1 应=2：$(printf '%s' "$L1" | tr '\n' ',')"
printf '%s\n' "$L1" | grep -q 'SHOULD-NOT-FIRE' \
  && bad "非触发族事件误入滤队" || ok "非触发族事件不入滤队"

CALLS_BEFORE=$(wc -l < "$STUB_LOG")
tick >/dev/null
CALLS_AFTER=$(wc -l < "$STUB_LOG")
[ "$CALLS_AFTER" -gt "$CALLS_BEFORE" ] && ok "pending 非空→拉起调用实证" \
  || bad "pending 非空零拉起调用（消费径缺席?）"
grep -q 'o-researcher-evt-' "$STUB_LOG" && ok "瞬态单元名 o-researcher-evt-*" \
  || bad "拉起单元名不符: $(tail -1 "$STUB_LOG")"
grep -q "$RUN_SH" "$STUB_LOG" && ok "拉起目标=researcher_run.sh" \
  || bad "拉起目标不符: $(tail -1 "$STUB_LOG")"
LAST_CALL=$(tail -1 "$STUB_LOG")
NEW_ID=$(sqlq "SELECT id FROM ops_event WHERE note LIKE '%合成演练%' ORDER BY id DESC LIMIT 1")
DID=$(sqlq "SELECT id FROM ops_event WHERE note LIKE 'venue=mexc%' ORDER BY id DESC LIMIT 1")
grep -qE "(^|[ ,])${NEW_ID}(,|$)" <<< "$LAST_CALL" && grep -qE "(^|[ ,])${DID}(,|$)" <<< "$LAST_CALL" \
  && ok "拉起参数携两事件 id（$NEW_ID,$DID）" \
  || bad "拉起参数缺事件 id: $LAST_CALL"

N_DIS=$(sqlq "SELECT count(*) FROM ops_event WHERE event='消' AND seat='研究员哨兵' AND note LIKE '触发#%' ")
[ "$N_DIS" = 2 ] && ok "逐事件消行落账=2" || bad "消行数=$N_DIS 应=2"
DIS_NOTE=$(sqlq "SELECT note FROM ops_event WHERE event='消' AND note LIKE '%$NEW_ID%' LIMIT 1")
printf '%s' "$DIS_NOTE" | grep -q "触发#$NEW_ID" \
  && ok "消行 note 含 触发#<id> 引证" || bad "消行引证缺: $DIS_NOTE"

L2=$(tick --list)
[ -z "$(printf '%s' "$L2" | tr -d '[:space:]')" ] && ok "消费落终态后滤队空（判据④）" \
  || bad "消费后仍有 pending: $L2"
CALLS_MID=$(wc -l < "$STUB_LOG")
tick >/dev/null
[ "$(wc -l < "$STUB_LOG")" = "$CALLS_MID" ] && ok "复拍零拉起（不重复烧会话）" \
  || bad "复拍重复拉起"

# ── 场景②：paused=true→零拉起零消行；翻键→排空 ──────────────────────
sqlq "INSERT INTO ops_event(seat,event,ticket,note,ts) VALUES
  ('venue哨兵','新品','META-VENUEWATCH','okx:ann:paused-window test','2026-01-02 00:00:00')" >/dev/null
sqlq "INSERT INTO app_config(key,value) VALUES('research.researcher.paused','true'::jsonb)" >/dev/null
CALLS_P0=$(wc -l < "$STUB_LOG")
tick >/dev/null
[ "$(wc -l < "$STUB_LOG")" = "$CALLS_P0" ] && ok "paused=true 零拉起" \
  || bad "paused=true 仍拉起"
LP=$(tick --list)
printf '%s' "$LP" | grep -q 'paused-window' \
  && ok "paused 期事件留 pending（翻键可排空）" || bad "paused 期事件被消费: $LP"
sqlq "UPDATE app_config SET value='false'::jsonb WHERE key='research.researcher.paused'" >/dev/null
tick >/dev/null
[ "$(wc -l < "$STUB_LOG")" -gt "$CALLS_P0" ] && ok "翻键 false→积压排空拉起" \
  || bad "翻键后未拉起"

# ── 场景③：拉起败→不落消行下拍重试 ──────────────────────────────────
sqlq "INSERT INTO ops_event(seat,event,ticket,note,ts) VALUES
  ('venue哨兵','新品','META-VENUEWATCH','okx:ann:retry-window test','2026-01-03 00:00:00')" >/dev/null
DIS_BEFORE=$(sqlq "SELECT count(*) FROM ops_event WHERE event='消'")
STUB_RC=1 tick >/dev/null
[ "$(sqlq "SELECT count(*) FROM ops_event WHERE event='消'")" = "$DIS_BEFORE" ] \
  && ok "拉起败不落消行（事件不吞）" || bad "拉起败仍落消行"
STUB_RC=0 tick >/dev/null
LR=$(tick --list)
[ -z "$(printf '%s' "$LR" | tr -d '[:space:]')" ] && ok "败后重试成功消费" \
  || bad "重试后仍 pending: $LR"

echo "researcher_tick scenarios: $pass pass / $fail fail"
[ "$fail" = 0 ]
