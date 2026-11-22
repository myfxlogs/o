#!/usr/bin/env bash
# escalate.sh —— 纪元3容错自报件（ops_esc.sh 继任——scripts/meta 随体裁撤,
# 探测层升件写径归唤知域）。工单O-30514 升件写径归一：语义正本迁库内
# 函件 ops_esc()（migrations/fin_0228——note=prefix||'——'||detail、
# 24h note 前缀同窗去重、class/scope 空→NULL、成功返行 id），本壳
# 只剩 DSN 发现+psql -v 参数化喂参——引号问题结构性绝根（原 QC()
# 手搓转义退役）。6 参签名不变（全仓调用面兼容）。
# 用法: escalate.sh <seat> <ticket> <class> <scope> <prefix> <detail>
#   seat=写席位(纪元3探测件建议'探测席');ticket=归属对象(无传'');
#   class/scope=判例匹配键(''=无类升件);prefix=去重键兼note头;
#   detail=note尾段(前缀与分隔符函件统一加)。
# 末环限度:写径败静默退(升件不能再升件——结构性限度在案);
# 函件返 NULL=同窗已报不写,非写败。
set -u
SEAT=${1:?usage: escalate.sh <seat> <ticket> <class> <scope> <prefix> <detail>}
TICKET=${2:-}; CLS=${3:-}; SCP=${4:-}; PREFIX=${5:?missing prefix}; DETAIL=${6:-}

FIN_DSN=
for ef in "${OPS_ESC_ENV_FILE:-}" "${REPORT_ENV_FILE:-}" \
          /opt/o-research/env /home/ubuntu/o-research/env; do
	[ -n "$ef" ] && [ -f "$ef" ] || continue
	FIN_DSN=$(grep '^FIN_PG_DSN=' "$ef" 2>/dev/null | cut -d= -f2- | tr -d '"')
	[ -n "$FIN_DSN" ] && break
done
[ -n "$FIN_DSN" ] || exit 0

# 函件径：:'var' 由 psql 做字面量转义——note/scope 内嵌引号不再可能
# 截断 SQL（#29766 引号断径病灶的结构性根修）；-qAt 旗形=deploy_watch
# esc() 主径识别锚（deploy_watch_test 桩按整词 -qAt 分径，勿并旗）。
# stdin 喂串不 -c：psql -c 不做 -v 变量插值（串面直发服务端，:'var'
# 原样成 SQL 语法错=写径全断——esc缺/esc写败心跳实证），stdin 径
# 插值照常且转义语义不变。
printf '%s\n' "SELECT ops_esc(:'seat',:'ticket',:'cls',:'scp',:'prefix',:'detail')" \
	| psql "$FIN_DSN" -qAt \
	-v seat="$SEAT" -v ticket="$TICKET" -v cls="$CLS" \
	-v scp="$SCP" -v prefix="$PREFIX" -v detail="$DETAIL" \
	>/dev/null 2>&1 || true # tolerate: 自报末环——写败静默退（结构性限度在案）
exit 0
