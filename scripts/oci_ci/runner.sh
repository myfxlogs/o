#!/usr/bin/env bash
# o-ci-run-once.sh —— OCI 侧单 sha 跑（提案「check_all 甩 OCI」档 A：筛）。
#
# 由 o-ci-runner@<sha>.service 以 sha 为实例启动（post-receive 每 ref 起一份）。
# 串行：flock 独占（多实例排队，先到先跑）；报告原子落 ~/o-ci/reports/<sha>.report，
# 本盒 oci_ci_accept.sh 拉取核签后落 check-green-<sha>（同 check_all 格式）。
#
# 档 A 边界：只跑 scoped/merge-scoped 定向谱；变更面含定向谱外件（web/ 等
# →check_all 会判 full）→不烧 OCI 全谱，报告 exit=75 note=surface-full，
# 本盒 fail-open 现径。ARM 绿≠x86 证——本盒终盖与分歧升件归 oci_ci_divergence.sh。
#
# 退出码：0=报告已落（绿/红/未跑均 0，systemd 不重启）；1=报告本身落盘败。
set -uo pipefail

CI_ROOT=${CI_ROOT:-$HOME/o-ci}
SHA=${1:?usage: o-ci-run-once.sh <sha>}
WORK="$CI_ROOT/work"
BARE="$CI_ROOT/scratch.git"
REPORTS="$CI_ROOT/reports"
ASSETS="$CI_ROOT/assets"
REPORT="$REPORTS/$SHA.report"
LOG="$REPORTS/$SHA.log"
LOCK="$CI_ROOT/runner.lock"

export PATH="$CI_ROOT/go/bin:$CI_ROOT/bin:/usr/local/bin:/usr/bin:/bin"
export GOMODCACHE=${GOMODCACHE:-$CI_ROOT/gomodcache}
mkdir -p "$REPORTS" "$GOMODCACHE" || exit 1
: > "$LOG" || exit 1
STARTED=$(date -Is)
BASE=""
NETGATES=full

# shellcheck source=report_lib.sh
. "$(dirname -- "${BASH_SOURCE[0]}")/report_lib.sh" || exit 1

write_report() { # $1=exit $2=mode $3=pkgs $4=run $5=fail $6=skip $7=note
  local h b
  h=$(sha256sum "$LOG" 2>/dev/null | cut -d' ' -f1 || true) # tolerate: 空 log 也给空哈希（读侧会拒）
  b=$(wc -c < "$LOG" 2>/dev/null | tr -d ' ' || true)
  oci_report_write "$REPORT" \
    "sha=$SHA" "exit=$1" "mode=$2" "pkgs=$3" "run=$4" "fail=$5" "skip=$6" \
    "arch=$(uname -m)" "baseline=$BASE" "host=$(hostname)" "netgates=$NETGATES" \
    "started=$STARTED" "finished=$(date -Is)" \
    "log_sha256=${h:-}" "log_bytes=${b:-0}" "note=$7"
}

not_run() { # $1=note
  echo "ℹ [o-ci-runner] sha=${SHA:0:12} 未跑：$1" >&2
  write_report 75 "" "" 0 0 0 "$1"
  exit 0
}

cd "$WORK" 2>/dev/null || not_run "work-clone-missing"

# 串行锁：多实例排队；超时不跑交本盒 fail-open（不并发抢测试库/CPU）。
exec 9>"$LOCK" || not_run "lock-open-fail"
if ! flock -w "${OCI_CI_LOCK_WAIT:-5400}" 9; then
  not_run "lock-timeout"
fi

# 基线=scratch master（本盒 oci_ci_push.sh 随支推出，scope 与本盒同径）。
git fetch --prune origin '+refs/heads/*:refs/remotes/origin/*' >>"$LOG" 2>&1 \
  || not_run "git-fetch-fail"
BASE=$(git rev-parse --verify -q origin/master || true)
[ -n "$BASE" ] || not_run "no-origin-master-baseline"
git cat-file -e "$SHA^{commit}" 2>/dev/null || not_run "sha-not-in-scratch"

# 定向谱预判（与 check_all 分类同集）：含集外件=full→OCI 不烧全谱。
MB=$(git merge-base "$SHA" "$BASE" 2>/dev/null || true)
[ -n "$MB" ] || not_run "merge-base-fail"
FULL=0
while IFS= read -r p; do
  [ -n "$p" ] || continue
  case "$p" in
    docs/*|skills/*|*.md|*.go|migrations/*.sql|proto/*|scripts/*|.github/*) ;;
    *) FULL=1 ;;
  esac
done <<< "$(git diff --name-only "$MB...$SHA" 2>/dev/null || true)"
[ "$FULL" = 0 ] || not_run "surface-full"

# 工作树归位：ignored 件（gen/dist）保留——bootstrap 快照径同本盒语义。
git checkout --force --detach "$SHA" >>"$LOG" 2>&1 || not_run "git-checkout-fail"
git clean -fd >>"$LOG" 2>&1 || true # tolerate: -x 不带=保留 ignored gen/dist；清败不阻断（check_all 以 status 判面）

# gen 件正径（Go-only 模板；TS/web gen 不参与 Go 闸——与 Dockerfile.core 同式）。
if ! buf generate --template buf.gen.core.yaml >>"$LOG" 2>&1 \
   || ! buf generate --template buf.gen.mtapi.yaml >>"$LOG" 2>&1; then
  not_run "buf-gen-fail"
fi
if [ ! -f web/dist/index.html ]; then
  [ -f "$ASSETS/web-dist.tar.gz" ] || not_run "web-dist-snapshot-missing"
  tar -xzf "$ASSETS/web-dist.tar.gz" -C "$WORK" >>"$LOG" 2>&1 || not_run "web-dist-extract-fail"
fi

# 模块缓存预热；go.sum 若被 go 改写（缺项补写）回合入态——否则 status 面
# 会把 go.sum 当工作树改动→check_all 判 full（档 A 外）。
go mod download >>"$LOG" 2>&1 || not_run "go-mod-download-fail"
git checkout -- go.mod go.sum 2>/dev/null || true # tolerate: 无改亦可

# 跑 check_all（全链同本盒；scoped 档内测试库租约由 o-postgres-test 族供给）。
# bridge-audit 延后：OCI 面 SG 凭据=restrict 隧道键（仅 15433 端口转发，不能
# 跑远端 psql）——本闸交本盒 accept 在**同 sha 净树**补跑，绿才落戳；延后
# 径本身不落 check-green（check_all 写 .deferred 标），故不构成绕闸产戳面。
CHECK_ALL_DEFER_BRIDGE=1 timeout "${OCI_CI_RUN_TIMEOUT:-5400}" bash scripts/check_all.sh >>"$LOG" 2>&1
RC=$?

MODE=""; PKGS=""
if [ -f "$WORK/.git/check-green-$SHA.mode" ]; then
  MODE=$(sed -n 1p "$WORK/.git/check-green-$SHA.mode")
  PKGS=$(tail -n +2 "$WORK/.git/check-green-$SHA.mode" | grep -v '^env-contention$' | tr '\n' ' ' | sed 's/  */ /g; s/^ //; s/ $//')
fi

if [ "$RC" = 124 ]; then
  not_run "run-timeout"
fi
if [ "$RC" != 0 ]; then
  echo "❌ [o-ci-runner] sha=${SHA:0:12} check_all 红 rc=$RC（日志 $LOG）" >&2
  write_report "$RC" "$MODE" "$PKGS" 0 0 0 "red-see-log"
  exit 0
fi

case "$MODE" in
  scoped|merge-scoped) ;;
  *) not_run "mode-${MODE:-unknown}-not-oci" ;;
esac
[ -n "$PKGS" ] || not_run "empty-pkg-list"
if [ -f "$WORK/.git/check-green-$SHA" ]; then
  NETGATES=full
elif [ -f "$WORK/.git/check-green-$SHA.deferred" ]; then
  NETGATES=local:bridge-audit
else
  not_run "green-marker-missing"
fi

# RUN/FAIL/SKIP 计数：同 GOCACHE 二次 go test -json 走缓存重放（毫秒级；
# 与 check_all 同缓存键，不重跑测试）。计数败=不核签交本盒终盖。
JSON="$REPORTS/$SHA.json"
if ! go test -json $PKGS >"$JSON" 2>>"$LOG"; then
  not_run "counts-json-fail"
fi
RUNN=$(grep -c '"Action":"run".*"Test":"' "$JSON" || true)
FAILN=$(grep -c '"Action":"fail".*"Test":"' "$JSON" || true)
SKIPN=$(grep -c '"Action":"skip".*"Test":"' "$JSON" || true)
[ "$FAILN" = 0 ] || not_run "counts-json-red"

write_report 0 "$MODE" "$PKGS" "$RUNN" "$FAILN" "$SKIPN" "green"
echo "✅ [o-ci-runner] sha=${SHA:0:12} mode=$MODE RUN=$RUNN FAIL=0 SKIP=$SKIPN" >&2

# 报告面保洁：>7 天清（日志/JSON/报告同拍；新报告在跑不受影响）。
find "$REPORTS" -maxdepth 1 -type f -mtime +7 -delete 2>/dev/null || true # tolerate: 保洁末环
exit 0
