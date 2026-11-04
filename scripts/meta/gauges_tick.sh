#!/usr/bin/env bash
# gauges_tick.sh —— T-0332 自愈巡检拍（零 LLM 纯判据——照 dm_tick 对形；
# 全表跑→触阈→ops_event '升件' 行（seat=巡检闸，note=量表名+实测+阈）；
# 滞回闸：每表连续2拍触才升+日频上限1——误报刷屏防；零待触=零行空跑如实）。
set -u
ENVF="${GAUGES_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
ST=/var/tmp/o-gauges; mkdir -p "$ST"

hb() { [ -n "$FIN_DSN" ] && psql "$FIN_DSN" -qAt -c "INSERT INTO ops_headless(ticket,seat,state,last_activity) VALUES('META-GAUGES','巡检闸','$1',now()) ON CONFLICT (ticket) DO UPDATE SET seat=EXCLUDED.seat,state=EXCLUDED.state,last_activity=now(),updated_at=now()" >/dev/null 2>&1 || true; }

[ -n "$FIN_DSN" ] || { hb stalled; exit 0; }
today=$(date +%F)
while IFS=$'\t' read -r name cls qt qry th tpl; do
  [ -n "$name" ] || continue
  case "$qt" in
    sql) v=$(psql "$FIN_DSN" -qAt -c "$qry" 2>/dev/null | head -1);;
    cmd) v=$(eval "$qry" 2>/dev/null | head -1);;
    *) continue;;
  esac
  [ -n "$v" ] || continue
  hit=0
  if [ "$qt" = "sql" ] || [ "$qt" = "cmd" ]; then
    if [ "$th" -ge 0 ]; then [ "${v%.*}" -ge "$th" ] 2>/dev/null && hit=1
    else [ "${v%.*}" -lt 1 ] 2>/dev/null && hit=1; fi
  fi
  if [ "$hit" = 1 ]; then
    echo "$today" > "$ST/${name}.hit"
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
hb alive
exit 0
