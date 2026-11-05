#!/usr/bin/env bash
# wt_gc_retire_test.sh —— T-0375 件④ 退役口径桩测：GC_REPO 覆写×裸仓
# fixture（日期受控 commit），三档闸逐支断言：
#   audit 支 14d+ 无 wt→删；audit 支 14d-→留；带 wt 支→不动；
#   dec-t 支同律；t/* 系支删枝律不回归（已合删/未合留）。
# 干跑先拍（零删证）→实拍（删留对账）。
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT

OLD_DATE=$(date -d '-35 days' '+%Y-%m-%dT%H:%M:%S')   # >RETIRE_D(14d)
MID_DATE=$(date -d '-20 days' '+%Y-%m-%dT%H:%M:%S')   # >RETIRE_D
FRESH_DATE=$(date -d '-2 days' '+%Y-%m-%dT%H:%M:%S')  # <RETIRE_D

export GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@t
export GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@t

# seed 仓造支（日期受控）→推裸仓 fixture
git init -q -b master "$TMP/seed"
echo base > "$TMP/seed/base.txt"
git -C "$TMP/seed" add base.txt
GIT_AUTHOR_DATE="$FRESH_DATE" GIT_COMMITTER_DATE="$FRESH_DATE" \
  git -C "$TMP/seed" commit -q -m base
mkbranch(){ # mkbranch <名> <日期> —— 切新支落一独立 commit（commit 落 HEAD
  # 所在支——光 branch 不切则针留在 master,实证桩）
  git -C "$TMP/seed" checkout -q -b "$1" master
  local f; f="${TMP}/seed/${1//\//-}.txt"
  echo "$1" > "$f"
  git -C "$TMP/seed" add "$f"
  GIT_AUTHOR_DATE="$2" GIT_COMMITTER_DATE="$2" git -C "$TMP/seed" commit -q -m "$1"
  git -C "$TMP/seed" checkout -q master
}
mkbranch audit-t-9001 "$OLD_DATE"   # audit 14d+ 无 wt → 删
mkbranch audit-t-9002 "$FRESH_DATE" # audit 14d- → 留
mkbranch dec-t-9003 "$MID_DATE"     # dec-t 14d+ 无 wt → 删
mkbranch dec-t-9004 "$MID_DATE"     # dec-t 14d+ 带 wt → 不动
git -C "$TMP/seed" branch -q t/9005-x master                    # t/* 已合 → 删（回归闸）
mkbranch t/9006-y "$FRESH_DATE"     # t/* 未合 → 留（回归闸）
git init -q --bare --initial-branch=master "$TMP/gc.git"
git -C "$TMP/seed" push -q "$TMP/gc.git" 'refs/heads/*:refs/heads/*'

# 带 wt 支挂工树（退役闸豁免对象）
git -C "$TMP/gc.git" worktree add -q "$TMP/wt-dec" dec-t-9004

export GC_REPO="$TMP/gc.git"
printf 'NO_FIN_DSN_HERE\n' > "$TMP/env"
export GC_ENV_FILE="$TMP/env"

branches(){ git -C "$TMP/gc.git" branch --format='%(refname:short)' | sort; }

# ── 干跑拍：零删证 ──
DRY_OUT=$(bash "$ROOT/scripts/meta/wt_gc.sh" --dry-run)
before=$(branches)
grep -q '退役枝: audit-t-9001' <<<"$DRY_OUT" || { echo "FAIL 干跑未见退役行 audit-t-9001"; exit 1; }
grep -q 'retire=2' <<<"$DRY_OUT" || { echo "FAIL 干跑 retire 计数非2:$(grep retire <<<"$DRY_OUT")"; exit 1; }
[ "$(branches)" = "$before" ] || { echo "FAIL 干跑动了支"; exit 1; }
echo "PASS 干跑零删（退役行在,支集不动）"

# ── 实拍：删/留对账 ──
out=$(bash "$ROOT/scripts/meta/wt_gc.sh")
expect(){ branches | grep -Fxq "$1" && { echo "PASS 留 $1（$2）"; return 0; } \
  || { echo "FAIL 应留 $1（$2）——实存:[$(branches | tr '\n' ' ')]"; return 1; }; }
gone(){ ! branches | grep -Fxq "$1" && { echo "PASS 删 $1（$2）"; return 0; } \
  || { echo "FAIL 应删 $1（$2）"; return 1; }; }

rc=0
gone     audit-t-9001 "audit 14d+ 无 wt 口径退役"      || rc=1
expect   audit-t-9002 "audit 14d- 留"                  || rc=1
gone     dec-t-9003  "dec-t 14d+ 无 wt 口径退役"       || rc=1
expect   dec-t-9004  "带 wt 支不动"                    || rc=1
grep -q "RETIRE-SKIP 带wt: dec-t-9004" <<<"$out" || { echo "FAIL 带wt豁免行缺"; rc=1; }
gone     t/9005-x    "t/* 已合删（回归闸）"            || rc=1
expect   t/9006-y    "t/* 未合留（回归闸）"            || rc=1
[ "$(grep -c '^退役枝:' <<<"$out")" = 2 ] || { echo "FAIL 实拍退役行数非2"; rc=1; }
[ -d "$TMP/wt-dec" ] || { echo "FAIL 带wt工树被摘"; rc=1; }

if [ "$rc" = 0 ]; then echo "─────"; echo "✅ wt_gc 退役口径桩测全绿"; exit 0
else echo "❌ 桩测红"; exit 1; fi
