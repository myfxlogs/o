#!/usr/bin/env bash
# gateway_test.sh —— T-0469 o-gateway 桩测八组（票面钉死面）：
# ①事件（notify→脏位→域即评/非调度词不脏/水位续位零重脏）
# ②席权（depgate 调用+死席唤醒+冷却幂等）
# ③路由（dispatched 补拉+rc75 候水+人工径滤+handoff 调用+零动作不落账）
# ④收敛（应发窗→mirror_push+faces 发射+FIRE/VERIFY 账）
# ⑤冻结（升件未决→全面零动作+冻结账；面冻→该面跳）
# ⑥幂等（部署行在案→不发；升件路由级幂等不重级）
# ⑦降级（无 DSN/关监听→轮询模式域照跑不炸）
# ⑧崩溃（kill -9→重启→水位/域拍续位零重复动作）
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'kill ${GWDAEMON_PID:-0} 2>/dev/null; rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin" "$TMP/tasks" "$TMP/state" "$TMP/dom" "$TMP/routes"
touch "$TMP/tasks/T-9001-resume.md" "$TMP/tasks/T-9002-resume.md"

# ── psql 桩：-c 直派；无 -c 按分号聚合语句派（coproc 流式兼容）──
cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
logq() { printf '%s\n' "$1" >> "${PSQL_QLOG:-/dev/null}"; }
dispatch() {
  local q="$1"
  logq "$q"
  case "$q" in
    *pg_try_advisory_lock*) printf '%s\n' "${GW_LOCK_RESULT:-t}"; return 0;;
    *"COALESCE(max(id),0)"*) printf '%s\n' "${GW_MAX_ID:-0}"; return 0;;
    *"id||'|'||seat"*) [ -f "${GW_DRAIN_FILE:-/nonexistent}" ] && cat "$GW_DRAIN_FILE"; return 0;;
    *"INSERT INTO ops_headless"*) printf 'hb\n' >> "${HB_LOG:-/dev/null}"; return 0;;
    *"INSERT INTO ops_event"*) printf '%s\n' "$q" >> "${LEDGER_LOG:-/dev/null}"; return 0;;
    *"EXTRACT(EPOCH FROM now()-max(last_activity))"*"ticket='CTO'"*) printf '%s\n' "${GW_STALE_CTO:-0}"; return 0;;
    *"EXTRACT(EPOCH FROM now()-max(last_activity))"*"ticket='CEO'"*) printf '%s\n' "${GW_STALE_CEO:-0}"; return 0;;
    *"event='呈批'"*"24 hours"*) printf '%s\n' "${GW_CEO_PENDING:-}"; return 0;;
    *"event='派单'"*"ORDER BY ts DESC"*) printf '%s\n' "${GW_MODEL_ROW:-swe-2-flash-high}"; return 0;;
    *"DEPLOY-FIRE-FAIL"*)
      case "$q" in
        *":image"*) printf '%s\n' "${GW_FF_IMG:-${GW_FACE_FREEZE:-}}";;
        *":oci"*)   printf '%s\n' "${GW_FF_OCI:-${GW_FACE_FREEZE:-}}";;
        *)          printf '%s\n' "${GW_FACE_FREEZE:-}";;
      esac
      return 0;;
    *"substring(note from"*"deploy-oci"*"10 minutes"*) printf '%s\n' "${GW_VERIFY_OCI:-}"; return 0;;
    *"substring(note from"*"deploy-oci"*) printf '%s\n' "${GW_IDEM_OCI:-}"; return 0;;
    *"substring(note from"*"deploy"*"10 minutes"*) printf '%s\n' "${GW_VERIFY_IMG:-}"; return 0;;
    *"substring(note from"*"deploy"*) printf '%s\n' "${GW_IDEM_IMG:-}"; return 0;;
    *"extract(epoch from max(ts))"*"升件路由"*)
      eid=$(printf '%s' "$q" | sed -n "s/.*升件#\([0-9][0-9]*\).*/\1/p" | head -1)
      { [ -n "$eid" ] && [ -f "${GW_ROUTES_DIR:-/nonexistent}/$eid" ]; } && cat "${GW_ROUTES_DIR}/$eid" || printf '0|0\n'
      return 0;;
    *"e.event='升件'"*"升件失领%"*"ORDER BY"*|*"升件失领%"*"NOT EXISTS"*) printf '%s\n' "${GW_POPEN:-1}"; return 0;;
    *"e.event='升件'"*"ORDER BY e.id"*) [ -f "${GW_ESC_ROWS:-/nonexistent}" ] && cat "$GW_ESC_ROWS"; return 0;;
    *"86400 seconds"*) printf '%s\n' "${GW_FREEZE_N:-0}"; return 0;;
    *"SELECT 1;"*)
      printf '1\n'
      # 通知旗模拟：keepalive 顺带吐 NOTIFY 行（piggyback 语义，与真 psql 一致）
      if [ -n "${GW_NOTIFY_FILE:-}" ] && [ -f "$GW_NOTIFY_FILE" ]; then
        rm -f "$GW_NOTIFY_FILE"
        printf 'Asynchronous notification "ev_pipeline" with payload "" received from server process with PID 1.\n'
      fi
      return 0;;
    *) return 0;;
  esac
}
has_c=0; prev=""
for a in "$@"; do [ "$prev" = "-c" ] && { dispatch "$a"; exit 0; }; prev="$a"; done
buf=""
while IFS= read -r line; do
  buf="${buf}${line}"$'\n'
  case "$line" in *';'*) dispatch "$buf"; buf="";; esac
done
[ -n "$buf" ] && dispatch "$buf"
exit 0
EOF
chmod +x "$TMP/bin/psql"

# ── 通用桩件 ──
cat > "$TMP/bin/systemctl" <<'EOF'
#!/usr/bin/env bash
case "$*" in
  *"is-active"*) printf '%s\n' "${GW_UNIT_STATE:-inactive}"; exit 3;;
  *"start --no-block"*|*"start"*) printf '%s\n' "$*" >> "${SYSCTL_LOG:-/dev/null}"; exit "${GW_START_RC:-0}";;
  *"list-units"*) : ;; # tolerate: 空集合法径
  *) : ;;
esac
exit 0
EOF
cat > "$TMP/bin/dm-queue" <<'EOF'
#!/usr/bin/env bash
case "${1:-}" in
  dispatched) printf '%b\n' "${GW_DISPATCHED_TIDS:-}" ;;
  *) printf '%b\n' "${GW_QUEUE_TIDS:-}" ;;
esac
EOF
cat > "$TMP/bin/depgate.sh" <<'EOF'
#!/usr/bin/env bash
echo depgate >> "${COMP_LOG:-/dev/null}"; exit "${GW_DEPGATE_RC:-0}"
EOF
cat > "$TMP/bin/handoff.sh" <<'EOF'
#!/usr/bin/env bash
echo handoff >> "${COMP_LOG:-/dev/null}"; exit "${GW_HANDOFF_RC:-0}"
EOF
cat > "$TMP/bin/order.sh" <<'EOF'
#!/usr/bin/env bash
cat
EOF
cat > "$TMP/bin/manual-lane" <<'EOF'
#!/usr/bin/env bash
exit "${GW_MANUAL_RC:-1}"
EOF
cat > "$TMP/bin/dispatch-worker" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "${DISPATCH_LOG:-/dev/null}"
exit "${GW_WORKER_RC:-0}"
EOF
cat > "$TMP/bin/water" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$1" >> "${WATER_LOG:-/dev/null}"
EOF
cat > "$TMP/bin/win.sh" <<'EOF'
#!/usr/bin/env bash
case "${GW_WINDOW_STATE:-批空}" in
  应发) printf 'state=应发\nfaces=%s\nreason=stub\n' "${GW_WINDOW_FACES:-image}" ;;
  *)     printf 'state=批空\nfaces=\nreason=stub\n'; exit 10 ;;
esac
EOF
cat > "$TMP/bin/mirror.sh" <<'EOF'
#!/usr/bin/env bash
echo mirror >> "${MIRROR_LOG:-/dev/null}"; exit "${GW_MIRROR_RC:-0}"
EOF
cat > "$TMP/bin/deploy_image_stub.sh" <<'EOF'
#!/usr/bin/env bash
echo image >> "${DEPLOY_LOG:-/dev/null}"; exit "${GW_DEPLOY_IMG_RC:-0}"
EOF
cat > "$TMP/bin/deploy_oci_stub.sh" <<'EOF'
#!/usr/bin/env bash
echo oci >> "${DEPLOY_LOG:-/dev/null}"; exit "${GW_DEPLOY_OCI_RC:-0}"
EOF
cat > "$TMP/bin/feishu" <<'EOF'
#!/usr/bin/env bash
echo "$*" >> "${FEISHU_LOG:-/dev/null}"
EOF
for d in seatauth route converge escalate; do
  cat > "$TMP/dom/$d.sh" <<EOF
#!/usr/bin/env bash
echo $d >> "\${DOM_LOG:-/dev/null}"
EOF
done
chmod +x "$TMP/bin/"* "$TMP/dom/"*.sh

# ── 仓桩：HEAD==origin/master（收敛态）──
REPO="$TMP/repo"; mkdir -p "$REPO"; cd "$REPO"
git init -q && git config user.email t@t && git config user.name t
echo x > f && git add f && git commit -qm init
git update-ref refs/remotes/origin/master HEAD
SHA12=$(git rev-parse --short=12 HEAD)

printf 'FIN_PG_DSN=fake\n' > "$TMP/env"
export PATH="$TMP/bin:$PATH"
export DM_ENV_FILE="$TMP/env" GW_ENV_FILE="$TMP/env"
export O_GATEWAY_STATE_DIR="$TMP/state" O_GATEWAY_REPO="$REPO"
export O_GATEWAY_DOMAIN_DIR="$TMP/dom"
export O_GW_DEPGATE_SH="$TMP/bin/depgate.sh" O_GW_DM_QUEUE_SH="$TMP/bin/dm-queue"
export O_GW_HANDOFF_SH="$TMP/bin/handoff.sh" O_DISPATCH_ORDER_SH="$TMP/bin/order.sh"
export O_MANUAL_LANE_SH="$TMP/bin/manual-lane"
export O_WORKER_DISPATCH_SCRIPT="$TMP/bin/dispatch-worker"
export O_DISPATCH_WATER_SH="$TMP/bin/water"
export O_WORKER_TASK_DIR="$TMP/tasks"
export O_GW_MIRROR_PUSH_SH="$TMP/bin/mirror.sh"
export O_GW_DEPLOY_WINDOW_SH="$TMP/bin/win.sh"
export O_DEPLOY_IMAGE_SH="$TMP/bin/deploy_image_stub.sh"
export O_DEPLOY_OCI_SH="$TMP/bin/deploy_oci_stub.sh"
export O_GW_FEISHU_SH="$TMP/bin/feishu"
export GW_ROUTES_DIR="$TMP/routes"
export PSQL_QLOG="$TMP/q.log"; : > "$PSQL_QLOG"
export HB_LOG="$TMP/hb.log"; : > "$HB_LOG"
export LEDGER_LOG="$TMP/ledger.log"; : > "$LEDGER_LOG"
export COMP_LOG="$TMP/comp.log"; : > "$COMP_LOG"
export DISPATCH_LOG="$TMP/dispatch.log"; : > "$DISPATCH_LOG"
export WATER_LOG="$TMP/water.log"; : > "$WATER_LOG"
export MIRROR_LOG="$TMP/mirror.log"; : > "$MIRROR_LOG"
export DEPLOY_LOG="$TMP/deploy.log"; : > "$DEPLOY_LOG"
export SYSCTL_LOG="$TMP/sysctl.log"; : > "$SYSCTL_LOG"
export FEISHU_LOG="$TMP/feishu.log"; : > "$FEISHU_LOG"
export DOM_LOG="$TMP/dom.log"; : > "$DOM_LOG"
export GW_DRAIN_FILE="$TMP/drain.txt"; : > "$GW_DRAIN_FILE"
export GW_ESC_ROWS="$TMP/esc_rows.txt"

GWDAEMON="$ROOT/scripts/meta/gatewayd.sh"
SEAT="$ROOT/scripts/meta/gateway/seatauth.sh"
ROUTE="$ROOT/scripts/meta/gateway/route.sh"
CONV="$ROOT/scripts/meta/gateway/converge.sh"
ESCAL="$ROOT/scripts/meta/gateway/escalate.sh"
reset_logs() { : > "$LEDGER_LOG"; : > "$COMP_LOG"; : > "$DISPATCH_LOG"; : > "$WATER_LOG"; : > "$MIRROR_LOG"; : > "$DEPLOY_LOG"; : > "$SYSCTL_LOG"; : > "$FEISHU_LOG"; : > "$DOM_LOG"; : > "$GW_DRAIN_FILE"; }

# ══ ① 事件：notify→脏位→域即评；非调度词不脏；水位续位零重脏 ══
# 预置四域 last=now（仅脏位驱动）+高 interval——只认事件不认拍
now_s=$(date +%s)
for d in seatauth route converge escalate; do printf '%s' "$now_s" > "$TMP/state/dom.$d.last"; done
rm -f "$TMP/state"/dom.*.dirty
(
  O_GATEWAY_LISTEN=1 O_GATEWAY_SELECT_SEC=1 O_GATEWAY_KEEPALIVE_SEC=1 \
  O_GATEWAY_DB_POLL_SEC=3600 O_GATEWAY_HB_SEC=3600 O_GATEWAY_HB_EVENT_SEC=3600 \
  O_GATEWAY_LOCK_RETRY_SEC=1 O_GATEWAY_STAMP_GAP_SEC=1 \
  O_GW_INT_SEAT=3600 O_GW_INT_ROUTE=3600 O_GW_INT_CONVERGE=3600 O_GW_INT_ESC=3600 \
  O_GW_GAP_SEAT=1 O_GW_GAP_ROUTE=1 O_GW_GAP_CONVERGE=1 O_GW_GAP_ESC=1 \
  GW_LOCK_RESULT=t GW_MAX_ID=600 GW_NOTIFY_FILE="$TMP/notify.flag" \
  bash "$GWDAEMON" >/dev/null 2>"$TMP/gw1.err" & echo $! > "$TMP/gw.pid"
)
sleep 3
# 无通知空拍：域零调用（事件门控实证——非等拍空转）
[ ! -s "$DOM_LOG" ] || { echo "FAIL ①空拍域被唤"; cat "$DOM_LOG"; exit 1; }
# 注事件行+通知旗：已批→席权/路由/收敛脏；升件→四域全脏；gateway 行滤
printf '500|CTO|已批|T-1\n501|CEO|升件|\n502|gateway|收敛|GATEWAY\n' > "$GW_DRAIN_FILE"
: > "$TMP/notify.flag"
sleep 5
grep -q '^seatauth$' "$DOM_LOG" && grep -q '^route$' "$DOM_LOG" \
  && grep -q '^converge$' "$DOM_LOG" && grep -q '^escalate$' "$DOM_LOG" \
  || { echo "FAIL ①事件未驱域"; cat "$DOM_LOG"; exit 1; }
echo "ok ①notify→脏位→四域即评（gateway 自行滤）"
# 水位续位：清 drain+再通知→零新脏零重跑
n1=$(wc -l < "$DOM_LOG")
: > "$GW_DRAIN_FILE"; : > "$TMP/notify.flag"
sleep 4
n2=$(wc -l < "$DOM_LOG")
[ "$n1" = "$n2" ] || { echo "FAIL ①水位续位重脏（$n1→$n2）"; exit 1; }
echo "ok ①水位续位——空批零重脏零重跑"
# 戳事件：check-green 新件→收敛脏 stamp 档即评（他域不动）
mkdir -p "$REPO/.git"; : > "$REPO/.git/check-green-deadbeefcafe"
n3=$(wc -l < "$DOM_LOG")
sleep 4
n4=$(wc -l < "$DOM_LOG")
[ $((n4-n3)) -eq 1 ] && tail -1 "$DOM_LOG" | grep -q '^converge$' \
  || { echo "FAIL ①戳事件未拉收敛（增量=$((n4-n3))）"; cat "$DOM_LOG"; exit 1; }
rm -f "$REPO/.git/check-green-deadbeefcafe"
echo "ok ①绿戳落件→收敛域 stamp 档即评（他域不动）"
kill "$(cat "$TMP/gw.pid")" 2>/dev/null; sleep 1

# ══ ② 席权：depgate 调用+死席唤醒+冷却 ══
reset_logs
GW_DEPGATE_RC=0 GW_STALE_CTO=0 GW_STALE_CEO=0 GW_QUEUE_TIDS='T-1' bash "$SEAT" 2>/dev/null
grep -q '^depgate$' "$COMP_LOG" || { echo "FAIL ②depgate 未调"; exit 1; }
grep -q 'seatwake' "$LEDGER_LOG" && { echo "FAIL ②新鲜道误唤"; exit 1; } || true # tolerate: 断言反向——grep 未中=期望径（||true 仅续行）
# 死席：CTO 道龄极大+队有件→start 单元+席权账；二跑冷却不重复
reset_logs
GW_DEPGATE_RC=0 GW_STALE_CTO=999999 GW_STALE_CEO=0 GW_QUEUE_TIDS='T-1' \
  GW_CEO_PENDING='' bash "$SEAT" 2>/dev/null
grep -q 'start --no-block o-cto-seat.service' "$SYSCTL_LOG" \
  || { echo "FAIL ②死席未唤"; cat "$SYSCTL_LOG"; exit 1; }
grep -q "seatwake CTO" "$LEDGER_LOG" || { echo "FAIL ②唤醒账未落"; cat "$LEDGER_LOG"; exit 1; }
reset_logs
GW_DEPGATE_RC=0 GW_STALE_CTO=999999 GW_STALE_CEO=0 GW_QUEUE_TIDS='T-1' \
  GW_CEO_PENDING='' bash "$SEAT" 2>/dev/null
[ ! -s "$SYSCTL_LOG" ] || { echo "FAIL ②冷却期重复唤"; cat "$SYSCTL_LOG"; exit 1; }
# depgate 败→升件自报（ops_esc→escalate 链→'升件'行 seat=gateway）
reset_logs
GW_DEPGATE_RC=3 GW_STALE_CTO=0 GW_STALE_CEO=0 bash "$SEAT" 2>/dev/null
grep -q "GW-SEATAUTH-DEPGATE" "$LEDGER_LOG" \
  || { echo "FAIL ②depgate 败未升件"; cat "$LEDGER_LOG"; exit 1; }
echo "ok ②席权：depgate 调用+死席唤醒+冷却幂等+败升件"

# ══ ③ 路由：dispatched 补拉+rc75 候水+handoff ══
reset_logs
GW_DISPATCHED_TIDS='T-9001' GW_QUEUE_TIDS='' GW_UNIT_STATE=inactive \
  GW_WORKER_RC=0 GW_MODEL_ROW='swe-2-medium' GW_MANUAL_RC=1 \
  GW_HANDOFF_RC=0 bash "$ROUTE" 2>/dev/null
grep -q 'T-9001 swe-2-medium' "$DISPATCH_LOG" \
  || { echo "FAIL ③补拉未派"; cat "$DISPATCH_LOG"; exit 1; }
grep -q "'拉起'" "$LEDGER_LOG" || { echo "FAIL ③拉起行未落"; cat "$LEDGER_LOG"; exit 1; }
grep -q '^handoff$' "$COMP_LOG" || { echo "FAIL ③handoff 未调"; exit 1; }
# 活单元在飞→跳过不补拉
reset_logs
GW_DISPATCHED_TIDS='T-9001' GW_UNIT_STATE=active GW_WORKER_RC=0 \
  GW_HANDOFF_RC=0 bash "$ROUTE" 2>/dev/null
[ ! -s "$DISPATCH_LOG" ] || { echo "FAIL ③活单元仍补拉"; cat "$DISPATCH_LOG"; exit 1; }
# rc75→候水标记
reset_logs
GW_DISPATCHED_TIDS='T-9001' GW_UNIT_STATE=inactive GW_WORKER_RC=75 \
  GW_HANDOFF_RC=0 bash "$ROUTE" 2>/dev/null
grep -q '^T-9001$' "$WATER_LOG" || { echo "FAIL ③rc75 未候水"; cat "$WATER_LOG"; exit 1; }
# 零候选+handoff rc0→零动作零汇总账（账本去噪）
reset_logs
GW_DISPATCHED_TIDS='' GW_HANDOFF_RC=0 bash "$ROUTE" 2>/dev/null
! grep -q "'路由'" "$LEDGER_LOG" || { echo "FAIL ③零动作落了汇总账"; cat "$LEDGER_LOG"; exit 1; }
echo "ok ③路由：补拉/活单元跳/rc75 候水/零动作不落账"

# ══ ④ 收敛：应发窗→push+faces+VERIFY ══
reset_logs
GW_FREEZE_N=0 GW_WINDOW_STATE=应发 GW_WINDOW_FACES='image,oci' \
  GW_IDEM_IMG='' GW_IDEM_OCI='' GW_FACE_FREEZE='' \
  GW_VERIFY_IMG=1 GW_VERIFY_OCI=1 GW_MIRROR_RC=0 \
  bash "$CONV" 2>/dev/null
grep -q '^mirror$' "$MIRROR_LOG" || { echo "FAIL ④mirror_push 未调"; exit 1; }
grep -qx 'image' "$DEPLOY_LOG" && grep -qx 'oci' "$DEPLOY_LOG" \
  || { echo "FAIL ④faces 未全发"; cat "$DEPLOY_LOG"; exit 1; }
grep -q ",'收敛'," "$LEDGER_LOG" && grep -q 'PLAN' "$LEDGER_LOG" \
  && grep -q 'FIRE' "$LEDGER_LOG" && grep -q 'VERIFY' "$LEDGER_LOG" \
  || { echo "FAIL ④收敛账链缺"; cat "$LEDGER_LOG"; exit 1; }
# ⑤ 冻结：升件未决→全面零动作+冻结账
reset_logs
GW_FREEZE_N=2 GW_WINDOW_STATE=应发 GW_WINDOW_FACES='image,oci' \
  GW_IDEM_IMG='' GW_IDEM_OCI='' bash "$CONV" 2>/dev/null
[ ! -s "$MIRROR_LOG" ] && [ ! -s "$DEPLOY_LOG" ] \
  || { echo "FAIL ⑤冻结期仍动作"; cat "$MIRROR_LOG" "$DEPLOY_LOG"; exit 1; }
grep -q 'freeze' "$LEDGER_LOG" || { echo "FAIL ⑤冻结账未落"; cat "$LEDGER_LOG"; exit 1; }
# 面冻：image 面 DEPLOY-FIRE-FAIL 在案→image 跳 oci 照发
reset_logs
GW_FREEZE_N=0 GW_WINDOW_STATE=应发 GW_WINDOW_FACES='image,oci' \
  GW_IDEM_IMG='' GW_IDEM_OCI='' GW_FF_IMG=1 GW_FF_OCI='' \
  GW_VERIFY_OCI=1 bash "$CONV" 2>/dev/null
grep -qx 'oci' "$DEPLOY_LOG" && ! grep -qx 'image' "$DEPLOY_LOG" \
  || { echo "FAIL ⑤面冻未按面隔离"; cat "$DEPLOY_LOG"; exit 1; }
echo "ok ④⑤收敛：应发全链+冻结全面不发+面冻按面隔离"

# ══ ⑥ 幂等：部署行在案→不发；升件路由级幂等 ══
reset_logs
GW_FREEZE_N=0 GW_WINDOW_STATE=应发 GW_WINDOW_FACES=image \
  GW_IDEM_IMG=1 GW_FACE_FREEZE='' bash "$CONV" 2>/dev/null
! grep -qx 'image' "$DEPLOY_LOG" || { echo "FAIL ⑥幂等命中仍发"; cat "$DEPLOY_LOG"; exit 1; }
# 升件路由：未认领升件→L1 归口唤起+升件路由账；档预置 L1→L2 冷却过→L2
reset_logs
printf '101|T-1||some escalation note|1700000000\n' > "$GW_ESC_ROWS"
GW_UNIT_STATE=inactive bash "$ESCAL" 2>/dev/null
grep -q "升件#101 L1" "$LEDGER_LOG" \
  || { echo "FAIL ⑥升件未路由 L1"; cat "$LEDGER_LOG"; exit 1; }
grep -q 'start --no-block o-cto-seat.service' "$SYSCTL_LOG" \
  || { echo "FAIL ⑥L1 未唤起"; cat "$SYSCTL_LOG"; exit 1; }
# 同级冷却内重跑→零重复（rstat 档=1|now）
printf '1|%s\n' "$(date +%s)" > "$TMP/routes/101"
reset_logs
bash "$ESCAL" 2>/dev/null
! grep -q "升件#101" "$LEDGER_LOG" || { echo "FAIL ⑥冷却期重复路由"; cat "$LEDGER_LOG"; exit 1; }
# 冷却过→L2
printf '1|%s\n' "$(( $(date +%s) - 8000 ))" > "$TMP/routes/101"
reset_logs
bash "$ESCAL" 2>/dev/null
grep -q "升件#101 L2" "$LEDGER_LOG" || { echo "FAIL ⑥未升 L2"; cat "$LEDGER_LOG"; exit 1; }
# L4 业主级→COS上行+feishu
printf '3|%s\n' "$(( $(date +%s) - 8000 ))" > "$TMP/routes/101"
reset_logs
bash "$ESCAL" 2>/dev/null
grep -q ",'COS上行'," "$LEDGER_LOG" && grep -q '升件#101' "$FEISHU_LOG" \
  || { echo "FAIL ⑥业主级未上行"; cat "$LEDGER_LOG" "$FEISHU_LOG"; exit 1; }
echo "ok ⑥幂等：部署行查重+升件路由级幂等/阶梯"

# ══ ⑥b 分类归领（T-0517）：机械可证类自销+裁断类让梯 ══
# 桩行=5段形 id|ticket|scope|note|ts（ladder 查询同桩兼容）
mkdir -p "$REPO/docs/tasks/done"; : > "$REPO/docs/tasks/done/T-0002-x.md"
cat > "$GW_ESC_ROWS" <<EOF
201|T-0002||层扫定性done-翻漏:T-0002——票件已入 done/ 而相位=verdicted|1700000000
202|SELFTEST-DBP3a-CLS-1||层扫定性无籍行:SELFTEST-DBP3a-CLS-1——相位行在案而两处皆无|1700000000
203|T-0242||层扫定性幽灵票件:T-0242——相位verdicted 在案而两处皆无|1700000000
204|dispatch_aged_water||层扫定性升件失领:dispatch_aged_water——升件行3件超4h无认领|1700000000
205|T-7777||CHECKALL-RACE-STAMP-EAT:check_all并发同SHA互噬|1700000000
EOF
reset_logs
# popen=0（失领母件全销）→204 自销；205 非层扫签名→让梯 L1 唤起
GW_POPEN=0 GW_UNIT_STATE=inactive bash "$ESCAL" 2>/dev/null
grep -q ",'完成','T-0002'" "$LEDGER_LOG" || { echo "FAIL ⑥b done-翻漏未自销翻"; cat "$LEDGER_LOG"; exit 1; }
grep -q ",'处置','','升件#202" "$LEDGER_LOG" || { echo "FAIL ⑥b 无籍测试域未销"; cat "$LEDGER_LOG"; exit 1; }
grep -q ",'作废','T-0242'" "$LEDGER_LOG" && grep -q ",'处置','','升件#203" "$LEDGER_LOG" \
  || { echo "FAIL ⑥b 幽灵作废双行缺"; cat "$LEDGER_LOG"; exit 1; }
grep -q ",'处置','','升件#204" "$LEDGER_LOG" || { echo "FAIL ⑥b 失领母销未自销"; cat "$LEDGER_LOG"; exit 1; }
grep -q "分类归领自销=4" "$LEDGER_LOG" || { echo "FAIL ⑥b 归领汇总账缺"; cat "$LEDGER_LOG"; exit 1; }
grep -q "升件#205 L1" "$LEDGER_LOG" || { echo "FAIL ⑥b 裁断类未让梯"; cat "$LEDGER_LOG"; exit 1; }
# popen>0（失领母件未销）→204 让梯不进自销
reset_logs
GW_POPEN=3 GW_UNIT_STATE=inactive bash "$ESCAL" 2>/dev/null
! grep -q "升件#204 销" "$LEDGER_LOG" || { echo "FAIL ⑥b 母件未销仍自销"; cat "$LEDGER_LOG"; exit 1; }
rm -rf "$REPO/docs"
echo "ok ⑥b分类归领：done翻漏完成翻/测试域处置销/幽灵作废双行/失领母销自销/裁断让梯+gw_ledger引号修复实证（汇总账落）"

# ══ ⑦ 降级：无 DSN→轮询模式域脚本自含退出不炸；网关死=席位自扫不受损 ══
reset_logs
GW_ENV_FILE=/nonexistent DM_ENV_FILE=/nonexistent bash "$SEAT" 2>/dev/null; [ $? -eq 0 ] || { echo "FAIL ⑦seatauth 无DSN炸"; exit 1; }
GW_ENV_FILE=/nonexistent DM_ENV_FILE=/nonexistent bash "$ROUTE" 2>/dev/null; [ $? -eq 0 ] || { echo "FAIL ⑦route 无DSN炸"; exit 1; }
GW_ENV_FILE=/nonexistent DM_ENV_FILE=/nonexistent bash "$CONV" 2>/dev/null; [ $? -eq 0 ] || { echo "FAIL ⑦converge 无DSN炸"; exit 1; }
GW_ENV_FILE=/nonexistent DM_ENV_FILE=/nonexistent bash "$ESCAL" 2>/dev/null; [ $? -eq 0 ] || { echo "FAIL ⑦escalate 无DSN炸"; exit 1; }
# 网关 daemon 无 DSN+关监听→flock 兜底单例，域拍照跑不炸
rm -rf "$TMP/state2"; mkdir -p "$TMP/state2"
(
  GW_ENV_FILE=/nonexistent DM_ENV_FILE=/nonexistent \
  O_GATEWAY_STATE_DIR="$TMP/state2" O_GATEWAY_LISTEN=0 \
  O_GATEWAY_SELECT_SEC=1 O_GATEWAY_DB_POLL_SEC=3600 \
  O_GATEWAY_HB_SEC=3600 O_GATEWAY_HB_EVENT_SEC=3600 \
  O_GW_INT_SEAT=1 O_GW_INT_ROUTE=1 O_GW_INT_CONVERGE=1 O_GW_INT_ESC=1 \
  O_GATEWAY_DOMAIN_DIR="$TMP/dom" \
  bash "$GWDAEMON" >/dev/null 2>"$TMP/gw7.err" & echo $! > "$TMP/gw7.pid"
)
sleep 4
kill -0 "$(cat "$TMP/gw7.pid")" 2>/dev/null || { echo "FAIL ⑦降级 daemon 死"; cat "$TMP/gw7.err"; exit 1; }
kill "$(cat "$TMP/gw7.pid")" 2>/dev/null; sleep 1
echo "ok ⑦降级：无 DSN 域件自含退出+daemon 轮询兜底存活"

# ══ ⑧ 崩溃：kill -9→重启→水位/域拍续位零重复 ══
rm -rf "$TMP/state3"; mkdir -p "$TMP/state3"
now_s=$(date +%s)
for d in seatauth route converge escalate; do printf '%s' "$now_s" > "$TMP/state3/dom.$d.last"; done
export GW_LOCK_RESULT=t GW_MAX_ID=700
(
  O_GATEWAY_LISTEN=1 O_GATEWAY_SELECT_SEC=1 O_GATEWAY_KEEPALIVE_SEC=1 \
  O_GATEWAY_DB_POLL_SEC=3600 O_GATEWAY_HB_SEC=3600 O_GATEWAY_HB_EVENT_SEC=3600 \
  O_GATEWAY_LOCK_RETRY_SEC=1 O_GATEWAY_STAMP_GAP_SEC=1 \
  O_GW_INT_SEAT=3600 O_GW_INT_ROUTE=3600 O_GW_INT_CONVERGE=3600 O_GW_INT_ESC=3600 \
  O_GW_GAP_SEAT=1 O_GW_GAP_ROUTE=1 O_GW_GAP_CONVERGE=1 O_GW_GAP_ESC=1 \
  GW_NOTIFY_FILE="$TMP/notify2.flag" \
  O_GATEWAY_STATE_DIR="$TMP/state3" \
  bash "$GWDAEMON" >/dev/null 2>"$TMP/gw8.err" & echo $! > "$TMP/gw8.pid"
)
sleep 3
printf '700|CTO|已批|T-9\n' > "$GW_DRAIN_FILE"; : > "$TMP/notify2.flag"
sleep 4
c1=$(grep -c '^seatauth$' "$DOM_LOG" || true) # tolerate: grep -c 零匹配 rc=1——计数 0 合法径
kill -9 "$(cat "$TMP/gw8.pid")" 2>/dev/null; sleep 1
# 重启同 state：水位已记 700→drain 空→零新脏零重跑
: > "$GW_DRAIN_FILE"
(
  O_GATEWAY_LISTEN=1 O_GATEWAY_SELECT_SEC=1 O_GATEWAY_KEEPALIVE_SEC=1 \
  O_GATEWAY_DB_POLL_SEC=3600 O_GATEWAY_HB_SEC=3600 O_GATEWAY_HB_EVENT_SEC=3600 \
  O_GATEWAY_LOCK_RETRY_SEC=1 O_GATEWAY_STAMP_GAP_SEC=1 \
  O_GW_INT_SEAT=3600 O_GW_INT_ROUTE=3600 O_GW_INT_CONVERGE=3600 O_GW_INT_ESC=3600 \
  O_GATEWAY_DOMAIN_DIR="$TMP/dom" \
  GW_NOTIFY_FILE="$TMP/notify2.flag" \
  O_GATEWAY_STATE_DIR="$TMP/state3" \
  bash "$GWDAEMON" >/dev/null 2>"$TMP/gw8b.err" & echo $! > "$TMP/gw8.pid"
)
sleep 4
c2=$(grep -c '^seatauth$' "$DOM_LOG" || true) # tolerate: grep -c 零匹配 rc=1——计数 0 合法径
[ "$c1" = "$c2" ] || { echo "FAIL ⑧重启重复执行（$c1→$c2）"; exit 1; }
kill "$(cat "$TMP/gw8.pid")" 2>/dev/null; sleep 1
echo "ok ⑧崩溃：kill -9 重启水位/域拍续位零重复动作"

echo "gateway scenarios PASS"
