#!/usr/bin/env bash
# land_queue_test.sh —— 落地件桩测（工单#22577 验收①）：
# 桩 git（本地 bare origin+clone）+桩 psql（PATH 注入按词面应答）+
# 桩 check_all（绿=盖戳）——验三件齐→落；缺戳/缺复审过行/复审席=
# 作者席/冲突/戳陈 五反例各不落；docs-only 差分免 b 正例。
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
LQ="$HERE/land_queue.sh"
T=$(mktemp -d /tmp/lq-test-XXXXXX)
trap 'rm -rf "$T"' EXIT
PASS=0; FAILN=0
ok()  { PASS=$((PASS+1)); echo "  ok   $1"; }
bad() { FAILN=$((FAILN+1)); echo "  FAIL $1"; }

# ── 桩件铺装 ──
mkdir -p "$T/bin" "$T/wt"
cat > "$T/bin/psql" <<'EOF'
#!/usr/bin/env bash
# psql 桩：按 -c SQL 词面应答，全部查询落 $PSQL_LOG 供断言
sql=""; prev=""
for a in "$@"; do [ "$prev" = "-c" ] && sql="$a"; prev="$a"; done
printf '%s\n' "$sql" >> "${PSQL_LOG:-/dev/null}"
case "$sql" in
  INSERT*) echo "INSERT 0 1";;
  *完工回执*) printf '%s\n' "${T_AUTHOR_SEAT:-施工席}";;
  *复审过*)
    # 模拟 seat<>作者席 谓词：同席→空集（分席不可证不落）
    [ "${T_REVIEW_OK:-0}" = 1 ] \
      && [ "${T_REVIEW_SEAT:-唤知}" != "${T_AUTHOR_SEAT:-施工席}" ] && echo 1;;
  *落地败*LIMIT*) [ "${T_FAILROW:-0}" = 1 ] && echo 1;;
esac
exit 0
EOF
chmod +x "$T/bin/psql"
cat > "$T/stub_check.sh" <<'EOF'
#!/usr/bin/env bash
# 桩合并点闸：绿=往 git-common-dir 盖戳（模拟 check_all.sh 末段）
[ "${STUB_CHECK_FAIL:-0}" = 1 ] && { echo "stub check_all red"; exit 1; }
gd=$(git rev-parse --path-format=absolute --git-common-dir) || exit 1
touch "$gd/check-green-$(git rev-parse HEAD)"
echo "stub check_all green"
EOF
chmod +x "$T/stub_check.sh"

# ── 夹具 ──
new_repo() { # $1=case目录 → bare origin + master 对齐的 clone
  local d=$1
  git init --bare -q "$d/origin.git"
  git init -q -b master "$d/o"
  git -C "$d/o" config user.email t@t
  git -C "$d/o" config user.name t
  git -C "$d/o" remote add origin "$d/origin.git"
  echo base > "$d/o/f.txt"
  git -C "$d/o" add f.txt
  git -C "$d/o" commit -qm init
  git -C "$d/o" push -qu origin master
}
mk_branch() { # $1=repo $2=支名 $3=文件名 $4=内容——从 master 切支推 origin 回 master
  (cd "$1" && git checkout -q -b "$2" master \
    && echo "$4" > "$3" && git add -A && git commit -qm "wip $2" \
    && git push -qu origin "$2" && git checkout -q master)
}
tip_of() { git -C "$1" rev-parse "$2"; }
stamp() { # $1=repo $2=sha [$3=age文字] —— 盖/陈绿戳
  local gd; gd=$(git -C "$1" rev-parse --path-format=absolute --git-common-dir)
  touch ${3:+-d "$3"} "$gd/check-green-$2"
}
run_lq() { # 余参=桩应答覆写（T_AUTHOR_SEAT/T_REVIEW_OK/T_REVIEW_SEAT/T_FAILROW/STUB_CHECK_FAIL）
  env PATH="$T/bin:$PATH" O_LAND_REPO="$T/o" O_LAND_DSN=stub \
    O_LAND_PSQL="$T/bin/psql" O_LAND_CHECK_ALL="bash $T/stub_check.sh" \
    O_LAND_BOOTSTRAP=true O_LAND_PUSH="git push" O_LAND_WT="$T/wt" \
    O_LAND_LOCK="$T/lq.lock" PSQL_LOG="$T/psql.log" "$@" \
    bash "$LQ" >/dev/null 2>&1
}
landed() { # $1=repo $2=tip → rc0=tip 已并入 origin/master
  git -C "$1" fetch -q origin 2>/dev/null
  git -C "$1" merge-base --is-ancestor "$2" origin/master 2>/dev/null
}
new_case() { # $1=名 → 重建 $T/o 夹具+清账
  rm -rf "$T/o" "$T/origin.git" "$T/wt"/* "$T/psql.log"
  mkdir -p "$T/wt"
  new_repo "$T"
}

# ═══ 正例：三件齐 → 落 ═══
echo "== 正例 三件齐→落 =="
new_case
mk_branch "$T/o" fix/aaa feat_aaa.txt aaa-content
TIP=$(tip_of "$T/o" fix/aaa); stamp "$T/o" "$TIP"
run_lq T_REVIEW_OK=1
if landed "$T/o" "$TIP" && grep -q '落成' "$T/psql.log"; then ok "三件齐落成"; else bad "三件齐未落"; fi

# ═══ 反例1：缺戳 → 不落 ═══
echo "== 反例 缺戳=="
new_case
mk_branch "$T/o" fix/nostamp feat_ns.txt ns
TIP=$(tip_of "$T/o" fix/nostamp)
run_lq T_REVIEW_OK=1
if landed "$T/o" "$TIP"; then bad "缺戳竟落"; else ok "缺戈不落"; fi

# ═══ 反例2：缺复审过行 → 不落 ═══
echo "== 反例 缺复审过行=="
new_case
mk_branch "$T/o" fix/norev feat_nr.txt nr
TIP=$(tip_of "$T/o" fix/norev); stamp "$T/o" "$TIP"
run_lq T_REVIEW_OK=0
if landed "$T/o" "$TIP"; then bad "缺复审过行竟落"; else ok "缺复审过行不落"; fi

# ═══ 反例3：复审席=作者席 → 不落 ═══
echo "== 反例 复审席=作者席=="
new_case
mk_branch "$T/o" fix/sameseat feat_ss.txt ss
TIP=$(tip_of "$T/o" fix/sameseat); stamp "$T/o" "$TIP"
run_lq T_REVIEW_OK=1 T_REVIEW_SEAT=施工席 T_AUTHOR_SEAT=施工席
if landed "$T/o" "$TIP"; then bad "复审席=作者席竟落"; else ok "分席不可证不落"; fi

# ═══ 反例4：冲突 → 不落+落地败行 ═══
echo "== 反例 冲突=="
new_case
# 支改 f.txt；master 侧同文件亦改 → merge-tree 冲突（同文件同行域）
(cd "$T/o" && git checkout -q -b fix/conflict master \
  && echo branch-side > f.txt && git add -A && git commit -qm "wip conflict" \
  && git push -qu origin fix/conflict && git checkout -q master)
TIP=$(tip_of "$T/o" fix/conflict); stamp "$T/o" "$TIP"
# master 侧改 f.txt 异内容→冲突；本地 master 推 origin 保持基线闸齐
echo master-side > "$T/o/f.txt"
git -C "$T/o" add f.txt && git -C "$T/o" commit -qm "master move" \
  && git -C "$T/o" push -qu origin master
run_lq T_REVIEW_OK=1
if landed "$T/o" "$TIP"; then bad "冲突支竟落"; else ok "冲突不落"; fi
grep -q '落地败.*冲突' "$T/psql.log" && ok "冲突落地败行在账" || bad "冲突无落地败行"

# ═══ 反例5：戳陈(>6h) → 不落 ═══
echo "== 反例 戳陈=="
new_case
mk_branch "$T/o" fix/stale feat_st.txt st
TIP=$(tip_of "$T/o" fix/stale); stamp "$T/o" "$TIP" "7 hours ago"
run_lq T_REVIEW_OK=1
if landed "$T/o" "$TIP"; then bad "陈戳竟落"; else ok "陈戳不落"; fi

# ═══ 正例2：docs-only 差分免复审行(b) ═══
echo "== 正例 docs-only免b=="
new_case
(cd "$T/o" && git checkout -q -b fix/docsonly master \
  && mkdir -p docs && echo doc > docs/x.md && git add -A && git commit -qm "doc" \
  && git push -qu origin fix/docsonly && git checkout -q master)
TIP=$(tip_of "$T/o" fix/docsonly); stamp "$T/o" "$TIP"
run_lq T_REVIEW_OK=0
if landed "$T/o" "$TIP"; then ok "docs-only免b落成"; else bad "docs-only未落"; fi

echo "== 汇总: PASS=$PASS FAIL=$FAILN =="
[ "$FAILN" -eq 0 ]
