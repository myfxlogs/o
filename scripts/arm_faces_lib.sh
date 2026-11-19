#!/usr/bin/env bash
# arm_faces_lib.sh —— OCI 臂谱系证空判公共件（工单#25481：谱系正本=
# deploy/deploy-faces.tsv 三词表 nondeploy/image_only/<arm名>——deploy_oci
# 谱系自检与 deploy_watch 臂滞后探同源共用，抽库防双写漂）。
# 判式：件在 nondeploy/image_only→对任臂证外；件在他臂谱系行→对本臂证外；
# 余者（含 tsv 缺/行缺/类缺）=不可证空（滥换方向不省错方向）。
# 用法：source scripts/arm_faces_lib.sh 后调 arm_stale <arm> <base>（rc=0
# =有件须换；diff 败/基线缺=不可证空保守 rc0）。
FACES_TSV="${FACES_TSV:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../deploy/deploy-faces.tsv}"
faces_rows() { grep -v '^#' "$FACES_TSV" 2>/dev/null || true; } # tolerate: 件缺=否列空→全面不可判全臂照换（滥换方向不省错方向）
in_class() { # in_class <path> <class> → rc=0 路径属该类行
	local p=$1 want=$2 c g
	while IFS=$'\t' read -r c g; do
		{ [ "$c" = "$want" ] && [ -n "$g" ]; } || continue
		case "$p" in $g) return 0;; esac
	done < <(faces_rows)
	return 1
}
in_any_arm() { # in_any_arm <path> → rc=0 属任一 OCI 谱系行
	local p=$1 c g
	while IFS=$'\t' read -r c g; do
		case "$c" in collecto|projector|research|eventbus|sweeper) ;; *) continue;; esac
		case "$p" in $g) return 0;; esac
	done < <(faces_rows)
	return 1
}
arm_provably_clean() { # arm_provably_clean <arm> <path> → rc=0 证该件不改本臂
	local a=$1 p=$2
	in_class "$p" nondeploy && return 0
	in_class "$p" image_only && return 0
	in_any_arm "$p" && ! in_class "$p" "$a" && return 0
	return 1
}
arm_stale() { # arm_stale <arm> <base> → rc=0 有件须换（diff 败=不可证空保守换）
	local a=$1 base=$2 p files
	files=$(git diff --name-only "$base..HEAD" 2>/dev/null) || return 0
	while IFS= read -r p; do
		[ -n "$p" ] || continue
		p=${p##* -> } # rename 形取新名
		arm_provably_clean "$a" "$p" || return 0
	done <<< "$files"
	return 1
}
