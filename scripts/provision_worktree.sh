#!/usr/bin/env bash
# provision_worktree.sh — 新 worktree 环境件补齐（gitignored 构建产物三件：
# internal/gen/* proto 生成件 + web/src/gen 前端生成件 + web/dist 前端构建件）。
# 背景：2026-10-27 实证——干净 worktree 里 staticcheck/vet/build/test 全炸
# （gen 包不在+embed dist 无），pre-push/CI 同链闸没法跑。
# T-0229：补 web/src/gen（四显形坑——缺则 vite dev/build 空白/炸，KB
# worktree-provision-missing-web-gen）；dist=快照件（仅供闸/dev，生产前端
# 唯一径=deploy_image.sh 镜像内 vite 重建——worktree dist 永不进生产）。
# 用法：scripts/provision_worktree.sh [worktree路径]（缺省=当前树）
# 源：优先本机 codegen（buf generate+npm build）；兜底从 /opt/o 部署树拷
# （gitignored 产物不入库，仓内无正本——部署树是唯一恒有面）。
set -euo pipefail
WT="${1:-$(git rev-parse --show-toplevel)}"
SRC=/opt/o

echo "[provision] $WT"
need=0
[ -d "$WT/internal/gen/mtapi/mt5" ] || need=1
[ -d "$WT/web/src/gen/dashboardv1" ] || need=1
[ -f "$WT/web/dist/index.html" ] || need=1
if [ "$need" = 0 ]; then
  echo "[provision] 环境件已在（gen+dist）——免补"
  exit 0
fi

# 源①：部署树拷（最快径——部署树恒有构建产物）
if [ -d "$SRC/internal/gen/mtapi/mt5" ] && [ -f "$SRC/web/dist/index.html" ]; then
  mkdir -p "$WT/internal/gen" "$WT/web/src"
  cp -r "$SRC/internal/gen/"* "$WT/internal/gen/" 2>/dev/null || true
  [ -d "$SRC/web/src/gen/dashboardv1" ] \
    && cp -r "$SRC/web/src/gen" "$WT/web/src/" 2>/dev/null || true
  cp -r "$SRC/web/dist" "$WT/web/" 2>/dev/null || true
  [ -d "$WT/web/src/gen/dashboardv1" ] \
    || (cd "$WT" && buf generate --template buf.gen.web.yaml 2>/dev/null) \
    || echo "  web gen 慢补败——vite dev/build 会炸（缺 ../gen/dashboardv1 引用）"
  echo "[provision] 从 $SRC 拷 gen×2+dist 完（dist=快照仅供闸/dev——生产走 deploy_image.sh）"
else
  echo "[provision] 部署树无件——走本机 codegen 慢径（需 buf/node）"
  cd "$WT"
  buf generate 2>/dev/null || echo "  buf generate 败——gen 件仍缺，go 闸会红"
  buf generate --template buf.gen.web.yaml 2>/dev/null \
    || echo "  web gen 败——vite dev/build 会炸（缺 ../gen/dashboardv1 引用）"
  (cd web && npm ci --legacy-peer-deps >/dev/null 2>&1 && npm run build >/dev/null 2>&1) \
    || echo "  npm build 败——web/dist 仍缺，core 编译会红"
fi

[ -d "$WT/internal/gen/mtapi/mt5" ] && [ -d "$WT/web/src/gen/dashboardv1" ] \
  && [ -f "$WT/web/dist/index.html" ] \
  && echo "[provision] 环境件齐——可过全链" \
  || { echo "[provision] 件仍缺——查上方败因" >&2; exit 1; }
