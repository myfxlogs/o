#!/usr/bin/env bash
# 文件规模门禁（D-077 / D-588 棘轮，自 nox 誊入）：
# Go/TS 源文件 >450 行硬阻断（exit 1）、>300 行棘轮（基线内只许缩、基线外新超标硬阻断）。
# 豁免 proto 生成代码与测试文件。由 pre-commit 调用，也可单独跑（查全仓，与 CI 同令）。
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
SOFT=300
HARD=450

# exempt：proto 生成代码（*/gen/）与 *_test.go + docs/ 下 .go 证据探针（//go:build ignore）
exempt() {
  case "$1" in
    internal/*/gen/*|internal/gen/*|web/src/gen/*|*_test.go) return 0 ;;
    docs/*.go) return 0 ;;
  esac
  return 1
}

# 既有超标文件白名单——O 新仓零存量，白名单天生为空（渐进收紧条款保留）。
LEGACY_OVERSIZE=()

is_legacy_oversize() {
  local rel="$1"
  for f in "${LEGACY_OVERSIZE[@]}"; do
    [ "$rel" = "$f" ] && return 0
  done
  return 1
}

# D-588 棘轮基线：基线内文件只许缩不许长；基线外新超标=硬阻断。
BASELINE="$ROOT/scripts/check-lines.baseline"
baseline_n() {
  [ -f "$BASELINE" ] && awk -v f="$1" '$1==f{print $2}' "$BASELINE" || true
}

hard=0 soft=0
while IFS= read -r f; do
  rel="${f#"$ROOT"/}"
  exempt "$rel" && continue
  n="$(wc -l < "$f")"
  if [ "$n" -gt "$HARD" ]; then
    if is_legacy_oversize "$rel"; then
      echo "⚠️  $rel: $n 行（> $HARD 既有超标，白名单豁免，待拆分）"
      soft=1
    else
      echo "❌ $rel: $n 行（> $HARD 硬阻断）"
      hard=1
    fi
  elif [ "$n" -gt "$SOFT" ]; then
    bn="$(baseline_n "$rel")"
    if [ -z "$bn" ]; then
      echo "❌ $rel: $n 行（> $SOFT 新超标硬阻断——拆分或减行；棘轮基线禁止新增）"
      hard=1
    elif [ "$n" -gt "$bn" ]; then
      echo "❌ $rel: $n 行（棘轮违例——基线 $bn 行只许缩不许长）"
      hard=1
    else
      echo "⚠️  $rel: $n 行（> $SOFT 基线内遗留 $bn，待拆分）"
      soft=1
    fi
  fi
done < <(find "$ROOT" -type f \( -name '*.go' -o -name '*.ts' -o -name '*.tsx' \) \
  -not -path '*/node_modules/*' -not -path '*/.git/*' -not -path '*/ref/*')

if [ "$hard" -eq 1 ]; then
  echo "❌ check-lines: 超标/棘轮违例存在，提交被阻断（>450 硬线 / >300 棘轮基线）"
  exit 1
fi
if [ "$soft" -eq 1 ]; then
  echo "⚠️  check-lines: 存在 >$SOFT 行的源文件（软警告，建议拆分）"
fi
exit 0
