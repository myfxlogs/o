#!/usr/bin/env bash
# kb_recall_test.sh —— 经验库召回面首阶桩测（提案-KB向量检索-经验库召回面）：
#   ①人造票面含 tolerate/锚 域词→召回带出既有锚坑条目；
#   ②检索物料/库缺席→零输出 rc0（fail-open 增强非闸件）；
#   ③dispatch_worker 装配面「相关经验」段注入/缺席跳过两态；
#   ④top-K 行数封顶+输出形态 `- [lib] id — display`。
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
RECALL="$ROOT/scripts/meta/kb_recall.sh"

# 人造票面：tolerate-report 闸 || true 锚误报案（判据①域词面）。
cat > "$TMP/tolerate-ticket.md" <<'EOF'
# T-9998 桩票——tolerate 锚报告闸误报修复

| 字段 | 值 |
|---|---|
| 涉及面 | scripts/check_tolerate_report.sh、scripts/meta/tests/、docs/容错位清单.md |

## 背景

check_all tolerate-report 段对新增 `|| true` 吞败符缺 `# tolerate:` 行内锚误报。
前票 T-0464/T-0465 连续撞同案——锚词面误中需收紧判定。

## 目标

修复 tolerate-report 闸对测试件吞败符行内锚的判定。
EOF

# ── ① tolerate/锚域词→锚坑条目召回 ──
OUT=$(bash "$RECALL" "$TMP/tolerate-ticket.md" 5)
[ -n "$OUT" ] || { echo "FAIL: tolerate/锚票面零召回" >&2; exit 1; }
grep -qE '锚|anchor|误触|子串匹配' <<< "$OUT" \
	|| { echo "FAIL: 召回未带出锚坑条目: $OUT" >&2; exit 1; }
echo "① tolerate/锚→锚坑条目 PASS"

# ── ④ top-K 封顶+形态 ──
LINES=$(wc -l <<< "$OUT")
[ "$LINES" -le 5 ] || { echo "FAIL: top-K 超限 lines=$LINES" >&2; exit 1; }
while IFS= read -r ln; do
	[[ "$ln" =~ ^-\ \[(eng|econ)\]\ .+\ —\ .+ ]] \
		|| { echo "FAIL: 输出形态漂移: $ln" >&2; exit 1; }
done <<< "$OUT"
K3=$(bash "$RECALL" "$TMP/tolerate-ticket.md" 3 | wc -l)
[ "$K3" -le 3 ] || { echo "FAIL: K=3 未封顶 lines=$K3" >&2; exit 1; }
echo "④ top-K 封顶+形态 PASS"

# ── ② 缺席面：物料/库全缺席→零输出 rc0 ──
EMPTY_REPO="$TMP/empty-repo"
mkdir -p "$EMPTY_REPO/docs"
if OUT2=$(O_KB_RECALL_REPO="$EMPTY_REPO" bash "$RECALL" "$TMP/tolerate-ticket.md"); then
	[ -z "$OUT2" ] || { echo "FAIL: 空仓召回非零: $OUT2" >&2; exit 1; }
else
	echo "FAIL: 空仓召回 rc 非零" >&2; exit 1
fi
# 票面文件缺席/空票面同径零输出
: > "$TMP/empty-ticket.md"
[ -z "$(bash "$RECALL" "$TMP/empty-ticket.md")" ] \
	|| { echo "FAIL: 空票面召回非零" >&2; exit 1; }
[ -z "$(bash "$RECALL" "$TMP/no-such-ticket.md")" ] \
	|| { echo "FAIL: 缺席票面召回非零" >&2; exit 1; }
echo "② 缺席面零输出 rc0 PASS"

# ── ③ dispatch_worker 装配面注入/缺席两态 ──
mkdir -p "$TMP/tasks" "$TMP/bin" "$TMP/state"
cp "$TMP/tolerate-ticket.md" "$TMP/tasks/T-9998-tolerate-anchor.md"

cat > "$TMP/bin/systemctl" <<'EOF'
#!/usr/bin/env bash
if [[ ${2:-} == is-active ]]; then echo inactive; exit 3; fi
if [[ ${2:-} == list-units ]]; then exit 0; fi
exit 2
EOF
cat > "$TMP/bin/systemd-run" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$LAUNCHES"
EOF
chmod +x "$TMP/bin/systemctl" "$TMP/bin/systemd-run"

export O_WORKER_RUNTIME_DIR="$TMP/state"
export O_WORKER_SYSTEMCTL="$TMP/bin/systemctl"
export O_WORKER_SYSTEMD_RUN="$TMP/bin/systemd-run"
export O_WORKER_DEV_RUN_SCRIPT="$ROOT/scripts/meta/dev_run.sh"
export O_WORKER_TASK_DIR="$TMP/tasks"
export O_WORKER_PARALLEL_LIMIT=1
export LAUNCHES="$TMP/launches"
: > "$LAUNCHES"

# 有料态：prompt 尾部带「相关经验」段+锚坑条目行
bash "$ROOT/scripts/meta/dispatch_worker.sh" T-9998 swe-2-medium "领单 T-9998" \
	|| { echo "FAIL: dispatch rc 非零" >&2; exit 1; }
grep -q '相关经验' "$LAUNCHES" \
	|| { echo "FAIL: prompt 未注入相关经验段" >&2; cat "$LAUNCHES" >&2; exit 1; }
grep -qE '锚|anchor|误触|子串匹配' "$LAUNCHES" \
	|| { echo "FAIL: 注入段无锚坑条目" >&2; exit 1; }
echo "③a 有料注入 PASS"

# 缺席态：物料/库不可达→跳过该段照发（fail-open 不炸不误拦）
: > "$LAUNCHES"
O_KB_RECALL_REPO="$EMPTY_REPO" \
	bash "$ROOT/scripts/meta/dispatch_worker.sh" T-9998 swe-2-medium "领单 T-9998" \
	|| { echo "FAIL: 缺席态 dispatch rc 非零" >&2; exit 1; }
[ "$(wc -l < "$LAUNCHES")" -eq 1 ] \
	|| { echo "FAIL: 缺席态未拉起" >&2; exit 1; }
if grep -q '相关经验' "$LAUNCHES"; then
	echo "FAIL: 缺席态仍注入相关经验段" >&2; exit 1
fi
echo "③b 缺席跳过照发 PASS"

echo "kb_recall tests PASS"
