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

# 工单板（D-0045 票管催制——「立票→开修」接续环）：OPEN 票=workable
# (未销未认领无非容量挂起)∨rulable(等裁定面——裁定件=唤知作业本票,
# fin_0190 语义层:票态单点视图,销/挂起/级/认领解析不再复制)。
# 票本身就是排期承诺:有 OPEN 票即唤,不等龄——票面即优先级。
TICKET_N=$(psql "$DSN" -qAt -c "
SELECT count(*) FROM ops_ticket_desk WHERE workable OR rulable" 2>/dev/null)
TICKET_N=${TICKET_N:-0}

# 票滞留探(工单面动线审计 2026-11-15⑤:在板票全谱晾龄≈立案龄零引证动账——
# 三选一「不许无声跳过」规则零执,会话只办门口件(升件/矿尸)作业本不拽活;
# 修复=把滞留票变成门口件——stall 阈后产 ticketstall_<id> 升件走升件道
# ~100%分诊率。键域=票行id(唯一锚);销后复火=敷衍销升件不动票=下拍复火
# 罚懒器;动账=状态事件(认领/处置/在办)引「工单#id」——散文引证不计时。
STALL_AGE=${COG_TICKET_STALL_AGE:-21600}   # 默认6h零动账=滞留(给挂起窗内缓办留量)
SUSPEND_AGE=${COG_TICKET_SUSPEND_AGE:-72}  # 挂起缓期h(挂起行ts起算非票立案ts——
# 升件#18484实证:锚错在票ts=老票挂起窗恒过期,销了又升8簇3轮每拍耗整会话;
# 级=挂起注记「挂起期限=Nh」→票面「期限=Nh」→本默认;到期回列复验语义不变)
STUCK_N=${COG_TICKET_STUCK_N:-3}        # 滞留第N次=慢性升档换问法
# 动账收窄(业主令2026-11-15⑥判词缓刑环实证预见):仅状态事件(认领/处置/
# 在办)引票算动账——自问/自述/工单注记等散文引证不计时,写评语不续命。
# 滞留第N次复火升class+换问法:不再问「有效吗」(可无限答)改问「为何结构
# 性办不了」——裁项终局化(认领落码/拆解/挂起给依赖/销),禁续判有效。
psql "$DSN" -qAt <<SQL >/dev/null 2>&1 || true # tolerate: 探行写败=下拍补,不阻唤起闸
INSERT INTO ops_event(seat,event,esc_class,esc_scope,note)
SELECT '探测席','升件',
  CASE WHEN p.n>=$STUCK_N THEN '票滞留·慢性' ELSE '票滞留' END,
  'ticketstall_'||p.id,
  '工单#'||p.id||' '||p.name||' 立案'||p.age_h||'h 零动账(第'||p.n||'次滞留)'
  ||CASE WHEN p.n>=$STUCK_N
      THEN '——有效判词已'||(p.n-1)||'轮仍零落码:裁=认领落码/拆解/挂起给依赖/销其一,禁续判有效'
      ELSE '——认领/拆解/挂起带条件/销尸票四裁其一,无声跳过=下拍复火' END
FROM (
  SELECT t.id,t.filed_ts,
    COALESCE(substring(t.note from '工单/([a-z0-9_-]+)'),'票') AS name,
    (SELECT count(*) FROM ops_event x2 WHERE x2.event='升件'
      AND x2.esc_scope='ticketstall_'||t.id AND x2.ts>now()-interval '30 days')+1 AS n,
    round(EXTRACT(EPOCH FROM (now()-t.filed_ts))/3600.0) AS age_h
  FROM ops_ticket_desk t
  WHERE t.workable
  AND t.last_active < now() - interval '1 second' * $STALL_AGE
  AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='升件'
    AND x.esc_scope='ticketstall_'||t.id AND x.ts>now()-interval '7 days'
    AND NOT EXISTS (SELECT 1 FROM ops_event y WHERE y.ts>x.ts
      AND y.note ~ ('升件#'||x.id::text||'([^0-9]|\$)')))
) p
SQL

# ticketstall 扫尾:票回活(引证动账<阈/已销/已挂起)=滞留条件消→恢复行
# 自闭未决 stall 升件(复绿=条件消自证非宣称,同 pendbr 扫尾契)。
# 坑处置(2026-11-15⑤锁堆实证):裸JOIN+相关子查询=规划器先跑贵谓词全表
# 重扫(单拍跑15min持行锁堆死迁移/写队)——CTE物化先收窄到ticketstall
# 未决小集(常空),JOIN/回活判定只在小集上跑,秒级钉死。
psql "$DSN" -qAt <<SQL >/dev/null 2>&1 || true # tolerate: 扫尾写败=下拍补
INSERT INTO ops_event(seat,event,note)
SELECT '探测席','恢复','升件#'||p.id||' 复绿——工单#'||p.tid
 ||' 回活(引证动账<阈或票已离板),滞留件归销'
FROM (
  SELECT e.id,substring(e.esc_scope from 13) AS tid FROM ops_event e
  WHERE e.event='升件' AND e.esc_scope LIKE 'ticketstall\_%'
    AND e.ts>now()-interval '7 days'
    AND NOT EXISTS (SELECT 1 FROM ops_event y WHERE y.ts>e.ts
      AND y.note ~ ('升件#'||e.id::text||'([^0-9]|\$)'))
) p
LEFT JOIN ops_ticket_desk t ON t.id=p.tid::bigint
WHERE t.id IS NULL -- 票龄出30d窗=死件,滞留升件归销
   OR t.last_active > now() - interval '1 second' * $STALL_AGE
   OR NOT t.workable
SQL

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
if [ -n "$OREPO" ] && [ "${PBR_FETCH:-0}" = 1 ]; then
	PBR_REFS=$(git -C "$OREPO" for-each-ref --no-merged origin/master \
		--format='%(refname:short) %(objectname:short) %(committerdate:unix)' \
		'refs/remotes/origin/fix/*' 'refs/remotes/origin/feat/*' 'refs/remotes/origin/audit/*' 2>/dev/null)
	PBR_GIT=$?
fi
if [ -n "$OREPO" ] && [ "$PBR_GIT" -eq 0 ] && [ "${PBR_FETCH:-0}" = 1 ]; then
	VALID_PBR=""
	while IFS=' ' read -r br sha; do
		[ -n "$br" ] || continue
		# 尖已在 master 祖链=支形残但件已并(merge 后支未删的过渡态)——不升
		git -C "$OREPO" merge-base --is-ancestor "$sha" origin/master 2>/dev/null && continue
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
AND NOT EXISTS (SELECT 1 FROM ops_ticket_desk t
	WHERE NOT t.closed AND t.note LIKE '%'||'$br'||'%')
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

# ── 器官镜组(业主令2026-11-15⑦批造——「自然人能力」基质外照层)───────────
# 对内盲区不可自省但可外照:基质病灶逻辑上不能靠基质自愈(拾取断实证——
# 修拾取的票本身也拾不起来),器官健康件=外席件非它作业。
# 四探共契:产出=升件走同分诊道(检测机械化,销裁仍归处置席——判断力不动);
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
# ③前提疑死(注册表重估): 立案契约「前提=<查>:<参>」周期重估——前提假=
# 死题疑云升件(只报疑似死,销题仍归处置席)。注册表白名单:
# eval_zero<t>/strat_open_zero<t>/esc_unres<scope>(服务端评估零注入面)。
psql "$DSN" -qAt <<SQL >/dev/null 2>&1 || true # tolerate: 探行写败=下拍补
WITH cand AS (
  SELECT t.id,(regexp_match(t.note,'前提=([a-z_]+):([a-z0-9_\-\.]+)')) m
  FROM ops_ticket_desk t
  WHERE (t.workable OR t.rulable)
    AND t.note ~ '前提=[a-z_]+:'
), chk AS (
  SELECT c.id,c.m[1] k,c.m[2] a,
    CASE c.m[1]
      WHEN 'eval_zero' THEN NOT EXISTS(SELECT 1 FROM fin_opp_eval e WHERE e.template=c.m[2] AND e.ts>now()-interval '30 days')
      WHEN 'strat_open_zero' THEN NOT EXISTS(SELECT 1 FROM fin_strategy s WHERE s.template_ref=c.m[2] AND s.state IN('proposed','paper','live'))
      WHEN 'esc_unres' THEN EXISTS(SELECT 1 FROM ops_esc_pending p WHERE p.esc_scope=c.m[2])
      ELSE NULL END AS alive
  FROM cand c WHERE c.m IS NOT NULL
)
INSERT INTO ops_event(seat,event,esc_class,esc_scope,note)
SELECT '探测席','升件','前提疑死','premdead_'||id,
 '工单#'||id||' 前提='||k||':'||a||' 评估=假——立案前提疑死(死题#15804同型):复核销题∨载新前提∨挂起条件'
FROM chk WHERE alive IS FALSE
AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='升件'
  AND x.esc_scope='premdead_'||chk.id AND x.ts>now()-interval '7 days'
  AND NOT EXISTS (SELECT 1 FROM ops_event y WHERE y.ts>x.ts
    AND y.note ~ ('升件#'||x.id::text||'([^0-9]|\$)')))
SQL
# ④判词失真(可验断言抽复): 判词载「merge=<sha>」须真在origin/master;
# 「零<模板>eval/产出」须与fin_opp_eval 30d实数相容——可证伪者升件
# (只升「判词与账矛盾」,判词对不判对错;fetch败=远端不可知全跳)。
if [ -n "$OREPO" ] && [ "${PBR_FETCH:-0}" = 1 ]; then
  psql "$DSN" -qAt <<'SQL' | while IFS=' ' read -r rid sha; do
SELECT id||' '||(regexp_match(note,'merge=([0-9a-f]{9,40})'))[1]
FROM ops_event WHERE ts>now()-interval '24 hours'
  AND event IN ('处置','工单','收口','在办','独语','独白','自述','自问')
  AND seat<>'探测席' AND note ~ 'merge=[0-9a-f]{9}'
SQL
    [ -n "$rid" ] && [ -n "$sha" ] || continue
    git -C "$OREPO" cat-file -e "${sha}^{commit}" 2>/dev/null || continue # 本地不识=不可验跳
    git -C "$OREPO" merge-base --is-ancestor "$sha" origin/master 2>/dev/null && continue # 真上岸
    psql "$DSN" -qAt <<SQL >/dev/null 2>&1 || true # tolerate: 探行写败=下拍补
INSERT INTO ops_event(seat,event,esc_class,esc_scope,note)
SELECT '探测席','升件','判词失真','verdfalse_$rid',
 '判词失真——行#$rid 称merge=$sha已上岸,实不在origin/master(判词与账矛盾):复核判词径∨口径自辩'
WHERE NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='升件'
  AND x.esc_scope='verdfalse_$rid')
SQL
  done
fi
psql "$DSN" -qAt <<SQL >/dev/null 2>&1 || true # tolerate: 探行写败=下拍补
WITH tmpl AS (SELECT DISTINCT template t FROM fin_opp_eval),
cand AS (
  SELECT e.id,t.t FROM ops_event e JOIN tmpl t ON e.note ILIKE '%'||t.t||'%'
  WHERE e.ts>now()-interval '24 hours'
    AND e.event IN ('处置','收口','在办','独语','独白','自述','自问')
    AND e.seat<>'探测席' AND e.note ~ '零.{0,8}(eval|产出)'
)
INSERT INTO ops_event(seat,event,esc_class,esc_scope,note)
SELECT '探测席','升件','判词失真','verdz_'||c.id||'_'||c.t,
 '判词失真——行#'||c.id||' 称'||c.t||'零产出,fin_opp_eval 30d实有'
 ||(SELECT count(*) FROM fin_opp_eval e WHERE e.template=c.t AND e.ts>now()-interval '30 days')
 ||'行(#17743断假前提同型):复核判词径∨口径自辩'
FROM cand c
WHERE EXISTS (SELECT 1 FROM fin_opp_eval e WHERE e.template=c.t AND e.ts>now()-interval '30 days')
AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='升件'
  AND x.esc_scope='verdz_'||c.id||'_'||c.t)
SQL
# ⑤器官自审(日级唤起): 把我手做的值守审计变成常驻节拍——量=滞留复火
# 直方图/判词失真未决/产线静默/stall→认领时延/立案→转化时延。
psql "$DSN" -qAt <<SQL >/dev/null 2>&1 || true # tolerate: 探行写败=下拍补
INSERT INTO ops_event(seat,event,esc_class,esc_scope,note)
SELECT '探测席','升件','器官自审','orgaudit_'||to_char(now() AT TIME ZONE 'Asia/Shanghai','YYYYMMDD'),
 '器官自审日拍——按runbook「第一阶段完成度账」逐格打数(五器官谱词态/作业本拾取/求生欲环/判词健康/产线活性/基质负载/点破计数);产出器官账行∨异常立案'
WHERE NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='升件'
  AND x.esc_scope='orgaudit_'||to_char(now() AT TIME ZONE 'Asia/Shanghai','YYYYMMDD'))
SQL
# ⑥提炼旱(教训→canon转化断): 验尸攒案不结晶=「记全了没提炼」——
# e551v/0c 实证(值守席判决 B2)。判据=verdicted≥50 且 canon转化<5%
# (eng 13/37=35%健阈不扰,econ 0/551=全断照火)。只述积压开药归会话。
psql "$DSN" -qAt <<SQL >/dev/null 2>&1 || true # tolerate: 探行写败=下拍补
WITH l AS (
  SELECT 'eng' tbl, count(*) FILTER (WHERE status='verdicted') v,
    count(*) FILTER (WHERE status='canon') c FROM fin_eng_lesson
  UNION ALL
  SELECT 'econ', count(*) FILTER (WHERE status='verdicted'),
    count(*) FILTER (WHERE status='canon') FROM fin_econ_lesson)
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
# ⑦盲区账缺(注册表缺席探): event='盲区' 行制=「不知道自己不知道什么」
# 清单(工单#18324——结构常数主动管理件,自指/同源/墙外三域种子)。
# 零行=地图未建升件;7d 复火窗=销判后仍空再鸣(同 consol_dry 窗语义)。
psql "$DSN" -qAt <<SQL >/dev/null 2>&1 || true # tolerate: 探行写败=下拍补
INSERT INTO ops_event(seat,event,esc_class,esc_scope,note)
SELECT '探测席','升件','器官自审','blindspot_map_absent',
 '盲区账缺——event=盲区 注册表零行:「不知道自己不知道什么」清单未建'
 ||'(自指/同源/墙外=结构性常数须主动管理,工单#18324 在板)——认领建图∨判口径'
WHERE NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='盲区')
AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='升件'
  AND x.esc_scope='blindspot_map_absent' AND x.ts>now()-interval '7 days'
  AND NOT EXISTS (SELECT 1 FROM ops_event y WHERE y.ts>x.ts
    AND y.note ~ ('升件#'||x.id::text||'([^0-9]|\$)')))
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
# glob 收窄 *.service(坑处置 2026-11-14②:timer 件恒 active 同入
# o-cognition-* glob——dispatch/monologue 双闸自锁死,models 侧零产实证)
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
	  AND e.note NOT LIKE '%trigger=monologue%'
	  AND NOT EXISTS (SELECT 1 FROM ops_event x
	    WHERE x.ts>e.ts AND (
	      (x.seat='唤知机' AND x.event IN ('唤起败','唤起果')
	        AND x.note LIKE '%'||substring(e.note from 'unit=[^ ]+')||'%')
	      OR x.event='会话收口')))
AND NOT EXISTS (
	-- 独白槽位令(2026-11-14):MONO 心跳剔出在飞面——独白=低优先
	-- 租客永不拦工(AUD- 本不入面先例;独白唤起行上行同剔)。
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
