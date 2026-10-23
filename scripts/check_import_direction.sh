#!/usr/bin/env bash
# check_import_direction.sh —— D-514：internal/* 包间 import 方向白名单门禁（自 nox 誊入）。
# PHASE 骨架未落位期：白名单=空集（实拍边集随 T-0006 骨架冻结——新增跨包 import 须评审
# 方向合规并更新本表）；删边不拦。无 Go 面=条件未备直通（如实跳过非谎报）。
# 模块名动态取 go list -m，不钉字面量。
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

if [ ! -f go.mod ] || [ ! -d internal ]; then
	echo "check_import_direction: 条件未备跳过（无 Go 面——go.mod/internal 未落位）"
	exit 0
fi
MOD=$(go list -m)

# 白名单冻结表（格式：src -> dst，一行一边）。T-0006 骨架实拍冻结：codegen 固有边。
ALLOWED=$(cat <<'EOF'
internal/data/okxpriv -> internal/platform/config
internal/data/okxpriv -> internal/platform/cryptoutil
internal/data/okxpriv -> internal/platform/rejects
internal/data/okxpriv -> internal/platform/roster
internal/data/okxpriv -> internal/platform/signutil
internal/data/okxws -> internal/platform/config
internal/portfolio/lifecycle -> internal/platform/config
internal/risk -> internal/platform/config
internal/trading/execution -> internal/platform/config
internal/trading/execution -> internal/platform/rejects
internal/trading/execution -> internal/platform/roster
internal/trading/execution -> internal/trading/okxexec
internal/risk -> internal/platform/rejects
internal/risk -> internal/platform/roster
internal/portfolio/lifecycle -> internal/platform/rejects
internal/portfolio/lifecycle -> internal/platform/roster
internal/research/funding -> internal/platform/config
internal/research/funding -> internal/platform/rejects
internal/research/funding -> internal/platform/roster
internal/data/okxws -> internal/platform/rejects
internal/data/okxws -> internal/platform/roster
internal/trading/okxexec -> internal/platform/signutil
internal/gen/dashboardv1/dashboardv1connect -> internal/gen/dashboardv1
EOF
)

RAW=$(go list -f '{{.ImportPath}}|{{join .Imports " "}}' ./internal/... 2>/dev/null)
ACTUAL=$(printf '%s\n' "$RAW" | awk -F'|' -v mod="$MOD" '{split($2,a," "); for(i in a) if(a[i] ~ "^"mod"/internal/") print $1" -> "a[i]}' | sed "s|${MOD}/||g" | sort -u)

VIOL=""
while IFS= read -r e; do
	[ -z "$e" ] && continue
	if ! printf '%s\n' "$ALLOWED" | grep -qxF "$e"; then
		VIOL="$VIOL$e\n"
	fi
done <<< "$ACTUAL"

if [ -n "$VIOL" ]; then
	echo "FAIL: 新增跨包 import 方向违例（D-514 分层白名单）——确需引入先评审方向并更新 scripts/check_import_direction.sh 白名单：" >&2
	printf '%b' "$VIOL" >&2
	exit 1
fi
echo "OK: import direction gate（$(printf '%s\n' "$ACTUAL" | grep -c .)/$(printf '%s\n' "$ALLOWED" | grep -c .) 边）"

# 三条独立性红线（架构约束 §2，白名单之外加判，空壳期零报）：
#   风控独立于投资/交易：internal/risk 不 import trading|research|portfolio
#   账本独立于交易：internal/ledger 不 import trading
#   研究独立于执行：internal/research 不 import trading
RED=""
while IFS= read -r e; do
	src="${e%% -> *}"; dst="${e##* -> }"
	case "$src" in
		internal/risk*)
			case "$dst" in internal/trading*|internal/research*|internal/portfolio*) RED="$RED$e\n";; esac;;
		internal/ledger*)
			case "$dst" in internal/trading*) RED="$RED$e\n";; esac;;
		internal/research*)
			case "$dst" in internal/trading*) RED="$RED$e\n";; esac;;
	esac
done <<< "$ACTUAL"
if [ -n "$RED" ]; then
	echo "FAIL: 独立性红线违例（risk↛trading/research/portfolio；ledger/research↛trading）：" >&2
	printf '%b' "$RED" >&2
	exit 1
fi
