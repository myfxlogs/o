#!/bin/bash
# ops_relay.sh —— RSI 工作面→业主飞书摆渡（15min 汇总拍，系统频 -bot sys）。
# 用法: ops_relay.sh [digest|urgent]
#   digest（缺省）: 处置席收口+审计席产出+量表有用帧（自模账/盈利报告）,
#           一拍一条多行汇总——量表心跳帧（自态/叙事/自观分拍）滤除
#           （业主令 2026-11-15：只留有信息的，黑话转人话）;
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

# humanize 黑话→人话渲染链（业主令：没有说人话全是黑话——渲染层去黑话，
# 账本原文不动）。管喂整行 ${seat}/${event} #id ${note}。
humanize() {
	sed -E \
		-e 's/unit=o-cognition-[0-9]*([0-9]{4})\b/唤知·尾\1/g' \
		-e 's/unit=o-cognition-monologue/唤知·独白/g' \
		-e 's/unit=o-cognition-[a-zA-Z0-9_-]+/唤知/g' \
		-e 's/host=([a-zA-Z0-9_-]+)/@\1/g' \
		-e 's/rc=0 ?//g' \
		-e 's/rc=([0-9]+)/退出码\1/g' \
		-e 's/reason=session_incomplete/会话未完整/g' \
		-e 's/reason=([a-z_]+)/因=\1/g' \
		-e 's/dur=([0-9]+)s/\1秒/g' \
		-e 's/deploy=master ?/部署 /g' \
		-e 's/NAV/净值/g' \
		-e 's/用帧=无/未用帧/g' \
		-e 's/自问=无/无自问/g' \
		-e 's/第([0-9]+)轮/第 \1 轮/g' \
		-e 's/票([0-9]+)逾([0-9]+)燃([0-9]+)/\1票·逾\2·燃\3/g' \
		-e 's/会话=([0-9]+)果([0-9]+)败/会话 \1成\2败/g' \
		-e 's/求=([0-9]+)/求助\1/g' \
		-e 's/——自我模型器官帧.*//g' \
		-e 's/v=([^|]{1,12})\|([^|]+)\|([0-9]+) 盈利报告: 24h净[^ ]+ 30d累[^ ]+ 在仓([0-9]+) live\+probe([0-9]+) 年化([^ ]+)vs目标([^— ]+).*/赚钱账 24h \1｜30d \2｜开\4仓·\5器｜年化 \6(目标 \7)/g' \
		-e 's/v=([0-9a-f]{7,12})\(([0-9a-z]+)\)/版本\1·活\2/g' \
		-e 's/v=([0-9a-f]{7,12}) /版本\1 /g' \
		-e 's/——冷启动账页.*//g'
}

case "$MODE" in
digest)
	# 汇总式：wm=已处理末行（跳过件亦推进防重扫）
	last=0; buf=""
	while IFS=$'\t' read -r id seat event note; do
		last=$id; buf+="${seat}/${event} #${id} $(printf '%s' "$note" | humanize)"$'\n'
	done < <(psql "$DSN" -qAt -F$'\t' -c "
SELECT id,seat,event,left(replace(note,E'\n',' '),600) FROM ops_event
WHERE id>$(cat "$WM" 2>/dev/null || echo 0) AND (
  (seat='处置席' AND event='处置' AND note LIKE '%收口%')
  OR (seat='审计席' AND event IN ('审计','审计疑点'))
  OR (seat='探测席' AND event='量表'
      AND (note LIKE '%自模账%' OR note LIKE '%盈利报告%'))) ORDER BY id")
	[ "$last" -eq 0 ] && exit 0
	[ -z "$buf" ] && { echo "$last" > "$WM"; exit 0; }
	SEND="${FEISHU_BIN:-$HOME/.local/bin/feishusend}"
	[ -x "$SEND" ] || SEND="go run ./tools/feishusend"
	DB_DSN="$DSN" O_MASTER_KEY_FILE="$KEY" $SEND -bot sys "[RSI]
$buf" >/dev/null 2>&1 && echo "$last" > "$WM"
	;;
urgent)
	WM="${WM}_urgent"
	# 急件集:求助值守+审计P1+部署/唤死类升件（探针类即时告警
	# 仍走 o-core 告警链；条件请求推送面归 o-feishu-bridge
	# 交互卡——钮回调结案,文本急推退役）
	while IFS=$'\t' read -r id seat event note; do
		last=$id
		SEND="${FEISHU_BIN:-$HOME/.local/bin/feishusend}"
		[ -x "$SEND" ] || SEND="go run ./tools/feishusend"
		DB_DSN="$DSN" O_MASTER_KEY_FILE="$KEY" $SEND -bot sys \
			"[RSI-急] ${seat}/${event} #${id} $(printf '%s' "$note" | humanize)" \
			>/dev/null 2>&1 && echo "$id" > "$WM"
	done < <(psql "$DSN" -qAt -F$'\t' -c "
SELECT id,seat,event,left(replace(note,E'\n',' '),600) FROM ops_event
WHERE id>$(cat "$WM" 2>/dev/null || echo 0) AND (
  event='求助值守'
  OR (seat='审计席' AND event='升件' AND note LIKE '%P1%')
  OR (event='升件' AND esc_scope IN ('deploy_lag','deploy_blocked',
    'cognition_dead','cognition_spawn_fail','audit_spawn_fail','audit_stall')))
ORDER BY id")
	;;
esac
