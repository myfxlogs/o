#!/usr/bin/env bash
# manual_lane_check.sh <T-NNNN> [task_dir] —— 人工径票判别（T-0438 件⑥；
# 坑条目 real-money-ticket-machine-dispatch-waste 律①②：真钱 venue 写票
# （下单/平仓/卖袋/转账/签名类）票面出生行必标「执行=人工径（业主在场）」
# ——状态位即明示；派单链（dm_depgate 候选集/dm_tick 补拉窗）按此滤除
# ——人工径票不派不挂起不计漏拉，补派件同域踢出候选）。
# 判据=票面行解析式（票面钦定二择其一，本件写死「票面行解析」非票态位）：
#   票件命中钉死字面「执行=人工径」=人工径票（律①出生行钦定格式字面）。
# task_dir 位置参优先（消费面自持票目录单源传入——桩测面零部署树耦合），
# 缺席回落 O_MANUAL_LANE_TASK_DIR，再回落 /opt/o/docs/tasks 产径。
# rc: 0=人工径票（消费面滤除）；1=可派；2=票件缺/不可读（消费面自处）；3=用法错。
set -u
[ $# -ge 1 ] && [ -n "$1" ] || { echo "usage: manual_lane_check.sh <T-NNNN> [task_dir]" >&2; exit 3; }
TID=$1
TASK_DIR=${2:-${O_MANUAL_LANE_TASK_DIR:-/opt/o/docs/tasks}}
FILE=$(compgen -G "$TASK_DIR/$TID-*.md" | head -1)
[ -n "$FILE" ] && [ -r "$FILE" ] || exit 2
if grep -qF '执行=人工径' "$FILE"; then
  exit 0
fi
exit 1
