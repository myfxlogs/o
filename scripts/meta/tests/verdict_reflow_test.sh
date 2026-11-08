#!/usr/bin/env bash
# verdict_reflow_test.sh —— GW-001 段D4 归流执行机真库桩测（测试 PG 独占
# 租约+专用 scratch 库——jsonb 语义/翻案 CAS/pending 防重唯一索引全真径）。
# 场景谱：
#   ① 通过→修法票 draft 径：原子 INSERT draft+翻案 verdict_ref 携
#     DM裁#<rid>:修法票草稿draft#<id>+归流行（幂等键 ruling#<rid>）；
#   ② 通过→并入径：同族 materialized 票 latest 非完成/作废→只翻案
#     零新 draft（并入在飞票<ticket>）；
#   ③ 驳回→入案卷：status 回 pending+diag_state=rejected+attempts+1
#     +归流行；
#   ④ 驳回×3→超限：第三轮翻 verdicted+无类升件行（esc_class 空）；
#   ⑤ 幂等重放：全场景重跑一遍→零新 draft 零重翻（已归流跳过）；
#   ⑥ 双执行者并存面A（旧会话先赢）：预插归流行模拟在飞裁定席旧 SKILL
#     先归流→本机零动作（无第二 draft——幂等键先到先得实证）；
#   ⑦ 双执行者并存面B（翻案断点）：案 verdicted+verdict_ref 携 DM裁#<rid>
#     而归流行缺（翻案→归流 crash 窗）→只补归流行零重动作（draft 断点
#     续账，无第二 draft）；
#   ⑧ 并发 draft 防重唯一索引真径：同 (signature,lesion_key) pending
#     第二 INSERT 撞 fin_0148 唯一=拒（DB 级硬闸开火实证）；
#   ⑨ malformed note：无裁词机读形→留原行零动作零标记（fail-soft）。
set -uo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin"

# ── 测试 PG 租约（check_all 同式——池空=skip 不打生产） ──
PG_NAMES=$(docker ps --format '{{.Names}}' 2>/dev/null || true) # tolerate: docker 缺席/查询败=按无 test PG 处理（SKIP 径——check_all 同式）
if [ -z "$PG_NAMES" ]; then echo "SKIP 测试 PG 缺席（docker 枚举败）"; exit 0; fi
source "$ROOT/scripts/testdb_lease.sh"
testdb_lease_acquire "$PG_NAMES" || { echo "SKIP 测试库租约败"; exit 0; }
[ -n "${O_TEST_PG_DSN:-}" ] || { echo "SKIP DSN 未注入（池空 skip 径）"; exit 0; }

BASE_DSN=${O_TEST_PG_DSN%/o_test*}
DBX="o_test_reflow_$$_$(date +%s)"
psql "$BASE_DSN/postgres" -qAt -c "DROP DATABASE IF EXISTS $DBX" >/dev/null 2>&1
psql "$BASE_DSN/postgres" -qAt -c "CREATE DATABASE $DBX" >/dev/null 2>&1 \
  || { echo "SKIP scratch 库建败（实例只读/配额）"; exit 0; }
trap 'psql "$BASE_DSN/postgres" -qAt -c "DROP DATABASE IF EXISTS $DBX" >/dev/null 2>&1; rm -rf -- "$TMP"' EXIT
DSN="$BASE_DSN/$DBX?sslmode=disable"

printf 'FIN_PG_DSN=%s\n' "$DSN" > "$TMP/env"
export DM_ENV_FILE="$TMP/env"

# 桩 ops_esc：升件行落档（真径写库亦可——桩更可断言）
cat > "$TMP/bin/opsesc.sh" <<'EOF'
#!/usr/bin/env bash
printf '%s|%s|%s\n' "$2" "$5" "$6" >> "$OPS_ESC_LOG"
EOF
export O_OPS_ESC_SH="$TMP/bin/opsesc.sh"
export OPS_ESC_LOG="$TMP/esc.log"; : > "$OPS_ESC_LOG"
chmod +x "$TMP/bin/opsesc.sh"

q() { psql "$DSN" -qAt -c "$1" 2>&1; }
pass=0; fail=0
ok()  { eval "$2" && { echo "PASS $1"; pass=$((pass+1)); } \
        || { echo "FAIL $1"; fail=$((fail+1)); }; }
rfl() { bash "$ROOT/scripts/meta/verdict_reflow.sh" 2>>"$TMP/reflow.err"; }

# ── 面表（真 schema 形——ops_event/fin_econ_lesson/fin_ticket_draft+防重索引）──
q "CREATE TABLE ops_event(id bigserial PRIMARY KEY, ts timestamptz NOT NULL DEFAULT now(),
  seat text NOT NULL, event text NOT NULL, ticket text NOT NULL DEFAULT '',
  note text NOT NULL DEFAULT '', esc_class text, esc_scope text)" >/dev/null
q "CREATE TABLE fin_econ_lesson(id bigserial PRIMARY KEY, ts timestamptz NOT NULL DEFAULT now(),
  kind text NOT NULL, entity_ref text NOT NULL, numbers jsonb NOT NULL DEFAULT '{}',
  status text NOT NULL DEFAULT 'pending', verdict_ref text NOT NULL DEFAULT '',
  UNIQUE (kind, entity_ref))" >/dev/null
q "CREATE TABLE fin_ticket_draft(id bigserial PRIMARY KEY, ts timestamptz NOT NULL DEFAULT now(),
  signature text NOT NULL, kind text NOT NULL, lesion_key text NOT NULL,
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','materialized','dismissed')),
  payload jsonb NOT NULL DEFAULT '{}', ticket text NOT NULL DEFAULT '', materialized_at timestamptz)" >/dev/null
q "CREATE UNIQUE INDEX fin_ticket_draft_pending_dedup ON fin_ticket_draft (signature, lesion_key)
  WHERE status='pending' AND kind IN ('rootcause_diag','trade_postmortem')" >/dev/null

# ── fixture：两案（701 走 draft 径；702 走并入径）+驳回案（703 三轮）──
q "INSERT INTO fin_econ_lesson(id,kind,entity_ref,numbers,status) VALUES
  (701,'rootcause_diag','position#101',
   '{\"case\":{},\"sources\":[],\"seeds\":[],\"diagnosis\":{\"family\":\"bench_fam701\",\"hypothesis\":\"双制量字段消费面单位语义错配\",\"evidence\":[\"SELECT leg_no FROM fin_order_leg WHERE execution_id=9\"],\"fix_surface\":\"平仓腿 qty 读 qty_base\"}}'::jsonb,
   'drafted'),
  (702,'rootcause_diag','position#102',
   '{\"diagnosis\":{\"family\":\"unit_mismatch\",\"hypothesis\":\"同族复发案\",\"evidence\":[],\"fix_surface\":\"对拍消费谱\"}}'::jsonb,
   'drafted'),
  (703,'rootcause_diag','position#103',
   '{\"diagnosis\":{\"family\":\"novel\",\"hypothesis\":\"待考古\",\"evidence\":[],\"fix_surface\":\"\"}}'::jsonb,
   'drafted'),
  (704,'rootcause_diag','position#104','{\"diagnosis\":{\"family\":\"unit_mismatch\",\"hypothesis\":\"防重案\",\"evidence\":[],\"fix_surface\":\"\"}}'::jsonb,'drafted')" >/dev/null
# 702 并入面：同族 materialized 票在飞（latest=起草）
q "INSERT INTO fin_ticket_draft(signature,kind,lesion_key,status,ticket) VALUES
  ('rootcause:unit_mismatch','rootcause_diag','position#999','materialized','T-0500')" >/dev/null
q "INSERT INTO ops_event(seat,event,ticket,note) VALUES('CTO','起草','T-0500','x')" >/dev/null
# 裁决行（rid 由序列定——扫描后按 note 反查）
q "INSERT INTO ops_event(seat,event,ticket,note) VALUES
  ('CTO','裁决','DIAG-701','诊断草案 fin_econ_lesson#701 通过：证据链成立族标合理'),
  ('CTO','裁决','DIAG-702','诊断草案 fin_econ_lesson#702 通过：并入在飞同族判词'),
  ('CTO','驳回','DIAG-703','诊断草案 fin_econ_lesson#703 驳回：证据锚跑不回'),
  ('CTO','裁决','DIAG-704','诊断草案 fin_econ_lesson#704 通过：防重径'),
  ('CTO','裁决','T-0431','诊断草案 fin_econ_lesson#705 无裁词坏形')" >/dev/null
# 场景⑦ 断点案：翻案已落归流行缺（他 ruling 先归流形态——706）
q "INSERT INTO fin_econ_lesson(id,kind,entity_ref,numbers,status,verdict_ref) VALUES
  (706,'rootcause_diag','position#106','{\"diagnosis\":{\"family\":\"unit_mismatch\",\"hypothesis\":\"断点案\",\"evidence\":[],\"fix_surface\":\"\"}}'::jsonb,
   'verdicted','diag:unit_mismatch:DM裁#999901:修法票草稿draft#77')" >/dev/null
# 场景⑥ 并存面A：旧会话先归流（归流行在案）——新 ruling 引同案 701？不行
# （701 已 drafted 待本机裁）——独立案 707+其归流行预插。
q "INSERT INTO fin_econ_lesson(id,kind,entity_ref,numbers,status) VALUES
  (707,'rootcause_diag','position#107','{\"diagnosis\":{\"family\":\"unit_mismatch\",\"hypothesis\":\"并存案\",\"evidence\":[],\"fix_surface\":\"\"}}'::jsonb,'drafted')" >/dev/null

# 取 rid（扫描序=id 序）
rid701=$(q "SELECT id FROM ops_event WHERE note LIKE '诊断草案 fin_econ_lesson#701 %'")
rid702=$(q "SELECT id FROM ops_event WHERE note LIKE '诊断草案 fin_econ_lesson#702 %'")
rid703=$(q "SELECT id FROM ops_event WHERE note LIKE '诊断草案 fin_econ_lesson#703 %'")
rid704=$(q "SELECT id FROM ops_event WHERE note LIKE '诊断草案 fin_econ_lesson#704 %'")
rid706=999901 # 断点案 ruling 预插（归流行缺——场景⑦）
q "INSERT INTO ops_event(id,seat,event,ticket,note) VALUES
  ($rid706,'CTO','裁决','DIAG-706','诊断草案 fin_econ_lesson#706 通过：断点')" >/dev/null
rid707=999902 # 并存面A ruling+归流行预插（旧会话先赢形态——场景⑥）
q "INSERT INTO ops_event(id,seat,event,ticket,note) VALUES
  ($rid707,'CTO','裁决','DIAG-707','诊断草案 fin_econ_lesson#707 通过：旧会话已归流')" >/dev/null
q "INSERT INTO ops_event(seat,event,ticket,note) VALUES
  ('裁定席','归流','DIAG-707','ruling#$rid707 fin_econ_lesson#707 修法票草稿draft#88（旧会话径）')" >/dev/null

# ══ 第一拍：全场景同机过 ══
rfl

# ① draft 径
ok "① draft 径：draft 落+翻案携 ruling 键+归流行" '
d=$(q "SELECT id FROM fin_ticket_draft WHERE payload->> '\''ruling_id'\''='\''$rid701'\''")
[ -n "$d" ] &&
q "SELECT status FROM fin_econ_lesson WHERE id=701" | grep -qx verdicted &&
q "SELECT verdict_ref FROM fin_econ_lesson WHERE id=701" | grep -q "DM裁#${rid701}:修法票草稿draft#${d}$" &&
q "SELECT count(*) FROM ops_event WHERE event='\''归流'\'' AND note LIKE '\''ruling#${rid701} %'\''" | grep -qx 1'

# ② 并入径
ok "② 并入径：零新 draft+翻案并入在飞票" '
[ "$(q "SELECT count(*) FROM fin_ticket_draft WHERE lesion_key='\''position#102'\''")" = 0 ] &&
q "SELECT verdict_ref FROM fin_econ_lesson WHERE id=702" | grep -q "DM裁#${rid702}:并入在飞票T-0500"'

# ③ 驳回径
ok "③ 驳回径：回 pending+错因入卷+attempts=1" '
q "SELECT status FROM fin_econ_lesson WHERE id=703" | grep -qx pending &&
q "SELECT numbers->>'\''diag_state'\'',numbers->>'\''diag_attempts'\'' FROM fin_econ_lesson WHERE id=703" | grep -qx "rejected|1" &&
q "SELECT count(*) FROM ops_event WHERE event='\''归流'\'' AND note LIKE '\''ruling#${rid703} %'\''" | grep -qx 1'

# ④⑤ 超限径（第二轮+第三轮驳回行补灌后跑两拍）
q "INSERT INTO ops_event(seat,event,ticket,note) VALUES
  ('CTO','驳回','DIAG-703','诊断草案 fin_econ_lesson#703 驳回：二轮证据仍缺')" >/dev/null
rid703b=$(q "SELECT id FROM ops_event WHERE note LIKE '诊断草案 fin_econ_lesson#703 驳回：二轮%'")
q "UPDATE fin_econ_lesson SET status='drafted' WHERE id=703" >/dev/null # 再诊断回草案态
rfl
q "UPDATE fin_econ_lesson SET status='drafted' WHERE id=703" >/dev/null
q "INSERT INTO ops_event(seat,event,ticket,note) VALUES
  ('CTO','驳回','DIAG-703','诊断草案 fin_econ_lesson#703 驳回：三轮仍不成立')" >/dev/null
rid703c=$(q "SELECT id FROM ops_event WHERE note LIKE '诊断草案 fin_econ_lesson#703 驳回：三轮%'")
rfl
ok "④a 超限翻案 verdicted+ref" '
q "SELECT status||'\''|'\''||verdict_ref FROM fin_econ_lesson WHERE id=703" | grep -q "verdicted|diag:驳回超限:"'
ok "④b 无类升件行已发+DB 零升件行" '
grep -q "诊断草案驳回超限:fin_econ_lesson#703" "$OPS_ESC_LOG" &&
q "SELECT count(*) FROM ops_event WHERE event='\''升件'\''" | grep -qx 0'

# ⑥ 并存面A：旧会话先归流（归流行在）——本案零新 draft
ok "⑥ 并存面A：归流行先在→幂等跳过零动作" '
[ "$(q "SELECT count(*) FROM fin_ticket_draft WHERE lesion_key='\''position#107'\''")" = 0 ] &&
q "SELECT status FROM fin_econ_lesson WHERE id=707" | grep -qx drafted'

# ⑦ 并存面B：翻案断点（verdict_ref 携 DM裁#R706X 归流行缺）→只补归流行
ok "⑦ 并存面B：断点续账——归流行补落 draft#77 零重动作" '
q "SELECT count(*) FROM ops_event WHERE event='\''归流'\'' AND note LIKE '\''ruling#${rid706} %'\''" | grep -qx 1 &&
q "SELECT note FROM ops_event WHERE event='\''归流'\'' AND note LIKE '\''ruling#${rid706} %'\''" | grep -q "draft#77" &&
[ "$(q "SELECT count(*) FROM fin_ticket_draft WHERE payload->>'\''ruling_id'\''='\''$rid706'\''")" = 0 ]'

# ⑧ 防重唯一索引真径：同 (signature,lesion_key) pending 第二 INSERT 拒
ok "⑧ pending 防重唯一索引开火" '
q "INSERT INTO fin_ticket_draft(signature,kind,lesion_key,payload) VALUES
  ('\''rootcause:unit_mismatch'\'','\''rootcause_diag'\'','\''position#108'\'','\''{}'\''::jsonb)" >/dev/null 2>&1
q "INSERT INTO fin_ticket_draft(signature,kind,lesion_key,payload) VALUES
  ('\''rootcause:unit_mismatch'\'','\''rootcause_diag'\'','\''position#108'\'','\''{}'\''::jsonb)" >/dev/null 2>&1
[ $? -ne 0 ]'

# ⑨ malformed：坏形 ruling 留原行零标记
ok "⑨ malformed note 留原行（fail-soft 不吞）" '
[ "$(q "SELECT count(*) FROM ops_event WHERE event='\''归流'\'' AND note LIKE '\''ruling%705%'\''")" = 0 ] &&
q "SELECT count(*) FROM ops_event WHERE note LIKE '\''诊断草案 fin_econ_lesson#705%'\''" | grep -qx 1'

# ⑤ 幂等重放：再跑一拍——零新 draft 零新归流行（全键已消费）
D1=$(q "SELECT count(*) FROM fin_ticket_draft")
G1=$(q "SELECT count(*) FROM ops_event WHERE event='归流'")
rfl
D2=$(q "SELECT count(*) FROM fin_ticket_draft")
G2=$(q "SELECT count(*) FROM ops_event WHERE event='归流'")
ok "⑤ 幂等重放：draft $D1→$D2 归流行 $G1→$G2 零增" '[ "$D1" = "$D2" ] && [ "$G1" = "$G2" ]'

echo "─────"
if [ "$fail" = 0 ]; then echo "✅ verdict_reflow 真库桩测 $pass/$((pass+fail)) 全绿"; exit 0
else echo "❌ 桩测红：$fail 败/$((pass+fail))（reflow.err:$(cat "$TMP/reflow.err" 2>/dev/null)）"
  q "SELECT 'L',id,status,verdict_ref FROM fin_econ_lesson ORDER BY id;
     SELECT 'D',id,signature,lesion_key,payload->>'''ruling_id''',status,ticket FROM fin_ticket_draft ORDER BY id;
     SELECT 'E',id,event,ticket,note FROM ops_event WHERE event IN ('''归流''','''裁决''','''驳回''') ORDER BY id" | tail -40
  exit 1; fi
