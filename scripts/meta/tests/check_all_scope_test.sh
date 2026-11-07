#!/usr/bin/env bash
# check_all_scope_test.sh —— T-0433 测谱分类器桩测（scratch git 仓+
# go/gofmt/buf/docker/门禁脚本全桩，逐场景断 .mode 档与包集）：
#   S0 docs-only 纯文档→docs-only 且 go test 零调用（旧档回归）；
#   S1 docs+scripts→full——首版基建件分支漏清 ALL_DOCS 致误判
#     docs-only 免测=fail-open，本场景=红桩回归锚；
#   S2 纯 .go→scoped（T-0409 语义零回归：无 gen/pgstore 混入）；
#   S3 .go+migrations+proto+scripts→merge-scoped（pgstore/gen 种子
#     +反向闭包+恒测册五域全在列=恒测册硬钉断言）；
#   S4 proto 无 gen 映射目录→full（映射不确定 fail-closed）；
#   S5 反向依赖图生成败→full（闭包不确定 fail-closed）；
#   S6 .go+go.mod→full（清单外件不缩谱）。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
FAIL=0
ok()  { echo "  ✅ $1"; }
bad() { echo "  ❌ $1"; FAIL=1; }

REPO="$TMP/repo"; BIN="$TMP/bin"; mkdir -p "$REPO" "$BIN"
cd "$REPO" && git init -q && git config user.email t@t && git config user.name t

# ── 桩件 ──
cat > "$BIN/go" <<'EOF'
#!/usr/bin/env bash
set -uo pipefail
cmd=${1:-}; shift || true  # tolerate: 无参调用=合法空集（stub go 兜底 exit 0）
case "$cmd" in
  build|vet) exit 0 ;;
  test)
    printf 'go %s\n' "$cmd $*" >> "${GO_STUB_LOG:?}"
    exit 0 ;;
  list)
    if [ "${1:-}" = "-f" ]; then shift 2; cat "${GO_STUB_GRAPH:?}" || exit 1; exit 0; fi
    wild=0; for a in "$@"; do case "$a" in *...*) wild=1;; esac; done
    if [ "$wild" = 1 ]; then
      printf '%s\n' o/internal/risk o/internal/risk/check o/internal/ledger \
        o/internal/ledger/funding o/internal/ledger/holding o/internal/ledger/report \
        o/internal/ledger/retention o/internal/trading o/internal/trading/execution \
        o/internal/trading/mtexec o/internal/trading/okxexec o/internal/guard \
        o/internal/portfolio/lifecycle
      exit 0
    fi
    d=${1:-}; d=${d#./}
    [ -n "$d" ] && [ -d "$d" ] && { echo "o/$d"; exit 0; }
    exit 1 ;;
  *) exit 0 ;;
esac
EOF
printf '#!/usr/bin/env bash\nexit 0\n' > "$BIN/gofmt"
printf '#!/usr/bin/env bash\nexit 0\n' > "$BIN/buf"
printf '#!/usr/bin/env bash\nexit 0\n' > "$BIN/docker"
chmod +x "$BIN"/*

# ── 仓内脚本面：真 check_all.sh（含 T-0435 source 的 testdb_lease.sh）+ 八门禁桩 ──
mkdir -p scripts
cp "$ROOT/scripts/check_all.sh" scripts/check_all.sh
cp "$ROOT/scripts/testdb_lease.sh" scripts/testdb_lease.sh
for g in check_import_direction check-env-reads check_rejects_keys \
         check_config_write_domain check_migrations check_bridge_registry \
         check_tolerate_report check_ops_chain; do
  printf '#!/usr/bin/env bash\nexit 0\n' > "scripts/$g.sh"
done
# go list ./dir 桩按目录实存判——gen/pgstore 预置目录=已登记映射
mkdir -p internal/store/pgstore internal/gen/dashboardv1
printf 'package pgstore\n'   > internal/store/pgstore/pgstore.go
printf 'package dashboardv1\n' > internal/gen/dashboardv1/gen.go

# 反向依赖图 fixture：o/internal/foo 与 gen/dashboardv1 的反向
# 依赖者=o/cmd/core（闭包实证）；恒测册包零依赖叶。
GRAPH="$TMP/graph.txt"
cat > "$GRAPH" <<'EOF'
o/cmd/core|o/internal/foo,o/internal/gen/dashboardv1,o/internal/risk||
o/internal/foo|||
o/internal/gen/dashboardv1|||
o/internal/risk|||
o/internal/risk/check|||
o/internal/ledger|||
o/internal/ledger/funding|||
o/internal/ledger/holding|||
o/internal/ledger/report|||
o/internal/ledger/retention|||
o/internal/trading|||
o/internal/trading/execution|||
o/internal/trading/mtexec|||
o/internal/trading/okxexec|||
o/internal/guard|||
o/internal/portfolio/lifecycle|||
o/internal/store/pgstore|||
EOF

LOG="$TMP/go.log"; : > "$LOG"
git add -A && git commit -qm base
BASE=$(git rev-parse HEAD)
git update-ref refs/remotes/origin/master "$BASE"
git checkout -qb work

put() { mkdir -p "$(dirname "$1")"; printf '%s\n' "$2" > "$1"; }
reset_case() {
  git reset -q --hard "$BASE" && git clean -fdq
  rm -f .git/check-green-*
  : > "$LOG"
}
run_check() { # $1=graph路径（缺省=$GRAPH）
  (cd "$REPO" && PATH="$BIN:$PATH" GO_STUB_GRAPH="${1:-$GRAPH}" GO_STUB_LOG="$LOG" \
     bash scripts/check_all.sh) >"$TMP/ca.out" 2>"$TMP/ca.err" \
    || { echo "── check_all 执行败:"; tail -20 "$TMP/ca.err"; exit 1; }
}
SHA=""; modefile() { echo "$REPO/.git/check-green-$SHA.mode"; }
expect_mode() { # $1=desc $2=want
  got=$(head -1 "$(modefile)")
  [ "$got" = "$2" ] && ok "$1 → mode=$got" || bad "$1 want=$2 got=$got"
}
expect_pkg() { # $1=desc $2=pkg $3=want(present|absent)
  if [ "$3" = present ]; then
    grep -qx "$2" "$(modefile)" && ok "$1 含 $2" || bad "$1 缺 $2"
  else
    grep -qx "$2" "$(modefile)" && bad "$1 不该含 $2" || ok "$1 不含 $2"
  fi
}
ALWAYS="o/internal/risk o/internal/risk/check o/internal/ledger o/internal/ledger/funding o/internal/ledger/holding o/internal/ledger/report o/internal/ledger/retention o/internal/trading o/internal/trading/execution o/internal/trading/mtexec o/internal/trading/okxexec o/internal/guard o/internal/portfolio/lifecycle"

# ══ S0 docs-only 回归 ══
reset_case
put docs/guide.md '# guide'
git add -A && git commit -qm s0 && SHA=$(git rev-parse HEAD)
run_check
expect_mode 'S0 纯docs' docs-only
grep -q 'go test' "$LOG" && bad 'S0 docs-only 不该跑 go test' || ok 'S0 go test 零调用'

# ══ S1 docs+scripts→full（fail-open 回归锚）══
reset_case
put docs/a.md '# a'; put scripts/x.sh '#!/usr/bin/env bash
exit 0'
git add -A && git commit -qm s1 && SHA=$(git rev-parse HEAD)
run_check
expect_mode 'S1 docs+scripts' full
grep -q 'go test ./\.\.\.' "$LOG" && ok 'S1 full 谱实跑' || bad 'S1 full 未跑 go test'

# ══ S2 纯 .go→scoped ══
reset_case
put internal/foo/x.go 'package foo'
git add -A && git commit -qm s2 && SHA=$(git rev-parse HEAD)
run_check
expect_mode 'S2 纯.go' scoped
expect_pkg 'S2' o/internal/foo present
expect_pkg 'S2' o/cmd/core present        # 反向闭包实证
expect_pkg 'S2' o/internal/gen/dashboardv1 absent
expect_pkg 'S2' o/internal/store/pgstore absent
for p in $ALWAYS; do expect_pkg 'S2 恒测册' "$p" present; done

# ══ S3 码迁proto基建混合→merge-scoped ══
reset_case
put internal/foo/x.go 'package foo'
put migrations/fin_0001_t.sql 'SELECT 1;'
put proto/dashboardv1/x.proto 'syntax = "proto3";'
put scripts/y.sh '#!/usr/bin/env bash
exit 0'
git add -A && git commit -qm s3 && SHA=$(git rev-parse HEAD)
run_check
expect_mode 'S3 混合面' merge-scoped
for p in o/internal/foo o/internal/store/pgstore o/internal/gen/dashboardv1 o/cmd/core; do
  expect_pkg 'S3' "$p" present
done
for p in $ALWAYS; do expect_pkg 'S3 恒测册' "$p" present; done

# ══ S4 proto 无映射→full ══
reset_case
put proto/o/x.proto 'syntax = "proto3";'
git add -A && git commit -qm s4 && SHA=$(git rev-parse HEAD)
run_check
expect_mode 'S4 proto未映射' full

# ══ S5 闭包图败→full ══
reset_case
put internal/foo/x.go 'package foo'
git add -A && git commit -qm s5 && SHA=$(git rev-parse HEAD)
run_check "$TMP/no-such-graph"
expect_mode 'S5 图生成败' full

# ══ S6 .go+go.mod→full ══
reset_case
put internal/foo/x.go 'package foo'; put go.mod 'module o'
git add -A && git commit -qm s6 && SHA=$(git rev-parse HEAD)
run_check
expect_mode 'S6 go.mod混入' full

[ "$FAIL" = 0 ] && echo "✅ check_all_scope_test 全场景过" || { echo "❌ check_all_scope_test 有红"; exit 1; }
