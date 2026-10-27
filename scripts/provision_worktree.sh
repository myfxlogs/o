#!/usr/bin/env bash
# provision_worktree.sh — 新 worktree 环境件补齐（gitignored 构建产物：
# internal/gen/* proto 生成件 + web/dist 前端构建件）。
# 背景：2026-10-27 实证——干净 worktree 里 staticcheck/vet/build/test 全炸
# （gen 包不在+embed dist 无），pre-push/CI 同链闸没法跑。
# 用法：scripts/provision_worktree.sh [worktree路径]（缺省=当前树）
# 源：优先本机 codegen（buf generate+npm build）；兜底从 /opt/o 部署树拷
# （gitignored 产物不入库，仓内无正本——部署树是唯一恒有面）。
set -euo pipefail
WT="${1:-$(git rev-parse --show-toplevel)}"
SRC=/opt/o

echo "[provision] $WT"
need=0
[ -d "$WT/internal/gen/mtapi/mt5" ] || need=1
[ -f "$WT/web/dist/index.html" ] || need=1
if [ "$need" = 0 ]; then
  echo "[provision] 环境件已在（gen+dist）——免补"
  exit 0
fi

# 源①：部署树拷（最快径——部署树恒有构建产物）
if [ -d "$SRC/internal/gen/mtapi/mt5" ] && [ -f "$SRC/web/dist/index.html" ]; then
  mkdir -p "$WT/internal/gen" "$WT/web"
  cp -r "$SRC/internal/gen/"* "$WT/internal/gen/" 2>/dev/null || true
  cp -r "$SRC/web/dist" "$WT/web/" 2>/dev/null || true
  echo "[provision] 从 $SRC 拷 gen+dist 完"
else
  echo "[provision] 部署树无件——走本机 codegen 慢径（需 buf/node）"
  cd "$WT"
  buf generate 2>/dev/null || echo "  buf generate 败——gen 件仍缺，go 闸会红"
  (cd web && npm ci --legacy-peer-deps >/dev/null 2>&1 && npm run build >/dev/null 2>&1) \
    || echo "  npm build 败——web/dist 仍缺，core 编译会红"
fi

[ -d "$WT/internal/gen/mtapi/mt5" ] && [ -f "$WT/web/dist/index.html" ] \
  && echo "[provision] 环境件齐——可过全链" \
  || { echo "[provision] 件仍缺——查上方败因" >&2; exit 1; }
