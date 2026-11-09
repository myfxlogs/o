#!/usr/bin/env bash
# image_tag_gc.sh —— T-0466 件②：o-core 镜像 tag 保留窗。
# 「一 commit 一 tag 永不删」=镜像堆单调增长病灶（66 tag≈5G 实证）；
# 钦定：留近 N=5 个 sha tag+latest——N=5 覆盖近窗部署史保回滚径，
# 删 tag 不动在用镜像（docker rmi 只摘 tag；镜像被 latest/容器引用
# 不卸层，dangling 层由随后的 image prune -f 收）。
#
# 用法：scripts/image_tag_gc.sh——deploy_image.sh 尾段调用（纪元3自 meta/ 移入部署管道域）；
# 亦可独立跑（幂等）。环境覆写（桩测）：O_IMAGE_KEEP（默认5）、
# O_IMAGE_REPO（默认 o-core）、O_IMAGE_LIST_CMD/O_IMAGE_RMI_CMD
# （桩测注入面——缺省走真 docker）。
set -uo pipefail
REPO=${O_IMAGE_REPO:-o-core}
KEEP=${O_IMAGE_KEEP:-5}
LIST=${O_IMAGE_LIST_CMD:-docker images}
RMI=${O_IMAGE_RMI_CMD:-docker rmi}

# CreatedAt|Tag 行集——CreatedAt 词典序=时序（同 TZ 定宽串）；
# latest/<none> 排除（latest 钦定永留；none 非 tag）。
mapfile -t ROWS < <($LIST "$REPO" --format '{{.CreatedAt}}|{{.Tag}}' 2>/dev/null \
  | grep -v '|latest$' | grep -v '|<none>' | sort -r) # tolerate: docker 败/空集=ROWS 空——零删放行（保洁件不fail-closed炸部署尾）
TOTAL=${#ROWS[@]}
if [ "$TOTAL" -le "$KEEP" ]; then
  echo "[image_tag_gc] $REPO tag 数=$TOTAL ≤ 保留窗 $KEEP——零剪"
  exit 0
fi
# sort -r 最新在前——tail +KEEP+1 起=超窗旧 tag
mapfile -t OLDTAGS < <(printf '%s\n' "${ROWS[@]}" | tail -n "+$((KEEP+1))" | cut -d'|' -f2)
pruned=0; failed=0
for t in "${OLDTAGS[@]}"; do
  [ -n "$t" ] || continue
  if $RMI "$REPO:$t" >/dev/null 2>&1; then
    pruned=$((pruned+1)); echo "[image_tag_gc] 剪 $REPO:$t（超窗）"
  else
    failed=$((failed+1)); echo "⚠️ [image_tag_gc] 剪败 $REPO:$t" >&2
  fi
done
# dangling 层收口——只收无 tag 引用层（在用镜像/容器零触）
eval "${O_IMAGE_PRUNE_CMD:-docker image prune -f}" >/dev/null 2>&1 \
  || echo "⚠️ [image_tag_gc] dangling prune 败（不阻断）" >&2 # tolerate: prune 败=层残留下轮再收——保洁尽力件
echo "[image_tag_gc] done pruned=$pruned failed=$failed kept=$KEEP+latest（总 $TOTAL→$((${TOTAL}-pruned))）"
exit 0
