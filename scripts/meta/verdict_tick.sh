#!/usr/bin/env bash
# verdict_tick.sh —— 经济案裁定席批件拍壳（T-0312 第三刀——照 regime_tick.sh
# 同构逐字对形：pending 水印闸>0 才烧零空拍；每拍（含空拍/败拍）
# ops_headless META-VERDICT 心跳行；凭据归 /opt/o-research/env FIN_PG_DSN
# 隧道径——SG psql 直读）。
set -u
ENVF="${VERDICT_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')

# file 子命令（T-0464 问链立案写径）：why_chain_scan.sh 产案卷后经本径落
# fin_econ_lesson kind='rootcause_diag'——UNIQUE(kind,entity_ref) 幂等；
# roster_rw 列级 INSERT 授权面=fin_0146（status/verdict_ref 走 DEFAULT
# pending 不可伪造）。用法：verdict_tick.sh file <entity_ref> <numbers_json>
if [ "${1:-}" = "file" ]; then
  REF="${2:-}"; NUM="${3:-}"
  [ -n "$FIN_DSN" ] && [ -n "$REF" ] && [ -n "$NUM" ] || exit 1
  psql "$FIN_DSN" -qAt -v ref="$REF" -v num="$NUM" -c \
    "INSERT INTO fin_econ_lesson(kind,entity_ref,numbers)
     VALUES('rootcause_diag',:'ref',:'num'::jsonb)
     ON CONFLICT (kind,entity_ref) DO NOTHING" >/dev/null 2>&1 || true # tolerate: 立案写径败=下拍重试（去重闸在扫件侧）
  exit 0
fi

hb() { # $1=state 心跳行（ops_headless META-VERDICT——meta 席位口径）
  [ -n "$FIN_DSN" ] || return 0
  psql "$FIN_DSN" -qAt -c "INSERT INTO ops_headless(ticket,seat,state,last_activity)
    VALUES('META-VERDICT','裁定席','$1',now())
    ON CONFLICT (ticket) DO UPDATE SET seat=EXCLUDED.seat,
      state=EXCLUDED.state,last_activity=now(),updated_at=now()" >/dev/null 2>&1 || true
}

# ① 水印闸：econ+eng 两案 pending 合>0 才烧——否则 skip（空拍心跳照写）。
# T-0385 裁域扩：fin_eng_lesson（工程案矿工 pending）同径入裁；表缺
# （未部署窗）第二查询败=echo 0 兜底不炸。
if [ -n "$FIN_DSN" ]; then
  N=$(psql "$FIN_DSN" -qAt -c "SELECT count(*) FROM fin_econ_lesson WHERE status='pending'" 2>/dev/null || echo 0)
  E=$(psql "$FIN_DSN" -qAt -c "SELECT count(*) FROM fin_eng_lesson WHERE status='pending'" 2>/dev/null || echo 0)
  [ "$E" -eq "$E" ] 2>/dev/null || E=0
  N=$((N + E))
  if [ "$N" = "0" ]; then
    hb alive
    exit 0
  fi
fi

# ② devin 批拍（cwd=scratch——零仓件接触；runbook 绝对路径指件）。
# DEVIN_BIN=桩测覆写位（生产默认钦定径不动——T-0385 水印闸桩测面）。
cd /var/tmp/o-verdict || exit 1
"${DEVIN_BIN:-/home/mluser/.local/bin/devin}" -p --model swe --respect-workspace-trust false \
  --permission-mode dangerous -- \
  "读 /opt/o/skills/裁定席/SKILL.md runbook 拍经济案+工程案裁定"
RC=$?
# ③ 每拍心跳全件（成败同写——败拍=stalled 显形非隐）。
[ $RC -eq 0 ] && hb alive || hb stalled
exit 0
