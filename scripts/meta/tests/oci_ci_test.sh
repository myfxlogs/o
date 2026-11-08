#!/usr/bin/env bash
# oci_ci_test.sh —— 提案「check_all 甩 OCI」档 A 四件桩测（判据留证）。
#
# 覆盖：
#   A. 报告格式/核签——绿报告→check-green 同 check_all 格式（.mode 档+包清单）；
#      哈希篡改→拒；红→不落戳；full→不核签；已有鲜戳→保留不降级。
#   B. ARM↔x86 分歧探针——同 sha 侧档 arch≠本盒+本盒红→显式升件行+.diverged；
#      同架构静默退（判据③）。
#   C. fail-open——FORCE_LOCAL/remote 缺/push 败→本盒现径 rc 透传（判据④）。
#   D. 输送隔离——scratch-only sha 不入 origin/master 引用图；镜像 HEAD 包含
#      闸拒分支戳；合入后 OCI 戳被 mirror_push 同径消费（判据「半成品窗」）。
#   E. check_all fail() 已接分歧探针（线检）。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
FAIL=0
ok()  { echo "  ✅ $1"; }
bad() { echo "  ❌ $1"; FAIL=1; }

report() { # $1=file $2=sha $3=exit $4=mode $5=pkgs $6=hash $7=bytes [arch]
  cat > "$1" <<EOF
sha=$2
exit=$3
mode=$4
pkgs=$5
run=42
fail=0
skip=3
arch=${8:-aarch64}
baseline=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
host=oci-test
started=2026-11-09T00:00:00+08:00
finished=2026-11-09T00:10:00+08:00
log_sha256=$6
log_bytes=$7
note=green
EOF
}
hashof() { sha256sum "$1" | cut -d' ' -f1; }
bytesof() { wc -c < "$1" | tr -d ' '; }

# ══ A. 报告核签 ══
REPO="$TMP/r1"; mkdir -p "$REPO"; cd "$REPO"
git init -q && git config user.email t@t && git config user.name t
printf 'x\n' > f && git add f && git commit -qm base
git branch -M master
C1=$(git rev-parse HEAD)
LOG="$TMP/a1.log"; printf 'ok o/internal/risk 0.1s\nSKIP o/internal/ledger\n' > "$LOG"
RPT="$TMP/a1.report"
report "$RPT" "$C1" 0 merge-scoped "o/internal/risk o/internal/ledger" "$(hashof "$LOG")" "$(bytesof "$LOG")"
ACC() { ( cd "$REPO" && OCI_CI_REPORT_FILE="$1" OCI_CI_LOG_FILE="$2" bash "$ROOT/scripts/oci_ci_accept.sh" "$3" ); }

set +e; ACC "$RPT" "$LOG" "$C1" >/dev/null 2>"$TMP/a1.err"; RC=$?; set -e
STAMP="$REPO/.git/check-green-$C1"
if [ "$RC" = 0 ] && [ -f "$STAMP" ] && grep -qE '^[0-9]{4}-[0-9]{2}-[0-9]{2}T' "$STAMP" \
   && [ "$(sed -n 1p "$STAMP.mode")" = merge-scoped ] \
   && grep -qx 'o/internal/risk' "$STAMP.mode" && grep -qx 'o/internal/ledger' "$STAMP.mode"; then
  ok "A1 绿报告→戳落（date -Is）+ .mode 档/包清单同 check_all 格式"
else
  bad "A1 rc=$RC stamp=$(cat "$STAMP" 2>/dev/null) mode=$(cat "$STAMP.mode" 2>/dev/null | tr '\n' '|')"
fi
[ -f "$REPO/.git/oci-report-$C1" ] && grep -qx 'arch=aarch64' "$REPO/.git/oci-report-$C1" \
  && ok "A2 来源侧档 oci-report-<sha> 记 arch（分歧探针输入）" \
  || bad "A2 来源侧档缺/arch 未记"

# A3 先删戳再测篡改（否则保留鲜戳径干扰）
rm -f "$STAMP" "$STAMP.mode"
printf 'tamper\n' >> "$LOG"
set +e; ACC "$RPT" "$LOG" "$C1" >/dev/null 2>"$TMP/a3.err"; RC=$?; set -e
[ "$RC" = 2 ] && [ ! -f "$STAMP" ] && grep -q '哈希不符' "$TMP/a3.err" \
  && ok "A3 输出哈希篡改→rc=2 不落戳（防伪造面）" \
  || bad "A3 rc=$RC stamp=$([ -f "$STAMP" ] && echo 有 || echo 无)"

# A4 红报告
printf 'ok\n' > "$LOG"
report "$TMP/a4.report" "$C1" 1 scoped "o/internal/risk" "$(hashof "$LOG")" "$(bytesof "$LOG")"
set +e; ACC "$TMP/a4.report" "$LOG" "$C1" >/dev/null 2>&1; RC=$?; set -e
[ "$RC" = 1 ] && [ ! -f "$STAMP" ] && ok "A4 红报告→rc=1 不落戳（失败迭代离岸）" || bad "A4 rc=$RC"

# A5 full 档不核签
report "$TMP/a5.report" "$C1" 0 full "" "$(hashof "$LOG")" "$(bytesof "$LOG")"
set +e; ACC "$TMP/a5.report" "$LOG" "$C1" >/dev/null 2>&1; RC=$?; set -e
[ "$RC" = 75 ] && [ ! -f "$STAMP" ] && ok "A5 mode=full→rc=75 本盒终盖" || bad "A5 rc=$RC"

# A6 已有鲜戳=保留原戳（来源不降级）
printf 'LOCAL-SENTINEL\n' > "$STAMP"; date -Is > "$STAMP.mode"
report "$TMP/a6.report" "$C1" 0 scoped "o/internal/risk" "$(hashof "$LOG")" "$(bytesof "$LOG")"
set +e; ACC "$TMP/a6.report" "$LOG" "$C1" >/dev/null 2>&1; RC=$?; set -e
[ "$RC" = 0 ] && [ "$(cat "$STAMP")" = LOCAL-SENTINEL ] \
  && ok "A6 已有鲜戳→保留原戳（OCI 回报只落侧档）" || bad "A6 rc=$RC stamp=$(cat "$STAMP")"
rm -f "$STAMP" "$STAMP.mode" "$REPO/.git/oci-report-"*

# A7 report_lib 写读往返
. "$ROOT/scripts/oci_ci/report_lib.sh"
oci_report_write "$TMP/lib.report" "sha=$C1" "exit=0" "note=g r e e n"
[ "$(oci_report_get "$TMP/lib.report" sha)" = "$C1" ] \
  && [ "$(oci_report_get "$TMP/lib.report" note)" = "g r e e n" ] \
  && ok "A7 report_lib 写读往返（值含空格安全）" || bad "A7 report_lib 往返败"

# ══ B. ARM↔x86 分歧探针 ══
ESC_LOG="$TMP/esc.log"
cat > "$TMP/esc-stub.sh" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >> "$ESC_LOG"
EOF
chmod +x "$TMP/esc-stub.sh"
printf 'arch=aarch64\n' > "$REPO/.git/oci-report-$C1"
set +e; ( cd "$REPO" && O_CI_LOCAL_ARCH=x86_64 O_OPS_ESC_SH="$TMP/esc-stub.sh" \
  bash "$ROOT/scripts/oci_ci_divergence.sh" "$C1" ) >/dev/null 2>"$TMP/b1.err"; RC=$?; set -e
{ [ "$RC" = 0 ] && grep -q 'ARM↔x86 分歧' "$TMP/b1.err" \
  && [ -f "$REPO/.git/oci-report-$C1.diverged" ] \
  && grep -q 'OCI-ARM-X86-DIVERGENCE:' "$ESC_LOG"; } \
  && ok "B1 同 sha ARM 绿+本盒 x86 红→分歧显式报+升件行" \
  || bad "B1 rc=$RC err=$(cat "$TMP/b1.err") esc=$(cat "$ESC_LOG" 2>/dev/null)"
set +e; ( cd "$REPO" && O_CI_LOCAL_ARCH=aarch64 O_OPS_ESC_SH="$TMP/esc-stub.sh" \
  bash "$ROOT/scripts/oci_ci_divergence.sh" "$C1" ) >/dev/null 2>&1; RC=$?; set -e
[ "$RC" = 3 ] && ok "B2 同架构静默退（rc=3，非错）" || bad "B2 rc=$RC"
rm -f "$REPO/.git/oci-report-$C1"*

# ══ C. fail-open ══
set +e; ( cd "$REPO" && OCI_CI_FORCE_LOCAL=1 OCI_CI_LOCAL_FALLBACK='exit 42' \
  bash "$ROOT/scripts/oci_ci_push.sh" master ) >/dev/null 2>"$TMP/c1.err"; RC=$?; set -e
[ "$RC" = 42 ] && grep -q 'fail-open 本盒现径' "$TMP/c1.err" \
  && ok "C1 master 不走 OCI→本盒现径（rc 透传）" || bad "C1 rc=$RC"
set +e; ( cd "$REPO" && OCI_CI_FORCE_LOCAL=1 OCI_CI_LOCAL_FALLBACK='exit 43' \
  bash "$ROOT/scripts/oci_ci_push.sh" ) >/dev/null 2>"$TMP/c2.err"; RC=$?; set -e
[ "$RC" = 43 ] && ok "C2 FORCE_LOCAL=1→本盒现径（rc 透传）" || bad "C2 rc=$RC"

# C3 push 败→fallback（本地裸仓 pre-receive 拒，不走 SSH）
git branch -M master
printf 'y\n' > g && git add g && git commit -qm br
git checkout -qb br
git init -q --bare "$TMP/rej.git"
printf '#!/bin/sh\nexit 1\n' > "$TMP/rej.git/hooks/pre-receive"; chmod +x "$TMP/rej.git/hooks/pre-receive"
git remote add oci-ci "$TMP/rej.git"
set +e; ( cd "$REPO" && OCI_CI_LOCAL_FALLBACK='exit 44' OCI_CI_PUSH_TIMEOUT=30 \
  bash "$ROOT/scripts/oci_ci_push.sh" br ) >/dev/null 2>"$TMP/c3.err"; RC=$?; set -e
[ "$RC" = 44 ] && grep -q 'push oci-ci 败' "$TMP/c3.err" \
  && ok "C3 scratch push 败→本盒现径（判据④）" || bad "C3 rc=$RC err=$(tail -2 "$TMP/c3.err")"

# ══ D. 输送隔离 + 镜面消费（半成品窗）══
D="$TMP/iso"; mkdir -p "$D"; cd "$D"
git init -q --bare "$TMP/origin.git"
git init -q; git config user.email t@t; git config user.name t
git remote add origin "$TMP/origin.git"
printf 'base\n' > f && git add f && git commit -qm base && B0=$(git rev-parse HEAD)
git branch -M master
git push -q origin "$B0:refs/heads/master"
git update-ref refs/remotes/origin/master "$B0"
git init -q --bare "$TMP/scratch.git"
git remote add oci-ci "$TMP/scratch.git"
git checkout -qb br
printf 'c1\n' >> f && git commit -qam c1 && C1=$(git rev-parse HEAD)
git push -q oci-ci "master:refs/heads/master" "$C1:refs/heads/br"

ORIG_MASTER=$(git --git-dir="$TMP/origin.git" rev-parse refs/heads/master)
if [ "$ORIG_MASTER" = "$B0" ] && ! git --git-dir="$TMP/origin.git" cat-file -e "$C1^{commit}" 2>/dev/null; then
  ok "D1 scratch 分支 sha 不入 origin/master 引用图/对象图"
else
  bad "D1 origin master=$ORIG_MASTER（want $B0）或对象已泄"
fi

# 用本仓 accept 落 OCI 戳（C1）——之后回 master，模拟部署树 HEAD 不含分支 sha
LOG2="$TMP/d.log"; printf 'ok\n' > "$LOG2"
report "$TMP/d.report" "$C1" 0 scoped "o/internal/risk" "$(hashof "$LOG2")" "$(bytesof "$LOG2")"
( cd "$D" && OCI_CI_REPORT_FILE="$TMP/d.report" OCI_CI_LOG_FILE="$LOG2" bash "$ROOT/scripts/oci_ci_accept.sh" "$C1" ) >/dev/null 2>&1
git checkout -q master
PUSHLOG="$TMP/dpush.log"; : > "$PUSHLOG"
cat > "$TMP/dpush.sh" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >> "$PUSHLOG"
EOF
chmod +x "$TMP/dpush.sh"
MP() { O_MIRROR_REPO="$D" O_MIRROR_PUSH_SH="$TMP/dpush.sh" bash "$ROOT/scripts/mirror_push.sh" >/dev/null 2>&1; }
set +e; MP; set -e
[ ! -s "$PUSHLOG" ] && ok "D2 HEAD 不含分支 sha→OCI 戳被 HEAD 包含闸拒（镜面零推）" \
  || bad "D2 竟推: $(cat "$PUSHLOG")"

git merge -q --ff-only br
: > "$PUSHLOG"
set +e; MP; set -e
[ "$(cat "$PUSHLOG")" = "origin $C1:master" ] \
  && ok "D3 合入后 OCI 戳同径消费→mirror 推已绿 sha（戳语义不变）" \
  || bad "D3 pushlog=[$(cat "$PUSHLOG")] want origin $C1:master"

# D4 本地戳与 OCI 戳并存：格式同构、消费面无分叉（本地戳只是更陈不选中）
LSTAMP="$D/.git/check-green-$B0"; printf '2026-11-08T00:00:00+08:00\n' > "$LSTAMP"
printf 'scoped\no/internal/risk\n' > "$LSTAMP.mode"
touch -d '2 hours ago' "$LSTAMP"
: > "$PUSHLOG"; git update-ref refs/remotes/origin/master "$B0"
set +e; MP; set -e
[ "$(cat "$PUSHLOG")" = "origin $C1:master" ] \
  && ok "D4 本地/OCI 戳共存——同 glob 同格式，就近新鲜戳中选" \
  || bad "D4 pushlog=[$(cat "$PUSHLOG")]"

# ══ E. check_all fail() 接线 ══
grep -q 'oci_ci_divergence.sh' "$ROOT/scripts/check_all.sh" \
  && ok "E1 check_all fail() 已接分歧探针（判据③运行面）" || bad "E1 接线缺"

[ "$FAIL" = 0 ] && echo "✅ oci_ci_test 全场景过" || { echo "❌ 有红"; exit 1; }
