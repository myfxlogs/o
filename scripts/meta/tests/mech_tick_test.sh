#!/usr/bin/env bash
# mech_tick_test.sh —— GW-001 段A 机械轨桩测：psql/systemctl/go 全桩+
# 子件脚本桩（O_*_SH 覆写面），deploy_window_due 三态壳喂法（票面
# 「O_SHIFTEND_WINDOW_SH 同式」——O_DEPLOY_WINDOW_SH 覆写）。
# 面：①承接尾段六子件+ticketdraft+心跳全叫 ②应发→image/oci 各发其件
# ③部署行幂等跳 ④rc≠0→DEPLOY-FIRE-FAIL 升件+冻结期不重试 ⑤HEAD≠
# origin/master 未收敛跳过 ⑥批空/有挡零动作 ⑦锁拒即退零副作用
# ⑧派单补拉窗照旧（dispatch_worker 共用闸承接实证）。
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin" "$TMP/tasks"
touch "$TMP/tasks/T-9001-resume.md" "$TMP/tasks/T-9002-resume.md"

# ── psql 桩：锁会话/-qAt 形回 MECH_LOCK_RESULT；查询按词形分流 ──
cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
query="${*: -1}"
case "$query" in
  "-qAt") echo "${MECH_LOCK_RESULT:-t}" ;; # advisory-lock holder 桩
  *"INSERT INTO ops_headless"*) printf '%s\n' "$query" >> "$HB_LOG" ;;
  *"AND event='派单'"*"LIMIT 1"*) printf '%s\n' "${MODEL_ROW:-swe-2-flash-high}" ;; # 模型档恢复票级查询（GW-001 段B1 收编后形态）
  *"DEPLOY-FIRE-FAIL"*) printf '%s\n' "${FREEZE_RESULT:-}" ;; # 冻结态查询
  *"substring(note from"*) printf '%s\n' "${IDEM_RESULT:-}" ;; # 部署行幂等查询
  *"INSERT INTO ops_event"*) printf '%s\n' "$query" >> "$PULL_LOG" ;; # 拉起行
  *) : ;; # tolerate: 未识查询空答
esac
EOF
chmod +x "$TMP/bin/psql"

# ── 子件桩：六机械子件+autoapprove 各落名 ──
for n in depgate suspend handoff ledger verdict autoapprove; do
  cat > "$TMP/bin/$n.sh" <<EOF
#!/usr/bin/env bash
echo $n >> "\$COMP_LOG"
exit 0
EOF
done
# ticketdraft 径=go run——go 桩落名
cat > "$TMP/bin/go" <<'EOF'
#!/usr/bin/env bash
echo "ticketdraft $*" >> "$GO_LOG"
exit "${GO_RC:-0}"
EOF
# deploy_window_due 三态壳（票面同式喂法）
cat > "$TMP/bin/win.sh" <<'EOF'
#!/usr/bin/env bash
case "${WINDOW_STATE:-批空}" in
  应发) printf 'state=应发\nfaces=%s\nreason=stub\n' "${WINDOW_FACES:-image}" ;;
  有挡) printf 'state=有挡\nfaces=image,oci\nreason=stub-drag\n'; exit 11 ;;
  查询败) printf 'state=查询败\nreason=stub-qfail\n'; exit 1 ;;
  *)     printf 'state=批空\nfaces=\nreason=stub-empty\n'; exit 10 ;;
esac
EOF
# dm_queue 桩（GW-001 段B1：补拉候选集收编 dm_queue dispatched 单源）
cat > "$TMP/bin/dm-queue" <<'EOF'
#!/usr/bin/env bash
[ "${1:-}" = dispatched ] && printf '%b\n' "${DISPATCH_TIDS:-}"
EOF
chmod +x "$TMP/bin/dm-queue"
# 部署件桩：落名+rc 可控
cat > "$TMP/bin/deploy_image_stub.sh" <<'EOF'
#!/usr/bin/env bash
echo "image" >> "$DEPLOY_LOG"
exit "${DEPLOY_IMAGE_RC:-0}"
EOF
cat > "$TMP/bin/deploy_oci_stub.sh" <<'EOF'
#!/usr/bin/env bash
echo "oci" >> "$DEPLOY_LOG"
exit "${DEPLOY_OCI_RC:-0}"
EOF
# ops_esc 桩：$5=prefix $6=detail 落档
cat > "$TMP/bin/opsesc.sh" <<'EOF'
#!/usr/bin/env bash
printf '%s|%s\n' "$5" "$6" >> "$ESC_LOG"
EOF
# 补拉件桩（DISPATCH_RC_SEQ=前 N 次调用 rc75 序列——GW-001 段C1 续试喂法）
cat > "$TMP/bin/dispatch-worker" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$DISPATCH_LOG"
if [ -f "$DISPATCH_RC_SEQ" ] && [ "$(cat "$DISPATCH_RC_SEQ" 2>/dev/null)" -gt 0 ] 2>/dev/null; then
  echo $(( $(cat "$DISPATCH_RC_SEQ") - 1 )) > "$DISPATCH_RC_SEQ"
  exit 75
fi
exit "${DISPATCH_RC:-0}"
EOF
cat > "$TMP/bin/water" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$1" >> "$WATER_LOG"
EOF
# 漂移探针/自收敛桩（GW-001 段F：落名+rc 可控）
cat > "$TMP/bin/drift-probe.sh" <<'EOF'
#!/usr/bin/env bash
printf 'probe\n' >> "$DRIFT_LOG"
exit "${DRIFT_PROBE_RC:-0}"
EOF
cat > "$TMP/bin/drift-converge.sh" <<'EOF'
#!/usr/bin/env bash
printf 'converge\n' >> "$DRIFT_LOG"
exit "${DRIFT_CONVERGE_RC:-0}"
EOF
cat > "$TMP/bin/systemctl" <<'EOF'
#!/usr/bin/env bash
if [[ ${2:-} == is-active ]]; then echo inactive; exit 3; fi
exit 0
EOF
chmod +x "$TMP/bin/"*

# ── 仓桩：HEAD==origin/master（收敛态）；commit 推进=未收敛态 ──
REPO="$TMP/repo"; mkdir -p "$REPO"; cd "$REPO"
git init -q && git config user.email t@t && git config user.name t
echo x > f && git add f && git commit -qm init
git update-ref refs/remotes/origin/master HEAD
SHA12=$(git rev-parse --short=12 HEAD)

printf 'FIN_PG_DSN=fake\n' > "$TMP/env"
export PATH="$TMP/bin:$PATH"
export DM_ENV_FILE="$TMP/env"
export O_MECH_REPO="$REPO"
export O_OPS_ESC_SH="$TMP/bin/opsesc.sh"
export O_DM_DEPGATE_SH="$TMP/bin/depgate.sh"
export O_DM_SUSPEND_SH="$TMP/bin/suspend.sh"
export O_DM_HANDOFF_SH="$TMP/bin/handoff.sh"
export O_DM_LEDGER_SH="$TMP/bin/ledger.sh"
export O_VERDICT_DELIVER_SH="$TMP/bin/verdict.sh"
export O_AUTOAPPROVE_SH="$TMP/bin/autoapprove.sh"
export O_DEPLOY_WINDOW_SH="$TMP/bin/win.sh"
export O_DEPLOY_IMAGE_SH="$TMP/bin/deploy_image_stub.sh"
export O_DEPLOY_OCI_SH="$TMP/bin/deploy_oci_stub.sh"
export O_WORKER_DISPATCH_SCRIPT="$TMP/bin/dispatch-worker"
export O_DISPATCH_WATER_SH="$TMP/bin/water"
export O_DM_QUEUE_SH="$TMP/bin/dm-queue"
export O_DRIFT_PROBE_SH="$TMP/bin/drift-probe.sh"
export O_DRIFT_CONVERGE_SH="$TMP/bin/drift-converge.sh"
export O_DRIFT_STATE_DIR="$TMP/drift"
export O_WORKER_TASK_DIR="$TMP/tasks"
export O_MANUAL_LANE_SH=/bin/false
# GW-001 段E1 案卷回写唤件桩——尾块独立挂点实证（断言见⑨）
cat > "$TMP/bin/casewb.sh" <<'EOF'
#!/usr/bin/env bash
echo "casewb" >> "$CASEWB_LOG"
EOF
chmod +x "$TMP/bin/casewb.sh"
export O_CASE_WRITEBACK_SH="$TMP/bin/casewb.sh"
export CASEWB_LOG="$TMP/casewb.log"; : > "$CASEWB_LOG"
export COMP_LOG="$TMP/comp.log"; : > "$COMP_LOG"
export GO_LOG="$TMP/go.log"; : > "$GO_LOG"
export HB_LOG="$TMP/hb.log"; : > "$HB_LOG"
export PULL_LOG="$TMP/pull.log"; : > "$PULL_LOG"
export DEPLOY_LOG="$TMP/deploy.log"; : > "$DEPLOY_LOG"
export ESC_LOG="$TMP/esc.log"; : > "$ESC_LOG"
export DISPATCH_LOG="$TMP/dispatch.log"; : > "$DISPATCH_LOG"
export WATER_LOG="$TMP/water.log"; : > "$WATER_LOG"
export DRIFT_LOG="$TMP/drift.log"; : > "$DRIFT_LOG"
export DISPATCH_RC_SEQ="$TMP/rc.seq"; : > "$DISPATCH_RC_SEQ"

MECH="$ROOT/scripts/meta/mech_tick.sh"
reset_logs() { : > "$COMP_LOG"; : > "$GO_LOG"; : > "$HB_LOG"; : > "$PULL_LOG"; : > "$DEPLOY_LOG"; : > "$ESC_LOG"; : > "$DISPATCH_LOG"; : > "$WATER_LOG"; : > "$DRIFT_LOG"; }

# ── ① 承接面全件+心跳+批空窗零部署 ──
reset_logs
MECH_LOCK_RESULT=t DISPATCH_TIDS='' WINDOW_STATE=批空 bash "$MECH" 2>"$TMP/m1.err"
for c in depgate suspend handoff autoapprove ledger verdict; do
  grep -qx "$c" "$COMP_LOG" || { echo "FAIL ①缺子件 $c"; cat "$COMP_LOG"; exit 1; }
done
grep -q 'ticketdraft' "$GO_LOG" || { echo "FAIL ①缺 ticketdraft"; exit 1; }
grep -q 'ops_headless' "$HB_LOG" || { echo "FAIL ①缺 MECH 心跳"; exit 1; }
[ ! -s "$DEPLOY_LOG" ] && [ ! -s "$ESC_LOG" ] || { echo "FAIL ①批空窗却有部署/升件"; exit 1; }
echo "ok ①承接面六子件+ticketdraft+心跳，批空窗零部署"

# ── ② 应发 image→deploy_image 发；oci→deploy_oci 发 ──
reset_logs
MECH_LOCK_RESULT=t DISPATCH_TIDS='' WINDOW_STATE=应发 WINDOW_FACES=image \
  IDEM_RESULT='' FREEZE_RESULT='' DEPLOY_IMAGE_RC=0 bash "$MECH" 2>/dev/null
grep -qx 'image' "$DEPLOY_LOG" && [ ! -s "$ESC_LOG" ] \
  || { echo "FAIL ②image 应发未发或误升件"; cat "$DEPLOY_LOG" "$ESC_LOG"; exit 1; }
reset_logs
MECH_LOCK_RESULT=t DISPATCH_TIDS='' WINDOW_STATE=应发 WINDOW_FACES=oci \
  IDEM_RESULT='' FREEZE_RESULT='' DEPLOY_OCI_RC=0 bash "$MECH" 2>/dev/null
grep -qx 'oci' "$DEPLOY_LOG" && [ ! -s "$ESC_LOG" ] \
  || { echo "FAIL ②oci 应发未发或误升件"; cat "$DEPLOY_LOG" "$ESC_LOG"; exit 1; }
echo "ok ②应发窗 image/oci 各发其件零升件"

# ── ③ 部署行幂等：同 sha 已有 rc=0 行→跳过不发 ──
reset_logs
MECH_LOCK_RESULT=t DISPATCH_TIDS='' WINDOW_STATE=应发 WINDOW_FACES=image \
  IDEM_RESULT=1 FREEZE_RESULT='' bash "$MECH" 2>/dev/null
[ ! -s "$DEPLOY_LOG" ] && [ ! -s "$ESC_LOG" ] \
  || { echo "FAIL ③幂等命中仍发"; cat "$DEPLOY_LOG" "$ESC_LOG"; exit 1; }
echo "ok ③部署行幂等——已发 sha 重触零重发"

# ── ④ rc≠0→DEPLOY-FIRE-FAIL 升件+冻结期不重试 ──
reset_logs
MECH_LOCK_RESULT=t DISPATCH_TIDS='' WINDOW_STATE=应发 WINDOW_FACES=image \
  IDEM_RESULT='' FREEZE_RESULT='' DEPLOY_IMAGE_RC=1 bash "$MECH" 2>/dev/null
grep -qF "DEPLOY-FIRE-FAIL:$SHA12:image" "$ESC_LOG" \
  || { echo "FAIL ④败未升件"; cat "$ESC_LOG"; exit 1; }
# 冻结窗内重触：FREEZE_RESULT=1→不发不升（去重键面冻结）
reset_logs
MECH_LOCK_RESULT=t DISPATCH_TIDS='' WINDOW_STATE=应发 WINDOW_FACES=image \
  IDEM_RESULT='' FREEZE_RESULT=1 DEPLOY_IMAGE_RC=1 bash "$MECH" 2>/dev/null
[ ! -s "$DEPLOY_LOG" ] && [ ! -s "$ESC_LOG" ] \
  || { echo "FAIL ④冻结期仍发/再升"; cat "$DEPLOY_LOG" "$ESC_LOG"; exit 1; }
echo "ok ④败升件+24h 冻结不重试"

# ── ⑤ HEAD≠origin/master 未收敛→跳过不发 ──
echo y >> "$REPO/f" && git -C "$REPO" add f && git -C "$REPO" commit -qm ahead
reset_logs
MECH_LOCK_RESULT=t DISPATCH_TIDS='' WINDOW_STATE=应发 WINDOW_FACES=image \
  IDEM_RESULT='' FREEZE_RESULT='' bash "$MECH" 2>"$TMP/m5.err"
git -C "$REPO" reset -q --hard HEAD~1
[ ! -s "$DEPLOY_LOG" ] && [ ! -s "$ESC_LOG" ] \
  || { echo "FAIL ⑤未收敛仍发"; cat "$DEPLOY_LOG" "$ESC_LOG"; exit 1; }
grep -q '未收敛' "$TMP/m5.err" || { echo "FAIL ⑤缺未收敛日志"; cat "$TMP/m5.err"; exit 1; }
echo "ok ⑤HEAD≠origin/master 未收敛跳过"

# ── ⑥ 有挡/查询败窗零部署 ──
reset_logs
MECH_LOCK_RESULT=t DISPATCH_TIDS='' WINDOW_STATE=有挡 bash "$MECH" 2>/dev/null
[ ! -s "$DEPLOY_LOG" ] && [ ! -s "$ESC_LOG" ] \
  || { echo "FAIL ⑥有挡窗仍发"; cat "$DEPLOY_LOG" "$ESC_LOG"; exit 1; }
reset_logs
MECH_LOCK_RESULT=t DISPATCH_TIDS='' WINDOW_STATE=查询败 bash "$MECH" 2>/dev/null
[ ! -s "$DEPLOY_LOG" ] \
  || { echo "FAIL ⑥查询败窗仍发"; cat "$DEPLOY_LOG"; exit 1; }
echo "ok ⑥有挡/查询败窗零部署动作"

# ── ⑦ 锁拒即退：零子件零心跳零部署 ──
reset_logs
MECH_LOCK_RESULT=f DISPATCH_TIDS='' WINDOW_STATE=应发 bash "$MECH" 2>"$TMP/m7.err"
grep -q '机械轨闸未得' "$TMP/m7.err" || { echo "FAIL ⑦缺锁拒日志"; cat "$TMP/m7.err"; exit 1; }
[ ! -s "$COMP_LOG" ] && [ ! -s "$HB_LOG" ] && [ ! -s "$DEPLOY_LOG" ] && [ ! -s "$ESC_LOG" ] \
  || { echo "FAIL ⑦锁拒仍有副作用"; cat "$COMP_LOG" "$HB_LOG" "$DEPLOY_LOG"; exit 1; }
echo "ok ⑦锁拒即退零副作用"

# ── ⑧ 派单补拉窗承接：候选票+票件在→dispatch_worker 拉起+拉起行 ──
reset_logs
MECH_LOCK_RESULT=t WINDOW_STATE=批空 \
  DISPATCH_TIDS='T-9001' MODEL_ROW='swe-2-medium' bash "$MECH" 2>/dev/null
grep -q 'T-9001 swe-2-medium' "$DISPATCH_LOG" \
  || { echo "FAIL ⑧补拉未派"; cat "$DISPATCH_LOG"; exit 1; }
grep -q "'拉起'" "$PULL_LOG" \
  || { echo "FAIL ⑧拉起行未落"; cat "$PULL_LOG"; exit 1; }
echo "ok ⑧派单补拉窗承接实证（dispatch_worker+拉起行）"

# ── ⑨ GW-001 段C1：rc75 候水续试下一票不 break——首票候水，次票仍拉起 ──
reset_logs
printf '1\n' > "$DISPATCH_RC_SEQ" # 首次调用 rc75，其后 rc0
MECH_LOCK_RESULT=t WINDOW_STATE=批空 \
  DISPATCH_TIDS='T-9001
T-9002' bash "$MECH" 2>/dev/null
[ "$(grep -c . "$DISPATCH_LOG")" -eq 2 ] \
  || { echo "FAIL ⑨rc75 阻断次票（补拉循环未续试）"; cat "$DISPATCH_LOG"; exit 1; }
grep -qx 'T-9001' "$WATER_LOG" \
  || { echo "FAIL ⑨首票候水标记未落"; cat "$WATER_LOG"; exit 1; }
! grep -q 'T-9002' "$WATER_LOG" \
  || { echo "FAIL ⑨次票误标候水"; cat "$WATER_LOG"; exit 1; }
grep -q "T-9002.*拉起" "$PULL_LOG" || grep -q "'拉起'" "$PULL_LOG" \
  || { echo "FAIL ⑨次票拉起行未落"; cat "$PULL_LOG"; exit 1; }
echo "ok ⑨rc75 候水续试——首票候水次票拉起，全队不断"

# ── ⑩ GW-001 段C3 派单序：补拉候选过 dispatch_order 重排——postmortem 族先派 ──
reset_logs
printf '病灶签名：`postmortem:fast_close`（桩）\n' >> "$TMP/tasks/T-9002-resume.md"
MECH_LOCK_RESULT=t WINDOW_STATE=批空 \
  DISPATCH_TIDS='T-9001
T-9002' bash "$MECH" 2>/dev/null
head -1 "$DISPATCH_LOG" | grep -q '^T-9002 swe-2-flash-high' \
  || { echo "FAIL ⑩派单序未过 dispatch_order（首派=$(head -1 "$DISPATCH_LOG")）"; cat "$DISPATCH_LOG"; exit 1; }
echo "ok ⑩补拉候选派单序重排（postmortem 族先于输入序首派）"

# ── ⑪ 漂移对账窗默认半成品态：每拍只跑探针，执行器不启 ──────
reset_logs
MECH_LOCK_RESULT=t WINDOW_STATE=批空 DISPATCH_TIDS='' bash "$MECH" 2>/dev/null
grep -qx 'probe' "$DRIFT_LOG" \
  || { echo "FAIL ⑪漂移探针未每拍跑"; cat "$DRIFT_LOG"; exit 1; }
! grep -qx 'converge' "$DRIFT_LOG" \
  || { echo "FAIL ⑪默认却跑了自收敛（半成品窗破）"; cat "$DRIFT_LOG"; exit 1; }
echo "ok ⑪默认每拍只跑漂移探针（执行器显式开关）"

# ── ⑫ O_DRIFT_FIX_ENABLE=1→探针+自收敛同拍 ─────────────────
reset_logs
MECH_LOCK_RESULT=t WINDOW_STATE=批空 DISPATCH_TIDS='' O_DRIFT_FIX_ENABLE=1 bash "$MECH" 2>/dev/null
grep -qx 'probe' "$DRIFT_LOG" && grep -qx 'converge' "$DRIFT_LOG" \
  || { echo "FAIL ⑫开关后探针+收敛未同拍"; cat "$DRIFT_LOG"; exit 1; }
echo "ok ⑫O_DRIFT_FIX_ENABLE=1 探针+自收敛同拍"

# ── ⑬ fix_enable 件开写径；探针败升件不阻拍 ─────────────────
mkdir -p "$O_DRIFT_STATE_DIR"; : > "$O_DRIFT_STATE_DIR/fix_enable"
reset_logs
MECH_LOCK_RESULT=t WINDOW_STATE=批空 DISPATCH_TIDS='' DRIFT_PROBE_RC=1 bash "$MECH" 2>/dev/null
grep -qx 'converge' "$DRIFT_LOG" \
  || { echo "FAIL ⑬fix_enable 件未开写径"; cat "$DRIFT_LOG"; exit 1; }
grep -qF '漂移探针败' "$ESC_LOG" \
  || { echo "FAIL ⑬探针败未升件"; cat "$ESC_LOG"; exit 1; }
rm -f "$O_DRIFT_STATE_DIR/fix_enable"
echo "ok ⑬fix_enable 件开写径+探针败升件不阻拍"
# ── ⑭ GW-001 段E1 案卷回写尾块唤件——每拍恰唤一次（幂等扫在件内）──
: > "$CASEWB_LOG"
MECH_LOCK_RESULT=t DISPATCH_TIDS='' WINDOW_STATE=批空 bash "$MECH" 2>/dev/null
[ "$(grep -c '^casewb$' "$CASEWB_LOG")" = "1" ] \
  && echo "ok ⑭案卷回写尾块恰唤一次" || { echo "FAIL ⑭唤件计数=[$(cat "$CASEWB_LOG")]"; exit 1; }
# 败径 fail-soft：唤件 rc≠0 不挡机械轨 exit 0
cat > "$TMP/bin/casewb-fail.sh" <<'EOF'
#!/usr/bin/env bash
exit 1
EOF
chmod +x "$TMP/bin/casewb-fail.sh"
O_CASE_WRITEBACK_SH="$TMP/bin/casewb-fail.sh" \
  MECH_LOCK_RESULT=t DISPATCH_TIDS='' WINDOW_STATE=批空 bash "$MECH" 2>/dev/null
[ "$?" -eq 0 ] && echo "ok ⑮案卷回写败=fail-soft 零挡" || { echo "FAIL ⑮fail-soft 破"; exit 1; }

echo "mech tick scenarios PASS"
