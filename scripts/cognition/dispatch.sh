#!/usr/bin/env bash
# dispatch.sh —— 唤知机触发拍（纪元3 D-0045：盒内唯一新增常驻件）。
# 语义:扫信号面(未决升件+矿工候选)→达阈→systemd-run 单飞唤知会话。
# 阈面: 未决升件≥COG_MIN_N(默认1)且最老>COG_MIN_AGE(默认600s——
#   给在飞会话/人工处置留窗,防每拍对鲜信号空唤);
#   或未决≥COG_BURST_N(默认5)无视龄直唤(积压即唤);
#   或 OPEN 工单在案(未销∧无活在认领——票管催制的「派」接续,
#   票面即优先级不等龄);
#   或未入支在案(fix/feat/audit 支未入 master 且 tip 7d 内有动——
#   推支=未完链物理事实,工件即队列不靠会话散文接续;
#   merge/删支即自清,陈支不唤防古墓支死循环唤);
#   单飞闸=o-cognition-* 有 active/activating 即退(串行认知——
#   并行会话抢同一信号面=双处置双账,串行=天然去重);
#   静默退出=零信号/未达阈/单飞中(每拍静默是常态非异常)。
# 用法: systemd timer 周期驱(cognition.timer 建议 15min)或手跑。
set -u
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$DSN" ] || { echo "[dispatch] FIN_PG_DSN 读不出($ENVF)——fail-closed不猜" >&2; exit 1; }

# 提案半产件:先挖档案产改进提案升件,后扫描面含新件同拍判。
"$SCRIPT_DIR/mine.sh" || true # tolerate: 矿工败=本拍无新提案,存量信号面处置照跑

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

# 工单板（D-0045 票管催制——「立票→开修」接续环）：OPEN 票=未销
# (无处置引「工单#id」)且无活在认领(认领行<2h 引同票——
# RuntimeMaxSec 对齐,认领陈=会话死透票自然回落可拣)。
# 票本身就是排期承诺:有 OPEN 票即唤,不等龄——票面即优先级。
TICKET_N=$(psql "$DSN" -qAt -c "
SELECT count(*) FROM ops_event t
WHERE t.event='工单' AND t.ts>now()-interval '30 days'
  AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='处置' AND x.ts>t.ts
    AND x.note ~ ('工单#'||t.id::text||'([^0-9]|\$)'))
  AND NOT EXISTS (SELECT 1 FROM ops_event c WHERE c.event='认领' AND c.ts>t.ts
    AND c.ts>now()-interval '2 hours'
    AND c.note ~ ('工单#'||t.id::text||'([^0-9]|\$)'))" 2>/dev/null)
TICKET_N=${TICKET_N:-0}

# 未入支板(坑处置: 升件#11240 散文接续缝根修——会话销案留「接续:merge」
# 散文把未完链挂出触发面,队列清零工件却活着。推支未入 master 即未完工作
# 物证,工件本身当信号不等会话纪律): 7d tip 龄界防古墓支永唤;
# merge/--no-merged 自清或会话 push --delete 判废。
# 仓径自寻(坑处置: 部署机 /opt/o 是构件目录非 git 仓,真仓在 /root/o——
# 写死单径=pendbr 在真唤会话机上哑火。候选序先 /opt/o 后 /root/o):
OREPO=/opt/o
git -C "$OREPO" rev-parse --git-dir >/dev/null 2>&1 || OREPO=/root/o
git -C "$OREPO" rev-parse --git-dir >/dev/null 2>&1 || OREPO=""
if [ -n "$OREPO" ]; then
	timeout 15 git -C "$OREPO" fetch origin -q 2>/dev/null || true # tolerate: fetch败=远端瞬时,本地 remote-ref 陈旧仍数支——fail-open 不误闸
fi
PEND_BR=$([ -n "$OREPO" ] && git -C "$OREPO" for-each-ref --no-merged origin/master \
	--format='%(committerdate:unix)' \
	'refs/remotes/origin/fix/*' 'refs/remotes/origin/feat/*' 'refs/remotes/origin/audit/*' \
	2>/dev/null | awk -v c=$(( $(date +%s) - 604800 )) '$1>c' | wc -l)
PEND_BR=${PEND_BR:-0}

NOW=$(date +%s); AGE=$(( NOW - OLDEST ))
TRIGGER=""
[ "$N" -ge "$BURST_N" ] && TRIGGER="burst:$N"
[ -z "$TRIGGER" ] && [ "$N" -ge "$MIN_N" ] && [ "$AGE" -ge "$MIN_AGE" ] && TRIGGER="aged:N=$N age=${AGE}s"
[ "$TICKET_N" -gt 0 ] && TRIGGER="${TRIGGER:+$TRIGGER }ticket:$TICKET_N"
[ "$PEND_BR" -gt 0 ] && TRIGGER="${TRIGGER:+$TRIGGER }pendbr:$PEND_BR"
[ -n "$TRIGGER" ] || exit 0

# 单飞闸: o-cognition-<epoch> 会话件 active 即退(串行认知)。
# 闸面剔自身——dispatch unit 名也在 o-cognition-* glob 下,不剔=自锁死
# (首拍实证:dispatcher 把自己算进在飞会话即退,2026-11-09)。
if systemctl --user list-units 'o-cognition-*' --state=active,activating --no-legend --plain 2>/dev/null \
	| grep -vE 'o-cognition-dispatch\.service' | grep -q .; then
	exit 0
fi

UNIT="o-cognition-$(date +%s)"

# 跨机单飞闸（T-0521 认知双活根修）：本机 systemd 闸只管本机面——第二台
# dispatch 机同拍=双会话双处置双账。PG advisory lock(7969) 序列化临界区,
# 「30min 唤起在案且无同 unit 唤起败∧无后随收口」为跨机在飞判据——
# INSERT...WHERE NOT EXISTS...RETURNING 单会话内持锁,胜方插行,败方
# 读到已提交行即退。
# spawn 落败→补'唤起败'同 unit 行=败迹清锁,下拍可重唤（append-only 径——
# 不删行,补事实行）。
# 收口释锁(坑处置: 僵尸锁根修——正常收口不写败迹,唤起行尸占30min TTL,
# 历史唤起间隔30min+1s整即其尸证;收口行=正常完会话的释锁契,
# x.ts>e.ts 界保在飞不被早释,TTL 兜底 SIGKILL 无收口黑洞)。
WON=$(psql "$DSN" -qAt <<SQL
SELECT pg_advisory_lock(7969);
INSERT INTO ops_event(seat,event,ticket,note)
SELECT '唤知机','唤起','','unit=$UNIT trigger=$TRIGGER'
WHERE NOT EXISTS (
	SELECT 1 FROM ops_event e
	WHERE e.seat='唤知机' AND e.event='唤起' AND e.ts>now()-interval '30 minutes'
	  AND NOT EXISTS (SELECT 1 FROM ops_event x
	    WHERE x.ts>e.ts AND (
	      (x.seat='唤知机' AND x.event='唤起败'
	        AND x.note LIKE '%'||substring(e.note from 'unit=[^ ]+')||'%')
	      OR x.event='会话收口')))
RETURNING id;
SELECT pg_advisory_unlock(7969);
SQL
)
WON=$(printf '%s\n' "$WON" | grep -oE '^[0-9]+$' | head -1)
[ -n "$WON" ] || exit 0 # 30min 内跨机唤起在案=在飞会话非本机辖——退
systemd-run --user --unit="$UNIT" --collect \
	--property=Description="唤知会话(trigger=$TRIGGER)" \
	--property=RuntimeMaxSec=7200 \
	--setenv=COG_UNIT="$UNIT" \
	"$SCRIPT_DIR/run.sh" || {
		psql "$DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('唤知机','唤起败','','unit=$UNIT trigger=$TRIGGER spawn落败')" >/dev/null 2>&1
		echo "[dispatch] systemd-run 唤起落败(败迹已清锁)" >&2; exit 1; }
echo "[dispatch] 唤起 $UNIT trigger=$TRIGGER (esc#$WON)"
