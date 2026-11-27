#!/usr/bin/env bash
# premise-dead-family-predicate.sh —— 审计/饥饿感知 复证 RED（复证工单#39796
# 族活性谓词根修尚未落地）：
#   deskPremiseDeath 的「族死」判定必须=「该 template_ref 族下零存活员」
#   （NOT EXISTS state<>'retired'），不是「任一退役员即中」。
# 实证：desklife_premise.go deskRetiredRefs 现收 `WHERE state='retired'`
# 全集——部分退役族（dir_perp_mom 209退役/182活、perp_trend_daily 2/1）
# 仍命中 → premise_dead 四连误报 #39747/48/49/40070（40070 在工单#39796
# 立票后仍发）。
# 本断言钉正确语义：退役集构建必须带活员排他谓词（NOT EXISTS / HAVING
# count(state<>'retired')=0 同类），修后自绿。
set -u
f="$(cd "$(dirname "$0")/../../.." && pwd)/internal/portfolio/hunger/desklife_premise.go"
# deskRetiredRefs 函数体截取（函数起→首个列0闭花——/^func / 起止
# 双中签名行=永 RED 坑实证 2026-11-27，闭花界=函数体完形）
body=$(awk '/func \(s \*Sensor\) deskRetiredRefs/,/^}/' "$f")
[ -n "$body" ] || { echo "RED: deskRetiredRefs 函数缺失"; exit 1; }
# 正确语义锚：族死=零活员——查询必须含活员排他谓词之一
echo "$body" | grep -qE "NOT EXISTS|state *<> *'retired'.*GROUP|HAVING|EXCEPT|count\(.*'retired'" \
  && { echo "GREEN: 退役集含活员排他谓词"; exit 0; }
echo "RED: deskRetiredRefs 收集所有含退役员的 template_ref——部分退役族误报 premise_dead"
echo "$body" | grep -n "SELECT\|WHERE\|retired" | head -6
exit 1
