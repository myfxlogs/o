#!/usr/bin/env bash
# check_import_direction.sh —— D-514：internal/* 包间 import 方向白名单门禁（自 nox 誊入）。
# O 骨架未落位期：白名单=空集（实拍边集随 T-0006 骨架冻结——新增跨包 import 须评审
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
internal/data/okxws -> internal/platform/notify
internal/ledger -> internal/platform/config
internal/ledger -> internal/platform/rejects
internal/ledger/holding -> internal/platform/rejects
internal/ledger/funding -> internal/platform/rejects
internal/ledger/retention -> internal/platform/rejects
internal/ledger/report -> internal/platform/config
internal/ledger/report -> internal/platform/cryptoutil
internal/ledger/report -> internal/platform/notify
internal/ledger/report -> internal/platform/rejects
internal/ledger -> internal/platform/notify
internal/earn -> internal/platform/notify
internal/guard -> internal/platform/notify
internal/trading/execution -> internal/platform/notify
internal/trading/mtexec -> internal/gen/mtapi/mt5
internal/trading/mtexec -> internal/trading/execution
internal/earn -> internal/platform/events
internal/guard -> internal/platform/events
internal/ledger -> internal/platform/events
internal/ledger/holding -> internal/platform/events
internal/portfolio/lifecycle -> internal/platform/events
internal/research/funding -> internal/platform/events
internal/review -> internal/platform/events
internal/data/okxpriv -> internal/platform/netmeter
internal/data/okxws -> internal/platform/netmeter
internal/platform/config -> internal/platform/cryptoutil
internal/data/collect -> internal/data/fact
internal/data/trx -> internal/data/fact
internal/data/trx -> internal/platform/safego
internal/data/trx -> internal/store/pgstore
internal/data/cmc -> internal/data/fact
internal/data/cmc -> internal/platform/safego
internal/data/cmc -> internal/store/pgstore
internal/data/fact -> internal/store/pgstore
internal/mt5 -> internal/platform/netmeter
internal/ledger/report -> internal/platform/roster
internal/ledger -> internal/platform/roster
internal/portfolio/lifecycle -> internal/platform/config
internal/risk -> internal/platform/config
internal/trading/execution -> internal/platform/config
internal/trading/execution -> internal/platform/rejects
internal/trading/execution -> internal/platform/roster
internal/trading/execution -> internal/trading/okxexec
internal/risk -> internal/platform/rejects
internal/risk -> internal/platform/roster
internal/guard -> internal/platform/config
internal/guard -> internal/platform/rejects
internal/guard -> internal/platform/roster
internal/guard -> internal/risk
internal/portfolio/lifecycle -> internal/platform/rejects
internal/portfolio/lifecycle -> internal/platform/roster
internal/review -> internal/platform/rejects
internal/review -> internal/platform/roster
internal/research/funding -> internal/platform/config
internal/research/funding -> internal/platform/rejects
internal/research/funding -> internal/platform/roster
internal/research/funding -> internal/platform/notify
internal/research/dcd -> internal/platform/config
internal/research/grammar -> internal/platform/config
internal/research/dcd -> internal/platform/events
internal/research/grammar -> internal/platform/events
internal/research/dcd -> internal/platform/notify
internal/research/dcd -> internal/platform/rejects
internal/research/grammar -> internal/platform/rejects
internal/research/dcd -> internal/research/armwatch
internal/research/armwatch -> internal/platform/notify
internal/research/researcher -> internal/platform/notify
internal/research/sigdist -> internal/platform/rejects
internal/research/sigdist -> internal/platform/roster
internal/research/sigdist -> internal/platform/config
internal/research/funding -> internal/research/armwatch
internal/research/gapscan -> internal/research/armwatch
internal/research/hypothesis -> internal/research/armwatch
internal/research/gapscan -> internal/platform/config
internal/research/gapscan -> internal/platform/rejects
internal/research/gapscan -> internal/platform/roster
internal/research/gapscan -> internal/platform/notify
internal/research/hypothesis -> internal/platform/config
internal/research/hypothesis -> internal/platform/events
internal/research/hypothesis -> internal/platform/rejects
internal/research/hypothesis -> internal/platform/roster
internal/research/hypothesis -> internal/platform/notify
internal/portfolio/trxvote -> internal/platform/safego
internal/portfolio/trxvote -> internal/store/pgstore
internal/store/pgstore -> internal/platform/pgguard
internal/platform/testutil -> internal/platform/pgguard
internal/data/okxws -> internal/platform/rejects
internal/data/okxws -> internal/platform/roster
internal/gen/mtapi/mt5/mt5connect -> internal/gen/mtapi/mt5
internal/facts -> internal/gen/mtapi/mt5
internal/facts -> internal/mt5
internal/facts -> internal/platform/rejects
internal/facts -> internal/platform/roster
internal/facts -> internal/trading/okxexec
internal/mt5 -> internal/gen/mtapi/mt5
internal/mt5 -> internal/platform/config
internal/mt5 -> internal/platform/cryptoutil
internal/mt5 -> internal/platform/rejects
internal/mt5 -> internal/platform/roster
internal/earn -> internal/ledger
internal/earn -> internal/platform/config
internal/earn -> internal/platform/rejects
internal/earn -> internal/platform/roster
internal/earn -> internal/trading/okxexec
internal/trading/okxexec -> internal/platform/signutil
internal/gen/dashboardv1/dashboardv1connect -> internal/gen/dashboardv1
EOF
)

RAW=$(go list -f '{{.ImportPath}}|{{join .Imports " "}}' ./internal/... 2>/dev/null)
ACTUAL=$(printf '%s\n' "$RAW" | awk -F'|' -v mod="$MOD" '{split($2,a," "); for(i in a) if(a[i] ~ "^"mod"/internal/") print $1" -> "a[i]}' | sed "s|^${MOD}/||; s| -> ${MOD}/| -> |" | sort -u)

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
