#!/usr/bin/env bash
# verdict_deliver.sh —— 判词送达闸（T-0438）：需业主动作判词→fin_condition_gap
# 请求箱+feishu 推送（坑条目 real-money-ticket-machine-dispatch-waste 律④
# 「裁了≠送到了」机内化——DM 裁词落 ops_event 即止的送达断链根修）。
# 载位=dm_tick.sh 机械轨尾窗（裁决行落库同拍尾窗即转箱推送——宁搭车不新造
# 进程）；亦可单跑（env/覆写位同 dm_tick 族口径）。
# 判别（票面①词表首版钉死，禁模糊匹配）：ops_event 决策席 裁决/通过/驳回 行
#   note 命中钉死词表（人工径|业主在场|需业主|凭据缺口；O_VERDICT_KEYWORDS
#   覆写=桩测面）→判业主动作件。词→kind 映射：凭据缺口→credential，余→
#   executor（box kind 闭集语义；holding 留箱不用——首版无词映射）。
# 对行防连推：决策席同判词「裁决+通过/驳回」成对双行（同票同 note，T-0424
#   实据 90/101）——(ticket,note) 首现行（rn=1）唯一转写，对行零双推。
# 幂等（票面④）：同 ops_event 行只转一次——幂等键=ops_event 送达标记行
#   （event=送达，seat=META-DM，ticket=''零票态污染——队列推导/工作台视图
#   按 ticket 取 latest，带票号会藏收口行，故标记行落空票号、票号在 note），
#   note 前缀 ref=<event_id>|。箱写+推送双成才落标记；推送败=不落标记下拍
#   重试（箱 upsert 刷新语义重入无害）；箱写败推送已出=标记 gap=none 留痕
#   （不重推防刷屏——缺口由 rc≠0→dm_tick 升件自报显形）。
# 箱写径：fin_condition_gap INSERT…ON CONFLICT (venue,surface,kind) WHERE
#   state='pending' DO UPDATE（condreq.Request 同构——pending 撞键=刷新
#   requested_at+payload 非重行；结案后重转=新 pending 行，condreq 同律）。
#   venue=<票号> surface=verdict——per-票 per-kind 一 pending 行，同票连裁
#   刷同行不刷屏，异票异行不互撞。BOX_DSN 默认 META_PG_DSN（role o 箱全权
#   ——FIN_PG_DSN/roster_rw 对箱零授权实勘；迁移免依赖即插即用）。
# 推送径：feishu-notify 现成载（deploy_image.sh/deploy_oci.sh 同式直调），
#   文案票面③钉死「⚠ 需业主：<票号> <判词一行>」——败=best-effort 下拍重试。
# rc: 0=空集/全闭环；1=DSN 缺；2=扫描写径败；3=有未闭环件（下拍重试）。
set -u
ENVF="${VERDICT_DELIVER_ENV_FILE:-${DM_ENV_FILE:-/opt/o-research/env}}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$FIN_DSN" ] || exit 1
BOX_DSN=${O_VERDICT_BOX_DSN:-}
[ -n "$BOX_DSN" ] || BOX_DSN=$(grep '^META_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$BOX_DSN" ] || { echo "[verdict-deliver] 箱写 DSN 缺（META_PG_DSN/O_VERDICT_BOX_DSN）——未闭环" >&2; exit 1; }
NOTIFY_SH=${O_VERDICT_NOTIFY_SH:-/opt/meta/scripts/feishu-notify.sh}
NOTIFY_TIMEOUT=${O_VERDICT_NOTIFY_TIMEOUT:-15}
WINDOW=${O_VERDICT_WINDOW:-14 days}
# 词表首版钉死（票面①）——竖线分隔子串字面量；覆写位=桩测面，产径不动。
KEYWORDS=${O_VERDICT_KEYWORDS:-人工径|业主在场|需业主|凭据缺口}
HB_TICKET=META-DELIVER

hb() { # $1=state 心跳行（ops_headless META-DELIVER——meta 席位口径，查席纪律观测面）
  psql "$FIN_DSN" -qAt -c "INSERT INTO ops_headless(ticket,seat,state,last_activity)
    VALUES('$HB_TICKET','送达闸','$1',now())
    ON CONFLICT (ticket) DO UPDATE SET seat=EXCLUDED.seat,
      state=EXCLUDED.state,last_activity=now(),updated_at=now()" >/dev/null 2>&1 || true # tolerate: 心跳末环败静默——verdict_tick/dm_tick 同律（观测位非主务位）
}
hb alive

# 钉死词表→SQL LIKE 析取链（词面单引号防御性双写——钉死词无引号，覆写面兜）。
KW_SQL=""
IFS='|' read -ra KWS <<< "$KEYWORDS"
for kw in "${KWS[@]}"; do
  [ -n "$kw" ] || continue
  kw_sql=${kw//\'/\'\'}
  KW_SQL="${KW_SQL:+$KW_SQL OR }note LIKE '%${kw_sql}%'"
done
[ -n "$KW_SQL" ] || { echo "[verdict-deliver] 词表空——判别面不可用" >&2; hb stalled; exit 2; }

# 扫描：候选=决策席终态判词行∩词表命中∩窗内∩(ticket,note)首现∩零送达标记。
# 摘要列 SQL 侧裁（字符安全+空白折叠——bash 字节截断会劈 UTF-8）。
SCAN_SQL="
SELECT c.id,c.ticket,c.kind,c.exc160,c.exc300 FROM (
  SELECT id,ticket,
    CASE WHEN note LIKE '%凭据缺口%' THEN 'credential' ELSE 'executor' END AS kind,
    left(regexp_replace(note,'[[:space:]]+',' ','g'),160) AS exc160,
    left(regexp_replace(note,'[[:space:]]+',' ','g'),300) AS exc300,
    row_number() OVER (PARTITION BY ticket,note ORDER BY id) AS rn,
    ts
  FROM ops_event
  WHERE seat='决策席' AND event IN ('裁决','通过','驳回') AND ticket<>''
    AND ($KW_SQL)
) c
WHERE c.rn=1 AND c.ts>now()-interval '$WINDOW'
  AND NOT EXISTS (SELECT 1 FROM ops_event m
    WHERE m.event='送达' AND m.note LIKE 'ref='||c.id||'|%')
ORDER BY c.id"
CANDS=$(psql "$FIN_DSN" -qAt -F $'\t' -c "$SCAN_SQL" 2>/dev/null) || {
  echo "[verdict-deliver] 扫描写径败——候选集不可知" >&2; hb stalled; exit 2; }

UNDELIVERED=0
while IFS=$'\t' read -r rid tk kd exc160 exc300; do
  [ -n "$rid" ] || continue
  # ② 转写请求箱（condreq 同构 upsert——RETURNING id 供标记留痕）
  GAP=$(psql "$BOX_DSN" -v ON_ERROR_STOP=1 -qAt \
    -v rid="$rid" -v kd="$kd" -v tk="$tk" -v nv="$exc300" \
    <<'SQL' 2>/dev/null
WITH up AS (
  INSERT INTO fin_condition_gap(kind,venue,surface,payload)
  VALUES (:'kd', :'tk', 'verdict',
          jsonb_build_object('ref_event_id',:'rid'::bigint,
                             'ticket',:'tk','verdict',:'nv','source','T-0438'))
  ON CONFLICT (venue,surface,kind) WHERE state='pending'
  DO UPDATE SET requested_at=now(),payload=EXCLUDED.payload
  RETURNING id)
SELECT id FROM up;
SQL
  ) && [ -n "$GAP" ] || GAP=
  # ③ 推送（feishu-notify 现成载——deploy 脚本同式 best-effort）
  PUSHRC=0
  timeout "$NOTIFY_TIMEOUT" "$NOTIFY_SH" "⚠ 需业主：$tk $exc160" >/dev/null 2>&1 || PUSHRC=$?
  if [ -n "$GAP" ] && [ "$PUSHRC" = 0 ]; then
    # ④ 幂等标记（双成才落——note 前缀 ref=<id>| 即幂等键；ticket=''零票态污染）
    NOTE=$(printf 'ref=%s|gap=%s|push=ok|%s|%s 判词送达(T-0438)' "$rid" "$GAP" "$tk" "$kd")
    psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
      VALUES('META-DM','送达','','$(printf '%s' "$NOTE" | sed "s/'/''/g")')" >/dev/null 2>&1 \
      || true # tolerate: 标记末环败静默——下拍重转=箱刷新+重推一行，幂等键自愈不刷屏
    echo "[verdict-deliver] $tk 判词#$rid 已送达（gap=$GAP push=ok kind=$kd）" >&2
  elif [ -n "$GAP" ]; then
    UNDELIVERED=$((UNDELIVERED + 1)) # 箱已落推送败——下拍重试推送（箱 upsert 刷新无害）
    echo "[verdict-deliver] $tk 判词#$rid 推送败 rc=$PUSHRC——下拍重试" >&2
  elif [ "$PUSHRC" = 0 ]; then
    UNDELIVERED=$((UNDELIVERED + 1)) # 推送已出箱写败——不重推；标记留痕+rc 显形升件
    NOTE=$(printf 'ref=%s|gap=none|push=ok|%s|%s 箱写败留痕(T-0438)' "$rid" "$tk" "$kd")
    psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
      VALUES('META-DM','送达','','$(printf '%s' "$NOTE" | sed "s/'/''/g")')" >/dev/null 2>&1 \
      || true # tolerate: 标记末环败静默——升件面 rc≠0 已显形（T-0369 限度）
    echo "[verdict-deliver] $tk 判词#$rid 箱写败（推送已出）——gap=none 留痕" >&2
  else
    UNDELIVERED=$((UNDELIVERED + 1)) # 箱/推送双败——零标记下拍双重试
    echo "[verdict-deliver] $tk 判词#$rid 箱写+推送双败——下拍重试" >&2
  fi
done <<< "$CANDS"

if [ "$UNDELIVERED" -gt 0 ]; then
  hb stalled
  exit 3
fi
hb alive
exit 0
