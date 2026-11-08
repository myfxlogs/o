#!/usr/bin/env bash
# dm_autoapprove.sh —— 机批类册闸（T-0455 件①②④⑤落地 + T-0456 批权归径
# 钦定：seat='CEO' 已批行=CEO 无头本体）。dm_tick 物料化窗后同拍消费：
# 最新 ops_event='起草' 的候选票逐件判——
#   ①人工径票（manual_lane_check rc0）→跳过，起草滞留>2h→升件（人工径
#     死锁消化径：滞留即上行呈 CEO，不再等会话拍；阈 24h→2h=GW-001 段C2）；
#   ②病灶签名命中 meta.autoapprove.registry 册→范围节钦定类册径+
#     commit+ops_event seat='CEO' '已批' 行（机批=批权归 CEO 席域）；
#   ③册外签名→零机批；起草滞留>2h→升件呈 DM（类册外件永等 DM/COS 裁，
#     滞留升件即消化径——机批闸不吞件）。
# 幂等：最新行≠起草即跳（已批/驳回/已升件自然出集）；册缺/坏 json→零批
# fail-closed 升件一次。账面撤销径=ops_event '驳回' 行（最新行翻驳回→
# approved 集自然出列——T-0455 账面件钦定）。
# 负目录断言（GW-001 段C2 落码非散文）：册值径 ∩ autoapprove_negative_
# catalog.tsv（internal/exec、ledger 写径、venue 下单面）=册坏级 fail-closed
# 零批升件——族误映射把草案机批进钱径的防线在扫描闸。
# 批件保鲜（GW-001 段C3 水位律）：已批 7 天未动（latest=已批无尾随）→回
# 候选（草案行重落，候选集下拍重入：册内件机械重批，册外件等 DM 重裁）
# +升件呈 CEO（同票一次）。
# 用法：dm_autoapprove.sh   env: FIN_DSN + REPO(缺省 /opt/o) +
#   OPS_ESC_SH(缺省 $SCRIPT_DIR/ops_esc.sh) + O_AUTOAPPROVE_STALE_S
#   (起草滞留阈秒缺省 7200) + O_AUTOAPPROVE_NEGCAT(负目录表缺省同目录
#   autoapprove_negative_catalog.tsv) + O_AUTOAPPROVE_BATCH_STALE_S
#   (批件保鲜阈秒缺省 604800)。rc0=净；rc1=DB 读败/册坏/负目录冲突/
#   批中件败（拍壳升件）。
set -u
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
REPO=${REPO:-/opt/o}
OPS_ESC_SH=${OPS_ESC_SH:-"$SCRIPT_DIR/ops_esc.sh"}
KEY_GATE=${O_KEY_GATE_SH:-"$SCRIPT_DIR/ops_event_key_gate.sh"} # T-0499 键卫生闸（缺席 fail-open）
STALE_S=${O_AUTOAPPROVE_STALE_S:-7200}
NEGCAT=${O_AUTOAPPROVE_NEGCAT:-"$SCRIPT_DIR/autoapprove_negative_catalog.tsv"}
BATCH_STALE_S=${O_AUTOAPPROVE_BATCH_STALE_S:-604800}
FIN_DSN=${FIN_DSN:-${FIN_PG_DSN:-}}
[ -n "$FIN_DSN" ] || exit 0

esc() { "$OPS_ESC_SH" CTO "${3:-}" "" "" "$1" "$2" || true; } # tolerate: 升件尽力而为—升件败不阻机批主径
QC() { printf '%s' "${1:-}" | sed "s/'/''/g"; }

# 册读径（fail-closed：键缺/行空→零批静默退；键存值坏 json=psql 败→rc1 升件）。
REG_KEYS=$(psql "$FIN_DSN" -qAt -c \
  "SELECT COALESCE((SELECT string_agg(k,E'\n') FROM app_config c,
    jsonb_object_keys(c.value) k WHERE c.key='meta.autoapprove.registry'),'')" 2>/dev/null)
[ $? -eq 0 ] || { esc "机批册读败" "app_config meta.autoapprove.registry 读径败——零批 fail-closed"; exit 1; }

# 负目录全册扫描（GW-001 段C2 落码断言——机检非散文）：册任一键值径
# 命中负目录前缀=册坏级（族误映射把草案机批进钱径）→零批 fail-closed
# 升件一次 rc1。表缺/空=不可判 fail-closed 同律（断言面自证在册）。
neg_path_hit() { # $1=册值径 → 0=命中负目录
  local p=$1 n
  while IFS= read -r n; do
    case "$n" in ''|'#'*) continue ;; esac
    [ "$p" = "$n" ] && return 0
    case "$p" in "$n"/*) return 0 ;; esac
  done < "$NEGCAT"
  return 1
}
NEG_ROWS=$(psql "$FIN_DSN" -qAt -c \
  "SELECT k||E'\t'||x FROM app_config c,
    jsonb_object_keys(c.value) k,
    jsonb_array_elements_text(c.value->k) x
  WHERE c.key='meta.autoapprove.registry'" 2>/dev/null)
[ $? -eq 0 ] || { esc "机批册负目录扫描败" "册值径展开读径败——零批 fail-closed"; exit 1; }
if [ ! -s "$NEGCAT" ]; then
  esc "机批册负目录表缺" "$NEGCAT 缺/空——断言面不可判零批 fail-closed"; exit 1
fi
NEG_HIT=$(printf '%s\n' "$NEG_ROWS" | while IFS=$'\t' read -r _k p; do
  [ -n "$p" ] || continue
  neg_path_hit "$p" && { printf '%s\n' "$p"; break; }
done)
[ -z "$NEG_HIT" ] || {
  esc "机批册负目录冲突" "册值径 $NEG_HIT 命中负目录（internal/exec/ledger写径/venue下单面永不入册）——零批 fail-closed"
  exit 1
}

# 起草滞留消化径（人工径+册外签名共用）：>2h 未消化→升件一次（本件查重——
# 同票滞留升件已行即不再升；阈 24h→2h=GW-001 段C2 起草滞留阈）。
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
  [ -z "$dup" ] && esc "候选滞留:$tid" "$why——起草滞留>${STALE_S}s 零消化，上行呈 CEO（T-0455 滞留消化径）" "$tid"
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

RC=0
while IFS= read -r TID; do
  [ -n "$CANDS" ] || break
  [ -n "$TID" ] || continue
  # T-0499 键卫生闸：候选集 ticket 出自 ops_event GROUP BY——拼接伪键
  # （换行/空格/逗号串）loud 拒写整票跳过（闸缺席 fail-open）。
  [ ! -x "$KEY_GATE" ] || "$KEY_GATE" "$TID" "event=已批" || continue
  FILE=$(compgen -G "$REPO/docs/tasks/$TID-*.md" | head -1)
  [ -n "$FILE" ] && [ -r "$FILE" ] || { esc "候选滞留:$TID" "起草票件不可读——上行呈 CEO" "$TID"; continue; }

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

# ── GW-001 段C3 批件保鲜水位律：已批 7 天未动→回候选+升件 ──
# 判据=latest 相位已批（无 派单/领单/拉起/裁决/完成/驳回/挂起 尾随、同刻
# 并列反词同 dm_queue approved 面）∧已批龄>BATCH_STALE_S。动作=原子单句
# INSERT 草案行（回候选——latest 翻起草入候选集：册内件下拍机械重批=重核
# 签名仍命中，册外件等 DM 重裁）+逐票升件呈 CEO（同票一次去重）。批件超期
# =重核假设仍成立（collab 批件保鲜 7 天律机检化——防规漂）。
if [ "$BATCH_STALE_S" -gt 0 ] 2>/dev/null; then
  # T-0499 键卫生闸：stale 集 ticket 出自 ops_event GROUP BY——先取票列
  # 过闸（伪键 loud 拒写不入 IN 表），INSERT...SELECT 内嵌 stale 条件
  # 原子重估（相位竞态语义与原单句同——只多一道键过滤不拆事务）。
  STALE_CTE="WITH stale AS (
      SELECT l.ticket, l.mt FROM (
        SELECT ticket,max(ts) mt FROM ops_event WHERE ticket<>'' GROUP BY ticket) l
      WHERE EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
        AND e.ts=l.mt AND e.event='已批')
        AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ticket=l.ticket
          AND x.ts=l.mt AND x.event IN ('派单','领单','拉起','裁决','完成','驳回','挂起'))
        AND l.mt < now()-($BATCH_STALE_S * interval '1 second'))"
  STALE_TIDS=$(psql "$FIN_DSN" -qAt -c \
    "$STALE_CTE SELECT s.ticket FROM stale s ORDER BY s.ticket" 2>/dev/null) || STALE_RC=$?
  if [ "${STALE_RC:-0}" -eq 0 ]; then
    CLEAN_IN=""
    while IFS= read -r st; do
      [ -n "$st" ] || continue
      [ ! -x "$KEY_GATE" ] || "$KEY_GATE" "$st" "event=草案" || continue
      CLEAN_IN="${CLEAN_IN:+$CLEAN_IN,}'$(QC "$st")'"
    done <<< "$STALE_TIDS"
    if [ -n "$CLEAN_IN" ]; then
      STALES=$(psql "$FIN_DSN" -qAt -c \
        "$STALE_CTE INSERT INTO ops_event(seat,event,ticket,note)
    SELECT 'CTO','草案',s.ticket,
      '批件保鲜超期回候选——已批 未动超阈，重核假设后重批（GW-001 段C3 批件保鲜律；册内件下拍机械重批，册外件呈 DM 重裁）'
    FROM stale s WHERE s.ticket IN ($CLEAN_IN) RETURNING ticket" 2>/dev/null) || STALE_RC=$?
    fi
  fi
  if [ "${STALE_RC:-0}" -ne 0 ]; then
    esc "批件保鲜扫描败" "已批超期扫描/回候选写径败——保鲜律本拍滑行（下拍重试）"
  fi
  STALES=${STALES:-} # 净键集空/扫描败=未插行——下方升件循环空转合法径
  while IFS= read -r TID; do
    [ -n "$TID" ] || continue
    dup=$(psql "$FIN_DSN" -qAt -c \
      "SELECT 1 FROM ops_event WHERE ticket='$TID' AND event='升件'
       AND note LIKE '批件超期:$TID%' LIMIT 1" 2>/dev/null)
    [ -z "$dup" ] && esc "批件超期:$TID" "已批未动>7d 回候选——重核假设仍成立再重批（GW-001 段C3 批件保鲜）" "$TID"
  done <<< "$STALES"
fi
exit $RC
