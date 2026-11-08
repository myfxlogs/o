#!/usr/bin/env bash
# heartbeat_dead_caliber_test.sh —— T-0457 死道口径修真库对拍（票面③桩/对拍件）：
# scratch schema 造具 ops_headless/ops_event 行，gauges.tsv heartbeat_dead 行
# SQL 原样抽出执行（PGOPTIONS search_path 隔离——不碰产表），断言死道集
# =应活未活道：常驻道（META-svc/o-*/席名自名）+未终态近态票道入数；
# 终态票道/无账道/近态尽道全剔。用后 DROP SCHEMA——零残留。
# O_TEST_PG_DSN 缺席=skip 径（同 Go 桩约定 pgguard_test）；psql 缺席同 skip。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
DSN=${O_TEST_PG_DSN:-}
[ -n "$DSN" ] || { echo "SKIP: O_TEST_PG_DSN 未设——独立测试实例缺席"; exit 0; }
command -v psql >/dev/null 2>&1 || { echo "SKIP: psql 缺席"; exit 0; }

SCH=o_gauge_hbdead_caliber
fail() { echo "❌ $*" >&2; exit 1; }

psql "$DSN" -qAt -c "DROP SCHEMA IF EXISTS $SCH CASCADE; CREATE SCHEMA $SCH" >/dev/null
trap 'psql "$DSN" -qAt -c "DROP SCHEMA IF EXISTS '"$SCH"' CASCADE" >/dev/null 2>&1 || true' EXIT # tolerate: 清理末环——DROP 尽力而为，残 schema 由 gc_test_schemas 兜底非语义（同族规 ops_esc 自报末环例）

psql "$DSN" -qAt -v ON_ERROR_STOP=1 >/dev/null <<'SQL'
SET search_path = o_gauge_hbdead_caliber;
CREATE TABLE ops_headless(ticket text PRIMARY KEY, seat text NOT NULL DEFAULT '',
  state text NOT NULL DEFAULT '', last_activity timestamptz);
CREATE TABLE ops_event(id bigserial PRIMARY KEY, seat text, event text,
  ticket text, ts timestamptz NOT NULL DEFAULT now());
-- 应计面：常驻道死（META-svc×2/o-*/席名自名）+活票道死（近态无终态）
INSERT INTO ops_headless(ticket,seat,state,last_activity) VALUES
 ('META-GAUGES','巡检闸','alive', now()-interval '8 days'),
 ('META-VERDICT','裁定席','alive', now()-interval '9 days'),
 ('o-testsvc','哨兵','alive',       now()-interval '8 days'),
 ('CTO','CTO','exit',               now()-interval '10 days'),
 ('META-Dev-T-0500','施工席','alive', now()-interval '8 days'),
 ('T-0501','Dev','done',            now()-interval '8 days'),
 ('T-0403','Dev','done',            now()-interval '30 days');
-- 应剔面：终态票道（裁决/完成）/无账道/近态尽道/活道（<7d 不入死集）
INSERT INTO ops_headless(ticket,seat,state,last_activity) VALUES
 ('META-Dev-T-0400','施工席','alive', now()-interval '30 days'),
 ('T-0401','Dev','done',            now()-interval '30 days'),
 ('DBP2','DM','done',               now()-interval '20 days'),
 ('T-PROBE1','Dev','dead',          now()-interval '20 days'),
 ('META-Dev-T-0402','施工席','alive', now()-interval '30 days'),
 ('META-DM','决策席','alive',       now()-interval '1 days'),
 ('T-0600','Dev','alive',           now()-interval '1 days');
INSERT INTO ops_event(seat,event,ticket,ts) VALUES
 ('x','裁决','T-0400', now()-interval '25 days'),  -- 终态→剔
 ('x','完成','T-0401', now()-interval '25 days'),  -- 终态→剔
 ('x','裁决','T-0402', now()-interval '20 days'),  -- 终态优先于近态→剔
 ('x','施工中','T-0402', now()-interval '2 days'),
 ('x','领单','T-0500', now()-interval '2 days'),   -- 近态无终态→计
 ('x','领单','T-0501', now()-interval '3 days'),   -- 近态无终态→计
 ('x','领单','T-0403', now()-interval '2 days'),   -- 近态无终态→计
 ('x','领单','DBP2',  now()-interval '20 days'),   -- 仅老事件→剔
 ('x','领单','T-0600', now()-interval '1 days');
SQL

# gauges.tsv heartbeat_dead 行 SQL 原样抽出（字段4=query——tab 分列不猜位）
SQL_Q=$(awk -F'\t' '$1=="heartbeat_dead" {print $4; exit}' "$ROOT/scripts/meta/gauges.tsv")
[ -n "$SQL_Q" ] || fail "gauges.tsv heartbeat_dead 行缺/字段错"

V=$(PGOPTIONS="-c search_path=$SCH" psql "$DSN" -qAt -c "$SQL_Q")
[ "$V" = "7" ] || fail "死道计数应=7（4常驻+3活票道）实得=${V:-空}"

# 成员对拍：同 SQL 头换 SELECT ticket 出道名行集（WHERE 原样不动——行输出
# 经 sort|paste 归一逗号清单，免 delimiter 引号进 shell 展开坑）
SQL_D=${SQL_Q/SELECT count\(\*\)/SELECT ticket}
[ "$SQL_D" != "$SQL_Q" ] || fail "SQL 头替换失败——抽出串形漂移"
GOT=$(PGOPTIONS="-c search_path=$SCH" psql "$DSN" -qAt -c "$SQL_D" | sort | paste -sd, -)
WANT=$(printf '%s\n' CTO META-Dev-T-0500 META-GAUGES META-VERDICT T-0403 T-0501 o-testsvc | sort | paste -sd, -)
[ "$GOT" = "$WANT" ] || fail "死道集不符——实得=$GOT 期望=$WANT"

echo "✅ heartbeat_dead 口径对拍过——常驻道+活票道入数，终态/无账/尽态道全剔"
