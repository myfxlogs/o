#!/usr/bin/env bash
# next_ticket.sh —— 新票取号统一入口（T-0219）：git fetch 后扫
# origin/master:docs/tasks/ 全树（含 done/）最大号+1。双席并行「本地最大+1」猜测=撞号
# 根因（T-0217 实证）——远端为准；fetch 败降级按本地 origin/master 快照
# 并报 WARN（此时输出仅参考，push 时 --remote 闸仍会复核）。
# 用法：next_ticket.sh [seat]——seat=取号席位名（T-0373 领号台账行入参，
# 缺省 '席位未名'）。分配成功顺手写 ops_event '领号' 行（谁取了几号可溯
# ——台账审计件非硬依赖：DSN 缺席/写败静默退不阻断取号，|| true+# tolerate:
# 锚照容错必报律）。
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
git fetch origin master --quiet 2>/dev/null \
  || echo "WARN: git fetch 败——按本地 origin/master 快照取号（推送前 --remote 闸复核）" >&2
max=$(git ls-tree -r --name-only origin/master -- docs/tasks/ 2>/dev/null \
  | grep -E '^docs/tasks/(done/)?T-[0-9]{4,}[[:alpha:]]?-' \
  | grep -oE 'T-[0-9]{4,}' | sed 's/^T-//' | sort -n | tail -1)
TID=$(printf 'T-%04d\n' "$((10#${max:-0} + 1))")

# T-0373 领号台账行：seat 入参 $1；env 探测径与 ops_esc.sh 同源。
SEAT=${1:-席位未名}
FIN_DSN=
for ef in "${OPS_ESC_ENV_FILE:-}" "${DM_ENV_FILE:-}" "${REPORT_ENV_FILE:-}" \
          /opt/o-research/env /home/ubuntu/o-research/env; do
	[ -n "$ef" ] && [ -f "$ef" ] || continue
	FIN_DSN=$(grep '^FIN_PG_DSN=' "$ef" 2>/dev/null | cut -d= -f2- | tr -d '"')
	[ -n "$FIN_DSN" ] && break
done
if [ -n "$FIN_DSN" ]; then
	SQC=$(printf '%s' "$SEAT" | sed "s/'/''/g")
	psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note) VALUES('$SQC','领号','$TID','取号径 next_ticket.sh——origin/master max+1')" >/dev/null 2>&1 || true # tolerate: 台账审计件非硬依赖——DSN 缺席/写败静默退不阻断取号
fi
printf '%s\n' "$TID"
