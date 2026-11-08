#!/usr/bin/env bash
# drift_probe_test.sh —— GW-001 段F2 漂移只读探针桩测：systemctl/go/psql
# 全桩+真 git 仓（sha/祖先/落差面真判），覆盖：
#   ① 基线零漂移+--count/--keys ② unit 内容漂移→streak 累加+漂移行去重+
#   回绿出漂移消 ③ unit/timer 缺件 ④ timer disabled/inactive ⑤ 迁移
#   欠账/超登记/读败 ⑥ binary stale/无 deps 前进不报/money-path/deps
#   不可证/sha 不可考/sha 未并/缺件 ⑦ master 落差/origin 不可解 ⑧ streak
#   断窗归1 ⑨ manifest 缺席 rc1/用法错 rc2。
# 零写径断言：全程只读 live 桩，写面仅 $ST 快照+ops_event 桩日志。
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
PROBE="$ROOT/scripts/meta/drift_probe.sh"
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin" "$TMP/units" "$TMP/on" "$TMP/active" "$TMP/state"

# ── 桩：systemctl（cat 读件目录；enabled/active 读标记件） ──────
cat > "$TMP/bin/systemctl" <<'EOF'
#!/usr/bin/env bash
verb=${2:-}; name=${3:-}
case "$verb" in
  cat)
    f="$SYS_UNIT_DIR/$name"
    [ -f "$f" ] || exit 1
    printf '# /stub/%s\n' "$name"
    cat "$f" ;;
  is-enabled)
    [ -f "$SYS_ON_DIR/$name" ] && { cat "$SYS_ON_DIR/$name"; exit 0; }
    echo disabled ;;
  is-active)
    [ -f "$SYS_ACTIVE_DIR/$name" ] && { cat "$SYS_ACTIVE_DIR/$name"; exit 0; }
    echo inactive; exit 3 ;;
  *) exit 0 ;;
esac
EOF

# ── 桩：go（version -m 读二进制首行 vcs=；list -deps 回 GO_DEPS_BODY） ──
cat > "$TMP/bin/go" <<'EOF'
#!/usr/bin/env bash
case "${1:-}" in
  version)
    [ "${2:-}" = -m ] || exit 2
    sha=$(sed -n '1s/^vcs=//p' "${3:-}" 2>/dev/null)
    [ -n "$sha" ] || exit 1
    printf '%s: devel\n\tbuild\tvcs.revision=%s\n' "$3" "$sha" ;;
  list)
    [ "${GO_LIST_RC:-0}" = 0 ] || exit 1
    printf '%s\n' "${GO_DEPS_BODY:-}" ;;
  *) exit 2 ;;
esac
EOF

# ── 桩：schema live 查询（域→号；rc 可控） ─────────────────────
cat > "$TMP/bin/schema.sh" <<'EOF'
#!/usr/bin/env bash
case "${1:-}" in
  fin)
    [ "${SCHEMA_FIN_RC:-0}" = 0 ] || exit 1
    echo "${SCHEMA_FIN:-149}" ;;
  md)
    [ "${SCHEMA_MD_RC:-0}" = 0 ] || exit 1
    echo "${SCHEMA_MD:-1092}" ;;
  *) exit 2 ;;
esac
EOF

# ── 桩：psql（观测行 INSERT 落日志；去重查询空答） ─────────────
cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
q="${*: -1}"
case "$q" in
  *"INSERT INTO ops_event"*) printf '%s\n' "$q" >> "$OPS_LOG" ;;
  *"SELECT 1 FROM ops_event"*) printf '%s\n' "${DEDUPE_HIT:-}" ;;
  *) : ;;
esac
EOF
chmod +x "$TMP/bin/"*

# ── 真 git 仓：C1→C2 改 deps 目录（binary 落后判）；C2→C3 只改 docs ──
REPO="$TMP/repo"; mkdir -p "$REPO/deploy" "$REPO/internal/store/pgstore" \
  "$REPO/internal/core" "$REPO/internal/exec" "$REPO/cmd/livebin" "$REPO/docs"
cd "$REPO"
git init -q && git config user.email t@t && git config user.name t
printf 'unit foo v1\n' > deploy/foo.service
printf 'timer bar v1\n' > deploy/bar.timer
printf 'package core\n' > internal/core/c.go
printf 'package exec\n' > internal/exec/e.go
printf 'module o\n\ngo 1.22\n' > go.mod
printf 'package main\n' > cmd/livebin/main.go
printf 'package pgstore\nvar migrationTable = []struct{ Version int; Name string }{\n' > internal/store/pgstore/migrate_table.go
printf '\t{149, "fin_0149_autoapprove_family.sql"},\n' >> internal/store/pgstore/migrate_table.go
printf '\t{1092, "md_1092_some_name.sql"},\n' >> internal/store/pgstore/migrate_table.go
printf '}\n' >> internal/store/pgstore/migrate_table.go
printf 'v1\n' > docs/note.md
git add -A && git commit -qm c1
C1=$(git rev-parse HEAD)
printf 'package core // v2\n' > internal/core/c.go
git commit -qam c2
C2=$(git rev-parse HEAD)
printf 'v2\n' > docs/note.md
git commit -qam c3
C3=$(git rev-parse HEAD)
git update-ref refs/remotes/origin/master "$C3"
git checkout -q -b side "$C1"
printf 'side\n' > side.txt && git add side.txt && git commit -qm side
SIDE=$(git rev-parse HEAD)
git checkout -q -

# ── manifest + 环境 ───────────────────────────────────────────
MANI="$TMP/manifest.tsv"
{
  printf 'kind\tname\trepo\texpect\n'
  printf 'unit\tfoo.service\tdeploy/foo.service\tsync\n'
  printf 'timer\tbar.timer\tdeploy/bar.timer\tenabled\n'
  printf 'binary\t%s/bin/livebin\t./cmd/livebin\tgreen\n' "$TMP"
  printf 'migration\tfin\tinternal/store/pgstore/migrate_table.go\tmax\n'
} > "$MANI"

ST="$TMP/state"
OPS_LOG="$TMP/ops.log"; : > "$OPS_LOG"
export SYS_UNIT_DIR="$TMP/units" SYS_ON_DIR="$TMP/on" SYS_ACTIVE_DIR="$TMP/active"
export OPS_LOG
export GO_DEPS_BODY="$REPO/internal/core|o/internal/core"
export PATH="$TMP/bin:$PATH"
export DM_ENV_FILE=/dev/null
export O_DRIFT_REPO="$REPO" O_DRIFT_MANIFEST="$MANI" O_DRIFT_STATE_DIR="$ST"
export O_DRIFT_SYSTEMCTL="$TMP/bin/systemctl" O_DRIFT_GO="$TMP/bin/go"
export O_DRIFT_GIT=git O_DRIFT_SCHEMA_CMD="$TMP/bin/schema.sh"
export O_DRIFT_FIN_DSN=probe-test-dsn O_DRIFT_STREAK_WINDOW=1800
export O_DRIFT_SG_HOST=stub O_DRIFT_MD_HOST=stub

seed() {
  cp deploy/foo.service "$SYS_UNIT_DIR/foo.service"
  cp deploy/bar.timer "$SYS_UNIT_DIR/bar.timer"
  printf 'enabled\n' > "$SYS_ON_DIR/bar.timer"
  printf 'active\n' > "$SYS_ACTIVE_DIR/bar.timer"
  printf 'vcs=%s\n' "$C3" > "$TMP/bin/livebin"
  rm -rf "$ST"; mkdir -p "$ST"; : > "$OPS_LOG"
}
row() { awk -F'\t' -v k="$1" '$1==k{print $5"\t"$7"\t"$8; exit}' "$ST/drift.tsv"; }
expect_row() { # key subtype fixable streak
  local got; got=$(row "$1")
  [ "$got" = "$2"$'\t'"$3"$'\t'"$4" ] \
    || { echo "FAIL 行不符 $1: 期望=$2/$3/$4 实得=${got:-<无>}"; cat "$ST/drift.tsv"; exit 1; }
}
expect_absent() {
  ! grep -q "^$1"$'\t' "$ST/drift.tsv" \
    || { echo "FAIL 不应出现 $1"; cat "$ST/drift.tsv"; exit 1; }
}
runp() { bash "$PROBE"; }

# ── ① 基线零漂移 + --count/--keys ────────────────────────────
seed
runp
[ ! -s "$ST/drift.tsv" ] || { echo "FAIL ①基线有漂移"; cat "$ST/drift.tsv"; exit 1; }
[ "$(O_DRIFT_STATE_DIR="$ST" bash "$PROBE" --count)" = 0 ] || { echo "FAIL ①--count≠0"; exit 1; }
[ -z "$(O_DRIFT_STATE_DIR="$ST" bash "$PROBE" --keys)" ] || { echo "FAIL ①--keys 非空"; exit 1; }
echo "ok ①基线零漂移+count/keys 读面"

# ── ② unit 内容漂移：报出→streak 累加+漂移行去重→回绿漂移消 ──
seed
printf 'unit foo drift\n' > "$SYS_UNIT_DIR/foo.service"
O_DRIFT_NOW=1000000 runp
expect_row unit:foo.service content 1 1
grep -qF '漂移 unit:foo.service' "$OPS_LOG" || { echo "FAIL ②漂移行未落"; cat "$OPS_LOG"; exit 1; }
O_DRIFT_NOW=1000010 runp
expect_row unit:foo.service content 1 2
[ "$(grep -cF '漂移 unit:foo.service' "$OPS_LOG")" = 1 ] \
  || { echo "FAIL ②漂移行重复（同键应去重）"; cat "$OPS_LOG"; exit 1; }
cp deploy/foo.service "$SYS_UNIT_DIR/foo.service"
O_DRIFT_NOW=1000020 runp
expect_absent unit:foo.service
grep -qF '漂移消 unit:foo.service' "$OPS_LOG" || { echo "FAIL ②漂移消未落"; cat "$OPS_LOG"; exit 1; }
echo "ok ②unit 内容漂移→streak 2→回绿漂移消（观测行去重）"

# ── ③ unit/timer 缺件 ───────────────────────────────────────
seed
rm -f "$SYS_UNIT_DIR/foo.service"
O_DRIFT_NOW=1000100 runp
expect_row unit:foo.service missing 1 1
seed
rm -f "$SYS_UNIT_DIR/bar.timer"
O_DRIFT_NOW=1000200 runp
expect_row timer:bar.timer missing 1 1
echo "ok ③unit/timer live 缺件报出（fixable=1）"

# ── ④ timer disabled/inactive ───────────────────────────────
seed
rm -f "$SYS_ON_DIR/bar.timer"
O_DRIFT_NOW=1000300 runp
expect_row timer:bar.timer disabled 1 1
seed
rm -f "$SYS_ACTIVE_DIR/bar.timer"
O_DRIFT_NOW=1000400 runp
expect_row timer:bar.timer inactive 1 1
seed
rm -f "$SYS_ON_DIR/bar.timer" "$SYS_ACTIVE_DIR/bar.timer"
O_DRIFT_NOW=1000500 runp
expect_row timer:bar.timer disabled+inactive 1 1
echo "ok ④timer disabled/inactive/双面报出"

# ── ⑤ 迁移欠账/超登记/读败 ───────────────────────────────────
seed
SCHEMA_FIN=148 O_DRIFT_NOW=1000600 runp
expect_row migration:fin debt 0 1
seed
SCHEMA_FIN=150 O_DRIFT_NOW=1000700 runp
expect_row migration:fin ahead 0 1
seed
SCHEMA_FIN_RC=1 O_DRIFT_NOW=1000800 runp
expect_row migration:fin unreadable - 1
seed
O_DRIFT_NOW=1000900 runp
expect_absent migration:fin
echo "ok ⑤迁移欠账/超登记只升件（fixable=0），读败观测盲（fixable=-）"

# ── ⑥ binary 六态 ───────────────────────────────────────────
seed
printf 'vcs=%s\n' "$C1" > "$TMP/bin/livebin"
O_DRIFT_NOW=1001000 runp
expect_row "binary:$TMP/bin/livebin" stale 1 1
seed
printf 'vcs=%s\n' "$C2" > "$TMP/bin/livebin" # C2→C3 只改 docs：deps 未前进=不报
O_DRIFT_NOW=1001100 runp
expect_absent "binary:$TMP/bin/livebin"
seed
printf 'vcs=%s\n' "$C3" > "$TMP/bin/livebin" # sha==HEAD：money deps 亦不报（无漂移）
GO_DEPS_BODY="$REPO/internal/exec|o/internal/exec" O_DRIFT_NOW=1001200 runp
expect_absent "binary:$TMP/bin/livebin"
seed
printf 'vcs=%s\n' "$C2" > "$TMP/bin/livebin"
GO_DEPS_BODY="$REPO/internal/exec|o/internal/exec" O_DRIFT_NOW=1001300 runp
expect_row "binary:$TMP/bin/livebin" money-path 0 1
seed
printf 'vcs=%s\n' "$C2" > "$TMP/bin/livebin"
GO_LIST_RC=1 O_DRIFT_NOW=1001400 runp
expect_row "binary:$TMP/bin/livebin" dep-unknown 0 1
seed
: > "$TMP/bin/livebin"
O_DRIFT_NOW=1001500 runp
expect_row "binary:$TMP/bin/livebin" sha-unrecoverable 0 1
seed
printf 'vcs=deadbeefdeadbeef\n' > "$TMP/bin/livebin"
O_DRIFT_NOW=1001600 runp
expect_row "binary:$TMP/bin/livebin" sha-unrecoverable 0 1
seed
printf 'vcs=%s\n' "$SIDE" > "$TMP/bin/livebin"
O_DRIFT_NOW=1001700 runp
expect_row "binary:$TMP/bin/livebin" sha-unmerged 0 1
seed
rm -f "$TMP/bin/livebin"
O_DRIFT_NOW=1001800 runp
expect_row "binary:$TMP/bin/livebin" missing 1 1
echo "ok ⑥binary stale/不前进/money/不可证/不可考/未并/缺件七态"

# ── ⑦ repo master 落差/不可解 ───────────────────────────────
seed
git update-ref refs/remotes/origin/master "$C1"
O_DRIFT_NOW=1001900 runp
expect_row repo:master-gap master-gap 0 1
git update-ref refs/remotes/origin/master "$C3"
seed
git update-ref -d refs/remotes/origin/master
O_DRIFT_NOW=1002000 runp
expect_row repo:master-gap unreadable - 1
git update-ref refs/remotes/origin/master "$C3"
echo "ok ⑦master 落差只升件（fixable=0），origin 不可解=观测盲"

# ── ⑧ streak 断窗归1（久隔两拍不算连续） ─────────────────────
seed
printf 'unit foo drift\n' > "$SYS_UNIT_DIR/foo.service"
O_DRIFT_NOW=1100000 runp
expect_row unit:foo.service content 1 1
O_DRIFT_NOW=1100100 runp
expect_row unit:foo.service content 1 2
O_DRIFT_NOW=1101901 runp # 距上拍 1801s > 窗 1800
expect_row unit:foo.service content 1 1
echo "ok ⑧streak 窗内外累加/归1（防久隔误判持续）"

# ── ⑨ manifest 缺席 rc1、用法错 rc2 ──────────────────────────
if O_DRIFT_MANIFEST="$TMP/no-such.tsv" bash "$PROBE" 2>/dev/null; then
  echo "FAIL ⑨manifest 缺席仍 rc0"; exit 1
else
  [ $? -eq 1 ] || { echo "FAIL ⑨manifest 缺席 rc≠1"; exit 1; }
fi
if bash "$PROBE" --bogus 2>/dev/null; then
  echo "FAIL ⑨用法错仍 rc0"; exit 1
else
  [ $? -eq 2 ] || { echo "FAIL ⑨用法错 rc≠2"; exit 1; }
fi
echo "ok ⑨manifest 缺席 rc1/用法错 rc2"

echo "drift probe scenarios PASS"
