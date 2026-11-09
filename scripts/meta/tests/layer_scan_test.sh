#!/usr/bin/env bash
# layer_scan_test.sh —— T-0507 层扫窗桩测：psql 全桩（/*ls:* 标记分发+
# ops_esc 24h 前缀去重桩判重——壳写词形与 SQL 查词形不自洽即炸）+
# 真 escalate.sh 写径落日志 + 真子件桩（dm_queue/manual_lane/deploy/
# systemctl/真 git stub 仓）。判据（票面回放三病灶+外围）：
# ①回放：dispatched-无拉 / pulled-无unit / 幽灵票件 各对应升件行落账；
# ②24h dedup：同对象同类目复跑零重报（ops_esc 前缀去重实证）；
# ③digest 行每窗必落（含全绿态）+ 队列老化/done翻漏/done无收口链/
#   账本抽查/外事聚合/容量 streak 阶梯各面正负对照；
# ④降级面：FIN_DSN 缺=静默退 / 相位查询败=exit1 / systemctl 败=unit=unk
#   不误报 / dm_queue 败=cap=skip 不评 streak / 候水票剔出候选集。
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin" "$TMP/repo" "$TMP/state"

# ── git stub 仓：master+refs/remotes/origin/master 双钉（账本抽查
#    --grep 实解+origin 落后龄双用；fetch 无远端败=滑行径）──────────
git -C "$TMP/repo" init -q -b master
git -C "$TMP/repo" config user.email t@t && git -C "$TMP/repo" config user.name t
echo base > "$TMP/repo/f" && git -C "$TMP/repo" add f
git -C "$TMP/repo" commit -qm 'base'
git -C "$TMP/repo" update-ref refs/remotes/origin/master master
SYNC_SHA=$(git -C "$TMP/repo" rev-parse master)

# ── psql 全桩：/*ls:* 标记分发场景数据；dedup SELECT 按 LIKE 字面
#    对 ESC_SQL_LOG 判重（evergreen——复跑重报即炸）；INSERT 分
#    digest('层扫','层扫')/升件('升件') 两日志 ─────────────────────
cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
query="${*: -1}"
case "$query" in
  *"/*ls:phasemap*/"*)    printf '%b\n' "${PHASE_ROWS:-}" ;;
  *"/*ls:machev*/"*)      printf '%b\n' "${MACHEV_ROWS:-}" ;;
  *"/*ls:donechain*/"*)   printf '%b\n' "${DONECHAIN_ROWS:-}" ;;
  *"/*ls:done5*/"*)       printf '%b\n' "${DONE5_ROWS:-}" ;;
  *"/*ls:water*/"*)       printf '%b\n' "${WATER_ROWS:-}" ;;
  *"/*ls:pulls*/"*)       printf '%s\n' "${PULLS_N:-0}" ;;
  *"/*ls:endpoint*/"*)    printf '%s\n' "${DEPLOY_TS_ROWS:-}" ;;
  *"/*ls:unclaimed*/"*)   printf '%b\n' "${UNCLAIMED_ROWS:-}" ;;
  *"note LIKE '"*"24 hours"*)
    # ops_esc 24h 前缀去重——LIKE 字面对升件写径日志判（壳写=SQL 查同形）
    prefix=$(sed -nE "s/.*note LIKE '([^']*)%'.*/\1/p" <<<"$query")
    [ -n "$prefix" ] && grep -Fq "$prefix" "$ESC_SQL_LOG" 2>/dev/null && echo 1
    ;;
  *"INSERT INTO ops_event"*"'层扫','层扫'"*)
    printf '%s\n' "$query" >> "$DIGEST_SQL_LOG" ;;
  *"INSERT INTO ops_event"*)
    printf '%s\n' "$query" >> "$ESC_SQL_LOG" ;;
esac
EOF
chmod +x "$TMP/bin/psql"

# ── 子件桩 ────────────────────────────────────────────────────────
cat > "$TMP/bin/dm-queue" <<'EOF'
#!/usr/bin/env bash
[ "${DMQ_RC:-0}" -eq 0 ] || exit "$DMQ_RC"
[ "${1:-}" = dispatched ] && printf '%b\n' "${CAND_TIDS:-}"
EOF
cat > "$TMP/bin/manual-lane" <<'EOF'
#!/usr/bin/env bash
case " ${MANUAL_TIDS:-} " in *" $1 "*) exit 0;; esac
exit 1
EOF
cat > "$TMP/bin/deploy-due" <<'EOF'
#!/usr/bin/env bash
printf '%b\n' "${DEPLOY_DUE_OUT:-state=批空\nfaces=无\nreason=批空不发窗}"
EOF
cat > "$TMP/bin/systemctl" <<'EOF'
#!/usr/bin/env bash
[ "${SCTL_RC:-0}" -eq 0 ] || exit "$SCTL_RC"
printf '%b\n' "${UNITS_OUT:-}"
EOF
cat > "$TMP/bin/curl" <<'EOF'
#!/usr/bin/env bash
# 端点探针桩——末参=URL；Ping 径/Ping 词分码，余=SPA 根码
url="${@: -1}"
case "$url" in
  *Ping*) printf '%s' "${PING_CODE:-200}" ;;
  *)      printf '%s' "${SPA_CODE:-200}" ;;
esac
EOF
chmod +x "$TMP/bin/dm-queue" "$TMP/bin/manual-lane" "$TMP/bin/deploy-due" "$TMP/bin/systemctl" "$TMP/bin/curl"

printf 'FIN_PG_DSN=fake\n' > "$TMP/env"
: > "$TMP/env-empty"
export PATH="$TMP/bin:$PATH"
export DM_ENV_FILE="$TMP/env"
export O_LAYER_SCAN_REPO="$TMP/repo"
export O_LAYER_SCAN_TASK_DIR="$TMP/repo/docs/tasks"
export O_LAYER_SCAN_DONE_DIR="$TMP/repo/docs/tasks/done"
export O_LAYER_SCAN_STATE_DIR="$TMP/state"
export O_LAYER_SCAN_SYSTEMCTL="$TMP/bin/systemctl"
export O_DM_QUEUE_SH="$TMP/bin/dm-queue"
export O_MANUAL_LANE_SH="$TMP/bin/manual-lane"
export O_LAYER_SCAN_DEPLOY_SH="$TMP/bin/deploy-due"
export LSCAN="$ROOT/scripts/meta/layer_scan.sh"
export ESC_SQL_LOG="$TMP/esc.sql.log"; : > "$ESC_SQL_LOG"
export DIGEST_SQL_LOG="$TMP/digest.sql.log"; : > "$DIGEST_SQL_LOG"
mkdir -p "$O_LAYER_SCAN_TASK_DIR" "$O_LAYER_SCAN_DONE_DIR"

esc_count() { grep -c "INSERT INTO ops_event" "$ESC_SQL_LOG" || true; } # tolerate: grep 零行=计 0 合法径
esc_grep() { grep -F "层扫定性$1" "$ESC_SQL_LOG"; }                    # rc 透传——断言面判据
reset_logs() { : > "$ESC_SQL_LOG"; : > "$DIGEST_SQL_LOG"; rm -f "$TMP/state/layer_scan.state"; }

# ══ ① 回放三病灶：dispatched-无拉 / pulled-无unit(+worker-无收口同源)
#    / 幽灵票件 —— 各对应升件行落账；digest 同窗落 ══════════════════
reset_logs
PHASE_ROWS='T-9001\x1fdispatched\x1f派单\x1f4000\nT-9010\x1fpulled\x1f领单\x1f4000\nT-0040\x1fapproved\x1f批件\x1f100'
MACHEV_ROWS='T-9010'
# 票件补齐——T-9001/T-9010 有件（幽灵检不误伤）；T-0040 故意无件=幽灵
# （合法号域 T-[0-8]xxx——T-9xxx 无件归 g 无籍行非 b①）
printf '# T-9001\n' > "$O_LAYER_SCAN_TASK_DIR/T-9001-x.md"
printf '# T-9010\n' > "$O_LAYER_SCAN_TASK_DIR/T-9010-x.md"
PHASE_ROWS="$PHASE_ROWS" MACHEV_ROWS="$MACHEV_ROWS" bash "$LSCAN"
esc_grep 'dispatched-无拉:T-9001' >/dev/null \
  || { echo "FAIL ① dispatched-无拉未报"; cat "$ESC_SQL_LOG"; exit 1; }
esc_grep 'pulled-无unit:T-9010' >/dev/null \
  || { echo "FAIL ① pulled-无unit未报"; cat "$ESC_SQL_LOG"; exit 1; }
esc_grep 'worker-无收口:T-9010' >/dev/null \
  || { echo "FAIL ① worker-无收口未报（pulled 滞留同源面）"; cat "$ESC_SQL_LOG"; exit 1; }
esc_grep '幽灵票件:T-0040' >/dev/null \
  || { echo "FAIL ① 幽灵票件未报"; cat "$ESC_SQL_LOG"; exit 1; }
! esc_grep '无籍行:T-0040' | grep -q . \
  || { echo "FAIL ① 合法域幽灵误报无籍行"; cat "$ESC_SQL_LOG"; exit 1; }
grep -q "VALUES('MECH','层扫','层扫'" "$DIGEST_SQL_LOG" \
  || { echo "FAIL ① digest 行未落"; cat "$DIGEST_SQL_LOG"; exit 1; }
grep -q 'queue=[1-9]' "$DIGEST_SQL_LOG" || grep -q 'entity=[1-9]' "$DIGEST_SQL_LOG" \
  || { echo "FAIL ① digest 异常计数缺席"; cat "$DIGEST_SQL_LOG"; exit 1; }
echo "ok ①回放三病灶各报+digest同窗落"

# ══ ② 24h dedup：同对象同类目复跑零重报（ops_esc 前缀去重实证）═══
N1=$(esc_count)
PHASE_ROWS='T-9001\x1fdispatched\x1f派单\x1f4000' bash "$LSCAN"
N2=$(esc_count)
[ "$N1" = "$N2" ] || { echo "FAIL ② dedup 破——复跑重报 $N1→$N2"; cat "$ESC_SQL_LOG"; exit 1; }
echo "ok ②同对象同类目 24h 一行不刷屏"

# ══ ③ 队列面阴性对照：unit 在飞/人工径/龄未超阈=零报 ══════════════
reset_logs
PHASE_ROWS='T-9002\x1fdispatched\x1f派单\x1f4000\nT-9003\x1fdispatched\x1f派单\x1f4000\nT-9004\x1fdispatched\x1f派单\x1f100\nT-9011\x1fpulled\x1f领单\x1f4000\nT-9012\x1fpulled\x1f领单\x1f4000'
MACHEV_ROWS='T-9012' # T-9011 无机器拉起证=交互径合法不判
UNITS_OUT='o-worker-T-9002.service loaded active running x\no-worker-T-9012.service loaded active running x'
MANUAL_TIDS='T-9003'
for t in T-9002 T-9003 T-9004 T-9011 T-9012; do printf '# %s\n' "$t" > "$O_LAYER_SCAN_TASK_DIR/$t-x.md"; done
PHASE_ROWS="$PHASE_ROWS" MACHEV_ROWS="$MACHEV_ROWS" UNITS_OUT="$UNITS_OUT" MANUAL_TIDS="$MANUAL_TIDS" bash "$LSCAN"
[ "$(esc_count)" -eq 0 ] || { echo "FAIL ③ 阴性对照误报"; cat "$ESC_SQL_LOG"; exit 1; }
echo "ok ③unit在飞/人工径/龄未超/无机器证 全零报"

# ══ ④ 老化面正报：review/approval_pending>2h、draft>1h ════════════
reset_logs
PHASE_ROWS='T-9030\x1freview\x1f审件\x1f8000\nT-9031\x1freview\x1f审件\x1f100\nT-9032\x1fapproval_pending\x1f呈批\x1f8000\nT-9033\x1fdraft\x1f草案\x1f4000\nT-9034\x1fdraft\x1f草案\x1f100'
for t in T-9030 T-9031 T-9032 T-9033 T-9034; do printf '# %s\n' "$t" > "$O_LAYER_SCAN_TASK_DIR/$t-x.md"; done
PHASE_ROWS="$PHASE_ROWS" bash "$LSCAN"
esc_grep 'review-老化:T-9030' >/dev/null || { echo "FAIL ④ review-老化未报"; cat "$ESC_SQL_LOG"; exit 1; }
esc_grep 'approval-老化:T-9032' >/dev/null || { echo "FAIL ④ approval-老化未报"; cat "$ESC_SQL_LOG"; exit 1; }
esc_grep 'draft-滞留:T-9033' >/dev/null || { echo "FAIL ④ draft-滞留未报"; cat "$ESC_SQL_LOG"; exit 1; }
! esc_grep 'review-老化:T-9031' | grep -q . || { echo "FAIL ④ 龄未超误报"; cat "$ESC_SQL_LOG"; exit 1; }
! esc_grep 'draft-滞留:T-9034' | grep -q . || { echo "FAIL ④ draft 阴性误报"; cat "$ESC_SQL_LOG"; exit 1; }
echo "ok ④老化三类目正报+龄未超零报"

# ══ ⑤ done/ 翻漏 + done 无收口链 ═══════════════════════════════════
reset_logs
PHASE_ROWS='T-9050\x1freview\x1f审件\x1f9000\nT-9051\x1freview\x1f审件\x1f9000\nT-9052\x1fdone\x1f收口\x1f9000'
printf '# T-9050\n' > "$O_LAYER_SCAN_DONE_DIR/T-9050-x.md"
printf '# T-9051\n' > "$O_LAYER_SCAN_DONE_DIR/T-9051-x.md"
printf '# T-9052\n' > "$O_LAYER_SCAN_DONE_DIR/T-9052-x.md"
touch -d '2 hours ago' "$O_LAYER_SCAN_DONE_DIR/T-9050-x.md"   # 件龄>宽限
touch -d '10 minutes ago' "$O_LAYER_SCAN_DONE_DIR/T-9051-x.md" # 件龄<宽限
touch -d '2 hours ago' "$O_LAYER_SCAN_DONE_DIR/T-9052-x.md"
DONECHAIN_ROWS='T-9060'
PHASE_ROWS="$PHASE_ROWS" DONECHAIN_ROWS="$DONECHAIN_ROWS" bash "$LSCAN"
esc_grep 'done-翻漏:T-9050' >/dev/null || { echo "FAIL ⑤ done-翻漏未报"; cat "$ESC_SQL_LOG"; exit 1; }
! esc_grep 'done-翻漏:T-9051' | grep -q . || { echo "FAIL ⑤ 宽限内误报"; cat "$ESC_SQL_LOG"; exit 1; }
! esc_grep 'done-翻漏:T-9052' | grep -q . || { echo "FAIL ⑤ done相位误报翻漏"; cat "$ESC_SQL_LOG"; exit 1; }
esc_grep 'done-无收口链:T-9060' >/dev/null || { echo "FAIL ⑤ done-无收口链未报"; cat "$ESC_SQL_LOG"; exit 1; }
rm -f "$O_LAYER_SCAN_DONE_DIR"/T-905*.md
echo "ok ⑤done翻漏宽限/相位守卫+无收口链正报"

# ══ ⑥ 账本抽查：commit 可解+回填在案=零报；不可解/回填缺席=报 ════
reset_logs
# T-9201 全绿（commit 含票号+票面状态=完成）；T-9202 无 commit；T-9203 有 commit 票面=施工中
git -C "$TMP/repo" commit -qm '[T-9201] 收口' --allow-empty
git -C "$TMP/repo" commit -qm '[T-9203] 收口' --allow-empty
git -C "$TMP/repo" update-ref refs/remotes/origin/master master # 同步——origin 面不串扰
printf '# T-9201\n状态：完成——x\n' > "$O_LAYER_SCAN_TASK_DIR/T-9201-x.md"
printf '# T-9203\n状态：施工中——x\n' > "$O_LAYER_SCAN_TASK_DIR/T-9203-x.md"
DONE5_ROWS='T-9201\nT-9202\nT-9203' bash "$LSCAN"
! esc_grep '账本-commit不可解:T-9201' | grep -q . || { echo "FAIL ⑥ commit 可解误报"; cat "$ESC_SQL_LOG"; exit 1; }
esc_grep '账本-commit不可解:T-9202' >/dev/null || { echo "FAIL ⑥ commit 不可解未报"; cat "$ESC_SQL_LOG"; exit 1; }
esc_grep '账本-回填缺席:T-9203' >/dev/null || { echo "FAIL ⑥ 回填缺席未报"; cat "$ESC_SQL_LOG"; exit 1; }
rm -f "$O_LAYER_SCAN_TASK_DIR"/T-920*.md
echo "ok ⑥账本抽查三态（全绿/commit不可解/回填缺席）"

# ══ ⑦ 外事聚合：origin 落后>2h / deploy 批龄>12h ══════════════════
reset_logs
# origin 落后：master 领先一件且 ct=远古（>2h 阈）
GIT_AUTHOR_DATE='2020-01-01T00:00:00Z' GIT_COMMITTER_DATE='2020-01-01T00:00:00Z' \
  git -C "$TMP/repo" commit -qm 'unpushed old' --allow-empty
DEPLOY_DUE_OUT='state=应发\nfaces=image\nreason=批非空 commits=3 tickets=T-1,T-2,T-3 img=abc oci=x(stat=y) 批首件龄=50000s' \
  bash "$LSCAN"
esc_grep 'origin-落后:origin/master' >/dev/null || { echo "FAIL ⑦ origin-落后未报"; cat "$ESC_SQL_LOG"; exit 1; }
esc_grep 'deploy-批龄:deploy面' >/dev/null || { echo "FAIL ⑦ deploy-批龄未报"; cat "$ESC_SQL_LOG"; exit 1; }
# 收回：master 回同步点+deploy 批龄新鲜——双面归零
git -C "$TMP/repo" update-ref refs/heads/master "$SYNC_SHA"
reset_logs
DEPLOY_DUE_OUT='state=应发\nfaces=image\nreason=批非空 commits=1 tickets=T-1 img=abc oci=x(stat=y) 批首件龄=100s' \
  bash "$LSCAN"
! esc_grep 'origin-落后' | grep -q . || { echo "FAIL ⑦ origin 阴性误报"; cat "$ESC_SQL_LOG"; exit 1; }
! esc_grep 'deploy-批龄' | grep -q . || { echo "FAIL ⑦ deploy 阴性误报"; cat "$ESC_SQL_LOG"; exit 1; }
echo "ok ⑦外事聚合正报+收敛归零"

# ══ ⑧ 容量水位 streak：候选非空∧拉起=0 连 2 拍=拉起-空转、3 拍=
#    spawn-连败；候水票剔出/有拉起/候选空=streak 归零 ═══════════════
reset_logs
# 预置态：上拍基线 400s 前、streak=1——本拍 candidates 非空 pulls=0 → streak=2
printf 'last_scan=%s\nstreak=1\n' "$(( $(date +%s) - 400 ))" > "$TMP/state/layer_scan.state"
CAND_TIDS='T-9080' PULLS_N=0 WATER_ROWS='' bash "$LSCAN"
esc_grep '拉起-空转:候选集' >/dev/null || { echo "FAIL ⑧ 拉起-空转未报"; cat "$ESC_SQL_LOG"; exit 1; }
! esc_grep 'spawn-连败' | grep -q . || { echo "FAIL ⑧ 未满 3 拍误报连败"; cat "$ESC_SQL_LOG"; exit 1; }
# 再拍：streak=2→3 → spawn-连败上档（dedup 对象同=拉起-空转不重报）
CAND_TIDS='T-9080' PULLS_N=0 WATER_ROWS='' bash "$LSCAN"
esc_grep 'spawn-连败:候选集' >/dev/null || { echo "FAIL ⑧ spawn-连败未报"; cat "$ESC_SQL_LOG"; exit 1; }
# 阴性：候选全候水标记 → EFF 空 → streak 归零零报
reset_logs
printf 'last_scan=%s\nstreak=2\n' "$(( $(date +%s) - 400 ))" > "$TMP/state/layer_scan.state"
CAND_TIDS='T-9080' PULLS_N=0 WATER_ROWS='T-9080' bash "$LSCAN"
! esc_grep '拉起-空转' | grep -q . || { echo "FAIL ⑧ 候水票误计败拍"; cat "$ESC_SQL_LOG"; exit 1; }
# 有拉起事件 → streak 归零
grep -q 'streak=0' "$TMP/state/layer_scan.state" \
  || { echo "FAIL ⑧ streak 未归零"; cat "$TMP/state/layer_scan.state"; exit 1; }
echo "ok ⑧容量 streak 阶梯+候水剔出+归零"

# ══ ⑨ f 用户端点健康：全绿零报 / 红+无部署行=报 / 红+部署行≤5min
#    =瞬断豁免 / 红+部署行>5min=报 / 部署行查询败=按无部署行报 ═══════
reset_logs
SPA_CODE=200 PING_CODE=200 bash "$LSCAN"
! esc_grep '端点失健' | grep -q . || { echo "FAIL ⑨ 端点全绿误报"; cat "$ESC_SQL_LOG"; exit 1; }
# 红+无部署行（DEPLOY_TS_ROWS 空）→ 报
SPA_CODE=502 PING_CODE=200 DEPLOY_TS_ROWS='' bash "$LSCAN"
esc_grep '端点失健:dashboard' >/dev/null || { echo "FAIL ⑨ 红+无部署行未报"; cat "$ESC_SQL_LOG"; exit 1; }
# 红+Ping 独红（SPA 绿）→ 同报（任一红即对象红）
reset_logs
SPA_CODE=200 PING_CODE=502 DEPLOY_TS_ROWS='' bash "$LSCAN"
esc_grep '端点失健:dashboard' >/dev/null || { echo "FAIL ⑨ Ping 独红未报"; cat "$ESC_SQL_LOG"; exit 1; }
# 红+部署行新鲜（≤5min）→ 瞬断豁免零报
reset_logs
SPA_CODE=502 PING_CODE=502 DEPLOY_TS_ROWS="$(date +%s)" bash "$LSCAN"
! esc_grep '端点失健' | grep -q . || { echo "FAIL ⑨ 重启瞬断误报"; cat "$ESC_SQL_LOG"; exit 1; }
# 红+部署行老（>5min）→ 报
reset_logs
SPA_CODE=502 PING_CODE=502 DEPLOY_TS_ROWS="$(( $(date +%s) - 400 ))" bash "$LSCAN"
esc_grep '端点失健:dashboard' >/dev/null || { echo "FAIL ⑨ 重启后仍红>5min 未报"; cat "$ESC_SQL_LOG"; exit 1; }
echo "ok ⑨端点健康四态（绿/无部署红/瞬断豁免/老部署红）"

# ══ ⑩ 全绿态：零升件但 digest 照落（静默≠缺席）══════════════════════
reset_logs
PHASE_ROWS='' MACHEV_ROWS='' DONECHAIN_ROWS='' DONE5_ROWS='' CAND_TIDS='' \
  UNITS_OUT='' DEPLOY_DUE_OUT='state=批空\nfaces=无\nreason=批空' bash "$LSCAN"
[ "$(esc_count)" -eq 0 ] || { echo "FAIL ⑩ 全绿态误报"; cat "$ESC_SQL_LOG"; exit 1; }
grep -q "VALUES('MECH','层扫','层扫'" "$DIGEST_SQL_LOG" \
  || { echo "FAIL ⑩ 全绿 digest 未落"; cat "$DIGEST_SQL_LOG"; exit 1; }
grep -q 'queue=0 entity=0 capacity=0 ledger=0 external=0' "$DIGEST_SQL_LOG" \
  || { echo "FAIL ⑩ digest 异常计数非零"; cat "$DIGEST_SQL_LOG"; exit 1; }
echo "ok ⑩全绿态零升件+digest照落"

# ══ ⑪ 降级面：FIN_DSN 缺=静默退 / 相位查询败=exit1 / systemctl 败
#    =unit=unk 不误报 / dm_queue 败=cap=skip ═════════════════════════
reset_logs
DM_ENV_FILE="$TMP/env-empty" bash "$LSCAN"
[ ! -s "$ESC_SQL_LOG" ] && [ ! -s "$DIGEST_SQL_LOG" ] \
  || { echo "FAIL ⑪ FIN_DSN 缺仍写行"; exit 1; }
echo "ok ⑪a FIN_DSN 缺静默退"

# 相位查询败=本体败 exit1（T-0372 败≠空集——mech_tick 升件面）
mkdir -p "$TMP/bin-fail"
printf '#!/usr/bin/env bash\nexit 9\n' > "$TMP/bin-fail/psql"
chmod +x "$TMP/bin-fail/psql"
PATH_SAVE=$PATH; PATH="$TMP/bin-fail:$PATH"
if DM_ENV_FILE="$TMP/env" bash "$LSCAN" 2>/dev/null; then
  echo "FAIL ⑪b 相位查询败未 exit1"; PATH=$PATH_SAVE; exit 1
fi
PATH=$PATH_SAVE
echo "ok ⑪b 相位查询败=exit1"

# systemctl 败=unit 面不可判降级（digest 载 unit=unk——dispatched 老票
# 无 unit 判据整体跳过不误报全场无 unit）
reset_logs
PHASE_ROWS='T-9001\x1fdispatched\x1f派单\x1f4000'
printf '# T-9001\n' > "$O_LAYER_SCAN_TASK_DIR/T-9001-x.md"
SCTL_RC=1 PHASE_ROWS="$PHASE_ROWS" bash "$LSCAN"
! esc_grep 'dispatched-无拉' | grep -q . || { echo "FAIL ⑪c unit 不可判误报"; cat "$ESC_SQL_LOG"; exit 1; }
grep -q 'unit=unk' "$DIGEST_SQL_LOG" || { echo "FAIL ⑪c digest 未载 unit=unk"; cat "$DIGEST_SQL_LOG"; exit 1; }
rm -f "$O_LAYER_SCAN_TASK_DIR/T-9001-x.md"
echo "ok ⑪c systemctl 败=不可判降级不误报"

# dm_queue 败=cap 面不可判（streak 态保留——digest 载 cap=skip）
reset_logs
printf 'last_scan=%s\nstreak=2\n' "$(( $(date +%s) - 400 ))" > "$TMP/state/layer_scan.state"
DMQ_RC=1 PHASE_ROWS='' bash "$LSCAN"
grep -q 'cap=skip' "$DIGEST_SQL_LOG" || { echo "FAIL ⑪d digest 未载 cap=skip"; cat "$DIGEST_SQL_LOG"; exit 1; }
grep -q 'streak=2' "$TMP/state/layer_scan.state" \
  || { echo "FAIL ⑪d streak 态被清（未知≠空集破）"; cat "$TMP/state/layer_scan.state"; exit 1; }
echo "ok ⑪d dm_queue 败=streak 保留+cap=skip"

# ══ ⑫ g 无籍相位行 + h 升件失领（返修1 补面——T-9001 幻影/失领
#    38h 实证业主令）═══════════════════════════════════════════
reset_logs
# g① T-9xxx 夹具段无件=无籍行（非幽灵票件——两族对象域分判钉）
PHASE_ROWS='T-9015\x1fdispatched\x1f派单\x1f4000\nBAD-KEY-9\x1fworking\x1f开工\x1f9000\nT-0044\x1fdone\x1f完成\x1f90000\nT-9016\x1fdispatched\x1f派单\x1f4000'
printf '# T-9016\n' > "$O_LAYER_SCAN_TASK_DIR/T-9016-x.md" # T-9xxx 有件=不判
PHASE_ROWS="$PHASE_ROWS" bash "$LSCAN"
esc_grep '无籍行:T-9015' >/dev/null \
  || { echo "FAIL ⑫ T-9xxx 无件未报无籍行"; cat "$ESC_SQL_LOG"; exit 1; }
! esc_grep '幽灵票件:T-9015' | grep -q . \
  || { echo "FAIL ⑫ T-9xxx 错报幽灵票件（对象域混）"; cat "$ESC_SQL_LOG"; exit 1; }
esc_grep '无籍行:BAD-KEY-9' >/dev/null \
  || { echo "FAIL ⑫ 非T票形伪键未报"; cat "$ESC_SQL_LOG"; exit 1; }
esc_grep '无籍行:T-0044' >/dev/null \
  || { echo "FAIL ⑫ 合法域终相无件未报"; cat "$ESC_SQL_LOG"; exit 1; }
! esc_grep '无籍行:T-9016' | grep -q . \
  || { echo "FAIL ⑫ T-9xxx 有件误报"; cat "$ESC_SQL_LOG"; exit 1; }
# g② dedup：同对象同类目 24h 一行
N1=$(esc_count)
PHASE_ROWS="$PHASE_ROWS" bash "$LSCAN"
[ "$(esc_count)" = "$N1" ] || { echo "FAIL ⑫ g dedup 破"; cat "$ESC_SQL_LOG"; exit 1; }
# h① 失领升件分组报出+认领后零报+dedup
reset_logs
PHASE_ROWS='' UNCLAIMED_ROWS='T-7777\x1f2\x1f14400\nT-7778\x1f1\x1f20000' bash "$LSCAN"
esc_grep '升件失领:T-7777' >/dev/null \
  || { echo "FAIL ⑫ 失领未报 T-7777"; cat "$ESC_SQL_LOG"; exit 1; }
esc_grep '升件失领:T-7778' >/dev/null \
  || { echo "FAIL ⑫ 失领未报 T-7778"; cat "$ESC_SQL_LOG"; exit 1; }
N2=$(esc_count)
PHASE_ROWS='' UNCLAIMED_ROWS='T-7777\x1f2\x1f14400' bash "$LSCAN"
[ "$(esc_count)" = "$N2" ] || { echo "FAIL ⑫ h dedup 破"; cat "$ESC_SQL_LOG"; exit 1; }
reset_logs
PHASE_ROWS='' UNCLAIMED_ROWS='' bash "$LSCAN"
! esc_grep '升件失领' | grep -q . \
  || { echo "FAIL ⑫ 认领后零报破"; cat "$ESC_SQL_LOG"; exit 1; }
rm -f "$O_LAYER_SCAN_TASK_DIR/T-9016-x.md"
echo "ok ⑫ g无籍行四域分判+h失领报出/认领零报/dedup"

echo "layer scan scenarios PASS"
