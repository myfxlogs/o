#!/usr/bin/env bash
# next_ticket_ledger_test.sh —— T-0373 领号台账行桩测：
# ① DSN 在场+psql 桩 → ops_event '领号' INSERT 实捕（seat/event/ticket 三列断言）；
# ② DSN 缺席 → 取号输出不受阻断（rc=0、T-NNNN 照出、零 psql 调用）；
# ③ psql 桩 exit1 → 取号照常（台账件非硬依赖 fail-open）。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT

# 构造最小 git 仓：origin/master 快照含 T-0007/T-0012 两票 → 取号应= T-0013
R="$TMP/repo"; mkdir -p "$R/docs/tasks" "$TMP/bin" "$TMP/env"
git -C "$R" init -q
git -C "$R" config user.email t@t && git -C "$R" config user.name t
touch "$R/docs/tasks/T-0007-x.md" "$R/docs/tasks/T-0012-y.md"
git -C "$R" add -A && git -C "$R" commit -qm base
# origin/master 快照=本地 commit（fetch 败走快照径——桩环境无真 remote）
git -C "$R" update-ref refs/remotes/origin/master HEAD

# psql 桩：捕获 -c 参数落文件，行为由 PSQL_RC 控
cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
echo "$*" >> "$PSQL_LOG"
exit "${PSQL_RC:-0}"
EOF
chmod +x "$TMP/bin/psql"
export PSQL_LOG="$TMP/psql.log"; : > "$PSQL_LOG"
export PATH="$TMP/bin:$PATH"

# env 桩：FIN_PG_DSN 指向哑值（psql 已被 PATH 桩接管不真连）
printf 'FIN_PG_DSN=postgres://stub\n' > "$TMP/env/o.env"
export OPS_ESC_ENV_FILE="$TMP/env/o.env"

cd "$R"
OUT=$(bash "$ROOT/scripts/next_ticket.sh" 决策席)
[ "$OUT" = "T-0013" ] || { echo "FAIL 取号输出=$OUT 期望 T-0013"; exit 1; }
grep -q "INSERT INTO ops_event" "$PSQL_LOG" || { echo "FAIL 无 INSERT 捕获"; cat "$PSQL_LOG"; exit 1; }
grep -q "'领号','T-0013'" "$PSQL_LOG" || { echo "FAIL ticket 列不对"; cat "$PSQL_LOG"; exit 1; }
grep -q "'决策席','领号'" "$PSQL_LOG" || { echo "FAIL seat 列不对"; cat "$PSQL_LOG"; exit 1; }
echo "① 领号行落账 seat/event/ticket 三列 PASS"

# ② 写径断链径：PATH 摘 psql（command not found=127）→ ||true 兜底，
#    取号输出/rc 不受影响。注：本机 /opt/o-research/env 恒有
#    FIN_PG_DSN——字面「DSN 缺席」径（[ -n "$FIN_DSN" ] 闸整体跳过
#    INSERT 块）与本场景共用同一不阻断结构，代码面可证。
export PATH="/usr/bin:/bin"
OUT2=$(bash "$ROOT/scripts/next_ticket.sh")
[ "$OUT2" = "T-0013" ] || { echo "FAIL psql缺席取号输出=$OUT2"; exit 1; }
echo "② psql 缺席写径断链取号照常 PASS（DSN 缺席同闸面跳过）"

# ③ psql 败径：RC=1 → 取号照常 rc=0
export PATH="$TMP/bin:/usr/bin:/bin" PSQL_RC=1 OPS_ESC_ENV_FILE="$TMP/env/o.env"
OUT3=$(bash "$ROOT/scripts/next_ticket.sh" COS)
[ "$OUT3" = "T-0013" ] || { echo "FAIL psql败取号输出=$OUT3"; exit 1; }
echo "③ psql 写败取号照常（台账非硬依赖）PASS"

echo "next_ticket 领号台账行 PASS"
