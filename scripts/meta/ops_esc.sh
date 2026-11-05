#!/usr/bin/env bash
# ops_esc.sh —— 容错自报共享件（T-0372：T-0369 usage_esc 同式提共享——
# dm_tick/dev_run/deploy_sync 同式消费面复用，禁新通知栈）。
# ops_event '升件' 行+note 前缀 24h 同窗去重（去重查询自身败=空答→
# 照常升，报侧 fail-open 不吞讯）；escalate.sh 在则走带类升件径，
# 不在（OCI 单件部署面无 scripts/meta 树）直落同构 INSERT。
# 末环限度：本件写径败=静默退（升件不能再升件——T-0369 结构性限度
# 在案，对拍面兜底）。
# 用法: ops_esc.sh <seat> <ticket> <class> <scope> <prefix> <detail>
#   seat=写席位；ticket=归属票（无归属传 ''）；class/scope=判例匹配键
#   （传 ''=无类升件永不入判例面照常上行）；prefix=去重键兼 note 头
#   （同前缀 24h 至多一行）；detail=note 尾段（前缀与分隔符本件统一加）。
set -u
SEAT=${1:?usage: ops_esc.sh <seat> <ticket> <class> <scope> <prefix> <detail>}
TICKET=${2:-}; CLS=${3:-}; SCP=${4:-}; PREFIX=${5:?missing prefix}; DETAIL=${6:-}

FIN_DSN=; WIN_EF=
for ef in "${OPS_ESC_ENV_FILE:-}" "${DM_ENV_FILE:-}" "${REPORT_ENV_FILE:-}" \
          /opt/o-research/env /home/ubuntu/o-research/env; do
	[ -n "$ef" ] && [ -f "$ef" ] || continue
	FIN_DSN=$(grep '^FIN_PG_DSN=' "$ef" 2>/dev/null | cut -d= -f2- | tr -d '"')
	[ -n "$FIN_DSN" ] && { WIN_EF=$ef; break; }
done
[ -n "$FIN_DSN" ] || exit 0

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
QC(){ printf '%s' "${1:-}" | sed "s/'/''/g"; }

# 同窗去重：note 前缀 24h 内已有升件行=本窗已报（跨消费面同键共享——
# 同根因多面齐发只报一行防事件暴）。
psql "$FIN_DSN" -qAt -c "SELECT 1 FROM ops_event WHERE event='升件'
  AND note LIKE '$(QC "$PREFIX")%' AND ts>now()-interval '24 hours'
  LIMIT 1" 2>/dev/null | grep -qx 1 && exit 0

if [ -x "$SCRIPT_DIR/escalate.sh" ]; then
	# escalate.sh 自走 ESCALATE_ENV_FILE 径——与本件解析同源传递（解析到
	# 哪份 env，带类 INSERT 就落哪份）。
	[ -n "$WIN_EF" ] && export ESCALATE_ENV_FILE="$WIN_EF"
	"$SCRIPT_DIR/escalate.sh" "$SEAT" "$TICKET" "$CLS" "$SCP" "$PREFIX——$DETAIL" || true # tolerate: 自报末环——escalate 写径败静默退（结构性限度在案）
	exit 0
fi

# 无 escalate.sh 径（OCI 单件部署面）：直落同构 INSERT（esc 列缺席窗
# 回落无类行——escalate.sh 同款容错）。
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
