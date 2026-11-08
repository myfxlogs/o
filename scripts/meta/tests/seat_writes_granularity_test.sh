#!/usr/bin/env bash
# seat_writes_granularity_test.sh —— T-0460 判别粒度修桩测：
# scratch git 仓合成 commit（--author 钉 claude-glm@local 同署面），
# CHECK_SEAT_RANGE=<merge>^..<merge> / HEAD^..HEAD 覆盖径逐场景断
# check_seat_writes.sh 进出码：
#   S1 [角色:施工] 标+glm 署+触非白名单=PASS（修前伪阳性红桩）；
#   S2 [角色:决策] 标+触非白名单=FAIL（T-0145 catch 必保）；
#   S3 无角色标+glm 署+触非白名单=FAIL（无头签名兜底仍在）；
#   S4 merge 支侧全施工标=PASS（支侧施工 commit 不再计入 allmine）；
#   S5 merge 支侧含决策标触码=FAIL（自写自合硬拒不动）。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
GATE="$ROOT/scripts/check_seat_writes.sh"
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
FAIL=0
ok()  { echo "  ✅ $1"; }
bad() { echo "  ❌ $1"; FAIL=1; }

REPO="$TMP/repo"; mkdir -p "$REPO"; cd "$REPO"
git init -q && git config user.email t@t && git config user.name t
DEF=$(git branch --show-current); [ -n "$DEF" ] || DEF=master
GLM='T <claude-glm@local>'

commit_as() { # $1=subj $2=author $3=file
  mkdir -p "$(dirname "$3")"; printf 'x\n' > "$3"
  git add "$3" && git commit -qm "$1" --author="$2"
}
run_gate() { # $1=range → stdout+rc
  set +e
  out=$(cd "$REPO" && CHECK_SEAT_RANGE="$1" bash "$GATE" 2>&1); rc=$?
  set -e
}
expect() { # $1=desc $2=want(rc0|rc1) $3=range
  run_gate "$3"
  case "$2:$rc" in
    rc0:0|rc1:1) ok "$1（rc=$rc）" ;;
    *) bad "$1 want=$2 got rc=$rc out=[$(echo "$out" | tail -2)]" ;;
  esac
}
merge_side() { # $1=branch $2=subj $3=author $4=file ——支侧单 commit 合回主干
  git checkout -qb "$1" "$DEF"
  commit_as "$2" "$3" "$4"
  git checkout -q "$DEF"
  git merge -q --no-ff "$1" -m "merge $1" >/dev/null
  git rev-parse HEAD
}

# ── 基件：base 提交垫父链 ──
commit_as 'chore: base' 'T <t@t>' README.md

# ══ S1 施工标+glm 署+触码面 → PASS ══
commit_as '[角色:施工][T-0001] probe' "$GLM" scripts/probe_s1.sh
expect 'S1 施工标触码面' rc0 'HEAD^..HEAD'

# ══ S2 决策标+触码面 → FAIL ══
commit_as '[角色:决策] probe' "$GLM" scripts/probe_s2.sh
expect 'S2 决策标触码面' rc1 'HEAD^..HEAD'

# ══ S3 无标+glm 署+触码面 → FAIL ══
commit_as 'plain probe no tag' "$GLM" scripts/probe_s3.sh
expect 'S3 无标glm触码面' rc1 'HEAD^..HEAD'

# ══ S4 merge 支侧=施工标 → PASS ══
M4=$(merge_side side_s4 '[角色:施工][T-0002] side work' "$GLM" internal/side_s4.go)
expect 'S4 merge支侧施工标' rc0 "$M4^..$M4"

# ══ S5 merge 支侧含决策标触码 → FAIL ══
M5=$(merge_side side_s5 '[角色:决策] side impl' "$GLM" scripts/probe_s5.sh)
expect 'S5 merge支侧决策标触码' rc1 "$M5^..$M5"

[ "$FAIL" = 0 ] && echo "✅ seat_writes_granularity_test 全场景过" || { echo "❌ 有红"; exit 1; }
