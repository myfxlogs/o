#!/usr/bin/env bash
# check_immune.sh —— 免疫件写域闸（纪元3评估器物理隔离件）。
#
# 判据：merge-base(origin/master,HEAD)..HEAD 逐 commit 扫免疫面清单——
#   触件 commit 缺「免疫件:」trailer → FAIL（静默改裁判=RSI 命门——
#     DGM 关幻觉检查判例:改裁判是首要作弊面,必须显式宣告+永留痕）。
#   全触件有宣告 → 放行（trailer=commit 级永久审计轨;ops_event 追加
#     落行在 check_all 侧——本件只判不记,保持零依赖可独立跑）。
#
# 域界：裁判面∈单（闸件/hooks/部署闸/risk/guard/凭据产能/migrations/
# 清单自指）；钱径执行件∉——它们是被判对象,闸拦+账追已足。
# 边界：HEAD=origin/master 已推面不重复判（只判增量）;无 base 时
# 退化判 HEAD~1..HEAD（单机直提面同待遇）。
set -uo pipefail
cd "$(git rev-parse --show-toplevel)" || exit 1

MANIFEST=deploy/immune_paths.txt
if [ ! -f "$MANIFEST" ]; then
	echo "❌ [immune] $MANIFEST 缺席——免疫清单本体是裁判面,缺件 fail-closed"
	exit 1
fi
mapfile -t PATS < <(grep -vE '^\s*(#|$)' "$MANIFEST")
[ ${#PATS[@]} -gt 0 ] || { echo "❌ [immune] 免疫清单空集——判据失载 fail-closed"; exit 1; }

base=$(git merge-base HEAD origin/master 2>/dev/null || true) # tolerate: origin/master 缺席时退单机位
if [ -z "$base" ]; then
	base=$(git rev-parse --verify HEAD~1 2>/dev/null || true) # tolerate: 根 commit 无父,空=让位
fi
[ -z "$base" ] && { echo "OK: check_immune（无 base 可比——首 commit 面让位）"; exit 0; }

# 逐 commit 扫描——防「宣告在一 commit、偷渡在另一 commit」的分摊逃。
bad=0; touched_any=0
while IFS= read -r sha; do
	hit=0
	for p in "${PATS[@]}"; do
		# 禁 grep -q：pipefail 下 -q 命中即退→上游续写吃 SIGPIPE→管线判假败
		# （免疫触面漏检=偷渡窗——工单#11368 同期捕获同型病灶,check_import_direction.sh 详）
		if git diff-tree --no-commit-id --name-only -r "$sha" -- "$p" 2>/dev/null | grep -c . >/dev/null; then
			hit=1; touched_any=1
			# merge commit 的 diff-tree 相对双亲语义差异——merge 尖本体
			# 不携带新触件（触件在支 commit 里）,放行不重复判。
			if git show -s --format='%B' "$sha" | grep -c '^免疫件:' >/dev/null; then
				echo "⚠️ [immune] ${sha:0:9} 触面已宣告（免疫件: 在案）: $p"
			else
				echo "❌ [immune] ${sha:0:9} 触面未宣告: $p"
				bad=1
			fi
			break # 本 commit 命中一面即定（trailer 判是 commit 级非 path 级）
		fi
	done
	[ "$hit" -eq 1 ] && continue
done < <(git rev-list --no-merges "$base"..HEAD 2>/dev/null)

if [ "$bad" -eq 1 ]; then
	echo "❌ [immune] 免疫面触件未宣告——触件 commit 须携「免疫件:」trailer"
	exit 1
fi
if [ "$touched_any" -eq 1 ]; then
	echo "OK: check_immune（免疫面触件已宣告）"
else
	echo "OK: check_immune（免疫面零触）"
fi
exit 0
