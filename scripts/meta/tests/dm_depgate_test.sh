#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin" "$TMP/wt"

REPO="$TMP/repo"
mkdir -p "$REPO/docs/tasks" "$REPO/scripts" "$REPO/internal/api" "$REPO/web/src/react"
cd "$REPO"
git init -q
git config user.name test
git config user.email test@example.com
printf 'base\n' > scripts/check_all.sh
printf 'base\n' > internal/api/quotes_watch_face.go
printf 'base\n' > web/src/react/App.tsx
git add .
git commit -qm base
git branch -M master

mk_ticket() {
	cat >"$REPO/docs/tasks/$1-$2.md"
}

# 在审票 T-9001 改 scripts/check_all.sh；T-9007 改 internal/api/quotes_watch_face.go。
git checkout -qb t/9001-review
printf 'review\n' > scripts/check_all.sh
git commit -qam review-checkall
git checkout -q master
git checkout -qb t/9007-review
printf 'review\n' > internal/api/quotes_watch_face.go
git commit -qam review-quotes-face
git checkout -q master

mk_ticket T-9001 review <<'EOF'
# T-9001 review
## 背景
收口待审。
EOF
mk_ticket T-9007 review <<'EOF'
# T-9007 review
## 背景
收口待审。
EOF
mk_ticket T-9002 candidate <<'EOF'
# T-9002 candidate
## 范围
1. web/src/react/ 呈现层。
EOF
mk_ticket T-9003 candidate <<'EOF'
# T-9003 candidate
## 范围
1. scripts/check_all.sh 同面相撞。
EOF
mk_ticket T-9004 candidate <<'EOF'
# T-9004 candidate
## 背景
缺范围节——不可判排队。
EOF
mk_ticket T-9005 candidate <<'EOF'
# T-9005 candidate
## 范围
1. 只写语义面不写文件清单。
EOF
mk_ticket T-9006 candidate <<'EOF'
# T-9006 candidate
## 范围
1. quotes_watch_face.go 裸文件名也须按 basename 对拍。
EOF
mk_ticket T-9008 candidate <<'EOF'
# T-9008 candidate
## 范围
1. docs/** 纯文档面。
EOF
mk_ticket T-9010 candidate <<'EOF'
# T-9010 candidate
## 范围
1. web/src/react/ 呈现层。
EOF
mk_ticket T-9011 candidate <<'EOF'
# T-9011 candidate
## 范围
1. web/src/react/ 呈现层。
EOF
mk_ticket T-9012 candidate <<'EOF'
# T-9012 candidate
## 范围
1. web/src/react/ 呈现层。
EOF
mk_ticket T-9013 candidate <<'EOF'
# T-9013 candidate
## 范围外
1. scripts/check_all.sh 只写禁区不算范围清单。
EOF
mk_ticket T-9014 candidate <<'EOF'
# T-9014 candidate

- 执行=人工径（业主在场）——真钱 venue 写票出生行（坑条目律①）
## 范围
1. web/src/react/ 呈现层。
EOF

cat > "$TMP/bin/dm-queue" <<'EOF'
#!/usr/bin/env bash
case "${1:-all}" in
	review) cat "$REVIEW_LIST" ;;
	approved) cat "$APPROVED_LIST" ;;
	all) cat "$REVIEW_LIST" "$APPROVED_LIST" ;;
	*) exit 2 ;;
esac
EOF
cat > "$TMP/bin/dispatch" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$DISPATCH_LOG"
exit "${DISPATCH_RC:-0}"
EOF
cat > "$TMP/bin/esc" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$ESC_LOG"
EOF
cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
sqlf=
while (($#)); do
	if [ "$1" = -f ]; then sqlf=$2; shift 2; else shift; fi
done
[ -n "$sqlf" ] || exit 0
cat "$sqlf" >> "$EVENT_SQL_LOG"
if grep -q "INSERT INTO ops_event" "$sqlf" && [ "${INSERT_DISPATCH:-1}" = 1 ]; then
	grep -oE 'T-[0-9]{4}' "$sqlf" | head -1
fi
EOF
chmod +x "$TMP/bin/"*

printf 'FIN_PG_DSN=fake\n' > "$TMP/env"
export PATH="$TMP/bin:$PATH"
export DM_ENV_FILE="$TMP/env"
export O_DM_QUEUE_SH="$TMP/bin/dm-queue"
export O_WORKER_DISPATCH_SCRIPT="$TMP/bin/dispatch"
export O_DEPGATE_ESC_SH="$TMP/bin/esc"
export O_DEPGATE_REPO="$REPO"
export O_DEPGATE_TASK_DIR="$REPO/docs/tasks"
export O_DEPGATE_WT_ROOT="$TMP/wt"
export O_DEPGATE_MASTER_REF=master
export REVIEW_LIST="$TMP/review.list"
export APPROVED_LIST="$TMP/approved.list"
export DISPATCH_LOG="$TMP/dispatch.log"
export EVENT_SQL_LOG="$TMP/event.sql.log"
export ESC_LOG="$TMP/esc.log"

printf 'T-9001\nT-9007\n' > "$REVIEW_LIST"
printf 'T-9002\nT-9003\nT-9004\nT-9005\nT-9006\nT-9008\nT-9013\nT-9014\n' > "$APPROVED_LIST"
: > "$DISPATCH_LOG"; : > "$EVENT_SQL_LOG"; : > "$ESC_LOG"
bash "$ROOT/scripts/meta/dm_depgate.sh" >"$TMP/out.log" 2>&1

grep -q '^T-9002 swe-2-flash-high ' "$DISPATCH_LOG"
grep -q '^T-9008 swe-2-flash-high ' "$DISPATCH_LOG"
! grep -q '^T-9003 ' "$DISPATCH_LOG"
! grep -q '^T-9004 ' "$DISPATCH_LOG"
! grep -q '^T-9005 ' "$DISPATCH_LOG"
! grep -q '^T-9006 ' "$DISPATCH_LOG"
! grep -q '^T-9013 ' "$DISPATCH_LOG"
! grep -q '^T-9014 ' "$DISPATCH_LOG" # T-0438 件⑥：人工径票不派不挂起不计漏拉
grep -q "ticket='T-9002'" "$EVENT_SQL_LOG"
grep -q "ticket='T-9008'" "$EVENT_SQL_LOG"
! grep -q "ticket='T-9003'" "$EVENT_SQL_LOG"
! grep -q "ticket='T-9013'" "$EVENT_SQL_LOG"
! grep -q "ticket='T-9014'" "$EVENT_SQL_LOG" # 人工径票零派单行零升件行
grep -q 'DEPGATE-SCOPE-EMPTY-T-9005' "$ESC_LOG"
grep -q '范围∩在审diff非空' "$TMP/out.log"
grep -q '缺文件清单' "$TMP/out.log"
grep -q '人工径票' "$TMP/out.log"

# 水位/额度满：派单行已落但 dispatch rc=75——留常径补拉，后续候选不继续烧闸。
printf 'T-9010\nT-9011\n' > "$APPROVED_LIST"
: > "$DISPATCH_LOG"; : > "$EVENT_SQL_LOG"; : > "$ESC_LOG"
DISPATCH_RC=75 bash "$ROOT/scripts/meta/dm_depgate.sh" >"$TMP/out75.log" 2>&1
grep -q '^T-9010 swe-2-flash-high ' "$DISPATCH_LOG"
! grep -q '^T-9011 ' "$DISPATCH_LOG"
grep -q "ticket='T-9010'" "$EVENT_SQL_LOG"
! grep -q "ticket='T-9011'" "$EVENT_SQL_LOG"
grep -q 'water/quota full' "$TMP/out75.log"
unset DISPATCH_RC

# 在审票面不可判：任何候选都不得快派，且落升件。
printf 'T-9099\n' > "$REVIEW_LIST"
printf 'T-9012\n' > "$APPROVED_LIST"
: > "$DISPATCH_LOG"; : > "$EVENT_SQL_LOG"; : > "$ESC_LOG"
bash "$ROOT/scripts/meta/dm_depgate.sh" >"$TMP/out-unknown.log" 2>&1
[[ ! -s "$DISPATCH_LOG" ]]
[[ ! -s "$EVENT_SQL_LOG" ]]
grep -q 'DEPGATE-REVIEW-UNKNOWN-T-9099' "$ESC_LOG"
grep -q '保守陪绑' "$TMP/out-unknown.log"

# GW-001 段C3 派单序：即派面同过 dispatch_order——postmortem 族候选先派
# （输入序在后者跳前；psql 桩 -c 形空答=已批ts 0，rank 判序）。
mk_ticket T-9015 ord-blind <<'EOF'
# T-9015 观测盲桩票
## 范围
1. web/src/react/ 呈现层。

病灶签名：`观测盲:face:z`（桩）
EOF
mk_ticket T-9016 ord-pm <<'EOF'
# T-9016 postmortem 桩票
## 范围
1. web/src/react/ 呈现层。

病灶签名：`postmortem:guard_loop`（桩）
EOF
printf 'T-9001\n' > "$REVIEW_LIST" # 可判在审面（scripts/check_all.sh——候选零交集）
printf 'T-9015\nT-9016\n' > "$APPROVED_LIST"
: > "$DISPATCH_LOG"; : > "$EVENT_SQL_LOG"; : > "$ESC_LOG"
bash "$ROOT/scripts/meta/dm_depgate.sh" >"$TMP/out-order.log" 2>&1
head -1 "$DISPATCH_LOG" | grep -q '^T-9016 swe-2-flash-high '
grep -q '^T-9015 swe-2-flash-high ' "$DISPATCH_LOG"

echo "dm depgate path-overlap tests PASS"
