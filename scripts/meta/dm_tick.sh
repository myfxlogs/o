#!/usr/bin/env bash
# dm_tick.sh —— 决策席待裁批件拍壳（T-0312 第六刀——照 verdict_tick.sh
# 逐字对形：ops_ticket_state 待裁集水印闸>0 才烧零空拍；每拍（含空拍/
# 败拍）ops_headless META-DM 心跳行；FIN_DSN 隧道径 SG 直读）。
set -u
# T-0323：裁毕通报踢 O-DM 交互席+自报 seat=决策席 正标（O_SEAT 误标案销）。
export REPORT_KICK=O-DM
export O_SEAT=决策席
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')

hb() { # $1=state 心跳行（META-DM——meta 席位口径）
  [ -n "$FIN_DSN" ] || return 0
  psql "$FIN_DSN" -qAt -c "INSERT INTO ops_headless(ticket,seat,state,last_activity)
    VALUES('META-DM','决策席','$1',now())
    ON CONFLICT (ticket) DO UPDATE SET seat=EXCLUDED.seat,
      state=EXCLUDED.state,last_activity=now(),updated_at=now()" >/dev/null 2>&1 || true
}

# ① 待裁闸：ops_ticket_state 待裁事件>0 才烧——否则 skip（心跳照写）。
if [ -n "$FIN_DSN" ]; then
  # T-0315②：最新刻事件集判定（max(ts) 时刻集含终态词 裁决/完成/驳回
  # 即离集——同 ts 撞族 DISTINCT ON 无序漂移根修；ops_event 唯一真源
  # 不票面交叉）。
  N=$(psql "$FIN_DSN" -qAt -c "SELECT count(*) FROM (
      SELECT ticket, max(ts) mt FROM ops_event WHERE ticket<>'' GROUP BY ticket
    ) l WHERE EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
        AND e.ts=l.mt AND e.event IN ('收口','移交','收口待审'))
      AND NOT EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
        AND e.ts=l.mt AND e.event IN ('裁决','完成','驳回'))" 2>/dev/null || echo 0)
  if [ "$N" = "0" ]; then
    hb alive
    exit 0
  fi
fi

# ② devin 批拍（cwd=scratch——裁决协议 runbook 绝对路径指件）。
cd /var/tmp/o-decision || exit 1
/home/mluser/.local/bin/devin -p --model swe --respect-workspace-trust false \
  --permission-mode dangerous -- \
  "读 /opt/o/skills/决策席/SKILL.md 裁决协议节 拍待裁件裁决"
RC=$?
# ③ 每拍心跳全件（成败同写——败拍=stalled 显形非隐）。
[ $RC -eq 0 ] && hb alive || hb stalled

# T-0324 漏拉清扫闸（返修①）：ops_event 派单行后同票无领单且无活
# o-worker-<tid> unit→dev_run.sh 幂等补拉。三守卫钦定——①最新任意事件
# 恰为派单（终态 收口/移交/裁决/完成/驳回/拉起/领单 超越免拉，T-0324
# 误派行收口先于派单案在案）②补拉落 `拉起` 行入 filter 集自标幂等
# ③无票文件免拉（T-9999 类测试行天然排）。is-active 按行精确比对——
# 'inactive' 字面含 'active' grep 误配恒判活=闸永不补拉病灶根修。
if [ -n "$FIN_DSN" ]; then
  psql "$FIN_DSN" -qAt -c "SELECT ticket FROM (
      SELECT ticket,max(ts) mt FROM ops_event WHERE event IN
        ('派单','领单','拉起','收口','移交','收口待审','裁决','完成','驳回')
        AND ticket<>'' GROUP BY ticket) l
    WHERE EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
        AND e.ts=l.mt AND e.event='派单')" 2>/dev/null | while read -r tid; do
    compgen -G "/opt/o/docs/tasks/${tid}-*.md" >/dev/null || continue # ③无票免拉
    st=$(systemctl --user is-active "o-worker-$tid.service" 2>/dev/null || true)
    case "$st" in active|activating) continue;; esac # 精确比对误配恒死修
    if systemd-run --user --unit="o-worker-$tid" --collect         /opt/o/scripts/meta/dev_run.sh "$tid"         "读 /opt/o/skills/施工席/SKILL.md 施工协议 领 $tid 施工" >/dev/null 2>&1; then
      psql "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
        VALUES('META-DM','拉起','$tid','漏拉闸补拉幂等自标')" >/dev/null 2>&1 || true
    fi
  done
fi
exit 0
