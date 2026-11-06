#!/usr/bin/env bash
# dm_ledger.sh —— DM 机械轨台账/票翻件（T-0414：机械层出会话——零裁量
# 簿记不再烧 token）。每拍由 dm_tick.sh 尾段调用（kick/兜底拍同轨——
# ops_event 新行即办，会话开班只为判断件）。两条互补收敛规则：
#   A 台账→文件（票翻）：票内最新事件=完成，或 裁决/通过(note 通过%)
#     且收口件已证并 master（收口类 note 内 SHA/票支 tip/merge commit
#     三层取证）→ 票件 git mv 入 done/ + 状态行翻「完成」（机械模板），
#     commit [角色:决策] 机械串+check_all 绿戳+git_push。
#     裁决通过但合并不可证=跳过；裁决龄>O_LEDGER_VERDICT_STALE 仍不可证
#     →升件不静默。
#   B 文件→台账（补登）：票在 done/ + 有收口类事件史（收口/移交/
#     收口待审/审计中——管道期票）+ 无完成行 + 最新事件非负词
#     （驳回/异常/挂起）→ 补 ops_event 完成行（seat=META-DM 机械轨
#     可辨，note 前缀「机械轨」；advisory xact 锁+行内重查双防重）。
#   git 写径=dm_repolock.sh 同台互斥（会话 merge/push 同锁——忙即本拍
#   让位不抢写；抢陈锁/冲突/脏树/推败→升件不静默）。
#   用法: dm_ledger.sh [all|flip|reconcile]（缺省 all=A 后 B；
#   flip=只票翻面，reconcile=只补登面）。
# env 覆写: DM_ENV_FILE O_LEDGER_REPO O_LEDGER_TASK_DIR O_LEDGER_DONE_DIR
#   O_LEDGER_WT_ROOT O_LEDGER_MASTER_REF O_LEDGER_LOCK_SH O_LEDGER_CHECK_ALL
#   O_LEDGER_GIT_PUSH O_LEDGER_MAX_BATCH O_LEDGER_VERDICT_STALE
#   O_LEDGER_LOCK_WAIT O_LEDGER_OPS_ESC
set -u
MODE=${1:-all}
case "$MODE" in all|flip|reconcile) ;; *)
	echo "用法: dm_ledger.sh [all|flip|reconcile]" >&2; exit 2 ;;
esac
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$FIN_DSN" ] || exit 0
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO=${O_LEDGER_REPO:-/opt/o}
TASK_DIR=${O_LEDGER_TASK_DIR:-$REPO/docs/tasks}
DONE_DIR=${O_LEDGER_DONE_DIR:-$REPO/docs/tasks/done}
WT_ROOT=${O_LEDGER_WT_ROOT:-/opt/o-wt}
MASTER_REF=${O_LEDGER_MASTER_REF:-origin/master}
LOCK_SH=${O_LEDGER_LOCK_SH:-"$SCRIPT_DIR/dm_repolock.sh"}
CHECK_ALL=${O_LEDGER_CHECK_ALL:-$REPO/scripts/check_all.sh}
GIT_PUSH=${O_LEDGER_GIT_PUSH:-$REPO/scripts/git_push.sh}
OPS_ESC_SH=${O_LEDGER_OPS_ESC:-"$SCRIPT_DIR/ops_esc.sh"}
MAX_BATCH=${O_LEDGER_MAX_BATCH:-40}
VERDICT_STALE=${O_LEDGER_VERDICT_STALE:-7200}
LOCK_WAIT=${O_LEDGER_LOCK_WAIT:-90}

QC() { printf '%s' "${1:-}" | sed "s/'/''/g"; }
esc() { # $1=ticket $2=prefix $3=detail —— 升件自报末环败静默退（结构性限度在案）
	"$OPS_ESC_SH" META-DM "$1" "" "" "$2" "$3" || true # tolerate: 升件自报末环——ops_esc 写径败静默退（结构性限度在案）
}
task_file()  { compgen -G "$TASK_DIR/$1-*.md" 2>/dev/null | head -1; }
done_file()  { compgen -G "$DONE_DIR/$1-*.md" 2>/dev/null | head -1; }
now_date()   { date +%F; }

# 票面状态行机械翻面。形：
#   `- 状态：X` / `> 状态：X` → 同前缀 `状态：完成（tmpl）`
#   `## 状态` 节下首 `**词**` 行 → `**完成**（tmpl）`
# rc: 0=已改 1=已是完成态/无改 2=未识别状态行（不动件）
flip_status() {
	local f=$1 tmpl=$2 tmp rc
	tmp=$(mktemp "${TMPDIR:-/tmp}/o-ledger-flip.XXXXXX") || return 2
	awk -v tmpl="$tmpl" '
		BEGIN { hit=0; sec=0 }
		{
			if (hit==0) {
				if ($0 ~ /^[[:space:]]*[-*>][[:space:]]*状态[:：]/) {
					if ($0 ~ /状态[:：][[:space:]*]*(完成|作废|废弃|取消)/) { hit=2; print; next }
					sub(/状态[:：].*/, "状态：完成（" tmpl "）")
					hit=1; print; next
				}
				if ($0 ~ /^#{1,4}[[:space:]]*状态/) { sec=1; print; next }
				if (sec==1 && $0 ~ /^#{1,4}[[:space:]]/) { sec=0 }
				if (sec==1 && $0 ~ /^[[:space:]]*\*\*[^*]+\*\*/) {
					if ($0 ~ /\*\*(完成|作废|废弃|取消)\*\*/) { hit=2; print; next }
					sub(/^[[:space:]]*\*\*[^*]+\*\*.*/, "**完成**（" tmpl "）")
					hit=1; print; next
				}
			}
			print
		}
		END { exit (hit==1?0:(hit==2?1:3)) }
	' "$f" > "$tmp"
	rc=$?
	case $rc in
		0) cat "$tmp" >"$f"; rm -f -- "$tmp"; return 0 ;;
		1) rm -f -- "$tmp"; return 1 ;;
		*) rm -f -- "$tmp"; return 2 ;;
	esac
}

# 状态行现态探测（只读——flip_status 同锚同式防漂）。
# rc: 0=已是完成态 1=状态行在非完成态（须翻） 2=未识别状态行
status_done() {
	awk '
		BEGIN { rc=2; hit=0; sec=0 }
		hit==0 && /^[[:space:]]*[-*>][[:space:]]*状态[:：]/ {
			rc=($0 ~ /状态[:：][[:space:]*]*(完成|作废|废弃|取消)/ ? 0 : 1); hit=1; next
		}
		hit==0 && /^#{1,4}[[:space:]]*状态/ { sec=1; next }
		hit==0 && sec==1 && /^#{1,4}[[:space:]]/ { sec=0 }
		hit==0 && sec==1 && /^[[:space:]]*\*\*[^*]+\*\*/ {
			rc=($0 ~ /\*\*(完成|作废|废弃|取消)\*\*/ ? 0 : 1); hit=1
		}
		END { exit rc }
	' "$1"
}

# 收口件已证并 master：收口类 note 内 SHA ∪ 票支 tip ∪ wt HEAD ∪
# merge commit 题名——任一层证 ancestor/在录即过。
merged_proof() {
	local tid=$1 num=${tid#T-} sha ref
	while IFS= read -r sha; do
		[ -n "$sha" ] || continue
		git -C "$REPO" merge-base --is-ancestor "$sha" "$MASTER_REF" 2>/dev/null && return 0
	done < <(psql "$FIN_DSN" -qAt -c "SELECT note FROM ops_event WHERE ticket='$(QC "$tid")'
		AND event IN ('收口','移交','收口待审') ORDER BY ts DESC,id DESC LIMIT 8" 2>/dev/null \
		| grep -oE '[0-9a-f]{7,40}' | sort -u | head -20)
	while IFS= read -r ref; do
		[ -n "$ref" ] || continue
		sha=$(git -C "$REPO" rev-parse --verify -q "$ref" 2>/dev/null) || continue
		git -C "$REPO" merge-base --is-ancestor "$sha" "$MASTER_REF" 2>/dev/null && return 0
	done < <(git -C "$REPO" for-each-ref --format='%(refname:short)' \
		"refs/heads/t/$num" "refs/heads/t/$num-*" "refs/remotes/origin/t/$num" "refs/remotes/origin/t/$num-*" 2>/dev/null)
	if [ -d "$WT_ROOT/t$num" ]; then
		sha=$(git -C "$WT_ROOT/t$num" rev-parse --verify -q HEAD 2>/dev/null) && \
			git -C "$REPO" merge-base --is-ancestor "$sha" "$MASTER_REF" 2>/dev/null && return 0
	fi
	git -C "$REPO" log "$MASTER_REF" --format=%H -80 --grep="Merge t/$num-" 2>/dev/null | grep -q . && return 0
	return 1
}

# ── A 面扫描（读径）：定票翻动作集——mv/翻面在锁内才真动手 ────────────
ACT_T=(); ACT_TF=(); ACT_TMPL=()
if [ "$MODE" != "reconcile" ]; then
	# 远端先行判据面——先 fetch 再取证（读径不进写锁）
	git -C "$REPO" fetch origin master --quiet 2>/dev/null || true # tolerate: fetch 败=按本地 origin/master 快照判——合并取证照常降级运行
	ROWS=$(psql "$FIN_DSN" -qAt -F $'\x1f' -c "SELECT DISTINCT ON (l.ticket) l.ticket, e.event,
		to_char(e.ts,'YYYY-MM-DD'), extract(epoch from now()-e.ts)::bigint
	FROM (SELECT ticket,max(ts) mt FROM ops_event WHERE ticket<>'' GROUP BY ticket) l
	JOIN ops_event e ON e.ticket=l.ticket AND e.ts=l.mt
	WHERE e.event='完成' OR (e.event IN ('裁决','通过') AND e.note LIKE '通过%')
	ORDER BY l.ticket, CASE WHEN e.event='完成' THEN 0 ELSE 1 END, e.id DESC" 2>/dev/null) || { esc "" "LEDGER-SCAN-FAIL" "最新终态推导查询败——本拍票翻滑行"; exit 1; }

	while IFS=$'\x1f' read -r tid ev vdate age; do
		[ -n "$tid" ] || continue
		case "$tid" in T-[0-9][0-9][0-9][0-9]) ;; *) continue ;; esac
		[ "${#ACT_T[@]}" -lt "$MAX_BATCH" ] || { echo "[dm-ledger] 批帽 $MAX_BATCH 到——余票下拍续" >&2; break; }
		tf=$(task_file "$tid") df=$(done_file "$tid")
		if [ -n "$tf" ] && [ -n "$df" ]; then
			esc "$tid" "LEDGER-DUP-$tid" "票件双存（tasks/ 与 done/ 同票两档）——取 done/ 面，存根待人工清"
			tf=
		fi
		[ -n "$tf" ] || [ -n "$df" ] || { esc "$tid" "LEDGER-NOFILE-$tid" "终态票无票件（tasks/done 双缺）——票翻不可为"; continue; }
		# 裁决通过面归档前先证收口已并 master；完成行面台账即权威直接翻。
		if [ "$ev" != "完成" ] && [ -n "$tf" ]; then
			if ! merged_proof "$tid"; then
				if [ "${age:-0}" -gt "$VERDICT_STALE" ]; then
					esc "$tid" "LEDGER-MERGE-UNPROVEN-$tid" "裁决通过龄>${VERDICT_STALE}s 仍收口未并 master 不可证——票翻压住呈裁"
				else
					echo "[dm-ledger] $tid 裁决通过但合并不可证——本拍压下拍续" >&2
				fi
				continue
			fi
		fi
		# 已收敛票不进动作集（df 在且状态已完成=零活——防每拍全终态集进锁）：
		# rc1=确认识别非终态才入集；rc2 未识别形视同不属本轨翻面责任直接跳
		# （T-0417 钦定判 A——古票异形态恒入集吃批帽饿殍病灶根修，识不得的形不碰）
		if [ -z "$tf" ]; then
			src=0; status_done "$df" || src=$?
			[ "$src" -eq 1 ] || continue
		fi
		if [ "$ev" = "完成" ]; then
			tmpl="ops_event 完成行 $vdate 在案——机械轨票翻"
		else
			tmpl="决策席裁决通过 $vdate——判词在案·机械轨票翻"
		fi
		ACT_T+=("$tid"); ACT_TF+=("$tf"); ACT_TMPL+=("$tmpl")
	done <<< "$ROWS"
fi

# ── B 面预扫（读径）：done/ 票无完成行且状态行未翻——补登票的状态翻
# 面与 A 集同批进锁同拍落（验收「补登+票翻完成」同拍钦定）；完成行写径
# 走锁后重扫（新归档票同拍补登）。
if [ "$MODE" != "flip" ]; then
	BROWS1=$(psql "$FIN_DSN" -qAt -F $'\x1f' -c "SELECT DISTINCT ON (l.ticket) l.ticket, e.event
	FROM (SELECT ticket,max(ts) mt FROM ops_event WHERE ticket<>'' GROUP BY ticket) l
	JOIN ops_event e ON e.ticket=l.ticket AND e.ts=l.mt
	WHERE EXISTS (SELECT 1 FROM ops_event c WHERE c.ticket=l.ticket
		AND c.event IN ('收口','移交','收口待审','审计中'))
	  AND NOT EXISTS (SELECT 1 FROM ops_event c WHERE c.ticket=l.ticket AND c.event='完成')
	  AND e.event NOT IN ('驳回','异常','挂起')
	ORDER BY l.ticket, e.id DESC" 2>/dev/null) || { esc "" "LEDGER-BSCAN-FAIL" "补登面推导查询败——本拍滑行"; exit 1; }
	while IFS=$'\x1f' read -r tid _lev; do
		[ -n "$tid" ] || continue
		df=$(done_file "$tid")
		[ -n "$df" ] || continue
		printf '%s\n' "${ACT_T[@]:-}" | grep -Fxq "$tid" 2>/dev/null && continue # A 集已载同票——一集一动作防重录
		src=0; status_done "$df" || src=$?
		[ "$src" -eq 1 ] || continue # rc1=确定非完成态才翻；rc2 未识别形不动（保守）
		ACT_T+=("$tid"); ACT_TF+=(""); ACT_TMPL+=("机械轨台账对账补登——收口行在案·完成行同拍")
	done <<< "$BROWS1"
fi

# ── git 写径：同台互斥锁内 mv/翻面/commit+绿戳+push ─────────────────
# 未推判=HEAD 与 MASTER_REF 非相等（领先/分歧皆须推——落后在锁段先对齐）。
push_needed() {
	! git -C "$REPO" merge-base --is-ancestor HEAD "$MASTER_REF" 2>/dev/null \
		|| ! git -C "$REPO" merge-base --is-ancestor "$MASTER_REF" HEAD 2>/dev/null
}
NEED_GIT=0
[ "${#ACT_T[@]}" -gt 0 ] && NEED_GIT=1
git -C "$REPO" status --porcelain -- docs/tasks 2>/dev/null | grep -q . && NEED_GIT=1
push_needed && NEED_GIT=1
if [ "$NEED_GIT" -eq 1 ]; then
	LOCK_RC=0 TOKEN=
	TOKEN=$("$LOCK_SH" acquire "META-DM-ledger" "$LOCK_WAIT" 2>/dev/null) || LOCK_RC=$?
	if [ "$LOCK_RC" -eq 6 ]; then
		TOKEN=$(printf '%s\n' "$TOKEN" | head -1)
		esc "" "REPO-LOCK-STEAL" "写径锁抢陈锁——前持锁者陈死（holder 件 stolen= 行在案）"
	elif [ "$LOCK_RC" -ne 0 ]; then
		echo "[dm-ledger] 写径锁忙（会话 merge/push 在飞）——本拍让位下拍续" >&2
		TOKEN=
	fi
	if [ -n "$TOKEN" ]; then
		trap '"$LOCK_SH" release "$TOKEN" >/dev/null 2>&1 || true' EXIT # tolerate: 释锁末环——锁目已失/释败由陈锁抢兜底（结构性限度在案）
		DONE_TS=()
		if [ -n "$(git -C "$REPO" status --porcelain 2>/dev/null)" ]; then
			# 锁内仍脏=锁外写者越闸（会话未纳闸期/人工）——不代收，升件显报
			esc "" "REPO-DIRTY-INLOCK" "持锁后部署树仍脏——锁外写者越闸疑案，本拍不动收"
		else
			# 落后 ff 对齐；分歧（本地未推+远端先行）rebase 合流——
			# master_push_converge 同径；冲突即 --abort 回原状升件。
			if git -C "$REPO" merge-base --is-ancestor HEAD "$MASTER_REF" 2>/dev/null \
				&& ! git -C "$REPO" merge-base --is-ancestor "$MASTER_REF" HEAD 2>/dev/null; then
				git -C "$REPO" merge --ff-only "$MASTER_REF" >/dev/null 2>&1 || esc "" "REPO-FF-FAIL" "落后对齐 ff-only 败——异常分歧"
			elif ! git -C "$REPO" merge-base --is-ancestor "$MASTER_REF" HEAD 2>/dev/null \
				&& ! git -C "$REPO" merge-base --is-ancestor HEAD "$MASTER_REF" 2>/dev/null; then
				if ! O_ALLOW_NONFF=1 git -C "$REPO" rebase "$MASTER_REF" >/dev/null 2>&1; then
					git -C "$REPO" rebase --abort >/dev/null 2>&1 || true # tolerate: abort 败=本不在 rebase
					esc "" "REPO-REBASE-CONFLICT" "分歧 rebase 冲突——已回原状，人工合流兜底"
				fi
			fi
			if [ -z "$(git -C "$REPO" status --porcelain 2>/dev/null)" ]; then
				for i in "${!ACT_T[@]}"; do
					tid=${ACT_T[$i]} tf=${ACT_TF[$i]} tmpl=${ACT_TMPL[$i]}
					target=$(done_file "$tid") moved=0
					if [ -n "$tf" ]; then
						mkdir -p "$DONE_DIR" 2>/dev/null || true # tolerate: done/ 恒在——幂等兜底
						if mv -- "$tf" "$DONE_DIR/" 2>/dev/null; then
							target="$DONE_DIR/${tf##*/}"; moved=1
						else
							target=$(done_file "$tid") # 会话抢先归档=目标已就位——续翻状态行不视为败
						fi
					fi
					[ -n "$target" ] || { esc "$tid" "LEDGER-MV-FAIL-$tid" "票件 mv 入 done/ 败且 done/ 亦无件——票翻半途压住"; continue; }
					fsrc=0; flip_status "$target" "$tmpl" || fsrc=$?
					[ "$moved" -eq 1 ] || [ "$fsrc" -eq 0 ] && DONE_TS+=("$tid")
				done
			fi
			if [ -n "$(git -C "$REPO" status --porcelain -- docs/tasks 2>/dev/null)" ]; then
				git -C "$REPO" add -- docs/tasks
				N=${#DONE_TS[@]}
				if [ "$N" -le 1 ]; then
					MSG="[角色:决策][${DONE_TS[0]:-机械轨}] 票翻完成（机械轨——ops_event 在案）。坑处置:无坑"
				else
					MSG="[角色:决策] 票翻完成×$N（机械轨——$(printf '%s ' "${DONE_TS[@]}" | sed 's/ $//')台账在案）。坑处置:无坑"
				fi
				git -C "$REPO" commit -o -m "$MSG" -- docs/tasks >/dev/null 2>&1 || esc "" "REPO-COMMIT-FAIL" "票翻 commit 败——件在工作树留待下拍"
			fi
		fi
		if push_needed; then
			PUSH_OK=0
			for _try in 1 2; do
				if (cd "$REPO" && "$CHECK_ALL") >/dev/null 2>&1 && "$GIT_PUSH" >/dev/null 2>&1; then
					PUSH_OK=1; break
				fi
				git -C "$REPO" fetch origin master --quiet 2>/dev/null || true # tolerate: 合流闸在 pre-push 层——本处重戳再试一拍
			done
			[ "$PUSH_OK" -eq 1 ] || esc "" "REPO-PUSH-FAIL" "票翻推 origin/master 两试皆败——本地 commit 滞留待下拍"
		fi
		"$LOCK_SH" release "$TOKEN" >/dev/null 2>&1 || true # tolerate: 释锁末环
		trap - EXIT
	fi
fi

# ── B 面：文件→台账（补登）─────────────────────────────────────────
# B 集在锁段后重扫——A 面同拍新归档票一并补登（账行同拍钦定）。
if [ "$MODE" != "flip" ]; then
	BROWS=$(psql "$FIN_DSN" -qAt -F $'\x1f' -c "SELECT DISTINCT ON (l.ticket) l.ticket, e.event
	FROM (SELECT ticket,max(ts) mt FROM ops_event WHERE ticket<>'' GROUP BY ticket) l
	JOIN ops_event e ON e.ticket=l.ticket AND e.ts=l.mt
	WHERE EXISTS (SELECT 1 FROM ops_event c WHERE c.ticket=l.ticket
		AND c.event IN ('收口','移交','收口待审','审计中'))
	  AND NOT EXISTS (SELECT 1 FROM ops_event c WHERE c.ticket=l.ticket AND c.event='完成')
	  AND e.event NOT IN ('驳回','异常','挂起')
	ORDER BY l.ticket, e.id DESC" 2>/dev/null) || { esc "" "LEDGER-BSCAN-FAIL" "补登面推导查询败——本拍滑行"; exit 1; }
	printf '%s\n' "$BROWS" | while IFS=$'\x1f' read -r tid lev; do
		[ -n "$tid" ] || continue
		case "$tid" in T-[0-9][0-9][0-9][0-9]) ;; *) continue ;; esac # 非票键不补（0414 事故钉——无档键误登在案）
		df=$(done_file "$tid")
		[ -n "$df" ] || continue # 文件未归档不补——B 面只认 done/ 铁证（判存在须验串非空不可凭 rc——head 恒 0 教训在案）
		case "$lev" in
			裁决|通过) note="机械轨——判词在案·收口已并master·管道退场（票翻同拍补登）" ;;
			*)         note="机械轨——台账对账补登（票在 done/·收口行在案·无完成行）" ;;
		esac
		sqlf=$(mktemp "${TMPDIR:-/tmp}/o-ledger-ins.XXXXXX") || continue
		cat >"$sqlf" <<SQL
BEGIN;
SELECT pg_advisory_xact_lock(hashtext('dm_ledger:$(QC "$tid")'));
WITH ins AS (
  INSERT INTO ops_event(seat,event,ticket,note)
  SELECT 'META-DM','完成','$(QC "$tid")','$(QC "$note")'
  WHERE NOT EXISTS (SELECT 1 FROM ops_event WHERE ticket='$(QC "$tid")' AND event='完成')
  RETURNING ticket)
SELECT ticket FROM ins;
COMMIT;
SQL
		out=$(psql "$FIN_DSN" -v ON_ERROR_STOP=1 -qAt -f "$sqlf" 2>/dev/null)
		irc=$?
		rm -f -- "$sqlf"
		if [ "$irc" -ne 0 ]; then
			esc "$tid" "LEDGER-INS-FAIL-$tid" "完成行写径败——补登未成票留 drift"
		elif grep -Fxq "$tid" <<<"$out"; then
			echo "[dm-ledger] $tid 完成行补登（latest=$lev）" >&2
		fi
	done
fi
exit 0
