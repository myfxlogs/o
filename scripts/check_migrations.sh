#!/usr/bin/env bash
# check_migrations.sh —— 迁移七断言硬闸（T-0109 三断言为基，T-0167 分链制扩七）：
#   ① migrate_table.go 登记 version 链内唯一（同链重复号=红——同号后登记者被 silent-skip）；
#   ② 每条登记 {N, "md|fin_NNNN_*.sql"} 的链内序号=登记号 N 且文件实存；
#   ③ migrations/*.sql 每个文件恰有一条登记行（孤儿文件=红）；
#   ④ T-0123：新 md_* 表（链内号>49）须声明 upsert 键或保留期（追加式快照全废机检化）；
#   ⑤ T-0167：两链登记各自单调（登记序=链内版本升序——分链后链即序，乱序登记=红）；
#   ⑥ T-0167/D-0032 §三-5：八桥表 DDL 归一化同形——md 链正表 vs fin 链镜像：
#      列定义全项（名/类型/NOT NULL/DEFAULT）+CHECK/UNIQUE/FK 约束+索引集
#      +hypertable/保留期，归一化（去注释/去 IF NOT EXISTS/折空白）后语句集相等
#      （回滚径「SG 镜像表即正表（DDL 相同）」前提机检钉死）；
#   ⑦ T-0167/D-0030 §六：域表白名单（migrations/domain.tsv 表名→md|fin|bridge|
#      bridge-reverse）——链内 DDL/DML 触表出现非本域且非桥（bridge/
#      bridge-reverse）即红（串链）；触表缺册=红。
# 判的是链内号重复非连续——链内单调由⑤另钉。全域一致性检查（工作树面），
# 由 pre-commit 门禁链调用，可单独跑（与 check-lines 同构）。
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

REG="internal/store/pgstore/migrate_table.go"
TSV="migrations/domain.tsv"
fail=0

if [ ! -f "$REG" ]; then
	echo "check_migrations: 条件未备跳过（登记表缺席）"
	exit 0
fi

# ①②⑤ 登记表面（链内唯一/单调+序号失配+幽灵登记）
declare -A SEEN=()
declare -A LAST=()
while IFS= read -r line; do
	[ -z "$line" ] && continue
	ver=$(sed -E 's/^\s*\{([0-9]+),.*/\1/' <<< "$line")
	name=$(sed -E 's/^\s*\{[0-9]+, *"([^"]+)".*/\1/' <<< "$line")
	chain=${name%%_*}
	rest=${name#*_}
	num=$((10#${rest%%_*}))
	if [ "$chain" != "md" ] && [ "$chain" != "fin" ]; then
		echo "FAIL $REG: 登记 $name 前缀非 md|fin（分链制命名 md_NNNN_*/fin_NNNN_*）" >&2
		fail=1
		continue
	fi
	# ① 链内唯一
	key="$chain/$ver"
	if [ -n "${SEEN[$key]:-}" ]; then
		echo "FAIL $REG: $chain 链版本号 $ver 重复登记（${SEEN[$key]} 与 $name）——同号后登记者被 silent-skip" >&2
		fail=1
	fi
	SEEN[$key]=$name
	# ⑤ 链内单调（登记序=版本升序）
	if [ -n "${LAST[$chain]:-}" ] && [ "$ver" -le "${LAST[$chain]}" ]; then
		echo "FAIL $REG: $chain 链登记非单调（$name 乱序）——分链后链即序，乱序=红" >&2
		fail=1
	fi
	LAST[$chain]=$ver
	# ② 链内序号=登记号 + 文件实存
	if [ "$num" != "$ver" ]; then
		echo "FAIL $REG: 登记 $ver 与文件链内序号 $num 失配（$name）——序号语义分裂" >&2
		fail=1
	fi
	if [ ! -f "migrations/$name" ]; then
		echo "FAIL $REG: 登记 $ver 的 $name 文件不存在（幽灵登记——部署即炸）" >&2
		fail=1
	fi
done < <(grep -E '^\s*\{[0-9]+, *"(md|fin)_[0-9]+_' "$REG")

# ③ 孤儿文件（migrations/*.sql 每文件恰一条登记）
while IFS= read -r f; do
	name=$(basename "$f")
	[ "$name" = "migrations.go" ] && continue
	[ "$name" = "domain.tsv" ] && continue
	grep -qF "\"$name\"" "$REG" || {
		echo "FAIL migrations/$name: 孤儿迁移文件（登记表无行——永不执行）" >&2
		fail=1
	}
done < <(find migrations -maxdepth 1 \( -name '*.sql' -o -name 'domain.tsv' \) | sort)

# ④ 新 md_* 表 upsert 键/保留期（链内号>49）
while IFS= read -r f; do
	name=$(basename "$f")
	num=$((10#$(basename "$name" .sql | cut -d_ -f2)))
	[ "${name%%_*}" = "md" ] && [ "$num" -gt 1000 ] && num=$((num-1000)) # md 链版本+1000 错位——判阈用历史号
	[ "$num" -le 49 ] && continue
	grep -qE 'CREATE TABLE[^(]*\bmd_' "$f" || continue
	grep -qE 'PRIMARY KEY|UNIQUE|add_retention_policy' "$f" || {
		echo "FAIL $name: 新 md_* 表未声明 upsert 键/窗口保留期（追加式快照全废——T-0123）" >&2
		fail=1
	}
done < <(find migrations -maxdepth 1 -name '*.sql' | sort)

# 触表提取：全文件 → 行式 chain/table/kind/normalized_stmt（⑥⑦ 共用）
EXTRACT=$(mktemp)
trap 'rm -f "$EXTRACT" "$EXTRACT".* 2>/dev/null || true' EXIT
for f in migrations/*.sql; do
	chain=$(basename "$f"); chain=${chain%%_*}
	awk -v chain="$chain" '
		BEGIN { s = "" }
		{
			line = $0
			sub(/--.*/, "", line)           # 去行注释
			gsub(/[\t\r]/, " ", line)
			s = s " " line                  # 语句流拼接
		}
		END {
			gsub(/;/, ";\n", s)             # 分号切语句
			nstmt = split(s, st, "\n")
			for (i = 1; i <= nstmt; i++) {
				t = st[i]
				gsub(/[ ]+/, " ", t); sub(/^ /, "", t); sub(/ $/, "", t)
				if (t == "") continue
				u = t
				gsub(/IF NOT EXISTS /, "", u)
				gsub(/ OR REPLACE /, " ", u)
				tbl = ""; kind = ""
				if (u ~ /^CREATE TABLE /) {
					tbl = u; sub(/^CREATE TABLE /, "", tbl); sub(/[ (].*/, "", tbl); kind = "create"
				} else if (u ~ /^ALTER TABLE /) {
					tbl = u; sub(/^ALTER TABLE /, "", tbl); sub(/ .*/, "", tbl); kind = "alter"
				} else if (u ~ /^CREATE (UNIQUE )?INDEX /) {
					if (match(u, / ON [a-zA-Z_][a-zA-Z0-9_]*/)) {
						tbl = substr(u, RSTART + 4, RLENGTH - 4); kind = "index"
					}
				} else if (u ~ /create_hypertable\(/) {
					if (match(u, /create_hypertable\( *.[a-zA-Z_][a-zA-Z0-9_]*/)) {
						tbl = substr(u, RSTART, RLENGTH)
						sub(/create_hypertable\( *./, "", tbl); kind = "hypertable"
					}
				} else if (u ~ /add_retention_policy\(/) {
					if (match(u, /add_retention_policy\( *.[a-zA-Z_][a-zA-Z0-9_]*/)) {
						tbl = substr(u, RSTART, RLENGTH)
						sub(/add_retention_policy\( *./, "", tbl); kind = "retention"
					}
				} else if (u ~ /^DROP TABLE /) {
					tbl = u; sub(/^DROP TABLE (IF EXISTS )?/, "", tbl); sub(/ .*/, "", tbl); kind = "drop"
				} else if (u ~ /^INSERT INTO /) {
					tbl = u; sub(/^INSERT INTO /, "", tbl); sub(/ .*/, "", tbl); kind = "insert"
				} else if (u ~ / TRIGGER .* ON /) {
					if (match(u, / ON [a-zA-Z_][a-zA-Z0-9_]*/)) {
						tbl = substr(u, RSTART + 4, RLENGTH - 4); kind = "trigger"
					}
				} else if (u ~ /^UPDATE /) {
					tbl = u; sub(/^UPDATE /, "", tbl); sub(/ .*/, "", tbl)
					if (tbl == "SET" || tbl == "ON" || tbl == "ONLY") tbl = ""
					else kind = "update"
				} else if (u ~ /^DELETE FROM /) {
					tbl = u; sub(/^DELETE FROM /, "", tbl); sub(/ .*/, "", tbl); kind = "delete"
				}
				if (tbl != "" && kind != "" && tbl ~ /^[a-z_][a-z0-9_]*$/)
					printf "%s\t%s\t%s\t%s\t%s\n", chain, tbl, kind, u, FILENAME
			}
		}
	' "$f"
done > "$EXTRACT"

# ⑦ 域表白名单：触表缺册=红；链内触表非本域且非 bridge=红
declare -A TD=()
if [ -f "$TSV" ]; then
	while IFS=$'\t' read -r t d; do
		[ -z "$t" ] && continue
		case "$t" in \#*) continue;; esac
		TD[$t]=$d
	done < "$TSV"
else
	echo "FAIL $TSV: 域表白名单缺席（分链制必备）" >&2
	fail=1
fi
while IFS=$'\t' read -r ch tbl kind stmt src; do
	d=${TD[$tbl]:-}
	if [ -z "$d" ]; then
		echo "FAIL $src: 触表 $tbl 缺 domain.tsv 册（新表先入册再落迁移）" >&2
		fail=1
	elif [ "$d" != "$ch" ] && [ "$d" != "bridge" ] && [ "$d" != "bridge-reverse" ]; then
		echo "FAIL $src: $ch 链触表 $tbl 域=$d（非本域非 bridge——串链）" >&2
		fail=1
	fi
done < "$EXTRACT"

# ⑥ 八桥表+反向桥表 DDL 归一化同形：链间形状语句集（create/alter/index/
# hypertable/retention）相等（T-0172：bridge-reverse 正本在 fin 链、md 链
# 镜像——同形前提回滚径两向共用）。
BRIDGE="md_ticks md_mark_index md_signal md_dcd_product md_funding md_macro_event fin_proposal fin_opp_eval fin_strategy fin_venue_untradeable"
	for bt in $BRIDGE; do
		if [ "${TD[$bt]:-}" != "bridge" ] && [ "${TD[$bt]:-}" != "bridge-reverse" ]; then
			echo "FAIL $TSV: 桥表 $bt 未以 bridge|bridge-reverse 入册（同形断言清单对齐）" >&2
			fail=1
			continue
		fi
	for side in md fin; do
		grep -P "^${side}\t${bt}\t(create|alter|index|hypertable|retention)\t" "$EXTRACT" \
			| cut -f4 | sort -u > "$EXTRACT.$side" || true
	done
	if ! diff -u "$EXTRACT.md" "$EXTRACT.fin" > "$EXTRACT.diff"; then
		echo "FAIL 桥表 $bt DDL 归一化漂移（md 链正表 vs fin 链镜像语句集不等——回滚径「DDL 相同」前提破）：" >&2
		sed 's/^/    /' "$EXTRACT.diff" >&2
		fail=1
	fi
done

if [ "$fail" -eq 1 ]; then
	echo "❌ check_migrations: 断言红——撞号/串链/桥表漂移/白名单缺册禁提交（canon migration-number-collision + D-0030 §六）" >&2
	exit 1
fi
echo "OK: check_migrations（七断言过——链内唯一/序号实存/零孤儿/md_* 键域/两链单调/八桥 DDL 同形/域白名单）"
