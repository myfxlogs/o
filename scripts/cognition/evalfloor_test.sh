#!/usr/bin/env bash
# evalfloor_test.sh —— evalfloor.sh 审批带宽保底闸桩测（工单O-30294）。
# 桩 psql 按 SQL 词面分派应答（全量落 PSQL_LOG 可查）+桩 env 文件——
# 案件态经 STUB_* 环境变量注入，fail-open 全程无真库。
# 案件（spec 三案+闸内细分+O-31103 谓词案，全可重放）：
#   ①积压超龄∧零审批连拍∧无P0 → evalfloor 保底拍触发
#     （他件在飞不传 --sole=因子搭车；--sole 且无冷却=保底独唤）
#   ②阈下积压 → 不触发（evalage_h/eidle 因子仍机读出）
#   ③P0 在飞 → 让位（evalfloor 抑制，因子照出）
#   ④idle 阈下 → 不触发
#   ⑤--sole 冷却窗内 → evalfloor 抑制（防静默期保底独唤风暴）
#   ⑥app_config 阈可裁 + env 终裁覆盖 + 零积压静默
#   ⑦O-31103 sticky 谓词：积压腿走 FIN_PG_RO_DSN（fin_proposal 反联
#     须 o_fin_ro 读面）+SQL 载 NOT EXISTS(eval_id) 反联钉+
#     消费态双用例（有提案=不计积压静默/无提案=计积压出因子）
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
EF="$HERE/evalfloor.sh"
T=$(mktemp -d /tmp/evalfloorXXXXXX)
trap 'rm -rf "$T"' EXIT
PASS=0; FAILN=0
ok()  { PASS=$((PASS+1)); echo "  ok   $1"; }
bad() { FAILN=$((FAILN+1)); echo "  FAIL $1"; }

mkdir -p "$T/bin"
printf 'FIN_PG_DSN=stub://evalfloor\nFIN_PG_RO_DSN=stub-ro://evalfloor\n' > "$T/env"

# 桩 psql：按 -c SQL 词面+连接串分派应答；STUB_* 注入案件态。
# conn 判别：argv 含 stub-ro://=RO 径（积压腿），stub://=RW 径。
cat > "$T/bin/psql" <<'STUB'
#!/usr/bin/env bash
sql=""; conn=""; prev=""
for a in "$@"; do
	[ "$prev" = "-c" ] && sql="$a"
	case "$a" in stub-ro://*) conn=ro;; stub://*) [ "$conn" = ro ] || conn=rw;; esac
	prev="$a"
done
printf '%s\n' "$sql" >> "${PSQL_LOG:-/dev/null}"
case "$sql" in
	*fin_opp_eval*)
		if [ "$conn" = ro ]; then printf '%s\n' "${STUB_RO_BACKLOG:-${STUB_BACKLOG:-0|0}}"
		else printf '%s\n' "${STUB_BACKLOG:-0|0}"; fi ;;
	*app_config*)              printf '%s\n' "${STUB_CFG:-}" ;;
	*资金闸*)                  printf '%s\n' "${STUB_P0:-f}" ;;
	*evalfloor*)               printf '%s\n' "${STUB_LASTFLOOR:-999999999}" ;;
	*裁决*)                    printf '%s\n' "${STUB_IDLE:-0}" ;;
esac
exit 0
STUB
chmod +x "$T/bin/psql"
export PATH="$T/bin:$PATH" PSQL_LOG="$T/psql.log" REPORT_ENV_FILE="$T/env"
export STUB_BACKLOG STUB_RO_BACKLOG STUB_IDLE STUB_P0 STUB_CFG STUB_LASTFLOOR

run() { # run [--sole] —— evalfloor.sh stdout 入 OUT
	OUT=$("$EF" "$@" 2>/dev/null)
}
has() { printf '%s' "$OUT" | grep -qw "$1"; }

echo "== 案① 积压超龄∧零审批∧无P0 → 保底拍触发 =="
STUB_BACKLOG="254|31.7" STUB_IDLE="12" STUB_P0="f" STUB_CFG="" STUB_LASTFLOOR="999999999"
run
has evalfloor && has evalage_h=31.7 && has eidle=12 \
	&& ok "①a 他件在飞拍(无--sole)：evalfloor 搭车出" \
	|| bad "①a OUT=[$OUT]"
run --sole
has evalfloor && ok "①b --sole 无冷却：保底独唤拍出 evalfloor" \
	|| bad "①b OUT=[$OUT]"

echo "== 案② 阈下积压 → 不触发 =="
STUB_BACKLOG="3|5.2" STUB_IDLE="12" STUB_P0="f" STUB_CFG="" STUB_LASTFLOOR="999999999"
run; run --sole
has evalage_h=5.2 && ! has evalfloor \
	&& ok "② 阈下：evalfloor 抑制，evalage_h/eidle 因子仍机读出" \
	|| bad "② OUT=[$OUT]"

echo "== 案③ P0 在飞 → 让位 =="
STUB_BACKLOG="254|31.7" STUB_IDLE="12" STUB_P0="t" STUB_CFG="" STUB_LASTFLOOR="999999999"
run; run --sole
! has evalfloor && has evalage_h=31.7 \
	&& ok "③ P0在飞：evalfloor 让位，积压龄因子照出不误" \
	|| bad "③ OUT=[$OUT]"

echo "== 案④ idle 阈下 → 不触发 =="
STUB_BACKLOG="254|31.7" STUB_IDLE="3" STUB_P0="f" STUB_CFG="" STUB_LASTFLOOR="999999999"
run
! has evalfloor && has eidle=3 \
	&& ok "④ 静默3拍<阈8：evalfloor 抑制" \
	|| bad "④ OUT=[$OUT]"

echo "== 案⑤ --sole 冷却窗内 → evalfloor 抑制 =="
STUB_BACKLOG="254|31.7" STUB_IDLE="12" STUB_P0="f" STUB_CFG="" STUB_LASTFLOOR="60"
run --sole
! has evalfloor && has evalage_h=31.7 \
	&& ok "⑤a sole+冷却60s<7200s：保底标记抑制（因子照出）" \
	|| bad "⑤a OUT=[$OUT]"
run   # 他件在飞拍不传 --sole=不冷却
has evalfloor \
	&& ok "⑤b 他件在飞拍：evalfloor 搭车不受冷却" \
	|| bad "⑤b OUT=[$OUT]"

echo "== 案⑥ 阈可裁+零积压 =="
STUB_BACKLOG="254|31.7" STUB_IDLE="12" STUB_P0="f" STUB_LASTFLOOR="999999999"
STUB_CFG=$'cognition.evalfloor_age_h|48\ncognition.evalfloor_idle_beats|20'
run
! has evalfloor && ok "⑥a app_config 阈48h/20拍：31.7h∧12拍不触发" \
	|| bad "⑥a OUT=[$OUT]"
STUB_CFG=""
OUT=$(COG_EVALFLOOR_AGE_H=12 COG_EVALFLOOR_BEATS=4 "$EF" 2>/dev/null)
printf '%s' "$OUT" | grep -qw evalfloor \
	&& ok "⑥b env 终裁覆盖 12h/4拍：触发" \
	|| bad "⑥b OUT=[$OUT]"
STUB_BACKLOG="0|0" STUB_CFG=""
OUT=$("$EF" 2>/dev/null)
[ -z "$OUT" ] && ok "⑥c 零积压：静默无因子" || bad "⑥c OUT=[$OUT]"

echo "== 案⑦ O-31103 sticky 谓词：RO径+反联钉+消费态双用例 =="
# ⑦a 谓词结构钉：发出的积压 SQL 必须载 fin_proposal 反联
STUB_BACKLOG="1|5.0" STUB_IDLE="12" STUB_P0="f" STUB_CFG="" STUB_LASTFLOOR="999999999"
: > "$PSQL_LOG"
run
# 桩按整条 SQL 落行（内嵌换行成多物理行）——三词同现于积压腿唯一查询
grep -q 'fin_opp_eval' "$PSQL_LOG" \
	&& grep -q 'fin_proposal' "$PSQL_LOG" \
	&& grep -q 'NOT EXISTS' "$PSQL_LOG" \
	&& grep -q 'eval_id' "$PSQL_LOG" \
	&& ok "⑦a 积压SQL载 fin_proposal NOT EXISTS(eval_id) 反联" \
	|| bad "⑦a 谓词缺席 LOG=[$(cat "$PSQL_LOG")]"
# ⑦b RO 路由钉（双向）：RW桩给 254 积压、RO桩给 0 → 必须静默
#    （若误走 RW 径会出 evalage_h=31.7）；反向 RO 给真积压 → 因子出
STUB_RO_BACKLOG="0|0" STUB_BACKLOG="254|31.7"
OUT=$("$EF" 2>/dev/null)
[ -z "$OUT" ] && ok "⑦b1 积压腿走RO径：RO=0消费尽→静默（RW=254不误取）" \
	|| bad "⑦b1 OUT=[$OUT]"
STUB_RO_BACKLOG="2|9.9" STUB_BACKLOG="0|0"
OUT=$("$EF" 2>/dev/null)
printf '%s' "$OUT" | grep -qw 'evalage_h=9.9' \
	&& ok "⑦b2 反向：RO=2真未裁→evalage_h=9.9 出因子（RW=0不遮）" \
	|| bad "⑦b2 OUT=[$OUT]"
# ⑦c 消费态双用例（spec 钦定两件）：
#    proposed+有提案=已消费不计积压 → RO桩回 0|0 → 静默
#    proposed+无提案=真未裁计积压     → RO桩回 1|30.0 → evalage_h 出
STUB_RO_BACKLOG="0|0" STUB_BACKLOG="0|0" STUB_IDLE="12" STUB_P0="f" STUB_CFG="" STUB_LASTFLOOR="999999999"
OUT=$("$EF" 2>/dev/null)
[ -z "$OUT" ] && ok "⑦c1 全消费(proposed∧有提案)：不计积压静默" \
	|| bad "⑦c1 OUT=[$OUT]"
STUB_RO_BACKLOG="1|30.0"
OUT=$("$EF" 2>/dev/null)
printf '%s' "$OUT" | grep -qw 'evalage_h=30.0' \
	&& ok "⑦c2 真未裁(proposed∧无提案)：计积压 evalage_h=30.0" \
	|| bad "⑦c2 OUT=[$OUT]"
STUB_RO_BACKLOG=""

echo "== 结果 PASS=$PASS FAIL=$FAILN =="
[ "$FAILN" = 0 ]
