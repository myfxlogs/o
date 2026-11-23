#!/usr/bin/env bash
# land_queue.sh —— 落地队列（工单#22577 / D-0048 裁定③）：
# 「复审过的支→master」机械化——models 定时件（deploy/o-land-queue.timer，
# 与 deploy_watch 同拍 15min）。候选支（fix|feat|t|audit|revfix|hotfix|canon/*，
# 尖 7d 内由 for-each-ref--no-merged 域界）三件事实齐即落：
#   (a) 支尖 check_all 绿戳龄≤6h（git-common-dir/check-green-<tip全sha>）；
#   (b) ops_event 有「复审过 sha=<尖全sha>」行且写行席≠该支完工回执席——
#       docs-only 差分支（diff ⊆ docs/**|skills/**|*.md，check_all
#       docs-only 同口径）免 b；
#   (c) 对 origin/master 可净合（merge-tree 预检+工树 merge --no-ff 实证；
#       净合前另过 depends_on 时序闸——支新增迁移所依须已在
#       origin/master 或同支自带，未合跳支不占落地账，工单#25800）。
# 落法=独立工树（非部署树）merge --no-ff→合并点跑 check_all→盖戳→
# 部署树发 <merge_sha>:master（pre-push master 只许 /opt/o 径）→清工树；
# deploy_watch 照旧 ff+部署。单写者=本件。
# 列车形（工单O-32267）：一拍取全部要件齐支（上界 TRAIN_MAX=5，
# committerdate 升序）串成一列——master 依次 --no-ff 合各支尖→
# 单棵树→一次合并点 check_all/一次推/一次 CI/一次部署；列内任支
# 合不净=剔出该支（landq_merge_conflict 升件径复用）其余支重列续走，
# 迁移依赖列车感知（列内先序支自带迁移计已合面，合并时对实合 HEAD
# 复核兜底——前序支被剔则依赖落空支同剔不合缺依迁移）；每支落成行
# 照旧载支名+尖+merge sha 审计粒度不丢；单支列车退化为现行一支一落。
# 红/冲突=不落+落地事实行（不升件不立票）；同尖败冷却 24h（落地行自载账）。
#
# 用法：scripts/land_queue.sh（无参；o-land-queue.timer 每 15min 唤）
# 桩测缝：O_LAND_REPO（仓径，缺省 /opt/o）/O_LAND_PSQL/O_LAND_DSN/
#   O_LAND_CHECK_ALL（合并点闸命令，缺省 bash scripts/check_all.sh）/
#   O_LAND_BOOTSTRAP/O_LAND_PUSH/O_LAND_WT。
set -uo pipefail

REPO="${O_LAND_REPO:-/opt/o}"
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN="${O_LAND_DSN:-${FIN_PG_DSN:-$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')}}"
PSQL="${O_LAND_PSQL:-psql}"
CHECK_ALL="${O_LAND_CHECK_ALL:-bash scripts/check_all.sh}"
BOOTSTRAP="${O_LAND_BOOTSTRAP:-bash scripts/bootstrap_worktree.sh}"
PUSH_CMD="${O_LAND_PUSH:-bash scripts/git_push.sh}"
WTBASE="${O_LAND_WT:-/tmp}"
LOCK="${O_LAND_LOCK:-/tmp/o-land-queue.lock}"
TRAIN_MAX=5 # 列车上界写死（工单O-32267 spec——N 封顶防一拍巨列放大爆炸半径）
WT="" PROV_F=""

# go 工具径自足（工单#25880）：合并点 check_all 的唤起面不定——正式
# systemd 单元 Environment 有 go 径，transient systemd-run/手工径最小
# PATH 缺 /usr/local/go/bin（实证 o-land-drain 2026-11-20 05:42Z 四支
# merge_red 同因 check_all.sh:87 go:command not found，非支病灶）。
# PATH 缺口在唤起面自足补齐——禁依赖外部 unit env 声明在场。
if ! command -v go >/dev/null 2>&1; then
	for _gobin in /usr/local/go/bin /usr/lib/go/bin /snap/go/current/bin 		"$HOME/go/bin" "$HOME/sdk/go/bin" /opt/go/bin; do
		if [ -x "$_gobin/go" ]; then PATH="$_gobin:$PATH"; export PATH; break; fi
	done
	unset _gobin
fi

say() { echo "[land_queue] $*"; }
sqlq() { printf '%s' "$1" | sed "s/'/''/g"; } # tolerate: sed 恒 rc0——纯转义器非闸
q() { # $1=sql → stdout（DSN/psql 败=空——调用方按空判不猜）
	[ -n "$DSN" ] || return 0
	"$PSQL" "$DSN" -qAt -c "$1" 2>/dev/null
}
fact() { # 落地事实行——seat=落地件 event=落地；$1=note（不升件不立票）
	local n; n=$(sqlq "$1")
	[ -n "$DSN" ] || { say "（无DSN——事实行未落账）$1"; return 0; }
	"$PSQL" "$DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('落地件','落地','','$n')" >/dev/null 2>&1 \
		|| say "事实行写败（不阻断）——$1" # tolerate: 事实行写败下拍补，主径不依账
}
esc() { # 可处置败类升件——无头世界静默晾死根修（工单#deploy-pipe-tax②）：
	# 同 deploy_watch escEdgeU dedup 语义（同 scope 24h 内未销才抑）。
	# $1=scope（按支分词——多支同卡各自升件不互抑） $2=note
	local scope="$1" note="$2" s n
	s=$(sqlq "$scope"); n=$(sqlq "$note")
	[ -n "$DSN" ] || return 0
	q "SELECT count(*) FROM ops_event e WHERE e.event='升件' AND e.esc_scope='$s'
		AND e.ts>now()-interval '24 hours'
		AND NOT EXISTS (SELECT 1 FROM ops_event x2 WHERE x2.ts>e.ts
			AND x2.note ~ ('升件#'||e.id::text||'([^0-9]|$)'))" | grep -qx 0 \
	&& "$PSQL" "$DSN" -c "INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('探测席','升件','','$n','部署','$s')" >/dev/null 2>&1 || true # tolerate: 升件写败只剩下拍重试，无二级径
}

cleanup() {
	[ -n "$WT" ] || { [ -n "$PROV_F" ] && rm -f "$PROV_F"; PROV_F=""; return 0; }
	git -C "$REPO" worktree remove --force "$WT/wt" >/dev/null 2>&1 || true # tolerate: 清树败=残件待 prune，不阻断收口
	rm -rf "$WT" 2>/dev/null || true # tolerate: rm 败=tmp 面系统代谢兜底
	WT=""
	[ -n "$PROV_F" ] && rm -f "$PROV_F" 2>/dev/null; PROV_F="" # tolerate: 临时件残留无害
}
trap cleanup EXIT

exec 9>"$LOCK"
flock -n 9 || { say "他实例在途——让位本拍"; exit 0; }

cd "$REPO" || { say "仓径不可达 $REPO"; exit 0; }
git fetch origin -q --prune 2>/dev/null || { say "fetch 败——远端真态不可知退出"; exit 0; }

# 部署树基线闸：本地 master 须=origin/master——部署树尖未上岸时推
# sha:master 必造 deploy_diverged 自伤（双席并推竞态面），让位下拍。
LOCAL_M=$(git rev-parse master 2>/dev/null) # tolerate: master 缺=空串落下行判拒
REMOTE_M=$(git rev-parse origin/master 2>/dev/null) # tolerate: 同上
[ -n "$LOCAL_M" ] && [ "$LOCAL_M" = "$REMOTE_M" ] \
	|| { say "部署树 master≠origin/master——让位合流/盖戳流"; exit 0; }

recent_fail() { # $1=tip全sha → rc0=24h内有「落地败」行（同尖冷却）
	[ -n "$DSN" ] || return 1
	q "SELECT 1 FROM ops_event WHERE seat='落地件' AND event='落地'
		AND note ~ '落地败' AND note ~ '$1'
		AND ts>now()-interval '24 hours' LIMIT 1" | grep -q .
}
stamp_ok() { # $1=tip全sha → rc0=鲜绿戳在（≤6h，同 pre-push 戳窗）
	local gd stamp age
	gd=$(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null) || return 1
	stamp="$gd/check-green-$1"
	[ -f "$stamp" ] || return 1
	age=$(( $(date +%s) - $(stat -c %Y "$stamp" 2>/dev/null || echo 0) )) # tolerate: stat 败=0→age 巨大判陈 fail-closed
	[ "$age" -le 21600 ]
}
docs_only_diff() { # $1=tip → rc0=差分全中 docs/**|skills/**|*.md（check_all docs-only 同口径）
	local mb p
	mb=$(git merge-base origin/master "$1" 2>/dev/null) || return 1
	[ -n "$mb" ] || return 1
	while IFS= read -r p; do
		[ -n "$p" ] || continue
		case "$p" in docs/*|skills/*|*.md) ;; *) return 1 ;; esac
	done < <(git diff --name-only "$mb" "$1" 2>/dev/null)
	return 0
}
review_ok() { # $1=br名 $2=tip全sha → rc0=复审过行在且写行席≠完工/候审/施工收口回执席
	# rc 分因（O-35491⑤/源票#35488——跳支日志三因合并串致唤知直造支
	# 「回执缺」与「复审缺」不可分诊）：rc2=无DSN rc3=完工回执缺/作者席
	# 不可证（分席不可证 fail-closed）rc4=复审过行缺/同席复审
	local br="$1" tip="$2" bre aseat
	[ -n "$DSN" ] || return 2
	bre=$(printf '%s' "$br" | sed 's/[][\\.*^$()+?{}|]/\\&/g') # tolerate: sed 恒 rc0——纯转义器非闸
	aseat=$(q "SELECT seat FROM ops_event
		WHERE ((note ~ '(完工回执|完工推支候审|候审回执|施工收口)' OR event ~ '完工')
			AND note !~ '^复审过[[:space:]]*sha=' AND seat NOT IN ('复审席','落地件')
			AND (note ~ ('$bre' || '([@[:space:]=]|\$)') OR note ~ '${tip:0:9}'))
		ORDER BY id DESC LIMIT 1")
	[ -n "$aseat" ] || return 3 # 作者席不可考=分席不可证→fail-closed
	aseat=$(printf '%s' "$aseat" | tr -d "\"'\\")
	[ "$aseat" = "落地件" ] && return 3
	q "SELECT 1 FROM ops_event
		WHERE note ~ '^复审过[[:space:]]*sha=[[:space:]]*${tip}'
		AND seat <> '$aseat' LIMIT 1" | grep -q . || return 4
	return 0
}
mergeable() { # $1=tip → rc0=对 origin/master 净合（merge-tree 预检，不动工树）
	git merge-tree --write-tree origin/master "$1" >/dev/null 2>&1
}
mig_dep_miss() { # $1=tip $2=base-ref [$3=provides文件] → stdout=缺依清单; rc0=齐 rc1=缺
	# 列车感知时序闸核心（工单O-32267）：base=已合基线（origin/master 或
	# 列车实合 HEAD sha），provides=列内先序支自带迁移 basename 文件。
	# 解析件取支尖自带版——deploy 树未含本闸件时亦能判（渐进启用）。
	local tip="$1" base="$2" prov="${3:-}" mb added libtmp miss rc
	mb=$(git merge-base origin/master "$tip" 2>/dev/null) || return 0 # tolerate: base 败=放——mergeable 已预检此支可合域
	added=$(git diff --name-only --diff-filter=A "$mb" "$tip" -- 'migrations/*.sql' 2>/dev/null)
	[ -n "$added" ] || return 0
	libtmp=$(mktemp /tmp/migdep-lib-XXXXXX 2>/dev/null) || return 0 # tolerate: mktemp 败=放闸下拍——支自带 check_all⑨已保底声明合法
	if ! git show "$tip:scripts/mig_dep_lib.sh" > "$libtmp" 2>/dev/null; then
		rm -f "$libtmp"; return 0 # tolerate: 支无解析件=无声明语义——放
	fi
	. "$libtmp"
	if [ -n "$prov" ]; then
		miss=$(printf '%s\n' "$added" | mig_dep_gate_check "$tip" "$base" "$prov" 2>/dev/null)
	else
		miss=$(printf '%s\n' "$added" | mig_dep_gate_check "$tip" "$base" 2>/dev/null)
	fi
	rc=$?
	rm -f "$libtmp"
	[ "$rc" -eq 0 ] && return 0
	printf '%s' "$miss"
	return 1
}
mig_dep_gate() { # $1=br名 $2=tip [$3=provides文件] → rc0=依赖序合法；rc1=所依未合（say 记跳因）
	local br="$1" tip="$2" prov="${3:-}" miss rc
	miss=$(mig_dep_miss "$tip" origin/master "$prov"); rc=$?
	[ "$rc" -eq 0 ] && return 0
	say "跳 $br——迁移依赖未合：$(printf '%s' "$miss" | tr '\n' ' ' | sed 's/ $//')（目标须先在 origin/master/列内先序支或同支自带）"
	return 1
}

# ── 扫支：七日内未并支按尖龄升序，要件齐支入列车（上界 TRAIN_MAX）──
CANDS=$(git for-each-ref --no-merged origin/master --sort=committerdate \
	--format='%(committerdate:unix) %(refname:short) %(objectname)' \
	'refs/remotes/origin/fix/*' 'refs/remotes/origin/feat/*' \
	'refs/remotes/origin/t/*' 'refs/remotes/origin/audit/*' \
	'refs/remotes/origin/revfix/*' 'refs/remotes/origin/hotfix/*' \
	'refs/remotes/origin/canon/*' 2>/dev/null \
	| awk -v c=$(( $(date +%s) - 604800 )) '$1>c {print $2,$3}' \
	| sed 's|^origin/||')

# 列车构造（工单O-32267）：要件三件闸语义不动（戳+复审+净合逐支同判），
# 只改「取一支」为「取至多 TRAIN_MAX 支」——列内依赖面=列内先序支
# 自带迁移（provides 文件逐支累积，合并时对实合 HEAD 复核兜底）。
TRAIN=""; STRANDED=""; TN=0; PROV_F=$(mktemp /tmp/lq-prov-XXXXXX 2>/dev/null || echo "") # tolerate: mktemp 败=空串→provides 闸退化为仅 master/同支（保守向）
: > "$PROV_F" 2>/dev/null || PROV_F="" # tolerate: 建文件败=同上退化
while read -r br tip; do
	[ -n "$br" ] && [ -n "$tip" ] || continue
	recent_fail "$tip" && { say "跳 $br——同尖24h败冷却"; continue; }
	stamp_ok "$tip" || { say "跳 $br——戳缺/戳陈"; STRANDED="$STRANDED $br"; continue; }
	if docs_only_diff "$tip"; then
		say "$br docs-only 差分——免复审行(b)"
	else
		review_ok "$br" "$tip"; _rrc=$?
		if [ "$_rrc" -ne 0 ]; then
			case "$_rrc" in
				2) say "跳 $br——无DSN(复审要件不可读)" ;;
				3) say "跳 $br——完工回执缺/作者席不可证(唤知直造支分席不可证→fail-closed)" ;;
				4) say "跳 $br——复审过行缺/同席复审" ;;
				*) say "跳 $br——复审要件未过(rc=$_rrc)" ;;
			esac
			STRANDED="$STRANDED $br"; continue
		fi
	fi
	mig_dep_gate "$br" "$tip" "$PROV_F" || { STRANDED="$STRANDED $br(依赖未合)"; continue; }
	if ! mergeable "$tip"; then
		fact "落地败 支=$br 尖=$tip 因=对origin/master不可净合(merge-tree冲突)——支须重基,同尖冷却24h"
		esc "landq_conflict_${br//\//_}" "部署/landq_merge_conflict: 支=$br 尖=$tip 对origin/master不可净合——须重基或人工裁决,支入24h冷却待处置"
		continue
	fi
	TRAIN="$TRAIN$br $tip
"
	# 入列支新增迁移 basename 并入 provides——供列内后续支依赖判
	_tmb=$(git merge-base origin/master "$tip" 2>/dev/null) # tolerate: 败=空→不产 provides 行
	[ -n "$_tmb" ] && [ -n "$PROV_F" ] \
		&& git diff --name-only --diff-filter=A "$_tmb" "$tip" -- 'migrations/*.sql' 2>/dev/null \
			| sed 's|.*/||' | grep . >> "$PROV_F" # tolerate: 追加败=provides 缺面→后续支依赖闸保守拒(安全向)
	TN=$((TN+1))
	[ "$TN" -ge "$TRAIN_MAX" ] && break # 列车上界——超出支下拍重列（spec N=5 写死）
done <<< "$CANDS"
TRAIN_N=$TN

# ── 补扫:扫描面外未并支一律入滞留集（同日实证:cmcweb无名单前缀且仅
# 本地、canon/*无前缀、macro-events超7d窗——进了扫描才有"跳"录,
# 面外支连跳因都不落,永不可见）。凡本地或远端未并 origin/master
# 且本轮未经候选评估之支,不问要件齐否皆滞留;脱困=改名入名单前缀/
# 推远端并补要件/判废清支。
SEEN=" "
while read -r ebr _etip; do
	[ -n "$ebr" ] && SEEN="$SEEN$ebr "
done <<< "$CANDS"
# 在办支剔集：被任一 worktree checkout 之支=在办非滞留(worktreepath 原子跨
# 版本不稳,porcelain 支行取——detached 工树无 branch 行天然不误剔)
WTACTIVE=$(git worktree list --porcelain 2>/dev/null \
	| awk '/^branch refs\/heads\//{sub("refs/heads/","",$2);print $2}' \
	| paste -sd' ' -) # 扁单行——多行 case " x " 空格定界失配漏剔(桩测实证)
while IFS= read -r ob; do
	case " $SEEN $WTACTIVE " in *" $ob "*) continue;; esac
	STRANDED="$STRANDED $ob(面外)"
done < <(git for-each-ref --no-merged origin/master --format='%(refname:short)' \
	'refs/heads/' 'refs/remotes/origin/' 2>/dev/null \
	| sed 's|^origin/||' | grep -vxe master -vxe HEAD | sort -u)

# 滞留支可见化（业主令2026-11-21断链根修——audit双修复支戳/复审缺在岸
# 静默滞留实证:#27164/#27183 修复已推origin但无戳无复审,三事实闸外
# 只 say 落日志=无人见的滞留）：缺戳或缺复审支集名件升件,归唤知会话
# 补戳+复审或判废清支——land_queue 只喊不办,要件齐自落。节流=滞留集
# 覆盖判（业主令2026-11-21②——旧「20h同scope零行才再发」粗节流把新
# 滞留支并进旧升件影子=候审支零升件零事驱,实证:07:15 audit/1121-life
# 升件后 fix/forensic-sim-finro 候审支当拍静默）:逐名对拍最近一次升件
# note——有支名未被覆盖即升件（事驱=滞留集变化）;全集已覆盖且<6h 不升
# （同集续滞防自喂）,≥6h 同集重升(轮询兜底——绑绿戳6h鲜度窗:滞留过窗戳反正陈,再喊边际成本为零)
if [ -n "$STRANDED" ] && [ -n "$DSN" ]; then
	LAST_NOTE=$(q "SELECT note FROM ops_event WHERE event='升件'
		AND esc_scope='landq_starved' ORDER BY id DESC LIMIT 1" 2>/dev/null)
	LAST_AGE_S=$(q "SELECT COALESCE(EXTRACT(EPOCH FROM now()-max(ts))::int,99999)
		FROM ops_event WHERE event='升件' AND esc_scope='landq_starved'" 2>/dev/null)
	NEED=0
	for sb in $STRANDED; do
		case "$LAST_NOTE" in *"${sb%%(*}"*) ;; *) NEED=1;; esac # 剔(面外)/(依赖未合)标缀取裸支名对拍
	done
	[ "$NEED" = 0 ] && [ "${LAST_AGE_S:-99999}" -lt 21600 ] || NEED=1
	if [ "$NEED" = 1 ]; then
		SLIST=$(printf '%s\n' $STRANDED | paste -sd, -)
		# note 整体过 sqlq——支名=会话可控输入,refname 合法含引号/分号
		# (复审驳#1实证:裸拼注入面可伪造复审过行绕闸),与 esc/fact 同径
		SNOTE=$(sqlq "部署/landq_starved: 未并支滞留——缺落地要件(戳缺/戳陈或复审过行缺)或扫描面外(无前缀/仅本地/超窗,标面外)或迁移依赖未合:$SLIST——归会话补戳+复审/改名入面推远端/判废清支;要件齐land_queue自落勿手工合")
		"$PSQL" "$DSN" -c "INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
			VALUES('探测席','升件','','$SNOTE','部署','landq_starved')" >/dev/null 2>&1 \
			|| say "landq_starved 升件写败（不阻断）"
	fi
fi

[ "$TRAIN_N" -gt 0 ] || { say "无三件齐候选——本拍空"; exit 0; }
say "列车 $TRAIN_N 支——起合（committerdate升序,上界$TRAIN_MAX）"

# ── 落法：独立工树 merge --no-ff 逐支串列→单树 check_all→一次推 ──
WT=$(mktemp -d "${WTBASE}/o-land-XXXXXX")
if ! git worktree add --detach "$WT/wt" origin/master >/dev/null 2>&1; then
	fact "落地暂缓 列车$TRAIN_N支 因=独立工树建立败"
	exit 0
fi

# 逐支串列：列内任支合不净=剔出该支（落地败行+landq_merge_conflict
# 升件径复用）其余支重列续走——一炸不全列死（工单O-32267 spec②）。
# 迁移依赖合并时复核：base=实合 HEAD 演进（origin/master→逐支 merge
# sha），前序支被剔则依赖落空支同剔不合缺依迁移。
LANDED=""; CUR_BASE="origin/master"
while read -r br tip; do
	[ -n "$br" ] && [ -n "$tip" ] || continue
	# 列车内迁移依赖复核（对实合面——provides 仅构造期乐观面，此处为准）
	_tmb=$(git merge-base origin/master "$tip" 2>/dev/null) # tolerate: 败=空→跳过复核放行(支自带 check_all⑨保底)
	_tadded=""
	[ -n "$_tmb" ] && _tadded=$(git diff --name-only --diff-filter=A "$_tmb" "$tip" -- 'migrations/*.sql' 2>/dev/null)
	if [ -n "$_tadded" ]; then
		_tlib=$(mktemp /tmp/migdep-lib-XXXXXX 2>/dev/null) || _tlib="" # tolerate: mktemp 败=_tlib空→复核跳过放行
		if [ -n "$_tlib" ] && git show "$tip:scripts/mig_dep_lib.sh" > "$_tlib" 2>/dev/null; then
			_tmiss=$(cd "$WT/wt" && . "$_tlib" && printf '%s\n' "$_tadded" | mig_dep_gate_check "$tip" "$CUR_BASE" 2>/dev/null); _trc=$?
			rm -f "$_tlib"
			[ "$_trc" -ne 0 ] && [ -n "$_tmiss" ] && {
				fact "落地暂缓 支=$br 尖=$tip 因=列车内迁移依赖落空(前序支被剔/未合):$_tmiss——下拍重扫"
				say "跳 $br——列车内依赖落空"
				continue
			}
		else
			[ -n "$_tlib" ] && rm -f "$_tlib"
		fi
	fi
	git merge-base --is-ancestor "$tip" origin/master 2>/dev/null \
		&& { fact "落地暂缓 支=$br 尖=$tip 因=尖已被他径并入master——候选态竞动,下拍重扫"; continue; }
	SUBJ=$(git log -1 --format=%s "$tip" 2>/dev/null | cut -c1-80) # tolerate: log 败=空串——subject 缺省不阻断合流
	TRAILERS=$(git log --format=%B "$CUR_BASE..$tip" 2>/dev/null \
		| grep -E '^(坑处置:|免疫件:)' | sort -u) # tolerate: grep 空集=支尖无trailer合法径
	MSG="merge: land-queue $br——${SUBJ}
落地件机械合（工单#22577/D-0048③+O-32267列车）：三件齐——鲜戳+复审过 sha=$tip+净合；列车第$(( $(printf '%s' "$LANDED" | grep -c .) + 1 ))支；
deploy_watch 照旧 ff+部署。"
	if [ -n "$TRAILERS" ]; then
		MSG="$MSG

承袭支尖：
$TRAILERS"
	fi
	MSG="$MSG

坑处置:合流机械步——判词承支尖commit体，无新坑"
	if ! (cd "$WT/wt" && git merge --no-ff -m "$MSG" "$tip" >/dev/null 2>&1); then
		(cd "$WT/wt" && git merge --abort >/dev/null 2>&1) || true # tolerate: abort 败=本不在merge态——rc1兜底
		fact "落地败 支=$br 尖=$tip 因=工树merge冲突(预检后竞动/列内互冲)——同尖冷却24h"
		esc "landq_conflict_${br//\//_}" "部署/landq_merge_conflict: 支=$br 尖=$tip 工树merge冲突(列车内)——须人工裁决,支入24h冷却"
		continue
	fi
	MSHA=$(git -C "$WT/wt" rev-parse HEAD 2>/dev/null) # tolerate: rev-parse 败=空串落下行戳验拒
	[ -n "$MSHA" ] || { fact "落地败 支=$br 尖=$tip 因=merge后HEAD不可解"; continue; }
	[ "$(git -C "$WT/wt" rev-parse 'HEAD^2' 2>/dev/null)" = "$tip" ] \
		|| { fact "落地暂缓 支=$br 尖=$tip 因=merge未成真合(Already-up-to-date竞动)——下拍重扫"; continue; }
	LANDED="$LANDED$br $tip $MSHA
"
	CUR_BASE="$MSHA"
	say "列车合 $br → $MSHA"
done <<< "$TRAIN"

LN=$(printf '%s' "$LANDED" | grep -c . || true) # tolerate: grep空集rc1——零落=0合法
[ "$LN" -gt 0 ] || { say "列车全剔——本拍无落"; exit 0; }
FINAL=$(git -C "$WT/wt" rev-parse HEAD 2>/dev/null) # tolerate: 败=空串落下行戳验拒
[ -n "$FINAL" ] || { fact "落地败 列车$LN支 因=串列后HEAD不可解"; exit 0; }

# ── 单树一次验：快径=合并树≡末支尖树（末支已含全列内容=已戳件戳随树
#    迁，工单#deploy-pipe-tax①同判逐支适用——树同判据不变）；否则合并
#    点 check_all 一次（列车全列共享一次 CI——spec①省 N-1 次串行验）──
GD=$(git -C "$WT/wt" rev-parse --path-format=absolute --git-common-dir 2>/dev/null) # tolerate: 败=空串落下行戳验拒
MTREE=$(git -C "$WT/wt" rev-parse 'HEAD^{tree}' 2>/dev/null)               # tolerate: 败=空串→不走快径退全验
LAST_TIP=$(printf '%s' "$LANDED" | tail -1 | awk '{print $2}')
TTREE=$(git rev-parse "$LAST_TIP^{tree}" 2>/dev/null)                      # tolerate: 同上
if [ -n "$MTREE" ] && [ -n "$TTREE" ] && [ "$MTREE" = "$TTREE" ] \
	&& [ -n "$GD" ] && [ -f "$GD/check-green-$LAST_TIP" ]; then
	if cp "$GD/check-green-$LAST_TIP" "$GD/check-green-$FINAL" 2>/dev/null; then
		fact "落成快径 列车$LN支 尖末=$LAST_TIP merge=$FINAL——合并树≡末支尖树跳合并点重验(内容=已戳件,戳随树迁)"
	else
		fact "落地暂缓 列车$LN支 因=快径戳迁写败——下拍全验径"
		exit 0
	fi
else
	if ! (cd "$WT/wt" && bash -c "$BOOTSTRAP" >/dev/null 2>&1); then
		fact "落地败 列车$LN支 因=工树bootstrap败(gen面缺)"
		exit 0
	fi
	COUT=$(cd "$WT/wt" && bash -c "$CHECK_ALL" 2>&1) # tolerate: check_all 败=rc非零落下行判据——输出截尾入账
	CRC=$?
	if [ "$CRC" -ne 0 ]; then
		TMEM=$(printf '%s' "$LANDED" | awk '{printf "%s@%s ",$1,substr($2,1,9)}')
		TSCOPE="landq_merge_red_train_$(printf '%s' "$LANDED" | md5sum | cut -c1-8)" # tolerate: md5sum 败=空→scope 退化 landq_merge_red_train_（仍分桶）
		fact "落地败 列车$LN支 成员=$TMEM 因=合并点check_all红 尾=$(printf '%s' "$COUT" | tail -3 | tr '\n' ' ' | cut -c1-160)"
		esc "$TSCOPE" "部署/landq_merge_red: 列车$LN支($TMEM)合并点check_all红 尾=$(printf '%s' "$COUT" | tail -2 | tr '\n' ' ' | cut -c1-120)——列组合红须人工定性(拆列/修支/撤票)"
		exit 0
	fi
	[ -n "$GD" ] && [ -f "$GD/check-green-$FINAL" ] \
		|| { fact "落地败 列车$LN支 因=check_all绿而合并点戳缺(自相矛盾)"; exit 0; }
fi

# 推前对拍：check_all 窗内 origin/master 若已移，sha:master 非ff必拒——
# 先验祖先关系，移了落暂缓行下拍重列（不烧推送重试）。
git fetch origin master -q 2>/dev/null || true # tolerate: fetch 败=沿用旧ref推——pre-push远端自拒兜底
if ! git merge-base --is-ancestor origin/master "$FINAL" 2>/dev/null; then
	fact "落地暂缓 列车$LN支 因=check_all窗内master已移——下拍重列"
	exit 0
fi
ORIG_M=$(git rev-parse origin/master 2>/dev/null) # tolerate: 败=空→自动销区间退化跳过(销行写径另有巡检兜)
if (cd "$REPO" && bash -c "$PUSH_CMD origin $FINAL:master" >/dev/null 2>&1); then
	say "列车落成 $LN 支 → master $FINAL"
	# 每支落成行照旧（支名+尖+merge sha 审计粒度不丢，spec③）
	while read -r br tip msha; do
		[ -n "$br" ] && fact "落成 支=$br 尖=$tip merge=$msha——deploy_watch接力ff+部署(列车$LN支)"
	done <<< "$LANDED"
	# 落成自动销票细胞（工单O-33182）：支名+commit面+合并commit面三源
	# 并集提 O-<id>/工单#<id> 号集→desk闸(未销∧未挂起∧非裁定)∧完工回执
	# 闸→seat=落地件 处置销行——「落成≠销案」簿记断尾根修（#32269/#32295
	# 码齐票挂实证）。取代旧「工单#机械合支随合即销」无闸径（工单#24624
	# 件——无回执强销/susp·rulable误销洞同修：多段epic干一半不冤杀）。
	# ORIG_M 空=区间退化整段跳（巡检旧径兜）。
	SEEN_TID=" "
	while [ -n "$ORIG_M" ] && read -r br tip msha; do
		[ -n "$br" ] || continue
		TSET=$( { printf '%s\n' "$br"
			git -C "$REPO" log --format=%B "$ORIG_M..$tip" 2>/dev/null
			git -C "$REPO" log -1 --format=%B "$msha" 2>/dev/null; } \
			| grep -oE '(^|[^[:alnum:]])O-[0-9]+|工单#[0-9]+' \
			| grep -oE '[0-9]+' | sort -un ) # tolerate: 提取面败/空集=跳支不销
		[ -n "$TSET" ] || continue
		CLOSESET=$(q "WITH nums AS (SELECT unnest(ARRAY[$(printf '%s' "$TSET" | paste -sd, -)]) AS tid)
			SELECT n.tid FROM nums n JOIN ops_ticket_desk d ON d.id=n.tid
			WHERE d.closed IS NOT TRUE AND d.susp_until IS NULL AND d.rulable IS NOT TRUE
			  AND EXISTS (SELECT 1 FROM ops_event r
			    WHERE (r.note ~ '(完工回执|完工推支候审|候审回执|施工收口)' OR r.event ~ '完工')
			      AND r.note !~ '^复审过[[:space:]]*sha=' AND r.seat NOT IN ('复审席','落地件')
			      AND (r.note ~ ('O-'||n.tid||'([^0-9]|$)') OR r.note ~ ('工单#'||n.tid||'([^0-9]|$)')
			        OR r.ticket ~ ('^(O-|工单#)?'||n.tid||'$')))")
		DID=""
		for tid in $CLOSESET; do
			case "$SEEN_TID" in *" $tid "*) continue;; esac # tolerate: case 空集不匹配=放行首见
			SEEN_TID="$SEEN_TID$tid "
			cn="工单#$tid 销——落成自动销 支=$br merge=$msha"
			# O-33705：DID 只在 INSERT 成功后追加——写败=票仍挂开，
			# ops_esc 升件载明（24h 前缀去重 per-tid）不再静默吞；
			# 汇总 csv 只含真销成 tid（称销未销假汇总行病灶根修）。
			if "$PSQL" "$DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
				VALUES('落地件','处置','工单','$(sqlq "$cn")')" >/dev/null 2>&1; then
				DID="$DID $tid"
			else
				q "SELECT ops_esc('落地件','工单#$tid','部署','landq_autoclose',
					'落成自动销写败 tid=$tid','支=$br merge=$msha——销行INSERT败票未销仍挂开,支已落不重试,值守/巡检人工处置')" >/dev/null
				say "自动销行写败 tid=$tid（升件已落/去重窗内）" # tolerate: 升件写败=值守面兜——支已落不重试
			fi
		done
		[ -n "$(printf '%s' "$DID" | tr -d '[:space:]')" ] \
			&& fact "落成自动销 支=$br 票=$(printf '%s' "$DID" | tr ' ' '\n' | sort -un | paste -sd, -)"
	done <<< "$LANDED"
else
	fact "落地暂缓 列车$LN支 因=sha:master推败(远端先行/闸拒)——下拍重列"
fi
exit 0
