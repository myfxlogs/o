#!/usr/bin/env bash
# dispatch_water.sh —— 派单候水标记行唯一 emit 口径（T-0463 CTO 裁定①）：
# ops_event 新事件词「候水」=spawn rc75（水位/额度满）账态扶正——断链≠排队，
# 派单行落账但工人未拉起的合法候队在账面分家（业主令「改个状态就行了，
# 而不是去改序」——派单/spawn 事件序不动）。
# 写点=dm_depgate 即派 rc75 分支+dm_tick 补拉窗 rc75 分支共用本件
# （单源防漂——两写点同一 INSERT 词形/幂等闸/seat 口径）。
# 幂等闸：票内最新事件已=候水→不重复落行——候水行龄=连续候水时长，
# 老龄量表 dispatch_aged_water（gauges.tsv）同口径读面，补拉窗每拍
# rc75 重试不刷屏。任何后续事件（领单/拉起/裁决/完成…）落行即脱态，
# 下次 rc75 重新落标记=新一轮候水账。
# 写败静默退（tolerate 语义内建——标记行是账面补充非链环：候派事实已由
# 派单行在案，哨兵/量表读面缺席按未候处理不炸；下拍 rc75 自然重试再落）。
# 用法: dispatch_water.sh <T-NNNN>
# 环境覆写（测试/对拍面）：DM_ENV_FILE（缺省 /opt/o-research/env 同拍壳）。
set -u
TID=${1:-}
case "$TID" in
	''|*[!A-Za-z0-9_-]*) echo "usage: dispatch_water.sh <ticket>" >&2; exit 2 ;;
esac
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$FIN_DSN" ] || exit 0
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
KEY_GATE=${O_KEY_GATE_SH:-"$SCRIPT_DIR/ops_event_key_gate.sh"} # T-0499 键卫生闸（缺席 fail-open）

# T-0499 键卫生闸：候水行 ticket 落库前拒伪键（字符集校验下恒过=
# 双保险；闸缺席 fail-open）。
[ ! -x "$KEY_GATE" ] || "$KEY_GATE" "$TID" "event=候水" || exit 0

# 钦定词形（T-0463 裁定①）：seat=写席（壳轨=CTO 同派单/拉起行口径），
# event=候水，note 字面一字不动——读面（handoff_pairs 接行集/gauges
# 老龄量表/ops_ticket_state 账态）全按此词锚。
psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
SELECT 'CTO','候水','$TID','候水——spawn rc75 水位/额度满，候选留队下拍重试（非断链；额度恢复即拉）'
WHERE (SELECT e.event FROM ops_event e WHERE e.ticket='$TID'
	ORDER BY e.ts DESC,e.id DESC LIMIT 1) IS DISTINCT FROM '候水'" \
	|| true # tolerate: 候水标记行尽力而为——账面补充非链环，写径败下拍 rc75 自然重落
exit 0
