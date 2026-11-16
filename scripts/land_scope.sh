#!/usr/bin/env bash
# land_scope.sh —— 完工支落地分流闸（施工席落地道件，业主裁面 2026-11-16
# 「普通改动自合自运；动钱的只交支复审」钦定）。
#
# 用法：land_scope.sh <branch|sha|origin/br> [base]
#   输出逐件「FREE/REVIEW <path>」+尾行判词；rc0=全件自由面（可自合自运），
#   rc2=含钱径面（只交支候审），rc1=用法/取数错。
#
# 判径=白名单 fail-closed：未列径一律 REVIEW（宁枉勿纵——误放=钱径无证入产）。
# 自由面只纳纯文档/知识件：
#   docs/**.md|txt  skills/**.md  .devin/**.md|txt  tools/**.md  根单层 *.md(剔 AGENTS.md)
# 钱径/裁判/调度/部署/脚本/码件全 REVIEW——一切 *.go/*.sh/*.sql/proto、
#   scripts/ deploy/ migrations/ internal/ cmd/ hooks/ go.mod go.sum AGENTS.md、
#   嵌套层 .md（runbook/契约件含内——席卡自改面不自开）。
# 本件自身在免疫册（immune_paths.txt 显式行）——改本闸=触免疫面。
set -u
br="${1:?用法: land_scope.sh <branch|sha|origin/br> [base]}"
base="${2:-origin/master}"
mb=$(git merge-base "$base" "$br" 2>/dev/null) || { echo "land_scope: merge-base 取不出 $base..$br" >&2; exit 1; }
tip=$(git rev-parse "$br" 2>/dev/null) || { echo "land_scope: 支名解不出 $br" >&2; exit 1; }

is_free() { # $1=path → rc0 自由面
	case "$1" in
		docs/*.md|docs/*.txt|skills/*.md|.devin/*.md|.devin/*.txt|tools/*.md) return 0 ;;
	esac
	case "$1" in
		*/*) return 1 ;;        # 非根单层=嵌套件全 REVIEW（runbook/契约件含内）
		AGENTS.md) return 1 ;;  # 契约本体=裁判面
		*.md) return 0 ;;
	esac
	return 1
}

rc=0
while IFS= read -r f; do
	if is_free "$f"; then
		echo "FREE   $f"
	else
		echo "REVIEW $f"; rc=2
	fi
done < <(git diff --name-only "$mb" "$tip")
if [ "$rc" -eq 0 ]; then
	echo "FREE 全件——可自合自运"
else
	echo "REVIEW 含钱径/裁判面——只交支候审（merge 留唤知/外席）"
fi
exit "$rc"
