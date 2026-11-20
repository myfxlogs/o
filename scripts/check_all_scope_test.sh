#!/usr/bin/env bash
# check_all_scope_test.sh —— 测谱分档判定件桩测（工单#25798 验收）：
# 直接喂变更路径集证四断——
# ①恒测册补全：无关变更（gridfit）下 cmd/core 依赖闭包内册外包
#   （internal/data/fact|mt5|earn|platform/config|pgstore|hunger）
#   仍入 TEST_PKGS=恒测册实效（旧册 risk/ledger/trading/guard/
#   lifecycle 之外的钱径喂数侧包从此每跑必测——编译绿≠行为绿根修）；
# ②册外不动：闭包外且不依赖变更包者（research/regime、cmd/collector）
#   不入 TEST_PKGS；
# ③docs-only/full 两档语义不动；
# ④merge-scoped：.go+migrations 伴 → merge-scoped+pgstore 种子在册。
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
SCOPE="$HERE/check_all_scope.sh"
cd "$(git rev-parse --show-toplevel)"
FAIL=0

run_scope() { printf '%s\n' "$1" | bash "$SCOPE" 2>/dev/null; }
has_pkg() { case " $2 " in *" $1 "*) return 0;; *) return 1;; esac; }

# ── ①② scoped 样本：internal/research/gridfit/eval.go 变更 ──
OUT=$(run_scope "internal/research/gridfit/eval.go")
MODE=$(printf '%s\n' "$OUT" | sed -n 's/^MODE=//p')
PKGS=$(printf '%s\n' "$OUT" | sed -n 's/^TEST_PKGS=//p')
[ "$MODE" = "scoped" ] || { echo "FAIL 样本① MODE=$MODE want=scoped"; FAIL=1; }
for p in o/internal/research/gridfit o/cmd/research; do
  has_pkg "$p" "$PKGS" || { echo "FAIL 样本① 缺闭包包 $p"; FAIL=1; }
done
for p in o/internal/data/fact o/internal/mt5 o/internal/earn \
         o/internal/platform/config o/internal/store/pgstore \
         o/internal/portfolio/hunger o/internal/platform/opschain; do
  has_pkg "$p" "$PKGS" || { echo "FAIL 样本① 恒测册缺 $p（闭包内册外包未入册）"; FAIL=1; }
done
for p in o/internal/research/regime o/cmd/collector o/cmd/sweeper; do
  has_pkg "$p" "$PKGS" && { echo "FAIL 样本① 册外包 $p 误入 TEST_PKGS"; FAIL=1; }
done

# ── ③ docs-only / full / scripts-only ──
OUT=$(run_scope "docs/foo.md
skills/x/SKILL.md")
MODE=$(printf '%s\n' "$OUT" | sed -n 's/^MODE=//p')
[ "$MODE" = "docs-only" ] || { echo "FAIL 样本③a MODE=$MODE want=docs-only"; FAIL=1; }
OUT=$(run_scope "go.mod")
MODE=$(printf '%s\n' "$OUT" | sed -n 's/^MODE=//p')
[ "$MODE" = "full" ] || { echo "FAIL 样本③b MODE=$MODE want=full"; FAIL=1; }
# ── ③c/d：scripts-sh 档（第五档——scripts/*.sh|.github/* 全中=scripts-sh
#   免测谱免锁；scripts/ 非 .sh 件如 tsv 被 Go 消费不中档仍 full）──
OUT=$(run_scope "scripts/check_all_scope.sh")
MODE=$(printf '%s\n' "$OUT" | sed -n 's/^MODE=//p')
[ "$MODE" = "scripts-sh" ] || { echo "FAIL 样本③c MODE=$MODE want=scripts-sh"; FAIL=1; }
OUT=$(run_scope "scripts/config-write-domain.tsv")
MODE=$(printf '%s\n' "$OUT" | sed -n 's/^MODE=//p')
[ "$MODE" = "full" ] || { echo "FAIL 样本③d MODE=$MODE want=full"; FAIL=1; }

# ── ④ merge-scoped：.go+migrations 伴 → merge-scoped+pgstore 种子 ──
OUT=$(run_scope "internal/research/gridfit/eval.go
migrations/fin_9999_test.sql")
MODE=$(printf '%s\n' "$OUT" | sed -n 's/^MODE=//p')
PKGS=$(printf '%s\n' "$OUT" | sed -n 's/^TEST_PKGS=//p')
[ "$MODE" = "merge-scoped" ] || { echo "FAIL 样本④ MODE=$MODE want=merge-scoped"; FAIL=1; }
has_pkg "o/internal/store/pgstore" "$PKGS" \
  || { echo "FAIL 样本④ 迁移附加种子 pgstore 缺席"; FAIL=1; }

if [ "$FAIL" = 0 ]; then
  echo "PASS check_all_scope_test 全锚过（恒测册=cmd/core 实算闭包并现册；册外不动；四档语义不动）"
else
  echo "FAIL check_all_scope_test 有锚红" >&2
  exit 1
fi
