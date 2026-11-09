# gateway/common.sh —— o-gateway 四域动作子件共享库（T-0469）
# 只做机械账面：env/DSN/SQL/ledger/state——判据件调用各归其主，本库零判据。
# 词表纪律：gateway 落账 event 限机械词（席权/路由/收敛/升件路由/心跳/
# 网关）——生命周期词与「升件」本体永不由本库直写（升件走 ops_esc/
# escalate.sh 带类径，语义归属席不归网关）。
set -u

GW_SELF_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
GW_META_DIR=$(cd -- "$GW_SELF_DIR/.." && pwd)
GW_REPO=${O_GATEWAY_REPO:-$(cd -- "$GW_META_DIR/../.." && pwd)}
GW_STATE_DIR=${O_GATEWAY_STATE_DIR:-/var/tmp/o-gateway}

# env 解析链同 ops_esc/mech_tick：GW_ENV_FILE→DM_ENV_FILE→REPORT_ENV_FILE
# →/opt/o-research/env→/home/ubuntu/o-research/env（先中先用）。
gw_env_file() {
	local ef
	for ef in "${GW_ENV_FILE:-}" "${DM_ENV_FILE:-}" "${REPORT_ENV_FILE:-}" \
		/opt/o-research/env /home/ubuntu/o-research/env; do
		[ -n "$ef" ] && [ -f "$ef" ] && { printf '%s\n' "$ef"; return 0; }
	done
	return 1
}

gw_dsn() {
	local ef; ef=$(gw_env_file 2>/dev/null) || return 0
	grep '^FIN_PG_DSN=' "$ef" 2>/dev/null | cut -d= -f2- | tr -d '"'
}

gw_qc() { printf '%s' "${1:-}" | sed "s/'/''/g"; }

# gw_sql <sql> —— -qAt 行集；DSN 缺席/查询败=空输出 rc0（降级不炸——
# 各消费面自带 fail-closed 判据层，读面不可知按面自裁）。
gw_sql() {
	local dsn; dsn=$(gw_dsn); [ -n "$dsn" ] || return 0
	psql "$dsn" -qAt -c "$1" 2>/dev/null
}

# gw_ledger <event> <ticket> <note> —— 账本先行行：seat=gateway 机械词表。
# 写败容错退但留样（write_fail.log——哑写死径不可再静默：裸值语法错
# 四域零落实证 KB heredoc-sql-unquoted-silent-swallow；账面补充非链环，
# 动作判据仍不依赖本行存在）。
gw_ledger() {
	local dsn; dsn=$(gw_dsn); [ -n "$dsn" ] || return 0
	local out
	out=$(psql "$dsn" -qAt 2>&1 <<SQL
INSERT INTO ops_event(seat,event,ticket,note)
VALUES('gateway','$(gw_qc "$1")','$(gw_qc "$2")','$(gw_qc "$3")');
SQL
	) || printf '%s gw_ledger写败 %s/%s — %s\n' "$(date -u +%FT%TZ)" "$1" "$2" \
		"$out" >> "${GW_STATE_DIR:-/var/tmp/o-gateway}/write_fail.log" 2>/dev/null || true
	return 0
}

# gw_esc <ticket> <prefix> <detail> —— 网关自报升件径（ops_esc 同件
# seat=gateway——24h 前缀去重自带；末环写败静默退结构性限度在案）。
gw_esc() {
	"$GW_META_DIR/ops_esc.sh" gateway "${1:-}" "" "" "${2:?prefix}" "${3:-}" \
		|| true # tolerate: 自报末环——ops_esc 写径败静默退（结构性限度在案）
}

gw_state_file() { printf '%s/%s' "$GW_STATE_DIR" "$1"; }

gw_state_get() { local f; f=$(gw_state_file "$1")
	[ -f "$f" ] && cat "$f" 2>/dev/null; return 0; }

gw_state_set() { local f; f=$(gw_state_file "$1")
	mkdir -p "$GW_STATE_DIR" 2>/dev/null
	printf '%s' "$2" >"$f" 2>/dev/null || true # tolerate: 状态件写败=下拍重写（幂等键在库不在件）
	return 0; }

gw_now() { date +%s; }
