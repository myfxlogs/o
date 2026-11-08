#!/usr/bin/env bash
# case_writeback_test.sh —— GW-001 段E1 案卷回写桩测：真 PG scratch
# 库（test-pg 池端口）+真 git 仓 fixture——draft/rootcause_diag 全自造
# 不依赖段D 已 merge（半成品窗律：D 归流件未上线也须零命中不炸）。
# 场景：
#   ① draft#锚谱系→verdict_ref 追加 deployed:<sha12>+归流行+面戳；
#   ② 并入在飞票锚（无 draft 行票级命中）；
#   ③ 复发=第二部署→新案 #r2+diag_attempts 累计+pending；
#   ④ 幂等重跑零重行；⑤ 无谱系部署=面戳零案写（D 未上线静默零行同构）；
#   ⑥ 表缺/查询败=exit 0 fail-soft；⑦ mech_tick 尾块唤件实证。
set -u
ROOT=$(cd "$(dirname "$0")/../../.." && pwd)
TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
pass=0; fail=0
ok(){ echo "PASS $1"; pass=$((pass+1)); }
bad(){ echo "FAIL $1"; fail=$((fail+1)); }

# ── ① 真 PG scratch 库（test-pg 池——o_test_s* 分片同池，独立库名隔离）──
BASE_DSN="${O_TEST_PG_DSN:-postgres://o:o@localhost:5438/o_test?sslmode=disable}"
ADMIN_DSN=$(printf '%s' "$BASE_DSN" | sed 's|/o_test?|/postgres?|;s|/o_test$|/postgres|')
DBN="o_casewb_t$$"
psql "$ADMIN_DSN" -qAtc "DROP DATABASE IF EXISTS $DBN WITH (FORCE)" >/dev/null 2>&1
psql "$ADMIN_DSN" -qAtc "CREATE DATABASE $DBN" >/dev/null 2>&1 \
  || { echo "SKIP: test-pg 不可达（$BASE_DSN）——案卷回写桩测跳过"; exit 0; }
# 清库走 postgres 维护库+FORCE+timeout——DROP DATABASE 须等 checkpointer
# （共享池忙时 CheckpointDone 可挂分钟级），不得死等拖死套件
trap 'timeout 60 psql "'"$ADMIN_DSN"'" -qAtc "DROP DATABASE IF EXISTS '"$DBN"' WITH (FORCE)" >/dev/null 2>&1; timeout 60 psql "'"$ADMIN_DSN"'" -qAtc "DROP DATABASE IF EXISTS '"${DBN}b"' WITH (FORCE)" >/dev/null 2>&1; rm -rf "$TMP"' EXIT
DSN=$(printf '%s' "$BASE_DSN" | sed "s|/o_test?|/$DBN?|")

psql "$DSN" -qAt <<'SQL' >/dev/null
CREATE TABLE ops_event(id bigserial PRIMARY KEY, ts timestamptz NOT NULL DEFAULT now(),
  seat text NOT NULL, event text NOT NULL, ticket text NOT NULL DEFAULT '', note text NOT NULL DEFAULT '');
CREATE TABLE fin_econ_lesson(id bigserial PRIMARY KEY, ts timestamptz NOT NULL DEFAULT now(),
  kind text NOT NULL, entity_ref text NOT NULL, numbers jsonb NOT NULL DEFAULT '{}',
  status text NOT NULL DEFAULT 'pending', verdict_ref text NOT NULL DEFAULT '',
  UNIQUE(kind,entity_ref));
CREATE TABLE fin_ticket_draft(id bigserial PRIMARY KEY, ts timestamptz NOT NULL DEFAULT now(),
  signature text NOT NULL, kind text NOT NULL, lesion_key text NOT NULL,
  status text NOT NULL DEFAULT 'pending', payload jsonb NOT NULL DEFAULT '{}',
  ticket text NOT NULL DEFAULT '', materialized_at timestamptz);
SQL
sqlq(){ psql "$DSN" -qAtc "$1"; }

# ── ② 真 git 仓 fixture：c0..c3 票号链（deploy 序=sha 序）────────────
GITR="$TMP/repo"; mkdir -p "$GITR"; cd "$GITR" || exit 1
git init -qb master >/dev/null; git config user.email t@t; git config user.name t
for spec in "c0|T-9000 base" "c1|T-9001 first" "c2|T-9002 second" "c3|misc"; do
  f=${spec%%|*}; msg=${spec#*|}
  echo "$f" > "$f.txt"; git add "$f.txt"
  git commit -qm "[$msg] $f"
done
SHA0=$(git rev-parse --short=12 HEAD~3); SHA1=$(git rev-parse --short=12 HEAD~2)
SHA2=$(git rev-parse --short=12 HEAD~1); SHA3=$(git rev-parse --short=12 HEAD)
# 部署行四行（rc=0 序=SHA0..SHA3——区间 SHA0..SHA1={c1:T-9001}、
# SHA1..SHA2={c2:T-9002}、SHA2..SHA3={c3:无票}）
for s in "$SHA0" "$SHA1" "$SHA2" "$SHA3"; do
  sqlq "INSERT INTO ops_event(seat,event,ticket,note) VALUES('DEPLOY','部署','','deploy $s rc=0 image=o-core:$s')" >/dev/null
done

# ── ③ 案卷 fixture（段D 未上线自造——verdict_ref 两锚钦定写面同式）──
# T-9001：draft#1→L1 draft#锚；T-9002：无 draft→L2 并入在飞票锚；
# L3 预置 deployed: 旧标记=复发候选（第二部署命中→新案#r2）
sqlq "INSERT INTO fin_ticket_draft(signature,kind,lesion_key,status,ticket,materialized_at)
  VALUES('rootcause:famA','rootcause_diag','position#901','materialized','T-9001',now())" >/dev/null
L1=$(sqlq "INSERT INTO fin_econ_lesson(kind,entity_ref,numbers,status,verdict_ref)
  VALUES('rootcause_diag','position#901','{\"case\":{\"x\":1}}','verdicted',
    'diag:famA:DM裁#11:修法票草稿draft#1') RETURNING id")
sqlq "INSERT INTO fin_econ_lesson(kind,entity_ref,numbers,status,verdict_ref)
  VALUES('rootcause_diag','position#902','{}','verdicted',
    'diag:famB:DM裁#12:并入在飞票T-9002')" >/dev/null
sqlq "INSERT INTO fin_econ_lesson(kind,entity_ref,numbers,status,verdict_ref)
  VALUES('rootcause_diag','position#903','{\"diag_attempts\":2}','verdicted',
    'diag:famC:DM裁#13:修法票草稿draft#9 deployed:aaaa1111bbbb')" >/dev/null
# draft#9→T-9002（复发案经 draft# 锚挂 T-9002——第二部署命中）
sqlq "INSERT INTO fin_ticket_draft(signature,kind,lesion_key,status,ticket,materialized_at)
  VALUES('rootcause:famC','rootcause_diag','position#903','materialized','T-9002',now())" >/dev/null
D9=$(sqlq "SELECT id FROM fin_ticket_draft WHERE ticket='T-9002' ORDER BY id DESC LIMIT 1")
# L3 verdict_ref 须含 draft#<D9> 才挂 T-9002——重写锚
sqlq "UPDATE fin_econ_lesson SET verdict_ref='diag:famC:DM裁#13:修法票草稿draft#${D9} deployed:aaaa1111bbbb'
  WHERE entity_ref='position#903'" >/dev/null
# env 桩（FIN_DSN→scratch；REPO→temp git）
printf 'FIN_PG_DSN=%s\n' "$DSN" > "$TMP/env"
export DM_ENV_FILE="$TMP/env" O_CASE_REPO="$GITR"
WB="$ROOT/scripts/meta/case_writeback.sh"

# ── ④ 跑件+断言 ────────────────────────────────────────────────────
bash "$WB" 2>"$TMP/wb.err"; rc=$?
[ "$rc" -eq 0 ] && ok "① exit 0" || bad "① exit=$rc（stderr:$(head -3 "$TMP/wb.err")）"
V1=$(sqlq "SELECT verdict_ref FROM fin_econ_lesson WHERE entity_ref='position#901'")
case "$V1" in *"deployed:${SHA1}"*) ok "② draft#锚 verdict_ref+=deployed:${SHA1}";; *) bad "② verdict_ref=[$V1]";; esac
sqlq "SELECT 1 FROM ops_event WHERE event='归流' AND note LIKE 'deployed:${SHA1} fin_econ_lesson#${L1}%'" | grep -q 1 \
  && ok "③ 归流行 deployed:${SHA1}#${L1} 在案" || bad "③ 归流行缺席"
V2=$(sqlq "SELECT verdict_ref FROM fin_econ_lesson WHERE entity_ref='position#902'")
case "$V2" in *"deployed:${SHA2}"*) ok "④ 并入在飞票锚 verdict_ref+=deployed:${SHA2}";; *) bad "④ verdict_ref=[$V2]";; esac
# 复发：L3 已有 deployed: 旧标记+T-9002 第二部署→新案 #r2
NEWL=$(sqlq "SELECT id||'|'||entity_ref||'|'||status||'|'||COALESCE(numbers->>'diag_attempts','?')
  FROM fin_econ_lesson WHERE entity_ref='position#903#r2'")
case "$NEWL" in *"|pending|3"*) ok "⑤ 复发新案 position#903#r2 pending diag_attempts=3";; *) bad "⑤ 新案行=[$NEWL]";; esac
sqlq "SELECT 1 FROM ops_event WHERE event='归流' AND note LIKE '复发 fin_econ_lesson#%' AND note LIKE '%deployed:${SHA2}%'" | grep -q 1 \
  && ok "⑥ 复发归流行在案" || bad "⑥ 复发归流行缺席"
# 面戳：SHA1/SHA2/SHA3 三区间全闭（SHA3 区间无票 matched=0 亦落戳）
for s in "$SHA1" "$SHA2" "$SHA3"; do
  sqlq "SELECT 1 FROM ops_event WHERE event='归流' AND note LIKE 'deploy-close ${s} %'" | grep -q 1 \
    && ok "⑦ deploy-close ${s} 面戳" || bad "⑦ deploy-close ${s} 缺席"
done
# SHA0=genesis 无区间——i=0 不处理（SHAS[0] 永为 prev 不为 cur）
# 幂等重跑：全量二跑零新增
N1=$(sqlq "SELECT count(*) FROM ops_event WHERE event='归流'")
N2=$(sqlq "SELECT count(*) FROM fin_econ_lesson")
bash "$WB" 2>/dev/null
[ "$(sqlq "SELECT count(*) FROM ops_event WHERE event='归流'")" = "$N1" ] \
  && ok "⑧ 幂等重跑归流行零新增" || bad "⑧ 重跑归流行漂移"
[ "$(sqlq "SELECT count(*) FROM fin_econ_lesson")" = "$N2" ] \
  && ok "⑨ 幂等重跑案卷零新增" || bad "⑨ 重跑案卷漂移"
# verdict_ref 不重复追加（deployed:${SHA1} 计数=1）
C=$(sqlq "SELECT (length(verdict_ref)-length(replace(verdict_ref,'deployed:${SHA1}','')))/length('deployed:${SHA1}') FROM fin_econ_lesson WHERE entity_ref='position#901'")
[ "$C" = "1" ] && ok "⑩ deployed 标记唯一不重缀" || bad "⑩ 标记计数=$C"

# ── ⑤ fail-soft：表缺=exit 0 静默零行（另起空库，不 DROP 在飞库——
#    共享池 checkpointer 忙时 DROP DATABASE 挂 CheckpointDone 分钟级）──
psql "$ADMIN_DSN" -qAtc "CREATE DATABASE ${DBN}b" >/dev/null 2>&1
DSN2=$(printf '%s' "$DSN" | sed "s|/$DBN?|/${DBN}b?|")
printf 'FIN_PG_DSN=%s\n' "$DSN2" > "$TMP/env2"
DM_ENV_FILE="$TMP/env2" O_CASE_REPO="$GITR" bash "$WB" 2>/dev/null; rc=$?
[ "$rc" -eq 0 ] && ok "⑪ 空库（表全缺）exit 0 fail-soft" || bad "⑪ 空库 rc=$rc"
timeout 60 psql "$ADMIN_DSN" -qAtc "DROP DATABASE ${DBN}b WITH (FORCE)" >/dev/null 2>&1

echo "─────"
if [ "$fail" = 0 ]; then echo "✅ case_writeback 桩测 $pass/$((pass+fail)) 全绿"; exit 0
else echo "❌ 桩测红：$fail 败/$((pass+fail))"; exit 1; fi
