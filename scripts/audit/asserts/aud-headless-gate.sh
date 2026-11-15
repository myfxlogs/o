#!/usr/bin/env bash
# 审计/唤知环 RED 锚（audit/1115-cogring）：audit dispatch 跨机在飞判据须含
# ops_headless 心跳腿（AUD- 票缀），与 cognition dispatch 同构——唤起行窗
# 30min < RuntimeMaxSec 5400s，长会话唤起行过期即失在飞保护：30~90min
# 窗内二次 dispatch（手唤/跨机）=审计臂双活双账（cognition T-0521 同型
# 坑，85.35 双唤实证 2026-11-12）。run.sh 已为 AUD 会话写 ops_headless
# 心跳（HB_PREFIX=AUD-，60s 续期）——产了没人吃=死数黑洞。
# 正确语义=dispatch.sh 在飞 SQL 含 ops_headless 行且票键限 '^AUD-'；
# 修好断言自然绿。
set -u
F=${1:-scripts/audit/dispatch.sh}
[ -f "$F" ] || { echo "assert: dispatch.sh 缺席 $F"; exit 2; }

# ① ops_headless 消费腿在案否（在飞判据第二 NOT EXISTS 块，cognition
#    dispatch.sh 同构件——h.ticket ~ '^COG-'）。
if ! grep -q "ops_headless" "$F"; then
	echo "assert RED: audit/dispatch.sh 在飞判据零 ops_headless 消费——唤起行 30min 窗过期后 30~90min（RuntimeMaxSec 5400s）长会话无心跳保护，二次 dispatch=审计臂双活（cognition 85.35 双唤同型坑未移植）"
	exit 1
fi
# ② 心跳票键须钉 AUD- 前缀（裸 ops_headless 存在性谓词会把 COG- 会话
#    也算在飞=审计臂被唤知臂误拦——槽位令 AUD 独立槽不互拦）。
if ! grep -qE "ticket\s*~\s*'\^AUD-'" "$F"; then
	echo "assert RED: ops_headless 消费腿未钉 '^AUD-' 票缀——唤知/独白心跳将误拦审计槽（槽位令2026-11-14 AUD 独立槽不互拦被反吃）"
	exit 1
fi
echo "assert GREEN: 在飞判据含 ops_headless AUD- 心跳腿——30~90min 长会话窗受保护且不互拦"
exit 0
