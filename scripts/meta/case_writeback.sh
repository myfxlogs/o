#!/usr/bin/env bash
# case_writeback.sh —— GW-001 段E1 案卷回写（部署成功后案卷闭环）：
# 部署行（ops_event event='部署' note '... rc=0'，image/oci 两面 sha12
# 混合按首现 ts 序去重）逐 sha 扫 old..new commit 票号集
# （git log --format=%s 提 T-NNNN）→fin_ticket_draft.ticket→
# fin_econ_lesson 谱系两锚（裁定席归流段钦定写面：verdict_ref
# '修法票草稿draft#<id>'／'并入在飞票<T-NNNN>'）→
#   ① verdict_ref||=' deployed:<sha12>'（NOT LIKE 幂等）
#   ② ops_event 归流行（seat=MECH，note='deployed:<sha12>
#      fin_econ_lesson#<id> 案卷回写'——note 键查重幂等）
# 复发=新案+diag_attempts 累计：同谱系第二次部署（verdict_ref 已有
# deployed: 标记）→INSERT fin_econ_lesson kind='rootcause_diag'
# entity_ref=<base>#r<round> status='pending'——numbers 继承旧案+
# diag_attempts+1+recurrence_of/recurrence_deploy 锚（UNIQUE
# (kind,entity_ref) 故 #rN 后缀代次；ON CONFLICT 幂等）。
# 半成品窗律（段C/段D 并行施工——D 归流件未上线不依赖）：
#   fin_ticket_draft/fin_econ_lesson 表缺/查询败=静默零行——真实案
#   零命中不炸部署主径（fail-soft 全径 exit 0）；git 区间不可解/
#   prev 非 cur 祖先=跳该段不落 deploy-close 戳（下拍重试）。
# 幂等三层：deploy-close 面戳（event='归流' note 'deploy-close <sha12>
# ...'——ticket='' 管道级行不入票队）→verdict_ref NOT LIKE→归流行
# note 键。挂点=mech_tick 部署窗后每拍扫（O_CASE_WRITEBACK_SH 桩测
# 覆写位）——他段挂尾窗时本件独立块不覆盖。
set -u
REPO=${O_CASE_REPO:-/opt/o}
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
KEY_GATE=${O_KEY_GATE_SH:-"$SCRIPT_DIR/ops_event_key_gate.sh"} # T-0499 键卫生闸（缺席 fail-open）
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$FIN_DSN" ] || exit 0 # tolerate: 无 DSN=静默零行（部署主径不受影响）

q() { psql "$FIN_DSN" -qAt -F "$(printf '\037')" -c "$1" 2>/dev/null; } # tolerate: 查询败=rc≠0 由调用面判（fail-soft）

# ① 部署 sha 序：两面混合、首现 ts 序、全局去重（A,B,A 形=前向部署
# 序列下同 sha 重发仅相邻——全局去重保首现序，区间语义不受影响）
mapfile -t SHAS < <(psql "$FIN_DSN" -qAt -c \
  "SELECT COALESCE(substring(note from 'deploy-oci ([0-9a-f]+)'),
                   substring(note from 'deploy ([0-9a-f]+)'))
   FROM ops_event WHERE event='部署' AND note LIKE '% rc=0 %'
   ORDER BY ts,id" 2>/dev/null | awk 'NF && !seen[$0]++') # tolerate: 查询败=空集→零行退出
[ "${#SHAS[@]}" -ge 2 ] || exit 0 # 首发无 old..new 基线=零行

# 面戳集一次取（每拍全史扫的常态路径——closed 集合一次查非逐 sha 问）
declare -A CLOSED_MAP=()
while IFS= read -r c; do [ -n "$c" ] && CLOSED_MAP[$c]=1; done < <(
  psql "$FIN_DSN" -qAt -c \
    "SELECT substring(note from 'deploy-close ([0-9a-f]+)')
     FROM ops_event WHERE event='归流' AND note LIKE 'deploy-close %'" \
    2>/dev/null | awk 'NF' | sort -u) # tolerate: 查询败=空集→零闭集（全量重扫幂等无害）
for ((i=1; i<${#SHAS[@]}; i++)); do
  cur=${SHAS[$i]}; prev=${SHAS[$((i-1))]}
  # 面戳查重——已闭区间跳过（每拍幂等扫的常态路径）
  [ -n "${CLOSED_MAP[$cur]:-}" ] && continue
  # 区间可解闸：prev 须 cur 祖先（部署序列前向语义；不可解=跳段
  # 不落面戳——下拍重试，历史改写面不臆造票号集）
  git -C "$REPO" merge-base --is-ancestor "$prev" "$cur" 2>/dev/null || {
    echo "[case_writeback] $prev..$cur 不可解/非祖先——跳段待下拍" >&2; continue; }
  TICKETS=$(git -C "$REPO" log --format=%s "$prev..$cur" 2>/dev/null \
    | grep -oE 'T-[0-9]{4}' | sort -u) # tolerate: git 败=空集→matched=0 仍落面戳
  SEG_MATCHED=0
  while IFS= read -r tid; do
    [ -n "$tid" ] || continue
    # T-0499 键卫生闸：归流行 ticket 落库前拒伪键（git 提取面恒净=
    # 双保险；闸缺席 fail-open——跳过本案下拍重扫幂等无害）。
    [ ! -x "$KEY_GATE" ] || "$KEY_GATE" "$tid" "event=归流" || continue
    # draft→lesson 谱系：票→draft id 集→verdict_ref 两锚查 lesson
    # （draft#<id> 边界用正则防 draft#1 命中 draft#12；并入在飞票锚
    # 票级独立——并入既有在飞票的案无 draft 行也须命中，故无 draft
    # 不跳过）
    DIDS=$(q "SELECT id FROM fin_ticket_draft WHERE ticket='$tid'")
    CONDS="verdict_ref ~ '并入在飞票${tid}([^0-9]|$)'"
    while IFS= read -r did; do
      [ -n "$did" ] || continue
      CONDS="$CONDS OR verdict_ref ~ 'draft#${did}([^0-9]|$)'"
    done <<< "$DIDS"
    LROWS=$(q "SELECT id,entity_ref,status,verdict_ref FROM fin_econ_lesson
      WHERE kind='rootcause_diag' AND ($CONDS)")
    [ -n "$LROWS" ] || continue
    while IFS="$(printf '\037')" read -r lid eref lstatus vref; do
      [ -n "$lid" ] || continue
      DEPS=$(printf '%s\n' "$vref" | grep -o 'deployed:[0-9a-f]*' | wc -l)
      if [ "$DEPS" -eq 0 ]; then
        # ① 首部署闭环：verdict_ref 追加 deployed:<sha12>+归流行
        q "UPDATE fin_econ_lesson SET verdict_ref=verdict_ref||' deployed:${cur}'
           WHERE id=$lid AND verdict_ref NOT LIKE '%deployed:${cur}%'" || continue
        q "INSERT INTO ops_event(seat,event,ticket,note)
           SELECT 'MECH','归流','$tid','deployed:${cur} fin_econ_lesson#${lid} 案卷回写'
           WHERE NOT EXISTS (SELECT 1 FROM ops_event WHERE event='归流'
             AND note LIKE 'deployed:${cur} fin_econ_lesson#${lid}%')" || continue
        SEG_MATCHED=$((SEG_MATCHED+1))
      else
        # ② 复发=新案+diag_attempts 累计：entity_ref=<base>#r<round>
        # （去旧 #rN 尾再计代次——base∪#r% 行数+1）；numbers 继承+
        # diag_attempts+1+recurrence_of/deploy 锚；pending 重入诊断队
        base=${eref%%#r[0-9]*}
        RND=$(q "SELECT count(*)+1 FROM fin_econ_lesson
          WHERE kind='rootcause_diag'
            AND entity_ref ~ ('^' || '${base}' || '(#r[0-9]+)?$')")
        [ -n "$RND" ] || continue
        NEWID=$(q "INSERT INTO fin_econ_lesson(kind,entity_ref,numbers,status,verdict_ref)
          SELECT 'rootcause_diag','${base}#r${RND}',
            numbers || jsonb_build_object('diag_attempts',
              COALESCE((numbers->>'diag_attempts')::int,0)+1,
              'recurrence_of',$lid,'recurrence_deploy','$cur'),
            'pending',''
          FROM fin_econ_lesson WHERE id=$lid
          ON CONFLICT (kind,entity_ref) DO NOTHING RETURNING id")
        [ -n "$NEWID" ] || continue # 撞代次=已落幂等跳过
        q "INSERT INTO ops_event(seat,event,ticket,note)
           SELECT 'MECH','归流','$tid','复发 fin_econ_lesson#${lid}→#${NEWID} deployed:${cur} 案卷回写'
           WHERE NOT EXISTS (SELECT 1 FROM ops_event WHERE event='归流'
             AND note LIKE '复发 fin_econ_lesson#${lid}→#${NEWID}%')" || continue
        SEG_MATCHED=$((SEG_MATCHED+1))
      fi
    done <<< "$LROWS"
  done <<< "$TICKETS"
  # 面戳落账——区间扫毕（matched=0 合法：无谱系案=静默零行不误伤）
  q "INSERT INTO ops_event(seat,event,ticket,note)
     VALUES('MECH','归流','','deploy-close ${cur} 案卷回写扫毕 matched=${SEG_MATCHED} range=${prev}..${cur}')" \
    || echo "[case_writeback] deploy-close ${cur} 面戳落账败——下拍重扫（幂等无害）" >&2
  [ -n "$SEG_MATCHED" ] && [ "$SEG_MATCHED" -gt 0 ] && \
    echo "[case_writeback] ${prev}..${cur} 案卷回写 ${SEG_MATCHED} 案" >&2
done
exit 0
