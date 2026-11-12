#!/usr/bin/env bash
# 审计/销案复审 RED 锚（audit/1113-close）：工单销案判据=引证行事件词表约束——
# 「挂起条件=」引证行不得在 stall/OPEN 面计为销票（挂起=开着但缓办,非完结;
# 挂票过期须回队复验）。生产实证:工单#11481 被「接续」行(#11769,无挂起词)
# +挂起行(#11771)引用=runbook OPEN 查询与 codework_stall 双双视已销,
# 票载缺口(日K数据面缺席#11773)未闭零跟进=死票;feed_esc tickEnded 判据
# 却认挂起未过期=未终——同语义两处判据不一致(维度A)。
# 正确语义=stall/OPEN 销票 EXISTS 排除挂起条件行(或要求销词)。修好断言自然绿。
set -u
FAIL=0
S=/opt/o/w 2>/dev/null; S=cmd/core/supply_esc.go
[ -f "$S" ] || S=/opt/o-wt/audit-1113-close/cmd/core/supply_esc.go
# ① codework_stall 销票 EXISTS 须剔挂起条件行（NOT LIKE '%挂起条件=%' 或等效闭词）
STALL=$(awk '/codework_stall|工单.*过期|stall/{f=1} f&&/NOT EXISTS|挂起条件|工单#/{print NR": "$0}' "$S" | head -8)
if grep -n "挂起条件" "$S" | grep -q "NOT LIKE\|NOT.*LIKE\|!<.*LIKE"; then
  echo "GREEN①: stall 销票面剔挂起条件行"
else
  echo "RED①: codework_stall EXISTS 判据未剔「挂起条件=」引证——挂票被视同销票,过期永不复验（活例:工单#11481 挂起+接续引证=stall 哑死）"
  FAIL=1
fi
# ② runbook OPEN 查询同款剔挂起（文档与码面一致）
R=scripts/cognition/runbook.md
if grep -n "NOT EXISTS.*处置" "$R" >/dev/null && grep -A3 "NOT EXISTS.*处置" "$R" | grep -q "挂起"; then
  echo "GREEN②: runbook OPEN 查询剔挂起引证"
else
  echo "RED②: runbook OPEN 查询任意处置引证即剔 OPEN——挂起/接续行均致票从施工队列消失（「挂起三选一」钦定面被谓词抹掉）"
  FAIL=1
fi
exit $FAIL
