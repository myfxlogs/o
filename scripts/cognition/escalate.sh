#!/usr/bin/env bash
# escalate.sh —— 纪元3容错自报件（ops_esc.sh 继任——scripts/meta 随体裁撤,
# 探测层升件写径归唤知域）。同签名同语义：ops_event '升件' 行+
# note 前缀 24h 同窗去重（去重查询自身败=空答照常升,报侧 fail-open 不吞讯）;
# 末环限度:写径败静默退(升件不能再升件——结构性限度在案)。
# 用法: escalate.sh <seat> <ticket> <class> <scope> <prefix> <detail>
#   seat=写席位(纪元3探测件建议'探测席');ticket=归属对象(无传'');
#   class/scope=判例匹配键(''=无类升件);prefix=去重键兼note头;
#   detail=note尾段(前缀与分隔符本件统一加)。
set -u
SEAT=${1:?usage: escalate.sh <seat> <ticket> <class> <scope> <prefix> <detail>}
TICKET=${2:-}; CLS=${3:-}; SCP=${4:-}; PREFIX=${5:?missing prefix}; DETAIL=${6:-}

FIN_DSN=
for ef in "${OPS_ESC_ENV_FILE:-}" "${REPORT_ENV_FILE:-}" \
          /opt/o-research/env /home/ubuntu/o-research/env; do
	[ -n "$ef" ] && [ -f "$ef" ] || continue
	FIN_DSN=$(grep '^FIN_PG_DSN=' "$ef" 2>/dev/null | cut -d= -f2- | tr -d '"')
	[ -n "$FIN_DSN" ] && break
done
[ -n "$FIN_DSN" ] || exit 0

QC(){ printf '%s' "${1:-}" | sed "s/'/''/g"; }

# 同窗去重：note 前缀 24h 内已有升件行=本窗已报（同根因多面齐发只报一行）。
psql "$FIN_DSN" -qAt -c "SELECT 1 FROM ops_event WHERE event='升件'
  AND note LIKE '$(QC "$PREFIX")%' AND ts>now()-interval '24 hours'
  LIMIT 1" 2>/dev/null | grep -qx 1 && exit 0

NOTE="$(QC "$PREFIX")——$(QC "$DETAIL")"
TQC="$(QC "$TICKET")"; SQC="$(QC "$SEAT")"
CCLS=NULL; CSCP=NULL
[ -n "$CLS" ] && CCLS="'$(QC "$CLS")'"
[ -n "$SCP" ] && CSCP="'$(QC "$SCP")'"
psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
  VALUES('$SQC','升件','$TQC','$NOTE',$CCLS,$CSCP)" >/dev/null 2>&1 \
	|| psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
  VALUES('$SQC','升件','$TQC','$NOTE')" >/dev/null 2>&1 || true # tolerate: 自报末环——INSERT 败静默退（结构性限度在案）
exit 0
