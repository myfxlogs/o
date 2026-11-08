#!/usr/bin/env bash
# drift_converge_test.sh —— GW-001 段F3 自收敛执行器桩测：systemctl/go/
# ops_esc 全桩+真 git 仓（HEAD/origin/绿戳/树净真判），覆盖：
#   ① streak=1 零动作（探针误报不触发写） ② unit 内容 sync+daemon-reload
#   ③ timer disabled+inactive→enable --now ④ timer 内容 sync+restart
#   ⑤ binary stale→绿戳重建+属主 restart+回探绿 ⑥ 无绿戳拒修→升件+冻结
#   ⑦ 冻结窗内不重试 ⑧ 钱径行只升件零 build ⑨ 迁移欠账只升件零写
#   ⑩ 观测盲 fixable=- 零动作 ⑪ 快照缺席零动作 ⑫ repo 路径越界拒修升件
#   ⑬ HEAD≠origin/master 拒重建升件。
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
CONV="$ROOT/scripts/meta/drift_converge.sh"
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin" "$TMP/units" "$TMP/on" "$TMP/active" "$TMP/state"

# ── 桩：systemctl（cat 读件目录；状态读标记件；写动词落日志） ──
cat > "$TMP/bin/systemctl" <<'EOF'
#!/usr/bin/env bash
verb=${2:-}
case "$verb" in
  cat)
    name=${3:-}; f="$SYS_UNIT_DIR/$name"
    [ -f "$f" ] || exit 1
    printf '# /stub/%s\n' "$name"; cat "$f" ;;
  is-enabled)
    name=${3:-}
    [ -f "$SYS_ON_DIR/$name" ] && { cat "$SYS_ON_DIR/$name"; exit 0; }
    echo disabled ;;
  is-active)
    name=${3:-}
    [ -f "$SYS_ACTIVE_DIR/$name" ] && { cat "$SYS_ACTIVE_DIR/$name"; exit 0; }
    echo inactive; exit 3 ;;
  daemon-reload)
    printf '%s\n' "$verb" >> "$SYSC_LOG" ;;
  restart)
    name=${3:-}; printf '%s %s\n' "$verb" "$name" >> "$SYSC_LOG"
    printf 'active\n' > "$SYS_ACTIVE_DIR/$name" ;;
  enable)
    name=${4:-${3:-}}; printf '%s %s\n' "$verb" "$name" >> "$SYSC_LOG"
    printf 'enabled\n' > "$SYS_ON_DIR/$name"
    printf 'active\n' > "$SYS_ACTIVE_DIR/$name" ;;
  *) : ;;
esac
EOF

# ── 桩：go（version -m 读首行 vcs=；list -deps；build 写 GO_BUILD_SHA） ──
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
  build)
    out=${3:-}; spec=${4:-}
    printf 'build %s\n' "$spec" >> "$GO_LOG"
    [ "${GO_BUILD_RC:-0}" = 0 ] || exit "${GO_BUILD_RC}"
    printf 'vcs=%s\n' "$GO_BUILD_SHA" > "$out"; chmod +x "$out" ;;
  *) exit 2 ;;
esac
EOF

# ── 桩：ops_esc（$1=seat $5=prefix $6=detail 落日志） ──────────
cat > "$TMP/bin/opsesc.sh" <<'EOF'
#!/usr/bin/env bash
printf '%s|%s|%s\n' "$1" "$5" "$6" >> "$ESC_LOG"
EOF
# ── 桩：psql（执行账 INSERT 落日志） ─────────────────────────
cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
q="${*: -1}"
case "$q" in
  *"INSERT INTO ops_event"*) printf '%s\n' "$q" >> "$LEDGER_LOG" ;;
  *) : ;;
esac
EOF
chmod +x "$TMP/bin/"*

# ── 真 git 仓：C1→C2 改 deps（stale 判据）；绿戳/落差可造 ──────
REPO="$TMP/repo"; mkdir -p "$REPO/deploy" "$REPO/internal/core" "$REPO/cmd/livebin"
cd "$REPO"
git init -q && git config user.email t@t && git config user.name t
printf 'unit foo v1\n' > deploy/foo.service
printf 'timer bar v1\n' > deploy/bar.timer
printf 'package core\n' > internal/core/c.go
printf 'package main\n' > cmd/livebin/main.go
git add -A && git commit -qm c1
C1=$(git rev-parse HEAD)
printf 'package core // v2\n' > internal/core/c.go
git commit -qam c2
C2=$(git rev-parse HEAD)
git update-ref refs/remotes/origin/master "$C2"
STAMP="$REPO/.git/check-green-$C2"
printf 'ok\n' > "$STAMP"

# ── 环境 ─────────────────────────────────────────────────────
ST="$TMP/state"; SNAP="$ST/drift.tsv"
SYSC_LOG="$TMP/sysc.log"; GO_LOG="$TMP/go.log"; ESC_LOG="$TMP/esc.log"
LEDGER_LOG="$TMP/ledger.log"
: > "$SYSC_LOG"; : > "$GO_LOG"; : > "$ESC_LOG"; : > "$LEDGER_LOG"
export SYS_UNIT_DIR="$TMP/units" SYS_ON_DIR="$TMP/on" SYS_ACTIVE_DIR="$TMP/active"
export SYSC_LOG GO_LOG ESC_LOG LEDGER_LOG
export GO_DEPS_BODY="$REPO/internal/core|o/internal/core"
export PATH="$TMP/bin:$PATH"
export DM_ENV_FILE=/dev/null
export O_DRIFT_REPO="$REPO" O_DRIFT_STATE_DIR="$ST"
export O_DRIFT_SYSTEMCTL="$TMP/bin/systemctl" O_DRIFT_GO="$TMP/bin/go"
export O_DRIFT_GIT=git O_DRIFT_UNIT_DIR="$TMP/units"
export O_DRIFT_BIN_ROOTS="$TMP/bin" O_DRIFT_OPS_ESC="$TMP/bin/opsesc.sh"
export O_DRIFT_FIN_DSN=conv-test-dsn O_DRIFT_NOW=2000000

seed() {
  cp deploy/foo.service "$SYS_UNIT_DIR/foo.service"
  cp deploy/bar.timer "$SYS_UNIT_DIR/bar.timer"
  printf 'enabled\n' > "$SYS_ON_DIR/bar.timer"
  printf 'active\n' > "$SYS_ACTIVE_DIR/bar.timer"
  printf 'vcs=%s\n' "$C1" > "$TMP/bin/livebin"
  printf 'ExecStart=%s/bin/livebin\n' "$TMP" > "$SYS_UNIT_DIR/livebin.service"
  printf 'active\n' > "$SYS_ACTIVE_DIR/livebin.service"
  rm -rf "$ST"; mkdir -p "$ST"
  : > "$SYSC_LOG"; : > "$GO_LOG"; : > "$ESC_LOG"; : > "$LEDGER_LOG"
}
snap() { printf '%s\n' "$@" > "$SNAP"; }
bin_snap() { # subtype fixable streak
  printf 'binary:%s/bin/livebin\tbinary\t%s/bin/livebin\t./cmd/livebin\t%s\tdetail\t%s\t%s\n' \
    "$TMP" "$TMP" "$1" "$2" "$3" > "$SNAP"
}
run_c() { bash "$CONV" 2>"$TMP/err"; }
freeze_path() { printf '%s/fix-fail-%s' "$ST" "$(printf '%s' "$1" | sha256sum | cut -c1-16)"; }
esc_has() { grep -qF -- "$1" "$ESC_LOG"; }

# ── ① streak=1 零动作 ───────────────────────────────────────
seed
snap $'unit:foo.service\tunit\tfoo.service\tdeploy/foo.service\tcontent\tlive=x repo=y\t1\t1'
run_c
grep -qx 'unit foo v1' "$SYS_UNIT_DIR/foo.service" || { echo "FAIL ①streak=1 仍改 live"; exit 1; }
[ ! -s "$SYSC_LOG" ] && [ ! -s "$GO_LOG" ] && [ ! -s "$ESC_LOG" ] \
  || { echo "FAIL ①streak=1 有动作"; cat "$SYSC_LOG" "$GO_LOG" "$ESC_LOG"; exit 1; }
echo "ok ①streak=1 零动作（双拍确认硬闸）"

# ── ② unit 内容 sync+daemon-reload ──────────────────────────
seed
snap $'unit:foo.service\tunit\tfoo.service\tdeploy/foo.service\tcontent\tlive=x repo=y\t1\t2'
run_c
cmp -s deploy/foo.service "$SYS_UNIT_DIR/foo.service" \
  || { echo "FAIL ②unit 未 sync"; exit 1; }
grep -qx 'daemon-reload' "$SYSC_LOG" || { echo "FAIL ②缺 daemon-reload"; cat "$SYSC_LOG"; exit 1; }
! grep -q '^restart' "$SYSC_LOG" || { echo "FAIL ②服务误 restart"; cat "$SYSC_LOG"; exit 1; }
[ ! -s "$ESC_LOG" ] || { echo "FAIL ②成功却升件"; cat "$ESC_LOG"; exit 1; }
grep -qF "'机修'" "$LEDGER_LOG" && grep -qF 'unit:foo.service' "$LEDGER_LOG" \
  || { echo "FAIL ②执行账未留"; cat "$LEDGER_LOG"; exit 1; }
echo "ok ②unit 内容 sync+daemon-reload+机修执行账（服务不 restart 在飞实例）"

# ── ③ timer disabled+inactive→enable --now ──────────────────
seed
rm -f "$SYS_ON_DIR/bar.timer" "$SYS_ACTIVE_DIR/bar.timer"
snap $'timer:bar.timer\ttimer\tbar.timer\tdeploy/bar.timer\tdisabled+inactive\tenabled=disabled active=inactive\t1\t2'
run_c
grep -qx 'enable bar.timer' "$SYSC_LOG" || { echo "FAIL ③enable --now 未调"; cat "$SYSC_LOG"; exit 1; }
[ "$(cat "$SYS_ON_DIR/bar.timer" 2>/dev/null)" = enabled ] \
  || { echo "FAIL ③回探 enabled 未绿"; exit 1; }
[ ! -s "$ESC_LOG" ] || { echo "FAIL ③成功却升件"; cat "$ESC_LOG"; exit 1; }
echo "ok ③timer disabled+inactive→enable --now 回探绿"

# ── ④ timer 内容 sync+restart ───────────────────────────────
seed
printf 'timer bar live drift\n' > "$SYS_UNIT_DIR/bar.timer"
snap $'timer:bar.timer\ttimer\tbar.timer\tdeploy/bar.timer\tcontent\tlive=x repo=y\t1\t2'
run_c
cmp -s deploy/bar.timer "$SYS_UNIT_DIR/bar.timer" || { echo "FAIL ④timer 未 sync"; exit 1; }
grep -qx 'restart bar.timer' "$SYSC_LOG" || { echo "FAIL ④timer 未 restart"; cat "$SYSC_LOG"; exit 1; }
echo "ok ④timer 内容 sync+restart 排程面"

# ── ④b path 内容 sync+restart（排程面同 timer 律） ──────────
seed
printf 'path foo v1\n' > deploy/foo.path
printf 'path foo live drift\n' > "$SYS_UNIT_DIR/foo.path"
snap $'unit:foo.path\tunit\tfoo.path\tdeploy/foo.path\tcontent\tlive=x repo=y\t1\t2'
run_c
cmp -s deploy/foo.path "$SYS_UNIT_DIR/foo.path" || { echo "FAIL ④b path 未 sync"; exit 1; }
grep -qx 'restart foo.path' "$SYSC_LOG" || { echo "FAIL ④b path 未 restart"; cat "$SYSC_LOG"; exit 1; }
rm -f deploy/foo.path
echo "ok ④bpath 内容 sync+restart 排程面"

# ── ⑤ binary stale→绿戳重建+属主 restart+回探绿 ─────────────
seed
bin_snap stale 1 2
GO_BUILD_SHA="$C2" run_c
[ "$(sed -n '1s/^vcs=//p' "$TMP/bin/livebin")" = "$C2" ] \
  || { echo "FAIL ⑤live 未换到 target sha"; cat "$TMP/bin/livebin"; exit 1; }
grep -qx 'build ./cmd/livebin' "$GO_LOG" || { echo "FAIL ⑤未走绿戳重建"; cat "$GO_LOG"; exit 1; }
grep -qx 'restart livebin.service' "$SYSC_LOG" || { echo "FAIL ⑤属主未 restart"; cat "$SYSC_LOG"; exit 1; }
[ ! -e "$TMP/bin/livebin.drift-bak" ] || { echo "FAIL ⑤回滚备份未清"; ls "$TMP/bin"; exit 1; }
[ ! -s "$ESC_LOG" ] || { echo "FAIL ⑤成功却升件"; cat "$ESC_LOG"; exit 1; }
grep -qF "'机修'" "$LEDGER_LOG" && grep -qF 'binary:' "$LEDGER_LOG" \
  || { echo "FAIL ⑤执行账未留"; cat "$LEDGER_LOG"; exit 1; }
echo "ok ⑤binary 绿戳重建+属主 restart+回探 sha=target+机修执行账"

# ── ⑥ 无绿戳拒修→升件+失败冻结 ─────────────────────────────
seed
rm -f "$STAMP"
bin_snap stale 1 2
run_c
[ ! -s "$GO_LOG" ] || { echo "FAIL ⑥无绿戳仍 build"; cat "$GO_LOG"; exit 1; }
esc_has "DRIFT-FIX-FAIL:binary:$TMP/bin/livebin" \
  || { echo "FAIL ⑥拒修未升件"; cat "$ESC_LOG"; exit 1; }
[ -f "$(freeze_path "binary:$TMP/bin/livebin")" ] \
  || { echo "FAIL ⑥失败冻结痕未落"; ls "$ST"; exit 1; }
grep -qF "'机修'" "$LEDGER_LOG" && grep -qF '机修败' "$LEDGER_LOG" \
  || { echo "FAIL ⑥机修败执行账未留"; cat "$LEDGER_LOG"; exit 1; }
[ "$(sed -n '1s/^vcs=//p' "$TMP/bin/livebin")" = "$C1" ] \
  || { echo "FAIL ⑥live 被误动"; exit 1; }

# ── ⑦ 冻结窗内不重试（绿戳恢复亦不 build） ──────────────────
printf 'ok\n' > "$STAMP"; : > "$GO_LOG"; : > "$ESC_LOG"
run_c
[ ! -s "$GO_LOG" ] || { echo "FAIL ⑦冻结窗内仍 build"; cat "$GO_LOG"; exit 1; }
[ ! -s "$ESC_LOG" ] || { echo "FAIL ⑦冻结窗内重复升件"; cat "$ESC_LOG"; exit 1; }
grep -q '失败冻结窗内' "$TMP/err" || { echo "FAIL ⑦缺冻结日志"; cat "$TMP/err"; exit 1; }
echo "ok ⑥⑦无绿戳拒修升件+失败冻结 24h 不重试"

# ── ⑧ 钱径行只升件零 build ──────────────────────────────────
seed
bin_snap money-path 0 2
run_c
[ ! -s "$GO_LOG" ] || { echo "FAIL ⑧钱径行仍 build"; cat "$GO_LOG"; exit 1; }
esc_has "DRIFT:binary:$TMP/bin/livebin" || { echo "FAIL ⑧钱径行未升件"; cat "$ESC_LOG"; exit 1; }
[ "$(sed -n '1s/^vcs=//p' "$TMP/bin/livebin")" = "$C1" ] || { echo "FAIL ⑧live 被误动"; exit 1; }
echo "ok ⑧钱径域只升件零 build/零换件"

# ── ⑨ 迁移欠账只升件零写 ────────────────────────────────────
seed
snap $'migration:fin\tmigration\tfin\tinternal/store/pgstore/migrate_table.go\tdebt\tlive=148 登记=149\t0\t2'
run_c
[ ! -s "$GO_LOG" ] && [ ! -s "$SYSC_LOG" ] \
  || { echo "FAIL ⑨迁移欠账有写动作"; cat "$GO_LOG" "$SYSC_LOG"; exit 1; }
esc_has "DRIFT:migration:fin" || { echo "FAIL ⑨迁移欠账未升件"; cat "$ESC_LOG"; exit 1; }
echo "ok ⑨迁移欠账只升件零写动作"

# ── ⑩ 观测盲 fixable=- 零动作 ───────────────────────────────
seed
snap $'migration:fin\tmigration\tfin\tinternal/store/pgstore/migrate_table.go\tunreadable\t读败\t-\t2'
run_c
[ ! -s "$GO_LOG" ] && [ ! -s "$SYSC_LOG" ] && [ ! -s "$ESC_LOG" ] \
  || { echo "FAIL ⑩观测盲有动作"; cat "$GO_LOG" "$SYSC_LOG" "$ESC_LOG"; exit 1; }
echo "ok ⑩观测盲只报不升（fixable=- 零动作）"

# ── ⑪ 快照缺席零动作 ────────────────────────────────────────
seed
rm -f "$SNAP"
run_c
[ ! -s "$GO_LOG" ] && [ ! -s "$SYSC_LOG" ] && [ ! -s "$ESC_LOG" ] \
  || { echo "FAIL ⑪快照缺席有动作"; exit 1; }
echo "ok ⑪快照缺席零动作"

# ── ⑫ repo 路径越界拒修升件 ─────────────────────────────────
seed
snap $'unit:foo.service\tunit\tfoo.service\tscripts/foo.service\tcontent\tlive=x repo=y\t1\t2'
run_c
grep -qx 'unit foo v1' "$SYS_UNIT_DIR/foo.service" || { echo "FAIL ⑫越界路径仍改 live"; exit 1; }
esc_has "DRIFT-FIX-FAIL:unit:foo.service" || { echo "FAIL ⑫越界未升件"; cat "$ESC_LOG"; exit 1; }
echo "ok ⑫repo 路径非 deploy/* 拒修升件"

# ── ⑬ HEAD≠origin/master 拒重建 ────────────────────────────
seed
git update-ref refs/remotes/origin/master "$C1"
bin_snap stale 1 2
GO_BUILD_SHA="$C2" run_c
git update-ref refs/remotes/origin/master "$C2"
[ ! -s "$GO_LOG" ] || { echo "FAIL ⑬未收敛仍 build"; cat "$GO_LOG"; exit 1; }
esc_has "DRIFT-FIX-FAIL:binary:$TMP/bin/livebin" || { echo "FAIL ⑬未收敛未升件"; cat "$ESC_LOG"; exit 1; }
echo "ok ⑬HEAD≠origin/master 未收敛拒重建升件"

echo "drift converge scenarios PASS"
