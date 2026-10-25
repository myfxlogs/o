#!/usr/bin/env bash
# 宿主侧定时工具重建——cron 裸奔的 go 二进制独立于 o-core 镜像管线，不重建=漂移
# （自 nox build_host_tools.sh 誊入；10-15 部署漏步实证）。
# 用法：./scripts/build_host_tools.sh（repo 根目录跑；需宿主 go toolchain + git）
# O 落位期：tools/* 尚未迁移——跳过缺件目录不炸（工具随各域票逐一落位）。
set -euo pipefail
cd "$(dirname "$0")/.."
OUT="${O_BIN_DIR:-./bin}"
mkdir -p "$OUT"
SHA="$(git rev-parse --short HEAD 2>/dev/null || echo nogit)"
built=0
for t in tools/*/; do
  [ -f "$t/main.go" ] || continue
  n="$(basename "$t")"
  go build -ldflags "-s -w -X main.buildSHA=$SHA" -o "$OUT/$n" "./$t"
  echo "built $OUT/$n @$SHA"
  built=$((built+1))
done
[ "$built" -eq 0 ] && echo "[build_host_tools] 条件未备跳过（tools/*/main.go 零件——工具未落位）"
