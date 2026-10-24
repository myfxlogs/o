#!/usr/bin/env bash
# check_no_forbidden_patterns.sh —— D-0008 业主令扩展：工程面禁用快照/定时任务/轮询。
# A 面：决策域文件引用快照表当现况即拒（原 check_no_decision_snapshot）。
# B 面：全 internal/+cmd/+tools/ 新增行引入周期调度原语即拒——
#       time.NewTicker / time.Tick / time.Sleep 循环拉取属轮询病灶；
#       期界等待/退避/心跳确属语义所需者须行内 `// sched-exempt-ok <理由>` 豁免留痕。
set -euo pipefail
fail=0

# A: 决策域禁读快照
for d in internal/risk/ internal/trading/execution/ tools/openfirst/; do
  hits=$(git diff --cached -- "$d" | grep -nE "^\+" | grep -vE "snapshot-audit-ok" | grep -E "fin_venue_account_snapshot|md_venue_fact|fin_ledger_snapshot" || true)
  if [ -n "$hits" ]; then
    echo "BLOCK(D-0008-A): $d 新增行读快照表当现况——venue 动态事实必现读："
    echo "$hits" | head -5; fail=1
  fi
done

# B: 全域禁新增周期调度原语
for d in internal/ cmd/ tools/; do
  hits=$(git diff --cached --name-only -- "$d" | { grep -E '\.go$' || true; } | while read -r f; do
    git diff --cached -- "$f" | grep -nE "^\+" | grep -vE "sched-exempt-ok" \
      | grep -E "time\.NewTicker|time\.Tick\(|time\.After\(|time\.Sleep\(" \
      | sed "s|^|$f:|" || true
  done)
  if [ -n "$hits" ]; then
    echo "BLOCK(D-0008-B): 新增周期调度原语（定时/轮询=禁用方法）——事件驱动/流订阅为正解，确属期界等待或退避者行内 sched-exempt-ok 豁免："
    echo "$hits" | head -8; fail=1
  fi
done
exit $fail
