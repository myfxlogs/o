#!/usr/bin/env bash
# check_resident_units_test.sh —— 常驻件登记对拍闸桩测（工单O-30515 验收）：
# 夹具树喂 RESIDENT_SCAN_ROOT/RESIDENT_UNITS_FILE——
# ①绿基线：登记↔实扫三集齐 rc=0；
# ②未登记 cmd 入口重放=红（cmd/ghost/main.go 无登记行）；
# ③登记件删除重放=红（登记 cmd/ghost 而实扫无此入口）；
# ④未登记单元=红 / ⑤登记脚本未被 oneshot 引用=红；
# ⑥类不符=红；⑦登记表缺席=红；⑧oneshot 引用而脚本 payload 缺席=红。
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
LINT="$HERE/check_resident_units.sh"
FAIL=0; PASS=0
T=$(mktemp -d); trap 'rm -rf "$T"' EXIT

ok()  { PASS=$((PASS+1)); echo "PASS $1"; }
bad() { FAIL=$((FAIL+1)); echo "FAIL $1" >&2; }

# 夹具基线：cmd/core + o-test oneshot→scripts/test.sh
mkroot() { # $1=dir
	mkdir -p "$1/cmd/core" "$1/deploy" "$1/scripts"
	printf 'package main\n' > "$1/cmd/core/main.go"
	cat > "$1/deploy/o-test.service" <<'EOF'
[Unit]
Description=fixture
[Service]
Type=oneshot
ExecStart=/bin/bash /opt/o/scripts/test.sh
EOF
	printf '#!/bin/bash\nexit 0\n' > "$1/scripts/test.sh"
	cat > "$1/reg.txt" <<'EOF'
cmd/core|cmd入口|常驻|SG|fixture core
o-test|oci单元|15min|dev|fixture oneshot
scripts/test.sh|定时脚本|15min|dev|fixture payload
EOF
}
run() { # $1=root $2=regfile → rc+out
	RESIDENT_SCAN_ROOT="$1" RESIDENT_UNITS_FILE="$2" bash "$LINT" 2>&1
}
expect_red() { # $1=name $2=root $3=reg $4=grep-needle
	out=$(run "$2" "$3"); rc=$?
	if [ "$rc" != 0 ] && printf '%s' "$out" | grep -q "$4"; then ok "$1"; else bad "$1 (rc=$rc want红含[$4])"; fi
}

# ① 绿基线
R=$T/green; mkroot "$R"
if run "$R" "$R/reg.txt" >/dev/null 2>&1; then ok "绿基线"; else bad "绿基线"; fi

# ② 未登记 cmd 入口
R=$T/unregcmd; mkroot "$R"; mkdir -p "$R/cmd/ghost"; printf 'package main\n' > "$R/cmd/ghost/main.go"
expect_red "未登记cmd入口" "$R" "$R/reg.txt" "未登记常驻件: cmd/ghost"

# ③ 登记件删除（登记 cmd/ghost 实扫无）
R=$T/delcmd; mkroot "$R"; printf 'cmd/ghost|cmd入口|常驻|SG|ghost\n' >> "$R/reg.txt"
expect_red "登记cmd已删" "$R" "$R/reg.txt" "登记件已删/不可判: cmd/ghost"

# ④ 未登记单元
R=$T/unregunit; mkroot "$R"; printf '[Service]\nType=simple\nExecStart=/bin/true\n' > "$R/deploy/o-ghost.service"
expect_red "未登记单元" "$R" "$R/reg.txt" "未登记常驻件: o-ghost"

# ⑤ 登记脚本未被 oneshot 引用（文件在仓但无单元指）
R=$T/delscript; mkroot "$R"; printf '#!/bin/bash\n' > "$R/scripts/ghost.sh"; printf 'scripts/ghost.sh|定时脚本|15min|dev|ghost\n' >> "$R/reg.txt"
expect_red "登记脚本未引用" "$R" "$R/reg.txt" "登记件已删/不可判: scripts/ghost.sh"

# ⑥ 类不符
R=$T/classbad; mkroot "$R"; sed -i 's#^cmd/core|cmd入口#cmd/core|oci单元#' "$R/reg.txt"
expect_red "类不符" "$R" "$R/reg.txt" "类不符: cmd/core"

# ⑦ 登记表缺席
R=$T/noreg; mkroot "$R"
expect_red "登记表缺席" "$R" "$R/no-such-reg.txt" "缺席"

# ⑧ oneshot 引用而脚本 payload 缺席（登记在案、单元在引、文件已删）
R=$T/nopayload; mkroot "$R"; rm "$R/scripts/test.sh"
expect_red "脚本payload缺席" "$R" "$R/reg.txt" "定时脚本 payload 缺席: scripts/test.sh"

echo "== 汇总: PASS=$PASS FAIL=$FAIL =="
[ "$FAIL" = 0 ]
