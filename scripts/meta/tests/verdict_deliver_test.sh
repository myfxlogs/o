#!/usr/bin/env bash
# verdict_deliver_test.sh —— T-0438 判词送达闸桩测（真库租约径——幂等键/
# 部分唯一索引 upsert/送达标记语义 mock 不可替；池空=skip 径不猜不占，
# testdb_lease fail-closed 同律）。schema=t_0438vd（t_* 命名域——gc_test_
# schemas 清扫面自收）。
# 场景谱（票面验收①②③+对行防连推/词→kind 映射/推送败重试/已标记跳过）：
#   ① 人工径词裁决行→箱行落(executor)+推送行出(mock)+送达标记行落；
#   ② 同行重拍=零重复（箱行/推送/标记三面幂等断言——票面验收②）；
#   ③ 非人工径词裁决行=零转写零推送（票面验收③——不误报业主）；
#   ④ 裁决+通过成对双行（同票同 note，T-0424 实据形态）→单转写单推送；
#   ⑤ 凭据缺口词→kind=credential；
#   ⑥ 推送败=零标记+rc≠0；恢复后单送达（箱 upsert 刷新零重行）；
#   ⑨ 箱写败=零推送零标记 rc≠0（半途假送达禁——前置闸）；
#   ⑦ 已有标记行=永跳（历史件不重送）；
#   ⑧ manual_lane_check 单测：rc0 人工径票/rc1 可派/rc2 票件缺/rc3 用法错。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin"

# —— 真库租约（check_all 同源件——池空=skip 径）——
# shellcheck source=/dev/null
source "$ROOT/scripts/testdb_lease.sh"
PG_NAMES=$(docker ps --format '{{.Names}}' 2>/dev/null || true) # tolerate: docker 缺席=池空 skip 径（fail-closed 不猜不占）
testdb_lease_acquire "$PG_NAMES"
if [ -z "${O_TEST_PG_DSN:-}" ]; then
	echo "⚠️ [verdict_deliver_test] 测试 PG 池空——真库场景 skip（不猜不占）"
	exit 0
fi
SCHEMA=t_0438vd
TDSN="${O_TEST_PG_DSN}&options=-csearch_path%3D${SCHEMA}"

psql "$TDSN" -q -v ON_ERROR_STOP=1 <<'SQL'
DROP SCHEMA IF EXISTS t_0438vd CASCADE;
CREATE SCHEMA t_0438vd;
CREATE TABLE ops_event(
  id bigserial PRIMARY KEY,
  ts timestamptz NOT NULL DEFAULT now(),
  seat text NOT NULL,
  event text NOT NULL,
  ticket text NOT NULL DEFAULT '',
  note text NOT NULL DEFAULT '');
CREATE TABLE fin_condition_gap(
  id bigserial PRIMARY KEY,
  kind text NOT NULL CHECK (kind IN ('capital','credential','permission','constraint','holding','executor')),
  venue text,
  surface text,
  payload jsonb NOT NULL DEFAULT '{}',
  state text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending','fulfilled','rejected','dismissed')),
  requested_at timestamptz NOT NULL DEFAULT now(),
  resolved_at timestamptz,
  resolve_note text NOT NULL DEFAULT '');
CREATE UNIQUE INDEX fin_condition_gap_pending_uq
  ON fin_condition_gap(venue,surface,kind) WHERE state='pending';
CREATE TABLE ops_headless(
  ticket text PRIMARY KEY,
  seat text NOT NULL,
  state text NOT NULL,
  last_activity timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now());
SQL

# —— 种子判词行（形态照 T-0424 实据：裁决+通过成对同 note；行 id 1..7）——
psql "$TDSN" -qAt <<'SQL'
INSERT INTO ops_event(seat,event,ticket,note) VALUES
 ('决策席','裁决','T-9001','裁COS升件执行径:机派驳回——执行=人工径(业主在场) 残留=earn赎回+全袋市价清→USDT 执毕收口回DM常径裁'),
 ('决策席','通过','T-9001','裁COS升件执行径:机派驳回——执行=人工径(业主在场) 残留=earn赎回+全袋市价清→USDT 执毕收口回DM常径裁'),
 ('决策席','裁决','T-9002','通过——处置终了全锚兑现:venue实拍清零敞口归零'),
 ('决策席','驳回','T-9002','通过——处置终了全锚兑现:venue实拍清零敞口归零'),
 ('决策席','裁决','T-9003','凭据缺口——ibkr adapter 凭据缺业主落地后开接'),
 ('决策席','裁决','T-9004','执行=人工径(业主在场)——预标记历史件'),
 ('决策席','裁决','T-9005','执行=人工径(业主在场)——推送败重试件');
INSERT INTO ops_event(seat,event,ticket,note) VALUES ('META-DM','送达','','ref=6|gap=0|push=ok|T-9004|executor 判词送达(T-0438)');
SQL

cat > "$TMP/bin/mock-notify" <<'EOF'
#!/usr/bin/env bash
if [ "${NOTIFY_FAIL:-0}" = 1 ]; then exit 1; fi
printf '%s\n' "$1" >> "$NOTIFY_LOG"
EOF
chmod +x "$TMP/bin/mock-notify"

printf 'FIN_PG_DSN=%s\n' "$TDSN" > "$TMP/env"
export VERDICT_DELIVER_ENV_FILE="$TMP/env"
export O_VERDICT_BOX_DSN="$TDSN"
export O_VERDICT_NOTIFY_SH="$TMP/bin/mock-notify"
export NOTIFY_LOG="$TMP/notify.log"
VD="$ROOT/scripts/meta/verdict_deliver.sh"

box_count() { psql "$TDSN" -qAt -c "SELECT count(*) FROM fin_condition_gap"; }
box_row()   { psql "$TDSN" -qAt -c "SELECT kind||'/'||coalesce(payload->>'ref_event_id','?') FROM fin_condition_gap WHERE venue='$1'"; }
mark_count(){ psql "$TDSN" -qAt -c "SELECT count(*) FROM ops_event WHERE event='送达'"; }

# ①④⑤ 首拍：T-9001 对行单转写(ref=1,executor)+T-9003(ref=5,credential)
# +T-9005(ref=7,executor)；T-9002 非词零转写；T-9004(ref=6)已标记永跳⑦。
RC=0; OUT=$("$VD" 2>&1) || RC=$?
echo "$OUT"
[ "$RC" -eq 0 ]
[ "$(box_count)" = 3 ]
grep -q '^executor/1$' <(box_row T-9001)
grep -q '^credential/5$' <(box_row T-9003)
grep -q '^executor/7$' <(box_row T-9005)
[ "$(box_row T-9002)" = "" ]
[ "$(mark_count)" = 4 ] # 种子标记1（T-9004 跳过不新增）+新落3（T-9001/03/05）
[ "$(wc -l < "$NOTIFY_LOG")" = 3 ]
grep -q '^⚠ 需业主：T-9001 ' "$NOTIFY_LOG"
grep -q '^⚠ 需业主：T-9003 ' "$NOTIFY_LOG"
psql "$TDSN" -qAt -c "SELECT (payload->>'ticket')||'|'||(payload->>'source') FROM fin_condition_gap WHERE venue='T-9001'" | grep -qx 'T-9001|T-0438'

# ② 重拍=三面零重复（幂等断言——票面验收②）
"$VD" >/dev/null 2>&1
[ "$(box_count)" = 3 ]
[ "$(mark_count)" = 4 ]
[ "$(wc -l < "$NOTIFY_LOG")" = 3 ]

# ③ 非人工径词裁决行零转写零推送（票面验收③——不误报业主）
! grep -q 'T-9002' "$NOTIFY_LOG"

# ⑥ 推送败=零标记+rc≠0；恢复后单送达零重箱行（T-9005 重试件清场重演）
psql "$TDSN" -qAt -c "DELETE FROM fin_condition_gap WHERE venue='T-9005'" >/dev/null
psql "$TDSN" -qAt -c "DELETE FROM ops_event WHERE event='送达' AND note LIKE 'ref=7|%'" >/dev/null
: > "$NOTIFY_LOG"
RC=0; NOTIFY_FAIL=1 "$VD" >"$TMP/fail.log" 2>&1 || RC=$?
cat "$TMP/fail.log"
[ "$RC" -eq 3 ]
[ "$(box_count)" = 3 ]  # 箱行照落（T-9005 重种——箱写先于推送）
[ "$(mark_count)" = 3 ] # T-9005 零标记（推送败不落——下拍重试）
[ "$(wc -l < "$NOTIFY_LOG")" = 0 ]
"$VD" >/dev/null 2>&1   # 推送恢复——单送达
[ "$(box_count)" = 3 ]  # upsert 刷新零重行
[ "$(mark_count)" = 4 ]
[ "$(wc -l < "$NOTIFY_LOG")" = 1 ]

# ⑨ 箱写败=零推送零标记 rc≠0（半途假送达禁——前置闸断言；新种子未标记件走箱败径）
psql "$TDSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
  VALUES('决策席','裁决','T-9006','执行=人工径(业主在场)——箱败径测试件')" >/dev/null
: > "$NOTIFY_LOG"
BEFORE_BOX=$(box_count); BEFORE_MARK=$(mark_count)
O_VERDICT_BOX_DSN="postgres://nobody:nobody@localhost:1/none" "$VD" >"$TMP/boxfail.log" 2>&1 \
  && { echo "FAIL: 箱写败应 rc≠0"; exit 1; } || RC=$?
cat "$TMP/boxfail.log"
[ "$RC" -eq 3 ]
[ "$(box_count)" = "$BEFORE_BOX" ]
[ "$(mark_count)" = "$BEFORE_MARK" ]
[ "$(wc -l < "$NOTIFY_LOG")" = 0 ]

# ⑧ manual_lane_check 单测（真文件判据；位置参票目录——零部署树耦合）
ML="$ROOT/scripts/meta/manual_lane_check.sh"
TD="$TMP/lane"; mkdir -p "$TD"
printf '# T-8001 真钱卖袋\n- 执行=人工径（业主在场）\n' > "$TD/T-8001-卖袋件.md"
printf '# T-8002 码件\n## 范围\n1. scripts/x.sh\n' > "$TD/T-8002-码件.md"
bash "$ML" T-8001 "$TD"
! bash "$ML" T-8002 "$TD" # rc1 可派
! bash "$ML" T-8003 "$TD" # rc2 票件缺
RC=0; bash "$ML" >/dev/null 2>&1 || RC=$?
[ "$RC" -eq 3 ] # rc3 用法错

echo "✅ verdict_deliver_test 全场景过（①②③④⑤⑥⑦⑧⑨）"
