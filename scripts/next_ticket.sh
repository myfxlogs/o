#!/usr/bin/env bash
# next_ticket.sh —— 新票取号统一入口（T-0219）：git fetch 后扫
# origin/master:docs/tasks/ 全树（含 done/）最大号+1。双席并行「本地最大+1」猜测=撞号
# 根因（T-0217 实证）——远端为准；fetch 败降级按本地 origin/master 快照
# 并报 WARN（此时输出仅参考，push 时 --remote 闸仍会复核）。
# 用法：next_ticket.sh [seat]——seat=取号席位名（T-0373 领号台账行入参，
# 缺省 '席位未名'）。分配成功顺手写 ops_event '领号' 行（谁取了几号可溯
# ——台账审计件非硬依赖：DSN 缺席/写败静默退不阻断取号，|| true+# tolerate:
# 锚照容错必报律）。
# 本地并扫（T-0443 撞号实证）：批物料化一推多件时本地已 commit 未推号
# 在远端不可见——只扫 origin 会同批重号；取 max(本地树,远端树)+1 兜底。
#
# 票号原子发号（提案-票号原子发号-DB序列消撞号，fin_0151）：
# FIN_DSN 在案且 fin_ticket_seq 可达→序列径——先 setval 地板
# GREATEST(last_value,repo_max) 再 nextval（并发取号全局唯一，
# 未并枝票号亦不撞——扫号径只见 origin/master+HEAD 的盲区正是
# 原子化要消的病灶）；地板自校正种子漂移（迁移后扫号径新立票
# 把 repo_max 推过 last_value 时自动抬升，不烧旧号不回跳）。
# DSN 缺席/序列缺席/查询败→fail-open 回现行扫号径（与今天行为
# 一致不差；半成品窗并存期撞号仍由 merge 期重编让号兜底，对账桩
# check_ticket_numbers.sh --seq 盯 last_value≥max票号）。
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
git fetch origin master --quiet 2>/dev/null \
  || echo "WARN: git fetch 败——按本地 origin/master 快照取号（推送前 --remote 闸复核）" >&2
max=$( { git ls-tree -r --name-only origin/master -- docs/tasks/ 2>/dev/null; \
         git ls-tree -r --name-only HEAD -- docs/tasks/ 2>/dev/null; } \
  | grep -E '^docs/tasks/(done/)?T-[0-9]{4,}[[:alpha:]]?-' \
  | grep -oE 'T-[0-9]{4,}' | sed 's/^T-//' | sort -n | tail -1)

# env 探测径与 ops_esc.sh 同源（序列径+领号台账共用）。
FIN_DSN=
for ef in "${OPS_ESC_ENV_FILE:-}" "${DM_ENV_FILE:-}" "${REPORT_ENV_FILE:-}" \
          /opt/o-research/env /home/ubuntu/o-research/env; do
	[ -n "$ef" ] && [ -f "$ef" ] || continue
	FIN_DSN=$(grep '^FIN_PG_DSN=' "$ef" 2>/dev/null | cut -d= -f2- | tr -d '"')
	[ -n "$FIN_DSN" ] && break
done

TID=
if [ -n "$FIN_DSN" ]; then
	# 序列径：setval 地板+nextval 两发分离 autocommit——setval 败
	# （序列未迁移/权限缺）整段空→回退扫号；nextval 独得号全局唯一。
	# GREATEST(last_value,repo_max)+is_called=true → 下发号≥
	# repo_max+1 且恒>已发号（last_value 未发过的种子值让一号=无害空洞）。
	psql "$FIN_DSN" -qAt -c \
	  "SELECT setval('fin_ticket_seq', GREATEST((SELECT last_value FROM fin_ticket_seq), ${max:-0}), true)" \
	  >/dev/null 2>&1 || true # tolerate: 序列径前置败——地板未立照走 nextval（种子即底）
	seqn=$(psql "$FIN_DSN" -qAt -c "SELECT nextval('fin_ticket_seq')" 2>/dev/null) || seqn=
	if [[ "$seqn" =~ ^[0-9]+$ ]]; then
		TID=$(printf 'T-%04d\n' "$seqn")
	fi
fi
[ -n "$TID" ] || TID=$(printf 'T-%04d\n' "$((10#${max:-0} + 1))")

# T-0373 领号台账行：seat 入参 $1。
SEAT=${1:-席位未名}
if [ -n "$FIN_DSN" ]; then
	SQC=$(printf '%s' "$SEAT" | sed "s/'/''/g")
	psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note) VALUES('$SQC','领号','$TID','取号径 next_ticket.sh')" >/dev/null 2>&1 || true # tolerate: 台账审计件非硬依赖——DSN 缺席/写败静默退不阻断取号
fi
printf '%s\n' "$TID"
