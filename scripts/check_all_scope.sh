#!/usr/bin/env bash
# check_all_scope.sh —— 测谱分档判定件（工单#25798 从 check_all.sh 抽出）：
# stdin=变更净路径集（每行一路径，check_all 已提交面∪工作树面并集）；
# stdout=两行 MODE=<docs-only|scripts-sh|scoped|merge-scoped|full> / TEST_PKGS=<包集>。
# 判据与原内联块逐字同——抽出理由=变更样本可测：check_all_scope_test.sh
# 直接喂路径集证「cmd/core 依赖闭包内册外包恒测」不必跑全链。
# 四档语义不动：docs-only 免测 / scoped 定向谱 / merge-scoped+迁移proto
# 附加种子 / full 回退 fail-safe；恒测册=现册∪cmd/core 实算依赖闭包
# （钱径喂数侧包 internal/data/*|mt5|earn|platform/* 册外闭包内者入册
# ——编译绿≠行为绿 B1 根修；生成式自裁：显式册随码面漂移必陈腐同病理，
# 实算闭包自维护；go list 败/空=判不准→SCOPE_OK=0 回退 full fail-closed）。
set -uo pipefail
cd "$(git rev-parse --show-toplevel)"

CHANGED=$(cat)
MODE=full; TEST_PKGS=""
ALWAYS_PKGS="o/internal/risk/... o/internal/ledger/... o/internal/trading/... o/internal/guard o/internal/portfolio/lifecycle/..."
ALL_DOCS=1; HAS_GO=0; HAS_MIG=0; HAS_PROTO=0; HAS_OTHER=0
CODE_SEEN=0; SH_ONLY=1; SH_SEEN=0
while IFS= read -r p; do
  [ -n "$p" ] || continue
  case "$p" in
    docs/*|skills/*|*.md) ;;
    *.go)             HAS_GO=1;    CODE_SEEN=1; ALL_DOCS=0; SH_ONLY=0 ;;
    migrations/*.sql) HAS_MIG=1;   CODE_SEEN=1; ALL_DOCS=0; SH_ONLY=0 ;;
    proto/*)          HAS_PROTO=1; CODE_SEEN=1; ALL_DOCS=0; SH_ONLY=0 ;;
    # scripts/*.sh 与 .github/*=scripts-sh 档候选（中性基建件——Go 对
    # .sh/CI-yaml 零耦合；ALL_DOCS 同旧律必清，docs+sh 混面免进
    # docs-only=T-0433 首版漏清 ALL_DOCS 红桩锚同坑防）。
    scripts/*.sh|.github/*) SH_SEEN=1; ALL_DOCS=0 ;;
    # 安全基建件=中性：不算 docs（ALL_DOCS=0——独现/伴 docs 落
    # full 不免测）不算清单外（HAS_OTHER 不置——伴码件入
    # merge-scoped 定向谱）。T-0433 首版漏清 ALL_DOCS 致
    # docs+scripts 面误判 docs-only 免测=fail-open，红桩锚在
    # scripts/meta/tests/check_all_scope_test.sh 场景①。
    # scripts/ 非 .sh 件（tsv 等被 Go 消费）不中 scripts-sh 档。
    scripts/*)        ALL_DOCS=0; SH_ONLY=0 ;;
    *)   HAS_OTHER=1; ALL_DOCS=0; SH_ONLY=0 ;;
  esac
done <<< "$CHANGED"
if [ "$ALL_DOCS" = 1 ]; then
  MODE=docs-only
elif [ "$SH_ONLY" = 1 ] && [ "$SH_SEEN" = 1 ]; then
  MODE=scripts-sh
elif [ "$HAS_OTHER" = 0 ] && [ "$CODE_SEEN" = 1 ]; then
  # 定向谱候选（scoped/merge-scoped 同构）：种子=.go 包映射+
  # 迁移→pgstore+proto→gen 包映射；闭包+恒测册同式；任一步不确定回退 full
  SCOPE_OK=1; SEEDS=""
  while IFS= read -r f; do
    [ -n "$f" ] || continue
    pkg=$(go list "./$(dirname "$f")" 2>/dev/null) \
      || { SCOPE_OK=0; break; }
    SEEDS="$SEEDS $pkg"
  done <<< "$(printf '%s\n' "$CHANGED" | grep '\.go$' || true)" # tolerate: 空集合法（mig/proto-only 面）
  # 迁移面定向（T-0433③）：fin_/md_ 迁移→pgstore migrate 测+受影响包
  # （check_migrations 闸在上段门禁位已恒跑）
  if [ "$SCOPE_OK" = 1 ] && [ "$HAS_MIG" = 1 ]; then
    pkg=$(go list ./internal/store/pgstore 2>/dev/null) \
      || SCOPE_OK=0
    [ "$SCOPE_OK" = 1 ] && SEEDS="$SEEDS $pkg"
  fi
  # proto 面定向：proto/<dir>/*→o/internal/gen/<dir> 包映射+buf build 闸
  if [ "$SCOPE_OK" = 1 ] && [ "$HAS_PROTO" = 1 ]; then
    command -v buf >/dev/null 2>&1 && buf build >/dev/null 2>&1 \
      || SCOPE_OK=0   # proto 闸：buf 缺席/编译败=判不准→full
    while IFS= read -r f; do
      [ -n "$f" ] || continue
      d=${f#proto/}; d=${d%/*}
      pkg=$(go list "./internal/gen/$d" 2>/dev/null) \
        || { SCOPE_OK=0; break; }
      SEEDS="$SEEDS $pkg"
    done <<< "$(printf '%s\n' "$CHANGED" | grep '^proto/' || true)" # tolerate: 空集合法
  fi
  if [ "$SCOPE_OK" = 1 ]; then
    IG=$(mktemp) || SCOPE_OK=0
    if [ "$SCOPE_OK" = 1 ]; then
      go list -f '{{.ImportPath}}|{{join .Imports ","}}|{{join .TestImports ","}}|{{join .XTestImports ","}}' ./... > "$IG" 2>/dev/null \
        || SCOPE_OK=0
    fi
    if [ "$SCOPE_OK" = 1 ]; then
      CLOSURE=$(python3 - "$IG" $SEEDS 2>/dev/null <<'PY'
import sys
ig, seeds = sys.argv[1], set(sys.argv[2:])
rev = {}
for line in open(ig):
    parts = line.rstrip("\n").split("|")
    if len(parts) < 4:
        continue
    for d in parts[1].split(",") + parts[2].split(",") + parts[3].split(","):
        if d:
            rev.setdefault(d, set()).add(parts[0])
seen, stack = set(seeds), list(seeds)
while stack:
    for imp in rev.get(stack.pop(), ()):
        if imp not in seen:
            seen.add(imp); stack.append(imp)
print(" ".join(sorted(seen)))
PY
) && [ -n "$CLOSURE" ] || SCOPE_OK=0
    fi
    [ -n "${IG:-}" ] && rm -f "$IG"
  fi
  if [ "$SCOPE_OK" = 1 ]; then
    # 恒测册包集展开（败=回退 full——册内包消失=结构变了不猜）
    AP=$(go list $ALWAYS_PKGS 2>/dev/null) || SCOPE_OK=0
    if [ "$SCOPE_OK" = 1 ]; then
      if [ "$HAS_MIG" = 1 ] || [ "$HAS_PROTO" = 1 ]; then
        MODE=merge-scoped
      else
        MODE=scoped
      fi
      # 恒测册∪cmd/core 实算依赖闭包（工单#25798——钱径喂数侧包入册
      # =编译绿→行为绿；本模块径 grep '^o/' 滤 std/外部模块依赖；
      # go list 败/空=判不准→SCOPE_OK=0 回退 full fail-closed）
      CORE_DEPS=$(go list -deps ./cmd/core 2>/dev/null | grep '^o/' | tr '\n' ' ')
      [ -n "$CORE_DEPS" ] || SCOPE_OK=0
      if [ "$SCOPE_OK" = 1 ]; then
        TEST_PKGS=$(printf '%s\n%s\n%s\n' "$CLOSURE" "$AP" "$CORE_DEPS" | tr ' ' '\n' | grep -v '^$' | sort -u | tr '\n' ' ')
      fi
    fi
  fi
  [ "$SCOPE_OK" = 1 ] || MODE=full
fi

echo "MODE=$MODE"
echo "TEST_PKGS=$TEST_PKGS"
