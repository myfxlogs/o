#!/usr/bin/env bash
# check_fix_red_first.sh —— T-0388 修复票先红桩裁决机检（决策席核查清单同槽：
# 修复类票收口无先红断言=驳回条件；工程纪律 §3 条款的机内化件）。
# 用法：check_fix_red_first.sh <票件.md>
# 出口：0=放（非修复类/豁免成立/先红断言在案）；1=驳回件（修复类无先红
#       断言，或豁免词面出界——豁免面只限新功能/纯文档，扩豁免即升件）；
#       2=用法/件缺（fail-closed 显形非默放）。
# 判据：题面（首行 `# ` 头）含修复信号词=修复类票；词表与 commit-msg
# 修复信号同族（去裸「修/坑」——票题面误伤面大）。断言面=票件全文
# （验收锚钦定位+结果回填证据位皆计——「收口无先红断言」=全文零携）。
set -u
F="${1:-}"
[ -n "$F" ] && [ -f "$F" ] || { echo "用法: check_fix_red_first.sh <票件.md>" >&2; exit 2; }

TITLE=$(grep -m1 '^# ' "$F" || true) # tolerate: 无 `# ` 题头=题面无修复信号——空 TITLE 落非修复类分支显判非静默吞
if ! printf '%s' "$TITLE" | grep -qiE 'fix|修复|根修|bug|bugfix|hotfix|regression|回归|病灶|patch|revert'; then
  echo "PASS 非修复类票（题面无修复信号）——先红桩不适用: $F"
  exit 0
fi

if grep -qE '先红桩豁免|先红.{0,8}豁免|豁免.{0,8}先红' "$F"; then
  if grep -qE '新功能|纯文档' "$F"; then
    echo "PASS 先红桩豁免成立（新功能/纯文档票面注明）: $F"
    exit 0
  fi
  echo "FAIL 豁免词面出界（豁免面只限新功能/纯文档——扩豁免即升件）: $F" >&2
  exit 1
fi

if grep -vE '豁免' "$F" | grep -qE '先红|修前红|病灶存在断言|红桩'; then
  echo "PASS 先红断言在案: $F"
  exit 0
fi
echo "FAIL 修复类票无先红断言——驳回条件（病灶存在断言修前红+同断言修后绿未携）: $F" >&2
exit 1
