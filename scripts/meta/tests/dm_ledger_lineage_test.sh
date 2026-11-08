#!/usr/bin/env bash
# dm_ledger_lineage_test.sh —— DGM 谱系投影桩测（提案-DGM变异档案取子集②）。
# 真 PG scratch 库（test-pg 池/外部 DSN 径）+真 git 仓 fixture 自造
# （docs/prompts 返修件）。场景：
#   ① 人造驳回→返修链→--lineage 得 v1 死法/v2 改法两行（判据①）；
#   ② 三量字段随行落账可查：死法类/复跑墙钟/断言增减（判据②口径——
#     复跑墙钟=驳回→下版讫 ts 现算，断言增减=返修件/驳回 token）；
#   ③ 零新表零写——投影只读：ops_event 行数不变、库表数不变、
#     无 git 写（判据③：谱系=投影非存储）；
#   ④ 存量面零扰：无驳回票单版谱系；--lineage 坏参 rc=2；
#     all/flip/reconcile 模式判别不被 --lineage 误吞。
set -u
ROOT=$(cd "$(dirname "$0")/../../.." && pwd)
TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
pass=0; fail=0
ok(){ echo "PASS $1"; pass=$((pass+1)); }
bad(){ echo "FAIL $1"; fail=$((fail+1)); }

# ── ① scratch PG ──
BASE_DSN="${O_TEST_PG_DSN:-postgres://o:o@localhost:5438/o_test?sslmode=disable}"
ADMIN_DSN=$(printf '%s' "$BASE_DSN" | sed -E 's#(postgres(ql)?://[^/]+/)[^/?]+#\1postgres#')
DBN="o_tlin_t$$"
psql "$ADMIN_DSN" -qAtc "DROP DATABASE IF EXISTS $DBN WITH (FORCE)" >/dev/null 2>&1
psql "$ADMIN_DSN" -qAtc "CREATE DATABASE $DBN" >/dev/null 2>&1 \
  || { echo "SKIP: test-pg 不可达（$BASE_DSN）"; exit 0; }
trap 'timeout 60 psql "'"$ADMIN_DSN"'" -qAtc "DROP DATABASE IF EXISTS '"$DBN"' WITH (FORCE)" >/dev/null 2>&1; rm -rf "$TMP"' EXIT
DSN=$(printf '%s' "$BASE_DSN" | sed -E "s#(postgres(ql)?://[^/]+/)[^/?]+#\1$DBN#")
psql "$DSN" -qAt <<'SQL' >/dev/null
CREATE TABLE ops_event(id bigserial PRIMARY KEY, ts timestamptz NOT NULL DEFAULT now(),
  seat text NOT NULL, event text NOT NULL, ticket text NOT NULL DEFAULT '', note text NOT NULL DEFAULT '');
SQL
sqlq(){ psql "$DSN" -qAtc "$1"; }

# ── ② fixture 仓：docs/prompts 返修件（## 变体差量 节三行）──
REPO="$TMP/repo"; mkdir -p "$REPO/docs/prompts" "$REPO/docs/tasks"
cat > "$REPO/docs/prompts/T-9001-返修1.md" <<'EOF'
# T-9001 返修1——迁移号让位

对象：t/9001-x 当前收口。

## 变体差量
- 上案死法：fin_0147 撞号未让位（死法类=码面）
- 本案改法：迁移改号 fin_0149 登记 migrate_table
- 断言增减：+0/-0
EOF
printf 'FIN_PG_DSN=%s\n' "$DSN" > "$TMP/env"
export DM_ENV_FILE="$TMP/env" O_LEDGER_REPO="$REPO" O_LEDGER_WT_ROOT="$TMP/wt"

# ── ③ 人造票链：v1 领单→收口→驳回(带〔变体差量〕)→v2 派单→收口→裁决 ──
sqlq "INSERT INTO ops_event(seat,event,ticket,note,ts) VALUES
 ('CTO','领单','T-9001','取径A根修','2026-11-08 10:00'),
 ('施工','收口','T-9001','径A落地','2026-11-08 10:30'),
 ('决策席','移交','T-9001','待验','2026-11-08 10:31'),
 ('CTO','驳回','T-9001','驳回：迁移号撞。〔变体差量〕死法类=码面；死因=fin_0147与在案撞号；改法=改号fin_0149；断言增减=+0/-0','2026-11-08 10:35'),
 ('CTO','派单','T-9001','返修1即派','2026-11-08 10:36'),
 ('施工','收口','T-9001','返修1让位','2026-11-08 11:06'),
 ('决策席','移交','T-9001','返修1待验','2026-11-08 11:07'),
 ('CTO','裁决','T-9001','通过','2026-11-08 11:20')" >/dev/null
# 干扰票：无驳回单版链
sqlq "INSERT INTO ops_event(seat,event,ticket,note,ts) VALUES
 ('CTO','领单','T-9002','取径B','2026-11-08 09:00'),
 ('施工','收口','T-9002','径B落地','2026-11-08 09:40')" >/dev/null

ROWS_BEFORE=$(sqlq "SELECT count(*) FROM ops_event")
TBL_BEFORE=$(sqlq "SELECT count(*) FROM information_schema.tables WHERE table_schema='public'")

OUT=$(bash "$ROOT/scripts/meta/dm_ledger.sh" --lineage T-9001 2>&1); rc=$?
[ $rc -eq 0 ] && ok "① --lineage rc=0" || bad "① rc=$rc out=$OUT"
# v1 死法行：死法类/死因/改法/断言增减四量在案
printf '%s' "$OUT" | grep -q '死法类=码面' && printf '%s' "$OUT" | grep -q '死因=fin_0147与在案撞号' \
  && printf '%s' "$OUT" | grep -q '改法=改号fin_0149' && printf '%s' "$OUT" | grep -q '断言增减=+0/-0' \
  && ok "② v1 死法四量随行落账可查" || bad "② 死法面缺——$OUT"
# v2 改法行：返修1 件差量回填上案死法/本案改法/断言增减
printf '%s' "$OUT" | grep -q '返修1 件' && printf '%s' "$OUT" | grep -q '上案死法=fin_0147 撞号未让位' \
  && printf '%s' "$OUT" | grep -q '本案改法=迁移改号 fin_0149 登记 migrate_table' \
  && ok "③ v2 返修件差量两栏在案" || bad "③ 返修面缺——$OUT"
# 复跑墙钟=驳回10:35→v2讫11:20=45m（投影现算）
printf '%s' "$OUT" | grep -q '复跑墙钟(驳回→v2讫)=45m' \
  && ok "④ 复跑墙钟投影现算=45m" || bad "④ 墙钟异——$(printf '%s' "$OUT" | grep 复跑)"
# 谱系两版行
printf '%s' "$OUT" | grep -q '^v1 ' && printf '%s' "$OUT" | grep -q '^v2 ' \
  && printf '%s' "$OUT" | grep -q '谱系 v1..v2' \
  && ok "⑤ v1..v2 两版行齐" || bad "⑤ 版行缺——$OUT"

# ── ④ 零新表零写：投影后行数/表数不变 ──
[ "$(sqlq 'SELECT count(*) FROM ops_event')" = "$ROWS_BEFORE" ] \
  && [ "$(sqlq "SELECT count(*) FROM information_schema.tables WHERE table_schema='public'")" = "$TBL_BEFORE" ] \
  && ok "⑥ 零新表零写（行数/表数不变）" || bad "⑥ 投影有写——谱系非投影？"

# ── ⑤ 存量面零扰 ──
OUT2=$(bash "$ROOT/scripts/meta/dm_ledger.sh" --lineage T-9002 2>&1)
printf '%s' "$OUT2" | grep -q '谱系 v1..v1' && ! printf '%s' "$OUT2" | grep -q '死@驳回' \
  && ok "⑦ 无驳回票单版谱系（无死行）" || bad "⑦ 单版面异——$OUT2"
bash "$ROOT/scripts/meta/dm_ledger.sh" --lineage T-XXXX >/dev/null 2>&1; rc=$?
[ $rc -eq 2 ] && ok "⑧ 坏参 rc=2" || bad "⑧ rc=$rc 期=2"
# 模式判别不误吞：--lineage 之外仍走原三模（用法面不炸）
bash "$ROOT/scripts/meta/dm_ledger.sh" bogus >/dev/null 2>&1; rc=$?
[ $rc -eq 2 ] && ok "⑨ 原模式面不变（bogus→rc2）" || bad "⑨ rc=$rc"

echo "─────"
if [ "$fail" = 0 ]; then echo "✅ dm_ledger_lineage 桩测 $pass/$((pass+fail)) 全绿"; exit 0
else echo "❌ 桩测红：$fail 败/$((pass+fail))"; exit 1; fi
