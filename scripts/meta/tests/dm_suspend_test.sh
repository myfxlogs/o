#!/usr/bin/env bash
# dm_suspend_test.sh —— T-0365 挂起黑洞看门狗消费窗桩测：psql/go/
# dispatch-worker/escalate 全桩，场景驱动验机械二择（绿窗复跑绿→
# 续跑派工 note「挂起续跑」；复跑红/非绿窗/无 await 未消→龄>4h 升件；
# 龄<4h 未消→静默等下拍；rate-limit:auto 排除谓词钉）。
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin" "$TMP/repo"

cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
query="${*: -1}"
case "$query" in
	# 挂起扫描窗——谓词钉：消化词表含升件/开工、rate-limit:auto 排除
	*"DISTINCT ON (ticket)"*)
		printf '%s\n' "$query" >> "$SCAN_SQL_LOG"
		grep -Fq "'升件'" <<<"$query"
		grep -Fq "'开工'" <<<"$query"
		grep -Fq "rate-limit:auto" <<<"$query"
		grep -Fq "event='挂起'" <<<"$query"
		[ "${DM_SUSPEND_SCENARIO:-empty}" = empty ] || printf '%b\n' "$SUSPEND_ROWS"
		;;
	# 模型档恢复查询（原派单备注 swe-2-medium 词）
	*"e.event='派单'"*) printf 'swe-2-medium\n' ;;
	*"INSERT INTO ops_event"*) printf '%s\n' "$query" >> "$SUSPEND_SQL_LOG" ;;
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
chmod +x "$TMP/bin/"*

printf 'FIN_PG_DSN=fake\n' > "$TMP/env"
export PATH="$TMP/bin:$PATH"
export DM_ENV_FILE="$TMP/env"
export ESCALATE_ENV_FILE="$TMP/env"
export O_SUSPEND_ESCALATE_SCRIPT="$ROOT/scripts/meta/escalate.sh"
export O_SUSPEND_REPO_DIR="$TMP/repo"
export O_WORKER_DISPATCH_SCRIPT="$TMP/bin/dispatch-worker"
export SCAN_SQL_LOG="$TMP/scan.sql.log"
export SUSPEND_SQL_LOG="$TMP/suspend.sql.log"
export DISPATCH_LOG="$TMP/dispatch.log"
export GO_LOG="$TMP/go.log"
export DM_SUSPEND_SCENARIO=empty
export GO_TEST_RC=0 DISPATCH_RC=0

DM_SH="$ROOT/scripts/meta/dm_suspend.sh"
reset_logs() { : > "$SUSPEND_SQL_LOG"; : > "$DISPATCH_LOG"; : > "$GO_LOG"; }
run() { DM_SUSPEND_SCENARIO=$1 SUSPEND_ROWS=$2 bash "$DM_SH"; }

# ① 空集=静默退（无未消化挂起行）
reset_logs
run empty ''
[[ ! -s "$DISPATCH_LOG" && ! -s "$SUSPEND_SQL_LOG" && ! -s "$GO_LOG" ]]

# ② 绿窗档复跑绿→续跑派工（note 标「挂起续跑」；模型档从派单备注恢复）
reset_logs
run resume-green 'T-9001\x1fflake窗 await=绿窗:TestFoo 上轮红\x1f施工席\x1f100\n'
grep -q -- '-count=1 -run \^TestFoo\$ \./\.\.\.$' "$GO_LOG"
grep -q '^T-9001 swe-2-medium ' "$DISPATCH_LOG"
grep -qF "'拉起','T-9001','挂起续跑" "$SUSPEND_SQL_LOG"

# ③ 绿窗档带包名→go test 收敛到包（不跑 ./... 全谱）
reset_logs
run resume-pkg 'T-9002\x1fflake await=绿窗:internal/research/dcd:TestBar\x1f施工席\x1f100\n'
grep -q 'internal/research/dcd' "$GO_LOG"
grep -q '\-run \^TestBar\$' "$GO_LOG"
! grep -q '\./\.\.\.' "$GO_LOG"

# ④ 绿窗复跑红+龄<4h→静默等下拍（零派单零升件）
reset_logs
export GO_TEST_RC=1
run wait-red-young 'T-9003\x1fflake await=绿窗:TestFoo\x1f施工席\x1f100\n'
[[ ! -s "$DISPATCH_LOG" && ! -s "$SUSPEND_SQL_LOG" ]]
export GO_TEST_RC=0

# ⑤ 绿窗复跑红+龄>4h→升件呈业主（无类升件永不入判例面）
reset_logs
export GO_TEST_RC=1
run escalate-old-red 'T-9004\x1fflake await=绿窗:TestFoo\x1f施工席\x1f20000\n'
grep -qF "'升件','T-9004'" "$SUSPEND_SQL_LOG"
grep -q '挂起超时未消' "$SUSPEND_SQL_LOG"
[[ ! -s "$DISPATCH_LOG" ]]
export GO_TEST_RC=0

# ⑥ 非绿窗档（无 await）+龄>4h→升件；龄<4h→静默
reset_logs
run escalate-old-plain 'T-9005\x1f外因等待凭据补齐\x1f施工席\x1f30000\n'
grep -qF "'升件','T-9005'" "$SUSPEND_SQL_LOG"
reset_logs
run wait-plain-young 'T-9006\x1f外因等待\x1f施工席\x1f100\n'
[[ ! -s "$SUSPEND_SQL_LOG" && ! -s "$DISPATCH_LOG" ]]

# ⑦ 混合行集：绿窗绿续跑+无await老挂起升件同拍各一
reset_logs
run mixed 'T-9007\x1fawait=绿窗:TestFoo\x1f施工席\x1f100\nT-9008\x1f裁量待定\x1f决策席\x1f20000\n'
grep -q '^T-9007 swe-2-medium ' "$DISPATCH_LOG"
grep -qF "'升件','T-9008'" "$SUSPEND_SQL_LOG"

# ⑧ await=绿窗 缺参（畸形）+龄>4h→不可机械判走升件兜底
reset_logs
run malformed 'T-9009\x1fawait=绿窗\x1f施工席\x1f20000\n'
grep -qF "'升件','T-9009'" "$SUSPEND_SQL_LOG"
[[ ! -s "$DISPATCH_LOG" ]]

echo "dm suspend watchdog scenarios PASS"
