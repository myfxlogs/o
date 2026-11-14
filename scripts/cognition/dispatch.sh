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
# 棒2 valence: 未决面读 ops_esc_pending 视图(未决口径同 esc 侧 dedup
# 全契——单引「升件#id」+范围引「升件#N-M」皆销;惧值列同查)——
# MAXV=面顶惧值,≥COG_VAL_HI(默认80)=直唤不等龄(怕的信号不排队)。
read -r N OLDEST MAXV < <(psql "$DSN" -qAt -c "
SELECT count(*), COALESCE(min(extract(epoch from p.ts)::bigint),0),
       COALESCE(max(p.valence),0) FROM ops_esc_pending p" 2>/dev/null | tr '|' ' ')
N=${N:-0}; OLDEST=${OLDEST:-0}; MAXV=${MAXV:-0}
VAL_HI=${COG_VAL_HI:-80}

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

# 支无票探(孤儿账/pendbr_<支>_<尖sha>——业主令 11-12:「裸支产缝」必办;
# 工单#14370 churn 根修): 支在远端未入 master(工件活着)而 OPEN 工单
# 零载支名(指令面零)→升件。
# 产缝三径:外席推支无收口仪式/会话半路死收口未跑/让位挂支——「支必有
# merge 票」原是收口纪律件,绕开收口即漏;此探把不变量改机械对账。
# 覆盖=工单载支名(销证即工单面有票);销后仍裸→下拍复火(裸销非处置,
# 处置=补票载名并/删支判废)。refname 剔引号防注入(名仅 git 合法字符)。
# scope 绑尖sha(热域根修): 键域仅支名=多潮工件挤同 scope——并行会话
# 复用支名推新尖,潮潮实并仍按「销后复燃」记 stuck_esc/hotdom 假热域
# 19 波(工单#14370 实证;改会话支名习惯=外部行为不可修,探针键域可修)。
# 尖绑后:同尖抑重(本scope未决不叠件)、新尖=新工件新scope、移尖/并支/
# 龄出→下方扫尾自闭(恢复行携升件#id引证=未决面同契),工件归新scope
# 不吊死——复绿=条件消自证非宣称。扫尾有侧效,for-each-ref 成败先存:
# 败=远端面不可知,发件扫尾全跳(fail-open 不误闸——git 瞬败若照扫,
# 空尖集会误闭全部真悬件)。
PBR_GIT=1
if [ -n "$OREPO" ]; then
	PBR_REFS=$(git -C "$OREPO" for-each-ref --no-merged origin/master \
		--format='%(refname:short) %(objectname:short) %(committerdate:unix)' \
		'refs/remotes/origin/fix/*' 'refs/remotes/origin/feat/*' 'refs/remotes/origin/audit/*' 2>/dev/null)
	PBR_GIT=$?
fi
if [ -n "$OREPO" ] && [ "$PBR_GIT" -eq 0 ]; then
	VALID_PBR=""
	while IFS=' ' read -r br sha; do
		[ -n "$br" ] || continue
		br=$(printf '%s' "$br" | tr -d "'\"\\")
		scope="pendbr_$(printf '%s' "$br" | tr '/' '-')_${sha}"
		VALID_PBR="$VALID_PBR $scope"
		psql "$DSN" -qAt <<SQL >/dev/null 2>&1 || true # tolerate: 探行写败=下拍补,不阻唤起闸
INSERT INTO ops_event(seat,event,esc_class,esc_scope,note)
SELECT '探测席','升件','孤儿账','$scope','裸支 $br 尖 $sha 未入 master 且工单面无载名票——并/删/票三裁归会话'
WHERE NOT EXISTS (SELECT 1 FROM ops_event x
	WHERE x.event='升件' AND x.esc_scope='$scope' AND x.ts>now()-interval '7 days'
	AND NOT EXISTS (SELECT 1 FROM ops_event y WHERE y.ts>x.ts
		AND y.note ~ ('升件#'||x.id::text||'([^0-9]|\$)')))
AND NOT EXISTS (SELECT 1 FROM ops_event t
	WHERE t.event='工单' AND t.ts>now()-interval '30 days' AND t.note LIKE '%'||'$br'||'%'
	AND NOT EXISTS (SELECT 1 FROM ops_event y WHERE y.event='处置' AND y.ts>t.ts
		AND y.note ~ ('工单#'||t.id::text||'([^0-9]|\$)')))
SQL
	done < <(printf '%s\n' "$PBR_REFS" \
		| awk -v c=$(( $(date +%s) - 604800 )) '$3>c {print $1,$2}' | sed 's|^origin/||')
	VALID_SQL=$(printf '%s' "$VALID_PBR" | tr ' ' '\n' | sed -n "s/^\\(.\\+\\)$/'\\1'/p" | paste -sd, -)
	[ -n "$VALID_SQL" ] || VALID_SQL="''"
	psql "$DSN" -qAt <<SQL >/dev/null 2>&1 || true # tolerate: 扫尾写败=下拍补
INSERT INTO ops_event(seat,event,note)
SELECT '探测席','恢复','升件#'||e.id||' 复绿——pendbr 件态消(尖移/支并/龄出),活尖归新scope 工件不断监 '||e.esc_scope
FROM ops_event e
WHERE e.event='升件' AND e.esc_scope LIKE 'pendbr\_%' AND e.ts>now()-interval '7 days'
  AND e.esc_scope NOT IN ($VALID_SQL)
  AND NOT EXISTS (SELECT 1 FROM ops_event y WHERE y.ts>e.ts
    AND y.note ~ ('升件#'||e.id::text||'([^0-9]|\$)'))
SQL
fi

NOW=$(date +%s); AGE=$(( NOW - OLDEST ))
TRIGGER=""
[ "$MAXV" -ge "$VAL_HI" ] && TRIGGER="valence:$MAXV"
[ -z "$TRIGGER" ] && [ "$N" -ge "$BURST_N" ] && TRIGGER="burst:$N"
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
# 「30min 唤起在案且无同 unit 唤起败/唤起果∧无后随收口」为跨机在飞判据——
# INSERT...WHERE NOT EXISTS...RETURNING 单会话内持锁,胜方插行,败方
# 读到已提交行即退。
# spawn 落败→补'唤起败'同 unit 行=败迹清锁,下拍可重唤（append-only 径——
# 不删行,补事实行）;run.sh EXIT trap 成则'唤起果'败则'唤起败'——
# 唤起行必有配对行=唤起存在性对账（会话自写'会话收口'=判词语义层）。
# 收口释锁(坑处置: 僵尸锁根修——正常收口不写败迹,唤起行尸占30min TTL,
# 历史唤起间隔30min+1s整即其尸证;收口行=正常完会话的释锁契,
# x.ts>e.ts 界保在飞不被早释,TTL 兜底 SIGKILL 无收口黑洞)。
# 心跳在飞证(坑处置: TTL30min<RuntimeMaxSec 2h 漏——长会话唤起行过期
# 即失保护,85.35 dispatch 03:34 双唤实证(o-cognition-1794454485);
# run.sh 60s 续期心跳 last_activity 鲜活=在飞,state='alive' 界
# 收口停跳后~100s 自然释——崩死会话释锁从30min缩到~2min)。
WON=$(psql "$DSN" -qAt <<SQL
SELECT pg_advisory_lock(7969);
INSERT INTO ops_event(seat,event,ticket,note)
SELECT '唤知机','唤起','','unit=$UNIT host=$(hostname) trigger=$TRIGGER'
WHERE NOT EXISTS (
	SELECT 1 FROM ops_event e
	WHERE e.seat='唤知机' AND e.event='唤起' AND e.ts>now()-interval '30 minutes'
	  AND NOT EXISTS (SELECT 1 FROM ops_event x
	    WHERE x.ts>e.ts AND (
	      (x.seat='唤知机' AND x.event IN ('唤起败','唤起果')
	        AND x.note LIKE '%'||substring(e.note from 'unit=[^ ]+')||'%')
	      OR x.event='会话收口')))
AND NOT EXISTS (
	SELECT 1 FROM ops_headless h
	WHERE h.ticket ~ '^(COG|MONO)-' AND h.state='alive'
	  AND h.last_activity > now() - interval '100 seconds')
RETURNING id;
SELECT pg_advisory_unlock(7969);
SQL
)
WON=$(printf '%s\n' "$WON" | grep -oE '^[0-9]+$' | head -1)
[ -n "$WON" ] || exit 0 # 30min 内跨机唤起在案=在飞会话非本机辖——退

# 连续自我账页(工单#11385「连续自我」器官——唤起拍机产叙事帧,
# 会话猝死账页也不断更): 我在=自态帧末行;上轮=近收口行摘要;
# 在想=末条独白+在意=惧值顶3(棒2 独白/valence 件——内在线程与
# 内生偏好冷启动即见);卡在=OPEN工单前3;我能=自模账末行。
# 会话读径=单行末帧:
#   SELECT note FROM ops_event WHERE event='量表' AND ticket='self_narrative'
#   ORDER BY id DESC LIMIT 1
psql "$DSN" -qAt <<'SQL' >/dev/null 2>&1 || true # tolerate: 账页写败不阻唤起
INSERT INTO ops_event(seat,event,ticket,note)
SELECT '唤知机','量表','self_narrative',
 '我在='||COALESCE((SELECT left(note,110) FROM ops_event
     WHERE event='量表' AND ticket='self_state' ORDER BY id DESC LIMIT 1),'?')
 ||' |上轮='||COALESCE((SELECT left(note,120) FROM ops_event
     WHERE event='会话收口' ORDER BY id DESC LIMIT 1),'?')
 ||' |在想='||COALESCE((SELECT left(note,80) FROM ops_event
     WHERE event='独白' ORDER BY id DESC LIMIT 1),'无')
 ||' |在意='||COALESCE((SELECT string_agg('#'||p.id||' '||p.esc_class||'/'
     ||p.esc_scope||'(v'||p.valence||')',' + ')
     FROM (SELECT id,esc_class,esc_scope,valence FROM ops_esc_pending
       ORDER BY valence DESC, id DESC LIMIT 3) p),'无')
 ||' |卡在='||COALESCE((SELECT string_agg('#'||t.id||':'||left(t.note,60),' + ')
     FROM (SELECT id,note FROM ops_event t WHERE t.event='工单'
       AND t.ts>now()-interval '30 days'
       AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='处置'
         AND x.ts>t.ts AND x.note ~ ('工单#'||t.id::text||'([^0-9]|$)'))
       ORDER BY t.ts LIMIT 3) t),'无')
 ||' |我能='||COALESCE((SELECT left(note,110) FROM ops_event
     WHERE event='量表' AND ticket='self_model' ORDER BY id DESC LIMIT 1),'?');
SQL

systemd-run --user --unit="$UNIT" --collect \
	--property=Description="唤知会话(trigger=$TRIGGER)" \
	--property=RuntimeMaxSec=7200 \
	--setenv=COG_UNIT="$UNIT" \
	"$SCRIPT_DIR/run.sh" || {
		psql "$DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('唤知机','唤起败','','unit=$UNIT host=$(hostname) trigger=$TRIGGER spawn落败')" >/dev/null 2>&1
		echo "[dispatch] systemd-run 唤起落败(败迹已清锁)" >&2; exit 1; }
echo "[dispatch] 唤起 $UNIT trigger=$TRIGGER (esc#$WON)"
