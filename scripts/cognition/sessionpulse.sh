#!/usr/bin/env bash
# sessionpulse.sh —— 会话在修龄阈监控（工单O-32268②）：
# 「唤起行龄>STALL_MIN ∧ 同 unit 无后随配对行（会话收口∨唤起果∨
# 唤起败——三态=生灭已决）」→ escalate.sh 升件——大件在会话内做=
# 一拍独占+半成风险，「什么算大件」沉为判据；监控必须盒内薄壳
# （禁会话自我报告替代机械监控——spec 禁线）。
# 唤起行有配对行三态语义（run.sh EXIT trap 果/败+会话收口）：
# 唯三无配对情形=①仍在飞（合法长会话——RuntimeMaxSec 2h 窗内
# >90min 即大件判据面，处置席核 ops_headless 活性定去留）
# ②SIGKILL/宿死黑洞（trap 写不了行——真 stall）。两者皆须升件
# 可见，判词归处置席非本件。
# 落点=dispatch.sh 拍前预扫（同 mine/digestspawn——15min 拍距同宿点
# =细胞件不立 resident_units）；在办脉冲在 run.sh 壳侧（①件）——
# 在办行=事实账非信号不进 ops_esc_pending（不吵面）。
# 升件面：class='会话'（新类——会话生灭域），scope/prefix=
# session_stall_<unit>——24h 同前缀去重按 unit 分账不互抑，
# 处置席见件转票/核活性销账。
# 阈值写死常量 STALL_MIN=90（spec 钉值——可调改下行+注释）。
set -u
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$DSN" ] || { echo "[sessionpulse] FIN_PG_DSN 读不出($ENVF)——fail-closed不猜" >&2; exit 1; }
STALL_MIN=90 # 会话在修龄阈（分钟）——写死常量：大件判据钉值，调整改此行

# Q1 候选=唤起行龄超阈且 unit= token 可解（旧形无 unit 行不判——
# 配对键缺席不可对账）。unit 字符集 [A-Za-z0-9_-] 兼容下行 ~ 正则直插。
# 扫窗下界 24h（工单O-32268 回火实证 2026-11-22：无下界扫全史——
# 配对行约定前老会话 86 件全升件信号洪水;RuntimeMaxSec=2h 下 >24h
# 唤起行=无可救药死尸,升件无可处置价值）。
CAND=$(psql "$DSN" -qAt -F'|' -c "
SELECT e.id, extract(epoch from e.ts)::bigint, e.seat,
       (regexp_match(e.note,'unit=([A-Za-z0-9_-]+)'))[1]
FROM ops_event e
WHERE e.event='唤起' AND e.ts<now()-interval '${STALL_MIN} minutes'
	AND e.ts>now()-interval '24 hours'
ORDER BY e.ts" 2>/dev/null) # tolerate: 查询败=空集本拍零监控——fail-open不阻主径
[ -n "$CAND" ] || exit 0

printf '%s\n' "$CAND" | while IFS='|' read -r eid ets seat unit; do
	[ -n "$eid" ] && [ -n "$unit" ] || continue
	# Q2 配对判：同 unit 后随 收口/果/败 任一=生灭已决不升。
	# 边界 ([^0-9]|$) 防 unit 前缀互吞（o-cognition-…53 vs …530）。
	PAIR=$(psql "$DSN" -qAt -c "SELECT count(*) FROM ops_event x
		WHERE x.ts>to_timestamp($ets)
		  AND x.event IN ('会话收口','唤起果','唤起败')
		  AND x.note ~ 'unit=$unit([^0-9]|\$)'" 2>/dev/null)
	PAIR=${PAIR:-0}
	[ "$PAIR" != "0" ] && continue
	age=$(( $(date +%s) - ets ))
	"$SCRIPT_DIR/escalate.sh" '探测席' '' '会话' "session_stall_$unit" \
		"session_stall_$unit" \
		"会话在修龄超阈——唤起行无配对收口/果/败(unit=$unit seat=$seat age=${age}s)——大件半成沉判据件:处置席核ops_headless活性,活会话转票拆件/亡会话补收口" \
		|| true # tolerate: 升件写败下拍重试——监控末环不自激
done
exit 0
