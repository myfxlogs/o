#!/usr/bin/env bash
# land_queue_test.sh —— 落地件桩测（工单#22577 验收①+2026-11-21滞留升件三件）：
# 桩 git（本地 bare origin+clone）+桩 psql（PATH 注入按词面应答）+
# 桩 check_all（绿=盖戳）——验三件齐→落；缺戳/缺复审过行/复审席=
# 作者席/冲突/戳陈 五反例各不落；docs-only 差分免 b 正例；
# aseat 取行双键（工单O-27955）：复审词面不毒化/施工收口与完工类
# event 认回执/自审仍拒/缺回执 fail-closed；
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
# psql 桩：按 -c SQL 词面应答，全部查询落 $PSQL_LOG 供断言。
# T_ASEAT_ROWS 在场=按行集评 author-seat 查询（id seat event note TSV，
# 首配胜=ORDER BY id ASC LIMIT 1 同义——O-38132 取证修）；缺省回
# T_AUTHOR_SEAT 旧径。
sql=""; prev=""
for a in "$@"; do [ "$prev" = "-c" ] && sql="$a"; prev="$a"; done
printf '%s\n' "$sql" >> "${PSQL_LOG:-/dev/null}"
if [[ "$sql" == *"SELECT seat FROM ops_event"* && "$sql" == *"seat NOT IN"* ]]; then
  seat=""
  if [ -n "${T_ASEAT_ROWS:-}" ]; then
    if [ -f "$T_ASEAT_ROWS" ]; then
      while IFS=$'\t' read -r rid rseat revent rnote; do
        [ -n "$rid" ] || continue
        [ -n "$seat" ] && break # ASC LIMIT 1——首配即取
        receipt=0
        [[ "$revent" == *完工* || "$revent" == *候审* || "$revent" == *施工收口* || "$revent" == *回执* ]] && receipt=1
        [[ "$rnote" =~ ^.{0,40}(完工回执|完工推支候审|候审回执|施工收口) ]] && receipt=1
        [[ "$revent" == "处置" && "$rnote" =~ ^O-[0-9]+[^@]*@[0-9a-f]{9,40} ]] && receipt=1
        [ "$receipt" = 1 ] || continue
        [[ "$rnote" =~ ^复审过[[:space:]]*sha= || "$rseat" == "复审席" || "$rseat" == "落地件" ]] && continue
        match=0
        if [ -z "${T_BRANCH:-}" ] && [ -z "${T_TIP9:-}" ]; then match=1; else
          [ -n "${T_BRANCH:-}" ] && [[ "$rnote" == *"$T_BRANCH"* ]] && match=1
          [ -n "${T_TIP9:-}" ] && [[ "$rnote" == *"$T_TIP9"* ]] && match=1
        fi
        [ "$match" = 1 ] && seat="$rseat"
      done < "$T_ASEAT_ROWS"
    fi
  else
    seat="${T_AUTHOR_SEAT:-施工席}"
  fi
  [ -n "$seat" ] && printf '%s\n' "$seat" > "${T_ASEAT_STATE:-/dev/null}"
  [ -n "$seat" ] && printf '%s\n' "$seat"
  exit 0
fi
case "$sql" in
  *ops_ticket_desk*unnest*|*unnest*ops_ticket_desk*)
    # 落成自动销票 desk∧回执闸桩（工单O-33182）：T_DESK_ROWS 行形
    # tid\tclosed(t/f/'')\tsusp(t/f)\trulable(t/f)\treceipt(t/f)——
    # 同 SQL 闸语义评估：未销∧未挂起∧非裁定∧有完工回执→返 tid
    if [ -n "${T_DESK_ROWS:-}" ] && [ -f "$T_DESK_ROWS" ]; then
      _tids=$(printf '%s' "$sql" | grep -oE 'ARRAY\[[0-9,]+\]' | grep -oE '[0-9]+')
      while IFS=$'\t' read -r dtid dclosed dsusp drulable drcpt; do
        [ -n "$dtid" ] || continue
        printf '%s\n' "$_tids" | grep -qx "$dtid" || continue
        [ "$dclosed" = "t" ] && continue
        [ "$dsusp" = "t" ] && continue
        [ "$drulable" = "t" ] && continue
        [ "$drcpt" != "t" ] && continue
        echo "$dtid"
      done < "$T_DESK_ROWS"
    fi;;
  *落地件*处置*)
    # 落成自动销销行拒写桩（工单O-33705）：T_CLOSE_REJECT_TIDS=逗号
    # 集——note 提 工单#<tid> 命中集→exit1 模拟 semgate 拒写；缺省
    # 全放行 echo INSERT 0 1（与 INSERT* 桩同义）。
    if [ -n "${T_CLOSE_REJECT_TIDS:-}" ]; then
      _ctid=$(printf '%s' "$sql" | grep -oE '工单#[0-9]+' | grep -oE '[0-9]+' | head -1)
      case ",$T_CLOSE_REJECT_TIDS," in *,"$_ctid",*) exit 1;; esac
    fi
    echo "INSERT 0 1";;
  *ops_esc\(*) echo "${T_ESC_ID:-900001}";; # ops_esc() 升件函件桩——返假 id
  INSERT*) echo "INSERT 0 1";;
  *复审过*)
    # 模拟 seat<>作者席 谓词：同席→空集（分席不可证不落）
    author="${T_AUTHOR_SEAT:-施工席}"
    [ -n "${T_ASEAT_STATE:-}" ] && [ -s "$T_ASEAT_STATE" ] && author=$(cat "$T_ASEAT_STATE")
    [ "${T_REVIEW_OK:-0}" = 1 ] \
      && [ "${T_REVIEW_SEAT:-唤知}" != "$author" ] && echo 1;;
  *落地败*LIMIT*) [ "${T_FAILROW:-0}" = 1 ] && echo 1;;
  *SELECT*note*ops_event*landq_starved*) echo "${T_STARVED_NOTE:-}";;
  *EXTRACT*landq_starved*) echo "${T_STARVED_AGE:-99999}";;
  *count*esc_scope*|*esc_scope*count*) echo "${T_ESC_ROWS:-0}";; # esc()去重计数桩——0=无在案升件→INSERT放行（列车冲突/合红升件断言面）
esac
exit 0
EOF
chmod +x "$T/bin/psql"
cat > "$T/stub_check.sh" <<'EOF'
#!/usr/bin/env bash
# 桩合并点闸：绿=往 git-common-dir 盖戳（模拟 check_all.sh 末段）
# GO_PROBE 在场=记 command -v go 实解（PATH 自足闸断言面，工单#25880）
[ -n "${GO_PROBE:-}" ] && { command -v go > "$GO_PROBE" 2>/dev/null || true; } # tolerate: 桩探针行——go 缺席=空文件如实断言面，写败不翻测判
[ -n "${CHECK_COUNT_FILE:-}" ] && echo run >> "$CHECK_COUNT_FILE" # tolerate: 计数面写败不翻测判——列车「一次验」断言用
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
run_lq() { # 余参=桩应答覆写（T_AUTHOR_SEAT/T_REVIEW_OK/T_REVIEW_SEAT/T_FAILROW/STUB_CHECK_FAIL/T_ASEAT_ROWS/T_BRANCH/T_TIP9）
  env PATH="$T/bin:$PATH" O_LAND_REPO="$T/o" O_LAND_DSN=stub \
    O_LAND_PSQL="$T/bin/psql" O_LAND_CHECK_ALL="bash $T/stub_check.sh" \
    O_LAND_BOOTSTRAP=true O_LAND_PUSH="git push" O_LAND_WT="$T/wt" \
    O_LAND_LOCK="$T/lq.lock" PSQL_LOG="$T/psql.log" T_ASEAT_STATE="$T/aseat.state" "$@" \
    bash "$LQ" >/dev/null 2>&1
}
landed() { # $1=repo $2=tip → rc0=tip 已并入 origin/master
  git -C "$1" fetch -q origin 2>/dev/null
  git -C "$1" merge-base --is-ancestor "$2" origin/master 2>/dev/null
}
new_case() { # $1=名 → 重建 $T/o 夹具+清账
  rm -rf "$T/o" "$T/origin.git" "$T/wt"/* "$T/psql.log" "$T/aseat.state" "$T/aseat_rows.tsv"
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

# ═══ aseat 取行双键（工单O-27955）：复审词面不毒化/施工收口认回执/缺回执fail-closed ═══
echo "== aseat 复审词面不误取 =="
new_case
mk_branch "$T/o" fix/aseat-poison feat_ap.txt ap
TIP=$(tip_of "$T/o" fix/aseat-poison); stamp "$T/o" "$TIP"
: > "$T/aseat_rows.tsv"
printf '%s\t%s\t%s\t%s\n' 1 施工席 记事 "完工回执——fix/aseat-poison@${TIP:0:9} 候审" >> "$T/aseat_rows.tsv"
printf '%s\t%s\t%s\t%s\n' 2 复审席 复审过 "复审过 sha=$TIP fix/aseat-poison 三维——候审回执在案" >> "$T/aseat_rows.tsv"
run_lq T_ASEAT_ROWS="$T/aseat_rows.tsv" T_BRANCH=fix/aseat-poison T_TIP9="${TIP:0:9}" T_REVIEW_OK=1 T_REVIEW_SEAT=复审席 T_AUTHOR_SEAT=施工席
if landed "$T/o" "$TIP"; then ok "复审行候审回执词面不误取aseat"; else bad "复审行词面仍毒化aseat"; fi

echo "== aseat 施工收口认回执 =="
new_case
mk_branch "$T/o" fix/aseat-closeout feat_ac.txt ac
TIP=$(tip_of "$T/o" fix/aseat-closeout); stamp "$T/o" "$TIP"
: > "$T/aseat_rows.tsv"
printf '%s\t%s\t%s\t%s\n' 1 施工席 记事 "[施工收口:O-27955] @ sha=$TIP 支=fix/aseat-closeout 已推 origin" >> "$T/aseat_rows.tsv"
run_lq T_ASEAT_ROWS="$T/aseat_rows.tsv" T_BRANCH=fix/aseat-closeout T_TIP9="${TIP:0:9}" T_REVIEW_OK=1 T_REVIEW_SEAT=唤知 T_AUTHOR_SEAT=施工席
if landed "$T/o" "$TIP"; then ok "施工收口行认作回执"; else bad "施工收口行未认回执"; fi

echo "== aseat event完工类认回执 =="
new_case
mk_branch "$T/o" fix/aseat-event feat_ae.txt ae
TIP=$(tip_of "$T/o" fix/aseat-event); stamp "$T/o" "$TIP"
: > "$T/aseat_rows.tsv"
printf '%s\t%s\t%s\t%s\n' 1 外席 完工回执 "O-25563 支=fix/aseat-event@${TIP:0:9} 已推 origin" >> "$T/aseat_rows.tsv"
run_lq T_ASEAT_ROWS="$T/aseat_rows.tsv" T_BRANCH=fix/aseat-event T_TIP9="${TIP:0:9}" T_REVIEW_OK=1 T_REVIEW_SEAT=复审席 T_AUTHOR_SEAT=外席
if landed "$T/o" "$TIP"; then ok "event完工类行认回执且外席可作作者"; else bad "event完工类行/外席未认回执"; fi

echo "== aseat 外派处置取证形认回执（O-38751）=="
new_case
mk_branch "$T/o" fix/aseat-foreign feat_af.txt af
TIP=$(tip_of "$T/o" fix/aseat-foreign); stamp "$T/o" "$TIP"
: > "$T/aseat_rows.tsv"
printf '%s\t%s\t%s\t%s\n' 1 施工席 处置 "O-38751 外派票根修@${TIP:0:9}：取证定案=回执谓词缺处置形" >> "$T/aseat_rows.tsv"
run_lq T_ASEAT_ROWS="$T/aseat_rows.tsv" T_BRANCH=fix/aseat-foreign T_TIP9="${TIP:0:9}" T_REVIEW_OK=1 T_REVIEW_SEAT=复审席 T_AUTHOR_SEAT=施工席
if landed "$T/o" "$TIP"; then ok "处置O票@sha形认回执aseat"; else bad "处置取证形未认回执——38751病复"; fi

echo "== aseat 处置非宣告形仍拒（负锚）=="
new_case
mk_branch "$T/o" fix/aseat-nondecl feat_an.txt an
TIP=$(tip_of "$T/o" fix/aseat-nondecl); stamp "$T/o" "$TIP"
: > "$T/aseat_rows.tsv"
printf '%s\t%s\t%s\t%s\n' 1 施工席 处置 "升件销——fix/aseat-nondecl@${TIP:0:9} 处置顺带提及支尖" >> "$T/aseat_rows.tsv"
run_lq T_ASEAT_ROWS="$T/aseat_rows.tsv" T_BRANCH=fix/aseat-nondecl T_TIP9="${TIP:0:9}" T_REVIEW_OK=1 T_REVIEW_SEAT=唤知 T_AUTHOR_SEAT=施工席
if landed "$T/o" "$TIP"; then bad "处置非宣告形竟充回执"; else ok "处置非宣告形仍fail-closed"; fi

echo "== aseat 自审仍拒 =="
new_case
mk_branch "$T/o" fix/aseat-selfreview feat_as.txt as
TIP=$(tip_of "$T/o" fix/aseat-selfreview); stamp "$T/o" "$TIP"
: > "$T/aseat_rows.tsv"
printf '%s\t%s\t%s\t%s\n' 1 施工席 处置 "完工回执——fix/aseat-selfreview@${TIP:0:9} 候审" >> "$T/aseat_rows.tsv"
run_lq T_ASEAT_ROWS="$T/aseat_rows.tsv" T_BRANCH=fix/aseat-selfreview T_TIP9="${TIP:0:9}" T_REVIEW_OK=1 T_REVIEW_SEAT=施工席 T_AUTHOR_SEAT=施工席
if landed "$T/o" "$TIP"; then bad "复审席=作者席竟落"; else ok "复审席=作者席仍拒"; fi

echo "== aseat 回执行缺fail-closed =="
new_case
mk_branch "$T/o" fix/aseat-noreceipt feat_an.txt an
TIP=$(tip_of "$T/o" fix/aseat-noreceipt); stamp "$T/o" "$TIP"
: > "$T/aseat_rows.tsv"
printf '%s\t%s\t%s\t%s\n' 1 复审席 复审过 "复审过 sha=$TIP fix/aseat-noreceipt 候审回执在案" >> "$T/aseat_rows.tsv"
run_lq T_ASEAT_ROWS="$T/aseat_rows.tsv" T_BRANCH=fix/aseat-noreceipt T_TIP9="${TIP:0:9}" T_REVIEW_OK=1 T_REVIEW_SEAT=唤知 T_AUTHOR_SEAT=施工席
if landed "$T/o" "$TIP"; then bad "回执行缺竟落"; else ok "回执行缺aseat空仍fail-closed"; fi

# ═══ O-38132：元行引契词不充回执/取行序首配/异形交付行不参选 ═══
echo "== aseat 元行引述契词不充回执 =="
new_case
mk_branch "$T/o" fix/aseat-meta feat_am.txt am
TIP=$(tip_of "$T/o" fix/aseat-meta); stamp "$T/o" "$TIP"
: > "$T/aseat_rows.tsv"
# #38132/#38213 实证形：诊断票/会话叙事引契词+支名在行中（>40 字符位）
printf '%s\t%s\t%s\t%s\n' 1 处置席 工单 "工单/landq-receipt-lexeme-gap: fix/aseat-meta land_queue 永跳——施工席#1回执用「交付@sha」非契词(候审回执|施工收口),review_ok作者席不可证rc3 fail-closed;戳鲜48min+复审过sha=${TIP:0:9}均在" >> "$T/aseat_rows.tsv"
printf '%s\t%s\t%s\t%s\n' 2 唤知 会话收口 "unit=o-cognition-1 第N轮:fix/aseat-meta契词缝诊——票landq-receipt-lexeme-gap P2立 用帧=无 自问=无 手感=landq-aseat:诊断票引契词+支名会被闸当作者回执(施工收口契词)" >> "$T/aseat_rows.tsv"
run_lq T_ASEAT_ROWS="$T/aseat_rows.tsv" T_BRANCH=fix/aseat-meta T_TIP9="${TIP:0:9}" T_REVIEW_OK=1 T_REVIEW_SEAT=唤知 T_AUTHOR_SEAT=施工席
if landed "$T/o" "$TIP"; then bad "元行引契词竟充回执落成"; else ok "元行引述契词不充回执(aseat空fail-closed)"; fi

echo "== aseat 元行不夺真回执(首配胜) =="
new_case
mk_branch "$T/o" fix/aseat-meta2 feat_am2.txt am2
TIP=$(tip_of "$T/o" fix/aseat-meta2); stamp "$T/o" "$TIP"
: > "$T/aseat_rows.tsv"
# 真回执在前(首配)、元行在后——旧 DESC 序会被元行夺 aseat=唤知致自审误拒
printf '%s\t%s\t%s\t%s\n' 1 施工席 施工收口 "[施工收口:O-38132] @ sha=$TIP 支=fix/aseat-meta2 已推 origin" >> "$T/aseat_rows.tsv"
printf '%s\t%s\t%s\t%s\n' 2 唤知 会话收口 "unit=o-cognition-2 第N轮:fix/aseat-meta2契词缝复诊——施工收口契词票待裁 用帧=无" >> "$T/aseat_rows.tsv"
run_lq T_ASEAT_ROWS="$T/aseat_rows.tsv" T_BRANCH=fix/aseat-meta2 T_TIP9="${TIP:0:9}" T_REVIEW_OK=1 T_REVIEW_SEAT=唤知 T_AUTHOR_SEAT=施工席
if landed "$T/o" "$TIP"; then ok "真回执首配胜——元行不夺aseat"; else bad "元行仍夺aseat致自审误拒"; fi

echo "== aseat 取行序取最早契词行 =="
new_case
mk_branch "$T/o" fix/aseat-first feat_af.txt af
TIP=$(tip_of "$T/o" fix/aseat-first); stamp "$T/o" "$TIP"
: > "$T/aseat_rows.tsv"
# 双回执异席：早=外席完工回执(首报工)，晚=施工席施工收口(补行)——
# 回执语义取最早→aseat=外席；复审席=外席=自审拒；复审席=唤知放行
printf '%s\t%s\t%s\t%s\n' 1 外席 完工回执 "完工回执 fix/aseat-first @ ${TIP:0:9}——首报工" >> "$T/aseat_rows.tsv"
printf '%s\t%s\t%s\t%s\n' 2 施工席 施工收口 "[施工收口:O-99999] @ sha=$TIP 支=fix/aseat-first 补行" >> "$T/aseat_rows.tsv"
run_lq T_ASEAT_ROWS="$T/aseat_rows.tsv" T_BRANCH=fix/aseat-first T_TIP9="${TIP:0:9}" T_REVIEW_OK=1 T_REVIEW_SEAT=外席 T_AUTHOR_SEAT=施工席
if landed "$T/o" "$TIP"; then bad "复审席=最早作者席竟落(取行序未取最早)"; else ok "取最早契词行——复审席=外席自审拒"; fi
run_lq T_ASEAT_ROWS="$T/aseat_rows.tsv" T_BRANCH=fix/aseat-first T_TIP9="${TIP:0:9}" T_REVIEW_OK=1 T_REVIEW_SEAT=唤知 T_AUTHOR_SEAT=施工席
if landed "$T/o" "$TIP"; then ok "复审席=唤知异席放行(aseat=外席)"; else bad "异席复审未放行"; fi

echo "== aseat 处置O票@sha形参选（O-38751翻案#38000）=="
new_case
mk_branch "$T/o" fix/aseat-nonlex feat_anl.txt anl
TIP=$(tip_of "$T/o" fix/aseat-nonlex); stamp "$T/o" "$TIP"
: > "$T/aseat_rows.tsv"
# #38000 曾锚「交付@sha非契词不充回执」——O-38751翻案：行首锚
# O-票…@hex=交付宣告形同格认（容错兜），fail-closed 只守非宣告形
printf '%s\t%s\t%s\t%s\n' 1 施工席 处置 "O-99999 venue面交付@${TIP:0:9}：①面A+面B 桩测绿戳在案。支fix/aseat-nonlex已推" >> "$T/aseat_rows.tsv"
run_lq T_ASEAT_ROWS="$T/aseat_rows.tsv" T_BRANCH=fix/aseat-nonlex T_TIP9="${TIP:0:9}" T_REVIEW_OK=1 T_REVIEW_SEAT=唤知 T_AUTHOR_SEAT=施工席
if landed "$T/o" "$TIP"; then ok "处置O票@sha形充回执(38751容错兜)"; else bad "宣告形处置回执仍拒——38751病复"; fi

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
# 节流：20h内同scope已有升件覆盖同集→不再发（桩回最近升件note+龄）
run_lq T_REVIEW_OK=1 T_STARVED_NOTE='landq_starved fix/stranded' T_STARVED_AGE=60
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

# ═══ 列车①：3 支齐→一列一次验一次推（工单O-32267 spec①）═══
echo "== 列车 三支齐一次落 =="
new_case
# 显式 committerdate 保列序确定性：ta<tb<tc
mk_branch_d() { # $1=repo $2=支名 $3=文件名 $4=内容 $5=ISO日期
  (cd "$1" && git checkout -q -b "$2" master \
    && echo "$4" > "$3" && git add -A \
    && GIT_COMMITTER_DATE="$5" git commit -qm "wip $2" --date="$5" \
    && git push -qu origin "$2" && git checkout -q master)
}
mk_branch_d "$T/o" fix/ta ta.txt ta-content "$(date -u -d '3 hours ago' +%Y-%m-%dT%H:%M:%SZ)"
mk_branch_d "$T/o" fix/tb tb.txt tb-content "$(date -u -d '2 hours ago' +%Y-%m-%dT%H:%M:%SZ)"
mk_branch_d "$T/o" fix/tc tc.txt tc-content "$(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ)"
TIPA=$(tip_of "$T/o" fix/ta); TIPB=$(tip_of "$T/o" fix/tb); TIPC=$(tip_of "$T/o" fix/tc)
stamp "$T/o" "$TIPA"; stamp "$T/o" "$TIPB"; stamp "$T/o" "$TIPC"
cat > "$T/count_push.sh" <<'EOF'
#!/usr/bin/env bash
echo push >> "${PUSH_COUNT_FILE:-/dev/null}" # tolerate: 计数面写败不翻测判
exec git "$@"
EOF
chmod +x "$T/count_push.sh"
rm -f "$T/check.count" "$T/push.count"
run_lq T_REVIEW_OK=1 CHECK_COUNT_FILE="$T/check.count" PUSH_COUNT_FILE="$T/push.count" O_LAND_PUSH="bash $T/count_push.sh push"
if landed "$T/o" "$TIPA" && landed "$T/o" "$TIPB" && landed "$T/o" "$TIPC"; then ok "列车三支全落"; else bad "列车三支未全落"; fi
[ "$(grep -c . "$T/check.count" 2>/dev/null || echo 0)" -eq 1 ] && ok "列车合并点验一次" || bad "列车验次数=$(cat "$T/check.count" 2>/dev/null | wc -l)"
[ "$(grep -c . "$T/push.count" 2>/dev/null || echo 0)" -eq 1 ] && ok "列车推送一次" || bad "列车推送次数>1"
[ "$(grep -c '落成 支=' "$T/psql.log")" -eq 3 ] && ok "列车三支落成行逐支在账" || bad "列车落成行数=$(grep -c '落成 支=' "$T/psql.log")"

# ═══ 列车②：中支冲突→剔除续列（工单O-32267 spec②）═══
echo "== 列车 中支冲突剔除续列 =="
new_case
# ta/tb 同改 f.txt 异内容（互不相让）；tc 独占文件——tb 对 origin/master
# 预检净合（master 未动）入列，合并时撞 ta 已合内容=列内互冲剔除
mk_branch_d "$T/o" fix/ta f.txt ta-side "$(date -u -d '3 hours ago' +%Y-%m-%dT%H:%M:%SZ)"
mk_branch_d "$T/o" fix/tb f.txt tb-side "$(date -u -d '2 hours ago' +%Y-%m-%dT%H:%M:%SZ)"
mk_branch_d "$T/o" fix/tc tc2.txt tc2-content "$(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ)"
TIPA=$(tip_of "$T/o" fix/ta); TIPB=$(tip_of "$T/o" fix/tb); TIPC=$(tip_of "$T/o" fix/tc)
stamp "$T/o" "$TIPA"; stamp "$T/o" "$TIPB"; stamp "$T/o" "$TIPC"
rm -f "$T/check.count" "$T/push.count"
run_lq T_REVIEW_OK=1 CHECK_COUNT_FILE="$T/check.count" PUSH_COUNT_FILE="$T/push.count" O_LAND_PUSH="bash $T/count_push.sh push"
if landed "$T/o" "$TIPA" && landed "$T/o" "$TIPC" && ! landed "$T/o" "$TIPB"; then ok "冲突支剔除余支续列"; else bad "剔除续列判错 ta=$(landed "$T/o" "$TIPA" && echo 1) tb=$(landed "$T/o" "$TIPB" && echo 1) tc=$(landed "$T/o" "$TIPC" && echo 1)"; fi
grep -q 'landq_merge_conflict.*fix/tb\|landq_merge_conflict_fix_tb' "$T/psql.log" && ok "冲突支升件在案" || bad "冲突支无升件"
grep -q '落地败 支=fix/tb' "$T/psql.log" && ok "冲突支落地败行在账" || bad "冲突支无落地败行"
[ "$(grep -c . "$T/check.count" 2>/dev/null || echo 0)" -eq 1 ] && ok "剔后列车仍一次验" || bad "剔后验次数异常"

# ═══ 列车③：全冲突→全剔升件（工单O-32267 spec③）═══
echo "== 列车 全冲突全剔 =="
new_case
# 两支均改 f.txt；master 侧亦改 f.txt 并推——两支对新 origin/master
# 预检皆冲突=扫描期全剔（合并时全剔需竞动注入，首支对净合基线必合
# 故全剔实径=预检径——同 landq_merge_conflict 升件面）
mk_branch_d "$T/o" fix/ca f.txt ca-side "$(date -u -d '3 hours ago' +%Y-%m-%dT%H:%M:%SZ)"
mk_branch_d "$T/o" fix/cb f.txt cb-side "$(date -u -d '2 hours ago' +%Y-%m-%dT%H:%M:%SZ)"
TIPCA=$(tip_of "$T/o" fix/ca); TIPCB=$(tip_of "$T/o" fix/cb)
stamp "$T/o" "$TIPCA"; stamp "$T/o" "$TIPCB"
echo master-moved > "$T/o/f.txt"
git -C "$T/o" add f.txt && git -C "$T/o" commit -qm "master move" && git -C "$T/o" push -qu origin master
rm -f "$T/check.count" "$T/push.count"
run_lq T_REVIEW_OK=1 CHECK_COUNT_FILE="$T/check.count" PUSH_COUNT_FILE="$T/push.count" O_LAND_PUSH="bash $T/count_push.sh push"
if ! landed "$T/o" "$TIPCA" && ! landed "$T/o" "$TIPCB"; then ok "全冲突支皆不落"; else bad "全冲突支竟落"; fi
[ "$(grep -c 'landq_merge_conflict' "$T/psql.log")" -ge 2 ] && ok "全冲突支逐支升件" || bad "全冲突升件数不足"
[ ! -s "$T/check.count" ] && [ ! -s "$T/push.count" ] && ok "全剔零验零推" || bad "全剔仍验/推"
[ "$(grep -c '落成 支=' "$T/psql.log")" -eq 0 ] && ok "全剔零落成行" || bad "全剔有落成行"

# ═══ 列车④：单支→退化为现行一支一落（工单O-32267 spec④=既有全例同径）═══

# ═══ 落成自动销票细胞（工单O-33182）：三源提号→desk∧回执双闸→落地件销行 ═══
echo "== 落成自动销票 =="
new_case
mk_branch_msg() { # $1=repo $2=支名 $3=文件名 $4=提交题（载票号面）
  (cd "$1" && git checkout -q -b "$2" master \
    && echo x > "$3" && git add -A && git commit -qm "$4" \
    && git push -qu origin "$2" && git checkout -q master)
}
# 案①commit面双源 O-N+工单#N / 案②支名面嵌号 / 案③四拒形(无回执·已销·挂起·裁定) / 案④跨支同号去重
mk_branch_msg "$T/o" fix/closereceipt feat_cr.txt "wip 工单O-88001 与 工单#88002 双源"
mk_branch_msg "$T/o" fix/O-88003-named feat_cn.txt "wip 支名嵌号"
mk_branch_msg "$T/o" fix/closereject feat_cj.txt "wip refs O-88004 O-88005 O-88006 O-88007"
mk_branch_msg "$T/o" fix/closededup feat_cd.txt "wip 重复 O-88001"
for b in fix/closereceipt fix/O-88003-named fix/closereject fix/closededup; do
  _t=$(tip_of "$T/o" "$b"); stamp "$T/o" "$_t"
done
: > "$T/desk_rows.tsv"
printf '88001\tf\tf\tf\tt\n88002\tf\tf\tf\tt\n88003\tf\tf\tf\tt\n88004\tf\tf\tf\tf\n88005\tt\tf\tf\tt\n88006\tf\tt\tf\tt\n88007\tf\tf\tt\tt\n' >> "$T/desk_rows.tsv"
rm -f "$T/psql.log"
run_lq T_REVIEW_OK=1 T_DESK_ROWS="$T/desk_rows.tsv"
[ "$(grep -c "工单#88001 销" "$T/psql.log")" -eq 1 ] && ok "O-N源销行且跨支去重一行" || bad "88001销行数=$(grep -c '工单#88001 销' "$T/psql.log")"
[ "$(grep -c "工单#88002 销" "$T/psql.log")" -eq 1 ] && ok "工单#N源销行" || bad "88002销行数=$(grep -c '工单#88002 销' "$T/psql.log")"
[ "$(grep -c "工单#88003 销" "$T/psql.log")" -eq 1 ] && ok "支名面嵌号销行" || bad "88003销行数=$(grep -c '工单#88003 销' "$T/psql.log")"
_rej=1
for badt in 88004 88005 88006 88007; do
  [ "$(grep -c "工单#$badt 销" "$T/psql.log")" -eq 0 ] || { bad "拒形票$badt竟销"; _rej=0; }
done
[ "$_rej" = 1 ] && ok "无回执/已销/挂起/裁定四拒形零销行"
[ "$(grep -c "VALUES('落地件','处置','工单'" "$T/psql.log")" -eq 3 ] && ok "销行seat=落地件/event=处置/ticket=工单三列形" || bad "销行列形数=$(grep -c "落地件','处置','工单'" "$T/psql.log")"
grep -q "销——落成自动销 支=fix/closereceipt merge=" "$T/psql.log" && ok "销行note形=工单#N销——落成自动销" || bad "销行note形不符"
grep -q "落成自动销 支=" "$T/psql.log" && ok "落成自动销簿记行在账" || bad "无自动销簿记行"

# ═══ 销行被拒（工单O-33705 spec④）：DID 成功才追加+写败落升件+csv 不含败 tid ═══
echo "== 销行被拒不吞 =="
new_case
mk_branch_msg "$T/o" fix/autoclose-reject feat_ar.txt "wip 工单#9101 与 工单#9102 双票"
TIP=$(tip_of "$T/o" fix/autoclose-reject); stamp "$T/o" "$TIP"
: > "$T/desk_rows.tsv"
printf '9101\tf\tf\tf\tt\n9102\tf\tf\tf\tt\n' >> "$T/desk_rows.tsv"
rm -f "$T/psql.log"
run_lq T_REVIEW_OK=1 T_DESK_ROWS="$T/desk_rows.tsv" T_CLOSE_REJECT_TIDS=9102
# 两票销行 INSERT 均尝试（账面可见）
[ "$(grep -c "工单#9101 销" "$T/psql.log")" -eq 1 ] && [ "$(grep -c "工单#9102 销" "$T/psql.log")" -eq 1 ] \
	&& ok "双票销行均尝试在账" || bad "销行尝试缺: $(grep -c '工单#91' "$T/psql.log")"
# 汇总 csv 只含真销成 tid——9101 在、9102 不在
_sumcsv=$(grep '落成自动销 支=' "$T/psql.log" | grep -oE '票=[0-9,]+' | head -1)
case "$_sumcsv" in
	*9101*) ok "汇总csv含真销成9101" ;;
	*) bad "汇总csv缺9101: $_sumcsv" ;;
esac
case "$_sumcsv" in
	*9102*) bad "汇总csv混入写败9102: $_sumcsv" ;;
	*) ok "汇总csv不含写败9102" ;;
esac
# 写败 tid 落升件（ops_esc 函件径）
grep -q 'ops_esc' "$T/psql.log" && grep -q '落成自动销写败 tid=9102' "$T/psql.log" \
	&& grep -q 'landq_autoclose' "$T/psql.log" \
	&& ok "写败tid落升件(ops_esc landq_autoclose)" || bad "写败tid无升件"
# 反向对照：全放行时 csv 双票俱在（DID 追加语义不破坏正径）
new_case
mk_branch_msg "$T/o" fix/autoclose-ok feat_ao.txt "wip 工单#9201 与 工单#9202"
TIP=$(tip_of "$T/o" fix/autoclose-ok); stamp "$T/o" "$TIP"
: > "$T/desk_rows.tsv"
printf '9201\tf\tf\tf\tt\n9202\tf\tf\tf\tt\n' >> "$T/desk_rows.tsv"
rm -f "$T/psql.log"
run_lq T_REVIEW_OK=1 T_DESK_ROWS="$T/desk_rows.tsv"
_sumcsv2=$(grep '落成自动销 支=' "$T/psql.log" | grep -oE '票=[0-9,]+' | head -1)
case "$_sumcsv2" in *9201*9202*|*9202*9201*) ok "全放行csv双票俱在" ;; *) bad "全放行csv异常: $_sumcsv2" ;; esac
! grep -q 'ops_esc' "$T/psql.log" && ok "全放行零升件" || bad "全放行竟升件"

echo "== 汇总: PASS=$PASS FAIL=$FAILN =="
[ "$FAILN" -eq 0 ]
