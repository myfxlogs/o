#!/usr/bin/env bash
# check_rejects_keys.sh —— rejects 计数门键对账硬闸（T-0148——okxws
# depth5_no_ctval 漏注册实案机检化：每次 Inc 落 WARN 洪泛+计数丢失，
# 日志量 ~96% 被单键占坑掩真信号）：
#   全仓非测试 Go 面静态对账——各包 .Inc("字面量") 键 ⊆ 同包 rejects.New
#   注册键集；漏注册=红。
#   非字面量 Inc（动态拒因）静态不可判——列 INFO 面留人审（risk deny
#   reason 域先例：deny 实参取值域全在注册表，T-0148 普查亲证）。
# 全域一致性检查（工作树面），由 pre-commit 门禁链调用，可单独跑。
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

if [ ! -f go.mod ] || [ ! -d internal ]; then
	echo "check_rejects_keys: 条件未备跳过（无 Go 面）"
	exit 0
fi

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
REG="$TMP/reg"
INC="$TMP/inc"
DYN="$TMP/dyn"

while IFS= read -r f; do
	d=$(dirname "$f")
	# 注册键提取（跨行调用折叠后抓 rejects.New("face","k1",...) 全字面量；
	# 首串=面名剔除）。
	src=$(tr '\n' ' ' <"$f")
	printf '%s\n' "$src" | grep -oP 'rejects\.New\(\s*"[^"]+"(?:\s*,\s*"[^"]+")*' \
		| grep -oP '"[^"]+"' | tr -d '"' | tail -n +2 \
		| while IFS= read -r k; do printf 'R\t%s\t%s\n' "$d" "$k"; done >>"$REG" || true
	# 字面量 Inc 键落账。
	grep -noP '\.Inc\(\s*"[^"]+"\s*\)' "$f" | while IFS=: read -r ln m; do
		k=$(printf '%s' "$m" | grep -oP '"[^"]+"' | tr -d '"')
		printf 'I\t%s\t%s\n' "$d" "$k"
	done >>"$INC" || true
	# 动态 Inc 留人审面。
	grep -n '\.Inc(' "$f" | grep -v '\.Inc("' \
		| while IFS=: read -r ln rest; do printf 'D\t%s:%s\t%s\n' "$f" "$ln" "$(printf '%s' "$rest" | sed 's/^[[:space:]]*//')"; done >>"$DYN" || true
done < <(find internal cmd -name '*.go' ! -name '*_test.go' 2>/dev/null | sort)

LEAKS=$(awk -F'\t' 'NR==FNR{if($1=="R")reg[$2"|"$3]=1;next}
	$1=="I" && !(($2"|"$3) in reg){print "  " $2 " 门键 \"" $3 "\" Inc 未注册"}' "$REG" "$INC")
if [ -n "$LEAKS" ]; then
	echo "FAIL: rejects 门键对账漏注册（Inc 键不在同包 rejects.New 注册集——运行期 WARN 洪泛+计数丢失，okxws depth5_no_ctval 实案）：" >&2
	echo "$LEAKS" >&2
	exit 1
fi
if [ -s "$DYN" ]; then
	echo "INFO: 动态拒因 Inc（静态不可判——人工对账取值域 ⊆ 注册表）:"
	cat "$DYN"
fi
echo "OK: check_rejects_keys（全仓 Inc 键 ⊆ rejects 注册表，零漏）"
