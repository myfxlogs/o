#!/usr/bin/env bash
# t0467_deploy_converge_test.sh —— T-0467 三面桩测：
#   A. mirror_push.sh 选择口径——最新鲜绿戳+ff可推+未合入 三断言
#      （HEAD 移动不影响选戳=跑步机根修实证；分叉/已合入/戳陈全灭
#      =镜面空窗合法）；
#   B. pre-push stamp_ok 读被推 sha 非 HEAD——戳在 P1、HEAD 已到
#      P2，推 P1:t/probe 过闸（件①机制实证）；
#   C. check_deploy_commit_seat.sh 部署树闸判件——施工前缀拒/
#      他标与无标放行/非部署树直通/MERGE_HEAD 豁免。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
FAIL=0
ok()  { echo "  ✅ $1"; }
bad() { echo "  ❌ $1"; FAIL=1; }

commit_as() { # $1=file $2=subj ——当前支落 commit（author=glm 同署面）
  mkdir -p "$(dirname "$1")"; printf 'x\n' > "$1"
  git add "$1" && git commit -qm "$2" --author='T <claude-glm@local>'
}
stamp() { printf 'ok\n' > ".git/check-green-$1"; }

# ══ A. mirror_push 选择 ══
REPO="$TMP/dt"; mkdir -p "$REPO"; cd "$REPO"
git init -q && git config user.email t@t && git config user.name t
DEF=$(git branch --show-current); [ -n "$DEF" ] || DEF=master
RROOT=$(realpath "$REPO")
PUSHLOG="$TMP/push.log"; : > "$PUSHLOG"
cat > "$TMP/pushstub.sh" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >> "$PUSHLOG"
exit 0
EOF
chmod +x "$TMP/pushstub.sh"
MP() { O_MIRROR_REPO="$REPO" O_MIRROR_PUSH_SH="$TMP/pushstub.sh" bash "$ROOT/scripts/mirror_push.sh"; }

commit_as README.md 'chore: base'
B0=$(git rev-parse HEAD)
git update-ref refs/remotes/origin/master "$B0"   # 无真 remote——fetch 败降级按本地 ref 判

commit_as docs/a.md '[角色:决策][T-1] c1' && C1=$(git rev-parse HEAD) && stamp "$C1"
commit_as docs/b.md 'untagged treadmill commit'                     # HEAD 移动=跑步机实证面（无戳）
commit_as docs/c.md '[角色:COS][T-2] c3' && C3=$(git rev-parse HEAD) && stamp "$C3"

# A1: 最新戳 C3 中选（HEAD 前序无戳 C2 不挡——戳绑 sha 非 HEAD）
MP >/dev/null 2>&1
[ "$(cat "$PUSHLOG")" = "origin $C3:master" ] \
  && ok "A1 最新绿戳 C3 中选（无戳 HEAD 前件不影响）" \
  || bad "A1 pushlog=[$(cat "$PUSHLOG")] want origin $C3:master"

# A2: C3 戳陈化（7h）→ 落 C1
: > "$PUSHLOG"; touch -d '7 hours ago' ".git/check-green-$C3"
MP >/dev/null 2>&1
[ "$(cat "$PUSHLOG")" = "origin $C1:master" ] \
  && ok "A2 戳龄>6h 跳过落 C1" \
  || bad "A2 pushlog=[$(cat "$PUSHLOG")] want origin $C1:master"
touch ".git/check-green-$C3"

# A3: origin/master 前进到 C3（C1/C3 俱已合入）→ 全灭空窗
: > "$PUSHLOG"; git update-ref refs/remotes/origin/master "$C3"
MP >"$TMP/mp3.out" 2>&1
{ [ ! -s "$PUSHLOG" ] && grep -q '空窗' "$TMP/mp3.out"; } \
  && ok "A3 全合入=镜面空窗零推送" \
  || bad "A3 空窗却推了: [$(cat "$PUSHLOG")]"

# A4: 分叉戳——side 支自 B0 分出，origin/master=C3 时 CS 非后代→跳过
: > "$PUSHLOG"
git checkout -qb sideA "$B0"; commit_as side/x.go '[角色:决策] side' && CS=$(git rev-parse HEAD) && stamp "$CS"
git checkout -q "$DEF"
MP >/dev/null 2>&1   # origin/master=C3：C1/C3 已合入跳、CS 非 ff 跳 → 空窗
{ [ ! -s "$PUSHLOG" ]; } && ok "A4 分叉戳（非 origin/master 后代）跳过=空窗" \
  || bad "A4 分叉戳竟推: [$(cat "$PUSHLOG")]"
# A5: origin/master 复位 B0 → CS 变可推且最新→中选（选戳不问支名）
git update-ref refs/remotes/origin/master "$B0"
MP >/dev/null 2>&1
[ "$(cat "$PUSHLOG")" = "origin $CS:master" ] \
  && ok "A5 远端回退后 CS 成最新可推戳中选" \
  || bad "A5 pushlog=[$(cat "$PUSHLOG")] want origin $CS:master"

# ══ B. pre-push 戳绑被推 sha（件①机制实证）══
cd "$TMP" && git init -q --bare origin.git
mkdir -p prec && cd prec && git init -q && git config user.email t@t && git config user.name t
git remote add origin "$TMP/origin.git"
mkdir -p scripts/hooks && cp "$ROOT/scripts/hooks/pre-push" scripts/hooks/pre-push && chmod +x scripts/hooks/pre-push
git config core.hooksPath scripts/hooks
printf 'x\n' > a.txt && git add a.txt && git commit -qm c1 && P1=$(git rev-parse HEAD)
printf 'ok\n' > ".git/check-green-$P1"
printf 'x\n' > b.txt && git add b.txt && git commit -qm 'c2 head moved'
set +e
git push origin "$P1:refs/heads/t/probe" >/dev/null 2>"$TMP/pp1.err"; rc1=$?
git push origin "HEAD:refs/heads/t/probe2" >/dev/null 2>"$TMP/pp2.err"; rc2=$?
set -e
[ "$rc1" = 0 ] && ok "B1 戳在 P1/HEAD 已走——推 P1:t/probe 过闸 rc=0" \
  || bad "B1 rc=$rc1 err=[$(tail -2 "$TMP/pp1.err")]"
[ "$rc2" != 0 ] && ok "B2 无戳 HEAD 推拒 rc=$rc2" || bad "B2 无戳竟过"

# ══ C. 部署树 commit 闸判件 ══
GATE="$ROOT/scripts/check_deploy_commit_seat.sh"
seat() { local r=0; bash "$GATE" "$@" >/dev/null 2>&1 || r=$?; echo "$r"; }
[ "$(seat "$REPO" master '[角色:施工][T-1] x' "$RROOT")" = 1 ] \
  && ok "C1 施工前缀部署树 master=拒" || bad "C1 施工前缀未拒"
[ "$(seat "$REPO" master '[角色:决策][T-1] x' "$RROOT")" = 0 ] \
  && ok "C2 决策标放行" || bad "C2 决策标被拒"
[ "$(seat "$REPO" master 'T-0440 untagged 票立' "$RROOT")" = 0 ] \
  && ok "C3 无标票立放行（DM流在案不误伤）" || bad "C3 无标被拒"
[ "$(seat "$REPO" t/x '[角色:施工] x' "$RROOT")" = 0 ] \
  && ok "C4 非 master 支直通" || bad "C4 支被拒"
[ "$(seat "$REPO" master '[角色:施工] x' /opt/o)" = 0 ] \
  && ok "C5 expected=/opt/o 对 scratch=直通（部署树限定）" || bad "C5 误伤"
# C6 MERGE_HEAD 在途豁免——手造 .git/MERGE_HEAD 实证判件豁免径
echo "$C1" > "$REPO/.git/MERGE_HEAD"
[ "$(seat "$REPO" master '[角色:施工][T-9] merge 中手滑' "$RROOT")" = 0 ] \
  && ok "C6 MERGE_HEAD 在途=豁免（合并中 commit 不拦）" || bad "C6 merge 在途被拒"
rm -f "$REPO/.git/MERGE_HEAD"

[ "$FAIL" = 0 ] && echo "✅ t0467_deploy_converge_test 全场景过" || { echo "❌ 有红"; exit 1; }
