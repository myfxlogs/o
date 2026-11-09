#!/usr/bin/env bash
# mine.sh —— 改进矿工（纪元3提案半产件，D-0045 唤知管道一段）。
# 语义：档案挖掘→系统改进提案升件——RSI 反应半(信号→处置)之外补
# 生成半:拒因谱/败因复发/卡壳管线达阈→ops_event 升件(改进提案类)
# →唤知会话评估处置(销=判据不成立 或 变体改码→戳→部署)。
# 规则面 v1（机械判据,阈值 env 可调）:
#   R1 评估拒因主导: 7d fin_eval_reject 单 reason≥MINE_R1_N(10)
#      且占比≥60% ——判读=族覆盖缺口或阈值失配待评估
#   R2 执行败因复发: 7d fin_execution failed/ambiguous 同
#      venue_err_code≥MINE_R2_N(3) ——判读=系统性病灶待根修
#   R3 卡壳管线: probe>2d 且该期无 pass 凭据 count>0 ——
#      判读=毕业域覆盖缺口
# 去重契：同规则+同主体未决升件(无后随处置行引 升件#id)在场→不重发;
#   已销同主体窗口内(7d)不重发——销过的提案有处置判据在案。
# 边界：只产信号不改码；零命中静默；DSN 缺=fail-closed 退。
set -u
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$DSN" ] || { echo "[mine] FIN_PG_DSN 读不出($ENVF)——fail-closed" >&2; exit 1; }

R1_N=${MINE_R1_N:-10}; R2_N=${MINE_R2_N:-3}
# psql -c 不认 :var 占位（dispatch 修过同款）——阈值整数化后内插。
[[ "$R1_N" =~ ^[0-9]+$ ]] || R1_N=10
[[ "$R2_N" =~ ^[0-9]+$ ]] || R2_N=3

# 提案落账+去重一体:同契约键('改进提案:Rn:subj')未决或 7d 内已发不重发。
# 同契=note 前缀 '%' 匹配——subj 含特殊字符先按字面值转义在下拼接。
emit() { # $1=rule $2=subj $3=note
	local key="改进提案:$1:$2"
	local dup
	dup=$(psql "$DSN" -qAt -v key="${key}%" <<'SQL' 2>/dev/null
SELECT count(*) FROM ops_event e WHERE e.event='升件'
 AND e.ts > now() - interval '7 days' AND e.note LIKE :'key'
SQL
)
	[ "${dup:-0}" -gt 0 ] && return 0
	psql "$DSN" -qAt -v n="$3" <<'SQL' >/dev/null 2>&1
INSERT INTO ops_event(seat,event,ticket,note,esc_class)
VALUES('改进矿工','升件','', :'n', '改进提案')
SQL
	[ "${dup:-0}" -gt 0 ] || echo "[mine] 提案落账: $key"
}

# R1 拒因主导
while IFS='|' read -r r n tot; do
	[ -n "$r" ] || continue
	emit "R1" "$r" "改进提案:R1:$r 拒因主导——7d fin_eval_reject ${n}/${tot} 件($(( n*100/tot ))%),判读=族覆盖缺口或阈值失配;处置径=评估判据成立否,成立→销+立变体提案(新模板/参数),不成立→销"
done < <(psql "$DSN" -qAt -c "
WITH g AS (SELECT reason,count(*) n FROM fin_eval_reject
  WHERE ts>now()-interval '7 days' GROUP BY reason),
t AS (SELECT sum(n) tot FROM g)
SELECT g.reason,g.n,t.tot FROM g,t
 WHERE g.n>=${R1_N} AND g.n*100>=60*t.tot ORDER BY 2 DESC" 2>/dev/null)

# R2 执行败因复发
while IFS='|' read -r c n; do
	[ -n "$c" ] || continue
	emit "R2" "$c" "改进提案:R2:$c 执行败因复发——7d failed/ambiguous ${n}件,判读=系统性病灶非偶发;处置径=核 venue_err_msg+台账对齐,已修→销,未修→变体改码"
done < <(psql "$DSN" -qAt -c "
SELECT venue_err_code,count(*) FROM fin_execution
 WHERE created_at>now()-interval '7 days' AND state IN ('failed','ambiguous')
   AND venue_err_code IS NOT NULL
 GROUP BY venue_err_code HAVING count(*)>=${R2_N} ORDER BY 2 DESC" 2>/dev/null)

# R3 卡壳管线(probe>2d 无同柄凭据)
while IFS='|' read -r n; do
	[[ "$n" =~ ^[0-9]+$ ]] && [ "$n" -gt 0 ] || continue
	emit "R3" "probe-stale" "改进提案:R3:probe-stale 毕业管线卡壳——probe>2d 策略 ${n}件且无 pass 凭据,判读=演习域覆盖缺口或凭据产线断;处置径=查 template_ref 覆盖面"
done < <(psql "$DSN" -qAt -c "
SELECT count(*) FROM fin_strategy s WHERE s.state='probe'
 AND s.entered_at<now()-interval '2 days'
 AND NOT EXISTS (SELECT 1 FROM fin_exec_drill d
   WHERE d.strategy_ref=s.id AND d.verdict='pass'
     AND d.ts>s.entered_at-interval '2 days')" 2>/dev/null)
exit 0
