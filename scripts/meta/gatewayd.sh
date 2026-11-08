#!/usr/bin/env bash
# gatewayd.sh —— o-gateway v2 总协调网关常驻主循环（T-0469）
#
# 状态机：IDLE → SENSE(事件采+水位线) → CLASSIFY(四域脏位) →
#   ACT(域子件 scripts/meta/gateway/*.sh) → LEDGER(ops_event
#   seat=gateway 全链行在子件内先行) → IDLE。
#
# 事件源（票面钦定面）：
#   ①ops_event 新行——PG LISTEN ev_pipeline（fin_0092 语句级触发器在案）；
#     psql 管道 stdin 非交互下异步通知随下一命令结算吐（活验在案），
#     故常驻会话挂 LISTEN+周期 SELECT 1 心跳拍——通知延迟≤心跳拍秒，
#     FIN_PG_DSN 缺席/会话断→纯轮询降级（时钟兜底不灭）。
#   ②.git/check-green-* 戳面——每迭代 stat 签名（新戳→收敛域即评）。
#   ③时钟兜底——DB_POLL_SEC 水位线重扫+各域 interval 全量拍
#     （域级拍频各自钉：席权/路由 60s、收敛 300s、升件 120s）。
#   ④systemd 席态——域子件内采（seatauth 道龄扫），非主驱。
#
# 单例：PG advisory lock 42540356（与 dm 42540354/mech 42540355/ceo
# 884201 分键）——锁随 psql 会话生死自清；锁拒=等待循环（failover
# 接管非抢占），FIN_DSN 缺席=flock 兜底单例。
# 心跳：ops_headless ticket='GATEWAY' 每 HB_SEC（gateway_hb 量表锚）；
#   ops_event '心跳' 行每 HB_EVENT_SEC（账面可见活性——量表读
#   ops_headless 不吃本行，票面「每拍落 ops_event」按 15min 节拍落防
#   账面洪泛——每 45s 一行=日增 ~1900 行不可辩）。
# 崩溃安全：域子件全幂等键在库（部署行/升件#id/票号单实例）——重启
# 重扫即续，零重复动作；账本先行由子件内 FIRE/PLAN 行承担。
# kill switch：systemctl --user stop o-gateway 即回席位自扫描兜底制。
set -u

# ── env/路径 ─────────────────────────────────────────────────
GW_ENV_FILE=${GW_ENV_FILE:-${DM_ENV_FILE:-${REPORT_ENV_FILE:-}}}
FIN_DSN=
for ef in "$GW_ENV_FILE" /opt/o-research/env /home/ubuntu/o-research/env; do
	[ -n "$ef" ] && [ -f "$ef" ] || continue
	FIN_DSN=$(grep '^FIN_PG_DSN=' "$ef" 2>/dev/null | cut -d= -f2- | tr -d '"')
	[ -n "$FIN_DSN" ] && break
done
FIN_DSN=${FIN_DSN:-}

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
GW_DIR=${O_GATEWAY_DOMAIN_DIR:-"$SCRIPT_DIR/gateway"} # 桩测覆写面（域子件目录）
REPO=${O_GATEWAY_REPO:-/opt/o}
ST=${O_GATEWAY_STATE_DIR:-/var/tmp/o-gateway}
mkdir -p "$ST" 2>/dev/null || true # tolerate: state 目录建败=mktemp 兜底
[ -d "$ST" ] || ST=$(mktemp -d "${TMPDIR:-/tmp}/o-gateway.XXXXXX")

SELECT_SEC=${O_GATEWAY_SELECT_SEC:-5}      # 主循环读超时拍
KEEPALIVE_SEC=${O_GATEWAY_KEEPALIVE_SEC:-2} # LISTEN 心跳拍（通知延迟上界）
DB_POLL_SEC=${O_GATEWAY_DB_POLL_SEC:-15}    # 水位线兜底重扫拍
HB_SEC=${O_GATEWAY_HB_SEC:-30}              # ops_headless 心跳拍
HB_EVENT_SEC=${O_GATEWAY_HB_EVENT_SEC:-900} # ops_event 心跳拍（15min）
LOCK_KEY=${O_GATEWAY_LOCK_KEY:-42540356}
LISTEN_CHAN=${O_GATEWAY_LISTEN_CHAN:-ev_pipeline}
LISTEN_EN=${O_GATEWAY_LISTEN:-auto}         # auto|0=关监听纯轮询（桩测面）

# 域表：名|全量拍秒|事件拉近最小间隔秒（CLASSIFY=脏位∪到期；env 可调=桩测面）
DOMAINS="seatauth route converge escalate"
domain_interval() { case "$1" in seatauth) echo "${O_GW_INT_SEAT:-60}";; route) echo "${O_GW_INT_ROUTE:-60}";; converge) echo "${O_GW_INT_CONVERGE:-300}";; escalate) echo "${O_GW_INT_ESC:-120}";; *) echo 300;; esac; }
domain_gap()      { case "$1" in seatauth) echo "${O_GW_GAP_SEAT:-10}";; route) echo "${O_GW_GAP_ROUTE:-10}";; converge) echo "${O_GW_GAP_CONVERGE:-45}";; escalate) echo "${O_GW_GAP_ESC:-30}";; *) echo 60;; esac; }
STAMP_GAP=${O_GATEWAY_STAMP_GAP_SEC:-15}    # 戳事件拉近收敛最小间隔
LOCK_RETRY=${O_GATEWAY_LOCK_RETRY_SEC:-30}  # 锁拒重试拍（failover 等待粒度）
case "$STAMP_GAP" in ''|*[!0-9]*) STAMP_GAP=15 ;; esac
case "$LOCK_RETRY" in ''|*[!0-9]*) LOCK_RETRY=30 ;; esac

QC() { printf '%s' "${1:-}" | sed "s/'/''/g"; }
gw_q() { psql "$FIN_DSN" -qAt -c "$1" 2>/dev/null; } # tolerate: 查询败=空答由消费面自裁
gw_now() { date +%s; }

# ── 单例+监听会话（coproc 双管道）─────────────────────────────
GW_PSQL_PID=; GW_CTL_FD=; GW_OUT_FD=; GW_FLOCK_FD=
GW_WM=0; GW_LAST_DB=0; GW_LAST_HB=0; GW_LAST_HBEV=0; GW_LAST_KA=0
GW_NOTIFY=0

session_stop() {
	[ -n "${GW_CTL_FD:-}" ] && eval "exec ${GW_CTL_FD}>&-" 2>/dev/null
	[ -n "${GW_OUT_FD:-}" ] && eval "exec ${GW_OUT_FD}<&-" 2>/dev/null
	[ -n "${GW_PSQL_PID:-}" ] && { kill "$GW_PSQL_PID" 2>/dev/null; wait "$GW_PSQL_PID" 2>/dev/null; }
	GW_PSQL_PID=; GW_CTL_FD=; GW_OUT_FD=
	return 0
}

session_start() {
	[ -n "$FIN_DSN" ] && [ "$LISTEN_EN" != "0" ] || return 1
	coproc GW_PSQL { psql "$FIN_DSN" -qAt 2>>"$ST/psql.err"; }
	GW_OUT_FD=${GW_PSQL[0]}; GW_CTL_FD=${GW_PSQL[1]}
	# 锁握手：拒=30s 重试（另一网关在飞→等待接管；会话断→上层重建）
	while :; do
		printf 'SELECT pg_try_advisory_lock(%s);\n' "$LOCK_KEY" >&"$GW_CTL_FD" || { session_stop; return 1; }
		local got
		IFS= read -r -t 15 got <&"$GW_OUT_FD" || { session_stop; return 1; }
		[ "$got" = "t" ] && break
		sleep "$LOCK_RETRY"
	done
	printf 'LISTEN %s;\n' "$LISTEN_CHAN" >&"$GW_CTL_FD" || { session_stop; return 1; }
	printf 'SELECT 1;\n' >&"$GW_CTL_FD" || { session_stop; return 1; }
	local pong
	IFS= read -r -t 15 pong <&"$GW_OUT_FD" || { session_stop; return 1; }
	[ "$pong" = "1" ] || { session_stop; return 1; }
	return 0
}

cleanup() { session_stop; [ -n "${GW_FLOCK_FD:-}" ] && eval "exec ${GW_FLOCK_FD}>&-" 2>/dev/null; exit 0; }
trap cleanup TERM INT

# ── 启动：监听优先，DSN 缺席/锁面退化→flock 兜底单例 ────────────
LISTEN_OK=0
if [ -n "$FIN_DSN" ] && [ "$LISTEN_EN" != "0" ]; then
	tries=0
	until session_start; do
		tries=$((tries+1))
		[ "$tries" -gt 3 ] && { echo "[gatewayd] 监听会话建不起——降级纯轮询" >&2; break; }
		sleep 2
	done
	[ -n "$GW_PSQL_PID" ] && LISTEN_OK=1
else
	exec {GW_FLOCK_FD}>"$ST/gatewayd.lock" 2>/dev/null || true # tolerate: 锁档开败=无锁运行降级（域件幂等键兜底——锁是优雅非正确性）
	if [ -n "$GW_FLOCK_FD" ]; then
		flock -n "$GW_FLOCK_FD" 2>/dev/null || { echo "[gatewayd] 单例锁已持——退" >&2; exit 0; }
	fi
fi

# ── 心跳 ─────────────────────────────────────────────────────
hb() { # $1=state——ops_headless 道（gateway_hb 量表锚）
	[ -n "$FIN_DSN" ] || return 0
	gw_q "INSERT INTO ops_headless(ticket,seat,state,last_activity)
	  VALUES('GATEWAY','gateway','$1',now())
	  ON CONFLICT (ticket) DO UPDATE SET seat=EXCLUDED.seat,
	    state=EXCLUDED.state,last_activity=now(),updated_at=now()" >/dev/null
	return 0
}
hb alive

hb_event() { # ops_event 心跳行（15min 节拍——账面活性面防洪泛）
	[ -n "$FIN_DSN" ] || return 0
	gw_q "INSERT INTO ops_event(seat,event,ticket,note)
	  VALUES('gateway','心跳','GATEWAY','gatewayd alive——域拍四账在巡')" >/dev/null
	return 0
}

# ── 水位线：ops_event id 单调——新行→脏位分类 ──────────────────
WM_FILE="$ST/watermark"
GW_WM=$(cat "$WM_FILE" 2>/dev/null || true) # tolerate: 水位档缺席/坏档=0 重建（启动水位取 max 不回放——域件自扫现态）
case "$GW_WM" in ''|*[!0-9]*) GW_WM=0 ;; esac
# 启动水位=当前 max（不回放历史——域子件自扫现态，水位只管唤醒延迟）
if [ -n "$FIN_DSN" ]; then
	mx=$(gw_q "SELECT COALESCE(max(id),0) FROM ops_event" | head -1)
	case "$mx" in ''|*[!0-9]*) ;; *) [ "$GW_WM" -eq 0 ] && GW_WM=$mx ;; esac
	printf '%s' "$GW_WM" >"$WM_FILE" 2>/dev/null || true # tolerate: 水位档写败=下拍重算（ST 兜底目录可写性非硬依赖）
fi

# 事件词→域脏位：生命周期族→席权/路由/收敛；升件/裁决族→升件+路由+
# 席权；gateway 自写行 seat='gateway' 全滤——自脉冲不回环。
classify_events() { # stdin=行集 id|seat|event|ticket（当前 shell——水位须回写）
	local id seat ev
	while IFS='|' read -r id seat ev _t; do
		[ -n "$id" ] || continue
		GW_WM=$id
		[ "$seat" = "gateway" ] && continue
		case "$ev" in
			升件|先例适用|部署)
				gw_dirty escalate; gw_dirty route; gw_dirty seatauth; gw_dirty converge ;;
			裁决|处置|挂起|完成|驳回)
				gw_dirty escalate; gw_dirty route; gw_dirty seatauth; gw_dirty converge ;;
			已批|派单|拉起|领单|收口|收口待审|移交|开工|外派|审计中|呈批|起草|退班|候水|等锁|异常|作废)
				gw_dirty seatauth; gw_dirty route; gw_dirty converge ;;
			心跳|北极星账|诊断草案|议题|送达|领号|升件路由|席权|路由|收敛|网关|GC)
				: ;; # 非调度词不脏
			*)
				gw_dirty seatauth; gw_dirty route; gw_dirty converge; gw_dirty escalate ;;
		esac
	done
}

gw_dirty() { printf '1' >"$ST/dom.$1.dirty" 2>/dev/null || true; return 0; } # tolerate: 脏位写败=该域退回全量拍径（漏唤醒不丢动作）
dom_dirty() { [ -s "$ST/dom.$1.dirty" ] && cat "$ST/dom.$1.dirty"; return 0; }
dom_dirty_clear() { rm -f "$ST/dom.$1.dirty" 2>/dev/null; }

drain_events() {
	[ -n "$FIN_DSN" ] || return 0
	local rows
	rows=$(gw_q "SELECT id||'|'||seat||'|'||event||'|'||ticket
	  FROM ops_event WHERE id>$GW_WM ORDER BY id LIMIT 500")
	[ -n "$rows" ] && classify_events <<< "$rows"
	printf '%s' "$GW_WM" >"$WM_FILE" 2>/dev/null || true # tolerate: 水位档写败=下拍重算（ST 兜底目录可写性非硬依赖）
	return 0
}

# ── 戳面签名（check-green-* 最新件名|mtime——变=收敛脏）──────────
GITDIR=""
stamp_sig() {
	[ -n "$GITDIR" ] || GITDIR=$(git -C "$REPO" rev-parse --git-common-dir 2>/dev/null || true) # tolerate: git-dir 不可解=戳面签名停用（收敛退回全量拍档）
	[ -n "$GITDIR" ] || return 0
	local newest
	newest=$(ls -t "$GITDIR"/check-green-* 2>/dev/null | head -1)
	[ -n "$newest" ] && stat -c '%n|%Y' "$newest" 2>/dev/null
	return 0
}

# ── 域调度：脏位∧最小间隔过｜全量拍到期 → 跑子件 ────────────────
dom_last() { local f="$ST/dom.$1.last"; [ -f "$f" ] && cat "$f" || echo 0; }
dom_run() { # $1=domain——同步跑（慢域阻塞期通知在管道积压不丢）
	local dom=$1 now last gap iv dirty due=0
	now=$(gw_now); last=$(dom_last "$dom"); last=${last:-0}
	iv=$(domain_interval "$dom"); gap=$(domain_gap "$dom")
	dirty=$(dom_dirty "$dom")
	if [ "$dom" = converge ] && [ "$dirty" = "stamp" ]; then gap=$STAMP_GAP; fi
	if [ $(( now - last )) -ge "$iv" ]; then due=1
	elif [ -n "$dirty" ] && [ $(( now - last )) -ge "$gap" ]; then due=1; fi
	[ "$due" -eq 1 ] || return 0
	printf '%s' "$now" >"$ST/dom.$1.last" 2>/dev/null || true # tolerate: 域拍档写败=下拍重估（早跑合法——域件幂等）
	dom_dirty_clear "$dom"
	bash "$GW_DIR/$dom.sh" >/dev/null 2>&1 || true # tolerate: 域子件自含降级——败不阻拍（域内自报升件）
	return 0
}

# ── 主循环 ────────────────────────────────────────────────────
GW_STAMP_LAST=$(stamp_sig)   # 启动基线=现存最新戳（历史面不脏）
while :; do
	now=$(gw_now)

	# ① 监听行到达（coproc out fd）——read -t SELECT_SEC 拍沿=主迭代
	line=""
	if [ -n "$GW_OUT_FD" ]; then
		IFS= read -r -t "$SELECT_SEC" line <&"$GW_OUT_FD" || true # tolerate: read -t 拍沿超时=主迭代节拍本身（非错误径）
		# 会话死活探：psql 已殁→重建（锁重握+LISTEN 重订+水位续位）
		if ! kill -0 "$GW_PSQL_PID" 2>/dev/null; then
			session_stop; sleep 2
			tries=0; until session_start; do tries=$((tries+1)); [ "$tries" -gt 3 ] && break; sleep 5; done
			continue
		fi
		case "$line" in
			*Asynchronous\ notification*) GW_NOTIFY=1 ;;
		esac
	else
		sleep "$SELECT_SEC"
	fi

	# ② LISTEN 心跳拍：冲刷挂起通知+会话活性（KEEPALIVE_SEC）
	if [ -n "$GW_CTL_FD" ] && [ $(( now - GW_LAST_KA )) -ge "$KEEPALIVE_SEC" ]; then
		printf 'SELECT 1;\n' >&"$GW_CTL_FD" 2>/dev/null || true # tolerate: keepalive 写败=会话死探下拍接管重建
		GW_LAST_KA=$now
	fi

	# ③ 水位线兜底重扫（通知驱动之外的漏缝——DB_POLL_SEC）
	if [ -n "$FIN_DSN" ] && { [ "$GW_NOTIFY" -eq 1 ] || [ $(( now - GW_LAST_DB )) -ge "$DB_POLL_SEC" ]; }; then
		GW_NOTIFY=0; GW_LAST_DB=$now; drain_events
	fi

	# ④ 戳面签名（每迭代 stat——新/变戳→收敛脏 'stamp' 专用拉近档）
	sig=$(stamp_sig)
	if [ -n "$sig" ] && [ "$sig" != "$GW_STAMP_LAST" ]; then
		GW_STAMP_LAST=$sig
		printf 'stamp' >"$ST/dom.converge.dirty" 2>/dev/null || true # tolerate: 戳脏位写败=收敛退回全量拍档（戳签名下拍重比对）
	fi

	# ⑤ 域调度
	for d in $DOMAINS; do dom_run "$d"; done

	# ⑥ 心跳
	if [ $(( now - GW_LAST_HB )) -ge "$HB_SEC" ]; then GW_LAST_HB=$now; hb alive; fi
	if [ $(( now - GW_LAST_HBEV )) -ge "$HB_EVENT_SEC" ]; then GW_LAST_HBEV=$now; hb_event; fi
done
