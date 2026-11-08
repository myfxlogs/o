#!/usr/bin/env bash
# ceo_northstar_test.sh —— T-0465 北极星账桩测（生产谱形 fixture）：
#  A 账页指标值+背离成行——七项取数断言/页面字段/议题行词形
#    （seat=CEO event=议题 ticket='' note='北极星背离:<键>=<值> vs 基线<基线>
#    ——问链引T-0464'）/快照行 kv 面/为什么型调问链+容量型不调/dep 探测
#    OPS_ESC_SH=/bin/true 只读压制；
#  B 24h 同指标去重——dup 桩全返 1 → 议题零落、快照照记；
#  C 全绿拍——零背离零议题、问链不调、快照照记、页载值；
#  D ceo_queue 议题读面——UNION 议题#id 出列+regex 引证关件（#5 不被 #50 误关）；
#  E ceo_tick 三步序挂点——北极星扫先于队闸（空队拍 northstar 仍跑）。
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
META="$ROOT/scripts/meta"
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT

# ══ 公共桩件 ════════════════════════════════════════════════════════
setup_ns() { # $1=场景目录——建 bin/psql+env+deploy/why 桩，导出 FX_*
  local d=$1; mkdir -p "$d/bin" "$d/out"
  cat > "$d/bin/psql" <<'EOF'
#!/usr/bin/env bash
query="${*: -1}"
case "$query" in
  *"INSERT INTO ops_event"*) printf '%s\n' "$query" >> "$SQL_LOG"; exit 0;;
  *"/*ns:pnl*/"*)    printf '%s\n' "$FX_PNL";;
  *"/*ns:armed*/"*)  printf '%s\n' "$FX_ARMED";;
  *"/*ns:open*/"*)   printf '%s\n' "$FX_OPEN";;
  *"/*ns:esc*/"*)    printf '%s\n' "$FX_ESC";;
  *"/*ns:water*/"*)  printf '%s\n' "$FX_WATER";;
  *"/*ns:prev*/"*)   printf '%s\n' "$FX_PREV";;
  *"percentile_cont"*) printf '%s\n' "$FX_MED";;
  *"event='议题'"*)  printf '%s\n' "$FX_DUP";;
  *"pg_try_advisory_lock"*) printf 't\n';;
  *"pg_advisory_unlock"*) : ;;
  *"ops_headless"*) : ;;
esac
exit 0
EOF
  chmod +x "$d/bin/psql"
  cat > "$d/bin/deploy-stub.sh" <<'EOF'
#!/usr/bin/env bash
echo "${OPS_ESC_SH:-unset}" > "$DEP_ESC_LOG"
cat "$DEP_OUT"
EOF
  chmod +x "$d/bin/deploy-stub.sh"
  cat > "$d/bin/why-stub.sh" <<'EOF'
#!/usr/bin/env bash
echo called >> "$WHY_LOG"
EOF
  chmod +x "$d/bin/why-stub.sh"
  printf 'FIN_PG_DSN=fake-fin\nFIN_PG_RO_DSN=fake-ro\n' > "$d/env"
  : > "$d/sql.log"; : > "$d/why.log"; : > "$d/dep.esc.log"
}

run_ns() { # $1=场景目录——跑 ceo_northstar 全覆写面
  local d=$1
  ( env -i PATH="$d/bin:$PATH" HOME="$HOME" \
      DM_ENV_FILE="$d/env" NORTHSTAR_ENV_FILE="$d/env" \
      O_NORTHSTAR_DIR="$d/out" O_NORTHSTAR_DEPLOY_SH="$d/bin/deploy-stub.sh" \
      O_NORTHSTAR_WHY_SH="$d/bin/why-stub.sh" \
      SQL_LOG="$d/sql.log" WHY_LOG="$d/why.log" DEP_ESC_LOG="$d/dep.esc.log" DEP_OUT="$d/dep.out" \
      FX_PNL="${FX_PNL:-}" FX_ARMED="${FX_ARMED:-}" FX_OPEN="${FX_OPEN:-}" \
      FX_ESC="${FX_ESC:-}" FX_WATER="${FX_WATER:-}" FX_PREV="${FX_PREV:-}" \
      FX_MED="${FX_MED:-}" FX_DUP="${FX_DUP:-}" \
      bash "$META/ceo_northstar.sh" )
}

# ══ A. 背离成行+问链调用 ════════════════════════════════════════════
A="$TMP/a"; setup_ns "$A"
cat > "$A/dep.out" <<'EOF'
state=有挡
faces=image,oci
reason=批非空 commits=5 tickets=T-0001 img=x oci=y 批首件龄=90000s
reason=挡件未清: 待裁:T-9999(img)——批待清窗压
EOF
export FX_PNL='1.50|-2.50|4|3' FX_ARMED='25|30.0' FX_OPEN='0|30.5' \
  FX_ESC='2' FX_WATER='3|5.5' FX_MED='0.5000' FX_DUP='' \
  FX_PREV='北极星账 pnl24=0.10 pnl7d=-1.00 closed7d=2 wins7d=1 armed_n=3 open24h=1 open_age_h=2.0 esc_uncl=0 dep_age_h=1.0 water_n=0 water_age_h=0.0'
run_ns "$A" > "$A/stdout.log" 2>&1

# A① 议题行词形钉——seat=CEO/event=议题/ticket='' note 钦定前缀+基线位
# （六件全背离：pnl7d/open_age_h/armed_n/esc_uncl=why 型+dep_age_h/water_age_h=容量型）
[[ $(grep -c "INSERT INTO ops_event(seat,event,ticket,note) VALUES('CEO','议题'" "$A/sql.log") -eq 6 ]]
grep -Fq "北极星背离:pnl7d=-2.5000 vs 基线0.5——问链引T-0464" "$A/sql.log"
grep -Fq "北极星背离:open_age_h=30.5h vs 基线0.5——问链引T-0464" "$A/sql.log"
grep -Fq "北极星背离:armed_n=n=25 age=30.0h vs 基线0.5——问链引T-0464" "$A/sql.log"
grep -Fq "北极星背离:dep_age_h=25.0h commits=5 vs 基线0.5——问链引T-0464" "$A/sql.log"
grep -Fq "北极星背离:water_age_h=n=3 age=5.5h vs 基线0.5——问链引T-0464" "$A/sql.log"
grep -Fq "北极星背离:esc_uncl=2 vs 基线0.5——问链引T-0464" "$A/sql.log"

# A② 快照行 kv 面钉（每拍一行——7d 滚动基线原料）
grep -Fq "VALUES('CEO','北极星账'" "$A/sql.log"
grep -Fq "pnl7d=-2.5" "$A/sql.log"
grep -Fq "armed_n=25" "$A/sql.log"
grep -Fq "dep_age_h=25 dep_commits=5" "$A/sql.log"
grep -Fq "water_age_h=5.50" "$A/sql.log"

# A③ 账页七项指标值上页（钱面+运营面+基线+背离计数）
grep -Fq "净PnL 24h=1.5000 7d=-2.5000" "$A/stdout.log"
grep -Fq "平仓7d=4 笔·胜率 75%" "$A/stdout.log"
grep -Fq "armed滞留=25 件·最老 30.0h" "$A/stdout.log"
grep -Fq "断链未领升件=2" "$A/stdout.log"
grep -Fq "部署批=龄 25.0h·积压 5 commits" "$A/stdout.log"
grep -Fq "候水票=3（最老 5.5h）" "$A/stdout.log"
grep -Fq "6 件新议题" "$A/stdout.log"
[[ -f "$A/out/northstar.txt" ]]
grep -Fq "北极星账" "$A/out/northstar.txt"

# A④ dep 探测只读——OPS_ESC_SH=/bin/true 压升件副作用
grep -Fxq "/bin/true" "$A/dep.esc.log"

# A⑤ 为什么型背离→问链调用（pnl/open/armed/esc=why1；dep/water=容量不问链——
#     本场景混合仍应调一次——why_chain 内部自决案卷）
[[ $(wc -l < "$A/why.log") -eq 1 ]]

# ══ B. 24h 同指标去重 ═══════════════════════════════════════════════
B="$TMP/b"; setup_ns "$B"
cat > "$B/dep.out" <<'EOF'
state=批空
faces=
reason=diff 双基线镜像+OCI面俱空
EOF
export FX_PNL='0.5|-1.0|2|1' FX_ARMED='30|40.0' FX_OPEN='0|50.0' \
  FX_ESC='5' FX_WATER='1|9.0' FX_MED='0.5000' FX_DUP='1' \
  FX_PREV='北极星账 pnl7d=-0.50'
run_ns "$B" > "$B/stdout.log" 2>&1
# dup 桩全返 1=同指标 24h 内已有议题行——议题零落、快照照记、问链不调
! grep -q "VALUES('CEO','议题'" "$B/sql.log"
grep -Fq "VALUES('CEO','北极星账'" "$B/sql.log"
[[ ! -s "$B/why.log" ]]

# ══ C. 全绿拍 ═══════════════════════════════════════════════════════
C="$TMP/c"; setup_ns "$C"
cat > "$C/dep.out" <<'EOF'
state=批空
faces=
reason=diff 双基线镜像+OCI面俱空
EOF
export FX_PNL='1.20|3.40|6|5' FX_ARMED='3|2.0' FX_OPEN='2|5.0' \
  FX_ESC='0' FX_WATER='0|0.0' FX_MED='0.5000' FX_DUP='' \
  FX_PREV='北极星账 pnl7d=2.00'
run_ns "$C" > "$C/stdout.log" 2>&1
! grep -q "VALUES('CEO','议题'" "$C/sql.log"
grep -Fq "VALUES('CEO','北极星账'" "$C/sql.log"
[[ ! -s "$C/why.log" ]]
grep -Fq "净PnL 24h=1.2000 7d=3.4000" "$C/stdout.log"
grep -Fq "0 件新议题" "$C/stdout.log"
grep -Fq "部署批=批空" "$C/stdout.log"

# ══ D. ceo_queue 议题读面 ════════════════════════════════════════════
D="$TMP/d"; mkdir -p "$D/bin"
# ceo_queue 用 heredoc 喂 SQL——psql 桩收集 stdin 留证
cat > "$D/bin/psql" <<'EOF'
#!/usr/bin/env bash
cat >> "$QUEUE_SQL_LOG"
EOF
chmod +x "$D/bin/psql"
printf 'FIN_PG_DSN=fake\n' > "$D/env"
: > "$D/queue.sql.log"
DM_ENV_FILE="$D/env" FIN_DSN=fake PATH="$D/bin:$PATH" \
  QUEUE_SQL_LOG="$D/queue.sql.log" bash "$META/ceo_queue.sh" || true
# 议题 UNION 面钉——seat=CEO event=议题 出列词形/引证 regex 边界/排序面
SQL=$(cat "$D/queue.sql.log")
grep -Fq "SELECT '议题#'||e.id::text FROM ops_event e" <<< "$SQL"
grep -Fq "WHERE e.event='议题' AND e.seat='CEO'" <<< "$SQL"
grep -Fq "c.note ~ ('议题#' || e.id::text || '([^0-9]|\$)')" <<< "$SQL"
grep -Fq "ORDER BY CASE WHEN u.ticket LIKE '议题#%' THEN 0 ELSE 1 END" <<< "$SQL"
# 原升件/呈批判据面不动（票面既有语义零回归）
grep -Fq "e.event IN ('升件','呈批')" <<< "$SQL"
grep -Fq "t.event IN ('裁决','完成','驳回','消','处置','挂起','领单')" <<< "$SQL"

# ══ E. ceo_tick 三步序挂点（账→议题→队）══════════════════════════════
E="$TMP/e"; mkdir -p "$E/bin" "$E/ceo"
cat > "$E/bin/psql" <<'EOF'
#!/usr/bin/env bash
query="${*: -1}"
case "$query" in
  *"pg_try_advisory_lock"*) printf 't\n';;
  *"pg_advisory_unlock"*) : ;;
  *"ops_headless"*) : ;;
esac
exit 0
EOF
cat > "$E/bin/northstar-stub.sh" <<'EOF'
#!/usr/bin/env bash
echo northstar >> "$ORDER_LOG"
EOF
cat > "$E/bin/queue-stub.sh" <<'EOF'
#!/usr/bin/env bash
echo queue >> "$ORDER_LOG"
EOF
cat > "$E/bin/devin-stub" <<'EOF'
#!/usr/bin/env bash
echo DEVIN-CALLED >> "$ORDER_LOG"
EOF
chmod +x "$E/bin/"*
printf 'FIN_PG_DSN=fake\n' > "$E/env"
: > "$E/order.log"
env -i PATH="$E/bin:$PATH" HOME="$HOME" DM_ENV_FILE="$E/env" \
  O_CEO_WORKDIR="$E/ceo" O_CEO_NORTHSTAR_SH="$E/bin/northstar-stub.sh" \
  O_CEO_QUEUE_SH="$E/bin/queue-stub.sh" O_DECISION_DEVIN_BIN="$E/bin/devin-stub" \
  O_WORKER_USAGE_BIN=/bin/false ORDER_LOG="$E/order.log" \
  bash "$META/ceo_tick.sh" >/dev/null 2>&1 || true
# 空队拍：northstar 先扫→queue 闸空→心跳退——会话件零调
[[ "$(cat "$E/order.log")" == $'northstar\nqueue' ]]
! grep -q DEVIN-CALLED "$E/order.log"

echo "ceo northstar scenarios PASS"
