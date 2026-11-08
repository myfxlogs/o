#!/usr/bin/env bash
# dispatch_water_state_test.sh —— T-0463 派单候水态桩测（ops_event 生产谱形
# fixture）：rc75→候水行→哨兵免报→老龄鸣全链断言。
# 面分五段：
#  A dispatch_water.sh emit 单测——钦定词形（seat=CTO/event=候水/note 字面）
#    +幂等闸谓词钉（最新事件 IS DISTINCT FROM '候水'）+FIN_DSN 缺静默退
#    +票号畸形拒写；
#  B dm_depgate 即派 rc75 写点——候选零交集→派单行落→dispatch rc75→
#    候水 emit 被调（rc0/rc1 面零调）；
#  C dm_tick 补拉窗 rc75 写点——候选集首件 rc75→候水 emit+break
#    （次件不烧闸）；rc0 面=拉起行零候水；
#  D 哨兵免报——dm_handoff 扫描 SQL 派单接行集含 '候水'（配对机制面
#    NOT EXISTS 序后判据由 dm_handoff_test 在案覆盖，本件钉词表接入）；
#  E 老龄鸣——gauges.tsv dispatch_aged_water 行经 gauges_tick 实跑：
#    count=0 静默/count≥1 首拍即鸣（连鸣律）/{d} 携老化票清单/次拍续鸣。
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
META="$ROOT/scripts/meta"

# ══ A. dispatch_water.sh emit 单测 ══════════════════════════════════
mkdir -p "$TMP/a/bin"
cat > "$TMP/a/bin/psql" <<'EOF'
#!/usr/bin/env bash
query="${*: -1}"
case "$query" in
	*"INSERT INTO ops_event"*) printf '%s\n' "$query" >> "$WATER_SQL_LOG" ;;
esac
EOF
chmod +x "$TMP/a/bin/psql"
printf 'FIN_PG_DSN=fake\n' > "$TMP/a/env"
: > "$TMP/a/env-empty"
export WATER_SQL_LOG="$TMP/a/sql.log" DM_ENV_FILE="$TMP/a/env"
export PATH="$TMP/a/bin:$PATH"

: > "$WATER_SQL_LOG"
bash "$META/dispatch_water.sh" T-9601
[[ $(grep -c 'INSERT INTO ops_event' "$WATER_SQL_LOG") -eq 1 ]]
grep -Fq "SELECT 'CTO','候水','T-9601','候水——spawn rc75 水位/额度满，候选留队下拍重试（非断链；额度恢复即拉）'" "$WATER_SQL_LOG"
# 幂等闸谓词钉——最新事件 IS DISTINCT FROM '候水'（DB 侧判重，桩面钉 SQL 形）
grep -Fq "IS DISTINCT FROM '候水'" "$WATER_SQL_LOG"
grep -Fq "ORDER BY e.ts DESC,e.id DESC LIMIT 1" "$WATER_SQL_LOG"

# FIN_DSN 缺=静默退零查询（拍壳同律）
: > "$WATER_SQL_LOG"
RC=0; DM_ENV_FILE="$TMP/a/env-empty" bash "$META/dispatch_water.sh" T-9601 || RC=$?
[[ $RC -eq 0 && ! -s "$WATER_SQL_LOG" ]]

# 票号畸形=拒写 exit 2（SQL 注入面字符硬界——dm_handoff 同款字符域）
RC=0; bash "$META/dispatch_water.sh" "T-9601');DROP" 2>/dev/null || RC=$?
[[ $RC -eq 2 && ! -s "$WATER_SQL_LOG" ]]

# ══ B. dm_depgate 即派 rc75 写点 ════════════════════════════════════
B="$TMP/b"; mkdir -p "$B/bin" "$B/wt"
REPO="$B/repo"; mkdir -p "$REPO/docs/tasks"
cd "$REPO" && git init -q && git config user.name test && git config user.email t@e.c
printf 'base\n' > check.txt && git add . && git commit -qm base && git branch -M master
printf '# T-9601 x\n## 范围\n1. docs/** 纯文档面。\n' > "$REPO/docs/tasks/T-9601-candidate.md"
printf '# T-9602 x\n## 范围\n1. docs/** 纯文档面。\n' > "$REPO/docs/tasks/T-9602-candidate.md"

cat > "$B/bin/dm-queue" <<'EOF'
#!/usr/bin/env bash
case "${1:-all}" in review) : ;; approved) cat "$APPROVED_LIST" ;; *) exit 2 ;; esac
EOF
cat > "$B/bin/dispatch" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$DISPATCH_LOG"
exit "${DISPATCH_RC:-0}"
EOF
cat > "$B/bin/water" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$WATER_LOG"
EOF
cat > "$B/bin/esc" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$ESC_LOG"
EOF
cat > "$B/bin/psql" <<'EOF'
#!/usr/bin/env bash
sqlf=
while (($#)); do
	if [ "$1" = -f ]; then sqlf=$2; shift 2; else shift; fi
done
[ -n "$sqlf" ] || exit 0
cat "$sqlf" >> "$EVENT_SQL_LOG"
if grep -q "INSERT INTO ops_event" "$sqlf"; then
	grep -oE 'T-[0-9]{4}' "$sqlf" | head -1
fi
EOF
chmod +x "$B/bin/"*
printf 'FIN_PG_DSN=fake\n' > "$B/env"
printf 'T-9601\nT-9602\n' > "$B/approved.list"
: > "$B/review.list"
export PATH="$B/bin:$PATH" DM_ENV_FILE="$B/env" \
	O_DM_QUEUE_SH="$B/bin/dm-queue" O_WORKER_DISPATCH_SCRIPT="$B/bin/dispatch" \
	O_DISPATCH_WATER_SH="$B/bin/water" O_DEPGATE_ESC_SH="$B/bin/esc" \
	O_DEPGATE_REPO="$REPO" O_DEPGATE_TASK_DIR="$REPO/docs/tasks" \
	O_DEPGATE_WT_ROOT="$B/wt" O_DEPGATE_MASTER_REF=master \
	APPROVED_LIST="$B/approved.list" REVIEW_LIST="$B/review.list" \
	DISPATCH_LOG="$B/dispatch.log" WATER_LOG="$B/water.log" \
	EVENT_SQL_LOG="$B/event.sql.log" ESC_LOG="$B/esc.log"
: > "$DISPATCH_LOG"; : > "$WATER_LOG"; : > "$EVENT_SQL_LOG"; : > "$ESC_LOG"

# B① rc75：首候选候水落标+break——次候选不烧闸不候水
DISPATCH_RC=75 bash "$META/dm_depgate.sh" >"$B/out75.log" 2>&1
grep -q '^T-9601 ' "$DISPATCH_LOG"
! grep -q 'T-9602' "$DISPATCH_LOG"
grep -qx 'T-9601' "$WATER_LOG"
! grep -q 'T-9602' "$WATER_LOG"
grep -q "ticket='T-9601'" "$EVENT_SQL_LOG"   # 派单行已落（候水前置账）
grep -q 'water/quota full' "$B/out75.log"

# B② rc1：非水位败=升件径——候水 emit 零调
: > "$DISPATCH_LOG"; : > "$WATER_LOG"; : > "$EVENT_SQL_LOG"; : > "$ESC_LOG"
DISPATCH_RC=1 bash "$META/dm_depgate.sh" >"$B/out1.log" 2>&1
[[ ! -s "$WATER_LOG" ]]
grep -q 'DEPGATE-DISPATCH-FAIL-T-9601' "$ESC_LOG"

# B③ rc0：派成=零候水
: > "$DISPATCH_LOG"; : > "$WATER_LOG"
DISPATCH_RC=0 bash "$META/dm_depgate.sh" >"$B/out0.log" 2>&1
[[ ! -s "$WATER_LOG" ]]

# ══ C. mech_tick 补拉窗 rc75 写点（GW-001 段A 迁轨——原 dm_tick 尾段） ══
C="$TMP/c"; mkdir -p "$C/bin" "$C/tasks"
touch "$C/tasks/T-9601-resume.md" "$C/tasks/T-9602-resume.md"
cat > "$C/bin/psql" <<'EOF'
#!/usr/bin/env bash
query="${*: -1}"
case "$query" in
	"-qAt") echo "${DM_LOCK_RESULT:-t}" ;; # 班次锁 holder 桩
	*"AND event='派单'"*"LIMIT 1"*) echo swe-2-medium ;; # 模型档恢复票级查询（段B1 收编后形态）
	*"INSERT INTO ops_event"*) printf '%s\n' "$query" >> "$EVENT_SQL_LOG" ;;
	*"INSERT INTO ops_headless"*) : ;;
esac
EOF
# dm_queue 桩（段B1：补拉候选集=dm_queue dispatched 单源）
cat > "$C/bin/dm-queue" <<'EOF'
#!/usr/bin/env bash
[ "${1:-}" = dispatched ] && printf 'T-9601\nT-9602\n'
EOF
cat > "$C/bin/opsevc" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
cat > "$C/bin/systemctl" <<'EOF'
#!/usr/bin/env bash
[ "${2:-}" = is-active ] && { echo inactive; exit 3; }
exit 0
EOF
cat > "$C/bin/dispatch" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$DISPATCH_LOG"
exit "${DISPATCH_RC:-0}"
EOF
cat > "$C/bin/water" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$WATER_LOG"
EOF
cat > "$C/bin/go" <<'EOF'
#!/usr/bin/env bash
exit 1 # ticketdraft 物料化径桩败——dm_esc 尽力而为面不挡本测断言
EOF
chmod +x "$C/bin/"*
printf 'FIN_PG_DSN=fake\n' > "$C/env"
export DM_ENV_FILE="$C/env" PATH="$C/bin:$PATH" \
	O_WORKER_DISPATCH_SCRIPT="$C/bin/dispatch" \
	O_DISPATCH_WATER_SH="$C/bin/water" \
	O_WORKER_TASK_DIR="$C/tasks" O_DECISION_DEVIN_BIN=/bin/true \
	O_DM_DEPGATE_SH=/bin/true O_DM_SUSPEND_SH=/bin/true O_DM_HANDOFF_SH=/bin/true \
	O_DM_LEDGER_SH=/bin/true O_VERDICT_DELIVER_SH=/bin/true \
	O_AUTOAPPROVE_SH=/bin/true O_DEPLOY_WINDOW_SH=/bin/true O_OPS_ESC_SH="$C/bin/opsevc" \
	O_DM_QUEUE_SH="$C/bin/dm-queue" O_MANUAL_LANE_SH=/bin/false \
	DISPATCH_LOG="$C/dispatch.log" WATER_LOG="$C/water.log" EVENT_SQL_LOG="$C/event.sql.log" \
	DM_LOCK_RESULT=t
: > "$DISPATCH_LOG"; : > "$WATER_LOG"; : > "$EVENT_SQL_LOG"

# C① rc75：首候选候水+break——次候选不烧闸；拉起行零落
DISPATCH_RC=75 bash "$META/mech_tick.sh" >/dev/null 2>&1
grep -q '^T-9601 swe-2-medium ' "$DISPATCH_LOG"
! grep -q 'T-9602' "$DISPATCH_LOG"
grep -qx 'T-9601' "$WATER_LOG"
! grep -q 'T-9602' "$WATER_LOG"
! grep -q "'拉起','T-9601'" "$EVENT_SQL_LOG"

# C② rc0：补拉成=拉起行落账，候水零调
: > "$DISPATCH_LOG"; : > "$WATER_LOG"; : > "$EVENT_SQL_LOG"
DISPATCH_RC=0 bash "$META/mech_tick.sh" >/dev/null 2>&1
[[ ! -s "$WATER_LOG" ]]
grep -q "'拉起','T-9601'" "$EVENT_SQL_LOG"
grep -q "'拉起','T-9602'" "$EVENT_SQL_LOG"

# C③ rc1：非水位败=报错径续试——候水零调
: > "$DISPATCH_LOG"; : > "$WATER_LOG"; : > "$EVENT_SQL_LOG"
DISPATCH_RC=1 bash "$META/mech_tick.sh" >/dev/null 2>&1
[[ ! -s "$WATER_LOG" ]]
grep -q 'T-9602' "$DISPATCH_LOG" # rc1 不 break——次候选照试

# ══ D. 哨兵免报——派单接行集含「候水」 ═══════════════════════════════
D="$TMP/d"; mkdir -p "$D"
cat > "$D/psql" <<'EOF'
#!/usr/bin/env bash
query="${*: -1}"
case "$query" in
	*"NOT EXISTS (SELECT 1 FROM ops_event r"*)
		printf '%s\n' "$query" >> "$SCAN_SQL_LOG" ;;
esac
exit 0
EOF
chmod +x "$D/psql"
printf 'FIN_PG_DSN=fake\n' > "$D/env"
: > "$D/scan.sql.log"
DM_ENV_FILE="$D/env" O_HANDOFF_PAIRS_TSV="$META/handoff_pairs.tsv" \
	O_HANDOFF_ESCALATE_SCRIPT=/bin/true SCAN_SQL_LOG="$D/scan.sql.log" \
	PATH="$D:$PATH" bash "$META/dm_handoff.sh"
# 派单接行集钉词——「候水」入集=派单→候水即「已被接」真死只报三词全无件
grep -Fq "r.event IN ('领单','拉起','候水','驳回','挂起','作废','完成','处置')" "$D/scan.sql.log"
# 候水行自身不配对接——全表交event 谱无「候水」交行扫描
! grep -Fq "e.event='候水'" "$D/scan.sql.log"

# ══ E. 老龄鸣——dispatch_aged_water 量表实跑 ════════════════════════
E="$TMP/e"; mkdir -p "$E"
grep -P '^dispatch_aged_water\t' "$META/gauges.tsv" > "$E/gauges.tsv"
[ -s "$E/gauges.tsv" ] || { echo "gauges.tsv dispatch_aged_water 行缺席" >&2; exit 1; }
cat > "$E/psql" <<'EOF'
#!/usr/bin/env bash
query="${*: -1}"
case "$query" in
	*"INSERT INTO ops_event"*) printf '%s\n' "$query" >> "$GAUGE_INSERTS" ;;
	*"string_agg(ticket"*) cat "$GAUGE_IDS" ;;
	*"event='候水'"*) cat "$GAUGE_COUNT" ;;
	*"INSERT INTO ops_headless"*) : ;;
esac
EOF
chmod +x "$E/psql"
printf 'FIN_PG_DSN=fake-fin\nFIN_PG_RO_DSN=fake-ro\n' > "$E/env"
export GAUGE_INSERTS="$E/inserts" GAUGE_COUNT="$E/count" GAUGE_IDS="$E/ids"
: > "$GAUGE_INSERTS"
run_gauge() {
	( env PATH="$E:$PATH" GAUGES_ENV_FILE="$E/env" GAUGES_TSV="$E/gauges.tsv" \
		O_GAUGES_ST="$E/state" bash "$META/gauges_tick.sh" )
}
mkdir -p "$E/state"

# E① count=0 阈内静默
printf '0\n' > "$GAUGE_COUNT"; printf 'T-9601\n' > "$GAUGE_IDS"
run_gauge
[[ ! -s "$GAUGE_INSERTS" ]]

# E② count=2 阈上首拍即鸣（连鸣律跳滞回/日频）+{d} 携老化票清单
printf '2\n' > "$GAUGE_COUNT"; printf 'T-9601,T-9602\n' > "$GAUGE_IDS"
run_gauge
[[ $(wc -l < "$GAUGE_INSERTS") -eq 1 ]]
grep -Fq "'巡检闸','升件','dispatch_aged_water'" "$GAUGE_INSERTS"
grep -Fq '派单候水老龄 count=2' "$GAUGE_INSERTS"
grep -Fq '（T-9601,T-9602）' "$GAUGE_INSERTS"
grep -Fq "'淤积','dispatch_aged_water'" "$GAUGE_INSERTS"

# E③ 次拍续鸣（连鸣=阈上每拍必落行——冻结必须持续发声）
run_gauge
[[ $(wc -l < "$GAUGE_INSERTS") -eq 2 ]]

echo "dispatch water state scenarios PASS"
