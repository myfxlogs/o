#!/usr/bin/env bash
# dm_autoapprove.sh —— 机批类册闸（T-0455 件①②④⑤落地 + T-0456 批权归径
# 钦定：seat='CEO' 已批行=CEO 无头本体）。dm_tick 物料化窗后同拍消费：
# 最新 ops_event='起草' 的候选票逐件判——
#   ①人工径票（manual_lane_check rc0）→跳过，起草滞留>24h→升件（人工径
#     死锁消化径：滞留即上行呈 DM，不再等会话拍）；
#   ②病灶签名命中 meta.autoapprove.registry 册→范围节钦定类册径+
#     commit+ops_event seat='CEO' '已批' 行（机批=批权归 CEO 席域）；
#   ③册外签名→零机批；起草滞留>24h→升件呈 DM（类册外件永等 DM/COS 裁，
#     滞留升件即消化径——机批闸不吞件）。
# 幂等：最新行≠起草即跳（已批/驳回/已升件自然出集）；册缺/坏 json→零批
# fail-closed 升件一次。账面撤销径=ops_event '驳回' 行（最新行翻驳回→
# approved 集自然出列——T-0455 账面件钦定）。
# 用法：dm_autoapprove.sh   env: FIN_DSN + REPO(缺省 /opt/o) +
#   OPS_ESC_SH(缺省 $SCRIPT_DIR/ops_esc.sh) + O_AUTOAPPROVE_STALE_S
#   (滞留阈秒缺省 86400)。rc0=净；rc1=DB 读败/册坏/批中件败（拍壳升件）。
set -u
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
REPO=${REPO:-/opt/o}
OPS_ESC_SH=${OPS_ESC_SH:-"$SCRIPT_DIR/ops_esc.sh"}
STALE_S=${O_AUTOAPPROVE_STALE_S:-86400}
FIN_DSN=${FIN_DSN:-${FIN_PG_DSN:-}}
[ -n "$FIN_DSN" ] || exit 0

esc() { "$OPS_ESC_SH" META-DM "${3:-}" "" "" "$1" "$2" || true; }

# 册读径（fail-closed：键缺/行空→零批静默退；键存值坏 json=psql 败→rc1 升件）。
REG_KEYS=$(psql "$FIN_DSN" -qAt -c \
  "SELECT COALESCE((SELECT string_agg(k,E'\n') FROM app_config c,
    jsonb_object_keys(c.value) k WHERE c.key='meta.autoapprove.registry'),'')" 2>/dev/null)
[ $? -eq 0 ] || { esc "机批册读败" "app_config meta.autoapprove.registry 读径败——零批 fail-closed"; exit 1; }

# 起草滞留消化径（人工径+册外签名共用）：>24h 未消化→升件一次（本件查重——
# 同票滞留升件已行即不再升）。
escalate_stale() {
  local tid=$1 why=$2
  local age
  age=$(psql "$FIN_DSN" -qAt -c \
    "SELECT extract(epoch from now()-max(ts))::int FROM ops_event
     WHERE ticket='$tid' AND event='起草'" 2>/dev/null)
  [ -n "$age" ] && [ "$age" -ge "$STALE_S" ] || return 0
  local dup
  dup=$(psql "$FIN_DSN" -qAt -c \
    "SELECT 1 FROM ops_event WHERE ticket='$tid' AND event='升件'
     AND note LIKE '候选滞留:$tid%' LIMIT 1" 2>/dev/null)
  [ -z "$dup" ] && esc "候选滞留:$tid" "$why——起草滞留>${STALE_S}s 零消化，上行呈 DM（T-0455 滞留消化径）" "$tid"
  return 0
}

# 册内匹配（精确键 或 尾*前缀键）→回钦定径清单（空格分隔）。
# 键字面径=bash 侧单引号转义拼 SQL（psql :'var' 插值在本面不可用实证——
# 键自 app_config 册出仍 ' 转义防御）。
reg_match() {
  local sig=$1 key kq
  [ -n "$REG_KEYS" ] || return 1
  while IFS= read -r key; do
    [ -n "$key" ] || continue
    case "$key" in
      *\*) [ "${sig#"${key%\*}"}" != "$sig" ] || continue ;;
      *)   [ "$sig" = "$key" ] || continue ;;
    esac
    kq=${key//\'/\'\'}
    psql "$FIN_DSN" -qAt -c \
      "SELECT string_agg(x,' ') FROM app_config c,
       jsonb_array_elements_text(c.value->'$kq') x
       WHERE c.key='meta.autoapprove.registry'" 2>/dev/null
    return
  done <<< "$REG_KEYS"
  return 1
}

# 候选集=最新行='起草'。
CANDS=$(psql "$FIN_DSN" -qAt -c "SELECT l.ticket FROM (
    SELECT ticket,max(ts) mt FROM ops_event WHERE ticket<>'' GROUP BY ticket
  ) l WHERE EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
    AND e.ts=l.mt AND e.event='起草') ORDER BY l.ticket" 2>/dev/null) || exit 1
[ -n "$CANDS" ] || exit 0

RC=0
while IFS= read -r TID; do
  [ -n "$TID" ] || continue
  FILE=$(compgen -G "$REPO/docs/tasks/$TID-*.md" | head -1)
  [ -n "$FILE" ] && [ -r "$FILE" ] || { esc "候选滞留:$TID" "起草票件不可读——上行呈 DM" "$TID"; continue; }

  bash "$SCRIPT_DIR/manual_lane_check.sh" "$TID" "$REPO/docs/tasks" && {
    escalate_stale "$TID" "人工径票（manual_lane_check 字面命中）"; continue; }

  # 签名提取禁单反引号字面（$( ) 内 backquote 重解析坑——\x60 等价径）。
  SIG=$(sed -n 's/.*病灶签名：\x60\([^\x60]*\)\x60.*/\1/p' "$FILE" | head -1)
  [ -n "$SIG" ] || { escalate_stale "$TID" "票面无病灶签名行——册判不可"; continue; }

  PATHS=$(reg_match "$SIG")
  if [ -n "$PATHS" ]; then
    # 范围节钦定——骨架行换类册钦定径（DM 裁时可改判语义保留）。
    sed -i "s|^1\. 明确文件路径或目录清单.*|1. 机批类册钦定文件径：${PATHS}——签名 ${SIG}（T-0455 册内候选，DM 裁时可改判）；|" "$FILE"
    (cd "$REPO" && git add "$FILE" && git commit -q -m \
      "[角色:决策][$TID] 机批类册钦定范围——$SIG（T-0455 册内候选）" >/dev/null 2>&1)
    N=$(echo "$PATHS" | wc -w)
    psql "$FIN_DSN" -qAt -c \
      "INSERT INTO ops_event(seat,event,ticket,note) VALUES('CEO','已批','$TID',
       '机批:$SIG——类册钦定 $N 径（T-0455/0456 批权归径；驳回行即撤销）')" >/dev/null 2>&1 \
      || { esc "机批闸账败" "rc=? $TID ops_event 已批行落败" "$TID"; RC=1; }
    echo "[autoapprove] $TID 机批已批——$SIG → $PATHS"
  else
    escalate_stale "$TID" "册外签名 $SIG——零机批滞留"
  fi
done <<< "$CANDS"
exit $RC
