#!/usr/bin/env bash
# closeoutcert.sh —— 收口「调用径实证」闸（工单O-32269）：
# 完工回执/施工收口行 ∧ 支 diff 命中触发类（CREATE FUNCTION/.sh 新改/
# psql 调用径）∧ note 缺「调用径实证:」段 → 审计/closeout_callpath
# 升件——桩测≠调用径（O-30514 实证：ops_esc() 函件语义复刻审过，
# psql -c 不做 -v 插值调用径未实测，esc 写径双径全断生产复踩
# #32177/fix/esc-psql-cvar）。纯 Go/文档码件豁免（桩测覆盖调用径
# ——spec 钦定）。
# 落点=dispatch.sh 拍前预扫（同 mine/digestspawn/sessionpulse——
# 15min 拍距同宿点=细胞件不立 resident_units）；在办/日计语义不
# 涉——本件=审计面机械查，违例批升件载「回执#id」供销账对拍。
# 用法:
#   closeoutcert.sh            扫描拍（dispatch 接线面）
#   closeoutcert.sh --check <sha> ['<note>']   单件判（复审/审计席
#     手跑面——rc0=过/豁免 rc1=触发缺实证 rc2=sha不可解 fail-open不猜）
# 阈值写死常量可调：SCAN_DAYS=7 扫面窗 / CERT_EFFECT=生效锚日
# （前件豁免——规则生效前收口件不追诉，升件面只报不修）。
set -u
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
REPO_DIR="${O_REPO_DIR:-/opt/o}"
SCAN_DAYS=7      # 收口词扫面窗（天）——写死常量：覆盖复审→合流全程，可调改此
CERT_EFFECT="${O_CERT_EFFECT:-2026-11-23}" # 实证段义务生效锚（≈本件落地日）——前件豁免，可调改此/env覆写
ESC_CLASS='审计'; ESC_SCOPE='closeout_callpath'

sha_of() { # $1=note → 首个40hex（收口形恒载 sha=40位全sha——规约D节）
	grep -oE '[0-9a-f]{40}' <<<"$1" | head -1
}

diff_trigger() { # $1=sha → rc0=命中触发类 rc1=豁免(纯码件) rc2=不可判
	local sha=$1 mb ns patch adds
	git -C "$REPO_DIR" cat-file -e "${sha}^{commit}" 2>/dev/null || {
		git -C "$REPO_DIR" fetch origin "$sha" --quiet 2>/dev/null \
			|| git -C "$REPO_DIR" fetch origin --quiet 2>/dev/null # tolerate: 拉取败=下判不可解
		git -C "$REPO_DIR" cat-file -e "${sha}^{commit}" 2>/dev/null || return 2
	}
	mb=$(git -C "$REPO_DIR" merge-base origin/master "$sha" 2>/dev/null) || return 2
	ns=$(git -C "$REPO_DIR" diff --name-status "$mb" "$sha" 2>/dev/null) || return 2
	# ①.sh 新/改（name-status A/M 态——D 态删件不入触发面 spec 字面）
	grep -E '^[AM][[:space:]].*\.sh([[:space:]]|$)' <<<"$ns" && return 0
	patch=$(git -C "$REPO_DIR" diff "$mb" "$sha" 2>/dev/null) || return 2
	adds=$(grep -E '^\+' <<<"$patch" | grep -vE '^\+\+\+')
	[ -n "$adds" ] || return 1 # 零增行（纯删/空diff）=无从触②③——豁免
	# ②diff 增行含 CREATE [OR REPLACE] FUNCTION（大小写不敏感）
	grep -qiE 'create[[:space:]]+(or[[:space:]]+replace[[:space:]]+)?function' <<<"$adds" && return 0
	# ③diff 增行含 psql 调用形（命令位 psql+参数/连接串——行首或
	#   管道/&&/||/;/$( 后接；注释行提及 psql 词不入形）
	grep -qE '^\+[[:space:]]*psql[[:space:]"]' <<<"$adds" && return 0
	grep -qE '[|&;(`][[:space:]]*psql[[:space:]"]' <<<"$adds" && return 0
	return 1
}

note_has_evidence() { # $1=note → rc0=「调用径实证:」段在且形合规
	# 形闸非真闸：标记+其后≥20字符且含数字（输出摘录形——真伪归
	# 复审/审计判词面，spec ban 禁「描述调用径」替代真实输出）。
	grep -qE '调用径实证[:：]' <<<"$1" || return 1
	local seg
	seg=$(sed -nE 's/.*调用径实证[:：]//p' <<<"$1")
	[ "${#seg}" -ge 20 ] || return 1
	grep -q '[0-9]' <<<"$seg"
}

if [ "${1:-}" = "--check" ]; then
	[ -n "${2:-}" ] || { echo "usage: closeoutcert.sh --check <sha> ['<note>']" >&2; exit 2; }
	diff_trigger "$2"; trc=$?
	[ "$trc" = 2 ] && { echo "[closeoutcert] sha=$2 不可解——fail-open不猜" >&2; exit 2; }
	[ "$trc" = 1 ] && { echo "[closeoutcert] sha=$2 豁免——非触发类（纯码件桩测覆盖调用径）"; exit 0; }
	note_has_evidence "${3:-}" \
		&& { echo "[closeoutcert] sha=$2 触发类+实证段在——过"; exit 0; }
	echo "[closeoutcert] sha=$2 触发类+实证段缺——要件缺不审过" >&2; exit 1
fi

[ -n "$DSN" ] || { echo "[closeoutcert] FIN_PG_DSN 读不出($ENVF)——fail-closed不猜" >&2; exit 1; }

# ── 扫描拍：近 SCAN_DAYS 收口词行（生效锚后）→逐件判→违例批升件 ──
CAND=$(psql "$DSN" -qAt -F'|' -c "
SELECT e.id, e.note FROM ops_event e
WHERE e.note ~ '^(完工回执|\\[施工收口：)'
  AND e.ts > now() - interval '${SCAN_DAYS} days'
  AND e.ts > TIMESTAMPTZ '${CERT_EFFECT}'
ORDER BY e.id" 2>/dev/null) # tolerate: 查询败=空集本拍零监控——fail-open不阻主径
[ -n "$CAND" ] || exit 0

# 已报集：同 scope 历史升件 note 载「回执#id」=该回执已升不重报
# （回执行 append-only 不可补证——违例一次报死，处置席裁豁免/补件）。
SEEN=$(psql "$DSN" -qAt -c "SELECT note FROM ops_event
	WHERE event='升件' AND esc_scope='$ESC_SCOPE'" 2>/dev/null \
	| grep -oE '回执#[0-9]+' | sort -u) # tolerate: 查败=空集照报不漏

# origin ref 一次性刷（fetch 败=用现有 ref 继续——merge-base 仍可解
# 已推支；逐件不重 fetch 防 N×fetch 放大）。
git -C "$REPO_DIR" fetch origin --quiet 2>/dev/null # tolerate: 败不阻——现有 ref 面已够多数判

VIOL=""
while IFS='|' read -r rid rnote; do
	[ -n "$rid" ] || continue
	sha=$(sha_of "$rnote")
	[ -n "$sha" ] || continue # 无 sha=收口形已残——滞留探针域非本件
	grep -qxF "回执#$rid" <<<"$SEEN" && continue
	diff_trigger "$sha"; trc=$?
	[ "$trc" != 0 ] && continue # rc1 豁免 / rc2 不可判 fail-open
	note_has_evidence "$rnote" && continue
	VIOL="${VIOL:+$VIOL; }回执#$rid@${sha:0:9}"
done < <(printf '%s\n' "$CAND")

[ -n "$VIOL" ] || exit 0
"$SCRIPT_DIR/escalate.sh" '探测席' '' "$ESC_CLASS" "$ESC_SCOPE" \
	"$ESC_SCOPE" \
	"收口调用径实证缺——$VIOL——触发类diff(.sh新改/CREATE FUNCTION/psql调用径)收口词未附真实调用回执摘录(规约D节):处置=核支diff补实证段(回执不可改→补证行引回执#id)∨判豁免销——探针只报不修" \
	|| true # tolerate: 升件写败下拍重试——监控末环不自激
exit 0
