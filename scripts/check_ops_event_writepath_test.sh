#!/usr/bin/env bash
# check_ops_event_writepath_test.sh —— 升件写径白名单闸桩测（工单
# O-30514 验收「植入违例重放=红」）：OPS_WRITEPATH_ROOT 夹具根三案——
# ①白名单外 esc 直写（列单 esc_class 形+event='升件' 字面量形）=红；
# ②白名单册内同形=绿；③非 esc 写（心跳/恢复 SELECT 引 esc_scope/
# '升件#' 销件引用）=绿不误咬。用法：bash scripts/check_ops_event_
# writepath_test.sh
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
LINT="$HERE/check_ops_event_writepath.sh"
T=$(mktemp -d /tmp/owp-test-XXXXXX)
trap 'rm -rf "$T"' EXIT
PASS=0; FAILN=0
ok()  { PASS=$((PASS+1)); echo "  ok   $1"; }
bad() { FAILN=$((FAILN+1)); echo "  FAIL $1"; }

mkdir -p "$T/root/scripts"
cat > "$T/root/scripts/ops_event_esc_whitelist.txt" <<'EOF'
# 桩测白名单
scripts/whitelisted.sh
EOF

# 桩 .sh 件（违例面——两判形各一）
cat > "$T/root/scripts/violator_a.sh" <<'EOF'
#!/usr/bin/env bash
psql "$DSN" -c "INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
	VALUES('探测席','升件','','$n','部署','$s')"
EOF
cat > "$T/root/scripts/violator_b.sh" <<'EOF'
#!/usr/bin/env bash
psql "$dsn" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note) VALUES('OCI-CRON','升件','','$1——$2')"
EOF
cat > "$T/root/scripts/whitelisted.sh" <<'EOF'
#!/usr/bin/env bash
psql "$dsn" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note) VALUES('OCI-CRON','升件','','$1——$2')"
EOF
# 非 esc 写三形——皆应放行（判据不误咬面）
cat > "$T/root/scripts/clean_hb.sh" <<'EOF'
#!/usr/bin/env bash
psql "$DSN" -c "INSERT INTO ops_event(seat,event,ticket,note)
	VALUES('DEPLOY','心跳','','心跳 拍始 sha=x base=y')"
EOF
cat > "$T/root/scripts/clean_recover.sh" <<'EOF'
#!/usr/bin/env bash
psql "$DSN" -qAt <<'SQL'
INSERT INTO ops_event(seat,event,note)
SELECT '探测席','恢复','升件#'||e.id||' 复绿'
FROM ops_event e WHERE e.event='升件' AND e.esc_scope LIKE 'pendbr\_%'
SQL
EOF
cat > "$T/root/scripts/clean_deploy.sh" <<'EOF'
#!/usr/bin/env bash
psql "$DSN" -c "INSERT INTO ops_event(seat,event,ticket,note) VALUES('DEPLOY','部署','','deploy rc=0')"
EOF

echo "== ① 白名单外 esc 直写两判形=红 =="
OPS_WRITEPATH_ROOT="$T/root" bash "$LINT" >/dev/null 2>&1; rc=$?
[ "$rc" != 0 ] && ok "违例闸红 rc=$rc" || bad "违例竟放行 rc=$rc"

echo "== ② 移除违例件→绿（白名单册内件不咬）=="
rm "$T/root/scripts/violator_a.sh" "$T/root/scripts/violator_b.sh"
OPS_WRITEPATH_ROOT="$T/root" bash "$LINT" >/dev/null 2>&1; rc=$?
[ "$rc" = 0 ] && ok "册内+非esc面全绿" || bad "误咬 rc=$rc"

echo "== ③ 纯非 esc 面=绿（册内路径失效不挡——非esc写本不入判）=="
rm "$T/root/scripts/whitelisted.sh"
cat > "$T/root/scripts/ops_event_esc_whitelist.txt" <<'EOF'
# 册内路径失效——非 esc 写不该被咬
scripts/nonexistent.sh
EOF
OPS_WRITEPATH_ROOT="$T/root" bash "$LINT" >/dev/null 2>&1; rc=$?
[ "$rc" != 0 ] && bad "非esc写误咬 rc=$rc" || ok "非esc写零咬（心跳/恢复/部署）"

echo "== ④ 白名单缺席=fail-closed 红 =="
rm "$T/root/scripts/ops_event_esc_whitelist.txt"
OPS_WRITEPATH_ROOT="$T/root" bash "$LINT" >/dev/null 2>&1; rc=$?
[ "$rc" != 0 ] && ok "缺席闸红 rc=$rc" || bad "缺席竟放行"

echo "== 汇总: PASS=$PASS FAIL=$FAILN =="
[ "$FAILN" -eq 0 ]
