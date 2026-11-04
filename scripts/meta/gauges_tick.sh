#!/usr/bin/env bash
# gauges_tick.sh —— T-0332 自愈巡检拍（零 LLM 纯判据——照 dm_tick 对形；
# 全表跑→触阈→ops_event '升件' 行（seat=巡检闸，note=量表名+实测+阈）；
# 滞回闸：每表连续2拍触才升+日频上限1——误报刷屏防；零待触=零行空跑如实）。
# 取数失败显形（T-0332 返修①）：rc≠0/stderr 非空→记 $ST/<name>.fetch_error
# 并挂 hb note fail 表名——失败自身即缺席类升件素材，绝不静默吞成绿灯。
set -u
ENVF="${GAUGES_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
RO_DSN=$(grep '^FIN_PG_RO_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
ST=/var/tmp/o-gauges; mkdir -p "$ST"

hb() { [ -n "$FIN_DSN" ] && psql "$FIN_DSN" -qAt -c "INSERT INTO ops_headless(ticket,seat,state,last_activity) VALUES('META-GAUGES','巡检闸','$1',now()) ON CONFLICT (ticket) DO UPDATE SET seat=EXCLUDED.seat,state=EXCLUDED.state,last_activity=now(),updated_at=now()" >/dev/null 2>&1 || true; }

[ -n "$FIN_DSN" ] || { hb stalled; exit 0; }
today=$(date +%F)
FAILS=""
while IFS=$'\t' read -r name cls qt qry th tpl; do
  [ -n "$name" ] || continue
  errf="$ST/${name}.fetch_error"
  v=""
  case "$qt" in
    sql_ro)
      if [ -z "$RO_DSN" ]; then echo "no FIN_PG_RO_DSN" > "$errf"; FAILS="$FAILS $name"; continue; fi
      v=$(psql "$RO_DSN" -qAt -c "$qry" 2>"$errf.tmp" | head -1); rc=$?
      if [ -s "$errf.tmp" ] || [ "$rc" != 0 ]; then mv "$errf.tmp" "$errf"; FAILS="$FAILS $name"; continue; fi
      rm -f "$errf.tmp" "$errf";;
    cmd)
      v=$(eval "$qry" 2>"$errf.tmp" | head -1); rc=$?
      if [ "$rc" != 0 ]; then mv "$errf.tmp" "$errf"; FAILS="$FAILS $name"; continue; fi
      rm -f "$errf.tmp" "$errf";;
    *) continue;;
  esac
  [ -n "$v" ] || { echo "empty result" > "$errf"; FAILS="$FAILS $name"; continue; }
  hit=0
  if [ "$th" -ge 0 ]; then [ "${v%.*}" -ge "$th" ] 2>/dev/null && hit=1
  else [ "${v%.*}" -lt 1 ] 2>/dev/null && hit=1; fi
  if [ "$hit" = 1 ]; then
    # 滞回闸：昨日也触（连续2拍）且今日未升过→升件行
    if [ -f "$ST/${name}.last_hit_date" ] && grep -qx "$(date -d yesterday +%F 2>/dev/null||echo x)" "$ST/${name}.last_hit_date" \
        && ! grep -qx "$today" "$ST/${name}.escalated" 2>/dev/null; then
      msg=$(echo "$tpl" | sed "s/{v}/$v/")
      psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note) VALUES('巡检闸','升件','$name','$msg 阈=$th')" >/dev/null 2>&1 || true
      echo "$today" >> "$ST/${name}.escalated"
    fi
    echo "$today" > "$ST/${name}.last_hit_date"
  else
    rm -f "$ST/${name}.last_hit_date"
  fi
done < "${GAUGES_TSV:-/opt/o/scripts/meta/gauges.tsv}"
# 取数失败显形：失败表名日频上限1升件行（缺席类素材），痕迹另落 $ST/*.fetch_error
if [ -n "$FAILS" ] && ! grep -qx "$today" "$ST/fetch_fail.escalated" 2>/dev/null; then
  psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note) VALUES('巡检闸','升件','META-GAUGES','量表取数失败（缺席类）：$FAILS —— 详见 /var/tmp/o-gauges/*.fetch_error')" >/dev/null 2>&1 \
    && echo "$today" >> "$ST/fetch_fail.escalated"
fi
hb alive
exit 0
