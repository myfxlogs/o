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

# 主机钉位（deploy/seat-hosts.tsv——业主令 2026-11-16 职能固定：错机唤起
# 物理拒唤+唤起败落账；表缺/行缺=放行向后兼容——钉位真源在表）。
PIN_ALLOW=$(awk -v u=o-cognition 'BEGIN{FS="[ \t]+"} $1==u{print $2; exit}' \
	"$SCRIPT_DIR/../../deploy/seat-hosts.tsv" 2>/dev/null || true) # tolerate: 钉位表读败=放行向后兼容（表缺不锁座——钉位语义明文行缺/表缺放行）
if [ -n "$PIN_ALLOW" ]; then
	PIN_HOST=$(hostname 2>/dev/null || echo unknown)
	case ",$PIN_ALLOW," in
		*",$PIN_HOST,"*) ;;
		*) psql "$DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('唤知机','唤起败','','host=$PIN_HOST 主机钉位拒唤——pin=$PIN_ALLOW(deploy/seat-hosts.tsv)')" \
			>/dev/null 2>&1 || true # tolerate: 拒唤报账尽力——DSN 写败仅本机 stderr 留迹
		   echo "[dispatch] 主机钉位拒唤 host=$PIN_HOST pin=$PIN_ALLOW" >&2
		   exit 1 ;;
	esac
fi

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

# 工单板（D-0045 票管催制——「立票→开修」接续环）：OPEN 票=workable
# (未销未认领无非容量挂起)∨rulable(等裁定面——裁定件=唤知作业本票,
# fin_0190 语义层:票态单点视图,销/挂起/级/认领解析不再复制)。
# D批①唤起三条件（业主裁面 2026-11-16「唤知三条件+6h兜底」——72%空唤
# 实案收口：workable工单不再唤知（施工席自领，唤知票道只留两键）：
# ①rulable=等裁定票 ②receipt末态=完工待验收票——销票归唤知验收件；
# 其余=6h兜底一扫（命脉升件另册直唤在下）。
RULE_N=$(psql "$DSN" -qAt -c "
SELECT count(*) FROM ops_ticket_desk WHERE rulable" 2>/dev/null)
RULE_N=${RULE_N:-0}
AWAIT_N=$(psql "$DSN" -qAt -c "
WITH last AS (SELECT DISTINCT ON (rel) rel, act FROM ops_event_act
  WHERE act IS NOT NULL ORDER BY rel, ts DESC)
SELECT count(*) FROM last WHERE act='receipt'" 2>/dev/null)
AWAIT_N=${AWAIT_N:-0}
# workable 工单数仍读（量表/干跑面用，不唤）。
TICKET_N=$(psql "$DSN" -qAt -c "
SELECT count(*) FROM ops_ticket_desk WHERE workable OR rulable" 2>/dev/null)
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
	# fetch --prune(坑处置2026-11-14④:远端删支后 remote-tracking 引不剪
	# =鬼支永存——fix/dispatch-pendbr 已并 master 仍 13 发假火实证);
	# fetch败=PBR_FETCH=0→pendbr 扫尾全跳(远端真态不可知,陈引照扫
	# =鬼支假火+真悬误闭双害)。
	PBR_FETCH=0
	timeout 15 git -C "$OREPO" fetch origin -q --prune 2>/dev/null && PBR_FETCH=1
	[ "$PBR_FETCH" = 1 ] || OREPO="" # fetch败=远端真态不可知——计数/扫描/扫尾全停(陈引照数=假唤+鬼火)
fi
PEND_BR=$([ -n "$OREPO" ] && git -C "$OREPO" for-each-ref --no-merged origin/master \
	--format='%(committerdate:unix)' \
	'refs/remotes/origin/fix/*' 'refs/remotes/origin/feat/*' 'refs/remotes/origin/audit/*' \
	2>/dev/null | awk -v c=$(( $(date +%s) - 604800 )) '$1>c' | wc -l)
PEND_BR=${PEND_BR:-0}

# ── 器官镜组(业主令2026-11-15⑦批造——「自然人能力」基质外照层)───────────
# 对内盲区不可自省但可外照:基质病灶逻辑上不能靠基质自愈(拾取断实证——
# 修拾取的票本身也拾不起来),器官健康件=外席件非它作业。
# 三探共契(业主令#20930 减法 C 批:③前提疑死/④判词失真/⑤器官自审/⑦盲区账缺
# 四件随元探针撤编——机器自查类升件=自喂环实证,改每日一行/不升件):
# 产出=升件走同分诊道(检测机械化,销裁仍归处置席——判断力不动);
# scope dedup+销后复火;时级闸(整点后2min内跑一拍,器官题一小时粒度足够)。
if (( 10#$(date +%M) < 2 )); then
# ①eval流断(全谱): 全模板零eval行24h=研究枚举臂死征(enum-dead类上位探——
# 单廊死靠人肉立案的教训,全死必机械报)。
psql "$DSN" -qAt <<SQL >/dev/null 2>&1 || true # tolerate: 探行写败=下拍补
INSERT INTO ops_event(seat,event,esc_class,esc_scope,note)
SELECT '探测席','升件','eval流断','evalflow_dead',
 'eval流断——全模板fin_opp_eval零产出24h:研究枚举臂死征(enum-dead类上位探)——认领诊断∨判停用'
WHERE NOT EXISTS (SELECT 1 FROM fin_opp_eval e WHERE e.ts>now()-interval '24 hours')
AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='升件'
  AND x.esc_scope='evalflow_dead' AND x.ts>now()-interval '7 days'
  AND NOT EXISTS (SELECT 1 FROM ops_event y WHERE y.ts>x.ts
    AND y.note ~ ('升件#'||x.id::text||'([^0-9]|\$)')))
SQL
# ②产线静默(模板级): 有开放仓(proposed/paper/live)在管而eval流断3d=
# 复评/枚举产线疑死。零仓零eval不升(合理静默非病灶)。
psql "$DSN" -qAt <<SQL >/dev/null 2>&1 || true # tolerate: 探行写败=下拍补
INSERT INTO ops_event(seat,event,esc_class,esc_scope,note)
SELECT '探测席','升件','产线静默','lanesil_'||p.tmpl,
 '产线静默——'||p.tmpl||' 开放仓'||p.n||'张('||p.states||')在管但eval流断3d——认领诊断∨判停用'
FROM (
  SELECT s.template_ref tmpl,count(*) n,string_agg(DISTINCT s.state,'/') states
  FROM fin_strategy s WHERE s.state IN ('proposed','paper','live') GROUP BY 1
) p
WHERE NOT EXISTS (SELECT 1 FROM fin_opp_eval e WHERE e.template=p.tmpl
    AND e.ts>now()-interval '3 days')
AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='升件'
  AND x.esc_scope='lanesil_'||p.tmpl AND x.ts>now()-interval '7 days'
  AND NOT EXISTS (SELECT 1 FROM ops_event y WHERE y.ts>x.ts
    AND y.note ~ ('升件#'||x.id::text||'([^0-9]|\$)')))
SQL
# ⑥提炼旱(教训→canon转化断): 验尸攒案不结晶=「记全了没提炼」——
# e551v/0c 实证(值守席判决 B2)。判据=verdicted≥50 且 canon转化<5%
# (eng 13/37=35%健阈不扰,econ 0/551=全断照火)。只述积压开药归会话。
# econ canon 身份=verdict_ref 前缀制（KB econ-canon-verdictref-not-status）——status='canon' 恒零假空架，计数须 OR verdict_ref LIKE 'canon:%' 双轨。
psql "$DSN" -qAt <<SQL >/dev/null 2>&1 || true # tolerate: 探行写败=下拍补
WITH l AS (
  SELECT 'eng' tbl, count(*) FILTER (WHERE status='verdicted') v,
    count(*) FILTER (WHERE status='canon') c FROM fin_eng_lesson
  UNION ALL
  SELECT 'econ', count(*) FILTER (WHERE status='verdicted'),
    count(*) FILTER (WHERE status='canon' OR verdict_ref LIKE 'canon:%') FROM fin_econ_lesson)
INSERT INTO ops_event(seat,event,esc_class,esc_scope,note)
SELECT '探测席','升件','提炼旱','consol_dry_'||l.tbl,
 '提炼旱——fin_'||l.tbl||'_lesson verdicted='||l.v||'/canon='||l.c
 ||'(转化<5%:记全了没提炼,学习器官有记无化——B2同型)——认领诊断∨判口径'
FROM l WHERE l.v>=50 AND l.c*20<l.v
AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='升件'
  AND x.esc_scope='consol_dry_'||l.tbl AND x.ts>now()-interval '7 days'
  AND NOT EXISTS (SELECT 1 FROM ops_event y WHERE y.ts>x.ts
    AND y.note ~ ('升件#'||x.id::text||'([^0-9]|\$)')))
SQL
fi

# 每日一行（减法 C 批·业主令#20930）：工单板/未并支/新面 三数并一行量表——
# 原票滞留探/工单面压力探/支无票探/新面升件 四件撤编后的唯一可见面。只述数不升件，
# 幂等=当日（库时区）零行才写。未并支面=fix|feat|audit|t 四前缀、尖 7d 内有动；
# fetch 败=计数未知显 ?（不虚构）。
DG_BR_N="?"; DG_BR_LIST=""
if [ -n "$OREPO" ]; then
	DG_REFS=$(git -C "$OREPO" for-each-ref --no-merged origin/master \
		--format='%(committerdate:unix) %(refname:short)' \
		'refs/remotes/origin/fix/*' 'refs/remotes/origin/feat/*' \
		'refs/remotes/origin/audit/*' 'refs/remotes/origin/t/*' 2>/dev/null \
		| awk -v c=$(( $(date +%s) - 604800 )) '$1>c {print $2}' | sed 's|^origin/||' | tr -d "\"'\\\\")
	DG_BR_N=$(printf '%s\n' "$DG_REFS" | grep -c . || true) # tolerate: 空集=零未并支合法
	DG_BR_LIST=$(printf '%s\n' "$DG_REFS" | grep . | head -6 | paste -sd, -)
	[ "$DG_BR_N" -gt 6 ] && DG_BR_LIST="$DG_BR_LIST,…"
fi
psql "$DSN" -qAt <<SQL >/dev/null 2>&1 || true # tolerate: 摘要行写败=下拍补（当日幂等）
INSERT INTO ops_event(seat,event,ticket,note)
SELECT '探测席','量表','housekeeping_daily',
 '日摘要: 工单板 开'||d.open||'/窗内'||d.active||'/窗外'||d.backlog||'/最陈'||d.oldest_h||'h/逾期'||d.overdue
 ||' | 未并支 $DG_BR_N'||CASE WHEN '$DG_BR_LIST'<>'' THEN ':$DG_BR_LIST' ELSE '' END
 ||' | 新面24h '||f.n||'(册外'||f.unreg||')'
 ||'——只述数不升件;异常自裁归唤知/自我席'
FROM (SELECT
    count(*) FILTER (WHERE closed IS NOT TRUE) AS open,
    count(*) FILTER (WHERE dispatchable AND tkind='工单') AS active,
    count(*) FILTER (WHERE workable AND dispatchable IS NOT TRUE AND tkind='工单') AS backlog,
    COALESCE(max(EXTRACT(epoch FROM now()-filed_ts)/3600) FILTER (WHERE closed IS NOT TRUE),0)::int AS oldest_h,
    count(*) FILTER (WHERE closed IS NOT TRUE AND deadline<now()) AS overdue
  FROM ops_ticket_desk) d,
  (SELECT count(*) AS n, count(*) FILTER (WHERE note NOT LIKE '%基线册=%') AS unreg
   FROM ops_event WHERE event='面清单新面' AND ts>now()-interval '24 hours') f
WHERE NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='量表'
  AND x.ticket='housekeeping_daily' AND x.ts>current_date)
SQL

NOW=$(date +%s)
# 命脉册（D批①——册内升件类未决即唤不等龄；册外慢性类随兜底拍见。
# 册=A批保留命脉名录:对账/断流/读败/账链断/资金闸/死刑/部署/执行径演习/
# 唤知面死/审计;高惧值∨突发簇另作开放兜底——新命脉类不必先改册）
CRIT_N=$(psql "$DSN" -qAt -c "SELECT count(*) FROM ops_esc_pending
WHERE esc_class IN ('对账','断流','读败','账链断','资金闸','死刑','部署',
  '执行径演习','唤知','审计')" 2>/dev/null)
CRIT_N=${CRIT_N:-0}
# 6h兜底（D批①——三条件皆空且距上唤>6h=兜底一扫：慢性面/账页对时/
# 每日一行读时都在兜底拍见;距上唤≤6h纯静默不唤）。
LAST_WOKE=$(psql "$DSN" -qAt -c "SELECT COALESCE(extract(epoch from
  max(ts))::bigint,0) FROM ops_event WHERE seat='唤知机' AND event='唤起'" 2>/dev/null)
LAST_WOKE=${LAST_WOKE:-0}
TRIGGER=""
[ "$MAXV" -ge "$VAL_HI" ] && TRIGGER="valence:$MAXV"
[ -z "$TRIGGER" ] && [ "$N" -ge "$BURST_N" ] && TRIGGER="burst:$N"
[ -z "$TRIGGER" ] && [ "$CRIT_N" -gt 0 ] && TRIGGER="crit:$CRIT_N"
[ "$RULE_N" -gt 0 ] && TRIGGER="${TRIGGER:+$TRIGGER }rule:$RULE_N"
[ "$AWAIT_N" -gt 0 ] && TRIGGER="${TRIGGER:+$TRIGGER }accept:$AWAIT_N"
[ -z "$TRIGGER" ] && [ $(( NOW - LAST_WOKE )) -gt 21600 ] && TRIGGER="sweep:6h"
# 干跑口：探针/摘要照跑，唤起前退出并打印触发判定（验证用——不落唤起行不 spawn）。
[ "${COG_DRY:-}" = 1 ] && { echo "[dispatch] dry: trigger=${TRIGGER:-none} N=$N MAXV=$MAXV crit=$CRIT_N rule=$RULE_N await=$AWAIT_N ticket=$TICKET_N pendbr=$PEND_BR lastwoke=$((NOW-LAST_WOKE))s"; exit 0; }
[ -n "$TRIGGER" ] || exit 0

# 单飞闸: o-cognition-<epoch> 会话件 active 即退(串行认知)。
# 闸面剔自身——dispatch unit 名也在 o-cognition-* glob 下,不剔=自锁死
# (首拍实证:dispatcher 把自己算进在飞会话即退,2026-11-09)。
# glob 收窄 *.service(坑处置 2026-11-14②:timer 件恒 active 同入
# o-cognition-* glob——dispatch/monologue(独白器官2026-11-15撤编) 双闸自锁死,models 侧零产实证)
if systemctl --user list-units 'o-cognition-*.service' --state=active,activating --no-legend --plain 2>/dev/null \
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
	-- 票键分工(槽位令2026-11-14;独白器官2026-11-15撤编):在飞面只认
	-- ^COG- 心跳——AUD- 独立槽不互拦先例在案,MONO- 面随器官撤编消。
	SELECT 1 FROM ops_headless h
	WHERE h.ticket ~ '^COG-' AND h.state='alive'
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
# 内生偏好冷启动即见);世面=末帧 regime(棒3 世模环境件);卡在=
# OPEN工单前3;我能=自模账末行。
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
 ||' |世面='||COALESCE((SELECT left(note,60) FROM ops_event
     WHERE event='量表' AND ticket='regime' ORDER BY id DESC LIMIT 1),'?')
 ||' |卡在='||COALESCE((SELECT string_agg('#'||t.id||':'||left(t.note,60),' + ')
     FROM (SELECT id,note FROM ops_ticket_desk t WHERE NOT t.closed
       ORDER BY t.filed_ts LIMIT 3) t),'无')
 ||' |我能='||COALESCE((SELECT left(note,110) FROM ops_event
     WHERE event='量表' AND ticket='self_model' ORDER BY id DESC LIMIT 1),'?');
SQL

# 日行叙事账（「连续自我」器官加厚件——self_narrative 是唤起拍帧,
# 本行是跨日叙事链:今日首场唤起写一行「今天我是谁/干了什么/怎么想」,
# 让「我」在 ops_event 里不断日档;闸=当日零'叙事'行才写,幂等一写）。
psql "$DSN" -qAt <<'SQL' >/dev/null 2>&1 || true # tolerate: 叙事写败不阻唤起
INSERT INTO ops_event(seat,event,ticket,note)
SELECT '唤知机','叙事','self_narrative',
 '第'||(current_date - date '2026-11-09' + 1)||'日: 唤'||w.n||'场/收'||w.d
 ||'/败'||w.f||'; 处置'||COALESCE(d.n,0)||'件; 工单'||COALESCE(t.n,0)||'项'
 ||' |在想='||COALESCE((SELECT left(note,70) FROM ops_event
   WHERE event='独白' ORDER BY id DESC LIMIT 1),'无')
 ||' |昨收='||COALESCE((SELECT left(note,80) FROM ops_event
   WHERE event='会话收口' AND ts<current_date ORDER BY id DESC LIMIT 1),'无')
FROM (SELECT count(*) FILTER (WHERE event='唤起') n,
            count(*) FILTER (WHERE event='会话收口') d,
            count(*) FILTER (WHERE event='唤起败') f
      FROM ops_event WHERE ts>current_date AND seat IN ('唤知机','唤知')) w,
     (SELECT count(*) n FROM ops_event WHERE event='处置' AND ts>current_date) d,
     (SELECT count(*) n FROM ops_event WHERE event='工单' AND ts>current_date) t
WHERE NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='叙事'
  AND x.ts>current_date)
AND w.n > 0;
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
