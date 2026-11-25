#!/usr/bin/env bash
# check_surface_lifecycle.sh —— 积面终末消费者出生闸（工单O-39400②B——
# 「表生而无终端消费者」同型病根修：fin_surface_gap 权威面三写家零消费
# 实证在案）：本分支新增/改动 migrations/*.sql 内 CREATE TABLE，须同
# 文件 INSERT INTO fin_surface_lifecycle 登记（consumer_ref 谁消费+
# terminal_path 怎么到终态），或表名入 docs/surface_lifecycle_registry.
# txt 豁免册（非积累终态面——联接/日志/中间表/登记册自身）。
# 存量面不追旧账（判据=diff 面非全量面——同 queue_birth 存量豁免律）。
# 桩测面：SURFACE_LIFECYCLE_MIG_DIR/REG/FILES 可指夹具
# （check_surface_lifecycle_test.sh 喂未登记/已登记/豁免三案）。
set -uo pipefail
ROOT=${SURFACE_LIFECYCLE_ROOT:-$(git rev-parse --show-toplevel 2>/dev/null)}
[ -n "$ROOT" ] && cd "$ROOT" || { echo "❌ [surface-lifecycle] 仓根不可解——fail-closed"; exit 1; }
MIGDIR="${SURFACE_LIFECYCLE_MIG_DIR:-$ROOT/migrations}"
REG="${SURFACE_LIFECYCLE_REG:-$ROOT/docs/surface_lifecycle_registry.txt}"
[ -f "$REG" ] || { echo "❌ [surface-lifecycle] $REG 缺席——豁免册未立" >&2; exit 1; }

# 迁移文件集：FILES 环境变量（夹具空白分隔）> diff-filter=AM vs
# merge-base origin/master（本分支面）> 兜底全扫（无 origin 史时
# 全量面逐表过——存量表须已在册否则闸红=宁严勿漏）。
if [ -n "${SURFACE_LIFECYCLE_FILES:-}" ]; then
	mapfile -t MIGS < <(printf '%s\n' $SURFACE_LIFECYCLE_FILES)
elif [ -n "${SURFACE_LIFECYCLE_MIG_DIR:-}" ]; then
	mapfile -t MIGS < <(ls "$MIGDIR"/*.sql 2>/dev/null | sort)
else
	BASE=$(git merge-base origin/master HEAD 2>/dev/null || true) # tolerate: 无 origin 史=空→全扫兜底分支
	if [ -n "$BASE" ]; then
		mapfile -t MIGS < <(git diff --name-only --diff-filter=AM "$BASE"...HEAD -- 'migrations/*.sql' 2>/dev/null | sort)
	else
		mapfile -t MIGS < <(ls "$MIGDIR"/*.sql 2>/dev/null | sort)
	fi
fi
[ ${#MIGS[@]} -gt 0 ] || { echo "OK: check_surface_lifecycle（本分支零迁移面）"; exit 0; }

bad=""
for f in "${MIGS[@]}"; do
	[ -f "$f" ] || f="$MIGDIR/$(basename "$f")"
	[ -f "$f" ] || continue # tolerate: diff 列名在 worktree 外=已合他支——跳
	# 同文件登记判据（spec 钦定「同迁移」——跨文件登记不算数）；
	# 逐表核 VALUES 首列名——登一表不免同文件他表。
	regd=$(sed -n '/INSERT INTO fin_surface_lifecycle/,/;/p' "$f" 2>/dev/null \
		| grep -oE "\('[a-zA-Z_][a-zA-Z0-9_]*'" | tr -d "('" | sort -u)
	while IFS= read -r t; do
		[ -n "$t" ] || continue
		# 豁免册命中=非积累终态面——放行
		if grep -vE '^\s*(#|$)' "$REG" | grep -qxF "$t"; then continue; fi
		# fin_surface_lifecycle 自身=登记册非积面——恒豁免
		[ "$t" = "fin_surface_lifecycle" ] && continue
		if ! printf '%s\n' "$regd" | grep -qxF "$t"; then
			bad="$bad$f: CREATE TABLE $t 无 fin_surface_lifecycle 同迁移登记且不在豁免册"$'\n'
		fi
	done < <(grep -oE 'CREATE[[:space:]]+TABLE([[:space:]]+IF[[:space:]]+NOT[[:space:]]+EXISTS)?[[:space:]]+[a-zA-Z_][a-zA-Z0-9_]*' "$f" \
		| awk '{print $NF}' | sort -u)
done

if [ -n "$bad" ]; then
	printf '%s' "$bad" | while IFS= read -r b; do
		[ -n "$b" ] && echo "❌ [surface-lifecycle] $b"
	done
	echo "❌ [surface-lifecycle] 积面出生闸红——新积累终态表须同迁移登记 fin_surface_lifecycle(consumer_ref+terminal_path) 或入 docs/surface_lifecycle_registry.txt 豁免册（须注事由）" >&2
	exit 1
fi
echo "OK: check_surface_lifecycle（迁移面 ${#MIGS[@]} 件——积面登记/豁免全过）"
