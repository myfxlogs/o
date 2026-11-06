#!/usr/bin/env bash
# fix_red_first_test.sh —— T-0388 锚②桩测：check_fix_red_first.sh 假票矩阵
# 检出拒/放两态——修复类票无先红断言=驳回件检出（裁决面机检同槽）。
# 矩阵：A 修复票无先红→拒；B 修复票携先红锚→放；C 非修复票→放；
#       D 豁免成立（纯文档）→放；E 豁免词面出界（非新功能/纯文档）→拒；
#       F 仅回填携修前红证据→放；G 件缺→用法错 rc2（fail-closed 显形）。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
CHK="$ROOT/scripts/meta/check_fix_red_first.sh"
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT

cat > "$TMP/T-9001-某面nil崩根修.md" <<'EOF'
# T-9001 某面 nil 崩根修
- 状态：待验
## 验收锚
① 桩测全绿；② check_all 绿戳对口。
## 结果回填
桩测 PASS 已附。
EOF

cat > "$TMP/T-9002-某面nil崩根修.md" <<'EOF'
# T-9002 某面 nil 崩根修
## 验收锚
① 先红桩：修前 panic 复现断言红；② 同断言修后绿；③ check_all 绿戳。
## 结果回填
修前红证据：go test -run X panic 复现输出已附；修后同断言 PASS。
EOF

cat > "$TMP/T-9003-新报表面.md" <<'EOF'
# T-9003 新报表面——观测面新件
## 验收锚
① DOM 桩测渲染；② check_all 绿戳。
EOF

cat > "$TMP/T-9004-某面nil崩根修.md" <<'EOF'
# T-9004 某面 nil 崩根修
## 验收锚
① check_all 绿戳对口。（先红桩豁免：纯文档面——票体仅改 docs/机检文）
## 结果回填
纯文档件已落。
EOF

cat > "$TMP/T-9005-某面nil崩根修.md" <<'EOF'
# T-9005 某面 nil 崩根修
## 验收锚
① check_all 绿戳对口。（先红桩豁免：历史债面急修来不及桩）
## 结果回填
已修。
EOF

cat > "$TMP/T-9006-某面nil崩根修.md" <<'EOF'
# T-9006 某面 nil 崩根修
## 验收锚
① 桩测全绿；② check_all 绿戳。
## 结果回填
测试证据：修前红 go test -run Y FAIL 复现 panic 输出附；修后 PASS。
EOF

pass=0; fail=0
want() { # $1=档 $2=期望rc $3=释
  local rc=0 out
  out=$("$CHK" "$1" 2>&1) || rc=$?
  if [ "$rc" = "$2" ]; then echo "PASS $3（rc=$rc）"; pass=$((pass+1));
  else echo "FAIL $3（期望rc=$2 实rc=$rc 出:$out）"; fail=$((fail+1)); fi
}

want "$TMP/T-9001-某面nil崩根修.md" 1 "A 修复票无先红断言→拒"
want "$TMP/T-9002-某面nil崩根修.md" 0 "B 携先红锚+修前红证据→放"
want "$TMP/T-9003-新报表面.md"      0 "C 非修复票→放（不适用）"
want "$TMP/T-9004-某面nil崩根修.md" 0 "D 豁免成立（纯文档）→放"
want "$TMP/T-9005-某面nil崩根修.md" 1 "E 豁免词面出界→拒"
want "$TMP/T-9006-某面nil崩根修.md" 0 "F 回填携修前红证据→放"

rc=0; out=$("$CHK" "$TMP/不存在.md" 2>&1) || rc=$?
if [ "$rc" = "2" ]; then echo "PASS G 件缺→rc2 fail-closed"; pass=$((pass+1));
else echo "FAIL G 件缺（期望rc=2 实rc=$rc 出:$out）"; fail=$((fail+1)); fi

echo "─────"
if [ "$fail" = 0 ]; then echo "✅ 先红闸机检桩测 $pass/$((pass+fail)) 全绿"; exit 0
else echo "❌ 桩测红：$fail 败/$((pass+fail))"; exit 1; fi
