#!/usr/bin/env bash
# verdict_reflow.sh —— 诊断判词归流确定性执行机（GW-001 段D4——归流执行
# 从裁定席 SKILL 会话段剥离为机械轨件；判断归席（DM 裁决/驳回行），
# 机器只执行既裁：扫未归流裁决/驳回诊断草案行→翻案/INSERT draft/
# 三轮升件。裁定席会话零归流动作（SKILL 归流段退役——双执行者禁线）。
#
# 判据语义逐字承接（源=裁定席 SKILL 归流执行段退役版）：
#   扫描  ops_event event∈('裁决','驳回') note LIKE '诊断草案 fin_econ_lesson#%'
#   幂等键 ruling#<rid> —— 归流行 note='ruling#<rid> fin_econ_lesson#<lid>
#     <动作>'，有归流行=已归流跳过。并存期双执行者（本脚本×在飞裁定席
#     旧 SKILL 会话）防双归流三层：①归流行=消费标记先到先得；②draft 侧
#     fin_ticket_draft pending 防重唯一索引（fin_0148）DB 级硬闸——后到
#     INSERT 整语句败退下拍重扫按并入判收编；③翻案 status='drafted'
#     CAS——先到翻案者赢，后到按断点/让位径走。零重复零丢失，实证面=
#     tests/verdict_reflow_test.sh 场景⑥⑦。
#   通过 → 并入判：同 signature='rootcause:<family>' materialized 票
#     latest ops_event 非完成/作废→只翻案 verdict_ref='diag:<fam>:DM裁#
#     <rid>:并入在飞票<ticket>'；无在飞→同拍原子 INSERT draft（payload
#     携 ruling_id 追溯键+与 rootcause 归流模板同构）+翻案 verdict_ref=
#     'diag:<fam>:DM裁#<rid>:修法票草稿draft#<id>'——物料化归
#     ticketdraft 既有尾段（挂 o-mech 尾窗 reflow 在物料化窗前——同拍
#     出票），批准权=COS/DM 零自派。
#   驳回 → 错因入案卷：diag_state=rejected+reject_reason+diag_attempts+1
#     status 回 pending 再诊断（携错因不重蹈）；attempts≥3→翻 verdicted
#     verdict_ref='diag:驳回超限:<错因>'+ops_event 无类升件行（ops_esc
#     类 scope 传空——诊断草案面永不入判例自动裁，人机分界防自批回环）。
#     升件先于翻案（ops_esc 24h 前缀去重——断点重入零重复升件；翻案成功
#     蕴含升件已发）。
#   收尾 → ops_event 归流行（seat=MECH）——幂等标记+审计链；翻案
#     status→verdicted 同拍 ops_decision 镜像照旧（T-0415 触发器通用机制
#     不另设）。
#
# 断点收敛（半成品窗交叠面——crash 窗不丢不重）：
#   ①案行缺（永久性）→归流标记行留痕消费该 ruling（不空转不重扫）；
#   ②翻案已落归流行缺（翻案→归流间断点）→verdict_ref 携 DM裁#<rid>/
#     驳回超限 实证→只补归流行零重动作（draft 断点续账从 verdict_ref
#     draft#<id> 提取——INSERT 与翻案同语句原子，翻案在=draft 在）；
#   ③案已他裁（他 ruling 先归流/他径翻案）→让位标记行（幂等让位）。
# 败面：扫描败=exit 1（mech 尾窗升件面自理）；单 ruling 处理中 psql 败=
#     该 ruling 本拍跳过下拍重扫（fail-soft 不写标记——查询败≠已归流）；
#     malformed note（无 通过/驳回 词）=计数+WARN 留原行（会话误写径
#     人工纠——机器不强裁不吞行）。
set -u
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$FIN_DSN" ] || exit 78
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
OPS_ESC_SH=${O_OPS_ESC_SH:-"$SCRIPT_DIR/ops_esc.sh"}

QRC=0
q() { # $1=SQL → stdout；QRC=psql 退码（查败≠空集——调用面分径）
	local _o
	_o=$(psql "$FIN_DSN" -qAt -c "$1" 2>/dev/null)
	QRC=$?
	printf '%s' "$_o"
}
sq() { printf '%s' "${1:-}" | sed "s/'/''/g"; }

rows=$(q "SELECT id||'|'||event||'|'||COALESCE(ticket,'')||'|'||note FROM ops_event
	WHERE event IN ('裁决','驳回') AND note LIKE '诊断草案 fin_econ_lesson#%'
	ORDER BY id")
[ "$QRC" -eq 0 ] || { echo "[reflow] 扫描败 rc=$QRC——本拍弃下拍续" >&2; exit 1; }

N_ACT=0; N_SKIP=0; N_YIELD=0; N_MALF=0

# 归流收尾标记（$1=rid $2=ruling ticket $3=lid $4=动作一句话）
mark_reflow() {
	q "INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('MECH','归流','$(sq "$2")','ruling#${1} fin_econ_lesson#${3} $(sq "$4")')" >/dev/null
	[ "$QRC" -eq 0 ] || echo "[reflow] ruling#${1} 归流行落败 rc=$QRC——下拍断点续账重补" >&2
}

# 断点/让位径：案非 drafted 时只落标记零重动作（$4 断点实证描述）
settle_not_drafted() {
	local rid=$1 rticket=$2 lid=$3 st=$4 ref=$5
	case "$ref" in
		*"DM裁#${rid}:"*)
			mark_reflow "$rid" "$rticket" "$lid" "断点续账（$st 已在案：${ref}）——零重动作"
			;;
		*"驳回超限"*)
			mark_reflow "$rid" "$rticket" "$lid" "断点续账（驳回超限已在案）——零重动作"
			;;
		*)
			mark_reflow "$rid" "$rticket" "$lid" "让位（案已他裁 status=${st} ref=${ref:-空}）——幂等让位"
			N_YIELD=$((N_YIELD+1))
			;;
	esac
}

while IFS= read -r line; do
	[ -n "$line" ] || continue
	rid=${line%%|*}; rest=${line#*|}
	event=${rest%%|*}; rest=${rest#*|}
	rticket=${rest%%|*}; note=${rest#*|}
	case "$rid" in ''|*[!0-9]*) N_MALF=$((N_MALF+1)); continue ;; esac
	# note 机读形（CTO席 SKILL 诊断草案受理节）：诊断草案 fin_econ_lesson#<id> 通过|驳回：<…>
	lid=$(printf '%s' "$note" | sed -n 's/^诊断草案 fin_econ_lesson#\([0-9][0-9]*\) [通驳][过回]：.*/\1/p')
	verb=$(printf '%s' "$note" | sed -n 's/^诊断草案 fin_econ_lesson#[0-9][0-9]* \(通过\|驳回\)：.*/\1/p')
	if [ -z "$lid" ] || [ -z "$verb" ]; then
		N_MALF=$((N_MALF+1))
		echo "[reflow] ruling#${rid} note 无案锚/裁词机读形——留原行人工纠（不强裁）" >&2
		continue
	fi
	# 幂等闸+案态三查（归流行在=已归流；案态定动作/断点/让位）
	st=$(q "SELECT CASE WHEN EXISTS (SELECT 1 FROM ops_event WHERE event='归流'
			AND note LIKE 'ruling#${rid} %') THEN '__done__'
		ELSE COALESCE((SELECT status||'|'||COALESCE(verdict_ref,'') FROM fin_econ_lesson
			WHERE id=${lid} AND kind='rootcause_diag'),'__missing__') END")
	if [ "$QRC" -ne 0 ]; then
		echo "[reflow] ruling#${rid} 幂等闸查询败 rc=$QRC——本拍跳下拍续" >&2; continue
	fi
	case "$st" in
		__done__) N_SKIP=$((N_SKIP+1)); continue ;;
		__missing__)
			mark_reflow "$rid" "$rticket" "$lid" "案行缺——跳过留痕（ruling 引用失锚，人工纠）"
			continue ;;
	esac
	lst=${st%%|*}; ref=${st#*|}
	if [ "$lst" != "drafted" ]; then
		settle_not_drafted "$rid" "$rticket" "$lid" "$lst" "$ref"
		[ "$lst" = "verdicted" ] && N_SKIP=$((N_SKIP+1))
		continue
	fi

	if [ "$verb" = "驳回" ]; then
		reason=$(printf '%s' "$note" | sed -n 's/^诊断草案 fin_econ_lesson#[0-9][0-9]* 驳回：//p')
		att=$(q "SELECT COALESCE((numbers->>'diag_attempts')::int,0) FROM fin_econ_lesson WHERE id=${lid}")
		if [ "$QRC" -ne 0 ]; then
			echo "[reflow] ruling#${rid} attempts 读败 rc=$QRC——本拍跳下拍续" >&2; continue
		fi
		# 升件先于翻案（ops_esc 24h 前缀去重——断点重入零重复升件）
		if [ "$((att+1))" -ge 3 ]; then
			"$OPS_ESC_SH" MECH "$rticket" "" "" \
				"诊断草案驳回超限:fin_econ_lesson#${lid}" \
				"diag_attempts=$((att+1)) 达限——人裁接手；末次错因:${reason}" || true # tolerate: 升件写败静默（ops_esc 末环限度在案）
		fi
		nst=$(q "UPDATE fin_econ_lesson SET
			status = CASE WHEN COALESCE((numbers->>'diag_attempts')::int,0)+1 >= 3
				THEN 'verdicted' ELSE 'pending' END,
			verdict_ref = CASE WHEN COALESCE((numbers->>'diag_attempts')::int,0)+1 >= 3
				THEN 'diag:驳回超限:$(sq "$reason")' ELSE verdict_ref END,
			numbers = numbers || jsonb_build_object('diag_state','rejected',
				'reject_reason','$(sq "$reason")'::text,
				'diag_attempts',COALESCE((numbers->>'diag_attempts')::int,0)+1)
			WHERE id=${lid} AND kind='rootcause_diag' AND status='drafted'
			RETURNING status")
		if [ "$QRC" -ne 0 ]; then
			echo "[reflow] ruling#${rid} 驳回翻案败 rc=$QRC——本拍跳下拍续（升件去重面已兜）" >&2; continue
		fi
		if [ -n "$nst" ]; then
			if [ "$nst" = "verdicted" ]; then
				mark_reflow "$rid" "$rticket" "$lid" "驳回超限（attempts=$((att+1))——verdicted+无类升件已发，人裁接手）"
			else
				mark_reflow "$rid" "$rticket" "$lid" "驳回入案卷（attempts=$((att+1))——status 回 pending 再诊断携错因）"
			fi
			N_ACT=$((N_ACT+1))
		else
			st=$(q "SELECT status||'|'||COALESCE(verdict_ref,'') FROM fin_econ_lesson WHERE id=${lid}")
			[ "$QRC" -eq 0 ] || st="|"
			settle_not_drafted "$rid" "$rticket" "$lid" "${st%%|*}" "${st#*|}"
		fi
		continue
	fi

	# —— 通过径 ——
	fam=$(q "SELECT COALESCE(numbers->'diagnosis'->>'family','') FROM fin_econ_lesson WHERE id=${lid}")
	if [ "$QRC" -ne 0 ]; then
		echo "[reflow] ruling#${rid} family 读败 rc=$QRC——本拍跳下拍续" >&2; continue
	fi
	if [ -z "$fam" ]; then
		fam="novel"
		echo "[reflow] ruling#${rid} fin_econ_lesson#${lid} family 空——回落 novel 族" >&2
	fi
	# 并入判：同族 materialized 票 latest 非完成/作废→只翻案
	mt=$(q "SELECT COALESCE(ticket,'') FROM fin_ticket_draft
		WHERE signature='rootcause:$(sq "$fam")' AND status='materialized'
		ORDER BY ts DESC LIMIT 1")
	if [ "$QRC" -ne 0 ]; then
		echo "[reflow] ruling#${rid} 在飞票查询败 rc=$QRC——本拍跳下拍续" >&2; continue
	fi
	if [ -n "$mt" ]; then
		latest=$(q "SELECT event FROM ops_event WHERE ticket='$(sq "$mt")'
			ORDER BY ts DESC,id DESC LIMIT 1")
		if [ "$QRC" -ne 0 ]; then
			echo "[reflow] ruling#${rid} 票相位查询败 rc=$QRC——本拍跳下拍续" >&2; continue
		fi
		if [ "$latest" != "完成" ] && [ "$latest" != "作废" ]; then
			flipped=$(q "UPDATE fin_econ_lesson SET status='verdicted',
				verdict_ref='diag:$(sq "$fam"):DM裁#${rid}:并入在飞票$(sq "$mt")'
				WHERE id=${lid} AND kind='rootcause_diag' AND status='drafted'
				RETURNING id")
			if [ "$QRC" -ne 0 ]; then
				echo "[reflow] ruling#${rid} 并入翻案败 rc=$QRC——本拍跳下拍续" >&2; continue
			fi
			if [ -n "$flipped" ]; then
				mark_reflow "$rid" "$rticket" "$lid" "并入在飞票 ${mt}（同族判词并入——零新草案）"
				N_ACT=$((N_ACT+1)); continue
			fi
			st=$(q "SELECT status||'|'||COALESCE(verdict_ref,'') FROM fin_econ_lesson WHERE id=${lid}")
			[ "$QRC" -eq 0 ] || st="|"
			settle_not_drafted "$rid" "$rticket" "$lid" "${st%%|*}" "${st#*|}"
			continue
		fi
	fi
	# 修法票草案径：读草案载荷→同拍原子 INSERT draft+翻案（幂等三层同上）
	eref=$(q "SELECT entity_ref FROM fin_econ_lesson WHERE id=${lid}")
	hyp=$(q "SELECT COALESCE(numbers->'diagnosis'->>'hypothesis','') FROM fin_econ_lesson WHERE id=${lid}")
	fixs=$(q "SELECT COALESCE(numbers->'diagnosis'->>'fix_surface','') FROM fin_econ_lesson WHERE id=${lid}")
	evagg=$(q "SELECT COALESCE((SELECT string_agg(v,'；') FROM jsonb_array_elements_text(
		numbers->'diagnosis'->'evidence') v),'') FROM fin_econ_lesson WHERE id=${lid}")
	evrow=$(q "SELECT id FROM ops_event WHERE event='诊断草案'
		AND note LIKE '诊断草案 fin_econ_lesson#${lid} %'
		ORDER BY id DESC LIMIT 1")
	if [ "$QRC" -ne 0 ]; then
		echo "[reflow] ruling#${rid} 载荷读败 rc=$QRC——本拍跳下拍续" >&2; continue
	fi
	payload=$(python3 - "$fam" "$eref" "$hyp" "$fixs" "$evagg" "${evrow:-}" "$rid" <<'PYEOF'
import json, re, sys
fam, eref, hyp, fixs, evagg, evrow, rid = sys.argv[1:8]
pid = re.sub(r"[^0-9A-Za-z_-]", "", eref.split("#")[-1]) or "x"
print(json.dumps({
	"slug": f"rootcause-{fam}-{pid}",
	"title": f"根因诊断修法票:{hyp}"[:200],
	"signature": f"rootcause:{fam}",
	"kind": "rootcause_diag",
	"lesion_key": eref,
	"evidence_ids": [int(evrow)] if evrow.isdigit() else [],
	"evidence_note": (hyp + ("；证据链：" + evagg if evagg else "")),
	"evidence_query": evagg,
	"domain_hint": fixs,
	"ruling_id": rid,
}, ensure_ascii=False, separators=(",", ":")))
PYEOF
	)
	if [ -z "$payload" ]; then
		echo "[reflow] ruling#${rid} 载荷装配败——本拍跳下拍续" >&2; continue
	fi
	out=$(q "WITH ins AS (
		INSERT INTO fin_ticket_draft(signature,kind,lesion_key,payload)
		SELECT 'rootcause:$(sq "$fam")','rootcause_diag','$(sq "$eref")','$(sq "$payload")'::jsonb
		WHERE EXISTS (SELECT 1 FROM fin_econ_lesson WHERE id=${lid} AND status='drafted')
		  AND NOT EXISTS (SELECT 1 FROM fin_ticket_draft WHERE kind='rootcause_diag'
			AND status='pending' AND signature='rootcause:$(sq "$fam")'
			AND lesion_key='$(sq "$eref")')
		RETURNING id
	), upd AS (
		UPDATE fin_econ_lesson SET status='verdicted',
		  verdict_ref='diag:$(sq "$fam"):DM裁#${rid}:修法票草稿draft#' ||
		    COALESCE((SELECT id::text FROM ins),
		      (SELECT id::text FROM fin_ticket_draft WHERE kind='rootcause_diag'
			AND status='pending' AND signature='rootcause:$(sq "$fam")'
			AND lesion_key='$(sq "$eref")' ORDER BY id LIMIT 1),'缺')
		WHERE id=${lid} AND kind='rootcause_diag' AND status='drafted'
		RETURNING id
	)
	SELECT COALESCE((SELECT id::text FROM ins),'-')||'|'||(SELECT count(*) FROM upd)")
	if [ "$QRC" -ne 0 ]; then
		# 并存期竞态面：唯一索引拒=他执行者先落（fin_0148 硬闸开火）——
		# 本拍弃下拍重扫按幂等闸/并入判收编（fail-soft 不写标记）
		echo "[reflow] ruling#${rid} draft 原子径败 rc=$QRC（含 pending 防重唯一拒）——本拍跳下拍续" >&2
		continue
	fi
	ins_id=${out%%|*}; updn=${out##*|}
	if [ "$updn" = "1" ]; then
		if [ "$ins_id" = "-" ]; then
			adopted=$(q "SELECT COALESCE(verdict_ref,'') FROM fin_econ_lesson WHERE id=${lid}")
			mark_reflow "$rid" "$rticket" "$lid" "修法票草稿并入既有 pending（${adopted#*draft#}——防重索引收编零新行）"
		else
			mark_reflow "$rid" "$rticket" "$lid" "修法票草稿落 draft#${ins_id}（物料化归 ticketdraft 尾段，批准权=COS/DM）"
		fi
		N_ACT=$((N_ACT+1))
	else
		# CAS 败：自清本轮孤儿 draft（同语句原子插得—翻案败=无主）再断点/让位
		case "$ins_id" in
			-*|'') ;;
			*) q "DELETE FROM fin_ticket_draft WHERE id=${ins_id} AND status='pending'" >/dev/null
			   [ "$QRC" -eq 0 ] || echo "[reflow] ruling#${rid} 孤儿 draft#${ins_id} 自清败 rc=$QRC——下拍并入判收编" >&2 ;;
		esac
		st=$(q "SELECT status||'|'||COALESCE(verdict_ref,'') FROM fin_econ_lesson WHERE id=${lid}")
		[ "$QRC" -eq 0 ] || st="|"
		settle_not_drafted "$rid" "$rticket" "$lid" "${st%%|*}" "${st#*|}"
	fi
done <<<"$rows"

echo "[reflow] 本拍：动作=${N_ACT} 已归流跳过=${N_SKIP} 让位=${N_YIELD} malformed=${N_MALF}" >&2
exit 0
