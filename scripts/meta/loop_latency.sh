#!/usr/bin/env bash
# loop_latency.sh —— GW-001 A5 内循环 KPI 取数件（gauges.tsv cmd 径）：
# 案 ts→部署行 ts 中位（秒）。口径钉死：
#   部署行=ops_event event='部署' note 'deploy <sha12> rc=0'（镜像面——
#     oci 二进制面不入 KPI，口径在案）；
#   批域=相邻部署行 sha 区间 prev..cur 的 commit subject 提 T-NNNN
#     （DESC 取近 10 行配 9 段——最旧行无前锚跳过）；
#   案 ts=票级案锚代理：fin_ticket_draft∪ops_event 同票最早行 min(ts)
#     （段E 案卷 deployed: 回写落地后可换 fin_econ_lesson 真锚——
#     本件=段A 可算代理如实标注不冒充）；
#   负延迟（案锚晚于部署行=数据异常）剔除不入中位。
# 输出=中位秒整数；空样本/取数败=0（量表阈下不误报——fetch_error
# 显形径只留给真取数败）。
set -u
REPO=${O_LOOP_REPO:-/opt/o}
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$FIN_DSN" ] || { echo 0; exit 0; }
mapfile -t ROWS < <(psql "$FIN_DSN" -qAt -F '|' -c \
  "SELECT extract(epoch from ts)::bigint, substring(note from 'deploy ([0-9a-f]+)')
   FROM ops_event WHERE event='部署' AND note LIKE 'deploy % rc=0 %'
   ORDER BY ts DESC LIMIT 10" 2>/dev/null) # tolerate: 查询败=空集→0 阈下
[ "${#ROWS[@]}" -ge 2 ] || { echo 0; exit 0; }
LAT=""
for ((i=0; i<${#ROWS[@]}-1; i++)); do
  IFS='|' read -r ets sha _ <<<"${ROWS[$i]}"
  IFS='|' read -r _ prevsha <<<"${ROWS[$i+1]}"
  [ -n "$ets" ] && [ -n "$sha" ] && [ -n "$prevsha" ] || continue
  TICKETS=$(git -C "$REPO" log --format=%s "${prevsha}..${sha}" 2>/dev/null \
    | grep -oE 'T-[0-9]{4}' | sort -u) # tolerate: git 区间不可解=空票集跳段
  [ -n "$TICKETS" ] || continue
  TIN=$(printf '%s\n' "$TICKETS" | sed "s/.*/'&'/" | paste -sd, -)
  while IFS='|' read -r tid cts; do
    [ -n "$tid" ] && [ -n "$cts" ] || continue
    lat=$(( ets - cts ))
    [ "$lat" -gt 0 ] && LAT="$LAT $lat"
  done < <(psql "$FIN_DSN" -qAt -F '|' -c \
    "SELECT ticket, min(extract(epoch from ts))::bigint FROM (
       SELECT ticket, ts FROM fin_ticket_draft WHERE ticket IN ($TIN)
       UNION ALL
       SELECT ticket, ts FROM ops_event WHERE ticket IN ($TIN)
     ) u GROUP BY ticket" 2>/dev/null) # tolerate: 票级案锚查询败=该段跳（不炸整件）
done
[ -n "$LAT" ] || { echo 0; exit 0; }
printf '%s\n' $LAT | sort -n | awk '{a[NR]=$1} END{if(NR==0)print 0; else{m=int((NR+1)/2); if(NR%2)print a[m]; else print int((a[m]+a[m+1])/2)}}'
