#!/usr/bin/env bash
# fin_ro.sh —— 临时/探测/排查类 SQL 只读径（工单#25758 adhoc-query-mirror-route）。
#
# 默认：OCI PG-md 镜像读面（SSH + docker exec o-postgres-md psql）——分析查询
#       不压 SG 钱径主库。--sg：显式走 SG 主库窄读角色 FIN_PG_RO_DSN，仅用于
#       镜像未桥接表（ops_event/fin_cashflow/...）或钱径取证。
#
# 安全：客户端预检拒绝写/事务控制语句；会话 SET default_transaction_read_only=on；
#       statement_timeout 默认 30s；SG 径角色本身 SELECT-only。
#
# 用法：
#   scripts/fin_ro.sh -c "SELECT ..."
#   scripts/fin_ro.sh -f query.sql
#   cat query.sql | scripts/fin_ro.sh
#   scripts/fin_ro.sh --sg -c "SELECT ... FROM ops_event ..."
#   scripts/fin_ro.sh --tables     # 目标面表册
#   scripts/fin_ro.sh --check      # 连通性
#
# 选项：-t/--tuples（-tA）、--csv、--timeout <秒>、-q/--quiet、-h/--help
#
# 环境（可选覆写）：REPORT_ENV_FILE、O_FIN_RO_MIRROR_HOST、O_FIN_RO_MIRROR_CT、
#   O_FIN_RO_TIMEOUT；SSH 路由复用 scripts/oci_ssh_route.sh（O_OCI_HOST/O_OCI_JARG）。
# 注：本件是查询纪律件，不是安全边界——镜像面是副本，仍禁写；SG 径走窄读角色。
set -euo pipefail

SELF_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ENVF=${REPORT_ENV_FILE:-/opt/o-research/env}
MIRROR_HOST=${O_FIN_RO_MIRROR_HOST:-${O_OCI_HOST:-ubuntu@149.118.58.98}}
MIRROR_CT=${O_FIN_RO_MIRROR_CT:-o-postgres-md}
TIMEOUT=${O_FIN_RO_TIMEOUT:-30}

TARGET=mirror
ACTION=query
HAVE_SQL=0
SQL=""
OUT="-P pager=off"
OUT_SET=0
QUIET=0

# 镜像面未桥接表（2026-11-20 实测 information_schema 对拍：SG 39 表 / OCI 37 表；
# 以下=SG 有 OCI 无）。新增桥接表时同改本册与 docs/runbooks/adhoc-query-mirror-route.md。
SG_ONLY="app_secrets
fin_cashflow
fin_condition_gap
fin_econ_lesson
fin_econ_rule
fin_employee_registry
fin_eng_lesson
fin_eval_reject
fin_exec_drill
fin_execution
fin_guard_close_status
fin_host_metrics
fin_known_residual
fin_risk_decision
fin_ticket_draft
md_equity_curve
ops_chain_residual
ops_decision
ops_esc_pending
ops_esc_valence
ops_event
ops_event_act
ops_event_sem_matrix
ops_face_snapshot
ops_headless
ops_precedent
ops_seat_card
ops_seat_latest
ops_state
ops_state_row
ops_ticket_desk
ops_ticket_phase
ops_ticket_state
ops_ticket_usage"

die() { echo "❌ fin_ro: $*" >&2; exit 2; }

usage() {
	awk 'NR>1 { if ($0 ~ /^#/) { sub(/^# ?/, ""); print; next } exit }' "${BASH_SOURCE[0]}"
}

append_sql() {
	[ -n "$1" ] || return 0
	if [ "$HAVE_SQL" = 0 ]; then
		SQL=$1
		HAVE_SQL=1
	else
		SQL="$SQL"$'\n'"$1"
	fi
}

while [ $# -gt 0 ]; do
	case "$1" in
		-c|--command)
			[ $# -ge 2 ] || die "-c 缺 SQL 参数"
			append_sql "$2"
			shift 2
			;;
		-f|--file)
			[ $# -ge 2 ] || die "-f 缺文件参数"
			if [ "$2" = "-" ]; then
				append_sql "$(cat)"
			else
				[ -r "$2" ] || die "文件不可读: $2"
				append_sql "$(cat "$2")"
			fi
			shift 2
			;;
		--sg) TARGET=sg; shift ;;
		--mirror) TARGET=mirror; shift ;;
		-t|--tuples) OUT="-tA"; OUT_SET=1; shift ;;
		--csv) OUT="--csv"; OUT_SET=1; shift ;;
		--timeout)
			[ $# -ge 2 ] || die "--timeout 缺秒数"
			case "$2" in ''|*[!0-9]*) die "--timeout 须整数秒: $2" ;; esac
			[ "$2" -ge 1 ] || die "--timeout 须 ≥1: $2"
			TIMEOUT=$2
			shift 2
			;;
		--tables) ACTION=tables; shift ;;
		--check) ACTION=check; shift ;;
		-q|--quiet) QUIET=1; shift ;;
		-h|--help) usage; exit 0 ;;
		--) shift; break ;;
		-*) die "未知选项: $1（-h 看用法）" ;;
		*) die "未知参数: $1（SQL 用 -c/-f/stdin）" ;;
	esac
done

for v in "$MIRROR_HOST" "$MIRROR_CT"; do
	case "$v" in ''|*[!A-Za-z0-9_.@:-]*) die "镜像路由参数字符非法: $v" ;; esac
done

if [ "$ACTION" = tables ] && [ "$OUT_SET" = 0 ]; then
	OUT="-tA"
fi

if [ "$ACTION" = query ] && [ "$HAVE_SQL" = 0 ] && [ ! -t 0 ]; then
	append_sql "$(cat)"
fi
if [ "$ACTION" = query ] && [ "$HAVE_SQL" = 0 ]; then
	usage >&2
	die "缺 SQL——用 -c/-f 或 stdin"
fi

# 客户端只读预检：剥注释/单引号字符串后，拒写语句与事务控制/危险函数。
# 镜像面虽为副本，写坏反桥仍会污染研究面；SG 径角色本身 SELECT-only，双保险。
if [ "$ACTION" = query ]; then
	CLEAN=$(printf '%s\n' "$SQL" | sed -E 's/--.*$//' | sed -E "s/'((''|[^'])*)'/''/g")
	if printf '%s\n' "$CLEAN" | grep -Eiq '\b(insert|update|delete|merge|create|alter|drop|truncate|grant|revoke|copy|call|do|vacuum|reindex|cluster|refresh|comment|security[[:space:]]+label|listen|notify|unlisten|lock|set|reset|set_config|begin|commit|rollback|savepoint|release|into|pg_terminate_backend|pg_cancel_backend|pg_reload_conf|pg_rotate_logfile|lo_import|lo_export|dblink)\b'; then
		die "只读径拒写/事务控制/危险函数——本件只收 SELECT/WITH/EXPLAIN/SHOW/VALUES/TABLE"
	fi
fi

# 镜像面预检：查询引用 SG-only 表时 fail-closed，给 --sg 明示径（不静默回落）。
if [ "$ACTION" = query ] && [ "$TARGET" = mirror ]; then
	REFS=$(printf '%s\n' "$CLEAN" | grep -Eio '\b(from|join)[[:space:]]+(public\.)?[a-z_][a-z0-9_]*' \
		| awk '{print tolower($2)}' | sed 's/^public\.//' | sort -u || printf '')
	for t in $REFS; do
		if [ "$t" = app_secrets ]; then
			die "app_secrets 结构性禁读（凭据域）——任何查询径都不收"
		fi
		if printf '%s\n' "$SG_ONLY" | grep -qx "$t"; then
			echo "❌ fin_ro: 查询引用镜像面未桥接表: $t" >&2
			echo "   镜像面表册: scripts/fin_ro.sh --tables" >&2
			echo "   SG-only 表须显式 --sg（FIN_PG_RO_DSN 白名单；未授表=permission denied；请限时窗/LIMIT）" >&2
			exit 3
		fi
	done
fi

PRELUDE="SET default_transaction_read_only = on;
SET statement_timeout = '${TIMEOUT}s';
SET lock_timeout = '5s';
SET idle_in_transaction_session_timeout = '60s';"

run_mirror() {
	local sql=$1
	if [ "$QUIET" = 0 ]; then
		echo "[fin_ro] OCI mirror: ${MIRROR_HOST} ${MIRROR_CT}（read-only, timeout=${TIMEOUT}s）" >&2
	fi
	# shellcheck source=scripts/oci_ssh_route.sh
	source "$SELF_DIR/oci_ssh_route.sh" "$MIRROR_HOST" >/dev/null 2>&1
	# shellcheck disable=SC2086 # O_OCI_JARG 是多段 ssh 选项串（先例同 check_bridge_registry.sh）
	printf '%s\n' "$PRELUDE" "$sql" \
		| timeout $((TIMEOUT + 15))s ssh -o BatchMode=yes -o ConnectTimeout=10 $O_OCI_JARG "$MIRROR_HOST" \
			"sudo docker exec -i $MIRROR_CT psql -U o -d o -X -q -v ON_ERROR_STOP=1 $OUT"
}

run_sg() {
	local sql=$1 ro
	ro=$(grep '^FIN_PG_RO_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"' || printf '')
	[ -n "$ro" ] || die "FIN_PG_RO_DSN 缺（$ENVF）——--sg fail-closed，不自造凭据"
	if [ "$QUIET" = 0 ]; then
		echo "[fin_ro] SG read-only: FIN_PG_RO_DSN（o_fin_ro 白名单, timeout=${TIMEOUT}s）" >&2
	fi
	# shellcheck disable=SC2086 # OUT 是固定旗串（-P pager=off/-tA/--csv），非用户输入
	printf '%s\n' "$PRELUDE" "$sql" \
		| timeout $((TIMEOUT + 15))s psql "$ro" -X -q -v ON_ERROR_STOP=1 $OUT
}

run_target() {
	if [ "$TARGET" = sg ]; then run_sg "$1"; else run_mirror "$1"; fi
}

case "$ACTION" in
	tables)
		run_target "SELECT table_name FROM information_schema.tables
			WHERE table_schema='public' ORDER BY 1;"
		;;
	check)
		run_target "SELECT current_database() AS db, current_user AS usr,
			now() AS now,
			(SELECT count(*) FROM information_schema.tables
			 WHERE table_schema='public') AS tables;"
		;;
	query)
		run_target "$SQL"
		;;
	*)
		die "内部动作未知: $ACTION"
		;;
esac
