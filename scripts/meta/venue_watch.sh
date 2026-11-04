#!/usr/bin/env bash
# venue_watch.sh —— T-0348 venue 新品哨兵拍（零 LLM 纯判据——照 gauges_tick 对形；
# 在册 venue 公告面+产品目录 diff → ops_event '新品' 行（seat=venue哨兵）→ 研究员
# 触发信号。哨兵职责止于事件+心跳：不碰 fin_opp_hypothesis/fin_surface_gap/模板面
# （研究员窄门——票面红线）。
#
# 端点实探记录（2026-11-04 www.okx.com 公开面，协议面取证纪律——真实响应定字段）：
#   公告 GET /api/v5/support/announcements → code=0，data[0].details 恒 20 条，
#     项字段=annType/title/url/pTime/businessPTime——无 id 列且 url 带区域前缀
#     （同公告 en-us/en-sg 异 URL）→ 幂等键取 /help/ 后 slug；before/after 翻页
#     参数对公开面无效（实探回包=当前窗原样）→ 可视角=最新 20 条/面；
#     announcement-types 端点回显类型（4）少于默认流实见类型（8+，且带 -us-aus
#     区域后缀漂移）→ 类型枚举面不可靠，词面过滤为主、新上市类 annType 为辅。
#   目录 GET /api/v5/public/instruments：SPOT(1210 条~1.3MB)/MARGIN/SWAP/FUTURES
#     零参数全枚举在；instType=OPTION 无 uly → 50015（OPTION 本体不可盲枚举），
#     uly=BTC-USD → 1248 条（带到期日逐日上新——逐 instId 事件=常态噪声）；
#     GET /api/v5/public/underlying?instType=OPTION → data=[[SOL-USD,BTC-USD,
#     ETH-USD,XAU-USD,HYPE-USD]] 期权 uly 族清单在——新 uly 族=新期权面检测点。
#   简易期权（首天然锚）：目录零承载 instType（枚举+50015 实证）+ 公告窗 3×20
#     面零命中（option/期权/Earn 词面全扫）→ 「API 不可见面」取证结论在案
#     （票面锚②如实记分支）；哨兵词面在，未来公告落入 20 条窗即检出。
#
# 检出判据（类级闸防刷屏——票面钦定「新 instType/新 surface 类=新品候选」）：
#   公告面：annType=新上市类 ∨ 标题词面（期权/option/earn/new product 族，
#     脚本头常量精确词面）；slug 观测即记册（滑窗边界防倒灌重发），7d 新鲜度闸内
#     且未记册 → 事件 note=venue:ann:标题:url（落账成才记册——败=下拍重试）。
#   目录面：SPOT/MARGIN=quoteCcy 新面；SWAP/FUTURES=ctType 新面；
#     OPTION=underlying 新 uly 族——皆「该类首个新 instId」代位出处；新 instId
#     同类只入 seen 册零事件。目录面首拍=基线种册零事件（防装机即 20+ 事件刷屏；
#     公告面无首拍豁免——首天然锚检出窗，票面钦定）。
# 失败显形（照 gauges_tick 取数败同构）：curl 败/回包非 code=0/解析空 →
#   $ST/<源>.fetch_error 留痕 + hb state='fail:<源,…>'（ops_headless 无 note 列
#   ——state 列=心跳行唯一自由槽；不静默绿）。全绿=hb alive。
# seen-set（幂等——同品重放零重复事件）：$ST 文件径（照 gauges /var/tmp 暂存径；
#   PG 小表弃——纯脚本拍零迁移面）。事件落账成功才记 seen（落账败=下拍自愈重试）。
#   测试缝：VENUEWATCH_SYNTH_ANN=伪公告 JSON 路径 → 公告默认源吃注入件（合成
#   干跑径——验收锚①；生产 unset 即零歧义）。
#
# venue→探测函数 map（结构扩展位）：VENUES 列表每项对应 probe_<venue>()——
#   Binance/MT5 同类端点后续挂函数+列名即扩（本票只实装 okx，票面钦定不必扩）。
set -u
ENVF="${VENUEWATCH_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
ST="${VENUEWATCH_ST:-/var/tmp/o-venuewatch}"; mkdir -p "$ST"

# —— 新品类词面（脚本头常量钉死——精确词面勿正则地狱；宽匹配=事件刷屏）——
ANN_NEWLIST_TYPE='announcements-new-listings'  # 新上市类 annType（实探在案）
ANN_LOOKBACK_HOURS=168                          # 公告新鲜度闸=7d（ms 面换算见 ann_face）
KW_RE='期权|new products?|simple options?'       # 子串面（CJK/短语——词界无意义）
KW_WORD_RE='options?|earn'                      # 词界面（短词必须词界——裸 earn 撞 yearly/learning）

hb() { [ -n "$FIN_DSN" ] && psql "$FIN_DSN" -qAt -c "INSERT INTO ops_headless(ticket,seat,state,last_activity) VALUES('META-VENUEWATCH','venue哨兵','$1',now()) ON CONFLICT (ticket) DO UPDATE SET seat=EXCLUDED.seat,state=EXCLUDED.state,last_activity=now(),updated_at=now()" >/dev/null 2>&1 || true; }

emit() { # $1=note（venue:产品:出处）——ops_event 新品行；note 单引号双写转义（gauges 直拼同构——psql -c 对 :'var' 形不插值，实踩在案）
  local esc=${1//\'/\'\'}
  psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note) VALUES('venue哨兵','新品','META-VENUEWATCH','$esc')" >/dev/null 2>&1
}

fetch() { # $1=url $2=out $3=errf——限流/瞬断鲁棒：2 试+短退避；败=留痕返 1
  local url="$1" out="$2" errf="$3" i
  for i in 1 2; do
    if curl -sS --max-time 60 "$url" -o "$out" 2>"$errf.curl" && [ -s "$out" ]; then rm -f "$errf.curl"; return 0; fi
    sleep 10
  done
  mv "$errf.curl" "$errf" 2>/dev/null || true
  return 1
}

ok_payload() { # $1=json $2=jq 判式——code=0 且判式真；败写 fetch_error
  local out="$1" cond="$2" errf="$3"
  if jq -e "$cond" "$out" >/dev/null 2>&1; then rm -f "$errf"; return 0; fi
  echo "bad payload $(date -Is): $(head -c 200 "$out" 2>/dev/null)" > "$errf"
  return 1
}

ann_face() { # $1=venue $2=url $3=tag $4=默认源注入件路径（空=真取）
  local venue="$1" url="$2" tag="$3" synth="${4:-}"
  local out="$ST/$tag.json" errf="$ST/$tag.fetch_error" seen="$ST/$venue.ann.seen" tsv="$ST/$tag.tsv"
  if [ -n "$synth" ]; then cp "$synth" "$out" 2>"$errf" || { FAILS="$FAILS $tag"; return 0; }
  elif ! fetch "$url" "$out" "$errf"; then FAILS="$FAILS $tag"; return 0; fi
  ok_payload "$out" '.code=="0" and (.data[0].details|length>0)' "$errf" || { FAILS="$FAILS $tag"; return 0; }
  jq -r '.data[0].details[] | [.annType, (.url | if contains("/help/") then sub("^.*/help/";"") else . end), .title, .pTime, .url] | @tsv' "$out" > "$tsv" 2>"$errf" \
    || { FAILS="$FAILS $tag"; return 0; }
  rm -f "$errf"
  local nowms cutms atype slug title ptime aurl
  nowms=$(date +%s%3N); cutms=$((nowms - ANN_LOOKBACK_HOURS * 3600000))
  while IFS=$'\t' read -r atype slug title ptime aurl; do
    [ -n "$slug" ] || continue
    case "$atype" in "$ANN_NEWLIST_TYPE") ;; *)
      { printf '%s' "$title" | grep -qiE "$KW_RE" || printf '%s' "$title" | grep -qiwE "$KW_WORD_RE"; } || continue;;
    esac
    grep -qxF "$slug" "$seen" 2>/dev/null && continue
    if [ "$ptime" -ge "$cutms" ] 2>/dev/null; then
      emit "$venue:ann:$title:$aurl" && echo "$slug" >> "$seen"   # 落账成才记册（败=下拍重试）
    else
      echo "$slug" >> "$seen"   # 新鲜度闸外陈件：记册零事件——滑窗边界防倒灌重发
    fi
  done < "$tsv"
}

inst_face() { # $1=venue $2=instType——全枚举目录 diff（类级闸；首拍种册零事件）
  local venue="$1" it="$2"
  local out="$ST/$venue.inst.$it.json" errf="$ST/$venue.inst.$it.fetch_error"
  local cls="$ST/$venue.classes.$it.seen" ids="$ST/$venue.insts.$it.seen" tsv="$ST/$venue.inst.$it.tsv"
  fetch "https://www.okx.com/api/v5/public/instruments?instType=$it" "$out" "$errf" || { FAILS="$FAILS $venue.inst.$it"; return 0; }
  ok_payload "$out" '.code=="0" and (.data|length>0)' "$errf" || { FAILS="$FAILS $venue.inst.$it"; return 0; }
  case "$it" in
    SPOT|MARGIN) jq -r '.data[] | [.instId, .quoteCcy] | @tsv' "$out";;
    *)           jq -r '.data[] | [.instId, .ctType] | @tsv' "$out";;
  esac > "$tsv" 2>"$errf" || { FAILS="$FAILS $venue.inst.$it"; return 0; }
  rm -f "$errf"
  local boot=0; [ -f "$cls" ] && boot=1   # 类册在=非首拍
  local id sig newsig
  # ① 类级检出：现册类签 vs 类册——新类签=新品候选（代位出处=该类首个新 instId）
  cut -f2 "$tsv" | sort -u | while read -r sig; do
    [ -n "$sig" ] || continue
    grep -qxF "$sig" "$cls" 2>/dev/null && continue
    if [ "$boot" = 1 ]; then
      id=$(awk -F'\t' -v s="$sig" '$2==s {print $1; exit}' "$tsv")
      case "$it" in SPOT|MARGIN) sigp="quote=$sig";; *) sigp="ctType=$sig";; esac
      emit "$venue:$it/$sigp:instId=$id:catalog" && echo "$sig" >> "$cls"
    else
      echo "$sig" >> "$cls"   # 首拍种册零事件
    fi
  done
  # ② instId 册例行级合并（同类新 instId 零事件——每日到期期权/新币上市=常态噪声）
  cut -f1 "$tsv" | sort -u >> "$ids" 2>/dev/null
  sort -u -o "$ids" "$ids"
}

uly_face() { # $1=venue——OPTION uly 族清单 diff（underlying 端点；新族=新期权面）
  local venue="$1"
  local out="$ST/$venue.uly.json" errf="$ST/$venue.uly.fetch_error" seen="$ST/$venue.uly.seen"
  fetch "https://www.okx.com/api/v5/public/underlying?instType=OPTION" "$out" "$errf" || { FAILS="$FAILS $venue.uly"; return 0; }
  ok_payload "$out" '.code=="0" and (.data[0]|length>0)' "$errf" || { FAILS="$FAILS $venue.uly"; return 0; }
  local f boot=0; [ -f "$seen" ] && boot=1
  jq -r '.data[0][]' "$out" | while read -r f; do
    [ -n "$f" ] || continue
    grep -qxF "$f" "$seen" 2>/dev/null && continue
    if [ "$boot" = 1 ]; then
      emit "$venue:OPTION/uly=$f:underlying" && echo "$f" >> "$seen"
    else
      echo "$f" >> "$seen"   # 首拍种册零事件
    fi
  done
}

probe_okx() {
  local v=okx
  ann_face "$v" "https://www.okx.com/api/v5/support/announcements" "$v.ann" "${VENUEWATCH_SYNTH_ANN:-}"
  ann_face "$v" "https://www.okx.com/api/v5/support/announcements?annType=$ANN_NEWLIST_TYPE" "$v.anl" ""
  local it
  for it in SPOT MARGIN SWAP FUTURES; do inst_face "$v" "$it"; done
  uly_face "$v"
}

# —— venue→探测函数 map（扩展位：挂函数+列名即扩）——
VENUES="${VENUEWATCH_VENUES:-okx}"
FAILS=""
[ -n "$FIN_DSN" ] || { hb stalled; exit 0; }
for v in $VENUES; do
  case "$v" in
    okx) probe_okx;;
    *) : ;;   # 未知 venue 静默跳过（列名先行、函数后落——扩装期不炸拍）
  esac
done
if [ -n "$FAILS" ]; then hb "fail:${FAILS# }"; else hb alive; fi
exit 0
