#!/usr/bin/env bash
# check_all.sh — CI 同链全检（业主令 2026-10-27：绿了才推）——build→vet→
# import→gofmt→env→rejects→migrations→go test 全谱，绿后盖鲜戳
# .git/check-green-<SHA>（pre-push 验戳放行；无戳/陈戳=拒）。
# worktree 缺 gen/dist 件先跑 scripts/provision_worktree.sh。
# GW-001 A3 戳位归一：戳落 git-common-dir——linked worktree 的 --git-dir
# 是私有目录（.git/worktrees/<n>），戳须落公共仓戳面供 pre-push/mirror
# 读（worktree 盖戳与主树推/镜同域）。
set -uo pipefail
cd "$(git rev-parse --show-toplevel)"
GITDIR=$(git rev-parse --git-common-dir)
STAMP="$GITDIR/check-green-$(git rev-parse HEAD)"

fail() {
  echo "❌ [check-all] $1——绿了才推" >&2
  # 提案-check_all甩OCI 判据③：同 sha OCI(ARM) 绿而本盒红=分歧→升件+档复审禁静默。
  bash "$(pwd -P)/scripts/oci_ci_divergence.sh" "$(git rev-parse HEAD 2>/dev/null || true)" || true # tolerate: 分歧探针末环（无侧档=静默退）
  exit 1
}
BRIDGE_DEFERRED=0
step() { echo "── $1" >&2; }

# T-0409 刀①：同 checkout 全局互斥锁——锁键=git toplevel 真实路径
# 哈希（wt 与主树不同锁互不串；锁域=同一 checkout 内并发互斥，
# 非全机锁——各 wt 并发跑合法）。语义=阻塞等锁+超时上界 1800s：
# 入者即时报「等锁 holder pid」非静默睡死；超时 fail 报 holder
# 不报假绿不自杀。拿锁后才毁旧戳——等锁期不碰在途跑的戳面。
TOPKEY=$(pwd -P | sha256sum | cut -c1-16)
# T-0466 件⑦：GOCACHE 按 checkout 分家——并发跑互逐「非码面红」
# （T-0458 外审裁记实证）绝种；代价=各 wt 独立缓存目录，孤儿目录
# 由 wt_gc.sh 保洁面按「wt 已删+锁未持+龄>24h」三闸清。
export GOCACHE=${GOCACHE:-/tmp/gocache-${TOPKEY}}
mkdir -p "$GOCACHE" 2>/dev/null || true # tolerate: 建败=go 自建/报错——缓存目录非硬依赖
LOCK="/tmp/o-check_all-${TOPKEY}.lock"

# T-0466 件⑤：等锁面可观测——入等锁/锁到手/等锁超时各落 ops_event
# 行（seat=跑席按票面钦定；note=锁 key+holder+锁龄）。无头席撞锁
# 不再「败了重试→再撞」傻等不可见。DSN 缺席静默退（审计件非硬闸）。
oe_lock() { # $1=note
  local dsn
  dsn=$(grep '^FIN_PG_DSN=' /opt/o-research/env 2>/dev/null | cut -d= -f2- | tr -d '"')
  [ -n "$dsn" ] && psql "$dsn" -qAt -c \
    "INSERT INTO ops_event(seat,event,ticket,note) VALUES('跑席','等锁','T-0466','$1')" \
    >/dev/null 2>&1 || true # tolerate: 等锁行写败不阻断跑——审计件非硬闸（同 wt_gc GC 行式）
}

exec 9>"$LOCK"
if ! flock -n 9 2>/dev/null; then
  HOLDER=$(fuser "$LOCK" 2>/dev/null | tr -s ' ' '\n' | grep -v "^$$\$" | head -3 | tr '\n' ' ' || true) # tolerate: fuser 败/空集=holder 未知如实报 ?
  HPID=$(printf '%s' "$HOLDER" | awk '{print $1}') # tolerate: holder 空集=?——etimes 取空落 ? 如实报
  LOCK_AGE=$(ps -o etimes= -p "$HPID" 2>/dev/null | tr -d ' ' || true) # tolerate: holder 已退/ps 败=空串落 ?
  echo "⏳ [check-all] 等锁——另一跑在途 holder pid=[${HOLDER:-?}]（锁=$LOCK；阻塞等锁上界 1800s，超时 fail 不假绿）" >&2
  oe_lock "check_all 等锁 key=${TOPKEY} holder=[${HOLDER:-?}] 锁龄=${LOCK_AGE:-?}s"
  if ! flock -w 1800 9; then
    oe_lock "check_all 等锁超时 key=${TOPKEY} holder=[${HOLDER:-?}] 锁龄=${LOCK_AGE:-?}s"
    fail "等锁超时 1800s——holder pid=[${HOLDER:-?}]（锁=$LOCK）；前跑仍在途或锁泄漏，ps 核查后重试"
  fi
  oe_lock "check_all 锁到手 key=${TOPKEY}（前跑已退续跑）"
  echo "✅ [check-all] 锁到手（前跑已退）——续跑" >&2
fi
rm -f "$STAMP" "$STAMP.mode" "$STAMP.deferred"   # 重跑先毁旧戳/延后标（含 mode 痕）——中途败不留残戳

step "go build";    go build ./...                              || fail "go build 红"
step "go vet";      go vet ./...                                || fail "go vet 红"
step "import 方向"; bash scripts/check_import_direction.sh      || fail "import 门禁红"
step "gofmt"
UNFMT=$(gofmt -l . 2>/dev/null || true)
[ -z "$UNFMT" ] || fail "gofmt 未格式件: $UNFMT"
step "env-reads";   bash scripts/check-env-reads.sh             || fail "env 门禁红"
step "rejects";     bash scripts/check_rejects_keys.sh          || fail "rejects 键红"
step "config-write-domain"; bash scripts/check_config_write_domain.sh || fail "config 写域闸红"
step "migrations";  bash scripts/check_migrations.sh            || fail "迁移机检红"
step "bridge-audit"
if [ "${CHECK_ALL_DEFER_BRIDGE:-}" = 1 ]; then
  # 提案-check_all甩OCI：OCI 面 SG 内容凭据窄（隧道键 restrict 仅 15433）——
  # 本闸延后到本盒 accept 补跑同 sha 树（绿才落戳；延后跑本身不落戳，
  # 故本环境不可用于绕闸产戳）。本地部署树正常跑不设此 env=全量。
  BRIDGE_DEFERRED=1
  echo "⏸ [check-all] bridge-audit 延后（CHECK_ALL_DEFER_BRIDGE=1——本机无 SG 内容面凭据；本盒 accept 补跑后方落戳）" >&2
else
  bash scripts/check_bridge_registry.sh         || fail "桥域对账闸红"
fi
step "tolerate-report"; bash scripts/check_tolerate_report.sh      || fail "容错必报 lint 红"
step "ops-chain";   bash scripts/check_ops_chain.sh                || fail "ops 账链闸红"
step "seat-writes"; bash scripts/check_seat_writes.sh              || fail "席位写域闸红"
# 测谱分级（T-0409 刀②+④；T-0433 二期插 merge-scoped 档）：变更面=
# 已提交面（merge-base...HEAD）∪工作树面（status --porcelain 含
# untracked），与 docs-only 豁免同径收集不复制判据。四档——
#   docs-only：变更面全中白名单（docs/**、*.md 任意层、skills/**）→
#     免 go test 段（业主令 2026-10-31 既有语义不动）；
#   scoped：变更面=.go+docs/infra（无 mig/proto）→「变更包+反向
#     依赖闭包+恒测册」定向谱（T-0409 语义零回归）；
#   merge-scoped（T-0433）：变更面=.go+migrations/*.sql+proto/*+
#     安全基建件（scripts/**、.github/**）→同 scoped 谱+迁移附加
#     种子 pgstore+proto 附加种子 internal/gen/<dir> 映射包+
#     buf build proto 闸——码迁件免全谱（merge 场景实证病灶）；
#   full：变更面含任何清单外码件（go.mod/sum、web/、Dockerfile、
#     yaml/json、tools 非 go 等）或包映射/闭包/恒测册展开任一步
#     不确定→回退全谱 fail-safe 不 fail-open。
# CHECK_ALL_FORCE_FULL=1 无条件全谱；merge-base/命令败 fail-closed
# 走全谱。戳旁 .mode 四档留痕（scoped/merge-scoped 附实测包集清单
# ——T-0397 三档审裁读戳面同形）。
MODE=full; NCOM=0; NWT=0; MB=""; TEST_PKGS=""
ALWAYS_PKGS="o/internal/risk/... o/internal/ledger/... o/internal/trading/... o/internal/guard o/internal/portfolio/lifecycle/..."
if [ "${CHECK_ALL_FORCE_FULL:-}" != "1" ]; then
  MB=$(git merge-base HEAD origin/master 2>/dev/null) \
    || MB=$(git merge-base HEAD master 2>/dev/null) \
    || MB=$(git rev-parse --verify HEAD~1 2>/dev/null) || MB=""
  if [ -n "$MB" ] \
    && CFILES=$(git diff --name-only "$MB"...HEAD 2>/dev/null) \
    && WLIST=$(git status --porcelain 2>/dev/null); then
    # 净路径集=已提交面∪工作树面（porcelain 前 3 列状态剥除；rename 取新名）
    CHANGED=$( { printf '%s\n' "$CFILES"
                printf '%s\n' "$WLIST" | while IFS= read -r l; do
                  [ -n "$l" ] || continue
                  p=${l:3}
                  [[ "$p" == *" -> "* ]] && p=${p##* -> }
                  printf '%s\n' "$p"
                done; } | grep -v '^$' | sort -u ) # tolerate: 空集=零变更合法径
    NCOM=$(printf '%s\n' "$CFILES" | grep -c . || true) # tolerate: 空集=零已提交变更合法径
    NWT=$(printf '%s\n' "$WLIST" | grep -c . || true)  # tolerate: 空集=净工作树合法径
    ALL_DOCS=1; HAS_GO=0; HAS_MIG=0; HAS_PROTO=0; HAS_OTHER=0
    CODE_SEEN=0
    while IFS= read -r p; do
      [ -n "$p" ] || continue
      case "$p" in
        docs/*|skills/*|*.md) ;;
        *.go)             HAS_GO=1;    CODE_SEEN=1; ALL_DOCS=0 ;;
        migrations/*.sql) HAS_MIG=1;   CODE_SEEN=1; ALL_DOCS=0 ;;
        proto/*)          HAS_PROTO=1; CODE_SEEN=1; ALL_DOCS=0 ;;
        # 安全基建件=中性：不算 docs（ALL_DOCS=0——独现/伴 docs 落
        # full 不免测）不算清单外（HAS_OTHER 不置——伴码件入
        # merge-scoped 定向谱）。T-0433 首版漏清 ALL_DOCS 致
        # docs+scripts 面误判 docs-only 免测=fail-open，红桩锚在
        # scripts/meta/tests/check_all_scope_test.sh 场景①。
        scripts/*|.github/*) ALL_DOCS=0 ;;
        *)   HAS_OTHER=1; ALL_DOCS=0 ;;
      esac
    done <<< "$CHANGED"
    if [ "$ALL_DOCS" = 1 ]; then
      MODE=docs-only
    elif [ "$HAS_OTHER" = 0 ] && [ "$CODE_SEEN" = 1 ]; then
      # 定向谱候选（scoped/merge-scoped 同构）：种子=.go 包映射+
      # 迁移→pgstore+proto→gen 包映射；闭包+恒测册同式；任一步不确定回退 full
      SCOPE_OK=1; SEEDS=""
      while IFS= read -r f; do
        [ -n "$f" ] || continue
        pkg=$(go list "./$(dirname "$f")" 2>/dev/null) \
          || { SCOPE_OK=0; break; }
        SEEDS="$SEEDS $pkg"
      done <<< "$(printf '%s\n' "$CHANGED" | grep '\.go$' || true)" # tolerate: 空集合法（mig/proto-only 面）
      # 迁移面定向（T-0433③）：fin_/md_ 迁移→pgstore migrate 测+受影响包
      # （check_migrations 闸在上段门禁位已恒跑）
      if [ "$SCOPE_OK" = 1 ] && [ "$HAS_MIG" = 1 ]; then
        pkg=$(go list ./internal/store/pgstore 2>/dev/null) \
          || SCOPE_OK=0
        [ "$SCOPE_OK" = 1 ] && SEEDS="$SEEDS $pkg"
      fi
      # proto 面定向：proto/<dir>/*→o/internal/gen/<dir> 包映射+buf build 闸
      if [ "$SCOPE_OK" = 1 ] && [ "$HAS_PROTO" = 1 ]; then
        command -v buf >/dev/null 2>&1 && buf build >/dev/null 2>&1 \
          || SCOPE_OK=0   # proto 闸：buf 缺席/编译败=判不准→full
        while IFS= read -r f; do
          [ -n "$f" ] || continue
          d=${f#proto/}; d=${d%/*}
          pkg=$(go list "./internal/gen/$d" 2>/dev/null) \
            || { SCOPE_OK=0; break; }
          SEEDS="$SEEDS $pkg"
        done <<< "$(printf '%s\n' "$CHANGED" | grep '^proto/' || true)" # tolerate: 空集合法
      fi
      if [ "$SCOPE_OK" = 1 ]; then
        IG=$(mktemp) || SCOPE_OK=0
        if [ "$SCOPE_OK" = 1 ]; then
          go list -f '{{.ImportPath}}|{{join .Imports ","}}|{{join .TestImports ","}}|{{join .XTestImports ","}}' ./... > "$IG" 2>/dev/null \
            || SCOPE_OK=0
        fi
        if [ "$SCOPE_OK" = 1 ]; then
          CLOSURE=$(python3 - "$IG" $SEEDS 2>/dev/null <<'PY'
import sys
ig, seeds = sys.argv[1], set(sys.argv[2:])
rev = {}
for line in open(ig):
    parts = line.rstrip("\n").split("|")
    if len(parts) < 4:
        continue
    for d in parts[1].split(",") + parts[2].split(",") + parts[3].split(","):
        if d:
            rev.setdefault(d, set()).add(parts[0])
seen, stack = set(seeds), list(seeds)
while stack:
    for imp in rev.get(stack.pop(), ()):
        if imp not in seen:
            seen.add(imp); stack.append(imp)
print(" ".join(sorted(seen)))
PY
) && [ -n "$CLOSURE" ] || SCOPE_OK=0
        fi
        [ -n "${IG:-}" ] && rm -f "$IG"
      fi
      if [ "$SCOPE_OK" = 1 ]; then
        # 恒测册包集展开（败=回退 full——册内包消失=结构变了不猜）
        AP=$(go list $ALWAYS_PKGS 2>/dev/null) || SCOPE_OK=0
        if [ "$SCOPE_OK" = 1 ]; then
          if [ "$HAS_MIG" = 1 ] || [ "$HAS_PROTO" = 1 ]; then
            MODE=merge-scoped
          else
            MODE=scoped
          fi
          TEST_PKGS=$(printf '%s\n%s\n' "$CLOSURE" "$AP" | tr ' ' '\n' | grep -v '^$' | sort -u | tr '\n' ' ')
        fi
      fi
      [ "$SCOPE_OK" = 1 ] || MODE=full
    fi
  fi
fi

# 坑处置（条目 pipefail-grep-q-early-exit-false-negative）：pipefail
# 下 producer|grep -q 早退收 SIGPIPE=141 假阴——docker ps 先收变量
# 再 herestring 判，两处消费面同式。
PG_NAMES=$(docker ps --format '{{.Names}}' 2>/dev/null || true) # tolerate: docker 缺席/查询败=按无 test PG 处理（DSN 不注入）

# T-0435 件②：测试库租约——非 docs-only 才占库（字母族池化独占领取，
# DSN 注入与租约同源禁租A注B；池空不猜不占 skip 径；详见 testdb_lease.sh 头注）。
# 老双注入块（grep -q herestring 判 5434 直注）在此归并入租约件单点。
if [ "$MODE" != "docs-only" ]; then
  source "$(pwd -P)/scripts/testdb_lease.sh"
  testdb_lease_acquire "$PG_NAMES" || fail "测试库租约败——见上方 holder 行"
fi

# T-0435 件③：go test 红转判定——超时败（panic: test timed out）且栈卡
# DB（jackc/pgx 帧）→$STAMP.mode 附 env-contention 标（假红可区分于码红，
# 审裁读戳面 T-0397 同形；不禁真超时断言——非 pgx 栈的超时不标）。
# 输出经 tee 全吞落盘（KB pipefail-grep-q 修法②同形：下游不早退上游 rc
# 真传；标判 grep 走文件输入无管道无 SIGPIPE 雷）。
run_go_test() { # $1=fail-msg $2..=go test 参数（包集/./...）
  local msg="$1" out rc=0
  shift # fail-msg 出参数表——go test 实参从 $2 起（混入=malformed import path 假红）
  out=$(mktemp) || fail "mktemp 败——测试输出无处落"
  go test "$@" -timeout=1800s 8>&- 2>&1 | tee "$out" >&2 || rc=$? # 8>&-：租约 fd 不传子进程（泄漏测试进程不得持库锁）
  if [ "$rc" -ne 0 ]; then
    if grep -q 'panic: test timed out' "$out" && grep -q 'jackc/pgx' "$out"; then
      printf 'env-contention\n' >> "$STAMP.mode"
      echo "⚠️ [check-all] env-contention：go test 超时且栈卡 DB（pgx 帧）——假红标已附 .mode（重跑处置；DB 归属争议时对上方 testdb-lease 行）" >&2
    fi
    rm -f "$out"
    fail "$msg"
  fi
  rm -f "$out"
}

if [ "$MODE" = docs-only ]; then
  echo "── go test 全谱：docs-only 豁免跳过——已提交面 ${NCOM} 件+工作树面 ${NWT} 件全中白名单（docs/**、*.md、skills/**），merge-base=$MB；CHECK_ALL_FORCE_FULL=1 压过" >&2
  echo "docs-only" > "$STAMP.mode"
elif [ "$MODE" = scoped ] || [ "$MODE" = merge-scoped ]; then
  echo "── go test：$MODE 定向谱——变更包+反向依赖闭包+恒测册（risk/ledger/trading/guard/lifecycle）$( [ "$MODE" = merge-scoped ] && echo '+迁移pgstore/proto-gen附加种子' )；包集: $TEST_PKGS" >&2
  { echo "$MODE"; printf '%s\n' $TEST_PKGS; } > "$STAMP.mode"
  run_go_test "go test 红（$MODE）" $TEST_PKGS
else
  echo "full" > "$STAMP.mode"
  step "go test 全谱"
  # T-0212：去 -count=1 放行 Go 测试缓存——同 tree 复测近零本（业主令：全量
  # uncached 25-40min/次太贵）。缓存键=文件+依赖指纹非 git 态——外部依赖型
  # （真实 PG/venue）陈判窗口钦定放行（CI 服务端仍全谱 uncached 兜真伪）。
  run_go_test "go test 红" ./...
fi

if [ "$BRIDGE_DEFERRED" = 1 ]; then
  printf 'bridge-audit-deferred\n' > "$STAMP.deferred"
  echo "⏸ [check-all] 全链绿（bridge-audit 延后未验）——不落 check-green 戳；本盒 accept 补跑该闸绿后落" >&2
  exit 0
fi
date -Is > "$STAMP"
echo "✅ [check-all] 全链绿（mode=$MODE）——戳落 $STAMP" >&2
