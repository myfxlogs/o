#!/usr/bin/env bash
# dispatch_order.sh —— 派单序单源（GW-001 段C3）：stdin=票号行集（dm_queue
# approved/dispatched 面原样）→stdout=按 (优先级rank,已批ts,票号) 升序重排。
# rank 机检链（族机械映射票面钦定）：
#   票面「优先级」字段显式 P0/P1/P3 覆写（CTO/DM 裁时改判——P0=0/P1=1/P3=3）
#   > 族映射 postmortem:*/rootcause:*=0 > 写败:*/账链断:*=1 > 观测盲:*=2
#   > 其余/缺签名/缺票件=3（未知族垫底——已知急件先行，机检保守序）；
#   P2=起草默认档=不覆写（族映射定 rank——模板行「DM 裁时可调」P0/P1/P3
#   才是显式改判）。
# 已批ts=最新已批行 ts 秒升序（先批先派 FIFO；无行/查败=0 同 rank 垫底）；
# 同 rank 同 ts=票号序。DM_ENV_FILE 缺/psql 败=ts 全 0 仍按 rank 排（排序
# 是优化非链环——fail-open 原序保底派单主链不断）。
# 非法词形（非 T-NNNN——dispatch_worker 闸自拒面）原序垫尾透传。
# 用法: dispatch_order.sh   env: DM_ENV_FILE + O_DISPATCH_ORDER_REPO(缺省
#   /opt/o) + O_DISPATCH_ORDER_TASK_DIR(缺省 $REPO/docs/tasks)
set -u
REPO=${O_DISPATCH_ORDER_REPO:-/opt/o}
TASK_DIR=${O_DISPATCH_ORDER_TASK_DIR:-$REPO/docs/tasks}
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')

TMPD=$(mktemp -d "${TMPDIR:-/tmp}/dispatch-order.XXXXXX") || exit 1
trap 'rm -rf -- "$TMPD"' EXIT

IN=$(cat)
[ -n "$IN" ] || exit 0 # 空入空出

rank_of() { # $1=签名 $2=票面优先级字段 → rank 数字
  case "$2" in
    P0) echo 0; return ;;
    P1) echo 1; return ;;
    P3) echo 3; return ;;
  esac
  case "$1" in
    postmortem:*|rootcause:*) echo 0 ;;
    写败:*|账链断:*) echo 1 ;;
    观测盲:*) echo 2 ;;
    *) echo 3 ;;
  esac
}

VALID=()
while IFS= read -r tid; do
  [ -n "$tid" ] || continue
  case "$tid" in
    T-[0-9][0-9][0-9][0-9]) VALID+=("$tid") ;;
    *) printf '%s\n' "$tid" >> "$TMPD/invalid" ;;
  esac
done <<< "$IN"

# 已批ts 批量单查（IN 列表票号已白名单校验零注入面）；查败=全 0 fail-open。
declare -A TS=()
if [ "${#VALID[@]}" -gt 0 ] && [ -n "$FIN_DSN" ]; then
  INLIST=$(printf ",'%s'" "${VALID[@]}")
  ROWS=$(psql "$FIN_DSN" -qAt -c \
    "SELECT ticket,COALESCE(extract(epoch from max(ts))::bigint,0) FROM ops_event
     WHERE event='已批' AND ticket IN (${INLIST#,}) GROUP BY ticket" 2>/dev/null)
  if [ $? -eq 0 ]; then
    while IFS='|' read -r t s; do [ -n "$t" ] && TS["$t"]="$s"; done <<< "$ROWS"
  fi
fi

for tid in "${VALID[@]}"; do
  file=$(compgen -G "$TASK_DIR/$tid-*.md" 2>/dev/null | head -1)
  sig=
  prio=
  if [ -n "$file" ] && [ -r "$file" ]; then
    # 签名提取禁单反引号字面（\x60 等价径——dm_autoapprove 同式同源）。
    sig=$(sed -n 's/.*病灶签名：\x60\([^\x60]*\)\x60.*/\1/p' "$file" | head -1)
    prio=$(sed -n 's/^- 优先级：\(P[0-9]\).*/\1/p' "$file" | head -1)
  fi
  ts=${TS[$tid]:-0}
  printf '%s\t%s\t%s\n' "$(rank_of "$sig" "$prio")" "${ts:-0}" "$tid" >> "$TMPD/rows"
done
if [ -f "$TMPD/rows" ]; then
  LC_ALL=C sort -k1,1n -k2,2n -k3,3 "$TMPD/rows" | awk -F'\t' '{print $3}'
fi
[ -f "$TMPD/invalid" ] && cat "$TMPD/invalid"
exit 0
