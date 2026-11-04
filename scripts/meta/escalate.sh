#!/usr/bin/env bash
# escalate.sh —— 带类升件写径（T-0334 判例索引面）：ops_event '升件' 行
# 携 esc_class/esc_scope 落库——闸臂（fin_0111 ops_precedent_trg）同拍
# 精确匹配，命中即自动裁落 '先例适用' 行（本调用返回时判例面已判）。
# 类词参照 scripts/meta/precedent_classes.tsv（等值匹配册不参判；
# 红线类升件照写——闸臂结构性永不自动裁=照常上行）。
# 用法：escalate.sh <seat> <ticket> <class> <scope> <note>
#   ticket 无归属传 ''；class/scope 传 '' 即无类升件（永不入判例面）。
# 失败静默不阻断主务（report.sh 同款——升件本体不落≠席务停）。
set -u
[ $# -ge 5 ] || { echo "用法: escalate.sh <seat> <ticket> <class> <scope> <note>" >&2; exit 2; }
SEAT=$1; TID=$2; CLS=$3; SCP=$4; NOTE=$5
ENVF="${ESCALATE_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$DSN" ] || exit 0
QC(){ printf '%s' "${1:-}" | sed "s/'/''/g"; }
CCLS="NULL"; CSCP="NULL"
[ -n "$CLS" ] && CCLS="'$(QC "$CLS")'"
[ -n "$SCP" ] && CSCP="'$(QC "$SCP")'"
# 带类 INSERT 败（esc_* 列未随迁移落产窗期）回落无类升件——升件本体不丢。
psql "$DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
  VALUES('$(QC "$SEAT")','升件','$(QC "$TID")','$(QC "$NOTE")',$CCLS,$CSCP)" >/dev/null 2>&1 \
  || psql "$DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
  VALUES('$(QC "$SEAT")','升件','$(QC "$TID")','$(QC "$NOTE")')" >/dev/null 2>&1 || true
exit 0
