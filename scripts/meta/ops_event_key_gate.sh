#!/usr/bin/env bash
# ops_event_key_gate.sh —— ops_event ticket 键卫生闸（T-0499 根修件）：
# 拼接伪键（含空白/逗号族：空格/\n/\t/，/，或 T-NNNN 形态≥2 多票号）
# 在写面落库前 loud 拒写——非静默截断（截断会造新伪键）。判据与
# migrations/fin_0153 ops_ticket_phase 视图守卫同形双闸（写面拒+
# 投影面清）；尸源实证='T-0270\nT-0277' 形 grep -m1 -oE 多匹配换行串
# （条目 grep-m1-o-multimatch-newline-tid 在案）。
# 用法: ops_event_key_gate.sh <ticket> [ctx] [seat]
#   rc0=键净放行（空键 '' 合法——票外行通行证）；rc1=伪键已 loud 拒写。
#   loud=stderr 必达+ops_event '升件' 行尽力落（ticket='' 不入票域，
#   note 前缀「ticket键卫生拒写 <ctx> h=<md5-8>」24h 同键去重防刷屏）。
#   闸自身写径全尽力而为——env/DSN/psql 败只剩 stderr，绝不反噬调用面。
#   调用面约定：闸缺席（-x 判空）fail-open 放行——卫生面非链环，
#   OCI 单件部署无兄弟件先例（ops_esc escalate 径同律）在案。
set -u
TID=${1:-}; CTX=${2:-ops_event写面}; SEAT=${3:-MECH}

# 空键合法直通（票外升件/部署/巡检 META-* 行不受本闸语义影响——
# META-*/gauge 名等单 token 键同过：无空白/逗号/多票号形态）。
[ -z "$TID" ] && exit 0

dirty=
if [[ "$TID" =~ [[:space:],，] ]]; then dirty=1; fi
if [ -z "$dirty" ]; then
  n=$(printf '%s' "$TID" | grep -oE 'T-[0-9]{4}' | wc -l)
  [ "$n" -gt 1 ] && dirty=1
fi
[ -z "$dirty" ] && exit 0

QC() { printf '%s' "${1:-}" | sed "s/'/''/g"; }
H=$(printf '%s' "$TID" | md5sum | cut -c1-8)
# 预览=键前 200 字符空白归一（·）——原键含换行/制表符，预览防 SQL/日志
# 注入面（bash 展开定界——sed 按行读 \n 不入模式空间/tr 按字节切多
# 字节 '·' 双失效实证）。
PREVIEW=${TID:0:200}
PREVIEW=${PREVIEW//$'\n'/·}
PREVIEW=${PREVIEW//$'\t'/·}
PREVIEW=${PREVIEW//$'\r'/·}
PREVIEW=${PREVIEW// /·}
echo "ops_event_key_gate: 拒写伪键 ctx=$CTX key=$PREVIEW h=$H" >&2

DSN=
for ef in "${KEY_GATE_ENV_FILE:-}" "${OPS_ESC_ENV_FILE:-}" "${DM_ENV_FILE:-}" \
          "${REPORT_ENV_FILE:-}" /opt/o-research/env /home/ubuntu/o-research/env; do
  [ -n "$ef" ] && [ -f "$ef" ] || continue
  DSN=$(grep '^FIN_PG_DSN=' "$ef" 2>/dev/null | cut -d= -f2- | tr -d '"')
  [ -n "$DSN" ] && break
done
[ -n "$DSN" ] || exit 1
PSQL_BIN=${O_KEY_GATE_PSQL:-psql}
HEAD="ticket键卫生拒写 $CTX h=$H"
NOTE="$HEAD——key=$PREVIEW（写面落库前拒；截断会造新伪键故整行拒——T-0499）"
dup=$("$PSQL_BIN" "$DSN" -qAt -c \
  "SELECT 1 FROM ops_event WHERE event='升件' AND note LIKE '$(QC "$HEAD")%' AND ts>now()-interval '24 hours' LIMIT 1" \
  2>/dev/null) # tolerate: 去重查询败=空答→照常升（报侧 fail-open 不吞讯——ops_esc 同律）
if [ -z "$dup" ]; then
  "$PSQL_BIN" "$DSN" -qAt -c \
    "INSERT INTO ops_event(seat,event,ticket,note) VALUES('$(QC "$SEAT")','升件','','$(QC "$NOTE")')" \
    >/dev/null 2>&1 || true # tolerate: 升件末环败静默退——拒写事实已 stderr 在案，闸不反噬调用面
fi
exit 1
