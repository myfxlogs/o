#!/usr/bin/env bash
# bootstrap_worktree.sh — 新 worktree gen 件一键再生（T-0375 件②）。
# 病灶：生成件不入库（internal/gen+web/src/gen gitignored）——裸 wt build/
# vet/pre-commit 全炸，两案实证靠手拷部署树续命（T-0371 审查 wt、secp wt）。
# 律：正径=buf generate（本支 proto 自产——与本支 commit 恒同步）：
#   mtapi 模板纯 Go 插件裸 wt 即产（实测 rc=0）；
#   默认模板含 protoc-gen-es（web/node_modules 依赖）——node_modules 在才
#   试全谱（要全正径先 npm ci 再重跑本件）；buf 败/件缺=降级径 rsync 部署
#   树快照（明示非静默——快照可能与本支 proto 不同步；整面替换防混血）。
#   dist=go:embed 依赖（web/embed.go）快照拷，仅供闸/dev——生产前端唯一
#   径=deploy_image.sh 镜像内 vite 重建，wt dist 永不进生产。
# 用法：scripts/bootstrap_worktree.sh [worktree路径]（缺省=当前树）
set -euo pipefail
WT="${1:-$(git rev-parse --show-toplevel)}"
SRC="${BOOTSTRAP_SRC:-/opt/o}"

echo "[bootstrap] $WT"
{ [ -d "$WT/.git" ] || [ -f "$WT/.git" ]; } \
  || { echo "[bootstrap] 非 git 树：$WT" >&2; exit 1; }

# ── 正径：buf 再生（能产面逐模板试——失败不中止,降级面后判） ──
if command -v buf >/dev/null 2>&1; then
  echo "[bootstrap] 正径 buf mtapi 模板（internal/gen/mtapi——裸 wt 可行面）"
  (cd "$WT" && buf generate --template buf.gen.mtapi.yaml) \
    || echo "  buf mtapi 败——internal/gen 降级径接手"
  if [ -e "$WT/web/node_modules/.bin/protoc-gen-es" ]; then
    echo "[bootstrap] 正径 buf 默认模板（internal/gen 余面+web/src/gen）"
    (cd "$WT" && buf generate) \
      || echo "  buf 默认模板败——两 gen 面降级径接手"
  else
    echo "  protoc-gen-es 缺（无 web/node_modules）——全谱 buf 跳过（全正径先 npm ci 再重跑本件）"
  fi
else
  echo "  buf 缺席——gen 面全走降级径（部署树快照）"
fi

# ── 降级径：部署树 rsync（面缺才接手）。internal/gen 用叠加径（不 --delete
# ——保住 buf 鲜产的 mtapi 本支面）；web gen 无本产可保,整面快照 --delete。 ──
if [ ! -d "$WT/internal/gen/mtapi/mt5" ] || [ ! -d "$WT/internal/gen/dashboardv1" ]; then
  [ -d "$SRC/internal/gen/mtapi/mt5" ] \
    || { echo "[bootstrap] 降级源 $SRC/internal/gen 亦缺——先在部署树补 gen" >&2; exit 1; }
  echo "[bootstrap] 降级径：rsync $SRC/internal/gen →（叠加补缺——快照可能与本支 proto 不同步；全正径先 npm ci 再重跑本件）"
  mkdir -p "$WT/internal/gen"
  rsync -a "$SRC/internal/gen/" "$WT/internal/gen/"
fi
if [ ! -d "$WT/web/src/gen/dashboardv1" ]; then
  [ -d "$SRC/web/src/gen/dashboardv1" ] \
    || { echo "[bootstrap] 降级源 $SRC/web/src/gen 亦缺——先在部署树补 gen" >&2; exit 1; }
  echo "[bootstrap] 降级径：rsync $SRC/web/src/gen →（整面快照,可能与本支 proto 不同步）"
  mkdir -p "$WT/web/src"
  rsync -a --delete "$SRC/web/src/gen/" "$WT/web/src/gen/"
fi

# ── dist：go:embed 依赖——快照拷（生产唯一径=deploy_image.sh） ──
if [ ! -f "$WT/web/dist/index.html" ] && [ -f "$SRC/web/dist/index.html" ]; then
  echo "[bootstrap] dist 快照拷（仅供闸/dev——生产走 deploy_image.sh）"
  rm -rf "$WT/web/dist"
  cp -r "$SRC/web/dist" "$WT/web/dist"
fi

[ -d "$WT/internal/gen/mtapi/mt5" ] && [ -d "$WT/internal/gen/dashboardv1" ] \
  && [ -d "$WT/web/src/gen/dashboardv1" ] && [ -f "$WT/web/dist/index.html" ] \
  && echo "[bootstrap] gen×2+dist 齐——可过全链（构建验证=check_all）" \
  || { echo "[bootstrap] 件仍缺——查上方败因" >&2; exit 1; }
