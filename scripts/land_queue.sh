#!/usr/bin/env bash
# land_queue.sh —— 落地队列（工单#22577 / D-0048 裁定③）：
# 「复审过的支→master」机械化——models 定时件（deploy/o-land-queue.timer，
# 与 deploy_watch 同拍 15min）。候选支（fix|feat|t|audit|revfix|hotfix/*，
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
# deploy_watch 照旧 ff+部署。单写者=本件；一拍一支串行；红/冲突=不落+
# 落地事实行（不升件不立票）；同尖败冷却 24h（落地行自载账）。
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
WT=""

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
	[ -n "$WT" ] || return 0
	git -C "$REPO" worktree remove --force "$WT/wt" >/dev/null 2>&1 || true # tolerate: 清树败=残件待 prune，不阻断收口
	rm -rf "$WT" 2>/dev/null || true # tolerate: rm 败=tmp 面系统代谢兜底
	WT=""
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
review_ok() { # $1=br名 $2=tip全sha → rc0=复审过行在且写行席≠完工回执席
	local br="$1" tip="$2" bre aseat
	[ -n "$DSN" ] || return 1
	bre=$(printf '%s' "$br" | sed 's/[][\\.*^$()+?{}|]/\\&/g') # tolerate: sed 恒 rc0——纯转义器非闸
	aseat=$(q "SELECT seat FROM ops_event
		WHERE note ~ '(完工回执|完工推支候审|候审回执)' AND (note ~ '$bre' OR note ~ '${tip:0:9}')
		ORDER BY id DESC LIMIT 1")
	[ -n "$aseat" ] || return 1 # 作者席不可考=分席不可证→fail-closed
	aseat=$(printf '%s' "$aseat" | tr -d "\"'\\")
	[ "$aseat" = "落地件" ] && return 1
	q "SELECT 1 FROM ops_event
		WHERE note ~ '复审过' AND note ~ 'sha=[[:space:]]*${tip}'
		AND seat <> '$aseat' LIMIT 1" | grep -q .
}
mergeable() { # $1=tip → rc0=对 origin/master 净合（merge-tree 预检，不动工树）
	git merge-tree --write-tree origin/master "$1" >/dev/null 2>&1
}
mig_dep_gate() { # $1=br名 $2=tip → rc0=依赖序合法；rc1=所依未合（say 记跳因）
	# 工单#25800 合流时序闸：支新增迁移的 depends_on 目标须已在
	# origin/master 或同支自带——跨支未合依赖跳支不合（不占落地账、
	# 不烧冷却，下拍重扫）。判据核心=mig_dep_lib.sh::mig_dep_gate_check；
	# 解析件取支尖自带版——deploy 树未含本闸件时亦能判（渐进启用）。
	local br="$1" tip="$2" mb added libtmp miss rc
	mb=$(git merge-base origin/master "$tip" 2>/dev/null) || return 0 # tolerate: base 败=放——mergeable 已预检此支可合域
	added=$(git diff --name-only --diff-filter=A "$mb" "$tip" -- 'migrations/*.sql' 2>/dev/null)
	[ -n "$added" ] || return 0
	libtmp=$(mktemp /tmp/migdep-lib-XXXXXX 2>/dev/null) || return 0 # tolerate: mktemp 败=放闸下拍——支自带 check_all⑨已保底声明合法
	if ! git show "$tip:scripts/mig_dep_lib.sh" > "$libtmp" 2>/dev/null; then
		rm -f "$libtmp"; return 0 # tolerate: 支无解析件=无声明语义——放
	fi
	. "$libtmp"
	miss=$(printf '%s\n' "$added" | mig_dep_gate_check "$tip" origin/master 2>/dev/null)
	rc=$?
	rm -f "$libtmp"
	[ "$rc" -eq 0 ] && return 0
	say "跳 $br——迁移依赖未合：$(printf '%s' "$miss" | tr '\n' ' ' | sed 's/ $//')（目标须先在 origin/master 或同支自带）"
	return 1
}

# ── 扫支：七日内未并支按尖龄升序，一拍一支 ──
CANDS=$(git for-each-ref --no-merged origin/master --sort=committerdate \
	--format='%(committerdate:unix) %(refname:short) %(objectname)' \
	'refs/remotes/origin/fix/*' 'refs/remotes/origin/feat/*' \
	'refs/remotes/origin/t/*' 'refs/remotes/origin/audit/*' \
	'refs/remotes/origin/revfix/*' 'refs/remotes/origin/hotfix/*' 2>/dev/null \
	| awk -v c=$(( $(date +%s) - 604800 )) '$1>c {print $2,$3}' \
	| sed 's|^origin/||')

SB=""; STIP=""
while read -r br tip; do
	[ -n "$br" ] && [ -n "$tip" ] || continue
	recent_fail "$tip" && { say "跳 $br——同尖24h败冷却"; continue; }
	stamp_ok "$tip" || { say "跳 $br——戳缺/戳陈"; continue; }
	if docs_only_diff "$tip"; then
		say "$br docs-only 差分——免复审行(b)"
	elif ! review_ok "$br" "$tip"; then
		say "跳 $br——复审过行缺/分席不可证/无DSN"; continue
	fi
	mig_dep_gate "$br" "$tip" || continue
	if ! mergeable "$tip"; then
		fact "落地败 支=$br 尖=$tip 因=对origin/master不可净合(merge-tree冲突)——支须重基,同尖冷却24h"
		esc "landq_conflict_${br//\//_}" "部署/landq_merge_conflict: 支=$br 尖=$tip 对origin/master不可净合——须重基或人工裁决,支入24h冷却待处置"
		continue
	fi
	SB="$br"; STIP="$tip"; break
done <<< "$CANDS"

[ -n "$SB" ] || { say "无三件齐候选——本拍空"; exit 0; }
say "选中 $SB 尖=$STIP——起合"

# ── 落法：独立工树 merge --no-ff→check_all→戳→部署树推 sha:master ──
WT=$(mktemp -d "${WTBASE}/o-land-XXXXXX")
if ! git worktree add --detach "$WT/wt" origin/master >/dev/null 2>&1; then
	fact "落地暂缓 支=$SB 尖=$STIP 因=独立工树建立败"
	exit 0
fi
SUBJ=$(git log -1 --format=%s "$STIP" 2>/dev/null | cut -c1-80) # tolerate: log 败=空串——subject 缺省不阻断合流
TRAILERS=$(git log --format=%B origin/master.."$STIP" 2>/dev/null \
	| grep -E '^(坑处置:|免疫件:)' | sort -u) # tolerate: grep 空集=支尖无trailer合法径
MSG="merge: land-queue $SB——${SUBJ}
落地件机械合（工单#22577/D-0048③）：三件齐——鲜戳+复审过 sha=$STIP+净合；
deploy_watch 照旧 ff+部署。"
if [ -n "$TRAILERS" ]; then
	MSG="$MSG

承袭支尖：
$TRAILERS"
fi
MSG="$MSG

坑处置:合流机械步——判词承支尖commit体，无新坑"

git merge-base --is-ancestor "$STIP" origin/master 2>/dev/null \
	&& { fact "落地暂缓 支=$SB 尖=$STIP 因=尖已被他径并入master——候选态竞动,下拍重扫"; exit 0; }
if ! (cd "$WT/wt" && git merge --no-ff -m "$MSG" "$STIP" >/dev/null 2>&1); then
	(cd "$WT/wt" && git merge --abort >/dev/null 2>&1) || true # tolerate: abort 败=本不在merge态——rc1兜底
	fact "落地败 支=$SB 尖=$STIP 因=工树merge冲突(预检后竞动)——同尖冷却24h"
	esc "landq_conflict_${SB//\//_}" "部署/landq_merge_conflict: 支=$SB 尖=$STIP 工树merge冲突(预检后竞动)——须人工裁决,支入24h冷却"
	exit 0
fi
MSHA=$(git -C "$WT/wt" rev-parse HEAD 2>/dev/null) # tolerate: rev-parse 败=空串落下行戳验拒
[ -n "$MSHA" ] || { fact "落地败 支=$SB 尖=$STIP 因=merge后HEAD不可解"; exit 0; }
[ "$(git -C "$WT/wt" rev-parse 'HEAD^2' 2>/dev/null)" = "$STIP" ] \
	|| { fact "落地暂缓 支=$SB 尖=$STIP 因=merge未成真合(Already-up-to-date竞动)——下拍重扫"; exit 0; }

GD=$(git -C "$WT/wt" rev-parse --path-format=absolute --git-common-dir 2>/dev/null) # tolerate: 败=空串落下行戳验拒
MTREE=$(git -C "$WT/wt" rev-parse 'HEAD^{tree}' 2>/dev/null) # tolerate: 败=空串→不走快径退全验
TTREE=$(git rev-parse "$STIP^{tree}" 2>/dev/null)           # tolerate: 同上
if [ -n "$MTREE" ] && [ -n "$TTREE" ] && [ "$MTREE" = "$TTREE" ] \
	&& [ -n "$GD" ] && [ -f "$GD/check-green-$STIP" ]; then
	# 工单#deploy-pipe-tax①合并树≡支尖树快径：树同=内容同=合并点重验=
	# 复验同一份已戳内容（戳义=「本内容过全谱」随树迁非随sha——
	# master未动常态≈15min/件白烧实证）。戳名迁供 pre-push 出处闸，
	# 合并点check_all/bootstrap全跳。
	if cp "$GD/check-green-$STIP" "$GD/check-green-$MSHA" 2>/dev/null; then
		fact "落成快径 支=$SB 尖=$STIP merge=$MSHA——合并树≡支尖树跳合并点重验(内容=已戳件,戳随树迁)"
	else
		fact "落地暂缓 支=$SB 尖=$STIP 因=快径戳迁写败——下拍全验径"
		exit 0
	fi
else
	if ! (cd "$WT/wt" && bash -c "$BOOTSTRAP" >/dev/null 2>&1); then
		fact "落地败 支=$SB 尖=$STIP 因=工树bootstrap败(gen面缺)"
		exit 0
	fi
	COUT=$(cd "$WT/wt" && bash -c "$CHECK_ALL" 2>&1) # tolerate: check_all 败=rc非零落下行判据——输出截尾入账
	CRC=$?
	if [ "$CRC" -ne 0 ]; then
		fact "落地败 支=$SB 尖=$STIP 因=合并点check_all红 尾=$(printf '%s' "$COUT" | tail -3 | tr '\n' ' ' | cut -c1-160)"
		esc "landq_merge_red_${SB//\//_}" "部署/landq_merge_red: 支=$SB 尖=$STIP 合并点check_all红 尾=$(printf '%s' "$COUT" | tail -2 | tr '\n' ' ' | cut -c1-120)——须人工定性修支或撤票"
		exit 0
	fi
	[ -n "$GD" ] && [ -f "$GD/check-green-$MSHA" ] \
		|| { fact "落地败 支=$SB 尖=$STIP 因=check_all绿而合并点戳缺(自相矛盾)"; exit 0; }
fi

# 推前对拍：check_all 窗内 origin/master 若已移，sha:master 非ff必拒——
# 先验祖先关系，移了落暂缓行下拍重合（不烧推送重试）。
git fetch origin master -q 2>/dev/null || true # tolerate: fetch 败=沿用旧ref推——pre-push远端自拒兜底
if ! git merge-base --is-ancestor origin/master "$MSHA" 2>/dev/null; then
	fact "落地暂缓 支=$SB 尖=$STIP 因=check_all窗内master已移——下拍重合"
	exit 0
fi
if (cd "$REPO" && bash -c "$PUSH_CMD origin $MSHA:master" >/dev/null 2>&1); then
	fact "落成 支=$SB 尖=$STIP merge=$MSHA——deploy_watch接力ff+部署"
	say "落成 $SB → master $MSHA"
	# 工单#24624：机械合支自动销簿记——支体commits提「工单#N」销案
	# 行随落地写（现径=外席巡检滞后销缺口实证）。闸:仅机械合支
	# +号面≤5防爆销(超出面落备忘不销——人工复核域非机械域)。
	TNS=$(cd "$WT/wt" && git log --format='%B' "origin/master..$STIP" 2>/dev/null \
		| grep -oE '工单#[0-9]+' | sort -u | head -8)
	TNN=$(printf '%s\n' "$TNS" | grep -c '工单#' || true) # tolerate: grep空集rc1——计数器非闸
	if [ "$TNN" -le 5 ]; then
		for tn in $TNS; do
			tid="${tn#工单#}"
			cn="工单#$tid 销——机械合支 $SB 落成 $MSHA 自动销簿记（工单#24624件——支体commit载号随合即销,外席巡检滞后径退役）"
			"$PSQL" "$DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
				VALUES('处置席','处置','$tid','$(sqlq "$cn")')" >/dev/null 2>&1 \
				|| say "自动销行写败 tid=$tid（不阻断）" # tolerate: 销行写败=巡检旧径兜
		done
		[ "$TNN" -gt 0 ] && fact "自动销簿记 支=$SB 票=$TNS——$TNN 号随合销"
	else
		fact "自动销暂缓 支=$SB——commit工单号面=$TNN>5超机械域,人工复核"
	fi
else
	fact "落地暂缓 支=$SB 尖=$STIP 因=sha:master推败(远端先行/闸拒)——下拍重试"
fi
exit 0
