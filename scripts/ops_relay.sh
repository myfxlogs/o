#!/bin/bash
# ops_relay.sh —— RSI 工作面→业主飞书摆渡（15min 汇总拍）。
# 用法: ops_relay.sh [digest|urgent]
#   digest（缺省）: 处置席收口+审计席产出新行,一拍一条多行汇总。
#   urgent: 急件道——审计P1升件+部署/唤死类升件,逐条即发
#           （1min 拍驱动;与 digest 分道双水印互不重复）。
# 水印: ~/.local/state/o-worker/relay_last_id[_urgent]——
#   发送成→进（含跳过件:已处理不重扫）;发送败→留（下拍重试不丢）。
set -u
MODE="${1:-digest}"
DSN=$(grep '^FIN_PG_DSN=' /opt/o-research/env | cut -d= -f2- | tr -d '"')
KEY="/opt/o/secrets/o_master_key"
WM=~/.local/state/o-worker/relay_last_id
cd /opt/o || exit 0
case "$MODE" in
digest)
	# 汇总式：wm=已处理末行（跳过件亦推进防重扫）
	last=0; buf=""
	while IFS=$'\t' read -r id seat event note; do
		last=$id; buf+="${seat}/${event} #${id} ${note}"$'\n'
	done < <(psql "$DSN" -qAt -F$'\t' -c "
SELECT id,seat,event,left(replace(note,E'\n',' '),600) FROM ops_event
WHERE id>$(cat "$WM" 2>/dev/null || echo 0) AND (
  (seat='处置席' AND event='处置' AND note LIKE '%收口%')
  OR (seat='审计席' AND event IN ('审计','审计疑点'))) ORDER BY id")
	[ "$last" -eq 0 ] && exit 0
	[ -z "$buf" ] && { echo "$last" > "$WM"; exit 0; }
	SEND="${FEISHU_BIN:-$HOME/.local/bin/feishusend}"
	[ -x "$SEND" ] || SEND="go run ./tools/feishusend"
	DB_DSN="$DSN" O_MASTER_KEY_FILE="$KEY" $SEND "[RSI]
$buf" >/dev/null 2>&1 && echo "$last" > "$WM"
	;;
urgent)
	WM="${WM}_urgent"
	# 急件集:审计P1+部署类+唤死类升件（探针类即时告警仍走
	# o-core 告警链,此道只渡会话/审计产出与环死信号）
	while IFS=$'\t' read -r id seat event note; do
		last=$id
		SEND="${FEISHU_BIN:-$HOME/.local/bin/feishusend}"
		[ -x "$SEND" ] || SEND="go run ./tools/feishusend"
		DB_DSN="$DSN" O_MASTER_KEY_FILE="$KEY" $SEND "[RSI-急] ${seat}/${event} #${id} ${note}" \
			>/dev/null 2>&1 && echo "$id" > "$WM"
	done < <(psql "$DSN" -qAt -F$'\t' -c "
SELECT id,seat,event,left(replace(note,E'\n',' '),600) FROM ops_event
WHERE id>$(cat "$WM" 2>/dev/null || echo 0) AND (
  (seat='审计席' AND event='升件' AND note LIKE '%P1%')
  OR (event='升件' AND esc_scope IN ('deploy_lag','deploy_blocked',
    'cognition_dead','cognition_spawn_fail','audit_spawn_fail','audit_stall')))
ORDER BY id")
	;;
esac
