#!/usr/bin/env bash
# foundby.sh —— 修复提交人机发现比度量（审计臂产出账：升件发现的坑
# 修复占比=臂价值机读证）。不写库,stdout 一行汇总+逐条明细。
# 口径：近 N 天（$1,默认 7）commit 携「坑处置:」面=修复提交；
#   subject+body 引「升件#<id>」=机发现（升件环产物——唤知/审计同面）,
#   否则=人发现。
# 仓定位=本件所在仓（会话 cwd 在 sessions/ 目录外——禁依赖 cwd）。
# 逐 sha 取全文体（%b 多行体按行 read 会把体行当新记录虚增计数——
# 首版实坑,改 sha 列表+单取）。
set -u
N=${1:-7}
REPO=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
T=0; M=0; H=0; LINES=""
while IFS= read -r sha; do
	[ -n "$sha" ] || continue
	T=$((T+1))
	subj=$(git -C "$REPO" show -s --format=%s "$sha")
	if git -C "$REPO" show -s --format=%B "$sha" | grep -qE '升件#[0-9]+'; then
		M=$((M+1)); LINES="$LINES
机	${sha:0:9}	$subj"
	else
		H=$((H+1)); LINES="$LINES
人	${sha:0:9}	$subj"
	fi
done < <(git -C "$REPO" log --since="$N days ago" --grep='坑处置:' --format='%H')
printf '度量: 窗=%s天 修复提交=%d 机发现=%d 人发现=%d 机占比=%s\n' \
	"$N" "$T" "$M" "$H" "$([ "$T" -gt 0 ] && echo "$M/$T" || echo "0/0")"
[ -n "$LINES" ] && printf '%s\n' "$LINES"
exit 0
