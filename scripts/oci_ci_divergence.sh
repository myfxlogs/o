#!/usr/bin/env bash
# oci_ci_divergence.sh —— ARM↔x86 分歧探针（提案判据③：一旦发生→升件+档复审，禁静默）。
#
# 触发面：check_all.sh fail() 调本件（同 sha 跑红时）。判据：git-common-dir
# 有 oci_ci_accept.sh 落的 oci-report-<sha> 侧档（arch=OCI 实测架构），且
# OCI 架构≠本盒架构 → 同一 sha ARM 绿而本盒红=分歧。落 .diverged 痕+
# ops_event 升件行（ops_esc.sh，24h 同前缀去重；写径败静默退=末环限度）。
#
# 退出码：0=分歧已报；3=无侧档/同架构/不可判（静默退，非错）。
# 测试缝：O_CI_LOCAL_ARCH 覆写本盒架构；O_OPS_ESC_SH 替换升件写件。
set -uo pipefail

SHA=${1:-$(git rev-parse HEAD 2>/dev/null || true)} # tolerate: 无 HEAD=空 sha——下面 [ -n ] 静默退
[ -n "$SHA" ] || exit 3
GITDIR=$(git rev-parse --git-common-dir 2>/dev/null) || exit 3
GITDIR=$(cd "$GITDIR" 2>/dev/null && pwd -P) || exit 3
SIDE="$GITDIR/oci-report-$SHA"
[ -f "$SIDE" ] || exit 3
OARCH=$(grep -m1 '^arch=' "$SIDE" 2>/dev/null | cut -d= -f2-) || true # tolerate: 侧档读败=空 arch→下面 [ -n ] 判不可判静默退
[ -n "$OARCH" ] || exit 3

norm() { case "$1" in arm64) echo aarch64 ;; amd64) echo x86_64 ;; *) echo "$1" ;; esac; }
LARCH=${O_CI_LOCAL_ARCH:-$(uname -m)}
[ "$(norm "$OARCH")" != "$(norm "$LARCH")" ] || exit 3

echo "🚨 [check-all] ARM↔x86 分歧：sha=${SHA:0:12} OCI($OARCH) 绿、本盒($LARCH) 红——提案判据③升件+档A/B复审，禁静默" >&2
printf 'diverged %s oci=%s local=%s\n' "$(date -Is)" "$OARCH" "$LARCH" > "$SIDE.diverged" 2>/dev/null || true # tolerate: 痕写败不阻断升件

ESC=${O_OPS_ESC_SH:-"$(dirname -- "${BASH_SOURCE[0]}")/cognition/escalate.sh"}
[ -x "$ESC" ] && "$ESC" '跑席' '' '' '' "OCI-ARM-X86-DIVERGENCE:${SHA:0:12}" \
  "同 sha OCI($OARCH) 绿、本盒($LARCH) 红——提案判据③升件+档A/B复审（禁静默）" \
  >/dev/null 2>&1 || true # tolerate: 升件末环写败静默退（结构性限度在案）
exit 0
