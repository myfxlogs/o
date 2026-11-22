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
			VALUES('唤知机','唤起拒','','host=$PIN_HOST 主机钉位拒唤——pin=$PIN_ALLOW(deploy/seat-hosts.tsv)')" \
			>/dev/null 2>&1 || true # tolerate: 拒唤报账尽力——DSN 写败仅本机 stderr 留迹
		   echo "[dispatch] 主机钉位拒唤 host=$PIN_HOST pin=$PIN_ALLOW" >&2
		   exit 0 ;;
	esac
fi

# 提案半产件:先挖档案产改进提案升件,后扫描面含新件同拍判。
"$SCRIPT_DIR/mine.sh" || true # tolerate: 矿工败=本拍无新提案,存量信号面处置照跑

# digest 发现词面→求知票自产闸（工单O-31997）：研究席 digest 升件含
# 发现词面集∧desk 无 digest#id 引用票→机械产求知票喂研究席触发④；
# digest 销件语义不动（票自产≠信号滞留，处置席照常销）。
"$SCRIPT_DIR/digestspawn.sh" || true # tolerate: 自产闸败=本拍无新票,主触发面不受阻

# 会话在修龄阈监控（工单O-32268②）：唤起行龄>90min无配对收口/果/败
# →session_stall 升件——「大件在会话内做」沉为判据非会话判断；在办
# 脉冲在 run.sh 壳侧（①件），本件=②龄阈监控盒内薄壳面。
"$SCRIPT_DIR/sessionpulse.sh" || true # tolerate: 监控败=本拍无升件,主触发面不受阻

# 收口「调用径实证」闸（工单O-32269）：完工回执/施工收口行∧支diff
# 命中触发类(.sh新改/CREATE FUNCTION/psql调用径)∧缺实证段→
# 审计/closeout_callpath 升件——桩测≠调用径(O-30514 psql -c不插值
# 坑实证)；纯Go码件豁免。审计面机械查=本件,判据正本=工单规约D节。
"$SCRIPT_DIR/closeoutcert.sh" || true # tolerate: 监控败=本拍无升件,主触发面不受阻

# 体制读数到期判+对拍细胞（工单O-32603）：regime_digest 龄满评分落
# regime_score + 指纹变/超龄落 regime_due 标件——唤知会话经 trigger
# regime:due 按 regime_prompt.md 契约产读数；细胞件不立登记。
"$SCRIPT_DIR/regimedigest.sh" || true # tolerate: 读数面败=本拍无标件无评分,主触发面不受阻

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
# AWAIT 双计（工单O-32300②：fresh 窗内完工待验收票=直唤因子——
# AWAIT_N 总量供量表/干跑面，AWAIT_FRESH=近 COG_AWAIT_FRESH_H(默认2h)
# 内 receipt 末态票数供触发判据；陈账不独唤归 sweep/他唤同见）。
AWAIT_FRESH_H=${COG_AWAIT_FRESH_H:-2}
read -r AWAIT_N AWAIT_FRESH < <(psql "$DSN" -qAt -c "
WITH last AS (SELECT DISTINCT ON (rel) rel, act, ts FROM ops_event_act
  WHERE act IS NOT NULL ORDER BY rel, ts DESC)
SELECT count(*), count(*) FILTER (WHERE ts>now()-interval '$AWAIT_FRESH_H hours')
FROM last WHERE act='receipt'" 2>/dev/null | tr '|' ' ')
AWAIT_N=${AWAIT_N:-0}; AWAIT_FRESH=${AWAIT_FRESH:-0}
# 体制读数到期标件未消费数（regime_due 落后无 regime_digest=待产；
# 细胞件判据同源——此处只数标件不重复判龄/指纹，判据单点在 regimedigest.sh）
REGIME_DUE=$(psql "$DSN" -qAt -c "
SELECT count(*) FROM ops_event d WHERE d.event='量表' AND d.ticket='regime_due'
AND d.ts > COALESCE((SELECT max(ts) FROM ops_event
  WHERE event='量表' AND ticket='regime_digest'),'-infinity'::timestamptz)" 2>/dev/null)
REGIME_DUE=${REGIME_DUE:-0}
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

# ── 支面三探撤编→日计数（工单#22577/D-0048④：并支票族停立+落地队列
# scripts/land_queue.sh 接管「复审过→master」机械步——逐支升件回自喂环，
# pendbr/p1stall/verdfalse 三探的可见面收编每日一行量表计数）───────────
# 存量件收口：旧 scope 升件逐批复绿（探针不再发同件——账面清扫一次性
# 收口，不升件不立票；NOT EXISTS 自闭，下拍起零写）。
psql "$DSN" -qAt <<'SQL' >/dev/null 2>&1 || true # tolerate: 扫尾写败=下拍补
INSERT INTO ops_event(seat,event,note)
SELECT '探测席','恢复','升件#'||e.id||' 复绿——支面三探撤编归日计数(工单#22577/D-0048④):pendbr/p1stall/verdfalse 不再逐件升件,计数见每日一行 '||e.esc_scope
FROM ops_event e
WHERE e.event='升件' AND e.ts>now()-interval '30 days'
  AND (e.esc_scope LIKE 'pendbr\_%' OR e.esc_scope LIKE 'p1stall\_%'
       OR e.esc_scope LIKE 'verdfalse\_%')
  AND NOT EXISTS (SELECT 1 FROM ops_event y WHERE y.ts>e.ts
    AND y.note ~ ('升件#'||e.id::text||'([^0-9]|\$)'))
SQL
# 日计数三数（fetch 败=未知显 ?——不虚构；口径=原探同域同界）：
PENDBR_N="?"; P1STALL_N="?"; VERDFALSE_N="?"
if [ -n "$OREPO" ] && [ "${PBR_FETCH:-0}" = 1 ]; then
	# pendbr 计数=未并支(7d尖龄)中 OPEN 工单面无载名票者（原探同域；
	# 载名判定=票面载支名 token，原 LIKE 子串口径收紧为边界 token）。
	TICKETED_BR=$(psql "$DSN" -qAt -c "
SELECT DISTINCT m[1] FROM ops_ticket_desk t,
 LATERAL regexp_matches(t.note,'\y((fix|feat|t|audit|revfix|hotfix|canon)/[A-Za-z0-9._-]+)','g') m
WHERE NOT t.closed" 2>/dev/null)
	PBR_LIST=$(git -C "$OREPO" for-each-ref --no-merged origin/master \
		--format='%(committerdate:unix) %(refname:short)' \
		'refs/remotes/origin/fix/*' 'refs/remotes/origin/feat/*' \
		'refs/remotes/origin/audit/*' 'refs/remotes/origin/revfix/*' \
		'refs/remotes/origin/t/*' 'refs/remotes/origin/hotfix/*' \
		'refs/remotes/origin/canon/*' 2>/dev/null \
		| awk -v c=$(( $(date +%s) - 604800 )) '$1>c {print $2}' \
		| sed 's|^origin/||' | tr -d "\"'\\")
	PENDBR_N=0
	while IFS= read -r br; do
		[ -n "$br" ] || continue
		printf '%s\n' "$TICKETED_BR" | grep -qxF "$br" || PENDBR_N=$((PENDBR_N+1))
	done <<< "$PBR_LIST"
	# p1stall 计数=OPEN P1票(level=1,龄>60min)载支名∧支尖未入master 的挂支对数。
	P1CAND=$(psql "$DSN" -qAt <<'SQL'
SELECT DISTINCT t.id||' '||m[1]
FROM ops_ticket_desk t,
 LATERAL regexp_matches(t.note,'\y((fix|feat|t|audit|revfix|hotfix|canon)/[A-Za-z0-9._-]+)','g') m
WHERE NOT t.closed AND t.level=1
  AND t.filed_ts<now()-interval '60 minutes'
SQL
)
	P1STALL_N=0
	while IFS=' ' read -r tid br; do
		[ -n "$tid" ] && [ -n "$br" ] || continue
		bsha=$(git -C "$OREPO" rev-parse --verify --quiet "refs/remotes/origin/$br^{commit}" 2>/dev/null | cut -c1-9)
		[ -n "$bsha" ] || continue # 支不在远端=已删/本地件,不判
		git -C "$OREPO" merge-base --is-ancestor "$bsha" origin/master 2>/dev/null && continue
		P1STALL_N=$((P1STALL_N+1))
	done <<< "$P1CAND"
	# verdfalse 计数=24h判词载 merge=<sha> 不在 origin/master 的行数。
	VDROWS=$(psql "$DSN" -qAt <<'SQL'
SELECT id||' '||(regexp_match(note,'merge=([0-9a-f]{9,40})'))[1]
FROM ops_event WHERE ts>now()-interval '24 hours'
  AND event IN ('处置','工单','收口','在办','独语','独白','自述','自问')
  AND seat<>'探测席' AND note ~ 'merge=[0-9a-f]{9}'
SQL
)
	VERDFALSE_N=0
	while IFS=' ' read -r rid sha; do
		[ -n "$rid" ] && [ -n "$sha" ] || continue
		git -C "$OREPO" cat-file -e "${sha}^{commit}" 2>/dev/null || continue
		git -C "$OREPO" merge-base --is-ancestor "$sha" origin/master 2>/dev/null && continue
		VERDFALSE_N=$((VERDFALSE_N+1))
	done <<< "$VDROWS"
fi

# ── 器官镜组(业主令2026-11-15⑦批造——「自然人能力」基质外照层)───────────
# 对内盲区不可自省但可外照:基质病灶逻辑上不能靠基质自愈(拾取断实证——
# 修拾取的票本身也拾不起来),器官健康件=外席件非它作业。
# 况消自闭(减法 B 批·业主令#20931「不升件不唤醒,变了才更新一行,自我席每拍读」):
# ops_state 覆盖类 pending 升件其况已消→机械销(复绿自闭同型)。只销不升——
# 覆盖类(scope 白名单钦定:钦定类+模式类)永不再升件,视面行=持久可查;白名单外
# 类(对账nav_stale等同kind异scope件)不受本桥触——scope 级判非 kind 级。
psql "$DSN" -qAt <<SQL >/dev/null 2>&1 || true # tolerate: 自闭行写败=下拍补
INSERT INTO ops_event(seat,event,note)
SELECT '探测席','恢复','升件#'||e.id||' 况消自闭——ops_state已无此况'
FROM ops_esc_pending e
WHERE (e.esc_scope IN ('commercial_stall','live_dud','idle_capital','opp_eval_stall',
  'calib_drift','evalflow_dead','wip',
  'orphan_eng_lesson','orphan_econ_lesson','orphan_hypothesis_proposed',
  'orphan_hypothesis_parked','orphan_condition_gap')
  OR e.esc_scope LIKE 'lanesil\_%' ESCAPE '\'
  OR e.esc_scope LIKE 'consol\_dry\_%' ESCAPE '\'
  OR e.esc_scope LIKE 'ticketstall\_%' ESCAPE '\'
  OR e.esc_scope LIKE 'capability\_%' ESCAPE '\')
AND NOT EXISTS (SELECT 1 FROM ops_state s WHERE s.scope=e.esc_scope);
SQL

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
 ||' | 支面三计 pendbr='$PENDBR_N' p1stall='$P1STALL_N' verdfalse='$VERDFALSE_N
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
  '执行径演习')" 2>/dev/null) # 工单#22578②:唤知/审计自观类剔册——唤醒=钱/采集/部署败类+待裁+落地败行
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
# 工单O-32300③：非 crit 升件按 valence 阈直唤——valmid 档取
# COG_VAL_MID(默认60)>噪底谱（14d 实测：票滞留/审计/观测盲/孤儿账
# 45-55/default50），低权重慢性类仍 6h 兜底（D批①降噪意图保留）；
# MIN_AGE=600s 处置窗同阈面原义——鲜升件给在飞会话/人工处置留窗。
VAL_MID=${COG_VAL_MID:-60}
[ -z "$TRIGGER" ] && [ "$MAXV" -ge "$VAL_MID" ] && [ "$MAXV" -lt "$VAL_HI" ] \
	&& [ $(( NOW - OLDEST )) -gt "$MIN_AGE" ] && TRIGGER="valmid:$MAXV"
[ "$RULE_N" -gt 0 ] && TRIGGER="${TRIGGER:+$TRIGGER }rule:$RULE_N"
# 工单O-32300②：完工待验收票 fresh 窗内直唤（#22578② 原降级为兜底/搭车
# ——O-32300 普查证出必填项：wakebus 踹+timer 双径都需判据面认账，
# 否则踹=trigger=none 空转；fresh 界防滞留票每拍空唤回潮）。
[ "$AWAIT_FRESH" -gt 0 ] && TRIGGER="${TRIGGER:+$TRIGGER }await:$AWAIT_FRESH"
# 体制读数因子（O-32603）：他触发在=regime:due 随行零成本；本拍净+
# 距上唤>6h=独立唤（读数 12h 刷新窗内至多 ~2 场/日，不挤占救命带宽）。
[ "$REGIME_DUE" -gt 0 ] && [ -n "$TRIGGER" ] && TRIGGER="$TRIGGER regime:due"
[ -z "$TRIGGER" ] && [ "$REGIME_DUE" -gt 0 ] && [ $(( NOW - LAST_WOKE )) -gt 21600 ] && TRIGGER="regime:due"
[ -z "$TRIGGER" ] && [ $(( NOW - LAST_WOKE )) -gt 21600 ] && TRIGGER="sweep:6h"
# ── 审批带宽保底（工单O-30294——proposed→裁定=决策席人工件,救火信号
# 可无限挤占审批带宽:evalfloor.sh 出 evalage_h/eidle/evalfloor 议程
# 因子——fin_opp_eval proposed 最老龄>app_config阈 ∧ 连续N拍零审批
# 处置 → 保底拍;非P0救火件不挤(因子与救火因子并列不互斥),P0在飞
# (死刑/资金闸/对账/账链断升件∪守位强平在飞)让位——救命恒先。
# --sole=本拍无他触发因子才传——evalfloor.sh 内冷却闸防静默期保底
# 独唤风暴;他件在飞拍不传=因子搭车零成本不冷却)────────────────
FLOOR_TOKENS=$("$SCRIPT_DIR/evalfloor.sh" $([ -z "$TRIGGER" ] && printf -- '--sole') 2>/dev/null || true) # tolerate: 保底件败=本拍无保底因子,主触发面不受阻
# 因子并入:他件触发在=evalage_h/eidle/evalfloor 全随行(议程因子机读);
# 本拍净=仅 evalfloor 保底拍可成 TRIGGER(evalage_h/eidle 是账非信号
# ——净拍不单独唤,静默=常态)。
if [ -n "$TRIGGER" ]; then
	[ -n "$FLOOR_TOKENS" ] && TRIGGER="$TRIGGER $FLOOR_TOKENS"
elif printf '%s' "$FLOOR_TOKENS" | grep -qw evalfloor; then
	TRIGGER="$FLOOR_TOKENS"
fi
# regime:due 兜底随行——evalfloor sole 净拍等晚到因子也带上读数到期账
[ "$REGIME_DUE" -gt 0 ] && [ -n "$TRIGGER" ] && ! printf '%s' "$TRIGGER" | grep -q 'regime:due' \
	&& TRIGGER="$TRIGGER regime:due"
# 干跑口：探针/摘要照跑，唤起前退出并打印触发判定（验证用——不落唤起行不 spawn）。
[ "${COG_DRY:-}" = 1 ] && { echo "[dispatch] dry: trigger=${TRIGGER:-none} N=$N MAXV=$MAXV crit=$CRIT_N rule=$RULE_N regime=$REGIME_DUE await=$AWAIT_N ticket=$TICKET_N pendbr=$PEND_BR lastwoke=$((NOW-LAST_WOKE))s"; exit 0; }
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
	        AND x.note ~ (substring(e.note from 'unit=[^ ]+')||'([^0-9A-Za-z_-]|\$)'))
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
