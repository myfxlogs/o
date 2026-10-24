#!/usr/bin/env bash
# check_no_decision_snapshot.sh —— D-0008：决策域禁读快照当现况。
# 扫 staged 新增行：决策域文件引用快照/fact 表名即拒（审计/报表域豁免）。
set -euo pipefail
DOMAINS='internal/risk/ internal/trading/execution/ tools/openfirst/'
TABLES='fin_venue_account_snapshot|md_venue_fact|fin_ledger_snapshot'
fail=0
for d in $DOMAINS; do
  hits=$(git diff --cached -- "$d" | grep -nE "^\+" | grep -vE "snapshot-audit-ok" | grep -E "$TABLES" || true)
  if [ -n "$hits" ]; then
    echo "BLOCK(D-0008): 决策域 $d 新增行引用快照表当现况——venue 动态事实必现读："
    echo "$hits" | head -5
    fail=1
  fi
done
exit $fail
