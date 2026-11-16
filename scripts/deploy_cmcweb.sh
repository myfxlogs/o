#!/usr/bin/env bash
# deploy_cmcweb.sh —— o-cmcweb 镜像本机构建+船运 SG+compose up（独立于
# deploy_image.sh 主径——采集臂变更频次低，不拖慢钱径部署拍）。
# 用法: bash scripts/deploy_cmcweb.sh [sha-or-tag]
set -euo pipefail
SG="${SG_HOST:-sg}"
TAG="o-cmcweb:${1:-latest}"

die() { echo "❌ [deploy_cmcweb] $1" >&2; exit 1; }

cd "$(dirname "$0")/.."
echo "[deploy_cmcweb] 本机构建 $TAG（+latest 同层别名）"
docker build -q -f Dockerfile.cmcweb -t "$TAG" -t o-cmcweb:latest .
echo "[deploy_cmcweb] 船运→SG（gzip 流，双 ref 齐运）"
docker save "$TAG" o-cmcweb:latest | gzip -1 | ssh "$SG" 'gunzip | docker load'
ssh "$SG" "cd /opt/o && O_CMCWEB_IMAGE=$TAG docker compose -f docker-compose.yml -f docker-compose.sg.yml up -d o-cmcweb"

for i in $(seq 1 18); do
  st=$(ssh "$SG" "docker inspect o-cmcweb --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}'" 2>/dev/null || echo missing)
  if [ "$st" = healthy ] || [ "$st" = none ]; then break; fi
  [ "$i" = 18 ] && die "o-cmcweb 部署后 90s 未转 healthy（现态=$st）——docker logs o-cmcweb 查因"
  sleep 5
done
echo "✅ [deploy_cmcweb] 完工 image=$TAG health=$st"
