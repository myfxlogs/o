#!/usr/bin/env bash
# gc_test_schemas.sh —— T-0435 件①：测试库卫生清扫（三扫周期件；日拍 cron
# 05:23 契约不变——文件名不动=crontab 零手术）。原 T-0186 乙案（主实例
# t_* schema 日拍清）扩全语义：
#   扫① 陈旧事务：idle in transaction > GC_STALE_TX_MIN（默认30）分钟 →
#        pg_terminate_backend。active 查询永不杀（宁可漏不可错杀——
#        30min=go test 单包超时上界，超之的 idle-in-tx 不存在合法主体：
#        属主测试进程若活必已在 30min 撞闸死）；active 长事务只留痕观察。
#   扫② 孤儿 schema：t_* 族——带 _gc_born 出生标（testutil T-0435 起自
#        动落）且出生 >GC_SCHEMA_AGE_H（默认24）小时→回收；无标（历史
#        遗留/手工件）→T-0186 原语义照清（日拍全清先例在案，非收紧）。
#   扫③ 死元组：库级 Σn_dead_tup > GC_DEAD_TUP_MIN（默认200000 行）→
#        VACUUM (ANALYZE)（在线形，无锁升级；VACUUM FULL=一次性手术
#        不周期化——T-0426 10GB→15MB 那种深膨胀留人工径）。
#   扫④ 遗留库：o_test_<后缀>（KB parallel-checkall 专库隔离径遗留件）
#        零活连接→DROP；有连接跳过（宁可漏）。
# 目标=字母族测试实例（o-postgres-test + -b..-e，端口 5434-5438 白名单，
# 或 GC_DSN 单指测试族实例）——**生产面禁越界**：端口出白名单/db 非
# o_test 即拒不连（fail-closed rc2）；public schema 永不进清面（relic
# 处置=决策/管理席运维待办，条目 test-instance-public-relic-shadow）。
# 行为留痕：stdout 结构化行 `[gc-testdb] <ts> ...`（cron 重定向
# /opt/o-backups/gc_test_schemas.log——票面「ops_event 或日志戳」取日志戳，
# 不写生产 ops_event 面）。
# 用法：gc_test_schemas.sh [--dry-run]（默认 apply——cron 契约不变）。
set -uo pipefail

DRY=0
[ "${1:-}" = "--dry-run" ] && DRY=1

STALE_TX_MIN="${GC_STALE_TX_MIN:-30}"
SCHEMA_AGE_H="${GC_SCHEMA_AGE_H:-24}"
DEAD_TUP_MIN="${GC_DEAD_TUP_MIN:-200000}"
FAMILY_PORTS="${GC_FAMILY_PORTS:-5434 5435 5436 5437 5438}"

log() { echo "[gc-testdb] $(date -Is) $*"; }

# ── 单跑互斥（cron 与手工并发防自撞；在途 skip 非排队）──
exec 7>/tmp/o-gc-testdb.lock
if ! flock -n 7; then log "另一 gc-testdb 跑在途——skip"; exit 0; fi

# ── 目标 DSN 集：GC_DSN 单指（测试缝）或家族全扫 ──
declare -a DSNS=()
if [ -n "${GC_DSN:-}" ]; then
  DSNS+=("$GC_DSN")
else
  for p in $FAMILY_PORTS; do DSNS+=("postgres://o:o@localhost:$p/o_test?sslmode=disable"); done
fi
[ "${#DSNS[@]}" -gt 0 ] || { echo "❌ [gc-testdb] 目标集空（GC_FAMILY_PORTS/GC_DSN 均未定）" >&2; exit 2; }

# ── 守卫：端口白名单 + db=o_test（生产禁越界 fail-closed，票面钦定）──
for dsn in "${DSNS[@]}"; do
  port=""; db=""
  [[ "$dsn" =~ :([0-9]+)/ ]] && port="${BASH_REMATCH[1]}"
  [[ "$dsn" =~ /([A-Za-z0-9_]+)(\?|$) ]] && db="${BASH_REMATCH[1]}"
  ok=0
  for p in $FAMILY_PORTS; do [ "$port" = "$p" ] && ok=1; done
  if [ "$ok" != 1 ] || [ "$db" != "o_test" ]; then
    echo "❌ [gc-testdb] 守卫拒：DSN 出测试族白名单（port=$port db=$db）——生产面禁越界" >&2
    exit 2
  fi
done

# ── 逐实例三扫 ──
alive=0
for dsn in "${DSNS[@]}"; do
  port=""; [[ "$dsn" =~ :([0-9]+)/ ]] && port="${BASH_REMATCH[1]}"
  if ! psql "$dsn" -Atqc 'SELECT 1' >/dev/null 2>&1; then
    log "inst:$port 连不上（容器缺席/未起）——skip（宁可漏）"
    continue
  fi
  alive=$((alive + 1))
  q() { psql "$dsn" -Atqc "$1"; } # 实例内快查

  # 扫① 陈旧事务（dry-run 只列不杀）
  TXWHERE="datname=current_database() AND pid<>pg_backend_pid()
    AND state IN ('idle in transaction','idle in transaction (aborted)')
    AND xact_start < now() - make_interval(mins => $STALE_TX_MIN)"
  if [ "$DRY" = 1 ]; then
    while IFS='|' read -r pid xs qy; do
      [ -n "$pid" ] || continue
      log "inst:$port [dry] 陈旧事务 pid=$pid xact_start=$xs q=${qy:0:50}——apply 将 terminate"
    done < <(q "SELECT pid||'|'||COALESCE(xact_start::text,'?')||'|'||left(regexp_replace(query,E'[\\n\\r\\t]+',' ','g'),80) FROM pg_stat_activity WHERE $TXWHERE")
  else
    n=0
    while IFS='|' read -r pid xs qy; do
      [ -n "$pid" ] || continue
      n=$((n + 1))
      log "inst:$port 陈旧事务回收 pid=$pid xact_start=$xs q=${qy:0:50}（>$STALE_TX_MIN min idle in tx）"
    done < <(q "SELECT pid||'|'||COALESCE(xact_start::text,'?')||'|'||left(regexp_replace(query,E'[\\n\\r\\t]+',' ','g'),80) FROM pg_stat_activity WHERE $TXWHERE AND pg_terminate_backend(pid)")
    [ "$n" = 0 ] || log "inst:$port 扫①回收 $n 事务"
  fi
  # active 长事务观察行（不杀——留痕供人工判）
  while IFS='|' read -r pid st xs qy; do
    [ -n "$pid" ] || continue
    log "inst:$port 观察：active 长事务 pid=$pid state=$st xact_start=$xs q=${qy:0:50}（不杀——宁可漏）"
  done < <(q "SELECT pid||'|'||state||'|'||COALESCE(xact_start::text,'?')||'|'||left(regexp_replace(query,E'[\\n\\r\\t]+',' ','g'),80) FROM pg_stat_activity WHERE datname=current_database() AND pid<>pg_backend_pid() AND state='active' AND xact_start < now() - make_interval(mins => $STALE_TX_MIN)")

  # 扫② 孤儿 schema（t_*：带 _gc_born 标超龄收；无标=T-0186 原语义收）
  while IFS= read -r sch; do
    [ -n "$sch" ] || continue
    [[ "$sch" =~ ^t_[A-Za-z0-9_]+$ ]] || { log "inst:$port schema $sch 出 t_ 命名域——跳过（保守）"; continue; }
    born_state=$(q "SELECT CASE WHEN max(ts) < now() - make_interval(hours => $SCHEMA_AGE_H) THEN 'old' ELSE 'fresh' END FROM $sch._gc_born" 2>/dev/null) || born_state=""
    case "$born_state" in
      fresh) : ;; # 出生 24h 内——在租豁免（宁可漏）
      old)   if [ "$DRY" = 1 ]; then log "inst:$port [dry] 孤儿 schema $sch（_gc_born 超龄）——apply 将 DROP CASCADE"
             else q "DROP SCHEMA $sch CASCADE" >/dev/null && log "inst:$port 孤儿 schema 回收 $sch（_gc_born 超龄 >${SCHEMA_AGE_H}h）"; fi ;;
      *)     if [ "$DRY" = 1 ]; then log "inst:$port [dry] 无标 schema $sch——apply 将 DROP CASCADE（T-0186 原语义）"
             else q "DROP SCHEMA $sch CASCADE" >/dev/null && log "inst:$port 无标 schema 回收 $sch（T-0186 原语义：日拍全清）"; fi ;;
    esac
  done < <(q "SELECT nspname FROM pg_namespace WHERE nspname LIKE 't\\_%'")

  # 扫③ 死元组
  dead=$(q "SELECT COALESCE(SUM(n_dead_tup),0) FROM pg_stat_user_tables")
  case "$dead" in ''|*[!0-9]*) dead=0 ;; esac
  if [ "$dead" -gt "$DEAD_TUP_MIN" ]; then
    if [ "$DRY" = 1 ]; then
      log "inst:$port [dry] 死元组 $dead > $DEAD_TUP_MIN——apply 将 VACUUM (ANALYZE)"
    else
      log "inst:$port 死元组 $dead > $DEAD_TUP_MIN——VACUUM (ANALYZE) 起"
      if psql "$dsn" -qc "VACUUM (ANALYZE)" >/dev/null 2>&1; then
        after=$(q "SELECT COALESCE(SUM(n_dead_tup),0) FROM pg_stat_user_tables")
        log "inst:$port VACUUM 完成（dead $dead → $after）"
      else
        log "inst:$port VACUUM 败（库忙?）——下轮重扫（宁可漏）"
      fi
    fi
  fi

  # 扫④ 遗留库（o_test_<后缀> 专库径遗留；零活连接才收）
  while IFS= read -r dbn; do
    [ -n "$dbn" ] || continue
    conns=$(q "SELECT count(*) FROM pg_stat_activity WHERE datname='$dbn' AND pid<>pg_backend_pid()")
    case "$conns" in ''|*[!0-9]*) conns=9 ;; esac # 判不准=按有连接保守
    if [ "$conns" = 0 ]; then
      if [ "$DRY" = 1 ]; then log "inst:$port [dry] 遗留库 $dbn（0 活连接）——apply 将 DROP DATABASE"
      else q "DROP DATABASE $dbn" >/dev/null && log "inst:$port 遗留库回收 $dbn（0 活连接）"; fi
    else
      log "inst:$port 库 $dbn 有 $conns 活连接——跳过（宁可漏）"
    fi
  done < <(q "SELECT datname FROM pg_database WHERE datname ~ '^o_test_[a-z0-9_]+$'")
done

if [ "$alive" = 0 ]; then
  log "全实例不可达——本轮零清扫（rc1 可观测）"
  exit 1
fi
log "done（实例 $alive/${#DSNS[@]}；dry=$DRY）"
