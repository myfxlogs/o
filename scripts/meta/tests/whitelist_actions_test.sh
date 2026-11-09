#!/usr/bin/env bash
# whitelist_actions_test.sh —— T-0509 白名单动作面桩测：psql/systemctl/
# go/dispatch-worker/dm_queue/ceo_queue/dispatch_water 全桩，场景驱动验
# 三层架构（检测→status 判定→执行/呈批/升件）：
#   ① pulled 尸+规则#1 生效→dispatch_worker 重派+「规则#1命中」拉起行；
#     24h 内已有关#1 引证行→零二次动作（防循环钉死）；
#   ② 表外（status=待裁）→「呈批」行落账（seat=MECH ticket=''）零自决；
#   ③ 执行败（dispatch rc≠0/75外）→升件行+同（规则,对象）抑制期零重试；
#   ④ error 相位：伪键/无票件→作废；真票有件/DIAG 键→不可判升件不猜；
#   ⑤ suspended await=绿窗复跑绿→续跑；复跑红/rate-limit→零动作；
#   ⑥ 席位 kick：stall+队列非空→start+自决行；队列空/活跃/冷却→零动作；
#   ⑦ 界内微调：候水≥3 现值=域顶→步降覆写档+参数调整行；候水=0 现值=
#     域底→步升；表外→呈批零写档；覆写档写败→升件。
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
REAL_TSV="$ROOT/scripts/meta/whitelist_actions.tsv"
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin" "$TMP/tasks" "$TMP/repo" "$TMP/q" "$TMP/sc" "$TMP/runtime"

# ── psql 桩：查询全录 $PSQL_LOG；分类应答——检测行读 $Q_DIR/{r1,r2,r3}，
# 计数读 {shui,econ,eng}；去重/抑制/已动类 SELECT 走 $Q_DIR/state
# （行式 kind|ERE——ERE 对压平查询 grep -E）。INSERT 落 $SQL_LOG。 ──
cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
state_hit() { # $1=kind $2=query——state 行 kind|ERE 对压平查询匹配
	local kind=$1 qflat k pat
	qflat=$(printf '%s' "$2" | tr '\n' ' ')
	[ -f "$Q_DIR/state" ] || return 1
	while IFS='|' read -r k pat; do
		[ "$k" = "$kind" ] || continue
		[ -n "$pat" ] && printf '%s' "$qflat" | grep -qE "$pat" && return 0
	done < "$Q_DIR/state"
	return 1
}
query="${*: -1}"
printf '%s\n' "$query" >> "$PSQL_LOG"
[ -n "${PSQL_FAIL_MATCH:-}" ] && [[ "$query" == *"$PSQL_FAIL_MATCH"* ]] && exit 1
case "$query" in
	*"INSERT INTO ops_event"*) printf '%s\n' "$query" >> "$SQL_LOG" ;;
	*"phase='pulled'"*)        cat "$Q_DIR/r1" 2>/dev/null ;;
	*"phase='error'"*)         cat "$Q_DIR/r2" 2>/dev/null ;;
	*"phase='suspended'"*)     cat "$Q_DIR/r3" 2>/dev/null ;;
	*"白名单动作败"*)          if state_hit supp "$query"; then printf '1\n'; fi ;;
	*"event='呈批'"*)          if state_hit bm "$query"; then printf '1\n'; fi ;;
	*"event='拉起'"*"规则#"*"命中"*) if state_hit acted "$query"; then printf '1\n'; fi ;;
	*"event='拉起'"*"挂起"*)   if state_hit r3al "$query"; then printf '1\n'; fi ;;
	*"event='自决'"*"规则#4命中"*) if state_hit r4cd "$query"; then printf '1\n'; fi ;;
	*"event='参数调整'"*"规则#5命中"*) if state_hit r5dd "$query"; then printf '1\n'; fi ;;
	*"event='候水'"*)          cat "$Q_DIR/shui" 2>/dev/null ;;
	*"fin_econ_lesson"*)       cat "$Q_DIR/econ" 2>/dev/null ;;
	*"fin_eng_lesson"*)        cat "$Q_DIR/eng" 2>/dev/null ;;
	*"event='派单'"*)          printf 'swe-2-medium\n' ;;
esac
EOF
cat > "$TMP/bin/go" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$GO_LOG"
exit "${GO_TEST_RC:-0}"
EOF
cat > "$TMP/bin/dispatch-worker" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$DISPATCH_LOG"
exit "${DISPATCH_RC:-0}"
EOF
cat > "$TMP/bin/dispatch-water" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$WATER_LOG"
exit 0
EOF
cat > "$TMP/bin/dm-queue" <<'EOF'
#!/usr/bin/env bash
cat "$Q_DIR/dmqueue" 2>/dev/null
exit "${DMQ_RC:-0}"
EOF
cat > "$TMP/bin/ceo-queue" <<'EOF'
#!/usr/bin/env bash
cat "$Q_DIR/ceoqueue" 2>/dev/null
exit "${CEOQ_RC:-0}"
EOF
cat > "$TMP/bin/systemctl" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$SYSTEMCTL_LOG"
cmd=""; unit=""
for a in "$@"; do
	case "$a" in is-active|show|start) cmd=$a ;; *.service|*.timer) unit=$a ;; esac
done
safe=${unit//\//_}
case "$cmd" in
	is-active) f="$SC_DIR/active_$safe"; [ -f "$f" ] && cat "$f" || echo inactive ;;
	show)      f="$SC_DIR/inactive_$safe"; [ -f "$f" ] && cat "$f" || echo 0; exit 0 ;;
	start)     printf 'start %s\n' "$unit" >> "$START_LOG"; exit "${START_RC:-0}" ;;
esac
exit 0
EOF
chmod +x "$TMP/bin/"*

printf 'FIN_PG_DSN=fake\n' > "$TMP/env"
export PATH="$TMP/bin:$PATH"
export DM_ENV_FILE="$TMP/env" ESCALATE_ENV_FILE="$TMP/env"
export PSQL_LOG="$TMP/psql.log" SQL_LOG="$TMP/sql.log"
export Q_DIR="$TMP/q" SC_DIR="$TMP/sc"
export GO_LOG="$TMP/go.log" DISPATCH_LOG="$TMP/dispatch.log" WATER_LOG="$TMP/water.log"
export SYSTEMCTL_LOG="$TMP/systemctl.log" START_LOG="$TMP/start.log"
export O_WL_TASK_DIR="$TMP/tasks" O_WL_REPO_DIR="$TMP/repo"
export O_WL_DISPATCH="$TMP/bin/dispatch-worker" O_WL_DISPATCH_WATER="$TMP/bin/dispatch-water"
export O_WL_DM_QUEUE="$TMP/bin/dm-queue" O_WL_CEO_QUEUE="$TMP/bin/ceo-queue"
export O_WL_SYSTEMCTL="$TMP/bin/systemctl" O_WL_GO="$TMP/bin/go" O_WL_GO_TIMEOUT=5
export O_WL_MIN_AGE_S=600 O_WORKER_RUNTIME_DIR="$TMP/runtime"
export O_WORKER_PARALLEL_LIMIT=2 GO_TEST_RC=0 DISPATCH_RC=0 START_RC=0 DMQ_RC=0 CEOQ_RC=0
export PSQL_FAIL_MATCH=
WL="$ROOT/scripts/meta/whitelist_actions.sh"

# 票件+人工径判别面：T-9001/T-9002/T-9802 真票件在案（manual_lane rc1 放行）
for t in T-9001 T-9002 T-9802; do printf '# %s 桩票\n' "$t" > "$TMP/tasks/$t-stub.md"; done
printf '# T-9100 人工径票\n执行=人工径（业主在场）\n' > "$TMP/tasks/T-9100-manual.md"

# gen_tsv <out> <rule:status;...>——真 TSV status 列覆写（param/bound/note
# 原样；空字段补 '-' 占位——read 折叠连续 tab 丢位同律）
gen_tsv() {
	awk -F'\t' -v OFS='\t' -v map="$2" '
		BEGIN{n=split(map,a,";");for(i=1;i<=n;i++){split(a[i],b,"=");st[b[1]]=b[2]}}
		$1 ~ /^[0-9]+$/{for(i=1;i<=NF;i++) if($i=="") $i="-"; $5=st[$1];print}' "$REAL_TSV" > "$1"
}
ALL_PENDING="$TMP/wl-pending.tsv"; gen_tsv "$ALL_PENDING" '1=待裁;2=待裁;3=待裁;4=待裁;5=待裁'
R1_ON="$TMP/wl-r1.tsv";    gen_tsv "$R1_ON" '1=生效;2=待裁;3=待裁;4=待裁;5=待裁'
R2_ON="$TMP/wl-r2.tsv";    gen_tsv "$R2_ON" '1=待裁;2=生效;3=待裁;4=待裁;5=待裁'
R3_ON="$TMP/wl-r3.tsv";    gen_tsv "$R3_ON" '1=待裁;2=待裁;3=生效;4=待裁;5=待裁'
R4_ON="$TMP/wl-r4.tsv";    gen_tsv "$R4_ON" '1=待裁;2=待裁;3=待裁;4=生效;5=待裁'
R5_ON="$TMP/wl-r5.tsv";    gen_tsv "$R5_ON" '1=待裁;2=待裁;3=待裁;4=待裁;5=生效'

pass=0; fail=0
ok(){ echo "PASS $1"; pass=$((pass+1)); }
bad(){ echo "FAIL $1"; fail=$((fail+1)); }
ck(){ if eval "$2"; then ok "$1"; else bad "$1"; fi; }

reset_all() {
	for f in psql.log sql.log go.log dispatch.log water.log systemctl.log start.log; do
		: > "$TMP/$f"
	done
	for f in r1 r2 r3 state shui econ eng dmqueue ceoqueue; do
		: > "$Q_DIR/$f"
	done
	rm -f "$SC_DIR"/active_* "$SC_DIR"/inactive_* "$TMP/runtime/parallel-limit-override"
	export GO_TEST_RC=0 DISPATCH_RC=0 START_RC=0 DMQ_RC=0 CEOQ_RC=0 PSQL_FAIL_MATCH=
	export O_WORKER_PARALLEL_LIMIT=2 O_WORKER_RUNTIME_DIR="$TMP/runtime"
}
run_wl() { O_WL_TABLE=$1 bash "$WL"; }

# ══ ① 空集=静默退 ════════════════════════════════════════════════
reset_all
run_wl "$ALL_PENDING"
ck "空集零动作零落账" "[ ! -s '$DISPATCH_LOG' ] && [ ! -s '$SQL_LOG' ] && [ ! -s '$START_LOG' ]"

# ══ ② pulled 尸·规则#1 生效→重派+引证行；24h 已动→零二次 ═══════════
reset_all
printf 'T-9001\t700\n' > "$Q_DIR/r1"
run_wl "$R1_ON"
ck "尸重派 dispatch_worker 被唤" "grep -q '^T-9001 swe-2-medium ' '$DISPATCH_LOG'"
ck "拉起引证行落账（规则#1命中）" "grep -qF \"'拉起','T-9001','规则#1命中\" '$SQL_LOG'"
# 24h 防循环：state 注入已动引证行→同票再跑零动作
printf "acted|ticket='T-9001' AND event='拉起'\n" > "$Q_DIR/state"
: > "$DISPATCH_LOG"
run_wl "$R1_ON"
ck "24h 内二次跑零重派（防循环钉死）" "[ ! -s '$DISPATCH_LOG' ]"

# ══ ③ 表外（待裁）→呈批行零自决 ════════════════════════════════════
reset_all
printf 'T-9001\t700\n' > "$Q_DIR/r1"
run_wl "$ALL_PENDING"
ck "表外呈批行落账（seat=MECH event=呈批）" "grep -qF \"'MECH','呈批','','表外动作呈批：规则#1:T-9001\" '$SQL_LOG'"
ck "表外零自决（无 dispatch 无拉起行）" "[ ! -s '$DISPATCH_LOG' ] && ! grep -qF \"'拉起','T-9001'\" '$SQL_LOG'"
# 呈批 24h 去重：state 注入已呈→不再重复呈
printf "bm|表外动作呈批：规则#1:T-9001\n" > "$Q_DIR/state"
: > "$SQL_LOG"
run_wl "$ALL_PENDING"
ck "呈批 24h 同键去重" "! grep -q '呈批' '$SQL_LOG'"

# ══ ④ 执行败→升件+抑制期零重试 ════════════════════════════════════
reset_all
printf 'T-9001\t700\n' > "$Q_DIR/r1"
export DISPATCH_RC=1
run_wl "$R1_ON"
ck "执行败升件行落账（白名单动作败）" "grep -qF '白名单动作败：规则#1:T-9001' '$SQL_LOG'"
ck "执行败零拉起行" "! grep -qF \"'拉起','T-9001'\" '$SQL_LOG'"
# 升件行已在案→抑制期再跑：dispatch 不再被唤（零重试钉死）
printf "supp|白名单动作败：规则#1:T-9001\n" > "$Q_DIR/state"
: > "$DISPATCH_LOG"; : > "$SQL_LOG"
run_wl "$R1_ON"
ck "抑制期零重试（dispatch 不被唤）" "[ ! -s '$DISPATCH_LOG' ]"
ck "抑制期零新升件" "[ ! -s '$SQL_LOG' ]"
export DISPATCH_RC=0

# ══ ⑤ error 相位：作废/不可判双分支 ════════════════════════════════
reset_all
printf 'T-9802\n伪键-x\nDIAG-12\n' > "$Q_DIR/r2"   # T-9802 有票件→不可判；伪键-x→作废；DIAG-12→不可判
run_wl "$R2_ON"
ck "无票件伪键作废行（规则#2命中）" "grep -qF \"'MECH','作废','伪键-x','规则#2命中\" '$SQL_LOG'"
ck "DIAG 键不可判升件不猜" "grep -qF '白名单不可判：规则#2:DIAG-12' '$SQL_LOG'"
ck "真票有件 error 不可判升件不猜" "grep -qF '白名单不可判：规则#2:T-9802' '$SQL_LOG'"
ck "不可判零作废" "! grep -qF \"'作废','T-9802'\" '$SQL_LOG' && ! grep -qF \"'作废','DIAG-12'\" '$SQL_LOG'"
# 表外：voidable 走呈批，不可判升件仍落（status 不闸不可判径）
reset_all
printf 'T-9802\n伪键-x\nDIAG-12\n' > "$Q_DIR/r2"
run_wl "$ALL_PENDING"
ck "表外 voidable→呈批" "grep -qF '表外动作呈批：规则#2:伪键-x' '$SQL_LOG'"
ck "表外不可判升件仍落（不闸）" "grep -qF '白名单不可判：规则#2:DIAG-12' '$SQL_LOG'"
ck "表外零作废" "! grep -qF '作废' '$SQL_LOG'"

# ══ ⑥ suspended 醒票：绿窗绿→续跑；红/限流/已消解→零动作 ════════════
reset_all
printf 'T-9002\tflake await=绿窗:TestFoo 上轮红\t施工席\n' > "$Q_DIR/r3"
run_wl "$R3_ON"
ck "await 绿窗绿→续跑 dispatch" "grep -q '^T-9002 swe-2-medium ' '$DISPATCH_LOG'"
ck "续跑拉起行（规则#3命中 await 引证）" "grep -qF \"'拉起','T-9002','规则#3命中\" '$SQL_LOG'"
ck "go 复跑收敛 -run 钉" "grep -qF -- '-count=1 -run ^TestFoo' '$GO_LOG' && grep -qF -- './...' '$GO_LOG'"
# 复跑红→零动作零呈批
reset_all
export GO_TEST_RC=1
printf 'T-9002\tflake await=绿窗:TestFoo\t施工席\n' > "$Q_DIR/r3"
run_wl "$R3_ON"
ck "await 复跑红→零动作" "[ ! -s '$DISPATCH_LOG' ] && [ ! -s '$SQL_LOG' ]"
export GO_TEST_RC=0
# rate-limit:auto 挂起→结构性排除（归 dm_tick 既有续跑窗）
reset_all
printf 'T-9002\trate-limit:auto until=x\t施工席\n' > "$Q_DIR/r3"
run_wl "$R3_ON"
ck "rate-limit 挂起零动作（go 都不跑）" "[ ! -s '$GO_LOG' ] && [ ! -s '$DISPATCH_LOG' ]"
# 已消解（拉起晚于挂起）→零动作
reset_all
printf 'T-9002\tflake await=绿窗:TestFoo\t施工席\n' > "$Q_DIR/r3"
printf "r3al|ticket='T-9002' AND event='拉起'\n" > "$Q_DIR/state"
run_wl "$R3_ON"
ck "已消解挂起零续跑" "[ ! -s '$DISPATCH_LOG' ]"

# ══ ⑦ 席位 kick：stall+队列非空→start+自决行 ═══════════════════════
UNIT=o-cto-seat.service
STALL_US=$(( $(date +%s%6N) - 3000000000 ))   # ~50min 前（>2400s 界）
# 场景A：stall 超界+队列非空+生效→kick
reset_all
printf 'inactive\n' > "$SC_DIR/active_$UNIT"
printf '%s\n' "$STALL_US" > "$SC_DIR/inactive_$UNIT"
printf 'T-0506\nT-0507\n' > "$Q_DIR/dmqueue"
run_wl "$R4_ON"
ck "席位 stall+队列非空→systemctl start 唤起" "grep -q \"start $UNIT\" '$START_LOG'"
ck "自决行落账（规则#4命中 引证）" "grep -qF \"'MECH','自决','$UNIT','规则#4命中\" '$SQL_LOG'"
# 场景B：队列空→零 kick
reset_all
printf 'inactive\n' > "$SC_DIR/active_$UNIT"
printf '%s\n' "$STALL_US" > "$SC_DIR/inactive_$UNIT"
run_wl "$R4_ON"
ck "队列空→零 kick" "[ ! -s '$START_LOG' ]"
# 场景C：unit 活跃→零 kick
reset_all
printf 'active\n' > "$SC_DIR/active_$UNIT"
printf 'T-0506\n' > "$Q_DIR/dmqueue"
run_wl "$R4_ON"
ck "席位活跃→零 kick" "[ ! -s '$START_LOG' ]"
# 场景D：冷却窗内已有关#4 自决行→零 kick
reset_all
printf 'inactive\n' > "$SC_DIR/active_$UNIT"
printf '%s\n' "$STALL_US" > "$SC_DIR/inactive_$UNIT"
printf 'T-0506\n' > "$Q_DIR/dmqueue"
printf "r4cd|ticket='$UNIT'.*规则#4命中\n" > "$Q_DIR/state"
run_wl "$R4_ON"
ck "cooldown 内零 kick（防抖钉死）" "[ ! -s '$START_LOG' ]"
# 场景E：表外→呈批零 kick
reset_all
printf 'inactive\n' > "$SC_DIR/active_$UNIT"
printf '%s\n' "$STALL_US" > "$SC_DIR/inactive_$UNIT"
printf 'T-0506\n' > "$Q_DIR/dmqueue"
run_wl "$ALL_PENDING"
ck "表外席位 stall→呈批零 kick" "grep -qF '表外动作呈批：规则#4:o-cto-seat.service' '$SQL_LOG' && [ ! -s '$START_LOG' ]"
# 场景F：kick 败→升件零重试
reset_all
printf 'inactive\n' > "$SC_DIR/active_$UNIT"
printf '%s\n' "$STALL_US" > "$SC_DIR/inactive_$UNIT"
printf 'T-0506\n' > "$Q_DIR/dmqueue"
export START_RC=1
run_wl "$R4_ON"
ck "kick 败→升件行" "grep -qF '白名单动作败：规则#4:o-cto-seat.service' '$SQL_LOG'"
printf "supp|白名单动作败：规则#4:o-cto-seat.service\n" > "$Q_DIR/state"
: > "$START_LOG"; : > "$SQL_LOG"
run_wl "$R4_ON"
ck "kick 败抑制期零重试" "[ ! -s '$START_LOG' ] && [ ! -s '$SQL_LOG' ]"
export START_RC=0

# ══ ⑧ 界内微调：候水驱动步进+覆写档+参数调整行 ══════════════════════
reset_all
printf '4\n' > "$Q_DIR/shui"          # 候水≥3+现值=域顶(2)→步降 1
run_wl "$R5_ON"
ck "拥塞步降→覆写档=1" "grep -qx '1' '$TMP/runtime/parallel-limit-override'"
ck "参数调整行落账（规则#5命中 引证）" "grep -qF \"'MECH','参数调整','','规则#5命中\" '$SQL_LOG'"
# 同态再跑：现值=1=域底→零动作（且 24h dedup 双保险）
: > "$SQL_LOG"
run_wl "$R5_ON"
ck "已到域底零动作" "[ ! -s '$SQL_LOG' ] && grep -qx '1' '$TMP/runtime/parallel-limit-override'"
# 空载步升：候水=0+现值=域底→步升 2
reset_all
printf '0\n' > "$Q_DIR/shui"
printf '1\n' > "$TMP/runtime/parallel-limit-override"
export O_WORKER_PARALLEL_LIMIT=1
run_wl "$R5_ON"
ck "空载步升→覆写档=2" "grep -qx '2' '$TMP/runtime/parallel-limit-override'"
ck "步升参数调整行" "grep -qF '规则#5命中' '$SQL_LOG'"
export O_WORKER_PARALLEL_LIMIT=2
# 表外→呈批零写档
reset_all
printf '4\n' > "$Q_DIR/shui"
run_wl "$ALL_PENDING"
ck "表外界内微调触发→呈批零写档" "grep -qF '表外动作呈批：规则#5:O_WORKER_PARALLEL_LIMIT' '$SQL_LOG' && [ ! -f '$TMP/runtime/parallel-limit-override' ]"
# 覆写档写败（runtime dir 只读）→升件不静默
reset_all
printf '4\n' > "$Q_DIR/shui"
mkdir -p "$TMP/rodir" && chmod 555 "$TMP/rodir"
O_WORKER_RUNTIME_DIR="$TMP/rodir" O_WL_TABLE="$R5_ON" bash "$WL"
chmod 755 "$TMP/rodir"
ck "覆写档写败→升件行" "grep -qF '白名单动作败：规则#5:O_WORKER_PARALLEL_LIMIT' '$SQL_LOG'"
ck "写败零覆写档零参数调整" "[ ! -f '$TMP/rodir/parallel-limit-override' ] && ! grep -qF '参数调整' '$SQL_LOG'"

echo "whitelist_actions scenarios: $pass pass / $fail fail"
[ "$fail" = 0 ]
