#!/usr/bin/env bash
# mig_dep_lib.sh —— 迁移 depends_on 声明解析共用件（工单#25800）：
# 声明形=迁移文件头注释区（首个 SQL 行前的连续空行/-- 注释行）内
#   `-- depends_on: <file.sql>[, <file2.sql>...]`（可多行累声明）
# 头区外同名字样=普通注释不解析——check_migrations ⑨ 对头区外出现
# 另判「声明错位」红，防作者以为声明了实则闸不见。
# 消费方：check_migrations.sh 断言⑨（形校验）/ land_queue.sh 合流
# 时序闸（依赖在合否）——两径共用本解析，判读不分叉。
# 用法：mig_dep_parse < file.sql → stdout=依赖文件名逐行（token 原样，
# 形校验归调用方）；mig_dep_stray < file.sql → stdout=头区外 depends_on
# 命中 token 逐行（无=空）。

mig_dep_awk() { # stdin=SQL 全文；stdout="<region>\t<dep>"（region=header|body）
	awk '
		function emit(v,   n,i,t) {
			gsub(/,/, " ", v)
			n = split(v, t, /[[:space:]]+/)
			for (i = 1; i <= n; i++)
				if (t[i] != "") print region "\t" t[i]
		}
		BEGIN { region = "header" }
		/^[[:space:]]*$/ { next }                 # 空行=头区仍在（SQL 未启）
		/^[[:space:]]*--/ {                       # 注释行
			if ($0 ~ /^[[:space:]]*--[[:space:]]*depends_on[[:space:]]*:/) {
				v = $0
				sub(/^[[:space:]]*--[[:space:]]*depends_on[[:space:]]*:[[:space:]]*/, "", v)
				emit(v)
			}
			next
		}
		{ region = "body"; next }                 # 首个 SQL 行=头区止，此行起 body
	'
}

mig_dep_parse() { # stdin=SQL → 头区 depends_on 目标逐行
	mig_dep_awk | awk -F'\t' '$1 == "header" { print $2 }'
}

mig_dep_stray() { # stdin=SQL → 头区外 depends_on 命中 token 逐行（供错位判）
	mig_dep_awk | awk -F'\t' '$1 == "body" { print $2 }'
}

# mig_dep_gate_check —— land_queue 合流时序闸核心（工单#25800）：
#   $1=支尖 ref $2=已合基线 ref（origin/master）；stdin=支新增迁移
#   文件清单（migrations/ 相对路径逐行）。须在仓内 git 语境调用。
#   stdout=缺依文件名逐行；rc0=全依已合/同支自带，rc1=有缺依。
# 判据：声明目标须在 $2 树内实存（已合）或在支新增集内（合流原子
# 带入）——两皆无=跨支未合依赖→rc1。depends_on 形合法性归
# check_migrations ⑨（支须先过绿戳），本闸只判「在合否」。
mig_dep_gate_check() { # $1=tip-ref $2=base-ref；stdin=新增迁移路径清单
	local tip="$1" base="$2" added selfdeps f dep rc=0
	added=$(cat)
	[ -n "$added" ] || return 0
	selfdeps=""
	while IFS= read -r f; do
		[ -n "$f" ] || continue
		selfdeps="${selfdeps}${f##*/}"$'\n'
	done <<< "$added"
	while IFS= read -r f; do
		[ -n "$f" ] || continue
		while IFS= read -r dep; do
			[ -n "$dep" ] || continue
			git cat-file -e "$base:migrations/$dep" 2>/dev/null && continue
			grep -qxF "$dep" <<< "$selfdeps" && continue
			printf '%s\n' "$dep"
			rc=1
		done < <(git show "$tip:$f" 2>/dev/null | mig_dep_parse)
	done <<< "$added"
	return $rc
}

# 仅供 source——直跑即用法报错
if [ "${BASH_SOURCE[0]}" = "$0" ]; then
	echo "mig_dep_lib.sh: 供 source 用（parse/stray/gate_check 三件）" >&2
	exit 2
fi
return 0
