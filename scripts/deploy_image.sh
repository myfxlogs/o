#!/usr/bin/env bash
# deploy_image.sh — 本机构建镜像船运 SG（业主令 2026-10-24：SG 5Mbps 构建链慢，
# 镜像传输取代远端编译）。流：本机 build → save|gzip → SG load → compose up（无 --build）。
set -euo pipefail
SG="${PHASE_SG_HOST:-root@192.229.85.35}"
IMG=phase-core:latest
cd "$(dirname "$0")/.."
echo "[deploy_image] 本机构建 $IMG"
docker build -q -f Dockerfile.core -t "$IMG" .
echo "[deploy_image] 船运→SG（gzip 流）"
docker save "$IMG" | gzip -1 | ssh "$SG" 'gunzip | docker load'
ssh "$SG" "cd /opt/phase && docker compose -f docker-compose.yml -f docker-compose.sg.yml up -d phase-core"
echo "[deploy_image] done→$SG"
