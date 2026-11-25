#!/usr/bin/env bash
# check_migrations.sh —— 迁移十断言硬闸（T-0109 三断言为基，T-0167 分链制扩七，裸GRANT 闸为⑧，depends_on 声明合法性为⑨，破坏性 DDL 存量违例 lint 为⑩）：
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
#      bridge-reverse）即红（串链）；触表缺册=红；
#   ⑧ 裸 GRANT 闸（0071/0098/0161/0191 四案实证）：链外角色 GRANT 必须
#      在 DO $$ IF EXISTS(pg_roles) 块内——顶格 GRANT 只许链内自建角色
#      或 PUBLIC，否则 fresh 测试库无此角色即 SQLSTATE 42704 炸链。
#   ⑨ depends_on 声明合法性（工单#25800——跨支迁移序号倒挂治本）：
#      文件头注释区 `-- depends_on: <file.sql>`（mig_dep_lib 解析）——
#      目标须=登记在册同链迁移且链内号<声明者（Migrate 应用序=号序，
#      依赖必先施；号序倒挂=静默语义反转）；未登记/跨链/倒挂/值非法/
#      头区外错位声明=红。无声明=现状不禁（渐进启用——既有迁移零
#      声明零影响）。合流时序闸在 land_queue.sh（所依未合→不合）。
#   ⑩ 破坏性 DDL 存量违例 lint（工单#25799——测试 migrate 恒空表，
#      ADD CONSTRAINT CHECK/FOREIGN KEY、SET NOT NULL、新 UNIQUE/
#      唯一索引对生产存量行违例部署期才炸）：水位 fin≤215/md≤1100
#      （本闸落地时点存量全册）豁免不回杀；水位以上新迁移——
#      a) ADD CONSTRAINT <名> CHECK|FOREIGN KEY 须 NOT VALID+同文件
#         VALIDATE CONSTRAINT <名>（NOT VALID 不验=静默豁免洞）；
#      b) UNIQUE/PK/EXCLUDE/SET NOT NULL/ADD COLUMN NOT NULL 无
#         DEFAULT/CREATE UNIQUE INDEX/裸 ADD CHECK|FK 不可 NOT VALID
#         型须 `-- mig-rehearsal-ok: <理由>` 豁免行。
# 判的是链内号重复非连续——链内单调由⑤另钉。全域一致性检查（工作树面），
# 由 pre-commit 门禁链调用，可单独跑（与 check-lines 同构）。
set -euo pipefail
MIGDEP_LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/mig_dep_lib.sh"
if [ -f "$MIGDEP_LIB" ]; then . "$MIGDEP_LIB"; else mig_dep_parse() { :; }; mig_dep_stray() { :; }; fi
cd "$(git rev-parse --show-toplevel)"

# 登记表分件（migrate_table*.go——D-588 棘轮拆链：fin 链本件/md 链 _md 件）。
# ls 字典序 migrate_table.go<migrate_table_md.go 天然 fin 前 md 后，
# 链内单调判据序守恒；grep -h 抑文件名前缀保行形。
REGS=$(ls internal/store/pgstore/migrate_table*.go 2>/dev/null | tr '\n' ' ')
TSV="migrations/domain.tsv"
fail=0

if [ -z "$REGS" ]; then
	echo "check_migrations: 条件未备跳过（登记表缺席）"
	exit 0
fi

# ①②⑤ 登记表面（链内唯一/单调+序号失配+幽灵登记）
declare -A SEEN=()
declare -A LAST=()
declare -A VEROF=()   # 文件名→登记号（⑨ depends_on 判据）
while IFS= read -r line; do
	[ -z "$line" ] && continue
	ver=$(sed -E 's/^\s*\{([0-9]+),.*/\1/' <<< "$line")
	name=$(sed -E 's/^\s*\{[0-9]+, *"([^"]+)".*/\1/' <<< "$line")
	chain=${name%%_*}
	rest=${name#*_}
	num=$((10#${rest%%_*}))
	if [ "$chain" != "md" ] && [ "$chain" != "fin" ]; then
		echo "FAIL 登记表: 登记 $name 前缀非 md|fin（分链制命名 md_NNNN_*/fin_NNNN_*）" >&2
		fail=1
		continue
	fi
	# ① 链内唯一
	key="$chain/$ver"
	if [ -n "${SEEN[$key]:-}" ]; then
		echo "FAIL 登记表: $chain 链版本号 $ver 重复登记（${SEEN[$key]} 与 $name）——同号后登记者被 silent-skip" >&2
		fail=1
	fi
	SEEN[$key]=$name
	# ⑤ 链内单调（登记序=版本升序）
	if [ -n "${LAST[$chain]:-}" ] && [ "$ver" -le "${LAST[$chain]}" ]; then
		echo "FAIL 登记表: $chain 链登记非单调（$name 乱序）——分链后链即序，乱序=红" >&2
		fail=1
	fi
	LAST[$chain]=$ver
	# ② 链内序号=登记号 + 文件实存
	if [ "$num" != "$ver" ]; then
		echo "FAIL 登记表: 登记 $ver 与文件链内序号 $num 失配（$name）——序号语义分裂" >&2
		fail=1
	fi
	if [ ! -f "migrations/$name" ]; then
		echo "FAIL 登记表: 登记 $ver 的 $name 文件不存在（幽灵登记——部署即炸）" >&2
		fail=1
	fi
	VEROF[$name]=$ver
done < <(grep -hE '^\s*\{[0-9]+, *"(md|fin)_[0-9]+_' $REGS)

# ③ 孤儿文件（migrations/*.sql 每文件恰一条登记）
while IFS= read -r f; do
	name=$(basename "$f")
	[ "$name" = "migrations.go" ] && continue
	[ "$name" = "domain.tsv" ] && continue
	grep -qF "\"$name\"" $REGS || {
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
BRIDGE="md_ticks md_mark_index md_signal md_dcd_product md_funding md_macro_event fin_proposal fin_opp_eval fin_strategy fin_venue_untradeable fin_venue_candidate fin_venue_terminal_state fin_opp_hypothesis fin_opp_template fin_trx_vote_apr fin_trx_account fin_trx_txs fin_cmc_history fin_cmc_positions md_tick_scan fin_sub_request"
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

# ⑧ 裸 GRANT 闸（CI/fresh 测试库无生产手工角色——roster_rw 类裸授炸链
# 0071/0098/0161/0191 四案实证）：链外角色（非本链 CREATE 者）GRANT 必须
# 在 DO $$ IF EXISTS(pg_roles) 块内；顶格（DO 块外）GRANT 只许授给
# 链内自建角色或 PUBLIC。
CREATED_ROLES=$(grep -rhoE 'CREATE +(ROLE|USER) +[a-z_][a-z_0-9]*' migrations/*.sql \
	| awk '{print $3}' | sort -u)
while IFS= read -r hit; do
	f="${hit%%:*}"; rest="${hit#*:}"; ln="${rest%%:*}"; stmt="${rest#*:}"
	role=$(sed -nE 's/.*\bTO +([a-z_][a-z_0-9]*).*/\1/p' <<< "$stmt")
	[ -z "$role" ] || [ "$role" = "PUBLIC" ] && continue
	if ! grep -qx "$role" <<< "$CREATED_ROLES"; then
		echo "FAIL $f:$ln: 顶格 GRANT TO $role（链外角色无 IF EXISTS 门——fresh 测试库无此角色即 SQLSTATE 42704 炸链，canon grant-not-in-migration-silent-dead）" >&2
		fail=1
	fi
done < <(awk '
	/^[[:space:]]*DO[[:space:]]+\$[a-zA-Z_0-9]*\$/ {blk=1}
	/^[[:space:]]*END[[:space:]]+\$[a-zA-Z_0-9]*\$[[:space:]]*;/ {blk=0}
	/^[[:space:]]*GRANT[[:space:]]/ && blk==0 {print FILENAME":"FNR":"$0}
' migrations/*.sql)

# ⑨ depends_on 声明合法性（工单#25800——跨支迁移序号倒挂治本）：头区
# 声明解析（mig_dep_lib 共用件）——目标须在册同链且链内号<声明者
# （Migrate 应用序=号序，依赖必先施；号序倒挂=静默语义反转）；头区外
# 错位声明另红（防作者以为声明了实则闸不见）。无声明=现状不禁。
if [ ! -f "$MIGDEP_LIB" ]; then
	echo "FAIL $MIGDEP_LIB: depends_on 解析件缺席（⑨ 判据残缺）" >&2
	fail=1
fi
while IFS= read -r f; do
	name=$(basename "$f")
	dv=${VEROF[$name]:-}
	[ -n "$dv" ] || continue   # 未登记文件③已红——⑨不重复判
	chain=${name%%_*}
	while IFS= read -r dep; do
		[ -n "$dep" ] || continue
		case "$dep" in
			md_*.sql|fin_*.sql) ;;
			*)
				echo "FAIL $f: depends_on 值非法 '$dep'（须=迁移文件名 fin_NNNN_*.sql|md_NNNN_*.sql）" >&2
				fail=1
				continue ;;
		esac
		pv=${VEROF[$dep]:-}
		if [ -z "$pv" ]; then
			echo "FAIL $f: depends_on 目标 $dep 未登记（依赖不可施——先落依赖或改名）" >&2
			fail=1
			continue
		fi
		if [ "${dep%%_*}" != "$chain" ]; then
			echo "FAIL $f: depends_on 跨链 $dep（依赖须同链——两链应用域不同）" >&2
			fail=1
			continue
		fi
		if [ "$pv" -ge "$dv" ]; then
			echo "FAIL $f: depends_on $dep 版=$pv ≥ 声明者版=$dv（应用序=号序——依赖号须小；倒挂=静默语义反转）" >&2
			fail=1
		fi
	done < <(mig_dep_parse < "$f")
	while IFS= read -r stray; do
		[ -n "$stray" ] || continue
		echo "FAIL $f: depends_on 声明错位（头区外——只在文件头注释区生效；见 '$stray'）" >&2
		fail=1
	done < <(mig_dep_stray < "$f")
done < <(find migrations -maxdepth 1 -name '*.sql' | sort)

# ⑩ 破坏性 DDL 存量违例 lint（工单#25799——形①预演闸）：测试 migrate
# 恒空表，存量违例部署期才炸。本闸=lint 渐进启用——水位以下（fin≤215/
# md≤1100，本闸落地时点实测册）豁免不回杀在库件；水位以上新迁移
# 走 python 语句级扫描（剥注释/字符串/dollar-quote 体后按 ; 切——
# $$ 函数体内假命中防）。形②部署前快照影子库 dry-run 评估：
# backup_pg.sh 有 pg_dump 日滚 14 份径但无现成影子库重放管道件——
# dump 重建+双链域挂 Migrate=新管道件成本超本票档，记票面（存量
# 违例残面=VALIDATE 败显式炸+告警分诊径兜）。
REH_WM_FIN=215; REH_WM_MD=1100 # 水位=闸落地时点实测册：fin_0215/md_1100(gridfit词表扩)为闸写作→落地并线隙产物,归存量不回杀
NEW_MIGS=""
while IFS= read -r f; do
	b=$(basename "$f"); n=${b#*_}; n=${n%%_*}
	case "$n" in ''|*[!0-9]*) continue ;; esac
	case "$b" in
		fin_*) [ $((10#$n)) -gt "$REH_WM_FIN" ] && NEW_MIGS="${NEW_MIGS}${f}"$'\n' ;;
		md_*)  [ $((10#$n)) -gt "$REH_WM_MD"  ] && NEW_MIGS="${NEW_MIGS}${f}"$'\n' ;;
	esac
done < <(find migrations -maxdepth 1 -name '*.sql' | sort)
NEW_MIGS=$(printf '%s' "$NEW_MIGS" | sed '/^$/d')
if [ -n "$NEW_MIGS" ]; then
	if ! command -v python3 >/dev/null 2>&1; then
		echo "FAIL python3 缺席——⑩ 判据残缺（存量违例 lint 无法判）" >&2
		fail=1
	else
		mapfile -t MIG10 < <(printf '%s\n' "$NEW_MIGS")
		OUT10=$(python3 - "${MIG10[@]}" <<'PYEOF'
import re, sys

WAIVER = re.compile(r'mig-rehearsal-ok\s*:\s*\S')
WS = re.compile(r'\s+')
DOLLAR = re.compile(r'\$[A-Za-z_0-9]*\$')

def split_stmts(sql):
	out, buf, i, n = [], [], 0, len(sql)
	while i < n:
		if sql.startswith('--', i):
			j = sql.find('\n', i)
			i = n if j < 0 else j
			buf.append('\n')
			continue
		if sql.startswith('/*', i):
			j = sql.find('*/', i + 2)
			i = n if j < 0 else j + 2
			buf.append(' ')
			continue
		c = sql[i]
		if c == "'":
			j = i + 1
			while j < n:
				if sql[j] == "'":
					if j + 1 < n and sql[j + 1] == "'":
						j += 2
						continue
					break
				j += 1
			buf.append("''")
			i = j + 1
			continue
		m = DOLLAR.match(sql, i)
		if m:
			tag = m.group(0)
			j = sql.find(tag, i + len(tag))
			i = n if j < 0 else j + len(tag)
			buf.append(' $$ ')
			continue
		buf.append(c)
		if c == ';':
			out.append(''.join(buf))
			buf = []
		i += 1
	if buf:
		out.append(''.join(buf))
	return out

def lint(path):
	try:
		raw = open(path, encoding='utf-8').read()
	except OSError as e:
		return ['FAIL %s: 读取败 %s' % (path, e)]
	waiver = bool(WAIVER.search(raw))
	valids = set()
	stmts = []
	for st in split_stmts(raw):
		ns = WS.sub(' ', st)
		stmts.append(ns)
		m = re.search(r'VALIDATE\s+CONSTRAINT\s+("[^"]+"|[A-Za-z_0-9]+)', ns, re.I)
		if m:
			valids.add(m.group(1).strip('"'))
	bad = []
	for ns in stmts:
		marks = [m.start() for m in re.finditer(r'ADD\s+CONSTRAINT\b', ns, re.I)]
		for k, st0 in enumerate(marks):
			st1 = marks[k + 1] if k + 1 < len(marks) else len(ns)
			seg = ns[st0:st1]
			m = re.match(r'ADD\s+CONSTRAINT\s+("[^"]+"|[A-Za-z_0-9]+)\s+(CHECK|FOREIGN\s+KEY)\b', seg, re.I)
			if not m:
				continue
			name = m.group(1).strip('"')
			if not re.search(r'\bNOT\s+VALID\b', seg[m.end():], re.I):
				bad.append('FAIL %s: ADD CONSTRAINT %s %s 缺 NOT VALID——存量校验须两段式：ADD ... NOT VALID（即时挂）+同文件 ALTER TABLE ... VALIDATE CONSTRAINT %s（分离扫）（工单#25799⑩）' % (path, name, re.sub(r'\s+', ' ', m.group(2).upper()), name))
			elif name not in valids:
				bad.append('FAIL %s: ADD CONSTRAINT %s 带 NOT VALID 但同文件无 VALIDATE CONSTRAINT %s——存量行永不校验=静默豁免洞（工单#25799⑩）' % (path, name, name))
		risky = []
		if re.search(r'ADD\s+CONSTRAINT\s+("[^"]+"|[A-Za-z_0-9]+)\s+(UNIQUE|PRIMARY\s+KEY|EXCLUDE)\b', ns, re.I):
			risky.append('ADD CONSTRAINT UNIQUE/PK/EXCLUDE')
		if re.search(r'\bADD\s+(PRIMARY\s+KEY|UNIQUE|CHECK|FOREIGN\s+KEY|EXCLUDE)\s*\(', ns, re.I):
			risky.append('裸 ADD PK/UNIQUE/CHECK/FK/EXCLUDE')
		if re.search(r'\bSET\s+NOT\s+NULL\b', ns, re.I):
			risky.append('SET NOT NULL')
		if re.search(r'\bADD\s+COLUMN\b', ns, re.I) and re.search(r'\bNOT\s+NULL\b', ns, re.I) and not re.search(r'\bDEFAULT\b', ns, re.I):
			risky.append('ADD COLUMN NOT NULL 无 DEFAULT')
		if re.search(r'\bCREATE\s+UNIQUE\s+INDEX\b', ns, re.I):
			risky.append('CREATE UNIQUE INDEX')
		if risky and not waiver:
			bad.append('FAIL %s: 破坏性 DDL 不可 NOT VALID 型（%s）——存量违例部署期才炸；须 `-- mig-rehearsal-ok: <理由>` 豁免或改两段式（工单#25799⑩）' % (path, '、'.join(risky)))
	return bad

allbad = []
for p in sys.argv[1:]:
	try:
		allbad.extend(lint(p))
	except Exception as e:
		allbad.append('FAIL %s: lint 内部错 %s' % (p, e))
for b in allbad:
	print(b)
sys.exit(1 if allbad else 0)
PYEOF
) && CRC10=0 || CRC10=$?   # tolerate: 违例=rc1 落下行判据——输出非空即红；rc 非0 且输出空=lint 崩溃按残缺红
		if [ -z "$OUT10" ] && [ "$CRC10" -ne 0 ]; then
			echo "FAIL python3 破坏性 DDL lint 崩溃（rc=$CRC10 无判词——⑩ 判据残缺）" >&2
			fail=1
		elif [ -n "$OUT10" ]; then
			printf '%s\n' "$OUT10" >&2
			fail=1
		fi
	fi
fi

if [ "$fail" -eq 1 ]; then
	echo "❌ check_migrations: 断言红——撞号/串链/桥表漂移/白名单缺册/裸GRANT/depends_on 声明/破坏性 DDL 禁提交（canon migration-number-collision + D-0030 §六 + 工单#25800/#25799）" >&2
	exit 1
fi
echo "OK: check_migrations（十断言过——链内唯一/序号实存/零孤儿/md_* 键域/两链单调/八桥 DDL 同形/域白名单/裸GRANT 白名单/depends_on 声明合法/破坏性 DDL lint）"
