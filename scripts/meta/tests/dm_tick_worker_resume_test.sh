#!/usr/bin/env bash
# dm_tick_worker_resume_test.sh —— GW-001 A2 瘦身判据（原 worker 补拉面
# 整体迁 mech_tick.sh——本测改钉「判断壳零机械调用+尾段唤 o-mech」）：
#   ①empty 队零会话+唤 mech ②review 队开班+唤 mech ③approved-only 零会话
#     +唤 mech（已批待派面归机械轨不烧会话）④锁拒即退零唤零心跳
#   ⑤drain 收班判 exit 心跳 ⑥禁双轨——depgate/补拉查询/suspend/handoff/
#     ledger/verdict/autoapprove/ticketdraft(systemctl 唤 mech 除外)
#     桩件零调用=机械唯一执行者 o-mech 实证。
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin" "$TMP/tasks" "$TMP/cto"
touch "$TMP/tasks/T-0341-resume-test.md"

cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
query="${*: -1}"
case "$query" in
	# T-0354 advisory-lock holder 桩（无 -c=锁会话形）——DM_LOCK_RESULT=f 复现锁拒
	"-qAt") [ "${DM_LOCK_RESULT:-t}" = f ] && echo f || echo t ;;
	# GW-001 段B1：ops_ticket_phase 探针败=legacy 桥面（桩场景钉旧推导逐字）
	*"ops_ticket_phase"*) exit 1 ;;
	# T-0354：队列判据单源 dm_queue.sh——stub 按场景回票号行集；
	# drain 场景=闸内首查有件/班后复查空（DRAIN_MARK 幂等翻转）→收班 exit。
	# T-0414：会话闸=review 面——按 SQL 形分面回集（approved 场景 review 面
	# 空集=不开班，机械轨补拉窗归 mech_tick 本测不触）。
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
	# GW-001 A2 禁双轨探针：补拉候选集查询已迁 mech_tick——dm_tick 再发
	# 此查询=双轨实证，落 BAD_LOG 终判炸。
	*"SELECT l.ticket ||"*) printf 'DISPATCH-QUERY-FIRED\n' >> "$BAD_LOG" ;;
	*"INSERT INTO ops_headless"*)
		printf '%s\n' "$query" >> "$HB_LOG"
		;;
	*"INSERT INTO ops_event"*)
		printf 'OPS-EVENT-WRITE\n' >> "$BAD_LOG" # dm_tick 不再直写事件（自报走 ops_esc 桩）
		;;
esac
EOF
cat > "$TMP/bin/systemctl" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$SYSTEMCTL_LOG"
if [[ ${2:-} == is-active ]]; then
	echo inactive
	exit 3
fi
if [[ ${2:-} == list-units ]]; then
	exit 0
fi
exit 0
EOF
cat > "$TMP/bin/decision-devin" <<'EOF'
#!/usr/bin/env bash
echo called >> "$DECISION_LOG"
EOF
# 机械子件桩——GW-001 A2 后 dm_tick 不得再调（迁 mech_tick）
for n in depgate suspend handoff ledger verdict autoapprove dispatch-worker go; do
	cat > "$TMP/bin/$n" <<EOF
#!/usr/bin/env bash
echo "$n" >> "\$COMP_LOG"
exit 0
EOF
done
chmod +x "$TMP/bin/"*

printf 'FIN_PG_DSN=fake\n' > "$TMP/env"
export PATH="$TMP/bin:$PATH"
export DM_ENV_FILE="$TMP/env"
export O_CTO_WORKDIR="$TMP/cto"
export O_WORKER_USAGE_BIN=/bin/true
export O_OPS_ESC_SH="$TMP/bin/opsesc-stub"
cat > "$TMP/bin/opsesc-stub" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
chmod +x "$TMP/bin/opsesc-stub"
export SYSTEMCTL_LOG="$TMP/systemctl.log"; : > "$SYSTEMCTL_LOG"
export COMP_LOG="$TMP/comp.log"; : > "$COMP_LOG"
export BAD_LOG="$TMP/bad.log"; : > "$BAD_LOG"
export DECISION_LOG="$TMP/decision.log"
export PENDING_SQL_LOG="$TMP/pending.sql.log"; : > "$PENDING_SQL_LOG"
export HB_LOG="$TMP/hb.log"; : > "$HB_LOG"
export DRAIN_MARK="$TMP/drain.mark"
export O_DECISION_DEVIN_BIN="$TMP/bin/decision-devin"
export O_MECH_SYSTEMCTL="$TMP/bin/systemctl" # 唤机械轨桩——落 SYSTEMCTL_LOG 可验

mech_kicked() { grep -qF -- '--user start --no-block o-mech.service' "$SYSTEMCTL_LOG"; }
no_mech() { ! grep -q 'o-mech' "$SYSTEMCTL_LOG"; }

# Pure-empty queue stays zero-LLM but still kicks the mechanical track.
export DM_SCENARIO=empty DM_LOCK_RESULT=t
bash "$ROOT/scripts/meta/dm_tick.sh"
[[ ! -e "$DECISION_LOG" ]]
mech_kicked || { echo "FAIL empty 拍未唤 o-mech"; cat "$SYSTEMCTL_LOG"; exit 1; }
grep -q 'alive' "$HB_LOG"

# T-0414 会话面收窄：闸=review 面——review 待裁→开班；approved-only 待派
# 不开班（机械面归 mech 轨——本壳零会话零机械调用）。
export DM_SCENARIO=review
bash "$ROOT/scripts/meta/dm_tick.sh"
export DM_SCENARIO=approved
bash "$ROOT/scripts/meta/dm_tick.sh"
[[ $(wc -l < "$DECISION_LOG") -eq 1 ]] # 仅 review 拍开班——approved 拍零会话
# T-0354：队列推导单源 dm_queue.sh——empty/approved 各 1 拍（闸查即空跳过）
# +review 2 拍（闸查+班后复查）=4 次 SQL 落账。
[[ $(grep -c '^SELECT l\.ticket FROM (' "$PENDING_SQL_LOG") -eq 4 ]]
mech_kicked

# T-0354 件③：锁拒=后到即退（kick 连发/兜底拍撞车同款）——零决策零唤零心跳。
export DM_SCENARIO=locked DM_LOCK_RESULT=f
: > "$HB_LOG"; : > "$SYSTEMCTL_LOG"
bash "$ROOT/scripts/meta/dm_tick.sh" > "$TMP/locked.log" 2>&1
grep -q '班次闸未得' "$TMP/locked.log"
no_mech || { echo "FAIL 锁拒拍仍唤 o-mech"; cat "$SYSTEMCTL_LOG"; exit 1; }
[[ $(wc -l < "$DECISION_LOG") -eq 1 ]]
[[ ! -s "$HB_LOG" ]]

# T-0354 件②收班判：闸内有件→班后队列空→心跳 state=exit（六态正常终态）。
export DM_SCENARIO=drain DM_LOCK_RESULT=t
rm -f "$DRAIN_MARK"
: > "$HB_LOG"
bash "$ROOT/scripts/meta/dm_tick.sh" >/dev/null
[[ $(wc -l < "$DECISION_LOG") -eq 2 ]]
grep -qF "'CTO','CTO','exit'" "$HB_LOG" # hb 行形=VALUES('CTO','CTO','<state>')——原测「决策席」字面本就失配（前置断言先炸未暴露的潜伏错）
mech_kicked

# GW-001 A2 禁双轨终判：全程机械子件/补拉查询/事件直写零调用。
[[ ! -s "$COMP_LOG" ]] || { echo "FAIL dm_tick 调了机械子件:"; cat "$COMP_LOG"; exit 1; }
[[ ! -s "$BAD_LOG" ]] || { echo "FAIL dm_tick 发机械面查询/事件:"; cat "$BAD_LOG"; exit 1; }

echo "dm tick slim-shell scenarios PASS"
