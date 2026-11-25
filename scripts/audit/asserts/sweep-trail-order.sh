#!/usr/bin/env bash
# 审计锚 2026-11-25: sweepStaleEphemeral 留痕序断言——#27183 tx 原子化
# refactor 翻转原序(INSERT→UPDATE 成 UPDATE→INSERT),同事务内 INSERT
# 的 SELECT WHERE state='degraded' 只见新态'retired'→恒零行,存量清扫
# 零留痕。正确语义=留痕 INSERT 先于 UPDATE(或同查询谓词快照)。
set -u
f=internal/trading/simdrill/drill.go
body=$(awk '/func sweepStaleEphemeral/,/^}/' "$f")
up_line=$(printf '%s' "$body" | grep -n "UPDATE fin_strategy" | head -1 | cut -d: -f1)
ins_line=$(printf '%s' "$body" | grep -n "INSERT INTO fin_strategy_transition" | head -1 | cut -d: -f1)
if [ -z "$ins_line" ]; then
  echo "RED: sweepStaleEphemeral 无 transition 写径" >&2; exit 1
fi
if [ -n "$up_line" ] && [ "$up_line" -lt "$ins_line" ]; then
  echo "RED: UPDATE(L${up_line}) 先于 INSERT(L${ins_line})——同事务 SELECT 谓词恒空,清扫退役零留痕" >&2
  exit 1
fi
echo "GREEN: sweepStaleEphemeral 留痕先于迁移"
