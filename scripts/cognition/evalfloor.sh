#!/usr/bin/env bash
# evalfloor.sh —— 审批带宽保底因子件（工单O-30294）。
# 病灶：proposed→裁定=决策席会话人工件，救火信号可无限挤占审批带宽
#   （32h积压实证——CI红族+landq污锁+mt5停册P1连环吃满唤知40-45拍，
#   无保底闸）。本件=议程因子侧保底闸：fin_opp_eval proposed 最老龄
#   >阈 ∧ 审批静默≥阈拍 ∧ P0不在飞 → 因子出 evalfloor 保底拍。
# 出件（stdout 空格分隔议程因子，dispatch 并入 trigger 机读面）：
#   evalage_h=<h>  proposed 积压最老龄（小时，一位小数）——积压在案恒出；
#   eidle=<n>      审批静默拍数（末次审批处置距今 ÷900s dispatch拍）——同上；
#   evalfloor     保底拍标记——仅闸合时出（龄超阈∧静默超阈∧P0不在飞）。
# 阈面（app_config 可裁，键名施工自定）：
#   cognition.evalfloor_age_h       proposed 最老龄阈(h)，默认24=黄档；
#   cognition.evalfloor_idle_beats  连续零审批拍阈，默认8（≈2h@15min拍）；
#   env COG_EVALFLOOR_AGE_H / COG_EVALFLOOR_BEATS 终裁覆盖（同 dispatch 例）。
# 审批处置判词（零审批静默的销账面）：裁决族判词行 event IN
#   (裁决/裁定/驳回——fin_0129 决策记忆裁词全集；参数调整=DM写径机产
#   参数回灌非审批件不计) ∨ 处置行 ticket='hypo'(hypo decide 审计) ∨
#   处置行 note 携裁定/审批/假想/hypo/提案/评估词干（裁定向处置）。
# P0在飞让位（救命恒先）：ops_esc_pending 命脉最危子集
#   (死刑/资金闸/对账/账链断——账链与死刑面) ∪ fin_guard_close_status
#   守位强平在飞(state='closing'∧attempts>=2，dispatch_self 同锚)；
#   非P0救火件(CI红/landq/P1票/部署断流等)不挤保底——因子与救火因子
#   并列不互斥。P0 判读败=按在飞让位（fail-closed 救命恒先）。
# --sole：dispatch 本拍无他触发因子时传——冷却闸生效：距上枚
#   evalfloor 唤起<阈拍×900s 不重复保底（防静默期保底独唤风暴；
#   他件在飞拍不传 --sole=因子搭车零成本不冷却）。
# 用法：dispatch.sh 每拍调；桩测 evalfloor_test.sh。
# fail-open：库读败=静默无因子（保底件非主触发面，dispatch 不受阻）。
set -u
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$DSN" ] || exit 0
# 真积压谓词须读 fin_proposal（roster_rw 无读权——本腿走 FIN_PG_RO_DSN
#   o_fin_ro 审计读面，fin_opp_eval+fin_proposal 双表可读实证）；
#   RO DSN 缺席=fail-open 静默（同库读败语义）。
RODSN=$(grep '^FIN_PG_RO_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$RODSN" ] || exit 0
SOLE=0; [ "${1:-}" = "--sole" ] && SOLE=1

# ① proposed 真积压：计数+最老龄(h，一位小数)。零积压=无因子静默；
#    读败(空)=同静默——积压龄无从谈起不出假因子。
#    sticky 语义钉死（工单O-31103）：verdict='proposed' 为终态词——提案
#    径建 fin_proposal 不翻 verdict，schema 无 adopted 态；裸计 proposed
#    把已消费 eval 也算积压（evalage_h 实测 621h sticky 龄≠真积压龄）。
#    真未裁=proposed ∧ NOT EXISTS(fin_proposal p.eval_id=e.id)——
#    fin_proposal 全行带 eval_id（2789/2789 实证），建行=消费落账，
#    退役/armed 态皆已裁（裁果归 proposal 生命周期非 eval 积压）。
read -r EVAL_N EVAL_AGE_H < <(psql "$RODSN" -qAt -c "
SELECT count(*), COALESCE(round(EXTRACT(EPOCH FROM now()-min(e.ts))/3600,1),0)
FROM fin_opp_eval e
WHERE e.verdict='proposed'
  AND NOT EXISTS (SELECT 1 FROM fin_proposal p WHERE p.eval_id=e.id)" 2>/dev/null | tr '|' ' ')
EVAL_N=${EVAL_N:-0}; EVAL_AGE_H=${EVAL_AGE_H:-0}
case "$EVAL_N" in ''|*[!0-9]*) exit 0;; esac
[ "$EVAL_N" -gt 0 ] || exit 0

# ② 审批活动账：末次审批处置距今拍数（拍=900s dispatch 建议档）。
#    读败=0=「刚处置过」向不保底（fail-closed 不误保底）。
EVAL_IDLE=$(psql "$DSN" -qAt -c "
SELECT COALESCE(EXTRACT(EPOCH FROM now()-COALESCE(max(ts),'epoch'::timestamptz))/900,0)::int
FROM ops_event
WHERE event IN ('裁决','裁定','驳回')
   OR (event='处置' AND (ticket='hypo'
       OR note ~ '裁定|审批|假想|hypo|提案|评估'))" 2>/dev/null)
EVAL_IDLE=${EVAL_IDLE:-0}
case "$EVAL_IDLE" in ''|*[!0-9]*) EVAL_IDLE=0;; esac

# ③ 阈面：app_config 可裁 → env 终裁覆盖 → 内建默认兜底。
AGE_H=24; IDLE_N=8
while IFS='|' read -r k v; do
	case "$k" in
		cognition.evalfloor_age_h) AGE_H=$v ;;
		cognition.evalfloor_idle_beats) IDLE_N=$v ;;
	esac
done <<EOF
$(psql "$DSN" -qAt -c "
SELECT key, value #>> '{}' FROM app_config
WHERE key IN ('cognition.evalfloor_age_h','cognition.evalfloor_idle_beats')" 2>/dev/null)
EOF
AGE_H=${COG_EVALFLOOR_AGE_H:-$AGE_H}; IDLE_N=${COG_EVALFLOOR_BEATS:-$IDLE_N}
[[ "$AGE_H" =~ ^[0-9]+(\.[0-9]+)?$ ]] || AGE_H=24
[[ "$IDLE_N" =~ ^[0-9]+$ ]] || IDLE_N=8

# ④ P0 在飞判：命脉最危子集升件在案 ∪ 守位强平在飞。
#    读败=按在飞让位（fail-closed——救命判据不可读时不强占保底拍）。
P0Q=$(psql "$DSN" -qAt -c "
SELECT (EXISTS(SELECT 1 FROM ops_esc_pending
   WHERE esc_class IN ('死刑','资金闸','对账','账链断'))
   OR EXISTS(SELECT 1 FROM fin_guard_close_status
   WHERE state='closing' AND attempts>=2))" 2>/dev/null)
P0=0; [ "$P0Q" = "t" ] && P0=1; [ -z "$P0Q" ] && P0=1

# ⑤ 保底闸合判：龄超阈 ∧ 静默超阈拍 ∧ P0不在飞。
DUE=0
if [ "$P0" = 0 ] && awk -v a="$EVAL_AGE_H" -v t="$AGE_H" 'BEGIN{exit !(a>t)}' \
	&& [ "$EVAL_IDLE" -ge "$IDLE_N" ]; then
	DUE=1
fi

# ⑥ --sole 冷却闸：距上枚 evalfloor 唤起<阈拍×900s 抑制保底标记
#    （因子 evalage/eidle 照出——账不抹，只收拍）。
if [ "$SOLE" = 1 ] && [ "$DUE" = 1 ]; then
	LAST_FLOOR=$(psql "$DSN" -qAt -c "
SELECT COALESCE(EXTRACT(EPOCH FROM now()-max(ts))::bigint,999999999)
FROM ops_event WHERE seat='唤知机' AND event='唤起'
  AND note LIKE '%evalfloor%'" 2>/dev/null)
	LAST_FLOOR=${LAST_FLOOR:-999999999}
	case "$LAST_FLOOR" in ''|*[!0-9-]*) LAST_FLOOR=999999999;; esac
	[ "$LAST_FLOOR" -lt $((IDLE_N*900)) ] && DUE=0
fi

printf 'evalage_h=%s eidle=%s' "$EVAL_AGE_H" "$EVAL_IDLE"
[ "$DUE" = 1 ] && printf ' evalfloor'
echo
