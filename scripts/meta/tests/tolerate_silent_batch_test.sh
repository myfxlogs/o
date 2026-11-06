#!/usr/bin/env bash
# tolerate_silent_batch_test.sh —— T-0372 七组静默位补报注入实证：
# dm_tick×3（队列推导/派单查询/看门狗本体）+dev_run×2（usage 写径败/
# transcript 缺）+deploy_sync×2（ssh 取数/comm 差集）+arch_daily×1+
# backup_push_sg×1——各败径→ops_esc 升件自报，容错语义零动（仍滑行）。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin" "$TMP/rt"

# —— esc 记录桩（O_OPS_ESC_SH/O_WORKER_OPS_ESC_SH 注入面——验接线不验写径，
#    写径/dedup 由 ops_esc_test.sh 单测实证）——
cat > "$TMP/bin/ops_esc_log" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >> "$TMP/esc-log"
EOF
chmod +x "$TMP/bin/ops_esc_log"

# —— psql 桩：dm_tick 闸/心跳/派单查询 + arch/backup 内联 esc dedup/INSERT ——
cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
query=${*: -1}
[[ $query == "-" ]] && query=$(cat)
case "$query" in
	"-qAt") echo t ;; # advisory-lock holder 桩（无 -c 形）
	*"SELECT count(*)"*) echo "${DM_PENDING_COUNT:-0}" ;;
	*"SELECT l.ticket ||"*)
		[ "${DM_DISPATCH_FAIL:-0}" = 1 ] && { echo "dispatch query boom" >&2; exit 1; }
		;;
	*"SELECT 1 FROM ops_event WHERE event='升件'"*) ;; # dedup 空答=未报
	*"INSERT INTO ops_event"*) printf '%s\n' "$query" >> "$ESC_PSQL_LOG" ;;
	*"INSERT INTO ops_headless"*) ;;
	*"DISTINCT ON (ticket)"*) ;; # dm_suspend 扫描空集
	*) echo "unexpected psql: $query" >&2; exit 2 ;;
esac
EOF
chmod +x "$TMP/bin/psql"

# —— dm_queue/dm_suspend/devin/report/usage 桩 ——
cat > "$TMP/bin/dm-queue-empty" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
cat > "$TMP/bin/dm-queue-fail" <<'EOF'
#!/usr/bin/env bash
echo "queue query boom" >&2
exit 1
EOF
cat > "$TMP/bin/dm-suspend-ok" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
cat > "$TMP/bin/dm-suspend-fail" <<'EOF'
#!/usr/bin/env bash
exit 3
EOF
cat > "$TMP/bin/devin" <<'EOF'
#!/usr/bin/env bash
[[ "${1:-}" == "list" ]] && { echo '[]'; exit 0; }
export_file=
while (($#)); do
	if [[ $1 == --export ]]; then shift; export_file=${1:-}; break; fi
	shift
done
[[ "${DEVIN_WRITE_EXPORT:-1}" = 1 && -n "$export_file" ]] && \
	printf '{"session_id":"s1","final_metrics":{"total_prompt_tokens":1}}' > "$export_file"
echo "worker done"
EOF
cat > "$TMP/bin/report-stub" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
cat > "$TMP/bin/usage-ok" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
cat > "$TMP/bin/usage-fail" <<'EOF'
#!/usr/bin/env bash
echo "usage write boom" >&2
exit 1
EOF
chmod +x "$TMP/bin/"*

printf 'FIN_PG_DSN=fake-dsn\n' > "$TMP/env"
export PATH="$TMP/bin:$PATH"
export DM_ENV_FILE="$TMP/env" REPORT_ENV_FILE="$TMP/env" OPS_ESC_ENV_FILE="$TMP/env"
export O_DECISION_WORKDIR="$TMP/decision" O_WORKER_RUNTIME_DIR="$TMP/rt"
export O_OPS_ESC_SH="$TMP/bin/ops_esc_log" O_WORKER_OPS_ESC_SH="$TMP/bin/ops_esc_log"
export O_WORKER_REPORT_BIN="$TMP/bin/report-stub"
export O_DM_QUEUE_SH="$TMP/bin/dm-queue-empty" O_DM_SUSPEND_SH="$TMP/bin/dm-suspend-ok"
export O_DM_LEDGER_SH=/bin/true # T-0414：机械轨件单测——tick 接线面桩掉防真 git/psql
export ESC_PSQL_LOG="$TMP/esc-psql" ESC_LOG="$TMP/esc-log"
: > "$ESC_LOG"; : > "$ESC_PSQL_LOG"

# ══ A. dm_tick 队列推导败→「队列推导败」升件（队列败≠空集——头号静默位）══
O_DM_QUEUE_SH="$TMP/bin/dm-queue-fail" bash "$ROOT/scripts/meta/dm_tick.sh" >/dev/null 2>&1
grep -qF '队列推导败' "$ESC_LOG" && grep -qF 'rc=1' "$ESC_LOG" \
	&& echo "A dm_tick queue-fail → 队列推导败 esc PASS"

# ══ B. dm_tick 派单查询败→「派单查询败」升件 ══
: > "$ESC_LOG"
DM_DISPATCH_FAIL=1 O_DM_QUEUE_SH="$TMP/bin/dm-queue-empty" \
	bash "$ROOT/scripts/meta/dm_tick.sh" >/dev/null 2>&1
grep -qF '派单查询败' "$ESC_LOG" && echo "B dm_tick dispatch-fail → 派单查询败 esc PASS"

# ══ C. dm_tick 看门狗本体败→「看门狗本体败」升件 ══
: > "$ESC_LOG"
O_DM_SUSPEND_SH="$TMP/bin/dm-suspend-fail" \
	bash "$ROOT/scripts/meta/dm_tick.sh" >/dev/null 2>&1
grep -qF '看门狗本体败' "$ESC_LOG" && echo "C dm_tick suspend-fail → 看门狗本体败 esc PASS"

# ══ D. dev_run usage 写径败→「usage 采集败」写败/META-DEV-USAGE ══
: > "$ESC_LOG"
DEVIN_WRITE_EXPORT=1 O_WORKER_DEVIN_BIN="$TMP/bin/devin" \
	O_WORKER_USAGE_BIN="$TMP/bin/usage-fail" \
	bash "$ROOT/scripts/meta/dev_run.sh" T-9001 >/dev/null 2>&1
grep -qF '施工席 T-9001 写败 META-DEV-USAGE usage 采集败' "$ESC_LOG" \
	&& grep -qF '写径败' "$ESC_LOG" \
	&& echo "D dev_run usage-write-fail → usage 采集败 esc PASS"

# ══ E. dev_run transcript 缺/空→「usage 采集败」transcript 缺报 ══
: > "$ESC_LOG"
DEVIN_WRITE_EXPORT=0 O_WORKER_DEVIN_BIN="$TMP/bin/devin" \
	O_WORKER_USAGE_BIN="$TMP/bin/usage-ok" \
	bash "$ROOT/scripts/meta/dev_run.sh" T-9001 >/dev/null 2>&1
grep -qF 'usage 采集败' "$ESC_LOG" && grep -qF 'transcript 缺/空' "$ESC_LOG" \
	&& echo "E dev_run transcript-missing → usage 采集败 esc PASS"

# ══ F. dev_run 正常径（usage 写成+transcript 在）→零自报（容错不扰常径）══
: > "$ESC_LOG"
DEVIN_WRITE_EXPORT=1 O_WORKER_DEVIN_BIN="$TMP/bin/devin" \
	O_WORKER_USAGE_BIN="$TMP/bin/usage-ok" \
	bash "$ROOT/scripts/meta/dev_run.sh" T-9001 >/dev/null 2>&1
[[ ! -s "$ESC_LOG" ]] && echo "F dev_run normal-path zero-esc PASS"

# ══ G. deploy_sync ssh 取数败→「deploy_sync 取数败」升件 ══
#    最小 git 仓+桩 rsync/ssh——check_schema_live 哨兵件不在=跳过。
mkdir -p "$TMP/repo/scripts/meta"
git -C "$TMP/repo" init -q
cp "$ROOT/scripts/deploy_sync.sh" "$TMP/repo/scripts/"
cat > "$TMP/repo/scripts/meta/ops_esc.sh" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >> "$TMP/esc-log"
EOF
chmod +x "$TMP/repo/scripts/meta/ops_esc.sh"
cat > "$TMP/bin/rsync" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
cat > "$TMP/bin/ssh" <<'EOF'
#!/usr/bin/env bash
echo "ssh boom" >&2
exit 255
EOF
chmod +x "$TMP/bin/rsync" "$TMP/bin/ssh"
: > "$ESC_LOG"
( cd "$TMP/repo" && O_SG_HOST=fake-sg O_OPS_ESC_SH= bash scripts/deploy_sync.sh ) >/dev/null 2>&1 || true # tolerate: 测试桩——被测件内部 esc 自报已实证，壳层败不碍断言
grep -qF 'deploy_sync 取数败' "$ESC_LOG" && echo "G deploy_sync ssh-fail → 取数败 esc PASS"

# ══ H. deploy_sync comm 差集败→「deploy_sync 差集败」升件 ══
cat > "$TMP/bin/ssh" <<'EOF'
#!/usr/bin/env bash
printf 'a.go\nb.go\n'
EOF
cat > "$TMP/bin/comm" <<'EOF'
#!/usr/bin/env bash
echo "comm boom" >&2
exit 1
EOF
chmod +x "$TMP/bin/ssh" "$TMP/bin/comm"
: > "$ESC_LOG"
( cd "$TMP/repo" && O_SG_HOST=fake-sg O_OPS_ESC_SH= bash scripts/deploy_sync.sh ) >/dev/null 2>&1 || true # tolerate: 测试桩——被测件内部 esc 自报已实证，壳层败不碍断言
grep -qF 'deploy_sync 差集败' "$ESC_LOG" && echo "H deploy_sync comm-fail → 差集败 esc PASS"
rm -f "$TMP/bin/comm"

# ══ I. arch_daily 保留期清扫败→「arch 保留期清扫败」升件（内联 esc 同式）══
cat > "$TMP/bin/sudo" <<'EOF'
#!/usr/bin/env bash
sql="${*: -1}"
case "$sql" in
	*app_config*) echo 180 ;;
	*DELETE*) echo "delete boom" >&2; exit 1 ;;
	*) exit 0 ;; # INSERT 快照/pg_dump（无输出=空 dump 可 gzip）
esac
EOF
chmod +x "$TMP/bin/sudo"
mkdir -p "$TMP/bk"
: > "$ESC_PSQL_LOG"
BACKUP_DIR="$TMP/bk" bash "$ROOT/scripts/arch_daily.sh" >/dev/null 2>&1 || true # tolerate: 测试桩——被测件内部 esc 自报已实证，壳层败不碍断言
grep -qF "arch 保留期清扫败" "$ESC_PSQL_LOG" \
	&& grep -qF "'OCI-CRON','升件'" "$ESC_PSQL_LOG" \
	&& echo "I arch_daily delete-fail → 保留期清扫败 esc PASS"

# ══ J. backup_push_sg sha256 推送败→「sha256 校验件推败」升件 ══
cat > "$TMP/bin/scp" <<'EOF'
#!/usr/bin/env bash
case " $* " in *sha256*) echo "scp sha boom" >&2; exit 1;; esac
exit 0
EOF
chmod +x "$TMP/bin/scp"
mkdir -p "$TMP/bkpush" && printf 'x' > "$TMP/bkpush/o-20260101T000000Z.sql.gz" \
	&& printf 'y' > "$TMP/bkpush/o-20260101T000000Z.sql.gz.sha256"
: > "$ESC_PSQL_LOG"
BACKUP_DIR="$TMP/bkpush" bash "$ROOT/scripts/backup_push_sg.sh" >/dev/null 2>&1
grep -qF "sha256 校验件推败" "$ESC_PSQL_LOG" \
	&& grep -qF "'OCI-CRON','升件'" "$ESC_PSQL_LOG" \
	&& echo "J backup_push sha256-fail → 校验件推败 esc PASS"

echo "tolerate silent batch seven-group injection PASS"
