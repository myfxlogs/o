#!/usr/bin/env bash
# batchdesk_test.sh —— 批组拾桌查询件桩测（工单O-34419）。
# 桩 psql 按 SQL 词面分派：app_config batch_max 查询回 STUB_MAX；
# ops_ticket_desk top10 查询回 STUB_DESK（id\x1fnote 行集）。
# 案件：
#   ①≥2 REVIEW 道票→候选表列出+--count 同数
#   ②FREE 道票（spec obj 全落文档白名单）→剔出不入批
#   ③spec 缺席/obj 无路径词→REVIEW fail-closed 计入
#   ④dm.worker.batch_max=2 截批（3 REVIEW→出 2）
#   ⑤batch_max 缺键/非数→缺省 4
#   ⑥dispatch_worker.sh batchable= 接线锚+runbook 批组段锚在账
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
BD="$HERE/batchdesk.sh"
T=$(mktemp -d /tmp/batchdeskXXXXXX)
trap 'rm -rf "$T"' EXIT
PASS=0; FAILN=0
ok()  { PASS=$((PASS+1)); echo "  ok   $1"; }
bad() { FAILN=$((FAILN+1)); echo "  FAIL $1"; }

mkdir -p "$T/bin" "$T/prompts"
printf 'FIN_PG_DSN=stub://batchdesk\n' > "$T/env"
# 桩 spec 件：rev1/rev2=码件 obj（REVIEW）；free1=纯文档 obj（FREE）；
# nospec=无 spec 文件票（fail-closed REVIEW）；nopat=spec 在但 obj 无路径词。
mk_spec() { printf 'id: X\nobj: %s\n' "$2" > "$T/prompts/O-$1-test.md"; }
mk_spec 101 'internal/foo/bar.go——码件'
mk_spec 102 'scripts/cognition/x.sh + cmd/core/y.go'
mk_spec 103 'docs/工程经验库/条目/foo.md + skills/bar.md'
mk_spec 105 '词面闸全谱普查——散文 obj 无路径词'
# 104=spec 文件缺席（不建）

cat > "$T/bin/psql" <<'STUB'
#!/usr/bin/env bash
sql=""; prev=""
for a in "$@"; do [ "$prev" = "-c" ] && sql="$a"; prev="$a"; done
case "$sql" in
	*batch_max*) echo "${STUB_MAX:-4}" ;;
	*ops_ticket_desk*) printf '%s' "${STUB_DESK:-}" ;;
esac
exit 0
STUB
chmod +x "$T/bin/psql"

FS=$'\x1f'
run_bd() { # 余参=env 前缀覆写；stdout=批候选表
	env PATH="$T/bin:$PATH" REPORT_ENV_FILE="$T/env" \
		STUB_MAX="${STUB_MAX:-4}" STUB_DESK="${STUB_DESK:-}" \
		bash "$BD" "$@" "$T/prompts" 2>/dev/null
}

# 案① 三 REVIEW+一 FREE 混合桌 → REVIEW 三票出、FREE 剔、count=3
STUB_DESK="101${FS}工单/rev-a: 码件票 spec=/opt/o-research/prompts/O-101-test.md
102${FS}工单/rev-b: 码件票 spec=/opt/o-research/prompts/O-102-test.md
103${FS}工单/free-c: 文档票 spec=/opt/o-research/prompts/O-103-test.md
105${FS}工单/nopat-d: 散文票 spec=/opt/o-research/prompts/O-105-test.md"
out=$(STUB_DESK="$STUB_DESK" run_bd)
[ "$(printf '%s\n' "$out" | grep -c .)" -eq 3 ] && ok "混合桌 REVIEW 三票出" || bad "混合桌出数=$(printf '%s\n' "$out" | grep -c .) 应=3: $out"
printf '%s\n' "$out" | grep -q '^103|' && bad "FREE 票竟入批" || ok "FREE 票剔出不入批"
printf '%s\n' "$out" | grep -q '^105|' && ok "无路径词 obj 判 REVIEW 计入" || bad "无路径词 obj 未计入"
cnt=$(STUB_DESK="$STUB_DESK" run_bd --count)
[ "$cnt" = "3" ] && ok "--count=3 与候选表一致" || bad "--count=$cnt 应=3"

# 案② spec 缺席票 fail-closed REVIEW
STUB_DESK="104${FS}工单/nospec-e: 无 spec 票"
out=$(STUB_DESK="$STUB_DESK" run_bd)
printf '%s\n' "$out" | grep -q '^104|' && ok "spec 缺席 fail-closed REVIEW" || bad "spec 缺席未计入: $out"

# 案③ batch_max=2 截批——4 REVIEW 候选出 2
mk_spec 106 'internal/a.go'; mk_spec 107 'internal/b.go'; mk_spec 108 'internal/c.go'
STUB_DESK="101${FS}工单/a spec=x
102${FS}工单/b spec=x
106${FS}工单/c spec=x
107${FS}工单/d spec=x
108${FS}工单/e spec=x"
out=$(STUB_MAX=2 STUB_DESK="$STUB_DESK" run_bd --count)
[ "$out" = "2" ] && ok "batch_max=2 截批 count=2" || bad "截批 count=$out 应=2"
out2=$(STUB_MAX=2 STUB_DESK="$STUB_DESK" run_bd)
[ "$(printf '%s\n' "$out2" | grep -c .)" -eq 2 ] && ok "截批候选表=2 行" || bad "截批表行数异常: $out2"

# 案④ batch_max 缺键/非数→缺省 4
out=$(STUB_MAX= STUB_DESK="$STUB_DESK" run_bd --count)
[ "$out" = "4" ] && ok "batch_max 缺省=4" || bad "缺省 count=$out 应=4"
out=$(STUB_MAX=abc STUB_DESK="$STUB_DESK" run_bd --count)
[ "$out" = "4" ] && ok "batch_max 非数回落=4" || bad "非数 count=$out 应=4"

# 案⑤ 接线锚：dispatch_worker batchable= 段+runbook 批组规约在账
bash -n "$HERE/dispatch_worker.sh" && grep -q 'batchable=\$BATCHABLE_N' "$HERE/dispatch_worker.sh" \
	&& grep -q 'batchdesk.sh' "$HERE/dispatch_worker.sh" \
	&& ok "dispatch_worker batchable 接线在账" || bad "dispatch_worker 接线缺"
grep -q '批组规约' "$HERE/worker_runbook.md" && ! grep -q '同slug族' "$HERE/worker_runbook.md" \
	&& ok "runbook 批组段替换在案(旧界清)" || bad "runbook 批组段缺/旧界残"
grep -q 'dm.worker.batch_max' "$HERE/worker_runbook.md" \
	&& ok "batch_max 缺省语义入 runbook" || bad "batch_max 语义未载"
# 病灶锚：597b90b94 引入的拾桌谓词孤儿右括号（syntax error→TICKET_N
# 恒空→派发静默死五日）——两文件同谓词面禁再犯。
! grep -q "feed-' )" "$HERE/dispatch_worker.sh" "$HERE/batchdesk.sh" \
	&& ok "拾桌谓词无孤儿右括号(派发死症锚)" || bad "拾桌谓词孤儿括号残留"

echo "batchdesk_test: PASS=$PASS FAIL=$FAILN"
[ "$FAILN" -eq 0 ]
