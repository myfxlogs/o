#!/usr/bin/env bash
# check_tolerate_report.sh —— T-0369 容错必报 lint（机检候选供裁）：
# 全树壳件（*.sh + scripts/hooks/*）扫 `|| true`/`|| :` 容错位——无行内
# `# tolerate: <原因>` 锚且未入 docs/容错位清单.md tolerate-registry
# 机读块者即违例（新增位自觉加锚；存量位走清单对账豁免）。
# --emit-registry：机生当前树豁免键清单（path<TAB>归一行），供清单对账
# 刷新——registry 为机生面禁手编。清单缺席/空册=全量违例 fail-closed。
set -uo pipefail
cd "$(git rev-parse --show-toplevel)"
REG_DOC=docs/容错位清单.md

norm() { sed 's/^[[:space:]]*//;s/[[:space:]]*$//'; }

# 全部 `|| true`/`|| :` 命中行→「path<TAB>归一行」键流（纯注释行免）。
emit_hits() {
  git ls-files -z -- '*.sh' 'scripts/hooks/*' | while IFS= read -r -d '' f; do
    [ -f "$f" ] || continue
    grep -nE '\|\|[[:space:]]*(true|:)' "$f" 2>/dev/null \
      | while IFS=: read -r _ line; do
          case "${line#"${line%%[![:space:]]*}"}" in '#'*) continue;; esac
          printf '%s\t%s\n' "$f" "$(printf '%s' "$line" | norm)"
        done
  done
}

case "${1:-}" in
  --emit-registry) emit_hits | grep -vF '# tolerate:' | sort -u; exit 0 ;;
  -h|--help) sed -n '1,10p' "$0"; exit 0 ;;
  '') ;;
  *) echo "usage: $0 [--emit-registry]" >&2; exit 2 ;;
esac

[ -f "$REG_DOC" ] || { echo "❌ [tolerate-report] $REG_DOC 缺席——容错位豁免册未立" >&2; exit 1; }
REG=$(awk '/^```tolerate-registry/{f=1;next} /^```/{f=0} f' "$REG_DOC")
HITS=$(emit_hits | grep -vF '# tolerate:' | sort -u || true) # tolerate: grep 空集=零命中合法径
if [ -n "$HITS" ]; then
  if [ -n "$REG" ]; then
    BAD=$(printf '%s\n' "$HITS" | grep -vxF -f <(printf '%s\n' "$REG") || true) # tolerate: grep 空集=零违例合法径
  else
    BAD=$HITS
  fi
  if [ -n "$BAD" ]; then
    echo '❌ [tolerate-report] 新增/未册容错位——容错吞败符（双竖+true/冒号）失败面禁只打 stderr：' >&2
    printf '%s\n' "$BAD" | head -20 >&2
    echo "  正解二择：失败面接自报径（ops_event/升件）+行内 '# tolerate: <原因>' 锚；" >&2
    echo "  或确属存量豁免——入 $REG_DOC tolerate-registry 机读块对账" >&2
    exit 1
  fi
fi
echo "✅ [tolerate-report] 容错位全谱有锚/在册" >&2
exit 0
