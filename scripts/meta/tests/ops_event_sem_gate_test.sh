#!/usr/bin/env bash
# ops_event_sem_gate_test.sh —— T-0506 (seat,event) 语义闸桩测：
#   A 矩阵文件自检（无 DB）：fin_0154 迁移件 DDL 三件在案+票面钉死组合
#     断言+消费面活对拍（dm_queue 派单 seat 集/ceo_queue 呈批席集从
#     消费面 SQL 解析与矩阵逐字比对——消费面改判据→测试红逼矩阵对账，
#     单源不裂）；
#   B 在产组合枚举对拍（FIN_PG_DSN 只读 SELECT DISTINCT——读 /opt/
#     o-research/env 同 key_gate 径约；DSN 缺席/库不可达→skip loud
#     不猜不占）：全量 DISTINCT(seat,event) 逐组合分类——矩阵外事件
#     =UNGATED 直通；矩阵内命中=ALLOW/COND(条件放行)；无命中=DENY
#     必须 ⊆ EXPECTED_DENY 钉死清单（死席别名/病灶写/越权写 18 组合
#     在案）——清单外 DENY=新存量组合入闸未对账=红；
#   C 真库 trigger 实测（testdb_lease 池制——池空 skip 不猜不占）：
#     最小 ops_event+fin_0121 hash 链+fin_0154 实跑共存——裸 psql
#     INSERT（不经任何 shell helper=判据③「裸 SQL 不可绕」直接钉死）：
#     COS+派单拒（ERROR 含 seat/event/ticket 三元+loud 前缀）且行零落；
#     决策席+派单/CTO+派单/裁定席·CEO·外审席 裁决驳回/施工席族领单/
#     COS 族 cos-relay 派单 放行；说明类与矩阵外相位词任意 seat 放行；
#     越权组合逐一拒；合法行 hash/prev_hash 链戳照常=两 trigger 共存。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
MIG="$ROOT/migrations/fin_0154_ops_event_sem_gate.sql"
REAL_PSQL=$(command -v psql)
pass=0; fail=0
ok()  { echo "  ✅ $1"; pass=$((pass+1)); }
bad() { echo "  ❌ $1"; fail=$((fail+1)); }

# —— 矩阵行解析：fin_0154 INSERT 块抽 ('event','seat','note_like',...) ——
#    产 TSV event<TAB>seat<TAB>note_like；单源=迁移文件本身（测试不抄词）。
MAT_TSV=$(mktemp)
trap 'rm -f -- "$MAT_TSV"' EXIT
grep -oE "\('[^']*','[^']*','[^']*'," "$MIG" \
  | sed -E "s/^\('([^']*)','([^']*)','([^']*)',/\1\t\2\t\3/" > "$MAT_TSV"
mat_has() { # event seat [note_like] —— note_like 缺省='' 精确命中
  awk -F'\t' -v e="$1" -v s="$2" -v n="${3:-}" \
    '$1==e && $2==s && $3==n {f=1} END{exit !f}' "$MAT_TSV"
}
mat_event() { awk -F'\t' -v e="$1" '$1==e{f=1} END{exit !f}' "$MAT_TSV"; }
mat_seats_of() { awk -F'\t' -v e="$1" '$1==e{print $2}' "$MAT_TSV" | sort; }

echo "══ A 矩阵文件自检 ══"
[ -f "$MIG" ] && ok "A 迁移件存在" || { bad "A 迁移件缺"; echo "sem_gate: PASS=$pass FAIL=$fail"; exit 1; }
grep -q 'CREATE TABLE IF NOT EXISTS ops_event_sem_matrix' "$MIG" \
  && ok "A 矩阵表 DDL 在案" || bad "A 矩阵表 DDL 缺"
grep -q 'CREATE OR REPLACE FUNCTION ops_event_sem_gate' "$MIG" \
  && ok "A 闸函数 DDL 在案" || bad "A 闸函数 DDL 缺"
grep -q 'CREATE TRIGGER ops_event_sem_gate_trg BEFORE INSERT ON ops_event' "$MIG" \
  && ok "A BEFORE INSERT 触发器在案" || bad "A 触发器 DDL 缺"
grep -q 'RAISE EXCEPTION' "$MIG" && grep -q 'NEW.ticket' "$MIG" \
  && ok "A loud 拒写含 seat/event/ticket 三元" || bad "A loud 三元缺"

# A2 票面钉死组合
mat_has '派单' '决策席'    && ok "A2 派单×决策席 在矩阵" || bad "A2 派单×决策席 缺"
mat_has '派单' 'CTO'       && ok "A2 派单×CTO 在矩阵" || bad "A2 派单×CTO 缺"
mat_has '派单' 'COS' '%cos-relay%'     && ok "A2 派单×COS cos-relay 例外在案" || bad "A2 COS cos-relay 缺"
mat_has '派单' 'COS上行' '%cos-relay%' && ok "A2 派单×COS上行 cos-relay 在案" || bad "A2 COS上行 cos-relay 缺"
mat_has '派单' '席O-COS' '%cos-relay%' && ok "A2 派单×席O-COS cos-relay 在案" || bad "A2 席O-COS cos-relay 缺"
mat_has '派单' 'COS'       && bad "A2 裸派单×COS 无条件行=病灶复活" \
                        || ok "A2 裸派单×COS 无条件放行行（必拒语义在案）"
for s in 决策席 CTO 裁定席 CEO 外审席; do
  mat_has '裁决' "$s" && : || bad "A2 裁决×$s 缺（票面例）"
done
mat_has '领单' '施工席'      && ok "A2 领单×施工席 在矩阵" || bad "A2 领单×施工席 缺"
mat_has '领单' '无头施工席'  && ok "A2 领单×无头施工席 在矩阵" || bad "A2 领单×无头施工席 缺"
mat_has '已批' 'CEO'         && ok "A2 已批×CEO 在矩阵（机批钉死 seat）" || bad "A2 已批×CEO 缺"
mat_has '已批' 'COS'         && ok "A2 已批×COS 在矩阵（职卡代理批注）" || bad "A2 已批×COS 缺"
mat_has '起草' '决策席'      && ok "A2 起草×决策席 在矩阵" || bad "A2 起草×决策席 缺"
mat_has '拉起' 'CTO'         && ok "A2 拉起×CTO 在矩阵" || bad "A2 拉起×CTO 缺"

echo "══ A3 消费面活对拍（单源不裂）══"
# dm_queue dispatched 面钦定集 {决策席,CTO} ⊆ 矩阵派单行集
DQ="$ROOT/scripts/meta/dm_queue.sh"
DQSET=$(grep -oE "latest_seat IN \('[^)]*'\)|latest_seat +IN +\('[^)]*'\)" "$DQ" \
        | head -1 | grep -oE "'[^']*'" | tr -d "'" | sort | tr '\n' ' ')
[ -n "$DQSET" ] && ok "A3 dm_queue 派单 seat 判据解析=$DQSET" || bad "A3 dm_queue 判据解析败"
for s in $DQSET; do
  mat_seats_of '派单' | grep -qx "$s" \
    && ok "A3 消费面 seat '$s' ⊆ 矩阵派单集" || bad "A3 消费面 seat '$s' 不在矩阵=消费面说谎复活"
done
# ceo_queue 呈批席集 == 矩阵呈批行集逐字（双向——消费面增删席→红逼对账）
# 提取=e.seat IN (...) 判据块逐字（避开注释/事件名词）
CQ="$ROOT/scripts/meta/ceo_queue.sh"
CQSET=$(sed -n "/e\.seat IN/,/)$/p" "$CQ" | grep -oE "'[^']+'" | tr -d "'" | sort -u)
MAT_CQ=$(mat_seats_of '呈批')
if [ "$CQSET" = "$MAT_CQ" ]; then
  ok "A3 呈批席集=ceo_queue 消费面逐字（$(echo "$CQSET" | wc -w) 席）"
else
  bad "A3 呈批席集与 ceo_queue 消费面不齐——消费面:$(echo $CQSET|tr '\n' ' ') 矩阵:$(echo $MAT_CQ|tr '\n' ' ')"
fi

echo "══ B 在产组合枚举对拍（读径只读；缺席 skip）══"
BDSN="${O_TEST_FIN_PG_RO_DSN:-}"
ENVF=/opt/o-research/env
[ -z "$BDSN" ] && [ -f "$ENVF" ] \
  && BDSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
if [ -z "$BDSN" ]; then
  echo "  ⚠️ [ops_event_sem_gate_test] FIN_PG_DSN 缺席——B 段在产枚举对拍 skip（不猜不占）"
else
  COMBO_TSV=$(mktemp); trap 'rm -f -- "$MAT_TSV" "$COMBO_TSV"' EXIT
  if ! "$REAL_PSQL" "$BDSN" -qAt -c \
      "SELECT seat||'|'||event FROM ops_event GROUP BY 1" > "$COMBO_TSV" 2>/dev/null; then
    echo "  ⚠️ [ops_event_sem_gate_test] 在产库不可达——B 段 skip"
    COMBO_TSV=
  fi
  if [ -n "$COMBO_TSV" ] && [ -s "$COMBO_TSV" ]; then
    # EXPECTED_DENY=死席别名/病灶写/越权写钉死清单（矩阵判 DENY 的在产组合
    #   全集——票面判据④全覆盖：清单外 DENY=新组合入闸未对账=红）
    EXPECTED_DENY=$(cat <<'EOF'
裁决|Dev
裁决|DM
裁决|席MANAGE
裁决|席O-DM
裁决|施工席
裁决|无头施工席
呈批|席O-COS
领单|CTO
领单|DM
领单|Dev
领单|决策席
派单|DM
派单|O-Dev交互席
派单|席O-DM
派单|席O-Dev
派单|无头施工席
驳回|DM
驳回|席O-DM
EOF
)
    DENY_NOW=$(mktemp); trap 'rm -f -- "$MAT_TSV" "$COMBO_TSV" "$DENY_NOW"' EXIT
    UNGATED=0; ALLOW=0; COND=0
    while IFS='|' read -r seat event; do
      [ -n "$seat$event" ] || continue
      if ! mat_event "$event"; then UNGATED=$((UNGATED+1)); continue; fi
      if mat_has "$event" "$seat" ''; then ALLOW=$((ALLOW+1)); continue; fi
      # 条件放行行（note_like 非空）：组合级对拍=条件覆盖
      if awk -F'\t' -v e="$event" -v s="$seat" \
           '$1==e && $2==s && $3!="" {f=1} END{exit !f}' "$MAT_TSV"; then
        COND=$((COND+1)); continue
      fi
      echo "$event|$seat" >> "$DENY_NOW"
    done < "$COMBO_TSV"
    UNLISTED=$(comm -23 <(sort "$DENY_NOW") <(printf '%s\n' "$EXPECTED_DENY" | sort))
    STALE=$(comm -13 <(sort "$DENY_NOW") <(printf '%s\n' "$EXPECTED_DENY" | sort))
    [ -z "$UNLISTED" ] && ok "B 在产 DENY 全在钉死清单（$(wc -l < "$DENY_NOW") 组合）" \
                     || bad "B 清单外 DENY 组合（新存量入闸未对账）: $(echo "$UNLISTED" | tr '\n' ' ')"
    [ -z "$STALE" ] || echo "  ℹ️ B EXPECTED_DENY 陈旧项（在产已无此组合，清单可收）: $(echo "$STALE" | tr '\n' ' ')"
    ok "B 枚举分类完备：ALLOW=$ALLOW COND=$COND DENY=$(wc -l < "$DENY_NOW") UNGATED=$UNGATED（共 $(wc -l < "$COMBO_TSV") 组合）"
  fi
fi

echo "══ C 真库 trigger 实测（池空 skip）══"
# shellcheck source=/dev/null
source "$ROOT/scripts/testdb_lease.sh"
PG_NAMES=$(docker ps --format '{{.Names}}' 2>/dev/null || true) # tolerate: docker 缺席=池空 skip 径
testdb_lease_acquire "$PG_NAMES"
if [ -z "${O_TEST_PG_DSN:-}" ]; then
  echo "  ⚠️ [ops_event_sem_gate_test] 测试 PG 池空——C 段 trigger 实测 skip（不猜不占）"
  echo "sem_gate: PASS=$pass FAIL=$fail（C skip）"
  [ "$fail" -eq 0 ]; exit $?
fi
SCHEMA=t_0506sg
TDSN="${O_TEST_PG_DSN}&options=-csearch_path%3D${SCHEMA}"
"$REAL_PSQL" "$TDSN" -q -v ON_ERROR_STOP=1 <<'SQL'
DROP SCHEMA IF EXISTS t_0506sg CASCADE;
CREATE SCHEMA t_0506sg;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE ops_event(
  id bigserial PRIMARY KEY,
  ts timestamptz NOT NULL DEFAULT now(),
  seat text NOT NULL,
  event text NOT NULL,
  ticket text NOT NULL DEFAULT '',
  note text NOT NULL DEFAULT '',
  prev_hash text,
  hash text);
SQL
# fin_0121 hash 链+fin_0154 语义闸实跑共存（迁移序同生产）
"$REAL_PSQL" "$TDSN" -q -v ON_ERROR_STOP=1 \
  -f "$ROOT/migrations/fin_0121_ops_event_chain.sql" || { echo "❌ C fin_0121 应用败"; exit 1; }
"$REAL_PSQL" "$TDSN" -q -v ON_ERROR_STOP=1 \
  -f "$MIG" || { echo "❌ C fin_0154 应用败"; exit 1; }

C() { "$REAL_PSQL" "$TDSN" -qAt -c "$1"; }
try_ins() { # $1=seat $2=event $3=ticket $4=note → stdout+stderr 合流，rc=psql 码
  # ON_ERROR_STOP=1：SQL ERROR→rc3（裸 psql 默认 rc0 不吞错——拒写断言靠 rc）
  "$REAL_PSQL" "$TDSN" -qAt -v ON_ERROR_STOP=1 <<SQL 2>&1
INSERT INTO ops_event(seat,event,ticket,note) VALUES('$1','$2','$3',\$n\$$4\$n\$);
SQL
}
N0=$(C "SELECT count(*) FROM ops_event")

# C1 病灶复现拒：COS+派单（票面判据②字面——裸 psql INSERT 不可绕）
RC=0; OUT=$(try_ins 'COS' '派单' 'T-9998' '积压清账重入派单队') || RC=$?
N1=$(C "SELECT count(*) FROM ops_event")
[ "$RC" -ne 0 ] && [ "$N1" = "$N0" ] \
  && ok "C1 COS+派单 裸INSERT拒且行零落" || bad "C1 拒写失效 rc=$RC n=$N0→$N1"
echo "$OUT" | grep -q 'ops_event_sem_gate' \
  && ok "C1 ERROR loud 前缀在案" || bad "C1 ERROR 前缀缺"
echo "$OUT" | grep -q "seat=COS" && echo "$OUT" | grep -q "event=派单" \
  && echo "$OUT" | grep -q "ticket=T-9998" \
  && ok "C1 ERROR 三元 seat/event/ticket 全在案" || bad "C1 ERROR 三元缺"

# C2 放行集：票面判据②三件+裁决例+领单族+cos-relay 例外
try_ins '决策席' '派单' 'T-9997' '' >/dev/null && ok "C2 决策席+派单 放行" || bad "C2 决策席+派单 误拒"
try_ins 'CTO' '派单' 'T-9997' '' >/dev/null && ok "C2 CTO+派单 放行" || bad "C2 CTO+派单 误拒"
for s in 裁定席 CEO 外审席 外审 COS 席O-COS 审计席; do
  try_ins "$s" '裁决' 'T-9997' '' >/dev/null \
    && ok "C2 $s+裁决 放行" || bad "C2 $s+裁决 误拒"
done
for s in 施工席 无头施工席 席O-Dev claude-kimi 席MANAGE; do
  try_ins "$s" '领单' 'T-9997' '' >/dev/null \
    && ok "C2 $s+领单 放行" || bad "C2 $s+领单 误拒"
done
try_ins 'COS' '派单' 'T-9997' 'cos-relay——O-Dev交互席人派' >/dev/null \
  && ok "C2 COS+派单+cos-relay 例外放行" || bad "C2 cos-relay 例外误拒"
try_ins '席O-COS' '派单' 'T-9997' 'cos-relay 标' >/dev/null \
  && ok "C2 席O-COS+派单+cos-relay 放行" || bad "C2 席O-COS cos-relay 误拒"
try_ins '决策席' '起草' 'T-9997' '' >/dev/null \
  && ok "C2 决策席+起草 放行" || bad "C2 决策席+起草 误拒"
try_ins 'CTO' '拉起' 'T-9997' '' >/dev/null \
  && ok "C2 CTO+拉起 放行" || bad "C2 CTO+拉起 误拒"
try_ins 'META-DM' '拉起' 'T-9997' '' >/dev/null \
  && ok "C2 META-DM+拉起 放行" || bad "C2 META-DM+拉起 误拒"

# C3 说明类+矩阵外相位词任意 seat 放行（判据②宽容面）
for ev in 升件 处置 候水 送达 退班 领号 等锁 先例适用 部署 诊断草案 议题; do
  try_ins '任意新席X' "$ev" 'T-9997' '' >/dev/null \
    && ok "C3 任意席+$ev 说明类放行" || bad "C3 任意席+$ev 误拒"
done
for ev in 开工 外派 收口 移交 收口待审 审计中 完成 挂起 作废 异常; do
  try_ins '任意新席X' "$ev" 'T-9997' '' >/dev/null \
    && ok "C3 任意席+$ev 矩阵外相位词放行" || bad "C3 任意席+$ev 误拒"
done

# C4 越权组合逐一拒（消费面判据反面）
try_ins '席O-COS' '派单' 'T-9996' '业主令插队——部署前置件先消化' >/dev/null 2>&1 \
  && bad "C4 席O-COS+派单裸note 未拒" || ok "C4 席O-COS+派单无标 拒"
try_ins '施工席' '裁决' 'T-9996' '' >/dev/null 2>&1 \
  && bad "C4 施工席+裁决 未拒" || ok "C4 施工席+裁决 拒"
try_ins '决策席' '领单' 'T-9996' '' >/dev/null 2>&1 \
  && bad "C4 决策席+领单 未拒" || ok "C4 决策席+领单 拒"
try_ins '席O-COS' '呈批' 'T-9996' '' >/dev/null 2>&1 \
  && bad "C4 席O-COS+呈批（ceo_queue集外）未拒" || ok "C4 席O-COS+呈批 拒"
try_ins '施工席' '已批' 'T-9996' '' >/dev/null 2>&1 \
  && bad "C4 施工席+已批 自批未拒" || ok "C4 施工席+已批 拒"
try_ins 'CTO' '领单' 'T-9996' '' >/dev/null 2>&1 \
  && bad "C4 CTO+领单 未拒" || ok "C4 CTO+领单 拒"
try_ins 'DM' '派单' 'T-9996' '' >/dev/null 2>&1 \
  && bad "C4 死席DM+派单 未拒" || ok "C4 DM+派单 拒"

# C5 共存实证：合法行过双 trigger——hash 链戳照常落（sem gate 不破链写）
H1=$(C "SELECT hash IS NOT NULL AND prev_hash IS NULL FROM ops_event WHERE ticket='T-9997' AND seat='决策席' AND event='派单' ORDER BY id DESC LIMIT 1")
H2=$(C "SELECT count(*) FROM ops_event WHERE ticket='T-9996'")
[ "$H1" = "t" ] && ok "C5 合法行 hash 链戳落（fin_0121+fin_0154 共存）" || bad "C5 链戳缺 h=$H1"
[ "$H2" = "0" ] && ok "C5 拒写行零落账（T-9996 全拒）" || bad "C5 拒写漏行 n=$H2"
# 精算放行数：C2=2+7+5+2+3=19；C3=11+10=21；合计 40 放行+0 漏（C1/C4 全拒）
NTOT=$(C "SELECT count(*) FROM ops_event")
[ "$NTOT" -eq $((N0 + 40)) ] && ok "C5 账数钉死（放行40/拒8 零漏）" \
  || bad "C5 账数漂移 n=$NTOT want=$((N0+40))"

echo "sem_gate: PASS=$pass FAIL=$fail"
[ "$fail" -eq 0 ]
