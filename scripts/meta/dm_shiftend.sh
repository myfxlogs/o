#!/usr/bin/env bash
# dm_shiftend.sh —— 决策席班退闸（T-0437 件③——退不可静默）。班次 drain
# 判空后、退班前一拍必跑：机检全局账两源——①已批未领（dm_queue.sh approved
# 单源推导）②有挡项（deploy_window_due.sh state=有挡，reason 挡件面标即
# 冻结明细）。非空=落 ops_event「退班理由行」（谁挡/等什么/下一拍谁接
# 三字段必携——机械可核粒度）——班退带着账走；全账空=shiftend=clean
# 静默退合法。源面不可知（查询败）=fail-closed 照落行——不可知也必须发声。
# 写面判据（collab §1 机检界碑同形）：event='退班' 行只归决策席班退闸——
# 触发器面安全（ticket='' 不触 headless 轻碰/判例闸；NOTIFY 语句级纯唤醒）。
# 输出契约：shiftend=clean（全账空，零行）/ shiftend=reasoned（理由行已落）
# ；exit 0=闸跑净，exit 1=理由行写径败（消费席按 SKILL 升件自报禁静默退）。
# 环境覆写（桩测/对拍面）：DM_ENV_FILE、O_SHIFTEND_QUEUE_SH、
# O_SHIFTEND_WINDOW_SH（窗壳路径覆写——桩测不跑真 ssh/psql 源）、
# O_SHIFTEND_PSQL（psql 覆写——捕获 INSERT）。
set -u
ENVF="${DM_ENV_FILE:-/opt/o-research/env}"
FIN_DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
DM_QUEUE_SH=${O_SHIFTEND_QUEUE_SH:-"$SCRIPT_DIR/dm_queue.sh"}
WINDOW_SH=${O_SHIFTEND_WINDOW_SH:-"$SCRIPT_DIR/deploy_window_due.sh"}
PSQL_BIN=${O_SHIFTEND_PSQL:-psql}

LEDGER=""; WAIT=""; QUEUESRC=""
# ① 已批未领面（dm_queue.sh approved——T-0413/T-0414 已批待派单源推导）
QRC=0; APPROVED=$("$DM_QUEUE_SH" approved 2>/dev/null) || QRC=$?
if [ "$QRC" -ne 0 ]; then
  LEDGER="已批未领面不可知(rc=$QRC)"
  QUEUESRC="fail-closed——队列推导败不可知也必须发声"
else
  LIST=$(printf '%s\n' "$APPROVED" | awk 'NF' | paste -sd, -)
  if [ -n "$LIST" ]; then
    LEDGER="已批未领=$LIST"
    QUEUESRC="等派单——常径=dm_depgate 零交集即派/交集缺清单件按 T-0413 留队待审后下拍重判"
  fi
fi
# ② 有挡项面（deploy_window_due.sh state=有挡——reason 挡件面标即冻结明细）
WRC=0; WIN_OUT=$(bash "$WINDOW_SH" 2>/dev/null) || WRC=$?
WSTATE=$(sed -n 's/^state=//p' <<<"$WIN_OUT" | head -1)
WREASON=$(sed -n 's/^reason=//p' <<<"$WIN_OUT" | head -1)
if [ "$WRC" -ne 0 ] && [ "$WSTATE" != "有挡" ] && [ -z "$WSTATE" ]; then
  LEDGER="${LEDGER:+$LEDGER；}部署窗有挡项面不可知(rc=$WRC)"
  WAIT="${WAIT:+$WAIT；}窗壳本体败 fail-closed——不可知也必须发声"
elif [ "$WSTATE" = "有挡" ]; then
  LEDGER="${LEDGER:+$LEDGER；}部署窗有挡"
  WAIT="${WAIT:+$WAIT；}等挡件清窗——${WREASON:-reason不可读}"
fi

if [ -z "$LEDGER" ]; then
  echo "shiftend=clean"
  exit 0
fi

NEXT="dm_tick 30min 兜底拍（dm_depgate 补拉窗）→下班班次承接"
NOTE="退班理由：谁挡=$LEDGER；等什么=${QUEUESRC}${QUEUESRC:+；}${WAIT}；下一拍谁接=$NEXT"
if [ -z "$FIN_DSN" ]; then
  echo "[shiftend] FIN_DSN 缺——理由行写径不可达（不可静默退）" >&2
  exit 1
fi
QNOTE=$(printf '%s' "$NOTE" | sed "s/'/''/g")
if ! "$PSQL_BIN" "$FIN_DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
  VALUES('CTO','退班','','$QNOTE')" >/dev/null 2>&1; then
  echo "[shiftend] 退班理由行写径败——升件自报禁静默退" >&2
  exit 1
fi
echo "shiftend=reasoned"
echo "row=$NOTE"
exit 0
