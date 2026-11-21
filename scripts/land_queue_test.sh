#!/usr/bin/env bash
# land_queue_test.sh —— 落地件桩测（工单#22577 验收①+2026-11-21滞留升件三件）：
# 桩 git（本地 bare origin+clone）+桩 psql（PATH 注入按词面应答）+
# 桩 check_all（绿=盖戳）——验三件齐→落；缺戳/缺复审过行/复审席=
# 作者席/冲突/戳陈 五反例各不落；docs-only 差分免 b 正例；
# depends_on 合流时序闸（工单#25800）：依赖未合拒/同支自带放/
# 已合放/无声明放 四态。
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
  *esc_scope=\'landq_starved\'*) echo "${T_STARVED_N:-0}";; # 节流计=20h内同scope升件行数(引号转义=字面)
esac
exit 0
EOF
chmod +x "$T/bin/psql"
cat > "$T/stub_check.sh" <<'EOF'
#!/usr/bin/env bash
# 桩合并点闸：绿=往 git-common-dir 盖戳（模拟 check_all.sh 末段）
# GO_PROBE 在场=记 command -v go 实解（PATH 自足闸断言面，工单#25880）
[ -n "${GO_PROBE:-}" ] && { command -v go > "$GO_PROBE" 2>/dev/null || true; } # tolerate: 桩探针行——go 缺席=空文件如实断言面，写败不翻测判
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
  mkdir -p "$d/o/scripts" "$d/o/migrations"
  cp "$HERE/mig_dep_lib.sh" "$d/o/scripts/mig_dep_lib.sh"
  printf -- '-- fin_0001_old 基线迁移（夹具）\nSELECT 1;\n' > "$d/o/migrations/fin_0001_old.sql"
  git -C "$d/o" add -A
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

# ═══ 依赖闸1：所依未合 → 跳支不落不烧账（工单#25800）═══
echo "== 依赖闸 依赖未合拒=="
new_case
(cd "$T/o" && git checkout -q -b fix/depmiss master \
  && mkdir -p migrations \
  && printf -- '-- depmiss 依赖未合拒例\n-- depends_on: fin_0999_ghost.sql\nSELECT 1;\n' > migrations/fin_0002_dep.sql \
  && git add -A && git commit -qm "wip depmiss" \
  && git push -qu origin fix/depmiss && git checkout -q master)
TIP=$(tip_of "$T/o" fix/depmiss); stamp "$T/o" "$TIP"
run_lq T_REVIEW_OK=1
if landed "$T/o" "$TIP"; then bad "依赖未合支竟落"; else ok "依赖未合不落"; fi
grep -q '落地败 支=' "$T/psql.log" && bad "依赖未合竟烧落地败账" || ok "依赖未合不占落地账"
grep -q "landq_starved.*fix/depmiss(依赖未合)" "$T/psql.log" && ok "依赖未合支入滞留集" || bad "依赖未合支静默滞留"

# ═══ 依赖闸2：同支自带 → 放落 ═══
echo "== 依赖闸 同支自带放=="
new_case
(cd "$T/o" && git checkout -q -b fix/depself master \
  && printf -- '-- depself 基迁\nSELECT 1;\n' > migrations/fin_0002_a.sql \
  && printf -- '-- depself 子迁\n-- depends_on: fin_0002_a.sql\nSELECT 1;\n' > migrations/fin_0003_b.sql \
  && git add -A && git commit -qm "wip depself" \
  && git push -qu origin fix/depself && git checkout -q master)
TIP=$(tip_of "$T/o" fix/depself); stamp "$T/o" "$TIP"
run_lq T_REVIEW_OK=1
if landed "$T/o" "$TIP"; then ok "同支自带依赖放落"; else bad "同支自带依赖未落"; fi

# ═══ 依赖闸3：所依已在 origin/master → 放落 ═══
echo "== 依赖闸 已合放=="
new_case
(cd "$T/o" && git checkout -q -b fix/depmerged master \
  && printf -- '-- depmerged 子迁\n-- depends_on: fin_0001_old.sql\nSELECT 1;\n' > migrations/fin_0002_new.sql \
  && git add -A && git commit -qm "wip depmerged" \
  && git push -qu origin fix/depmerged && git checkout -q master)
TIP=$(tip_of "$T/o" fix/depmerged); stamp "$T/o" "$TIP"
run_lq T_REVIEW_OK=1
if landed "$T/o" "$TIP"; then ok "已合依赖放落"; else bad "已合依赖未落"; fi

# ═══ 依赖闸4：迁移无声明 → 放落（渐进启用）═══
echo "== 依赖闸 无声明放=="
new_case
(cd "$T/o" && git checkout -q -b fix/deplain master \
  && printf -- '-- deplain 迁移\nSELECT 1;\n' > migrations/fin_0002_plain.sql \
  && git add -A && git commit -qm "wip deplain" \
  && git push -qu origin fix/deplain && git checkout -q master)
TIP=$(tip_of "$T/o" fix/deplain); stamp "$T/o" "$TIP"
run_lq T_REVIEW_OK=1
if landed "$T/o" "$TIP"; then ok "无声明迁移放落"; else bad "无声明迁移未落"; fi

# ═══ PATH 自足闸（工单#25880）：唤起面最小 PATH 无 go 径→land_queue 自补 ═══
echo "== PATH 自足闸 =="
new_case
(cd "$T/o" && git checkout -q -b fix/gopath master \
  && echo gopath >> f.txt && git add -A && git commit -qm "wip gopath" \
  && git push -qu origin fix/gopath && git checkout -q master \
  && echo master-moved > m2.txt && git add -A && git commit -qm "master 进位" \
  && git push -qu origin master)
# master 进位→合并树≠支尖树——快径跳不过，合并点 check_all 必走（桩记 go 探针）
TIP=$(tip_of "$T/o" fix/gopath); stamp "$T/o" "$TIP"
# env -i 最小集——模拟 transient systemd-run 径（无 /usr/local/go/bin、
# 无继承环境；go 真件在 /usr/local/go/bin 供自足环探得）
env -i PATH="$T/bin:/usr/bin:/bin" HOME="$HOME" \
  O_LAND_REPO="$T/o" O_LAND_DSN=stub O_LAND_PSQL="$T/bin/psql" \
  O_LAND_CHECK_ALL="bash $T/stub_check.sh" O_LAND_BOOTSTRAP=true \
  O_LAND_PUSH="git push" O_LAND_WT="$T/wt" O_LAND_LOCK="$T/lq.lock" \
  PSQL_LOG="$T/psql.log" GO_PROBE="$T/go.probe" T_REVIEW_OK=1 \
  bash "$LQ" >/dev/null 2>&1
if [ -s "$T/go.probe" ] && landed "$T/o" "$TIP"; then
  ok "最小PATH唤起面go自足+支落地"
else
  bad "go自足败 probe=[$(cat "$T/go.probe" 2>/dev/null)]"
fi

# ═══ 滞留支可见化（业主令2026-11-21断链根修）：缺戳/缺复审支→landq_starved升件 ═══
echo "== 滞留升件 =="
new_case
mk_branch "$T/o" fix/stranded feat_st2.txt st2
run_lq T_REVIEW_OK=1
grep -q "landq_starved.*fix/stranded" "$T/psql.log" && ok "缺戳滞留支升件名支在案" || bad "滞留支无升件"
# 节流：20h内同scope已有升件→不再发（T_STARVED_N=1 桩应答）
run_lq T_REVIEW_OK=1 T_STARVED_N=1
[ "$(grep -c '未并支滞留' "$T/psql.log")" -eq 1 ] && ok "节流20h内不复发" || bad "节流失效复发"

# ═══ 补扫：扫描面外支（无前缀推远端/仅本地未推）→ 同入滞留集 ═══
echo "== 面外支补扫 =="
new_case
mk_branch "$T/o" plainbranch pb.txt pb          # 无名单前缀但已推 origin
(cd "$T/o" && git checkout -q -b localonly master \
  && echo x > lo.txt && git add -A && git commit -qm wip \
  && git checkout -q master)                     # 仅本地未推
mk_branch "$T/o" fix/stranded2 st2.txt st2       # 名单内缺戳对照
run_lq T_REVIEW_OK=1
grep -q "landq_starved.*plainbranch(面外)" "$T/psql.log" && ok "无前缀远端支入滞留集" || bad "无前缀支漏报"
grep -q "landq_starved.*localonly(面外)" "$T/psql.log" && ok "仅本地支入滞留集" || bad "仅本地支漏报"
grep -q "landq_starved.*fix/stranded2" "$T/psql.log" && ok "名单内缺戳仍入集" || bad "名单内滞留丢"
grep -q "fix/stranded2(面外)" "$T/psql.log" && bad "候选跳支重列面外" || ok "候选跳支不重列面外"

# ═══ 补扫边界：在办支剔集/超窗前缀支/支名注入转义 ═══
echo "== 补扫边界 =="
new_case
(cd "$T/o" && git checkout -q -b wtactive master \
  && echo x > wa.txt && git add -A && git commit -qm wip \
  && git checkout -q master && git worktree add -q "$T/wtactive_wt" wtactive) # 工树在办支=非滞留
(cd "$T/o" && git checkout -q -b fix/oldbranch master \
  && echo x > ob.txt && git add -A \
  && GIT_COMMITTER_DATE='2020-01-01T00:00:00Z' git commit -qm wip --date='2020-01-01T00:00:00Z' \
  && git push -qu origin fix/oldbranch && git checkout -q master)              # 名单内但尖超7d窗
mk_branch "$T/o" "fix/ev'il" evil.txt ev                                       # 支名带单引号=注入面
run_lq T_REVIEW_OK=1
grep -q "landq_starved.*wtactive" "$T/psql.log" && bad "在办支误标滞留" || ok "在办支不入滞留集"
grep -q "landq_starved.*fix/oldbranch(面外)" "$T/psql.log" && ok "超窗前缀支入面外" || bad "超窗支漏报"
grep -q "fix/ev''il" "$T/psql.log" && ok "支名单引号sqlq转义" || bad "支名注入裸拼"
rm -rf "$T/wtactive_wt"; git -C "$T/o" worktree prune 2>/dev/null

# ═══ canon/* 收入名单:docs-only 免复审即自落（会话自然命名零改面） ═══
echo "== canon 支自落 =="
new_case
(cd "$T/o" && git checkout -q -b canon/kb-entry master \
  && mkdir -p docs && echo kb > docs/kb.md && git add -A && git commit -qm "canon entry" \
  && git push -qu origin canon/kb-entry && git checkout -q master)
TIP=$(tip_of "$T/o" canon/kb-entry); stamp "$T/o" "$TIP"
run_lq T_REVIEW_OK=0
if landed "$T/o" "$TIP"; then ok "canon支docs-only无复审行自落"; else bad "canon支未落"; fi

# ═══ 滞留不误报：三件齐支不入滞留集 ═══
new_case
mk_branch "$T/o" fix/landable feat_l.txt l
TIP=$(tip_of "$T/o" fix/landable); stamp "$T/o" "$TIP"
run_lq T_REVIEW_OK=1
grep -q "landq_starved.*fix/landable" "$T/psql.log" && bad "可落支误入滞留集" || ok "可落支不入滞留集"

echo "== 汇总: PASS=$PASS FAIL=$FAILN =="
[ "$FAILN" -eq 0 ]
