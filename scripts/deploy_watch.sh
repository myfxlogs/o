#!/usr/bin/env bash
# deploy_watch.sh —— 部署管道自醒守望（业主令 2026-11-10「收口与命脱钩」：
# merge→push→deploy 最后一步不再靠会话寿命——master HEAD 与产线镜像 SHA
# 滞差即自动部署；同步失败/部署败→升件 deploy_lag/deploy_blocked 转信号，
# 唤知会话清障。被动件有 merge 才醒的钦定形态，15min 一拍幂等）。
# 工单#25760 o-core 换件部署窗：谱系判命中 cmd/core 依赖闭包=镜像必换+钱径
# 重启——随 merge 即换废止，收窄到 UTC 低活动窗集中放件（app_config
# deploy.ocore_window_utc，缺省 19:00-21:00）；窗外=入窗队列让位（不换
# o-core，OCI 臂面照走），越窗未放（最老待放 commit 龄>max_hold）升件；
# 首次部署（DEPLOYED 空）与会话紧急径（deploy_at_sha/deploy_image）不入窗。
# 测试时钟缝：O_DEPLOY_NOW_EPOCH（秒——窗内/窗外/越窗可重放，同 O_MIRROR_NOW 先例）。
set -uo pipefail
cd "$(git rev-parse --show-toplevel)" || exit 1
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN="${FIN_PG_DSN:-$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')}"
NOW_EPOCH=${O_DEPLOY_NOW_EPOCH:-$(date +%s)}
say() { echo "[deploy_watch] $*"; }
sqlq() { printf '%s' "$1" | sed "s/'/''/g"; } # tolerate: sed 恒 rc0——纯转义器非闸（同 land_queue.sh 件）
# ── 工单#25803 心跳面：每拍写 ops_event（seat=DEPLOY event=心跳）──
# 看门狗=hunger ProbeDeployWatch 读末行龄>45min（3×15min拍距）升
# deploy_watch_silent——timer死/服务崩/锁卡死=心跳停三因同覆。
# 写败容忍（心跳非钱径，下拍重写不翻部署判）。EXIT trap 全出口覆盖
# （含 ref缺/让位/败径——任一出口=本拍活着即心跳）。
# 节流取值说明：派工单钦定「判词变化∨距末心跳>2h」与看门狗阈
# 「末行龄>45min」自相矛盾——2h 节流窗下末行龄常态达 2h≫45min=健康
# 态恒误报；且拍间隔上界≈30min（15min拍+15min服务超时）下任何>15min
# 节流阈都可能顶穿 45min。取每拍一行（~96行/日≈ops_event日量7%——
# 非洪水）保验收可绿；判词随行载（sha/base/态）备查。
HB_STATE="拍始"; ESC_WRITE_FAIL=
# 工单#30087 态/实劈差自审映射——终态判词∈此表=本拍 esc() 应已产未销行;
# 行缺=主写径断（esc 崩/码路未达）→第二径补火+心跳载「esc缺」显形。
# 只列无条件 esc 态;条件阈态（CI待闸>90min/越窗>max_hold/oci_arm_*）条件
# 未达即无行属正常——无法区分「未达」与「断径」,不入表防误报。
declare -A ESC_AUDIT=(
	[CI红断]=deploy_ci_red [部署败]=deploy_blocked [脏树连拍]=deploy_blocked
	[push败]=deploy_push_blocked [diverged]=deploy_diverged
	[窗配置非法]=deploy_ocore_window_config
)
# 工单#30514：esc() 函件去重抑制登记——ops_esc 返 NULL=同窗已报非写败；
# 态/实劈差自审凭它区分「主写径断」与「去重合法抑制」，防误补火。
declare -A ESC_DEDUPED=()
hb_write() {
	[ -n "$DSN" ] || return 0
	local sha ascp last lst lage anchor
	sha=$(git rev-parse --short=12 master 2>/dev/null); sha=${sha:-?}
	ascp=${ESC_AUDIT[$HB_STATE]:-}
	if [ -n "$ascp" ] && ! esc_pending "$ascp" && [ -z "${ESC_DEDUPED[$ascp]:-}" ]; then
		bash scripts/cognition/escalate.sh '探测席' '' '部署' deploy_esc_missing \
			"部署/deploy_esc_missing:$ascp" \
			"态 $HB_STATE 判毕而升件 $ascp 无未销行——主写径断自审补火,归唤知会话查 esc() 径" >/dev/null 2>&1 || true # tolerate: 第二径再败无第三径——败痕已由下行 esc缺 载心跳显形,非静默吞
		HB_STATE="$HB_STATE esc缺"
	fi
	[ -n "$ESC_WRITE_FAIL" ] && HB_STATE="$HB_STATE esc写败"
	# 工单#30514 心跳降噪「态变+锚拍」：末行态同∧末行龄<锚距→跳写
	# （态恒同时复读行零信息）；锚距=app_config deploy.hb_anchor_min
	# 缺省20min——须>15min拍距才有降噪可达（spec字面≤5min与拍距互斥
	# =每拍皆锚零降噪,按「锚距<活性判据窗」约束取20min）且<45min
	# 活性窗（hunger deploywatch.go deployWatchSilentMaxAge）留
	# degraded 拍距余量。末行查询败/零行史=照写（fail-open 保鲜度）。
	anchor=$(cfg_get deploy.hb_anchor_min 20)
	case "$anchor" in ''|*[!0-9]*|0*) anchor=20;; esac
	last=$(psql "$DSN" -qAt -c "SELECT substr(split_part(note,' sha=',1),4)||'|'||floor(extract(epoch from now()-ts))::int
		FROM ops_event WHERE seat='DEPLOY' AND event='心跳' ORDER BY id DESC LIMIT 1" 2>/dev/null)
	lst=${last%%|*}; lage=${last##*|}
	if [ -n "$last" ] && [[ "$lage" =~ ^[0-9]+$ ]] && [ "$lst" = "$HB_STATE" ] \
		&& [ "$lage" -lt $((anchor*60)) ]; then
		say "心跳同态跳写（末行态同龄 ${lage}s<锚 ${anchor}min）"
		return 0
	fi
	if psql "$DSN" -c "INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('DEPLOY','心跳','','心跳 $HB_STATE sha=$sha base=${DEPLOYED:-?}')" >/dev/null 2>&1; then
		say "心跳: $HB_STATE"
	else
		say "心跳写败（不阻断，下拍重写）"
	fi
}
trap hb_write EXIT
esc_pending() { # esc_pending <scope> → rc0=未销升件在案（24h窗——esc dedup 同源判据，
	# 工单#25481 抽函数：自动换件重试风暴闸复用同判——有未销败件即让位本拍）
	local scope="$1" s
	s=$(sqlq "$scope")
	[ -n "$DSN" ] || return 1
	psql "$DSN" -qAt -c "
		SELECT count(*) FROM ops_event e WHERE e.event='升件' AND e.esc_scope='$s'
		AND e.ts>now()-interval '24 hours'
		AND NOT EXISTS (SELECT 1 FROM ops_event x2 WHERE x2.ts>e.ts
			AND x2.note ~ ('升件#'||e.id::text||'([^0-9]|$)'))" 2>/dev/null | grep -qxv 0
}
esc() { # 升件 dedup=未销才抑（同 escEdgeU 语义——销而未修重火）
	# 工单#30514 写径归一：主径=库内函件 ops_esc()（fin_0228——
	# escalate.sh 语义复刻：note=prefix||'——'||detail、24h note 前缀
	# 同窗去重、class/scope 空→NULL、psql -v 参数化喂参引号结构性
	# 绝根——#29766 引号断径病灶的根修）。esc_pending 未销预检保留
	# （scope 级未销判不变）；函件返 NULL=同窗已报=去重合法抑制非写
	# 败——ESC_DEDUPED 登记防态/实劈差自审误判「主写径断」补火。
	# 旗形注：主径用分立 -q -A -t（语义≡-qAt 但桩按整词 -qAt 识别
	# 兜底径——deploy_watch_test 分径锚，勿并旗改回 -qAt）。
	local scope="$1" note="$2" out
	[ -n "$DSN" ] || return 0
	esc_pending "$scope" && return 0
	# ON_ERROR_STOP=1：函件缺席/SQL 错必须 rc≠0 落兜底径——psql 缺省
	# SQL 错仍 rc0 会被误判「去重抑制」静默吞（写径假净回潮防线）。
	out=$(psql "$DSN" -q -A -t -v ON_ERROR_STOP=1 \
		-v seat='探测席' -v ticket='' -v cls='部署' -v scp="$scope" \
		-v prefix="部署/${scope}:" -v detail="$note" \
		-c "SELECT ops_esc(:'seat',:'ticket',:'cls',:'scp',:'prefix',:'detail')" 2>/dev/null) \
		&& { [ -n "$out" ] || ESC_DEDUPED[$scope]=1; return 0; }
	# 工单#30086 主径写败→第二径 escalate.sh 兜底（独立码路——同一坑不双断;
	# 函件归一后兜底=同函件异会话重试+失败显形，语义不变）+心跳载「esc写败」显形——升件#29764 实证
	# 「写败静默=信号面假净」,败留痕=写径健康自感。
	ESC_WRITE_FAIL=1
	bash scripts/cognition/escalate.sh '探测席' '' '部署' "$scope" \
		"部署/${scope}:" "$note" >/dev/null 2>&1 || true # tolerate: 兜底径再败无第三径——ESC_WRITE_FAIL 已置,心跳载 esc写败 显形非静默
}
# cfg_get <key> <default> —— app_config 只读取值（缺键/DSN 缺=default）。
cfg_get() {
	local v=""
	[ -n "$DSN" ] && v=$(psql "$DSN" -qAt -c "SELECT value #>> '{}' FROM app_config WHERE key='$1'" 2>/dev/null)
	[ -n "$v" ] && printf '%s' "$v" || printf '%s' "$2"
}
# ow_valid <HH:MM-HH:MM> —— 窗形/端点校验（start≠end；UTC 语义）。
ow_valid() {
	local s="$1"
	[[ "$s" =~ ^([01][0-9]|2[0-3]):[0-5][0-9]-([01][0-9]|2[0-3]):[0-5][0-9]$ ]] || return 1
	[ "${s%%-*}" != "${s##*-}" ]
}
ow_min() { local t="$1"; printf '%s' $(( 10#${t:0:2} * 60 + 10#${t:3:2} )); }
# ow_in_window <spec> <epoch> → rc0=UTC 窗内（左闭右开；start>end=跨午夜）。
ow_in_window() {
	local start end now
	start=$(ow_min "${1%%-*}"); end=$(ow_min "${1##*-}")
	now=$(( ($2 % 86400) / 60 ))
	if [ "$start" -le "$end" ]; then
		[ "$now" -ge "$start" ] && [ "$now" -lt "$end" ]
	else
		[ "$now" -ge "$start" ] || [ "$now" -lt "$end" ]
	fi
}
git fetch origin master -q 2>/dev/null
LOCAL=$(git rev-parse master 2>/dev/null); REMOTE=$(git rev-parse origin/master 2>/dev/null)
[ -n "$LOCAL" ] && [ -n "$REMOTE" ] || { HB_STATE=ref缺; say "ref 缺——退出"; exit 0; }
BASE=$(git merge-base master origin/master)
if [ "$LOCAL" != "$REMOTE" ]; then
	if [ "$BASE" = "$LOCAL" ]; then
		git merge --ff-only origin/master -q 2>/dev/null && say "ff 至 origin/master $(git rev-parse --short master)"
		LOCAL=$(git rev-parse master)
	elif [ "$BASE" = "$REMOTE" ]; then
		# 尖未盖戳=合流在途（merge→check_all→converge 序内）——让位盖戳流
		# 不自推不升件（盖戳后本拍或会话径自然上岸；盲推撞 pre-push 闸=
		# 自伤型升件源，2026-11-10 产线实证 deploy_push_blocked 三连发）。
		if [ ! -f ".git/check-green-$LOCAL" ]; then
			HB_STATE=让位盖戳; say "尖未盖戳——让位盖戳流（不推不升）"
			exit 0 # 让位=真让位（不落部署段——未推尖部署必遭出处闸拒=自伤升件）
		elif ! git push origin master 2>/dev/null; then
			# 推败先对拍：mirror_push 并行推同 sha 落地=竞窗让位非闸阻——
			# 远端已含本地尖=条件自愈如实（升件#9531→#11330 三连同型：
			# 逐案「瞬态」为真而触发器结构必再产——判错界根修）。
			git fetch origin master -q 2>/dev/null
			if git merge-base --is-ancestor "$LOCAL" origin/master 2>/dev/null; then
				HB_STATE=push并行让位; say "push 撞并行推件——远端已含尖，让位"
				exit 0
			fi
			HB_STATE=push败
			esc "deploy_push_blocked" "部署/deploy_push_blocked: 本地 master 超前未推成（pre-push 闸或网阻）——merge 在仓未上岸,归唤知会话查闸因"
			say "push 败——升件已落"
			exit 0 # 推败=尖未上岸——继续走部署=出处闸拒必升件（噪音类根修,stuck_esc#10657 实证）
		fi
	else
		HB_STATE=diverged
		esc "deploy_diverged" "部署/deploy_diverged: master 与 origin/master 分叉（本地=$LOCAL 远端=$REMOTE）——不自裁,归唤知会话"
		exit 0
	fi
fi
# image_dep_clean <repo相对路径> <dep包径集> → rc0=该件在 core 依赖闭包外
image_dep_clean() {
	local p="$1" dirs="$2" d
	while IFS= read -r d; do
		[ -n "$d" ] || continue
		case "$p" in "$d"/*|"$d") return 1;; esac
	done <<< "$dirs"
	return 0
}
# image_face_clean <deployed基线sha> → rc0=diff 全件证外（镜像面可跳零重启——
# 工单#deploy-pipe-tax③：go list -deps ./cmd/core 实算依赖闭包∪构建输入
# 硬列之外=o-core 二进制运行行为不可变；不可证空方向保守照换）。
image_face_clean() {
	local base="$1" files dirs p
	[ -n "$base" ] || return 1 # 基线缺（首部署）=不可证空照换
	git merge-base --is-ancestor "$base" "$HEAD" 2>/dev/null || return 1 # 非祖先=谱系岔不可证空
	files=$(git diff --name-only "$base" "$HEAD" 2>/dev/null) || return 1
	[ -n "$files" ] || return 1 # 空 diff 不应至此——不可证空
	local raw
	raw=$(go list -f '{{.ImportPath}}' -deps ./cmd/core 2>/dev/null) || return 1 # go list 败=闭包不完整→不可证空照换（半截闭包=假证空源）
	dirs=$(printf '%s\n' "$raw" | sed -n 's|^o/||p')
	[ -n "$dirs" ] || return 1 # go 面缺/空=不可证空照换
	printf '%s\n' "$dirs" | grep -qx 'cmd/core' || return 1 # 闭包不含主件自体=实算面失真不可证空
	while IFS= read -r p; do
		[ -n "$p" ] || continue
		p=${p##* -> } # rename 形取新名
		case "$p" in
			go.mod|go.sum|Dockerfile*|docker-compose*|.dockerignore) return 1;; # 构建输入硬列——碰即不证空
		esac
		image_dep_clean "$p" "$dirs" || return 1
	done <<< "$files"
	return 0
}
HEAD=$(git rev-parse master)
DEPLOYED=$([ -n "$DSN" ] && psql "$DSN" -qAt -c "SELECT (regexp_match(note,'o-core:([0-9a-f]+)'))[1] FROM ops_event WHERE seat='DEPLOY' AND event='部署' AND note LIKE '%rc=0%' AND note LIKE '%o-core:%' ORDER BY id DESC LIMIT 1" 2>/dev/null) # T-13283：行类滤先——deploy-oci 臂行同 seat/event/rc=0 无 o-core: 标记，取最新行后 extract 必空(假滞差→多余部署)
SHORT=${HEAD:0:12}
# ── 工单#25903 脏树连拍升件（拒部 esc_row 与滞差探并行不替）：deploy-gate
# 脏树拒* 行 45min(3拍窗)≥2→deploy_blocked 载 stray——连拍阈防瞬时脏
# （单拍即升被否）；行计数唤起面无关，每拍必经段置（早退径全覆）。
if [ -n "$DSN" ]; then
	DIRTYN=$(psql "$DSN" -qAt -c "SELECT count(*) FROM ops_event WHERE seat='DEPLOY' AND event='部署' AND note LIKE 'deploy-gate 脏树拒%' AND ts>now()-interval '45 minutes'" 2>/dev/null)
	case "$DIRTYN" in ''|*[!0-9]*) DIRTYN=0;; esac
	[ "$DIRTYN" -ge 2 ] && { HB_STATE=脏树连拍; esc "deploy_blocked" "部署/deploy_blocked: 脏树连拍拒部 ${DIRTYN} 次/45min stray=$(git status --porcelain 2>/dev/null | head -8 | tr '\n' ';' | cut -c1-240)——deploy_image 出处闸连拒,归唤知会话清障"; }
fi
if [ -z "$DEPLOYED" ] || [ "${HEAD:0:${#DEPLOYED}}" != "$DEPLOYED" ]; then
	# 工单#deploy-pipe-tax③镜像面谱系证空（对拍 deploy_oci arm_provably_clean
	# 先例——镜像面本无此检：见滞差即全量 rebuild+SG 重启钱径）：
	# diff DEPLOYED..HEAD 全件在 cmd/core 依赖闭包∪构建输入外=二进制
	# 运行行为不可变,落 o-core-clean 水位行跳本拍；水位不抵实部署
	# （下尖有证内件=照走锁/败避让/部署全段）。
	CLEANWM=$([ -n "$DSN" ] && psql "$DSN" -qAt -c "SELECT (regexp_match(note,'o-core-clean:([0-9a-f]+)'))[1] FROM ops_event WHERE seat='DEPLOY' AND event='部署' AND note LIKE '%o-core-clean:%' ORDER BY id DESC LIMIT 1" 2>/dev/null) # tolerate: DSN缺=空——退正常滞差部署
	if [ -n "$CLEANWM" ] && [ "${HEAD:0:${#CLEANWM}}" = "$CLEANWM" ]; then
		HB_STATE=clean水位; say "谱系证空水位在案(o-core-clean:$CLEANWM)——跳部署"
		exit 0
	fi
	if image_face_clean "$DEPLOYED"; then
		[ -n "$DSN" ] && psql "$DSN" -c "INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('DEPLOY','部署','','o-core-clean:$SHORT rc=0 base=$DEPLOYED 谱系证空——diff全件在cmd/core依赖闭包(go list -deps实算)∪构建输入外,镜像面零重启')" >/dev/null 2>&1 \
			|| say "水位行写败（不阻断）" # tolerate: 水位写败=下拍重判重发同一跳，不翻部署
		HB_STATE=证空跳部署; say "谱系证空——o-core-clean:$SHORT 水位落,SG零重启"
		exit 0
	fi
	# ── 工单#25760 o-core 换件部署窗（谱系证内=镜像必换+钱径重启）──
	# 窗外=入窗队列让位（本拍不换 o-core，继续走 OCI 臂面）；窗内=照走
	# 锁/败避让/部署全段。首次部署与紧急会话径不入窗（DEPLOYED 空=启机/
	# 灾备；会话 deploy_at_sha 直调 deploy_image 不归本闸）。
	OW_DEFER=0
	if [ -n "$DEPLOYED" ]; then
		OW_SPEC=$(cfg_get deploy.ocore_window_utc "19:00-21:00")
		OW_MAXH=$(cfg_get deploy.ocore_window_max_hold_h "26")
		case "$OW_MAXH" in ''|*[!0-9]*) OW_MAXH=26;; esac
		[ "$OW_MAXH" -ge 1 ] || OW_MAXH=26
		if ! ow_valid "$OW_SPEC"; then
			HB_STATE=窗配置非法
			esc "deploy_ocore_window_config" "部署/deploy_ocore_window_config: 窗配置非法 deploy.ocore_window_utc=$OW_SPEC（须 UTC HH:MM-HH:MM）——fail-closed 本拍不换钱径,归唤知会话修配置"
			OW_DEFER=1
			say "窗配置非法（$OW_SPEC）——fail-closed 本拍不换"
		elif ! ow_in_window "$OW_SPEC" "$NOW_EPOCH"; then
			OW_DEFER=1
			OW_QUEUED=$(psql "$DSN" -qAt -c "SELECT count(*) FROM ops_event WHERE seat='DEPLOY' AND event='部署' AND note LIKE 'ocore-window 入窗 sha=$SHORT%' AND ts>now()-interval '7 days'" 2>/dev/null)
			if [ "${OW_QUEUED:-0}" = 0 ]; then
				OW_NOTE=$(sqlq "ocore-window 入窗 sha=$SHORT base=$DEPLOYED win=$OW_SPEC UTC——谱系证内(镜像必换),窗外让位待窗放件")
				psql "$DSN" -c "INSERT INTO ops_event(seat,event,ticket,note)
					VALUES('DEPLOY','部署','','$OW_NOTE')" >/dev/null 2>&1 \
					|| say "入窗行写败（不阻断，下拍重写同事实）" # tolerate: 入窗行写败=事实面缺一行,下拍重写,不翻部署判
			fi
			# 越窗未放：最老待放 commit 龄>max_hold（日窗+跨度+漏一次检出面）→升件。
			OW_SINCE=$(git log --format=%ct "$DEPLOYED..$HEAD" 2>/dev/null | tail -1)
			if [ -n "$OW_SINCE" ]; then
				OW_AGE=$(( NOW_EPOCH - OW_SINCE ))
				if [ "$OW_AGE" -gt $(( OW_MAXH * 3600 )) ]; then
					esc "deploy_ocore_window_overdue" "部署/deploy_ocore_window_overdue: o-core换件越窗未放 待放=$SHORT 部署=$DEPLOYED 最老待放=$(date -u -d "@$OW_SINCE" +%m-%d\ %H:%M) 龄=$(( OW_AGE / 3600 ))h>max_hold=${OW_MAXH}h 窗=$OW_SPEC UTC——归唤知会话查deploy_watch/锁/窗配置"
				fi
			fi
			HB_STATE=窗外队列; say "窗外（窗=$OW_SPEC UTC）——入窗队列待放，本拍不换 o-core"
		fi
	fi
	if [ "$OW_DEFER" = 0 ]; then
		# 部署锁持=另径部署在途（会话 deploy_at_sha/本拍并发）——让位不撞锁
		# （撞锁 rc=1+败事件+deploy_blocked 升件=自伤型信号源，2026-11-10
		# 产线实证每次部署首试 rc=1→重试 rc=0 三连序列同型）。
		if ssh "${O_SG_HOST:-root@192.229.85.35}" "test -d /opt/o/.deploy-lock" 2>/dev/null; then
			HB_STATE=锁持让位; say "滞差但部署锁持（另径在途）——让位本拍"
			exit 0
		fi
		# 同 SHA 近窗败避让：DEPLOY 账内本 sha 近 15min 已有 rc!=0 试——
		# 让位本拍不落新行（重试风暴闸：唤源任意重复唤皆安全——
		# 2026-11-11 产线实证「HEAD 未推窗+事件唤链」rc=1 逐唤刷屏；
		# 窗期后真滞差照试不误发，升件侧 deploy_blocked 未销已抑复式）。
		if [ -n "$DSN" ]; then
			LASTFAIL=$(psql "$DSN" -qAt -c "SELECT id FROM ops_event WHERE seat='DEPLOY' AND event='部署' AND note LIKE '%o-core:${SHORT}%' AND note NOT LIKE '%rc=0%' AND ts>now()-interval '15 minutes' ORDER BY id DESC LIMIT 1" 2>/dev/null)
			[ -n "$LASTFAIL" ] && { HB_STATE=近窗败让位; say "同 sha 近窗部署败在案(#$LASTFAIL)——让位本拍"; exit 0; }
		fi
		# 首性审计整改：build-test CI 机械闸——master 可进伪戳/no-verify 件
		# （戳=本地文件可 touch,形闸/前缀=消息级可 spoof——首性审亲证），
		# 产线边界在此：check-runs 最新 build-test conclusion=success 才放
		# 部署。待闸/查询败=fail-closed 让位本拍；红=升件断供（闸不重置,
		# HEAD 换绿 sha 自解）。只罩真实部署径——谱系证空水位件无产线
		# 触碰,不耗 gh 调用。
		CIV=$(gh api "repos/myfxlogs/o/commits/$HEAD/check-runs?per_page=100" \
			--jq '.check_runs | map(select(.name=="build-test")) | sort_by(.id) | last | .status + " " + (.conclusion // "")' 2>/dev/null)
		case "$CIV" in
			"completed success") ;;
			"completed failure"|"completed cancelled"|"completed timed_out"|"completed startup_failure"|"completed action_required")
				HB_STATE=CI红断
				esc "deploy_ci_red" "部署/deploy_ci_red: master=$SHORT build-test conclusion='${CIV##* }'——CI 红件禁部署,闸不重置（HEAD 换绿 sha 自解）——归唤知会话修/revert"
				exit 0 ;;
			*)
				HB_STATE=CI待闸
				HAGE=$(git log -1 --format=%ct "$HEAD" 2>/dev/null); HAGE=$(( NOW_EPOCH - ${HAGE:-$NOW_EPOCH} ))
				[ "$HAGE" -gt 5400 ] && esc "deploy_ci_pending" "部署/deploy_ci_pending: master=$SHORT build-test 未绿(check='${CIV:-查询败/缺席}')待闸 $(( HAGE/60 ))min>90——部署挂起,归唤知会话查 gh auth/CI 断供"
				say "build-test 未绿（${CIV:-查询败/缺席}）——待闸让位本拍"
				exit 0 ;;
		esac
		say "滞差: 部署=$DEPLOYED vs master=$SHORT——起部署"
		out=$(bash scripts/deploy_image.sh 2>&1); rc=$?
		echo "$out" | tail -3 | while read -r l; do say "$l"; done
		if [ $rc -ne 0 ]; then # T-13283：&& 短路作末句=test 假值传染脚本 exit 1——成功部署也标 unit FAILURE（systemd 噪音掩真败）
			# 工单#16633 写手自证：败时刻 git status --porcelain 落升件 note——
			# 脏树写者归因不再靠事后猜（历史两窗 1dfaa8fcd/67ae07f90f41 写者
			# 未锁定实证）；截 8 行/240 字防爆 note 面，树径同载（多树并发归因键）。
			TOP=$(git rev-parse --show-toplevel 2>/dev/null)
			DIRTY=$(git status --porcelain 2>/dev/null | head -8 | tr '\n' ';' | cut -c1-240)
			HB_STATE=部署败
			esc "deploy_blocked" "部署/deploy_blocked: deploy_image rc=$rc（master=$SHORT 待部署）尾=$(echo "$out" | tail -2 | tr '\n' ' ' | cut -c1-120) 树=${TOP:-?} 脏=${DIRTY:-空}——归唤知会话清障（脏树/锁/网）"
			exit 1
		fi
		HB_STATE=部署成
	fi
else
	HB_STATE=已同步
	say "已同步（部署=$DEPLOYED = master）"
fi
# ── OCI 臂滞后探 → 自动换件（工单#25481：原案=纯探测>24h 升件等唤知——
# univ-expand 实证静默窗内 collecto 陈 ~10h 扩集特性空转。改直调径：
# 臂部署账 sha 非 HEAD 祖先面照走三闸（无账/不可解/分叉→升件）；
# 账陈先过 deploy-faces.tsv 谱系判（arm_stale——diff 全件证外=簿记面
# 陈化零烧，静默跳）；证内碰臂→本拍直调 deploy_oci.sh（其内谱系自检
# 复核+部署锁/环约对账/健康闸/败回滚全在）；换败→esc 即火（未销期间
# 让位不连试=重试风暴闸同 15min 镜像面避让意）；直调后账仍陈
# （内检证空快退=盒件比账鲜的簿记漂移/连试连败）→>24h 滞后升件揭之。
if [ -n "$DSN" ]; then
	source scripts/arm_faces_lib.sh
	for arm in collecto projector research; do
		ASHA=$(psql "$DSN" -qAt -c "SELECT (regexp_match(note,'deploy-oci ([0-9a-f]+)'))[1] FROM ops_event WHERE seat='DEPLOY' AND event='部署' AND note LIKE 'deploy-oci %rc=0%' AND note ~ ('arms=[^|]*$arm([^a-z]|$)') ORDER BY id DESC LIMIT 1" 2>/dev/null)
		if [ -z "$ASHA" ]; then
			esc "oci_arm_lag_$arm" "部署/oci_arm_lag: OCI臂$arm 无 deploy-oci rc=0 账行——换拍凭据缺席不可证陈/鲜,归唤知会话"
			continue
		fi
		if ! git rev-parse --verify -q "$ASHA^{commit}" >/dev/null 2>&1; then
			esc "oci_arm_lag_$arm" "部署/oci_arm_lag: OCI臂$arm 部署账 sha=$ASHA 本地不可解——镜像账/仓面对不上,归唤知会话"
			continue
		fi
		[ "$(git rev-parse "$ASHA")" = "$HEAD" ] && continue
		git merge-base --is-ancestor "$ASHA" "$HEAD" 2>/dev/null || {
			esc "oci_arm_lag_$arm" "部署/oci_arm_lag: OCI臂$arm 部署 sha=$ASHA 非 master 祖先——漂移分叉不自裁,归唤知会话"; continue; }
		base=$(git merge-base "$ASHA" "$HEAD" 2>/dev/null || true) # tolerate: mb 败=谱系岔不可证空→arm_stale 不询落直调保守换（deploy_oci 内检再复核），非静默省件
		if [ -n "$base" ] && ! arm_stale "$arm" "$base"; then
			continue # 谱系证空——陈化属簿记面（证外件动的账）非二进制差，静默
		fi
		if ! esc_pending "oci_arm_deploy_$arm"; then
			say "臂$arm 谱系证内陈化(账=$ASHA vs master=$SHORT)——直调 deploy_oci"
			out=$(bash scripts/deploy_oci.sh "$arm" 2>&1); rc=$?
			echo "$out" | tail -3 | while read -r l; do say "$l"; done
			if [ $rc -ne 0 ]; then
				# 工单#25658：锁竞争签名（deploy_oci die 两形——「部署锁占用 age<1800s」
				# /「抢锁败」）=他席活锁在途,非真败——退避让位下拍自然接管,不升件
				# （真败仍被下行 >24h lag 侦兜,不沉）。串臂每臂撞锁各升一件=
				# 自伤信号源实证。
				case "$out" in
					*部署锁占用*|*抢锁败*) say "臂$arm 撞部署锁（他席在途）——退避让位下拍";;
					*) esc "oci_arm_deploy_$arm" "部署/oci_arm_deploy: 自动换臂$arm rc=$rc（master=$SHORT 待换）尾=$(echo "$out" | tail -2 | tr '\n' ' ' | cut -c1-120)——归唤知会话清障（锁/ssh路由/环约缺键/健康闸）";;
				esac
				continue
			fi
			ASHA=$(psql "$DSN" -qAt -c "SELECT (regexp_match(note,'deploy-oci ([0-9a-f]+)'))[1] FROM ops_event WHERE seat='DEPLOY' AND event='部署' AND note LIKE 'deploy-oci %rc=0%' AND note ~ ('arms=[^|]*$arm([^a-z]|$)') ORDER BY id DESC LIMIT 1" 2>/dev/null)
			[ -n "$ASHA" ] && [ "$(git rev-parse "$ASHA" 2>/dev/null)" = "$HEAD" ] && continue
		fi
		SINCE=$(git log --format=%ct "$ASHA..$HEAD" 2>/dev/null | tail -1)
		if [ -n "$SINCE" ] && [ $(( $(date +%s) - SINCE )) -gt 86400 ]; then
			esc "oci_arm_lag_$arm" "部署/oci_arm_lag: OCI臂$arm 陈化>24h——部署=$ASHA vs master=$SHORT,缺件最早 $(date -d @$SINCE +%m-%d\ %H:%M) 起(自动换件连败或账盒漂移),归唤知会话deploy_oci"
		fi
	done
fi
exit 0
