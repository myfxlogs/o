#!/usr/bin/env bash
# batchdesk.sh —— 批组拾桌查询件（工单O-34419 批发摊薄机制·会话侧
# 件）：施工席拾桌时刻现查现组——派发侧零选票权不动（dispatch_worker
# 只唤不选；本件=会话工具非派发件，派发快照会陈故批组不预制）。
#
# 用法：
#   batchdesk.sh [--count] [prompts_dir]
#     缺省      —— 批候选表（每行 id|slug|spec径，REVIEW 道+批量上界
#                截批后序——会话逐票读 spec 复核后再认领）
#     --count   —— 只出 REVIEW 道可批数（dispatch_worker.sh 唤起
#                note batchable= 段同源复用——道判单点在本件）
#     prompts_dir —— spec 查找根（缺省 /opt/o-research/prompts；
#                桩测注入口）
#
# 批组判据（worker_runbook.md ⓪批组段同源——改规约必改两处）：
#   候选=ops_ticket_desk 拾桌 top10（级/期限序+自管类剔除——runbook
#   ⓪查询同口径）；道判=票面 spec obj 声明面对 land_scope.sh 白名单
#   前瞻判——声明面路径词全落自由面(docs/**.md|txt、skills/**.md、
#   .devin/**.md|txt、tools/**.md、根单层*.md剔AGENTS.md)=FREE 不入批
#   （照旧单办自合自运道）；含码/脚本/迁移/部署/契约件、嵌套层 .md、
#   或 spec 缺席/obj 无路径词不可判=REVIEW fail-closed（宁多计不漏
#   计——道判终审仍归会话读 spec 复核+支尖 land_scope 实判）。
#   批量上界=app_config dm.worker.batch_max（缺省 4——缺键/读败/
#   非数字回落缺省，同 dm.cog.wip_limit 族热配件语义）。
set -u
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$DSN" ] || { echo "[batchdesk] FIN_PG_DSN 读不出($ENVF)——fail-closed" >&2; exit 1; }

COUNT_ONLY=0
PROMPTS_DIR=/opt/o-research/prompts
for a in "$@"; do
	case "$a" in
		--count) COUNT_ONLY=1 ;;
		-*) echo "[batchdesk] 未知参 $a" >&2; exit 2 ;;
		*) PROMPTS_DIR="$a" ;;
	esac
done

# 批量上界——缺省 4（spec 钦定；读败/缺键/非数回落同 wip_limit 热配语义）
BATCH_MAX=$(psql "$DSN" -qAt -c \
	"SELECT COALESCE((SELECT value::int FROM app_config WHERE key='dm.worker.batch_max'),4)" 2>/dev/null)
case "$BATCH_MAX" in ''|*[!0-9]*) BATCH_MAX=4 ;; esac
[ "$BATCH_MAX" -ge 1 ] 2>/dev/null || BATCH_MAX=4

# is_free_obj <obj声明面文本> → rc0=路径词全落 FREE 白名单（判据与
# land_scope.sh is_free 同集——两处须同步改）；无路径词/混径=rc1。
is_free_obj() {
	local obj="$1" p seen=0 allfree=1 paths
	obj=$(printf '%s' "$obj" | sed 's|https\?://[^[:space:]]*||g')
	paths=$(printf '%s' "$obj" | tr " \t　、，。；：,;:()（）「」\"'\`" '\n' \
		| grep -E '/|\.(md|markdown|txt|sh|bash|go|sql|py|ts|tsx|js|jsx|json|ya?ml|toml|mod|sum|pb|proto|html|css|tsv)$')
	[ -n "$paths" ] || return 1
	while IFS= read -r p; do
		[ -n "$p" ] || continue
		seen=1
		case "$p" in
			docs/*.md|docs/*.txt|skills/*.md|.devin/*.md|.devin/*.txt|tools/*.md) ;;
			*/*|AGENTS.md) allfree=0; break ;;
			*.md) ;;
			*) allfree=0; break ;;
		esac
	done <<< "$paths"
	[ "$seen" = 1 ] && [ "$allfree" = 1 ]
}

# 拾桌 top10——worker_runbook ⓪/dispatch_worker.sh TICKET_N 同口径
# （级/期限序+自管类 slug 剔除面逐字同步——改一处必改三处）。
ROWS=$(psql "$DSN" -qAt -F $'\x1f' -c "
SELECT id, note FROM ops_ticket_desk
WHERE dispatchable AND tkind='工单'
AND note !~ '^工单/(dispatch|spawn|runbook|selfwatch|self-|stall|stuck|session|seat|pin-refuse|landq|queue|host-consol|close-finality|meta-|digest-|canon|kb-|org-|frame|probe|watch|beat|ticket-|worker-|cognition-|audit-|review-|lease|monologue|alert|feed-)'
ORDER BY COALESCE(level,2), deadline LIMIT 10" 2>/dev/null)

n=0
while IFS=$'\x1f' read -r tid tnote; do
	[ -n "$tid" ] || continue
	# spec 径解析：note 载 spec=/spec文件=/O-*.md 引用优先，缺席回落
	# O-<desk-id>-*.md glob（票号=ops_event id=prompt 件名前缀）。
	spec=$(printf '%s' "$tnote" | grep -oE '/opt/o-research/prompts/[^ ;）)"'"'"']+' | head -1)
	if [ -z "$spec" ]; then
		spec=$(printf '%s' "$tnote" | grep -oE 'O-[0-9]+-[A-Za-z0-9_.-]+\.md' | head -1)
		case "$spec" in /*) ;; ?*) spec="$PROMPTS_DIR/$spec" ;; esac
	fi
	[ -f "${spec:-/dev/null}" ] || \
		spec=$(find "$PROMPTS_DIR" -maxdepth 1 -name "O-${tid}-*.md" 2>/dev/null | head -1)
	# 道判——spec 缺席/obj 无路径词/不可判=REVIEW fail-closed。
	review=1
	if [ -n "$spec" ] && [ -f "$spec" ]; then
		obj=$(grep -m1 '^obj:' "$spec" 2>/dev/null | sed 's/^obj:[[:space:]]*//')
		is_free_obj "$obj" && review=0
	fi
	[ "$review" = 1 ] || continue
	[ "$n" -ge "$BATCH_MAX" ] && break # 上界截批——候选序=拾桌序
	n=$((n+1))
	slug=$(printf '%s' "$tnote" | grep -oE '工单/[^:：]+' | head -1 | sed 's|工单/||')
	[ "$COUNT_ONLY" = 1 ] || \
		printf '%s|%s|%s\n' "$tid" "$slug" "${spec:--}"
done <<< "$ROWS"
if [ "$COUNT_ONLY" = 1 ]; then echo "$n"; fi
