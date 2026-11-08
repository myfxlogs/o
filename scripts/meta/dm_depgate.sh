#!/usr/bin/env bash
# dm_depgate.sh —— 派单依赖感知闸（T-0413）：候选已批票施工范围 ∩
# 在审分支 diff 路径集为空→绕班序即派；有交集/不可判→留班内常径。
# 范围两源并集：票面「范围」节文件清单 ∪ 票支 merge-base diff（另查
# 约定 wt 脏面/未推面）；判的是路径集不是语义。
#   - 范围节缺文件清单=候选不可判，排队不升件；
#   - 范围节在但读不出路径（空读）/票件缺失/对拍败=升件自报；
#   - 任一在审件范围不可判=全体候选保守排队（未知面可能撞任意候选）。
# 环境覆写（测试/对拍面）：DM_ENV_FILE、O_DM_QUEUE_SH、
# O_WORKER_DISPATCH_SCRIPT、O_DEPGATE_ESC_SH、O_DEPGATE_REPO、
# O_DEPGATE_TASK_DIR、O_DEPGATE_WT_ROOT、O_DEPGATE_MASTER_REF、
# O_DEPGATE_MODEL。
set -u
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$FIN_DSN" ] || exit 0

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
DM_QUEUE_SH=${O_DM_QUEUE_SH:-"$SCRIPT_DIR/dm_queue.sh"}
DISPATCH_WORKER=${O_WORKER_DISPATCH_SCRIPT:-/opt/o/scripts/meta/dispatch_worker.sh}
OPS_ESC_SH=${O_DEPGATE_ESC_SH:-"$SCRIPT_DIR/ops_esc.sh"}
MANUAL_LANE_SH=${O_MANUAL_LANE_SH:-"$SCRIPT_DIR/manual_lane_check.sh"} # T-0438 件⑥人工径滤
REPO=${O_DEPGATE_REPO:-/opt/o}
TASK_DIR=${O_DEPGATE_TASK_DIR:-$REPO/docs/tasks}
WT_ROOT=${O_DEPGATE_WT_ROOT:-/opt/o-wt}
MASTER_REF=${O_DEPGATE_MASTER_REF:-origin/master}
MODEL=${O_DEPGATE_MODEL:-swe-2-flash-high}
TMPD=$(mktemp -d "${TMPDIR:-/tmp}/o-dm-depgate.XXXXXX") || exit 1
trap 'rm -rf -- "$TMPD"' EXIT

QC() { printf '%s' "${1:-}" | sed "s/'/''/g"; }
esc() { # $1=ticket $2=prefix $3=detail —— 自报末环败静默退（ops_esc 结构性限度在案）
	"$OPS_ESC_SH" CTO "$1" "" "" "$2" "$3" || true # tolerate: 升件写径败不能再升件——静默退
}

norm_path() {
	local p=${1%$'\r'}
	p=${p#./}
	case "$p" in
		"$REPO/"*) p=${p#"$REPO"/} ;;
		/opt/o/*) p=${p#/opt/o/} ;;
	esac
	p=${p#/}
	[ -n "$p" ] && printf '%s\n' "$p"
}

# 票面「范围」节路径词提取。rc: 0=有路径；2=票件缺/不可读；3=范围节缺；
# 4=范围节在但零路径词（空读）。
scope_paths() {
	local file=$1 section tok
	[ -f "$file" ] && [ -r "$file" ] || return 2
	grep -Eq '^##[[:space:]]*范围($|[^外])' "$file" || return 3
	section=$(awk '/^##[[:space:]]*范围($|[^外])/{f=1;next} f&&/^##[[:space:]]+/{exit} f{print}' "$file")
	while IFS= read -r tok; do
		[ -n "$tok" ] || continue
		case "$tok" in http://*|https://*|*@*) continue ;; esac
		case "$tok" in
			*/*|Dockerfile*|docker-compose*|Makefile|go.mod|go.sum|*.go|*.sh|*.sql|*.ts|*.tsx|*.js|*.jsx|*.proto|*.md|*.markdown|*.yaml|*.yml|*.json|*.css|*.scss|*.html|*.toml|*.service|*.timer)
				norm_path "$tok" ;;
		esac
	done < <(printf '%s\n' "$section" | sed -E "s/[\`|~(){}<>\"'，。；：、（）《》]/ /g; s/[][]/ /g" | tr '[:space:]' '\n') | sort -u >"$TMPD/scope.paths"
	[ -s "$TMPD/scope.paths" ] || return 4
	cat "$TMPD/scope.paths"
}

# 票支/工树路径集：本地支+远端支 merge-base diff ∪ 约定 wt HEAD diff/脏面。
ticket_branch_paths() {
	local tid=$1 num=${tid#T-} ref files line p rc=0 wtd
	while IFS= read -r ref; do
		[ -n "$ref" ] || continue
		if files=$(git -C "$REPO" diff --name-only "$MASTER_REF...$ref" 2>/dev/null); then
			while IFS= read -r p; do norm_path "${p##* -> }"; done <<< "$files"
		else
			rc=5
		fi
	done < <( { git -C "$REPO" branch --list "t/$num" "t/$num-*" --format='%(refname:short)' \
		&& git -C "$REPO" branch -r --list "origin/t/$num" "origin/t/$num-*" \
			--format='%(refname:short)'; } 2>/dev/null | sort -u )

	wtd="$WT_ROOT/t$num"
	if [ -d "$wtd" ] && [ -e "$wtd/.git" ]; then
		if files=$(git -C "$wtd" diff --name-only "$MASTER_REF...HEAD" 2>/dev/null); then
			while IFS= read -r p; do norm_path "${p##* -> }"; done <<< "$files"
		else
			rc=5
		fi
		if files=$(git -C "$wtd" status --porcelain 2>/dev/null); then
			while IFS= read -r line; do
				p=${line:3}; norm_path "${p##* -> }"
			done <<< "$files"
		else
			rc=5
		fi
	fi
	return "$rc"
}

# tid→路径集。产物：$out=并集、$out.scope=票面源、$out.branch=分支源、
# $out.src_rc=票面源状态。rc: 0=可取（集可空）；5=分支/diff 对拍败。
ticket_paths() {
	local tid=$1 out=$2 file src_rc=0 brc=0
	: >"$out"; : >"$out.scope"; : >"$out.branch"
	file=$(compgen -G "$TASK_DIR/$tid-*.md" | head -1)
	if [ -n "$file" ]; then
		scope_paths "$file" >"$out.scope" || src_rc=$?
	else
		src_rc=2
	fi
	ticket_branch_paths "$tid" >"$out.branch" || brc=$?
	printf '%s\n' "$src_rc" >"$out.src_rc"
	sort -u "$out.scope" "$out.branch" >"$out"
	return "$brc"
}

# 路径对拍：目录/词干前缀、裸文件名按 basename、glob 按壳模式；宁可多陪绑
# 不可漏放（有疑即交集=保守）。
path_match() {
	local cand=${1#./} actual=${2#./} abase
	[ -n "$cand" ] && [ -n "$actual" ] || return 1
	case "$cand" in
		*/) [ "${actual#"$cand"}" != "$actual" ] && return 0 ;;
	esac
	if [ "${cand%/*}" = "$cand" ]; then
		abase=${actual##*/}
		[ "$abase" = "$cand" ] && return 0
		case "$cand" in *'*'*|*'?'*|*'['*) [[ $abase == $cand ]] && return 0 ;; esac
	else
		[ "$actual" = "$cand" ] && return 0
		case "$actual" in "$cand"/*|"$cand".*) return 0 ;; esac
		case "$cand" in *'*'*|*'?'*|*'['*) [[ $actual == $cand ]] && return 0 ;; esac
	fi
	return 1
}
paths_overlap() { # $1=candidate paths file $2=review paths file；0=有交集 1=零交集
	local c r
	while IFS= read -r c; do
		[ -n "$c" ] || continue
		while IFS= read -r r; do
			[ -n "$r" ] || continue
			path_match "$c" "$r" && return 0
		done < "$2"
	done < "$1"
	return 1
}

insert_dispatch_event() { # 0=本拍落派单行；1=票态已竞态前移；5=写径败
	local tid=$1 note=$2 sqlf=$TMPD/ins-$tid.sql out
	cat >"$sqlf" <<SQL
BEGIN;
SELECT pg_advisory_xact_lock(hashtext('dm_depgate:$(QC "$tid")'));
WITH cur AS (
  SELECT event FROM ops_event WHERE ticket='$(QC "$tid")'
  ORDER BY ts DESC,id DESC LIMIT 1
), ins AS (
  INSERT INTO ops_event(seat,event,ticket,note)
  SELECT 'CTO','派单','$(QC "$tid")','$(QC "$note")'
  WHERE EXISTS (SELECT 1 FROM cur WHERE event='已批')
  RETURNING ticket
)
SELECT ticket FROM ins;
COMMIT;
SQL
	out=$(psql "$FIN_DSN" -v ON_ERROR_STOP=1 -qAt -f "$sqlf" 2>/dev/null) || return 5
	grep -Fxq "$tid" <<<"$out"
}

REVIEW_TIDS=$TMPD/review.tids
APPROVED_TIDS=$TMPD/approved.tids
"$DM_QUEUE_SH" review >"$REVIEW_TIDS" || exit 1
"$DM_QUEUE_SH" approved >"$APPROVED_TIDS" || exit 1
[ -s "$APPROVED_TIDS" ] || exit 0

REVIEW_PATHS=$TMPD/review.paths
: >"$REVIEW_PATHS"
review_unknown=0
while IFS= read -r tid; do
	[ -n "$tid" ] || continue
	paths=$TMPD/review-$tid.paths
	if ! ticket_paths "$tid" "$paths"; then
		esc "$tid" "DEPGATE-REVIEW-FAIL-$tid" "在审分支 diff/工树面对拍败——候选票保守排队"
		review_unknown=1
		continue
	fi
	src_rc=$(cat "$paths.src_rc")
	if [ "$src_rc" = 2 ] || [ "$src_rc" = 4 ] || [ ! -s "$paths" ]; then
		esc "$tid" "DEPGATE-REVIEW-UNKNOWN-$tid" "在审范围不可判（src_rc=$src_rc paths=$(wc -l <"$paths")）——候选票保守排队"
		review_unknown=1
		continue
	fi
	cat "$paths" >>"$REVIEW_PATHS"
done < "$REVIEW_TIDS"
sort -u -o "$REVIEW_PATHS" "$REVIEW_PATHS"
if [ "$review_unknown" -eq 1 ]; then
	echo "[dm-depgate] 在审范围不可判——候选票保守陪绑" >&2
	exit 0
fi

while IFS= read -r tid; do
	[ -n "$tid" ] || continue
	paths=$TMPD/candidate-$tid.paths
	if ! ticket_paths "$tid" "$paths"; then
		esc "$tid" "DEPGATE-CANDIDATE-FAIL-$tid" "候选票分支/工树面对拍败——排队"
		continue
	fi
	src_rc=$(cat "$paths.src_rc")
	case "$src_rc" in
		2) esc "$tid" "DEPGATE-CANDIDATE-FAIL-$tid" "候选票件缺失/不可读——排队"; continue ;;
		4) esc "$tid" "DEPGATE-SCOPE-EMPTY-$tid" "票面范围节零路径词（空读）——排队"; continue ;;
	esac
	# T-0438 件⑥ 人工径滤（坑条目 real-money-ticket-machine-dispatch-waste 律②：
	# 票面「执行=人工径」出生行=机器永不可为——候选即滤，不派不挂起不计漏拉；
	# rc2 票件缺不在此判（上方 src_rc=2 径已自报）——滤除只认 rc0）。
	if "$MANUAL_LANE_SH" "$tid" "$TASK_DIR"; then
		echo "[dm-depgate] $tid queued——人工径票（执行=人工径）不进派单链" >&2
		continue
	fi
	if [ ! -s "$paths.scope" ]; then
		echo "[dm-depgate] $tid queued——票面范围节缺文件清单" >&2
		continue
	fi
	if paths_overlap "$paths" "$REVIEW_PATHS"; then
		echo "[dm-depgate] $tid queued——候选范围∩在审diff非空" >&2
		continue
	fi

	note="依赖闸即派——候选范围∩在审diff=空 model=$MODEL"
	insert_dispatch_event "$tid" "$note"
	rc=$?
	if [ "$rc" -eq 5 ]; then
		esc "$tid" "DEPGATE-DISPATCH-FAIL-$tid" "派单事件写径败——未拉起，票留待派"
		continue
	fi
	[ "$rc" -eq 0 ] || continue
	prompt="读 /opt/o/skills/施工席/SKILL.md 施工协议 领 $tid 施工"
	"$DISPATCH_WORKER" "$tid" "$MODEL" "$prompt" >/dev/null 2>&1
	rc=$?
	if [ "$rc" -eq 0 ]; then
		echo "[dm-depgate] $tid dispatched——零交集绕班序即派" >&2
		continue
	fi
	if [ "$rc" -eq 75 ]; then
		echo "[dm-depgate] worker water/quota full——$tid 派单行已落，留常径补拉" >&2
		break
	fi
	esc "$tid" "DEPGATE-DISPATCH-FAIL-$tid" "dispatch_worker rc=$rc——派单行已落，留常径补拉"
done < "$APPROVED_TIDS"
exit 0
