#!/usr/bin/env bash
# check_migrations.sh —— 迁移序号三断言硬闸（T-0109，canon
# migration-number-collision 根修——三犯实案机检化）：
#   ① migrate_table.go 登记 version 唯一（重复号=红）；
#   ② 每条登记 {N, "NNNN_*.sql"} 的文件名前缀号=登记号 N 且文件实存；
#   ③ migrations/*.sql 每个文件恰有一条登记行（孤儿文件=红）。
# 判的是号重复非序号连续——46/47 并存式改号合法（不拦历史）；孤儿/失配/
# 幽灵登记=后到者被 silent-skip 的撞号态即红。全域一致性检查（工作树面），
# 由 pre-commit 门禁链调用，可单独跑（与 check-lines 同构）。
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

REG="internal/store/pgstore/migrate_table.go"
fail=0

if [ ! -f "$REG" ]; then
	echo "check_migrations: 条件未备跳过（登记表缺席）"
	exit 0
fi

# ① 登记号唯一
dup_reg=$(grep -oE '^\s*\{[0-9]+,' "$REG" | grep -oE '[0-9]+' | sort -n | uniq -d || true)
if [ -n "$dup_reg" ]; then
	echo "FAIL $REG: 版本号重复登记（$(echo "$dup_reg" | paste -sd' ' -)）——同号后登记者被 silent-skip" >&2
	fail=1
fi

# ② 前缀号=登记号 + 文件实存
while IFS= read -r line; do
	[ -z "$line" ] && continue
	ver=$(sed -E 's/^\s*\{([0-9]+),.*/\1/' <<< "$line")
	name=$(sed -E 's/^\s*\{[0-9]+, *"([^"]+)".*/\1/' <<< "$line")
	prefix=$((10#${name%%_*}))
	if [ "$prefix" != "$ver" ]; then
		echo "FAIL $REG: 登记 $ver 与文件名前缀 $prefix 失配（$name）——序号语义分裂" >&2
		fail=1
	fi
	if [ ! -f "migrations/$name" ]; then
		echo "FAIL $REG: 登记 $ver 的 $name 文件不存在（幽灵登记——部署即炸）" >&2
		fail=1
	fi
done < <(grep -E '^\s*\{[0-9]+, *"[0-9]+_' "$REG")

# ③ 孤儿文件（migrations/*.sql 每文件恰一条登记）
while IFS= read -r f; do
	name=$(basename "$f")
	[ "$name" = "migrations.go" ] && continue
	grep -qF "\"$name\"" "$REG" || {
		echo "FAIL migrations/$name: 孤儿迁移文件（登记表无行——永不执行）" >&2
		fail=1
	}
done < <(find migrations -maxdepth 1 -name '*.sql' | sort)

# ④ T-0123：新 md_* 表（>0049）须声明 upsert 键（PRIMARY KEY/UNIQUE）或窗口
# 读面坐标+保留期（add_retention_policy）——追加式快照全废机检化，缺=拦。
while IFS= read -r f; do
	name=$(basename "$f")
	num=$((10#${name%%_*}))
	[ "$num" -le 49 ] && continue
	grep -qE 'CREATE TABLE[^(]*\bmd_' "$f" || continue
	grep -qE 'PRIMARY KEY|UNIQUE|add_retention_policy' "$f" || {
		echo "FAIL $name: 新 md_* 表未声明 upsert 键/窗口保留期（追加式快照全废——T-0123）" >&2
		fail=1
	}
done < <(find migrations -maxdepth 1 -name '*.sql' | sort)

if [ "$fail" -eq 1 ]; then
	echo "❌ check_migrations: 断言红——撞号/silent-skip/快照表病灶禁提交（canon migration-number-collision）" >&2
	exit 1
fi
echo "OK: check_migrations（四断言过——登记唯一/前缀实存/零孤儿/md_* 键域或保留期）"
