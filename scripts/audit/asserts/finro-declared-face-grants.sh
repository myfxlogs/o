#!/usr/bin/env bash
# 审计锚 2026-11-25: grant-base-forget-view 判例第二发——法证宣告面
# o_fin_ro SELECT 缺位列。fin_0106 白名单制钦定:读面逐面显式钉;
# fin_0170/0226/0229 补迁先例=「新建宣告面须携读授」已是裁定义务。
# 本列=宣告面(对账/判词/豁免/报告读面),非内写面(白名单豁免域)。
# pg_catalog 普查——information_schema 对无权对象自盲(本场实证坑)。
set -u
DSN="${FIN_PG_DSN:?need FIN_PG_DSN}"
declared="fin_report fin_pnl_net fin_park_baseline fin_known_residual fin_exec_drill fin_guard_close fin_strategy_transition"
miss=""
for t in $declared; do
  ok=$(psql "$DSN" -qAtc "SELECT has_table_privilege('o_fin_ro','$t','SELECT')")
  [ "$ok" = "t" ] || miss="$miss $t"
done
if [ -n "$miss" ]; then
  echo "RED: 法证宣告面 o_fin_ro 读权缺位——$miss" >&2
  exit 1
fi
echo "GREEN: 法证宣告面列 o_fin_ro 读权在案"
