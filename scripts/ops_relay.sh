#!/usr/bin/env bash
# ops_relay.sh —— RSI 工作面飞书摆渡（业主令 2026-11-10「机器干活要公示——
# 蠢活=能力边界测绘,好调模型」：ops_event 收口/审计行按水Mark捞新→
# feishusend 直推。只报「席位产出」级——探针原始升件/心跳/锁事件不渡,
# 噪声不洪业主屏）。
set -uo pipefail
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" | cut -d= -f2- | tr -d '"')
KEY="${O_MASTER_KEY_FILE:-/opt/o/secrets/o_master_key}"
STATE="${XDG_STATE_HOME:-$HOME/.local/state}/o-worker"
mkdir -p "$STATE"; WM="$STATE/relay_last_id"
[ -s "$WM" ] || psql "$DSN" -qAt -c "SELECT COALESCE(max(id),0) FROM ops_event" > "$WM"
last=$(cat "$WM")
psql "$DSN" -qAt -F$'\t' -c "
SELECT id,seat,event,left(replace(note,E'\n',' '),600) FROM ops_event
WHERE id>$last AND (
  (seat='处置席' AND event='处置' AND note LIKE '%收口%')
  OR (seat='审计席' AND event IN ('审计','审计疑点','升件'))) ORDER BY id" |
buf=""
last=0
while IFS=$'\t' read -r id seat event note; do
	last=$id
	[ "$event" = "升件" ] && [[ "$note" != *P1* ]] && continue
	buf+="${seat}/${event} #${id} ${note}"$'\n'
done
[ "$last" -eq 0 ] && exit 0
[ -z "$buf" ] && { echo "$last" > "$WM"; exit 0; }
SEND="${FEISHU_BIN:-$HOME/.local/bin/feishusend}"
[ -x "$SEND" ] || SEND="go run ./tools/feishusend"
DB_DSN="$DSN" O_MASTER_KEY_FILE="$KEY" $SEND "[RSI]
$buf" >/dev/null 2>&1 && echo "$last" > "$WM"
