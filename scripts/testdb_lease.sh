#!/usr/bin/env bash
# testdb_lease.sh —— T-0435 件②：测试库租约（字母族池化独占领取）。
# 被 check_all.sh source（函数落调用 shell，导出 O_TEST_PG_DSN）。
# 单次调用语义：fd8 为租约持锁 fd——同 shell 二次调用会顶掉首租，禁。
#
# 协议（票面钦定，KB parallel-checkall-shared-testdb-migrate-hang 根修）：
#   池=o-postgres-test(5434)+b..e(5435..5438) 按容器名实存映射；
#   领取=flock -n 试锁 $TESTDB_LEASE_LOCKDIR/o-testdb-lease-<port>.lock，
#   先到先得；全忙→排队等首选库（O_TESTDB_LEASE_WAIT 秒，默认 1800 同
#   check_all 树锁纪律）超时报 holder fail——排队非互踩；
#   优先序=树 hash（TOPKEY）旋转（端口固定映射树 hash——两树同起少撞首选）；
#   DSN 注入与租约同源双向推导（外部 DSN→按其端口锁同文件；池内领取→
#   DSN 从锁得端口拼出）——禁「租A注B」。
# fail-closed（票面④）：池空/docker 枚举败→不注入 DSN（DB 桩测试 skip 径，
#   不猜不占）+warn；锁面文件开不了（判不准）→落共享库 5434+warn——
#   争用允许但可观测。
# 锁生命周期=进程退出自动释（flock fd），check_all 退出即还库。
# pipefail 纪律（条目 pipefail-grep-q-early-exit-false-negative）：本件
#   grep 全走 herestring，无「produce|grep -q」管道形。
# 并发语义注记：租约判锁全部落当前 shell（禁 $() 子壳持锁——子壳退锁即释）。

TESTDB_LEASE_LOCKDIR="${TESTDB_LEASE_LOCKDIR:-/tmp}" # 测试缝：锁文件目录可指沙箱
TESTDB_LEASE_WAIT="${TESTDB_LEASE_WAIT:-1800}"       # 排队上界（秒）

# _testdb_lease_try <port> —— 当前 shell 试锁一库；
# rc0=租到（fd8 持锁），rc1=忙，rc2=锁面坏（判不准）
# 坑位（条目 bash-local-multi-assign-expansion-order）：port 与 lock 的
# local 声明必须分行——同行多赋值时 lock 词先于 port 赋值展开（$port 沿
# 动态域取调用方未置局部=空→锁文件名丢端口→互斥失效）。
_testdb_lease_try() {
  local port="$1"
  local lock="$TESTDB_LEASE_LOCKDIR/o-testdb-lease-$port.lock"
  # 坑位（exec 持久重定向）：exec 重定向落本 shell 永久生效——此处禁搭
  # 2>/dev/null（会把调用方 stderr 整体永久吞掉=后续 fail/warn 全哑）。
  # 开文件败让 bash 自报错误→rc2 判不准径（fail-closed 可观测）。
  if ! exec 8>>"$lock"; then return 2; fi
  if ! flock -n 8 2>/dev/null; then exec 8>&-; return 1; fi
  return 0
}

# _testdb_lease_queue <port> —— 当前 shell 排队等指定库；rc0=租到，rc1=超时/锁面坏
# （local 分行同上坑位；exec 2>/dev/null 禁搭同上）
_testdb_lease_queue() {
  local port="$1"
  local lock="$TESTDB_LEASE_LOCKDIR/o-testdb-lease-$port.lock"
  exec 8>>"$lock" || return 1
  if flock -w "$TESTDB_LEASE_WAIT" 8 2>/dev/null; then return 0; fi
  exec 8>&-
  return 1
}

# _testdb_lease_holder <port> —— 锁 holder 报告（fuser 败/空集=如实 ?）
_testdb_lease_holder() {
  fuser "$TESTDB_LEASE_LOCKDIR/o-testdb-lease-$1.lock" 2>/dev/null \
    | tr -s ' ' '\n' | grep -v "^$$\$" | head -3 | tr '\n' ' ' || true # tolerate: fuser 败/空集=holder 未知如实报 ?（check_all 树锁同形）
}

# testdb_lease_acquire <PG_NAMES> —— 主入口：租约+DSN 单源导出。
# rc0=租到或合法不占；rc1=排队超时（调用方 fail）。
testdb_lease_acquire() {
  local names="${1:-}" dsn="${O_TEST_PG_DSN:-}" port rc
  # T-0468 族内分库：端口分租之后库内再分片——O_TEST_SHARDS=K 时
  # testutil 按调用方包目录散列落 o_test_s1..sK（pg advisory 锁域
  # 按库隔离=跨库并发 Migrate 互不排队，herd 消）；K=1/缺省=现行
  # 单库语义。分库由 testutil 首调自确保（CREATE DATABASE 败落
  # 基库 o_test=只慢不语义错），本件不直接建库。
  # 缺省 16：go test -p≈GOMAXPROCS(12) 并发包各得一库余量——
  # A/B 实测 K=8→16 五包墙钟 4m30→2m16（-59% vs 单库基线），
  # 再高无收益（并发上限封顶）只增建库面。
  export O_TEST_SHARDS="${O_TEST_SHARDS:-16}"
  # ── 外部 DSN 径（人工择闲，KB checkall-parallel-load-flaky-timing 惯例）：
  #    锁随 DSN 端口走——注什么锁什么，租A注A。
  if [ -n "$dsn" ]; then
    port=""
    [[ "$dsn" =~ @[^/:]+:([0-9]+)/ ]] && port="${BASH_REMATCH[1]}"
    case "$port" in
      5434|5435|5436|5437|5438)
        if _testdb_lease_try "$port"; then
          echo "🔑 [testdb-lease] 外部 DSN 端口 $port——租约到手（独占，进程退自动释）" >&2
        elif _testdb_lease_queue "$port"; then
          echo "🔑 [testdb-lease] 外部 DSN 端口 $port——前租已退，排队到手" >&2
        else
          echo "❌ [testdb-lease] 端口 $port 租约等 ${TESTDB_LEASE_WAIT}s 超时——holder pid=[$(_testdb_lease_holder "$port")]（同库他跑仍在途）" >&2
          return 1
        fi ;;
      *)
        echo "⚠️ [testdb-lease] 外部 DSN 端口不在字母族（port=${port:-解析不出}）——不锁不改动 DSN，人工径自负隔离（留痕）" >&2
        ;;
    esac
    return 0
  fi
  # ── 池化领取径：容器名→族端口池（herestring 判，KB 管道形豁免）──
  local pool=() infra=0 i idx pref=0
  grep -q '^o-postgres-test$'   <<< "$names" && pool+=(5434)
  grep -q '^o-postgres-test-b$' <<< "$names" && pool+=(5435)
  grep -q '^o-postgres-test-c$' <<< "$names" && pool+=(5436)
  grep -q '^o-postgres-test-d$' <<< "$names" && pool+=(5437)
  grep -q '^o-postgres-test-e$' <<< "$names" && pool+=(5438)
  if [ "${#pool[@]}" = 0 ]; then
    echo "⚠️ [testdb-lease] 测试 PG 池空（docker 枚举败/容器缺席）——不猜不占，DSN 不注入（DB 桩测试 skip 径）" >&2
    return 0
  fi
  # 优先序=树 hash 旋转（TOPKEY 首段 hex mod 池长；缺席=0）
  [ -n "${TOPKEY:-}" ] && pref=$(( 16#${TOPKEY:0:4} % ${#pool[@]} ))
  for ((i = 0; i < ${#pool[@]}; i++)); do
    idx=$(((pref + i) % ${#pool[@]}))
    _testdb_lease_try "${pool[idx]}"; rc=$?
    if [ "$rc" = 0 ]; then
      port="${pool[idx]}"
      export O_TEST_PG_DSN="postgres://o:o@localhost:$port/o_test?sslmode=disable"
      echo "🔑 [testdb-lease] 租约：o-postgres-test 族端口 $port（独占；DSN 与租约同源——禁租A注B）" >&2
      return 0
    elif [ "$rc" = 2 ]; then
      infra=1; break # 锁面坏——判不准，不再试（票面④）
    fi
  done
  if [ "$infra" = 1 ]; then
    # 票面④：租约判不准→落共享库+warn 留痕（争用允许但可观测）
    export O_TEST_PG_DSN="postgres://o:o@localhost:5434/o_test?sslmode=disable"
    echo "⚠️ [testdb-lease] 锁面异常（$TESTDB_LEASE_LOCKDIR 不可写?）——判不准，落共享库 5434+warn（fail-closed 可观测）" >&2
    return 0
  fi
  # 全忙→排队等首选（票面②：无空位排队非互踩）
  port="${pool[pref]}"
  echo "⏳ [testdb-lease] 池全忙（${pool[*]}）——排队等 $port（上界 ${TESTDB_LEASE_WAIT}s，超时 fail 报 holder）" >&2
  if _testdb_lease_queue "$port"; then
    export O_TEST_PG_DSN="postgres://o:o@localhost:$port/o_test?sslmode=disable"
    echo "🔑 [testdb-lease] 排队到手：端口 $port（DSN 与租约同源）" >&2
    return 0
  fi
  echo "❌ [testdb-lease] 排队等 $port 超时 ${TESTDB_LEASE_WAIT}s——holder pid=[$(_testdb_lease_holder "$port")]（前跑仍在途或锁泄漏，ps 核查后重试）" >&2
  return 1
}
