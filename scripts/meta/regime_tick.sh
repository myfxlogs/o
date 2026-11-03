#!/usr/bin/env bash
# regime_tick.sh —— 分析席批件拍壳（T-0311 返修①③④：30min 水印闸+每拍心跳
# 全件：闸败=skip 如实 skip 零 devin 调用；拍/空拍/败拍全写 ops_headless
# META-REGIME 行——无人值守可见性锚；凭据归 env 文件 /opt/o-research/env
# 或 REGIME_ENV_FILE 覆写，FIN_PG_DSN=SG psql、MD_PG_DSN=OCI PG-md psql）。
set -u
ENVF="${REGIME_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
MD_DSN=$(grep '^MD_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')

hb() { # $1=state 心跳行（ops_headless META-REGIME——meta 席位口径不冒生产岗）
  [ -n "$FIN_DSN" ] || return 0
  psql "$FIN_DSN" -qAt -c "INSERT INTO ops_headless(ticket,seat,state,last_activity)
    VALUES('META-REGIME','分析席','$1',now())
    ON CONFLICT (ticket) DO UPDATE SET seat=EXCLUDED.seat,
      state=EXCLUDED.state,last_activity=now(),updated_at=now()" >/dev/null 2>&1 || true
}

# ① 水印闸：md_ctx_fact 新行>regime 水印才烧——否则 skip（空拍心跳照写）。
if [ -n "$MD_DSN" ]; then
  GATE=$(psql "$MD_DSN" -qAt -c "SELECT CASE WHEN
    (SELECT max(id) FROM md_ctx_fact) >
    COALESCE((SELECT max((evidence->>'ctx_max_id')::bigint) FROM md_regime_fact),0)
    THEN 't' ELSE 'f' END" 2>/dev/null || echo f)
  if [ "$GATE" != "t" ]; then
    hb alive
    exit 0
  fi
fi

# ② devin 批拍（cwd=scratch——零仓件接触；runbook 绝对路径指件）。
cd /var/tmp/o-regime || exit 1
/home/mluser/.local/bin/devin -p --model swe --respect-workspace-trust false \
  "读 /opt/o/skills/分析席/SKILL.md runbook 拍 regime 判词"
RC=$?
# ③ 每拍心跳全件（成败同写——败拍=stalled 显形非隐）。
[ $RC -eq 0 ] && hb alive || hb stalled
exit 0
