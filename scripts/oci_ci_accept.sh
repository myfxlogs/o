#!/usr/bin/env bash
# oci_ci_accept.sh —— 回本盒核签：OCI 回报 + 整跑日志 → check-green-<sha>。
#
# 提案「check_all 甩 OCI」②戳语义：OCI 绿回报携包清单+RUN/FAIL/SKIP+
# 输出哈希回本盒；本盒核签（哈希复算）后落戳——与 check_all 同格式：
#   <git-common-dir>/check-green-<sha>       内容=date -Is
#   <git-common-dir>/check-green-<sha>.mode  scoped/merge-scoped 首行+包清单逐行
# 戳语义不变（「该 sha 这些测试绿过」），mirror_push/pre-push 消费面不区分
# 戳源（半成品窗共存期：同域同格式）。来源侧档 oci-report-<sha> 记 arch/
# 哈希/包集——ARM↔x86 分歧探针 oci_ci_divergence.sh 读它。
#
# 退出码：0=戳落/保留鲜戳；1=OCI 报红（不落戳，交施工修）；2=不可核
# （拉取/字段/哈希不符——调用方 fail-open 本盒现径）；75=未跑/非定向谱。
#
# 环境：OCI_CI_SSH / OCI_CI_ROOT（无文件注入时经 ssh 拉取）；
#       OCI_CI_REPORT_FILE / OCI_CI_LOG_FILE（测试面本地文件注入）。
set -uo pipefail

SHA=${1:-}
case "$SHA" in
  ""|*[!0-9a-f]*) echo "usage: oci_ci_accept.sh <40-hex-sha>" >&2; exit 2 ;;
esac
[ "${#SHA}" -eq 40 ] || { echo "usage: oci_ci_accept.sh <40-hex-sha>" >&2; exit 2; }

ROOT=$(git rev-parse --show-toplevel 2>/dev/null) || { echo "❌ [oci-ci-accept] 非 git 树" >&2; exit 2; }
SELF=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
GITDIR=$(git rev-parse --git-common-dir 2>/dev/null) || exit 2
GITDIR=$(cd "$GITDIR" 2>/dev/null && pwd -P) || exit 2
STAMP="$GITDIR/check-green-$SHA"
SIDE="$GITDIR/oci-report-$SHA"
SSH_TARGET=${OCI_CI_SSH:-ubuntu@149.118.58.98}
CI_ROOT=${OCI_CI_ROOT:-/home/ubuntu/o-ci}
REPORT=${OCI_CI_REPORT_FILE:-}
LOG=${OCI_CI_LOG_FILE:-}

# shellcheck source=oci_ci/report_lib.sh
. "$SELF/oci_ci/report_lib.sh" || exit 2

TMP=""
cleanup() { [ -n "$TMP" ] && rm -rf "$TMP"; }
trap cleanup EXIT

if [ -z "$REPORT" ] || [ -z "$LOG" ]; then
  TMP=$(mktemp -d) || exit 2
  REPORT="$TMP/report"; LOG="$TMP/run.log"
  ssh -o BatchMode=yes -o ConnectTimeout=10 "$SSH_TARGET" \
    "cat '$CI_ROOT/reports/$SHA.report'" > "$REPORT" 2>"$TMP/fetch.err" \
    || { echo "❌ [oci-ci-accept] 报告拉取败（$(tail -1 "$TMP/fetch.err" 2>/dev/null)）——不可核" >&2; exit 2; }
  ssh -o BatchMode=yes -o ConnectTimeout=10 "$SSH_TARGET" \
    "cat '$CI_ROOT/reports/$SHA.log'" > "$LOG" 2>>"$TMP/fetch.err" \
    || { echo "❌ [oci-ci-accept] 日志拉取败——不可核（哈希无从复算）" >&2; exit 2; }
fi

get() { oci_report_get "$REPORT" "$1"; }
R_SHA=$(get sha); R_EXIT=$(get exit); R_MODE=$(get mode); R_PKGS=$(get pkgs)
R_RUN=$(get run); R_FAIL=$(get fail); R_SKIP=$(get skip); R_ARCH=$(get arch)
R_BASE=$(get baseline); R_HOST=$(get host); R_START=$(get started); R_FIN=$(get finished)
R_HASH=$(get log_sha256); R_BYTES=$(get log_bytes); R_NOTE=$(get note); R_NETGATES=$(get netgates)

[ "$R_SHA" = "$SHA" ] || { echo "❌ [oci-ci-accept] 报告 sha 不符（$R_SHA≠$SHA）——不可核" >&2; exit 2; }
case "$R_EXIT" in
  0) ;;
  75) echo "ℹ [oci-ci-accept] OCI 未跑（note=$R_NOTE）——本盒现径接手" >&2; exit 75 ;;
  *)
    echo "❌ [oci-ci-accept] OCI 报红 exit=$R_EXIT mode=${R_MODE:-?}——不落戳（日志尾）:" >&2
    tail -30 "$LOG" >&2 2>/dev/null || true
    exit 1 ;;
esac
case "$R_MODE" in
  scoped|merge-scoped) ;;
  *) echo "ℹ [oci-ci-accept] mode=${R_MODE:-?} 非定向谱——不核签，本盒终盖" >&2; exit 75 ;;
esac
[ -n "$R_ARCH" ] || { echo "❌ [oci-ci-accept] 报告缺 arch——来源不可证，不可核" >&2; exit 2; }
[ -n "$R_PKGS" ] || { echo "❌ [oci-ci-accept] 报告缺包清单——不可核" >&2; exit 2; }
case "$R_PKGS" in *[!A-Za-z0-9_./\ -]*) echo "❌ [oci-ci-accept] 包清单字符异常——不可核" >&2; exit 2 ;; esac
[ "$R_FAIL" = 0 ] || { echo "❌ [oci-ci-accept] exit=0 但 fail=$R_FAIL 自相矛盾——不可核" >&2; exit 2; }

ACT_HASH=$(sha256sum "$LOG" 2>/dev/null | cut -d' ' -f1 || true) # tolerate: 读败=空→下面拒
ACT_BYTES=$(wc -c < "$LOG" 2>/dev/null | tr -d ' ' || true)
if [ -z "$R_HASH" ] || [ "${ACT_HASH,,}" != "${R_HASH,,}" ]; then
  echo "❌ [oci-ci-accept] 输出哈希不符（报 $R_HASH / 实 ${ACT_HASH:-?}）——不可核（防伪造面）" >&2
  exit 2
fi
if [ -n "$R_BYTES" ] && [ "$R_BYTES" != "$ACT_BYTES" ]; then
  echo "❌ [oci-ci-accept] 输出字节数不符（报 $R_BYTES / 实 $ACT_BYTES）——不可核" >&2
  exit 2
fi

# 闸面完整性：OCI 面 SG 内容凭据窄（restrict 隧道键）→ bridge-audit 延后；
# 本盒必须对**同一 sha 净树**补跑该闸，绿才落戳（延后径本身不产戳=不绕闸）。
case "$R_NETGATES" in
  full) ;;
  local:bridge-audit)
    [ "$(git rev-parse HEAD 2>/dev/null)" = "$SHA" ] \
      || { echo "❌ [oci-ci-accept] 延后闸补跑需 HEAD==被核 sha（当前 $(git rev-parse HEAD 2>/dev/null)）——不可核" >&2; exit 2; }
    [ -z "$(git status --porcelain 2>/dev/null)" ] \
      || { echo "❌ [oci-ci-accept] 延后闸补跑需净树（工作树有改动）——不可核，本盒现径" >&2; exit 2; }
    echo "▶ [oci-ci-accept] 本盒补跑延后闸 bridge-audit（同 sha 净树；SG/OCI 凭据在本盒）…" >&2
    if ! bash "$ROOT/scripts/check_bridge_registry.sh" >&2; then
      echo "❌ [oci-ci-accept] 延后闸 bridge-audit 本盒红——不落戳" >&2
      exit 1
    fi ;;
  *)
    echo "❌ [oci-ci-accept] netgates=${R_NETGATES:-?} 不可认——不可核" >&2
    exit 2 ;;
esac

STAMP_STATE=written
if [ -f "$STAMP" ]; then
  AGE=$(( $(date +%s) - $(date -r "$STAMP" +%s) ))
  if [ "$AGE" -le 21600 ]; then
    STAMP_STATE=kept-existing
    echo "ℹ [oci-ci-accept] 已有鲜戳（${AGE}s）——保留原戳（不降级来源；侧档照记 OCI 回报）" >&2
  fi
fi
if [ "$STAMP_STATE" = written ]; then
  date -Is > "$STAMP" || exit 2
  { printf '%s\n' "$R_MODE"; printf '%s\n' $R_PKGS; } > "$STAMP.mode" || exit 2
fi

oci_report_write "$SIDE" \
  "sha=$SHA" "arch=$R_ARCH" "local_arch=$(uname -m)" "mode=$R_MODE" "pkgs=$R_PKGS" \
  "run=$R_RUN" "fail=$R_FAIL" "skip=$R_SKIP" "baseline=$R_BASE" "host=$R_HOST" \
  "netgates=$R_NETGATES" \
  "started=$R_START" "finished=$R_FIN" "log_sha256=${ACT_HASH:-}" "log_bytes=${ACT_BYTES:-0}" \
  "report_fetched=$(date -Is)" "stamp=$STAMP_STATE" "note=$R_NOTE" || exit 2

NPKG=$(printf '%s\n' $R_PKGS | grep -c . || true)
echo "✅ [oci-ci-accept] sha=${SHA:0:12} mode=$R_MODE pkgs=$NPKG RUN=$R_RUN FAIL=0 SKIP=$R_SKIP arch=$R_ARCH" >&2
echo "   戳=$STAMP（$STAMP_STATE） 输出哈希=${ACT_HASH:0:12}… 来源侧档=$SIDE" >&2
exit 0
