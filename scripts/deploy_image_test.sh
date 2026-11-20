#!/usr/bin/env bash
# deploy_image_test.sh —— 出处闸拒部留痕桩测（工单#25903ⓑ）：
# 脏树拒→deploy-gate 行落 ops_event（stray 载单）；净树过闸零行。
# 桩件：psql（参数全录 PSQL_ARGS）、timeout（feishu 径吞掉零真通知）、
# ssh（拒径不到不了锁段仍桩兜底）；deploy_image 拷入夹具+env 径改指
# 桩 env（FIN_PG_DSN=stub——真账零触）。
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
DI="$HERE/deploy_image.sh"
T=$(mktemp -d /tmp/di-test-XXXXXX)
trap 'rm -rf "$T"' EXIT
PASS=0; FAILN=0
ok()  { PASS=$((PASS+1)); echo "  ok   $1"; }
bad() { FAILN=$((FAILN+1)); echo "  FAIL $1"; }

mkdir -p "$T/bin" "$T/home"
cat > "$T/bin/psql" <<'EOF'
#!/usr/bin/env bash
echo "$*" >> "${PSQL_ARGS:-/dev/null}"
exit 0
EOF
chmod +x "$T/bin/psql"
cat > "$T/bin/timeout" <<'EOF'
#!/usr/bin/env bash
# timeout 桩：feishu-notify 径吞掉（测试零真通知）；余剥首参时长直跑
case "$*" in *feishu-notify*) exit 0;; esac
shift; exec "$@"
EOF
chmod +x "$T/bin/timeout"
cat > "$T/bin/ssh" <<'EOF'
#!/usr/bin/env bash
exit 1
EOF
chmod +x "$T/bin/ssh"
printf 'FIN_PG_DSN=stub\n' > "$T/env"

# 夹具：bare origin + clone；deploy_image 拷入 scripts/（env 径改指桩）
new_repo() {
	rm -rf "$T/o" "$T/origin.git" "$T/args.log"
	git init --bare -q "$T/origin.git"
	git init -q -b master "$T/o"
	git -C "$T/o" config user.email t@t
	git -C "$T/o" config user.name t
	git -C "$T/o" remote add origin "$T/origin.git"
	mkdir -p "$T/o/scripts"
	sed "s|/opt/o-research/env|$T/env|" "$DI" > "$T/o/scripts/deploy_image.sh"
	chmod +x "$T/o/scripts/deploy_image.sh"
	echo base > "$T/o/f.txt"
	git -C "$T/o" add -A && git -C "$T/o" commit -qm init
	git -C "$T/o" push -qu origin master
}
run_di() {
	( cd "$T/o" && env -i PATH="$T/bin:/usr/bin:/bin" HOME="$T/home" \
		PSQL_ARGS="$T/args.log" bash scripts/deploy_image.sh >/dev/null 2>&1 )
}
has() { grep -q "$1" "$2" 2>/dev/null; }

# ═══ ① 脏树拒→deploy-gate 行载 stray ═══
echo "== ① 脏树拒留痕 =="
new_repo
echo stray1 > "$T/o/stray_a.txt"; echo stray2 > "$T/o/stray_b.txt"
run_di; rc=$?
[ "$rc" = 1 ] && ok "脏树拒 rc=1" || bad "脏树拒 rc=$rc"
has 'deploy-gate 脏树拒' "$T/args.log" && has 'stray_a.txt' "$T/args.log" \
  && ok "deploy-gate 行载 stray" || bad "deploy-gate 行缺/载面不全"
has 'deploy-gate 未并' "$T/args.log" && bad "脏树径竟落未并行" || ok "脏树径不误落未并行"

# ═══ ② 净树→零 deploy-gate 行（过出处闸进锁段——ssh 桩拒=后续败非闸）═══
echo "== ② 净树零闸行 =="
new_repo
run_di; rc=$?
has 'deploy-gate' "$T/args.log" && bad "净树竟落闸行" || ok "净树零 deploy-gate 行"

echo "== 汇总: PASS=$PASS FAIL=$FAILN =="
[ "$FAILN" -eq 0 ]
