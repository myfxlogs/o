#!/usr/bin/env bash
# t0403_gate_converge_test.sh —— T-0403 双件桩测：
# ① check_ticket_numbers.sh --remote 改名豁免矩阵（0392/0397 归档带改
#    形单推直过、0395 同目录改名形）：
#      A 归档纯改名        → 放（原界不松回归）
#      B 归档+状态行改      → 放（豁免节①=0392/0397 形）
#      B2 归档+多行状态字段收成单行 → 放（豁免节①字段界——T-0403 自体形）
#      C 归档+回填节改      → 放（豁免节②）
#      D 归档+正文改        → 拦（正文不豁——闸语义不松）
#      E 同目录同号改名+状态行 → 放（0395 冷库→候选形）
#      F 同目录改名+正文改   → 拦
#      G done/→tasks/ 反归档 → 拦（非豁径他徙）
# ② master_push_converge.sh 竞态（reference-transaction 真钩装机对拍——
#    无证 rebase 非 ff 位移会被 non-FF 闸拦死）：
#      H 远端先行分叉 → rc3 自动合流+双方 commit 零丢失
#      I 合流冲突     → rc1 拒推+原状回复零 rebase 残留
#      J 快进/远端不可达 → rc0 放行/降级让位
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
CHK="$ROOT/scripts/check_ticket_numbers.sh"
CONV="$ROOT/scripts/master_push_converge.sh"
TMP=$(mktemp -d); trap 'rm -rf -- "$TMP"' EXIT
pass=0
ok() { echo "PASS $1"; pass=$((pass+1)); }
bad() { echo "FAIL $1" >&2; exit 1; }

mkrepo() { # $1=dir：裸 origin+工作仓 master 基线+真 non-FF 钩
  git init -q --bare -b master "$1/origin.git"
  git init -q -b master "$1/repo"
  git -C "$1/repo" config user.email t@t
  git -C "$1/repo" config user.name t
  git -C "$1/repo" remote add origin "$1/origin.git"
  cp "$ROOT/scripts/hooks/reference-transaction" "$1/repo/.git/hooks/"
  chmod +x "$1/repo/.git/hooks/reference-transaction"
  mkdir -p "$1/repo/docs/tasks"
  : > "$1/repo/base.txt"
  (cd "$1/repo" && git add -A && git commit -qm init && git push -qu origin master)
}

seed_ticket() { # $1=repo $2=仓内票路径 $3=状态值——立票面直推远端（远端已有票态）
  mkdir -p "$1/$(dirname "$2")"
  cat > "$1/$2" <<EOF
# 票题
- 状态：$3
- 领取：x
## 验收锚
① 锚A；② 锚B。
## 结果回填
（施工写）
EOF
  (cd "$1" && git add "$2" && git commit -qm "seed $2" && git push -q origin master)
}

seed_ticket_ml() { # 多行状态字段票面（续行缩进——T-0403 自体形）
  mkdir -p "$1/$(dirname "$2")"
  cat > "$1/$2" <<EOF
# 票题
- 状态：$3（长注跨行实证：
  续行一/续行二）
- 领取：x
## 验收锚
① 锚A；② 锚B。
## 结果回填
（施工写）
EOF
  (cd "$1" && git add "$2" && git commit -qm "seed $2" && git push -q origin master)
}

gate() { # $1=repo → 打 rc 到 stdout（stderr 落 $1/.gate-out 备查）
  (cd "$1" && "$CHK" --remote >/dev/null 2>"$1/.gate-out"; echo $?)
}

archive() { (cd "$1" && mkdir -p docs/tasks/done && git mv "$2" "docs/tasks/done/" && git commit -qm archive); }

# ══ ① 改名豁免矩阵 ══
D="$TMP/A"; mkrepo "$D"
seed_ticket "$D/repo" "docs/tasks/T-9001-甲票.md" 已批
archive "$D/repo" "docs/tasks/T-9001-甲票.md"
[ "$(gate "$D/repo")" = 0 ] && ok "A 归档纯改名→放" || { cat "$D/repo/.gate-out"; bad "A"; }

D="$TMP/B"; mkrepo "$D"
seed_ticket "$D/repo" "docs/tasks/T-9002-乙票.md" 已批
(cd "$D/repo" && mkdir -p docs/tasks/done && git mv "docs/tasks/T-9002-乙票.md" "docs/tasks/done/" \
  && sed -i 's/- 状态：已批/- 状态：完成/' "docs/tasks/done/T-9002-乙票.md" && git commit -qam "归档+状态行")
rc=$(gate "$D/repo"); [ "$rc" = 0 ] && grep -q '票号身份未变' "$D/repo/.gate-out" \
  && ok "B 归档+状态行改→放（0392/0397形单推直过）" || { cat "$D/repo/.gate-out"; bad "B rc=$rc"; }

D="$TMP/B2"; mkrepo "$D"
seed_ticket_ml "$D/repo" "docs/tasks/T-9008-多行态票.md" 施工中
(cd "$D/repo" && mkdir -p docs/tasks/done && git mv "docs/tasks/T-9008-多行态票.md" "docs/tasks/done/")
# 归档翻态把多行状态字段收成单行——字段内改写应豁
cat > "$D/repo/docs/tasks/done/T-9008-多行态票.md" <<EOF
# 票题
- 状态：完成（裁决已合）
- 领取：x
## 验收锚
① 锚A；② 锚B。
## 结果回填
（施工写）
EOF
(cd "$D/repo" && git commit -qam "归档+多行态收成一行")
[ "$(gate "$D/repo")" = 0 ] && ok "B2 归档+多行状态字段收成一行→放" || { cat "$D/repo/.gate-out"; bad "B2"; }

D="$TMP/C"; mkrepo "$D"
seed_ticket "$D/repo" "docs/tasks/T-9003-丙票.md" 已批
(cd "$D/repo" && mkdir -p docs/tasks/done && git mv "docs/tasks/T-9003-丙票.md" "docs/tasks/done/" \
  && sed -i 's/（施工写）/桩测回填实录/' "docs/tasks/done/T-9003-丙票.md" && git commit -qam "归档+回填")
[ "$(gate "$D/repo")" = 0 ] && ok "C 归档+回填节改→放" || { cat "$D/repo/.gate-out"; bad "C"; }

D="$TMP/D"; mkrepo "$D"
seed_ticket "$D/repo" "docs/tasks/T-9004-丁票.md" 已批
(cd "$D/repo" && mkdir -p docs/tasks/done && git mv "docs/tasks/T-9004-丁票.md" "docs/tasks/done/" \
  && sed -i 's/锚A/锚X/' "docs/tasks/done/T-9004-丁票.md" && git commit -qam "归档+正文")
rc=$(gate "$D/repo"); [ "$rc" = 1 ] && grep -q '他票占用' "$D/repo/.gate-out" \
  && ok "D 归档+正文改→拦" || { cat "$D/repo/.gate-out"; bad "D rc=$rc"; }

D="$TMP/E"; mkrepo "$D"
seed_ticket "$D/repo" "docs/tasks/T-9005-冷库.md" 已批
(cd "$D/repo" && git mv "docs/tasks/T-9005-冷库.md" "docs/tasks/T-9005-候选.md" \
  && sed -i 's/- 状态：已批/- 状态：候选/' "docs/tasks/T-9005-候选.md" && git commit -qam "同目录改名")
[ "$(gate "$D/repo")" = 0 ] && ok "E 同目录改名+状态行→放（0395形）" || { cat "$D/repo/.gate-out"; bad "E"; }

D="$TMP/F"; mkrepo "$D"
seed_ticket "$D/repo" "docs/tasks/T-9006-冷库.md" 已批
(cd "$D/repo" && git mv "docs/tasks/T-9006-冷库.md" "docs/tasks/T-9006-候选.md" \
  && sed -i 's/锚A/锚Y/' "docs/tasks/T-9006-候选.md" && git commit -qam "同目录+正文")
rc=$(gate "$D/repo"); [ "$rc" = 1 ] && grep -q '他票占用' "$D/repo/.gate-out" \
  && ok "F 同目录改名+正文改→拦" || { cat "$D/repo/.gate-out"; bad "F rc=$rc"; }

D="$TMP/G"; mkrepo "$D"
seed_ticket "$D/repo" "docs/tasks/done/T-9007-庚票.md" 完成
(cd "$D/repo" && git mv "docs/tasks/done/T-9007-庚票.md" "docs/tasks/T-9007-庚票.md" \
  && git commit -qm "反归档")
rc=$(gate "$D/repo"); [ "$rc" = 1 ] && ok "G done/→tasks/ 反归档→拦" || { cat "$D/repo/.gate-out"; bad "G rc=$rc"; }

# ══ ② 竞态合流 ══
D="$TMP/H"; mkrepo "$D"
(cd "$D/repo" && echo seatA > a.txt && git add a.txt && git commit -qm "A席件")
git clone -q "$D/origin.git" "$D/repo2" 2>/dev/null
git -C "$D/repo2" config user.email t@t; git -C "$D/repo2" config user.name t
(cd "$D/repo2" && git checkout -q master && echo seatB > b.txt && git add b.txt \
  && git commit -qm "B席件" && git push -q origin master)
BSHA=$(git -C "$D/repo2" rev-parse master); ASHA=$(git -C "$D/repo" rev-parse master)
rc=$(cd "$D/repo" && "$CONV" "$ASHA" >/dev/null 2>&1; echo $?)
NSHA=$(git -C "$D/repo" rev-parse master)
{ [ "$rc" = 3 ] && [ "$NSHA" != "$ASHA" ] \
  && [ -f "$D/repo/a.txt" ] && [ -f "$D/repo/b.txt" ] \
  && git -C "$D/repo" merge-base --is-ancestor "$BSHA" master \
  && git -C "$D/repo" log --format=%s master | grep -q 'A席件'; } \
  && ok "H 远端先行分叉→rc3合流+双件零丢失" || bad "H rc=$rc"

D="$TMP/I"; mkrepo "$D"
(cd "$D/repo" && echo A > base.txt && git commit -qam "A改同行")
git clone -q "$D/origin.git" "$D/repo2" 2>/dev/null
git -C "$D/repo2" config user.email t@t; git -C "$D/repo2" config user.name t
(cd "$D/repo2" && git checkout -q master && echo B > base.txt && git commit -qam "B改同行" \
  && git push -q origin master)
ASHA=$(git -C "$D/repo" rev-parse master)
rc=$(cd "$D/repo" && "$CONV" "$ASHA" >/dev/null 2>&1; echo $?)
{ [ "$rc" = 1 ] && [ "$(git -C "$D/repo" rev-parse master)" = "$ASHA" ] \
  && ! git -C "$D/repo" rev-parse -q --verify REBASE_HEAD >/dev/null 2>&1; } \
  && ok "I 合流冲突→rc1拒推+原状回复" || bad "I rc=$rc"

D="$TMP/J"; mkrepo "$D"
SHA=$(git -C "$D/repo" rev-parse master)
rc=$(cd "$D/repo" && "$CONV" "$SHA" >/dev/null 2>&1; echo $?)
[ "$rc" = 0 ] && ok "J 快进→rc0放行" || bad "J rc=$rc"
git -C "$D/repo" remote set-url origin "$TMP/nonexistent.git"
rc=$(cd "$D/repo" && "$CONV" "$SHA" >/dev/null 2>&1; echo $?)
[ "$rc" = 0 ] && ok "J2 远端不可达→rc0降级让位" || bad "J2 rc=$rc"

echo "t0403 豁免矩阵×8 + 竞态×4 全 PASS（$pass 案）"
