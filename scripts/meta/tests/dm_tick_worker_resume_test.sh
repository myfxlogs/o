#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin" "$TMP/tasks"
touch "$TMP/tasks/T-0341-resume-test.md"

cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
query="${*: -1}"
case "$query" in
	# T-0354 advisory-lock holder 桩（无 -c=锁会话形）——DM_LOCK_RESULT=f 复现锁拒
	"-qAt") [ "${DM_LOCK_RESULT:-t}" = f ] && echo f || echo t ;;
	# T-0354：队列判据单源 dm_queue.sh——stub 按场景回票号行集；
	# drain 场景=闸内首查有件/班后复查空（DRAIN_MARK 幂等翻转）→收班 exit。
	# T-0414：会话闸=review 面——按 SQL 形分面回集（approved 场景 review 面
	# 空集=不开班，机械轨补拉窗同拍自验）。
	*"SELECT l.ticket FROM ("*)
		printf '%s\n' "$query" >> "$PENDING_SQL_LOG"
		if grep -Fq "e.event='已批'" <<<"$query"; then
			# approved/all 形——派单链终态集谓词钉（depgate 走面，本测不触）
			grep -Fq "('派单','领单','拉起','裁决','完成','驳回','挂起')" <<<"$query"
		else
			# review 形——收口类+终态裁决谓词钉
			grep -Fq "e.event IN ('收口','移交','收口待审')" <<<"$query"
			grep -Fq "('裁决','完成','驳回')" <<<"$query"
		fi
		case "$DM_SCENARIO" in
			empty|approved) ;;
			drain)
				if [ ! -e "$DRAIN_MARK" ]; then touch "$DRAIN_MARK"; printf 'T-9001\n'; fi
				;;
			review) printf 'T-9001\n' ;;
			*) echo "unknown scenario: $DM_SCENARIO" >&2; exit 2 ;;
		esac
		;;
	*"SELECT l.ticket ||"*)
		case "$DM_SCENARIO" in
			empty|approved) printf 'T-0341|swe-2-medium\n' ;;
			review|drain|locked) ;;
			*) echo "unknown scenario: $DM_SCENARIO" >&2; exit 2 ;;
		esac
		;;
	*"INSERT INTO ops_headless"*)
		printf '%s\n' "$query" >> "$HB_LOG"
		;;
esac
EOF
cat > "$TMP/bin/systemctl" <<'EOF'
#!/usr/bin/env bash
if [[ ${2:-} == is-active ]]; then
	echo inactive
	exit 3
fi
if [[ ${2:-} == list-units ]]; then
	exit 0
fi
exit 2
EOF
cat > "$TMP/bin/dispatch-worker" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$DISPATCH_LOG"
EOF
cat > "$TMP/bin/decision-devin" <<'EOF'
#!/usr/bin/env bash
echo called >> "$DECISION_LOG"
EOF
cat > "$TMP/bin/depgate" <<'EOF'
#!/usr/bin/env bash
echo called >> "$DEPGATE_LOG"
EOF
chmod +x "$TMP/bin/"*

printf 'FIN_PG_DSN=fake\n' > "$TMP/env"
export PATH="$TMP/bin:$PATH"
export DM_ENV_FILE="$TMP/env"
export DISPATCH_LOG="$TMP/dispatch.log"
export DECISION_LOG="$TMP/decision.log"
export PENDING_SQL_LOG="$TMP/pending.sql.log"
export HB_LOG="$TMP/hb.log"
export DRAIN_MARK="$TMP/drain.mark"
export DEPGATE_LOG="$TMP/depgate.log"
export O_WORKER_DISPATCH_SCRIPT="$TMP/bin/dispatch-worker"
export O_WORKER_TASK_DIR="$TMP/tasks"
export O_DECISION_DEVIN_BIN="$TMP/bin/decision-devin"
export O_DM_DEPGATE_SH="$TMP/bin/depgate"
export O_DM_LEDGER_SH=/bin/true # T-0414：机械轨件单测——tick 接线面桩掉防真 git/psql

# Pure-empty queue stays zero-LLM but still resumes a rate-limited worker.
export DM_SCENARIO=empty DM_LOCK_RESULT=t
bash "$ROOT/scripts/meta/dm_tick.sh"
[[ ! -e "$DECISION_LOG" ]]
grep -q '^T-0341 swe-2-medium ' "$DISPATCH_LOG"

# T-0414 会话面收窄：闸=review 面——review 待裁→开班；approved-only 待派
# 不开班（机械轨补拉窗同拍自验 T-0341 照拉）。
export DM_SCENARIO=review
bash "$ROOT/scripts/meta/dm_tick.sh"
export DM_SCENARIO=approved
bash "$ROOT/scripts/meta/dm_tick.sh"
[[ $(wc -l < "$DECISION_LOG") -eq 1 ]] # 仅 review 拍开班——approved 拍零会话
# T-0354：队列推导单源 dm_queue.sh——empty/approved 各 1 拍（闸查即空跳过）
# +review 2 拍（闸查+班后复查）=4 次 SQL 落账。
[[ $(grep -c '^SELECT l\.ticket FROM (' "$PENDING_SQL_LOG") -eq 4 ]]
[[ $(grep -c '^T-0341 ' "$DISPATCH_LOG") -eq 2 ]]

# T-0354 件③：锁拒=后到即退（kick 连发/兜底拍撞车同款）——零决策零派单零心跳。
export DM_SCENARIO=locked DM_LOCK_RESULT=f
: > "$HB_LOG"
: > "$DEPGATE_LOG"
bash "$ROOT/scripts/meta/dm_tick.sh" > "$TMP/locked.log" 2>&1
grep -q '班次闸未得' "$TMP/locked.log"
[[ $(wc -l < "$DEPGATE_LOG") -eq 1 ]] # T-0413：锁前依赖闸仍跑——kick 撞班不误零交集快派
[[ $(wc -l < "$DECISION_LOG") -eq 1 ]]
[[ ! -s "$HB_LOG" ]]

# T-0354 件②收班判：闸内有件→班后队列空→心跳 state=exit（六态正常终态）。
export DM_SCENARIO=drain DM_LOCK_RESULT=t
rm -f "$DRAIN_MARK"
: > "$HB_LOG"
bash "$ROOT/scripts/meta/dm_tick.sh" >/dev/null
[[ $(wc -l < "$DECISION_LOG") -eq 2 ]]
grep -qF "决策席','exit'" "$HB_LOG"

echo "dm tick pending gate three-state dry run PASS"
