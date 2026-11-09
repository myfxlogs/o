#!/usr/bin/env bash
# dispatch.sh —— 唤知机触发拍（纪元3 D-0045：盒内唯一新增常驻件）。
# 语义:扫信号面(未决升件+矿工候选)→达阈→systemd-run 单飞唤知会话。
# 阈面: 未决升件≥COG_MIN_N(默认1)且最老>COG_MIN_AGE(默认600s——
#   给在飞会话/人工处置留窗,防每拍对鲜信号空唤);
#   或未决≥COG_BURST_N(默认5)无视龄直唤(积压即唤);
#   单飞闸=o-cognition-* 有 active/activating 即退(串行认知——
#   并行会话抢同一信号面=双处置双账,串行=天然去重);
#   静默退出=零信号/未达阈/单飞中(每拍静默是常态非异常)。
# 用法: systemd timer 周期驱(cognition.timer 建议 15min)或手跑。
set -u
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$DSN" ] || { echo "[dispatch] FIN_PG_DSN 读不出($ENVF)——fail-closed不猜" >&2; exit 1; }

MIN_N=${COG_MIN_N:-1}; MIN_AGE=${COG_MIN_AGE:-600}; BURST_N=${COG_BURST_N:-5}

# 未决升件: 后随 ops_event 无 note 引「升件#<id>」处置行=未处置。
# (纪元3处置词='处置'——处置行 note 引源升件 id 为销账同契)。
read -r N OLDEST < <(psql "$DSN" -qAt -c "
SELECT count(*), COALESCE(min(extract(epoch from e.ts)::bigint),0) FROM ops_event e
WHERE e.event='升件' AND e.ts > now() - interval '7 days'
  AND NOT EXISTS (
    SELECT 1 FROM ops_event x WHERE x.ts > e.ts
      AND x.note ~ ('升件#'||e.id::text||'([^0-9]|\$)'))" 2>/dev/null | tr '|' ' ')
N=${N:-0}; OLDEST=${OLDEST:-0}
[ "$N" -gt 0 ] || exit 0

NOW=$(date +%s); AGE=$(( NOW - OLDEST ))
TRIGGER=""
[ "$N" -ge "$BURST_N" ] && TRIGGER="burst:$N"
[ -z "$TRIGGER" ] && [ "$N" -ge "$MIN_N" ] && [ "$AGE" -ge "$MIN_AGE" ] && TRIGGER="aged:N=$N age=${AGE}s"
[ -n "$TRIGGER" ] || exit 0

# 单飞闸: o-cognition-* active 即退(串行认知)。
if systemctl --user list-units 'o-cognition-*' --state=active,activating --no-legend --plain 2>/dev/null | grep -q .; then
	exit 0
fi

UNIT="o-cognition-$(date +%s)"
systemd-run --user --unit="$UNIT" --collect \
	--property=Description="唤知会话(trigger=$TRIGGER)" \
	"$SCRIPT_DIR/run.sh" || { echo "[dispatch] systemd-run 唤起落败" >&2; exit 1; }
echo "[dispatch] 唤起 $UNIT trigger=$TRIGGER"
# 唤起落账(ops_event 唤知行——唤起事件本身可查回放;note值=自产串零注入面)
psql "$DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
	VALUES('唤知机','唤起','', 'unit=$UNIT trigger=$TRIGGER')" >/dev/null 2>&1 || true # tolerate: 自报末环——唤起落账败静默退,不递归再报（纪元2 ops_esc 同款结构性限度）
