#!/usr/bin/env bash
# recall.sh —— 唤知会话 KB 召回件（T-0521：经验库只写不读病灶根修——
# 教训矿 canon 入库而处置面无召回径,检索物料.tsv 生成了没人 grep）。
# 语义：扫未决升件信号面→抽域键（esc_scope/esc_class 原子词+note 内
# ascii 域词）→按词面命中数排检索物料.tsv→TOP8 案卷行吐 stdout——
# run.sh 拼进会话 prompt,每场会话起手头已带命中案卷（先查案再判案）。
# fail-open：DSN 读不出/库缺/零未决=吐空静默退——召回败不阻会话主径。
set -u
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO=$(cd -- "$SCRIPT_DIR/../.." && pwd)
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$DSN" ] || exit 0

# 未决升件面（dispatch 同口径 7d 窗）：scope/class 原子词+note ascii 域词。
# scope/class 本身即召回键（orphan_eng_lesson/commercial_stall/商业产出——
# keywords 列钦定同词域）；note 只抽 ascii 域词不整句切（中文分词不进
# grep 径——中文召回键由 class 原子词承担）。
UNRES="SELECT e.esc_scope s,COALESCE(e.esc_class,'') c,COALESCE(e.note,'') n
	FROM ops_event e
	WHERE e.event='升件' AND e.ts>now()-interval '7 days'
	  AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
	    AND x.note ~ ('升件#'||e.id::text||'([^0-9]|\$)'))"
# scope/class=原子召回键（| 分列直取）；note 只抽 ascii 域词。
ROWS=$(psql "$DSN" -qAt -c "$UNRES" 2>/dev/null)
[ -n "$ROWS" ] || exit 0
TOKENS=$({
	printf '%s\n' "$ROWS" | awk -F'|' '{print $1; print $2}'
	printf '%s\n' "$ROWS" | cut -d'|' -f3 | grep -oE '[a-zA-Z][a-zA-Z0-9_]{2,}'
} | tr ' ' '\n' | grep -vE '^$' | sort -u)
[ -n "$TOKENS" ] || exit 0

# 打分：行=一条目,分=命中 token 数（TSV 全列同查——keywords 钦定键域,
# display/text 正文补 recall）。TOP8 封顶=注入预算闸防 prompt 淹没主务。
for lib in 工程经验库 经济经验库; do
	TSV="$REPO/docs/$lib/检索物料.tsv"
	[ -s "$TSV" ] || continue
	while IFS= read -r tok; do
		grep -F "$tok" "$TSV" 2>/dev/null
	done <<<"$TOKENS" | awk -F'\t' -v lib="$lib" '{
		c[$2]++; line[$2]=$0
	} END {
		for (i in c) print c[i] "\t" lib "\t" i "\t" line[i]
	}'
done | sort -rn | head -8 | awk -F'\t' -v repo="$REPO" '{
	# 行形:count lib id <TSV原行>——TSV 列=lib id kind status risk
	# display text keywords scope skills,故 display 落在第 9 列。
	printf "- %s:%s — %s（%s/docs/%s/条目/%s.md，命中 %d 词）\n",
		$2, $3, $9, repo, $2, $3, $1
}'
