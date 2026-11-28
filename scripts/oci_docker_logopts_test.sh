#!/usr/bin/env bash
# oci_docker_logopts_test.sh —— O-41390 对抗夹具（本地零 docker 依赖）。
# 三案合并逻辑+幂等双跑+assert 负案（sudo 假死→die 径报错行可见）。
# 跑法：bash scripts/oci_docker_logopts_test.sh
set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$HERE/oci_docker_logopts.sh"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT
FAIL=0

ok()  { echo "  ✅ $*"; }
bad() { echo "  ❌ $*"; FAIL=1; }

# 断言 helper
assert_eq() { [ "$2" = "$3" ] && ok "$1" || bad "$1（got='$2' want='$3'）"; }
assert_rc() { [ "$2" = "$3" ] && ok "$1" || bad "$1（rc=$2 want=$3）"; }

# restart 桩——记录调用次数（幂等证：已对零变=零调用）
RESTART_LOG="$T/restart.log"
: > "$RESTART_LOG"
cat > "$T/restart_stub.sh" <<EOF
#!/bin/sh
echo called >> "$RESTART_LOG"
EOF
chmod +x "$T/restart_stub.sh"
export DOCKER_LOGOPTS_RESTART="$T/restart_stub.sh"

echo "══ 案 a：daemon.json 不存在 → 写规范件 ══"
export DOCKER_DAEMON_JSON="$T/case-a/daemon.json"
mkdir -p "$T/case-a"
out=$("$TOOL" ensure 2>&1); rc=$?
assert_rc "ensure rc=0" "$rc" 0
got=$(python3 -c "import json;c=json.load(open('$DOCKER_DAEMON_JSON'));print(c.get('log-driver'),c['log-opts']['max-size'],c['log-opts']['max-file'])" 2>/dev/null || echo PARSE_FAIL)
assert_eq "规范件三键" "$got" "json-file 50m 3"
[ "$(wc -l < "$RESTART_LOG")" = "1" ] && ok "restart 调一次" || bad "restart 计数异常"

echo "══ 案 b：载 registry-mirrors+runtimes → 键全保+log-opts 补入 ══"
export DOCKER_DAEMON_JSON="$T/case-b/daemon.json"
mkdir -p "$T/case-b"
cat > "$DOCKER_DAEMON_JSON" <<'EOF'
{
  "registry-mirrors": ["https://mirror.example.com"],
  "runtimes": {"nvidia": {"path": "/usr/bin/nvidia-container-runtime"}}
}
EOF
: > "$RESTART_LOG"
out=$("$TOOL" ensure 2>&1); rc=$?
assert_rc "ensure rc=0" "$rc" 0
chk=$(python3 - "$DOCKER_DAEMON_JSON" <<'PY'
import json,sys
c=json.load(open(sys.argv[1]))
print(c.get("registry-mirrors")==["https://mirror.example.com"],
      "nvidia" in c.get("runtimes",{}),
      c.get("log-driver"), c["log-opts"]["max-size"], c["log-opts"]["max-file"])
PY
)
assert_eq "镜像源/运行时键全保+log-opts 补入" "$chk" "True True json-file 50m 3"

echo "══ 案 c：已载正确 log-opts → 幂等零变（不 restart）══"
export DOCKER_DAEMON_JSON="$T/case-c/daemon.json"
mkdir -p "$T/case-c"
cat > "$DOCKER_DAEMON_JSON" <<'EOF'
{"log-driver":"json-file","log-opts":{"max-size":"50m","max-file":"3"}}
EOF
: > "$RESTART_LOG"
out1=$("$TOOL" ensure 2>&1); rc1=$?
out2=$("$TOOL" ensure 2>&1); rc2=$?
assert_rc "双跑 rc 全 0" "$rc1$rc2" "00"
assert_eq "双跑输出同（幂等证）" "$out1" "$out2"
echo "$out1" | grep -q "零变" && ok "零变判词在案" || bad "零变判词缺"
[ ! -s "$RESTART_LOG" ] && ok "零 restart 调用" || bad "幂等案竟 restart"

echo "══ 幂等连跑（案 a 件再 ensure）══"
export DOCKER_DAEMON_JSON="$T/case-a/daemon.json"
out3=$("$TOOL" ensure 2>&1); rc3=$?
assert_rc "案a件再跑 rc=0" "$rc3" 0
echo "$out3" | grep -q "零变" && ok "二次 ensure 判零变" || bad "二次 ensure 未判零变: $out3"

echo "══ 负案：assert——缺 log-opts+sudo 假死 → die 径报错行可见 ══"
export DOCKER_DAEMON_JSON="$T/case-neg/daemon.json"
mkdir -p "$T/case-neg" "$T/fakebin"
cat > "$T/case-neg/daemon.json" <<'EOF'
{"log-driver":"json-file"}
EOF
# sudo 桩=恒死（模拟免密 sudo 不可得）
cat > "$T/fakebin/sudo" <<'EOF'
#!/bin/sh
echo "sudo: a password is required" >&2
exit 1
EOF
chmod +x "$T/fakebin/sudo"
out=$(PATH="$T/fakebin:$PATH" "$TOOL" assert 2>&1); rc=$?
assert_rc "assert rc=1" "$rc" 1
echo "$out" | grep -q "禁裸建容器" && ok "die 判词行可见" || bad "die 判词缺: $out"
echo "$out" | grep -q "手修径" && ok "修法行可见" || bad "修法行缺: $out"
# 负案后件面不被污染（assert 不写——ensure 才写；sudo 死=未写）
chk=$(python3 -c "import json;c=json.load(open('$DOCKER_DAEMON_JSON'));print('log-opts' in c)" 2>/dev/null || echo ERR)
assert_eq "负案零写污（log-opts 未入）" "$chk" "False"

echo "══ check 模只读断言 ══"
export DOCKER_DAEMON_JSON="$T/case-a/daemon.json"
"$TOOL" check >/dev/null 2>&1; assert_rc "已对件 check rc=0" "$?" 0
export DOCKER_DAEMON_JSON="$T/case-neg/daemon.json"
"$TOOL" check >/dev/null 2>&1; assert_rc "缺配件 check rc=1" "$?" 1

echo "════════════════════════════════════════"
if [ "$FAIL" = 0 ]; then echo "✅ oci_docker_logopts_test 全绿"; else echo "❌ 有断言败"; exit 1; fi
