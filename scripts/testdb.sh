#!/usr/bin/env bash
# testdb.sh —— 每席一次性测试库租据（T-0266 B 案业主裁：绿戳 DSN 隔离，5434 降兜底）
#
# 池位名册（端口段钦定——不撞 5433 生产/5434 共享兜底/5544 隧道）：
#   slot b→o-postgres-test-b@127.0.0.1:5435  c→5436  d→5437  e→5438
#        f→5439  g→5440  h→5441  i→5442                     （共 8 席）
# 租据态（仓外运行面，全树共享）：${O_TESTDB_HOME:-/opt/meta/testdb}/
#   leases/<slot> = "holder<TAB>pid<TAB>ts"（活约）；last/<slot> = 末任 holder。
#   holder 更替即 DROP+CREATE o_test 库=一次性语义；死 pid 残约 alloc 内联回收。
# 用法：
#   testdb.sh alloc <holder> [pid]   出租据、stdout 打 DSN（同 holder 幂等；
#                                  pid 缺省=$PPID——headless 显传 wrapper $$，
#                                  交互显传壳 $$；残约随 pid 死自收）
#   testdb.sh release <holder>       退租（重置延至下任 holder 租用时——退出径热）
#   testdb.sh status                 名册现态（容器/活约/末任一屏）
#   testdb.sh purge                  收死 pid 残约（兜底清扫——alloc 本已内联收）
# 失败语义：alloc 打空串 rc=1——调用方不设 O_TEST_PG_DSN 即落 5434 共享兜底
#   （并行失、串行正确性保）。在飞席不可中途改 DSN——注入只许发生
#   在 spawn 面（wrapper/会话起手），本器自身无任何改在飞进程 env 径。
set -uo pipefail

TDB_HOME="${O_TESTDB_HOME:-/opt/meta/testdb}"
LEASES="$TDB_HOME/leases"; LASTS="$TDB_HOME/last"; LOCK="$TDB_HOME/.lock"
IMG="timescale/timescaledb:2.30.2-pg18"
SLOTS=(b:5435 c:5436 d:5437 e:5438 f:5439 g:5440 h:5441 i:5442)

warn(){ echo "[testdb] $*" >&2; }
die(){ warn "$*"; exit 1; }
usage(){ sed -n '2,22p' "$0" >&2; exit 2; }
valid_holder(){ [[ "$1" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$ ]]; }
port_of(){ local sp; for sp in "${SLOTS[@]}"; do [ "${sp%%:*}" = "$1" ] && { echo "${sp##*:}"; return 0; }; done; return 1; }
dsn(){ echo "postgres://o:o@127.0.0.1:$1/o_test?sslmode=disable"; }

lease_live(){ # $1=slot：约存且 pid 活
  local f="$LEASES/$1" pid
  [ -f "$f" ] || return 1
  pid=$(cut -f2 "$f")
  [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null
}
slot_of_holder(){ # $1=holder → 活约 slot（死约视同无——打空串 rc=1）
  local sp s
  for sp in "${SLOTS[@]}"; do s=${sp%%:*}
    lease_live "$s" && [ "$(cut -f1 "$LEASES/$s")" = "$1" ] && { echo "$s"; return 0; }
  done; return 1
}
free_slot(){ # 打首个无约/死约 slot
  local sp s
  for sp in "${SLOTS[@]}"; do s=${sp%%:*}
    lease_live "$s" || { echo "$s"; return 0; }
  done; return 1
}
ensure_container(){ # $1=slot $2=port：缺造（spec 同 compose 钦定面）/停启/候 ready≤60s
  local name="o-postgres-test-$1" st t=60
  # inspect 失物时 stdout 仍吐空行——|| echo 追加会产出 "\nmissing" 穿 case，
  # 故 || true + 空串判缺（实证 2026-10-30）。
  st=$(docker inspect -f '{{.State.Status}}' "$name" 2>/dev/null || true); st=${st:-missing}
  case "$st" in
    running) ;;
    missing) docker run -d --name "$name" --restart unless-stopped \
        -p "127.0.0.1:$2:5432" --shm-size=1g --memory=2g --cpus=2 \
        -e POSTGRES_USER=o -e POSTGRES_PASSWORD=o -e POSTGRES_DB=o_test \
        "$IMG" postgres -c max_connections=500 \
        -c shared_preload_libraries=timescaledb,pg_stat_statements \
        >/dev/null || { warn "$name 起容器败"; return 1; } ;;
    *) docker start "$name" >/dev/null || { warn "$name start 败"; return 1; } ;;
  esac
  # 就绪判=真连 o_test select 1——pg_isready 在 entrypoint 临时服务器相即绿，
  # POSTGRES_DB 尚未建完即放 reset_db 会撞 init 并发（实证 2026-10-30：
  # DROP 扑空 NOTICE + CREATE 撞 pg_database 唯一键）。
  while [ $t -gt 0 ]; do
    [ "$(docker exec "$name" psql -U o -d o_test -tAc 'select 1' 2>/dev/null)" = "1" ] && return 0
    sleep 1; t=$((t-1))
  done
  warn "$name o_test 就绪逾 60s"; return 1
}
reset_db(){ # holder 更替=一次性语义：断连+drop+create
  docker exec "o-postgres-test-$1" psql -U o -d postgres -v ON_ERROR_STOP=1 -q \
    -c "DROP DATABASE IF EXISTS o_test WITH (FORCE);" \
    -c "CREATE DATABASE o_test;" >/dev/null
}

cmd_alloc(){
  local h=${1:-} pid=${2:-$PPID} s port
  valid_holder "${h:-}" || die "holder 非法：${h:-<空>}"
  command -v docker >/dev/null || die "docker 缺席——落兜底"
  mkdir -p "$LEASES" "$LASTS"
  exec 7>"$LOCK"; flock 7 || die "租据锁取败"
  s=$(slot_of_holder "$h" || true)
  if [ -z "$s" ]; then
    s=$(free_slot || true)
    [ -n "$s" ] || die "池尽（${#SLOTS[@]} 席全活）——落兜底"
  fi
  port=$(port_of "$s")
  ensure_container "$s" "$port" || die "slot $s 库未备——落兜底"
  if [ "$(cat "$LASTS/$s" 2>/dev/null || true)" != "$h" ]; then
    reset_db "$s" || die "slot $s 库重置败——落兜底"
    echo "$h" > "$LASTS/$s"
  fi
  printf '%s\t%s\t%s\n' "$h" "$pid" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$LEASES/$s"
  dsn "$port"
}
cmd_release(){
  local h=${1:-} s
  valid_holder "${h:-}" || die "holder 非法：${h:-<空>}"
  mkdir -p "$LEASES" "$LASTS"
  exec 7>"$LOCK"; flock 7 || die "租据锁取败"
  s=$(slot_of_holder "$h" || true)
  [ -n "$s" ] || { warn "无活约：$h"; return 0; }
  rm -f "$LEASES/$s"; warn "$h 退租 slot $s"
}
cmd_status(){
  local sp s p st l pid
  printf '%-5s %-6s %-22s %-14s %-9s %s\n' slot port container lease pid last
  for sp in "${SLOTS[@]}"; do s=${sp%%:*}; p=${sp##*:}
    st=$(docker inspect -f '{{.State.Status}}' "o-postgres-test-$s" 2>/dev/null || true); st=${st:-缺席}
    l="-"; pid="-"
    if [ -f "$LEASES/$s" ]; then
      l=$(cut -f1 "$LEASES/$s"); pid=$(cut -f2 "$LEASES/$s")
      kill -0 "$pid" 2>/dev/null || l="$l(死约)"
    fi
    printf '%-5s %-6s %-22s %-14s %-9s %s\n' "$s" "$p" "$st" "$l" "$pid" "$(cat "$LASTS/$s" 2>/dev/null || echo -)"
  done
}
cmd_purge(){
  local sp s n=0
  mkdir -p "$LEASES" "$LASTS"
  exec 7>"$LOCK"; flock 7 || die "租据锁取败"
  for sp in "${SLOTS[@]}"; do s=${sp%%:*}
    if [ -f "$LEASES/$s" ] && ! lease_live "$s"; then rm -f "$LEASES/$s"; n=$((n+1)); warn "收死约 slot $s"; fi
  done
  warn "purge 完——收 $n 约"
}

case "${1:-}" in
  alloc) shift; cmd_alloc "$@" ;;
  release) shift; cmd_release "$@" ;;
  status) cmd_status ;;
  purge) cmd_purge ;;
  *) usage ;;
esac
