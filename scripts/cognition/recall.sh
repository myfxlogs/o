#!/usr/bin/env bash
# recall.sh —— 唤知会话 KB 召回件（T-0521：经验库只写不读病灶根修——
# 教训矿 canon 入库而处置面无召回径,库攒了没人查）。
# 语义：扫未决升件信号面→抽域键（esc_scope/esc_class 原子词+note 内
# ascii 域词）→按词面命中数排条目→TOP8 案卷行吐 stdout——run.sh 拼进
# 会话 prompt,每场会话起手头已带命中案卷（先查案再判案）。
# 料面=docs/*经验库/条目/*.md 正本直查（检索物料.tsv=gitignored 再生物
# 不到 SG——grep 正本哪都齐不依赖生成件）。fail-open：败=吐空静默退。
set -u
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO=$(cd -- "$SCRIPT_DIR/../.." && pwd)
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$DSN" ] || exit 0

# 未决升件面（dispatch 同口径 7d 窗）：scope/class=原子召回键（| 分列
# 直取——scope 词即条目 keywords 钦定域）；note 只抽 ascii 域词不整句
# 切（中文分词不进 grep 径——中文召回键由 class 原子词承担）。
UNRES="SELECT e.esc_scope s,COALESCE(e.esc_class,'') c,COALESCE(e.note,'') n
	FROM ops_event e
	WHERE e.event='升件' AND e.ts>now()-interval '7 days'
	  AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
	    AND x.note ~ ('升件#'||e.id::text||'([^0-9]|\$)'))"
ROWS=$(psql "$DSN" -qAt -c "$UNRES" 2>/dev/null)
[ -n "$ROWS" ] || exit 0
TOKENS=$({
	printf '%s\n' "$ROWS" | awk -F'|' '{print $1; print $2}'
	printf '%s\n' "$ROWS" | cut -d'|' -f3 | grep -oE '[a-zA-Z][a-zA-Z0-9_]{2,}'
} | tr ' ' '\n' | grep -vE '^$' | sort -u)
[ -n "$TOKENS" ] || exit 0

# 打分：件=一条目,分=命中 token 数（frontmatter keywords/id/summary+
# 正文同查——词面命中即分）。TOP8 封顶=注入预算闸防 prompt 淹没主务。
for lib in 工程经验库 经济经验库; do
	DIR="$REPO/docs/$lib/条目"
	[ -d "$DIR" ] || continue
	while IFS= read -r tok; do
		grep -l -F "$tok" "$DIR"/*.md 2>/dev/null
	done <<<"$TOKENS"
done | sort | uniq -c | sort -rn | head -8 | while read -r n f; do
	# 显示名=summary 行（frontmatter）缺→首个 ## 标题；id=件名去 .md
	id=$(basename "$f" .md)
	disp=$(grep -m1 -oE '^summary: .*' "$f" | sed -E 's/^summary: //;s/ *$//;s/^(.{60}).*/\1…/')
	[ -n "$disp" ] || disp=$(grep -m1 -oE '^## .*' "$f" | sed -E 's/^## //;s/^(.{60}).*/\1…/')
	lib=$(basename "$(dirname "$(dirname "$f")")")
	printf -- "- %s:%s — %s（%s，命中 %d 词）\n" "$lib" "$id" "$disp" "$f" "$n"
done
